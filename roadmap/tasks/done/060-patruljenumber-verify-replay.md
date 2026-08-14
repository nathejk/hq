# 060 — Verify number assignment survives replay, then ship PRD 003

**Status:** done
**Priority:** medium
**Created:** 2026-08-13
**Picked up by:** zed agent session
**Started:** 2026-08-13
**Completed:** 2026-08-14

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

- [x] Assembled test: replaying history that already contains `numberassigned`
      for a team, then going live, publishes nothing for that team.
- [x] Assembled test: the saga is driven through `live.Notify` (as `main.go`
      wires it) and still goes live — i.e. `CaughtUp` reaches it through the
      wrapper.
- [x] Numbers issued in one process are strictly increasing and distinct.
- [x] Cancellation after acceptance: number retained, `maxNumber` unchanged, no
      new event.
- [x] Verified against a running stack (`docker compose up`) that a paid patrulje
      receives a number and that restarting the API neither renumbers it nor
      re-emits the event; result recorded in this task's log.
- [x] PRD 003 requirement checkboxes (§6) ticked, `Status: done`, `Shipped` set,
      file moved to `roadmap/prd/done/`.

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-13 17:05 — Task created from PRD 003 (approved same day).
- 2026-08-13 19:00 — Picked up. Plan: the assembled tests belong in
  `internal/live` rather than the saga package, because what they exercise is the
  seam between the two — and `nathejk/table/` may not import `internal/`. Write
  them against a fake catch-up-aware consumer plus the real saga where possible.
- 2026-08-13 19:15 — Put the assembled tests in `cmd/api` rather than
  `internal/live` after all: the composition root is the only package that
  imports both, and what is being verified is precisely what `main()` builds.
  (`nathejk/table/` may not import `internal/`, so the saga package could not host
  them.) Four tests written against the real saga wrapped in the real
  `live.Notify`.
- 2026-08-13 19:20 — ✅ Four assembled criteria complete. `goLive()` in the test
  does what the jetstream path does — assert the optional interface on the handler
  it was given and call it — so a dropped `CaughtUp` fails loudly instead of
  producing a saga that silently never assigns.
- 2026-08-13 19:25 — Checked the tests are load-bearing rather than
  self-satisfying: temporarily reverted `live.Notify` to return the plain
  notifier, and all four failed with "the wrapped consumer does not advertise
  CaughtUp". Restored (byte-identical, confirmed with git diff) and green again.
- 2026-08-14 07:25 — Live stack: the running `hq-api-1` picked the saga up on
  hot-reload. Advertised entity set unchanged and contains both `order` and
  `patrulje`, confirming 059's criterion against the real process rather than by
  reasoning.
- 2026-08-14 07:28 — ✅ End-to-end on real infrastructure. Published a live
  `NATHEJK.2026.order.<id>.paid` into the dev stream for an already-paid,
  6-seat, unnumbered patrulje (Skjoldungerne 22). It was assigned `teamNumber=1`
  and the stream holds exactly one `numberassigned` for it. This is the one thing
  no unit test can prove: that `CaughtUp` really does fire from the jetstream
  subscribe path *through* the decorator. Note the trigger was synthetic — a
  re-publish of an existing paid order, not an organic new payment — but the whole
  delivery path (jetstream → wrapper → gate → saga → publish → projector → DB)
  ran for real. Harmless to the order itself: the projector's paid UPDATE is
  conditional on status='open', so the duplicate was a no-op.
- 2026-08-14 07:32 — ✅ Restart property verified on the live stack: restarted
  `hq-api-1`, which replays the entire log. `teamNumber` stayed `1`, the
  `numberassigned` count for that subject stayed `1`, and the total numbered
  count stayed `1`. No renumbering, no duplicate emit — PRD 003 §9's third metric.
- 2026-08-14 07:35 — 🚧 Blocker for shipping the PRD. The live data says
  **76 patruljer already qualify** (≥3 paid seats, 2026) and all but the one I
  triggered are unnumbered. They never will be: every one of their `order.paid`
  events is historical (newest 2026-07-30), and the live-only gate means replay
  publishes nothing. The gate is right — without it every restart re-issues every
  number — but it leaves PRD 003 §9's *first* metric ("every patrulje that has
  paid for ≥3 seats has a non-empty teamNumber") unreachable, and §10 Phase 3
  assumed a backfill that no task provides.
  Created **061** for a reconciliation sweep at catch-up. Deliberately not folded
  into this task: it publishes for real against production data and deserves its
  own review and its own commit. This task stays in `doing/` until 061 lands,
  since shipping the PRD with an unmet success metric would be a lie in the board.
- 2026-08-14 08:36 — Unblocked: 061 shipped the catch-up sweep, and all 76
  qualifying patruljer are numbered 1–76 with one event each. PRD 003 §9's three
  success metrics are now all met and all three were checked against the live
  stack rather than only in tests.
- 2026-08-14 08:40 — ✅ Final criterion complete. PRD 003 §6 requirements ticked,
  Status set to done, Shipped dated, moved to roadmap/prd/done/.
- 2026-08-14 08:40 — Completed. Automatic patrulje numbering is shipped: saga
  (057), seeding (058), wiring (059), verification (060), backfill (061).
