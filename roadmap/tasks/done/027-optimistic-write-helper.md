# 027 — Optimistic write helper

**Status:** done
**Priority:** medium
**Created:** 2026-08-08
**Picked up by:** agent session (zed)
**Started:** 2026-08-08
**Completed:** 2026-08-08

## Description

PRD 004 requires that an operator's own action feels immediate: *"The operator's own
action must never wait on a round trip."* This matters most for PRD 001, where the
operator is typing a comment while on the phone. Depends on 024.

Apply the change to the cache immediately, issue the write, then reconcile:

- **Success:** the authoritative value arrives via the signal that follows the write
  (or the revalidation it triggers), replacing the optimistic value.
- **Failure:** roll back to the previous value and surface the error. A silent
  rollback would be worse than no optimism at all — the operator would believe
  something was recorded when it was not.

### Notes

- Reconciliation must be idempotent: the signal for the operator's own write will
  arrive too, so applying it after the optimistic update must be a no-op rather than a
  flicker.
- Errors already surface as toasts via `bus` from the axios plugin; do not build a
  second error channel.
- Keep the helper small — the interesting behaviour is rollback and reconciliation
  order, not a general mutation framework.

## Acceptance Criteria

- [x] A helper applies an optimistic value to a cache entry and issues the write
- [x] On success the value is reconciled from the server, not left as the optimistic
      guess
- [x] On failure the previous value is restored **and** the failure is surfaced
- [x] A signal arriving for the operator's own write does not cause a visible flicker
- [x] Two rapid writes to one key do not leave the cache holding the older result
- [x] No new `vue-tsc` errors in touched files; `npm run lint` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-08 15:53 — Task created. Depends on 024. Last of PRD 004 Phase 1.
- 2026-08-08 18:20 — Picked up. Plan: `optimisticWrite(resource, apply, write)` — snapshot,
  apply locally, issue the write, then revalidate from the server; restore the snapshot
  and rethrow on failure. Tests cover rollback, reconciliation and out-of-order writes.
- 2026-08-08 18:26 — Added `composables/optimisticWrite.ts`. Decisions:
  • **Revalidate by default** after a successful write, rather than trusting the
    optimistic value. If the server normalises anything — a phone number, a generated
    id, a derived status — keeping the guess would leave the screen quietly wrong.
    `{ revalidate: false }` is available for writes whose result cannot differ.
  • **Rethrow on failure** rather than swallowing. A silent rollback would leave the
    operator believing something was recorded when it was not — on a dispatch desk that
    is worse than not being optimistic at all. Errors already reach the operator as
    toasts via the axios plugin's bus, so no second error channel is built here.
  • **Roll back only if our optimistic value is still the current one.** If a signal
    for someone else's change, or a year flush, already replaced it, clobbering that
    with a stale snapshot would be worse than leaving it. Covered by a test.
  • Accepts an updater function so callers can append to a list without re-reading.
- 2026-08-08 18:32 — ✅ 7 tests, all passing: immediate visibility before the write
  resolves, reconciliation replacing the guess with the server's value, rollback +
  rethrow, updater form, no flicker when the operator's own signal races the
  revalidation, newer values left alone on rollback, and last-write-wins for rapid
  successive writes.
- 2026-08-08 18:34 — Completed. 28 tests green across the four Phase 1 modules;
  `vue-tsc` 107 vs the 109 baseline, none in live-update files; lint clean.
  **PRD 004 Phase 1 is complete.**
