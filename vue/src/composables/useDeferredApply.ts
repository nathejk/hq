import { ref, watch, type Ref } from 'vue'

/**
 * Apply incoming payloads, except while the operator is in the middle of something.
 *
 * Live updates and unsaved state are directly in conflict: a payload applied
 * while a form is half-typed or a drag half-finished destroys the operator's work
 * or their aim. The house rule is that such a page defers incoming payloads while
 * dirty and applies them when the edit ends (see `.rules`), which `KortView` and
 * `KlanListView` each solved with their own flag and their own "did anything
 * arrive?" bookkeeping.
 *
 * This is that pattern, once, so the awkward half is written and tested in one
 * place rather than re-derived per page.
 *
 * # Why the condition is watched, not the exits
 *
 * The obvious implementation calls an `applyIfDeferred()` helper from wherever an
 * edit finishes. That is what makes it fragile: a PrimeVue dialog closes via its
 * save button, its cancel button, the header ×, the Escape key and a click on the
 * mask, and a page with five dialogs has dozens of such paths. Missing one leaves
 * the screen permanently stale with no error anywhere — the exact failure mode the
 * live design exists to remove. Watching the *condition* cannot miss an exit,
 * because there is only one thing to observe.
 *
 * @param source  The cached payload, typically `data` from `useLiveResource`.
 *                `undefined` is ignored: it means "nothing loaded yet", not "apply
 *                emptiness".
 * @param paused  Whether applying must wait. Any reactive condition — a dialog
 *                being open, a drag in progress, a write in flight.
 * @param apply   Copies a payload into whatever the view renders. Called at most
 *                once per payload, and never while `paused`.
 */
export function useDeferredApply<T>(
  source: Ref<T | undefined>,
  paused: Ref<boolean>,
  apply: (value: T) => void,
): {
  /**
   * A payload arrived while paused and has not been applied yet.
   *
   * Exposed so the view can say so on screen: a page that has taught its operator
   * to trust it is live owes them a word the one time it is deliberately not.
   */
  updatesWaiting: Ref<boolean>
} {
  const updatesWaiting = ref(false)

  const sync = () => {
    const value = source.value
    if (value === undefined) return
    if (paused.value) {
      updatesWaiting.value = true
      return
    }
    updatesWaiting.value = false
    apply(value)
  }

  // `immediate` so a view that mounts on an already-cached value renders it in the
  // same tick, with no request and no empty frame. That is where the "instant" of
  // a warm cache actually comes from; waiting for the next payload would throw it
  // away.
  watch(source, sync, { immediate: true })

  // Note this reads `source.value` afresh rather than replaying what was held back:
  // several payloads may have arrived while paused, and only the newest is wanted.
  watch(paused, (now) => {
    if (!now && updatesWaiting.value) sync()
  })

  return { updatesWaiting }
}
