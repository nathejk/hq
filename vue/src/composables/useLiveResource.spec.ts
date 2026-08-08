import { describe, it, expect, beforeEach, vi } from 'vitest';
import { effectScope } from 'vue';
import bus from '@/plugins/bus';
import { SIGNAL_ENTITY_CHANGED, SIGNAL_RESYNC } from '@/plugins/live';
import { useLiveResource, clearLiveCache, liveCacheSize } from '@/composables/useLiveResource';

/** Let queued promise callbacks run. */
const flush = () => new Promise((resolve) => setTimeout(resolve, 0));

/**
 * Run inside an effect scope, as a component would, so onScopeDispose is valid.
 * Returns the scope so a test can stop it to simulate unmounting.
 */
function inScope<T>(fn: () => T): { result: T; stop: () => void } {
  const scope = effectScope();
  const result = scope.run(fn) as T;
  return { result, stop: () => scope.stop() };
}

const changed = (entity: string, id?: string, event?: string) =>
  bus.emit('live', { type: SIGNAL_ENTITY_CHANGED, entity, id, year: '2026', event });

beforeEach(() => {
  clearLiveCache();
});

describe('useLiveResource', () => {
  it('serves a second consumer from cache without refetching', async () => {
    const fetcher = vi.fn().mockResolvedValue('first');

    const a = inScope(() => useLiveResource('sos:list', fetcher, { dependsOn: ['sos'] }));
    await flush();
    expect(a.result.data.value).toBe('first');
    expect(fetcher).toHaveBeenCalledTimes(1);

    // A second view mounting the same key must not trigger a request.
    const b = inScope(() => useLiveResource('sos:list', fetcher, { dependsOn: ['sos'] }));
    await flush();
    expect(b.result.data.value).toBe('first');
    expect(fetcher).toHaveBeenCalledTimes(1);
  });

  it('renders from cache with no request after unmount and remount', async () => {
    const fetcher = vi.fn().mockResolvedValue('cached');

    const first = inScope(() => useLiveResource('sos:1', fetcher, { dependsOn: ['sos:1'] }));
    await flush();
    expect(fetcher).toHaveBeenCalledTimes(1);
    first.stop(); // navigate away

    const second = inScope(() => useLiveResource('sos:1', fetcher, { dependsOn: ['sos:1'] }));
    // Synchronously available: this is the "instant navigation" requirement.
    expect(second.result.data.value).toBe('cached');
    expect(second.result.pending.value).toBe(false);
    await flush();
    expect(fetcher).toHaveBeenCalledTimes(1);
  });

  it('reports pending only when there is nothing to show', async () => {
    const fetcher = vi.fn().mockResolvedValue('v1');
    const { result } = inScope(() => useLiveResource('k', fetcher, { dependsOn: [] }));

    expect(result.pending.value).toBe(true); // first load, nothing cached
    await flush();
    expect(result.pending.value).toBe(false);

    await result.refresh(); // revalidation must not flash a spinner
    expect(result.pending.value).toBe(false);
  });

  it('revalidates on a type-level signal, so new rows appear in lists', async () => {
    const fetcher = vi.fn().mockResolvedValueOnce(['a']).mockResolvedValueOnce(['a', 'b']);
    const { result } = inScope(() => useLiveResource('sos:list', fetcher, { dependsOn: ['sos'] }));
    await flush();
    expect(result.data.value).toEqual(['a']);

    // An id the client has never seen: only a type-level dependency can catch this.
    changed('sos', 'brand-new-id', 'created');
    await flush();

    expect(fetcher).toHaveBeenCalledTimes(2);
    expect(result.data.value).toEqual(['a', 'b']);
  });

  it('revalidates only the entry whose instance changed', async () => {
    const one = vi.fn().mockResolvedValue('one');
    const two = vi.fn().mockResolvedValue('two');

    inScope(() => useLiveResource('sos:1', one, { dependsOn: ['sos:1'] }));
    inScope(() => useLiveResource('sos:2', two, { dependsOn: ['sos:2'] }));
    await flush();
    expect(one).toHaveBeenCalledTimes(1);
    expect(two).toHaveBeenCalledTimes(1);

    changed('sos', '1', 'commented');
    await flush();

    expect(one).toHaveBeenCalledTimes(2);
    expect(two).toHaveBeenCalledTimes(1); // untouched
  });

  it('ignores signals for entities it does not depend on', async () => {
    const fetcher = vi.fn().mockResolvedValue('x');
    inScope(() => useLiveResource('payment:list', fetcher, { dependsOn: ['payment'] }));
    await flush();

    changed('scan', '999', 'scanned');
    await flush();

    expect(fetcher).toHaveBeenCalledTimes(1);
  });

  it('revalidates every held entry on resync', async () => {
    const a = vi.fn().mockResolvedValue('a');
    const b = vi.fn().mockResolvedValue('b');
    inScope(() => useLiveResource('a', a, { dependsOn: ['sos'] }));
    inScope(() => useLiveResource('b', b, { dependsOn: ['payment'] }));
    await flush();

    // What polling emits every tick, and what a reconnect or a server-side
    // buffer overflow emits: one path, shared.
    bus.emit('live', { type: SIGNAL_RESYNC, reason: 'poll' });
    await flush();

    expect(a).toHaveBeenCalledTimes(2);
    expect(b).toHaveBeenCalledTimes(2);
  });

  it('evicts on a 404 rather than surfacing an error', async () => {
    const fetcher = vi
      .fn()
      .mockResolvedValueOnce('here')
      .mockRejectedValueOnce({ response: { status: 404 } });

    const { result } = inScope(() => useLiveResource('sos:9', fetcher, { dependsOn: ['sos:9'] }));
    await flush();
    expect(liveCacheSize()).toBe(1);

    await result.refresh();
    await flush();

    expect(liveCacheSize()).toBe(0);
    expect(result.error.value).toBeUndefined();
  });

  it('keeps a non-404 error on the entry', async () => {
    const fetcher = vi
      .fn()
      .mockResolvedValueOnce('ok')
      .mockRejectedValueOnce({ response: { status: 500 } });

    const { result } = inScope(() => useLiveResource('k', fetcher, { dependsOn: [] }));
    await flush();
    await result.refresh();
    await flush();

    expect(result.error.value).toEqual({ response: { status: 500 } });
    expect(result.data.value).toBe('ok'); // last good value still shown
  });

  it('collapses overlapping revalidations into one request', async () => {
    let resolveFetch: ((v: unknown) => void) | undefined;
    const fetcher = vi.fn().mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveFetch = resolve;
        }),
    );

    const { result } = inScope(() => useLiveResource('k', fetcher, { dependsOn: ['sos'] }));
    // Three more triggers while the first is still in flight.
    void result.refresh();
    changed('sos', '1');
    changed('sos', '2');

    expect(fetcher).toHaveBeenCalledTimes(1);

    resolveFetch?.('done');
    await flush();
    expect(result.data.value).toBe('done');
  });

  it('drops everything when the cache is cleared', async () => {
    const fetcher = vi.fn().mockResolvedValue('v');
    inScope(() => useLiveResource('k', fetcher, { dependsOn: [] }));
    await flush();
    expect(liveCacheSize()).toBe(1);

    clearLiveCache();

    expect(liveCacheSize()).toBe(0);
  });
});
