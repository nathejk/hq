import type { LiveResource } from '@/composables/useLiveResource';

/**
 * Apply an operator's own change immediately, then reconcile with the server.
 *
 * PRD 004: "the operator's own action must never wait on a round trip". This
 * matters most on the nødtelefon screen, where someone is typing while on the
 * phone.
 *
 * The sequence is deliberate:
 *
 *  1. snapshot the current value
 *  2. apply the optimistic value so the UI updates in the same frame
 *  3. issue the write
 *  4. on success, revalidate — the server's value replaces the guess, so a
 *     server-side transformation (a normalised phone number, a generated id, a
 *     computed status) is never silently missing from the screen
 *  5. on failure, restore the snapshot **and** rethrow
 *
 * Step 5 is the one that matters: a silent rollback would leave the operator
 * believing something was recorded when it was not, which on a dispatch desk is
 * worse than never having been optimistic at all. Errors already surface as
 * toasts via the axios plugin's bus, so rethrowing is enough — this helper
 * deliberately does not build a second error channel.
 */
export type OptimisticWriteOptions = {
  /**
   * Skip the post-write revalidation.
   *
   * Only for writes whose response cannot differ from the optimistic value.
   * Default is to revalidate, because assuming the server agreed with you is how
   * caches drift.
   */
  revalidate?: boolean;
};

export async function optimisticWrite<T, R>(
  resource: LiveResource<T>,
  next: T | ((current: T | undefined) => T),
  write: () => Promise<R>,
  options: OptimisticWriteOptions = {},
): Promise<R> {
  const snapshot = resource.data.value;
  const optimistic =
    typeof next === 'function' ? (next as (current: T | undefined) => T)(snapshot) : next;

  resource.set(optimistic);

  try {
    const result = await write();

    if (options.revalidate !== false) {
      // Not awaited on the caller's behalf beyond this point: the value is
      // already correct enough to show, and the revalidation replaces it with
      // the authoritative one. Awaiting keeps the helper's contract simple —
      // when it resolves, the cache holds the server's value.
      await resource.refresh();
    }

    return result;
  } catch (error) {
    // Restore only if nothing else has moved on in the meantime. A signal for
    // someone else's change, or a year flush, may have already replaced the
    // value; clobbering that with our stale snapshot would be worse than
    // leaving it.
    if (resource.data.value === optimistic) {
      if (snapshot === undefined) {
        // There was nothing cached before: drop back to "nothing" by
        // revalidating rather than inventing a value.
        void resource.refresh();
      } else {
        resource.set(snapshot);
      }
    }
    throw error;
  }
}
