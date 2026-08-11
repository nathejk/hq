import { ref, shallowRef, computed, onScopeDispose, type Ref } from 'vue';
import bus from '@/plugins/bus';
import {
  isEntityChanged,
  isResync,
  signalMatches,
  validateDependencies,
  type LiveDependency,
  type LiveSignal,
} from '@/plugins/live';

/**
 * A live, cached resource.
 *
 * One cache entry per key, held at module level so it survives route changes:
 * navigating away and back renders from memory with no request at all, which is
 * where the "instant" in PRD 004 actually comes from. No transport choice fixes
 * that — a push-based app that refetches on every mount still feels slow.
 */

/** Thrown-away marker for "this key holds nothing yet". */
const EMPTY = Symbol('empty');

type Entry<T = unknown> = {
  /** Cached value, or EMPTY when never loaded. */
  data: Ref<T | typeof EMPTY>;
  /** True only while a first load is in flight — never during revalidation. */
  pending: Ref<boolean>;
  error: Ref<unknown>;
  /** Entity types and/or instances that invalidate this entry. */
  dependsOn: LiveDependency[];
  fetcher: () => Promise<unknown>;
  /** In-flight request, so overlapping revalidations collapse into one. */
  inFlight?: Promise<void>;
  /**
   * Bumped whenever the entry's contents are invalidated wholesale, e.g. a year
   * change. A response that resolves after its generation has passed is
   * discarded — otherwise a slow request from the previous year could land on
   * top of the new year's data, which is exactly the stale-data-that-looks-live
   * failure this whole design exists to avoid.
   */
  gen: number;
  /** Number of live consumers; an entry with none is kept (it is a cache). */
  refs: number;
};

const entries = new Map<string, Entry>();

/** How many consumers/entries exist — for diagnostics and tests. */
export function liveCacheSize(): number {
  return entries.size;
}

/**
 * Drop everything, discarding the entries themselves.
 *
 * Note this orphans any entry a mounted component is still holding, so it is for
 * teardown and tests. To reset a running app — a year change — use
 * {@link flushLiveCache}, which resets entries in place so mounted views follow
 * along.
 */
export function clearLiveCache(): void {
  entries.clear();
}

/**
 * Reset every entry in place and refetch.
 *
 * Used when the selected year changes (task 025). The cache is keyed by resource,
 * not by year, so everything held is now from the wrong year. Resetting in place
 * rather than dropping the map matters: components captured their entry at setup,
 * so a dropped entry would leave them displaying old-year data forever, with no
 * signal able to reach it. Bumping the generation also invalidates any in-flight
 * request from the previous year.
 */
export function flushLiveCache(): void {
  for (const [key, entry] of entries) {
    entry.gen += 1;
    entry.inFlight = undefined;
    entry.data.value = EMPTY;
    entry.error.value = undefined;
    entry.pending.value = false;
    void revalidate(key, entry);
  }
}

/** Remove a single key, e.g. after a delete. */
export function evictLiveResource(key: string): void {
  entries.delete(key);
}

/**
 * Put a known value into another key's cache entry, without fetching.
 *
 * For the write-then-navigate case: an operator creates something and is sent
 * straight to its own page. Because projections apply asynchronously, a fetch on
 * arrival races the projection and usually loses — the API answers 404 and the
 * operator is told the thing they just created does not exist. Seeding from the
 * create response means the destination renders immediately, and the live signal
 * replaces the seed with the projected row a moment later.
 *
 * The entry is created if it does not exist yet, with no dependencies; the view
 * that mounts on it supplies its own `dependsOn` and fetcher, and skips its initial
 * fetch precisely because a value is already present.
 */
export function seedLiveResource<T>(key: string, value: T): void {
  const entry = entries.get(key) as Entry<T> | undefined;
  if (entry) {
    // Bumped so an in-flight fetch from before the seed cannot land on top of it.
    entry.gen += 1;
    entry.inFlight = undefined;
    entry.data.value = value;
    entry.error.value = undefined;
    entry.pending.value = false;
    return;
  }
  entries.set(key, {
    data: shallowRef<T | typeof EMPTY>(value),
    pending: ref(false),
    error: ref<unknown>(undefined),
    dependsOn: [],
    fetcher: async () => value,
    gen: 0,
    refs: 0,
  } as Entry);
}

function isNotFound(error: unknown): boolean {
  const status = (error as { response?: { status?: number }; status?: number } | null)?.response
    ?.status ?? (error as { status?: number } | null)?.status;
  return status === 404;
}

/**
 * Fetch into an entry, collapsing concurrent calls.
 *
 * A 404 (or a `deleted` signal) evicts rather than erroring: a resource that no
 * longer exists is not a failure the operator needs to see, and leaving a stale
 * value on screen would be worse.
 */
function revalidate(key: string, entry: Entry): Promise<void> {
  if (entry.inFlight) return entry.inFlight;

  const firstLoad = entry.data.value === EMPTY;
  if (firstLoad) entry.pending.value = true;

  const gen = entry.gen;
  /** Is this response still wanted? */
  const current = () => entries.get(key) === entry && entry.gen === gen;

  const run = entry
    .fetcher()
    .then((value) => {
      if (!current()) return;
      entry.data.value = value;
      entry.error.value = undefined;
    })
    .catch((error: unknown) => {
      if (!current()) return;
      if (isNotFound(error)) {
        entries.delete(key);
        return;
      }
      entry.error.value = error;
    })
    .finally(() => {
      if (entry.gen !== gen) return; // a newer generation owns the entry now
      entry.pending.value = false;
      entry.inFlight = undefined;
    });

  entry.inFlight = run;
  return run;
}

/** Apply a signal to every entry that declared a dependency on it. */
function handleSignal(signal: LiveSignal): void {
  if (isResync(signal)) {
    for (const [key, entry] of entries) void revalidate(key, entry);
    return;
  }

  if (!isEntityChanged(signal)) return;

  for (const [key, entry] of entries) {
    const affected = entry.dependsOn.some((dep) => signalMatches(signal, dep));
    if (!affected) continue;

    // A delete removes the instance-keyed entry outright; anything that merely
    // depends on the type (a list, a count) revalidates instead.
    if (signal.event === 'deleted' && signal.id !== undefined) {
      const instanceDep = `${signal.entity}:${signal.id}`;
      if (entry.dependsOn.includes(instanceDep) && entry.dependsOn.length === 1) {
        entries.delete(key);
        continue;
      }
    }

    void revalidate(key, entry);
  }
}

// A single subscription for the whole app: entries are matched here rather than
// each consumer subscribing separately.
bus.on('live', handleSignal);

export type UseLiveResourceOptions = {
  /**
   * What invalidates this resource: entity types (`'sos'`) and/or instances
   * (`'sos:123'`).
   *
   * **Required, even when empty.** A signal names one instance, but lists and
   * derived aggregates do not map to an id: a newly created entity has an id the
   * client has never seen, and a count has no id at all. A missing declaration
   * therefore fails silently — a figure that never updates, with no error
   * anywhere — so the decision is forced into every call site and into review.
   */
  dependsOn: LiveDependency[];

  /** Skip the initial fetch; caller triggers it via `refresh()`. */
  immediate?: boolean;
};

export type LiveResource<T> = {
  /** Cached value, or undefined when nothing is loaded yet. */
  data: Readonly<Ref<T | undefined>>;
  /** True only when there is no cached value to show. */
  pending: Readonly<Ref<boolean>>;
  error: Readonly<Ref<unknown>>;
  /** Force a revalidation. */
  refresh: () => Promise<void>;
  /** Replace the cached value locally (used by the optimistic helper, task 027). */
  set: (value: T) => void;
};

/**
 * Compose a cached, self-invalidating resource.
 *
 * Renders from cache immediately and revalidates in the background
 * (stale-while-revalidate); `pending` is true only when there is nothing to show,
 * so a revisited page never flashes a spinner.
 */
export function useLiveResource<T>(
  key: string,
  fetcher: () => Promise<T>,
  options: UseLiveResourceOptions,
): LiveResource<T> {
  let entry = entries.get(key) as Entry<T> | undefined;

  if (!entry) {
    entry = {
      data: shallowRef<T | typeof EMPTY>(EMPTY),
      pending: ref(false),
      error: ref<unknown>(undefined),
      dependsOn: [...options.dependsOn],
      fetcher: fetcher as () => Promise<unknown>,
      gen: 0,
      refs: 0,
    };
    entries.set(key, entry as Entry);
  } else {
    // Keep the newest fetcher and dependencies: a remounted view may close over
    // fresher props than the one that created the entry.
    entry.fetcher = fetcher as () => Promise<unknown>;
    entry.dependsOn = [...options.dependsOn];
  }

  const current = entry;
  current.refs += 1;
  onScopeDispose(() => {
    current.refs -= 1;
  });

  // Dev-only: warn if a declared dependency names an entity no wired consumer can
  // emit. A wrong token is otherwise invisible — the page looks live and simply
  // never updates — which is exactly how two of six tokens were wrong in task 037.
  validateDependencies(key, options.dependsOn);

  const hasValue = current.data.value !== EMPTY;
  if (options.immediate !== false && !hasValue && !current.inFlight) {
    void revalidate(key, current as Entry);
  }

  return {
    data: computed(() => (current.data.value === EMPTY ? undefined : (current.data.value as T))),
    pending: computed(() => current.pending.value),
    error: computed(() => current.error.value),
    refresh: () => revalidate(key, current as Entry),
    set: (value: T) => {
      current.data.value = value;
    },
  };
}
