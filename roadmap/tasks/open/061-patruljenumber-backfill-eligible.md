# 061 — BFF: number the patruljer that already qualify

**Status:** open
**Priority:** high
**Created:** 2026-08-14
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] After `CaughtUp`, patruljer with ≥3 paid seats and no number are assigned
      consecutive numbers continuing from the high-water mark.
- [ ] Deterministic order: earliest qualifying payment first. Covered by a test
      with a shuffled input that asserts a fixed number-to-team mapping.
- [ ] Already-numbered patruljer (from events *or* from the read model) are
      skipped.
- [ ] Ineligible patruljer (<3 seats) are not numbered.
- [ ] Running the sweep twice publishes nothing the second time (idempotent), so
      a restart does not renumber.
- [ ] A failing sweep logs and leaves the saga live for new payments.
- [ ] The sweep logs a one-line summary including how many it assigned.
- [ ] `gofmt -l .`, `go vet ./...`, `go test ./...` clean on both resolution
      paths.
- [ ] Verified on the live stack: the qualifying patruljer receive numbers, the
      numbers are unique, and a subsequent restart neither renumbers nor re-emits.
      Record counts before and after in the log.

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-14 07:35 — Task created from the 060 verification finding: 76 patruljer
  qualify, 0 numbered, and the live-only gate means no future event will fix it.
