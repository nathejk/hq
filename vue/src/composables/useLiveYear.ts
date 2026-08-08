import { watch } from 'vue';
import { useGlobalState } from '@/composables/globalstate';
import { startLive, stopLive } from '@/plugins/live';
import { flushLiveCache } from '@/composables/useLiveResource';

/**
 * Keep live updates aligned with the selected event year.
 *
 * Two failures this prevents:
 *
 * 1. **Stale cross-year data.** The cache is keyed by resource, not by year, so
 *    without an explicit flush a 2025 list would linger while the operator
 *    believes they are looking at 2026. A brief spinner is much better than
 *    plausible wrong data.
 * 2. **Silent divergence.** REST calls carry the year via the `X-YearSlug`
 *    interceptor; the stream carries it as a subscribe-time parameter, because
 *    `EventSource` cannot set headers. If those two could drift apart, the client
 *    would receive signals for one year while fetching another and simply appear
 *    frozen — no error, nothing in a log. Both therefore read the *same*
 *    `globalstate` value, which is what makes divergence impossible rather than
 *    merely unlikely.
 *
 * An empty year means "the current calendar year": `globalstate` normalises it
 * that way and the axios interceptor omits the header, leaving the server to
 * apply the same default (`YearSlug()`, go/cmd/api/routes.go).
 */
let installed = false;
let stopWatching: (() => void) | undefined;

export function installLiveYearSync(): void {
  if (installed) return;
  installed = true;

  const { yearSlug } = useGlobalState();

  stopWatching = watch(
    yearSlug,
    (year, previous) => {
      if (previous !== undefined && year === previous) return;

      // Order matters: stop first so no in-flight signal lands mid-flush, then
      // reset the cache, then subscribe for the new year.
      stopLive();
      if (previous !== undefined) flushLiveCache();
      startLive(year);
    },
    { immediate: true },
  );
}

/**
 * Tear the sync down again.
 *
 * The watcher is created outside any component scope — deliberately, since it must
 * outlive every view — so it has to be stopped explicitly rather than relying on
 * scope disposal. Without this, tests (or a hot reload) accumulate watchers that
 * all react to the next year change.
 */
export function resetLiveYearSyncForTests(): void {
  stopWatching?.();
  stopWatching = undefined;
  installed = false;
}
