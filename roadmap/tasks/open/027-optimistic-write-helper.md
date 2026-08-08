# 027 — Optimistic write helper

**Status:** open
**Priority:** medium
**Created:** 2026-08-08
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] A helper applies an optimistic value to a cache entry and issues the write
- [ ] On success the value is reconciled from the server, not left as the optimistic
      guess
- [ ] On failure the previous value is restored **and** the failure is surfaced
- [ ] A signal arriving for the operator's own write does not cause a visible flicker
- [ ] Two rapid writes to one key do not leave the cache holding the older result
- [ ] No new `vue-tsc` errors in touched files; `npm run lint` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-08 15:53 — Task created. Depends on 024. Last of PRD 004 Phase 1.
