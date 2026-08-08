import { describe, it, expect, beforeEach, vi } from 'vitest';
import { effectScope, nextTick } from 'vue';
import { useGlobalState } from '@/composables/globalstate';
import {
  installLiveYearSync,
  resetLiveYearSyncForTests,
} from '@/composables/useLiveYear';
import {
  useLiveResource,
  clearLiveCache,
  flushLiveCache,
} from '@/composables/useLiveResource';
import { liveYear, startLive, stopLive } from '@/plugins/live';

const flush = () => new Promise((resolve) => setTimeout(resolve, 0));

function inScope<T>(fn: () => T): T {
  return effectScope().run(fn) as T;
}

beforeEach(() => {
  clearLiveCache();
  stopLive();
  resetLiveYearSyncForTests();
});

describe('year switching', () => {
  it('refetches every held resource and shows no pre-change value', async () => {
    const fetcher = vi.fn().mockResolvedValueOnce('2026 data').mockResolvedValueOnce('2025 data');
    const resource = inScope(() => useLiveResource('k', fetcher, { dependsOn: ['sos'] }));
    await flush();
    expect(resource.data.value).toBe('2026 data');

    flushLiveCache();

    // Critical: the mounted view must not still be showing last year's value.
    expect(resource.data.value).toBeUndefined();
    expect(resource.pending.value).toBe(true);

    await flush();
    expect(resource.data.value).toBe('2025 data');
    expect(fetcher).toHaveBeenCalledTimes(2);
  });

  it('discards a response from before the switch', async () => {
    // A slow request for the old year that resolves *after* the year changed
    // must not land on top of the new year's data.
    let resolveOld: ((v: unknown) => void) | undefined;
    const fetcher = vi
      .fn()
      .mockImplementationOnce(() => new Promise((r) => { resolveOld = r; }))
      .mockResolvedValueOnce('new year');

    const resource = inScope(() => useLiveResource('k', fetcher, { dependsOn: ['sos'] }));

    flushLiveCache(); // year changes while the first request is still open
    await flush();
    expect(resource.data.value).toBe('new year');

    resolveOld?.('old year'); // arrives late
    await flush();

    expect(resource.data.value).toBe('new year');
  });

  it('subscribes the transport to the selected year', async () => {
    const { setYearSlug } = useGlobalState();

    installLiveYearSync();
    await nextTick();
    // Empty means "current calendar year" — the same normalisation the axios
    // interceptor and the server default rely on.
    expect(liveYear()).toBe('');

    setYearSlug('2025');
    await nextTick();
    expect(liveYear()).toBe('2025');

    setYearSlug('2024');
    await nextTick();
    expect(liveYear()).toBe('2024');
  });

  it('flushes the cache when the year changes through global state', async () => {
    const { setYearSlug } = useGlobalState();
    setYearSlug('2026');

    installLiveYearSync();
    await nextTick();

    const fetcher = vi.fn().mockResolvedValueOnce('a').mockResolvedValueOnce('b');
    const resource = inScope(() => useLiveResource('k', fetcher, { dependsOn: ['sos'] }));
    await flush();
    expect(resource.data.value).toBe('a');

    setYearSlug('2025');
    await nextTick();
    await flush();

    expect(fetcher).toHaveBeenCalledTimes(2);
    expect(resource.data.value).toBe('b');
    expect(liveYear()).toBe('2025');
  });

  it('startLive is idempotent for the same year', () => {
    startLive('2025');
    startLive('2025');
    expect(liveYear()).toBe('2025');
  });
});
