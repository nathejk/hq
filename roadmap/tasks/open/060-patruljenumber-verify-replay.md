# 060 — Verify number assignment survives replay, then ship PRD 003

**Status:** open
**Priority:** medium
**Created:** 2026-08-13
**Picked up by:**
**Started:**
**Completed:**

## Description

Closes PRD 003. The three preceding tasks are unit-tested in isolation; this one
verifies the properties that only hold when the pieces are assembled, then moves
the PRD to `done/`.

The success metrics from PRD 003 §9 are the checklist:

- Every patrulje that has paid for ≥3 seats has a non-empty `teamNumber`.
- Numbers are unique per year and never reused; each new assignment is strictly
  greater than every number previously assigned that year. Gaps are fine.
- Restarting the API emits no new `numberassigned` events for already-numbered
  teams, and changes no existing `teamNumber`.

The restart property is the one worth real effort, because it is the one that
degrades quietly and the one a unit test can only approximate: hq replays its
whole read model from the stream on every start, so a mistake here means numbers
churn on every deploy.

Prefer an assembled test over a manual click-through where possible — construct
the saga with fakes over the same wrapped-consumer path `main.go` uses
(`live.Notify`, not the bare saga), drive a replay of history followed by
`CaughtUp`, and assert nothing was published. That closes the loop on the
decorator forwarding `CaughtUp`, which no other test covers end to end.

Also confirm the no-reuse rule: cancelling an accepted patrulje's order leaves its
number in place and does not lower `maxNumber`, so the number is obsoleted rather
than recycled.

## Acceptance Criteria

- [ ] Assembled test: replaying history that already contains `numberassigned`
      for a team, then going live, publishes nothing for that team.
- [ ] Assembled test: the saga is driven through `live.Notify` (as `main.go`
      wires it) and still goes live — i.e. `CaughtUp` reaches it through the
      wrapper.
- [ ] Numbers issued in one process are strictly increasing and distinct.
- [ ] Cancellation after acceptance: number retained, `maxNumber` unchanged, no
      new event.
- [ ] Verified against a running stack (`docker compose up`) that a paid patrulje
      receives a number and that restarting the API neither renumbers it nor
      re-emits the event; result recorded in this task's log.
- [ ] PRD 003 requirement checkboxes (§6) ticked, `Status: done`, `Shipped` set,
      file moved to `roadmap/prd/done/`.

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-13 17:05 — Task created from PRD 003 (approved same day).
