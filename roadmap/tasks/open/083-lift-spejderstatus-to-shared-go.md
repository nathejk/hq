# 083 — Lift spejderstatus to shared-go

**Status:** open
**Priority:** low
**Created:** 2026-08-17
**Picked up by:**
**Started:**
**Completed:**

## Description

From **PRD 006** §10. **Post-stabilisation follow-up — do not start early.**

Move `go/nathejk/table/spejderstatus/` to `shared-go/tables/spejderstatus/` and consume it
from hq, as task 055 does for the SOS package.

The point of the lift is that the schema has **stopped changing**. Doing it while the member
lifecycle is still settling means paying the cross-repo cost on every adjustment, which is why
task 055 was explicitly deferred and why this one is too.

Handlers stay in hq permanently; only the table package moves.

## Notes

- Prerequisites, all of which are about stability rather than completeness:
  - `spejderstatus` schema unchanged for a while
  - the member events (task 063) settled in shared-go
  - task 081's lift-readiness test green, which should make this a file move
- **`shared-go/tables/spejder/GetInactive` becomes revivable at the same time.** It is
  commented out at `tables/spejder/querier.go:90-93` with the note "Re-enable it once
  spejderstatus (or a status column on spejder) lives here" — this task is what satisfies that
  condition. Its doc comment also records a bug to fix when reviving it: *"the query has two
  placeholders but passes four args."*
- That in turn unblocks the **Udgået** page: `vue/src/components/Navigation.vue:160` links to
  `/ude`, a route that does not exist in the router. Out of scope for PRD 006 (§4) and still
  formally an open question there — but this is the task after which building it is trivial, so
  raise it here rather than losing it.
- Coordinate with task 055 (lift SOS) — same pattern, and doing them together may be cheaper
  than twice.

## Acceptance Criteria

- [ ] `spejderstatus` lives in `shared-go/tables/`
- [ ] hq consumes it with no local copy left behind
- [ ] hq's `go.mod` pin updated; build, vet and tests clean
- [ ] Replay against the real stream produces identical rows to before the lift
- [ ] `spejder.GetInactive` revived (with its placeholder/arg-count bug fixed) or a note
      recording why not
- [ ] Follow-up raised for the `/ude` Udgået page

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006 as a deliberate follow-up, not part of the delivery.
