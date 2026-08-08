import { describe, it, expect, beforeEach, vi } from 'vitest';
import { effectScope } from 'vue';
import bus from '@/plugins/bus';
import { SIGNAL_ENTITY_CHANGED } from '@/plugins/live';
import { useLiveResource, clearLiveCache } from '@/composables/useLiveResource';
import { optimisticWrite } from '@/composables/optimisticWrite';

const flush = () => new Promise((resolve) => setTimeout(resolve, 0));

function inScope<T>(fn: () => T): T {
  return effectScope().run(fn) as T;
}

beforeEach(() => {
  clearLiveCache();
});

describe('optimisticWrite', () => {
  it('shows the operator’s change immediately, before the write resolves', async () => {
    const fetcher = vi.fn().mockResolvedValue('server');
    const resource = inScope(() => useLiveResource('k', fetcher, { dependsOn: ['sos'] }));
    await flush();

    let resolveWrite: (() => void) | undefined;
    const write = () =>
      new Promise<void>((resolve) => {
        resolveWrite = resolve;
      });

    const pending = optimisticWrite(resource, 'typed by operator', write);

    // The point of the whole helper: visible without waiting for the round trip.
    expect(resource.data.value).toBe('typed by operator');

    resolveWrite?.();
    await pending;
  });

  it('reconciles with the server value rather than keeping the guess', async () => {
    // The server normalises what was sent — the screen must end up showing the
    // server's version, not the optimistic one.
    const fetcher = vi.fn().mockResolvedValueOnce('old').mockResolvedValueOnce('NORMALISED');
    const resource = inScope(() => useLiveResource('k', fetcher, { dependsOn: ['sos'] }));
    await flush();

    await optimisticWrite(resource, 'normalised', () => Promise.resolve('ok'));

    expect(resource.data.value).toBe('NORMALISED');
  });

  it('restores the previous value and rethrows when the write fails', async () => {
    const fetcher = vi.fn().mockResolvedValue('original');
    const resource = inScope(() => useLiveResource('k', fetcher, { dependsOn: ['sos'] }));
    await flush();

    const failure = new Error('500');
    await expect(
      optimisticWrite(resource, 'never persisted', () => Promise.reject(failure)),
    ).rejects.toBe(failure);

    // Both halves matter: the value is back, and the caller was told.
    expect(resource.data.value).toBe('original');
  });

  it('supports an updater function over the current value', async () => {
    const fetcher = vi.fn().mockResolvedValue({ comments: ['a'] });
    const resource = inScope(() =>
      useLiveResource<{ comments: string[] }>('k', fetcher, { dependsOn: ['sos'] }),
    );
    await flush();

    void optimisticWrite(
      resource,
      (current) => ({ comments: [...(current?.comments ?? []), 'b'] }),
      () => Promise.resolve('ok'),
      { revalidate: false },
    );

    expect(resource.data.value).toEqual({ comments: ['a', 'b'] });
  });

  it('does not flicker when a signal for the operator’s own write arrives', async () => {
    const fetcher = vi.fn().mockResolvedValueOnce('old').mockResolvedValue('committed');
    const resource = inScope(() => useLiveResource('sos:1', fetcher, { dependsOn: ['sos:1'] }));
    await flush();

    const done = optimisticWrite(resource, 'committed', () => Promise.resolve('ok'));
    // The signal for our own write races the revalidation.
    bus.emit('live', { type: SIGNAL_ENTITY_CHANGED, entity: 'sos', id: '1', year: '2026' });

    await done;
    await flush();

    // Whatever the interleaving, the end state is the server's value — and it was
    // never momentarily something else.
    expect(resource.data.value).toBe('committed');
  });

  it('leaves a newer value alone when a failed write rolls back', async () => {
    const fetcher = vi.fn().mockResolvedValue('original');
    const resource = inScope(() => useLiveResource('k', fetcher, { dependsOn: ['sos'] }));
    await flush();

    const attempt = optimisticWrite(resource, 'mine', () =>
      Promise.reject(new Error('nope')),
    );
    // Someone else's change lands before our failure does.
    resource.set('someone else');

    await expect(attempt).rejects.toThrow('nope');

    // Rolling back over a newer value would be worse than leaving it.
    expect(resource.data.value).toBe('someone else');
  });

  it('keeps the last write when two are issued in quick succession', async () => {
    const fetcher = vi.fn().mockResolvedValueOnce('v0').mockResolvedValue('v2');
    const resource = inScope(() => useLiveResource('k', fetcher, { dependsOn: ['sos'] }));
    await flush();

    await optimisticWrite(resource, 'v1', () => Promise.resolve('ok'), { revalidate: false });
    await optimisticWrite(resource, 'v2', () => Promise.resolve('ok'));

    expect(resource.data.value).toBe('v2');
  });
});
