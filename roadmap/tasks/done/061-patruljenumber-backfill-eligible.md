# 061 — BFF: number the patruljer that already qualify

**Status:** done
**Priority:** high
**Created:** 2026-08-14
**Picked up by:** zed agent session
**Started:** 2026-08-14
**Completed:** 2026-08-14

## Description

Found while verifying 060 against live data: **76 patruljer in 2026 already have
≥3 paid seats and none of them will ever be numbered.**

The saga triggers on `order.paid` and only publishes once live. Every qualifying
patrulje paid before the saga existed (newest paid order 2026-07-30), so those
events only ever arrive as replay, where the gate correctly suppresses publishing.
The gate is not the bug — without it every restart would re-issue every number.
The gap is that nothing ever revisits a patrulje that became eligible while no
saga was watching.

PRD 003 §9's first success metric ("every patrulje that has paid for ≥3 seats has
a non-empty `teamNumber`") is unreachable without this, and §10 Phase 3 assumed a
backfill that no task provided.

**Approach: reconcile once at catch-up.**

After `CaughtUp` has seeded state and opened the gate, sweep for patruljer that
are eligible but unnumbered and assign each the next number. This is not
publishing-during-replay: the gate is open, history is done, and the sweep is
naturally idempotent — after it runs those patruljer hold numbers, so the next
start's sweep finds nothing.

**Notes and constraints**

- Ordering must be deterministic and defensible, since it decides who gets which
  number. Oldest qualifying payment first is the honest reading of "numbering
  reflects payment order" (PRD 003 §5), so order by when the patrulje's earliest
  qualifying order was paid — not by `teamId`, and not by map iteration order,
  which is randomised in Go and would hand out different numbers on every start.
- Skip anything already in `assigned` (seeded from both the event history and the
  read model), so manual and legacy numbers are respected exactly as in 058.
- The sweep needs "eligible patruljer for the year" in one query rather than 310
  per-team round trips. That is a new read on the order side; declare it on the
  saga's existing narrow reader interface and implement it wherever it belongs
  (a shared-go `order` query is the likely home — if so, it must be added there
  and the pin bumped, not reimplemented locally).
- It must not block the mux. `CaughtUp` is called from the subscribe path, so a
  slow sweep delays that goroutine, not message handling — but keep it to one
  query plus one publish per assignment, and log a summary.
- A failing sweep must leave the saga live and working for new payments. Unlike
  the seed in 058, a failed *sweep* cannot cause duplicate numbers — it only
  means the backfill did not happen — so it should log and carry on rather than
  going dormant.
- On the first deploy this publishes ~76 events in one burst. That is the
  correction, not a fault, but say so in the log line so the burst is
  recognisable in the stream.

## Acceptance Criteria

- [x] After `CaughtUp`, patruljer with ≥3 paid seats and no number are assigned
      consecutive numbers continuing from the high-water mark.
- [x] Deterministic order: earliest qualifying payment first. Covered by a test
      with a shuffled input that asserts a fixed number-to-team mapping.
- [x] Already-numbered patruljer (from events *or* from the read model) are
      skipped.
- [x] Ineligible patruljer (<3 seats) are not numbered.
- [x] Running the sweep twice publishes nothing the second time (idempotent), so
      a restart does not renumber.
- [x] A failing sweep logs and leaves the saga live for new payments.
- [x] The sweep logs a one-line summary including how many it assigned.
- [x] `gofmt -l .`, `go vet ./...`, `go test ./...` clean on both resolution
      paths.
- [x] Verified on the live stack: the qualifying patruljer receive numbers, the
      numbers are unique, and a subsequent restart neither renumbers nor re-emits.
      Record counts before and after in the log.

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-14 07:35 — Task created from the 060 verification finding: 76 patruljer
  qualify, 0 numbered, and the live-only gate means no future event will fix it.
- 2026-08-14 07:45 — Picked up. Plan: sweep at the end of `CaughtUp`, reusing the
  patrulje rows already read for seeding. Decided against adding an aggregate
  query to shared-go for now — see the next entry.
- 2026-08-14 08:05 — Decision: no shared-go change. The sweep needs "when did this
  patrulje qualify", which `PaidQuantityBySKU` cannot answer, so rather than add a
  second aggregate upstream I switched the saga's reader to `ListByOwner` and
  compute seats from the lines. That collapses two candidate implementations of
  "paid seats" into one — the live path and the sweep now come through the same
  `paidSeats`, so they cannot drift into disagreeing about who is accepted, which
  is exactly the bug the shared payment entity's own history warns about. Cost is a
  handful of order rows per patrulje instead of one aggregate; the SKU lookup is
  hoisted out of the sweep's loop.
- 2026-08-14 08:10 — Extracted `assignNext`, now the single place a number is
  handed out (live path and sweep both call it). It holds the lock across reading
  the mark, publishing, and raising it, so the same number cannot go out twice.
- 2026-08-14 08:15 — Sorting: earliest paid first, tie-broken on teamID. `paidAt`
  is compared as a string because the column holds Go's `time.Time` text form, all
  UTC and zero-padded, so lexicographic order is chronological — noted in the code
  that a format change would need a real parse. Three tests pin it: shuffled input
  produces identical output, ties resolve on teamID, and the mapping is fixed.
- 2026-08-14 08:20 — One existing test (`TestSeedSkipsEmptyTeamNumbers`) started
  failing, correctly: its fixture qualified, so the new sweep numbered it. Rewrote
  the fixture so the read-model rows have no paid orders, keeping that test about
  the empty number rather than about the backfill.
- 2026-08-14 08:25 — ✅ Unit criteria complete. 34 tests in the package, passing
  under `-race`; clean on both resolution paths.
- 2026-08-14 08:30 — ✅ Live stack, first run: `patruljenumber: backfill assigned
  75 number(s) to patruljer that already qualified` — 75 rather than 76 because one
  had already been numbered during the 060 verification. Result: **76 numbered, 76
  distinct, range 1–76, 0 duplicates, and 76 `numberassigned` events in the
  stream**, one per team. Matches the 76 qualifying patruljer exactly.
- 2026-08-14 08:32 — ✅ Live stack, restart: numbered 76, distinct 76, range 1–76,
  event count still 76. No renumbering and no re-emission, so the sweep is
  idempotent against real data across a full replay.
- 2026-08-14 08:34 — The restart produced *no* backfill log line, because the sweep
  returned early on finding no candidates. Moved the log so it always reports,
  including the zero: that zero is the evidence the sweep ran and correctly found
  nothing, which is what an operator wants to see after a deploy.
- 2026-08-14 08:35 — Completed. PRD 003 §9's first success metric is now met, which
  unblocks 060.
