# 071 — Summarising sos event per operation, and its timeline entries

**Status:** open
**Priority:** high
**Created:** 2026-08-17
**Picked up by:**
**Started:**
**Completed:**

## Description

From **PRD 006** §8 ("Every operation is N member events plus one case event") and §11
Decisions. This task implements the case-facing half of that rule.

**The rule:** a member-changing operation publishes **one event on the `spejder` entity per
updated member** (task 063), and then **one event on the `sos` entity summarising the whole
operation**. Uniformly — a single member going `waiting` is one plus one, exactly like a
three-member collection is three plus one. No special case for the single-member path,
which is what stops the timeline and the projection disagreeing about what an "operation"
is.

This task adds:

1. **The summarising event bodies** in `go/nathejk/table/sos/messages.go`, on
   `NATHEJK.{year}.sos.{id}.*` — one per kind of operation (member status changed, members
   moved, team collected).
2. **Handling in `table/sos/consumer.go`** so each appends one `sos_activity` row and
   advances `lastActivityAt`, exactly as the twelve existing SOS events do.
3. **New `type` values** for `sos_activity`. No schema change: PRD 001 built that table to
   be extensible for precisely this (`type VARCHAR(49)`, `value TEXT`).

**Why a case event at all,** rather than the sos consumer subscribing to member subjects:
it keeps `sosId` off the member events, which is what lets the future car and shelter
interfaces publish the same member events without knowing about cases. It also means this
consumer needs no new subject pattern and no knowledge of the member domain.

## Notes

- **Published last.** The summary comes after the member events it describes, so anything
  reading it is guaranteed those changes are already in the log.
- **The payload must be self-contained** — who changed, from what to what, and the team's
  resulting strength — enough to render the timeline line without joining to current state.
  A line that re-derives its text from today's member rows shows today's truth on
  yesterday's entry, which is the one thing a handover log cannot do.
- One row per operation. That is what makes "hele patruljen hentes" read as a single line
  while three members changed status. `sos_activity` is keyed by the event's stream
  sequence, so N member events would otherwise have produced N rows.
- `sosId` is a parameter of the **command**, not a field on the member event: required for
  the withdrawal request, optional for the override and the move. When absent, the member
  events are published and the summary simply is not.
- Live signals: N+1 events would be N+1 signals, but the hub coalesces within
  `DefaultCoalesceWindow` (75ms, `internal/live/hub.go:12-22`) — whose comment cites mass
  operations as the reason it exists — so an operation emits effectively one `spejder` and
  one `sos` signal.
- No exception entry type: exceptions do not exist (PRD 006 §11 Decisions). No
  team-discontinued entry either: discontinuation has no event, so what the timeline shows
  is the operation that took the count to zero.
- The consumer already logs unknown subjects as a no-op rather than erroring (task 043),
  so partial deployment is safe in either order.

## Acceptance Criteria

- [ ] Summarising event bodies defined in `table/sos/messages.go`, one per operation kind
- [ ] Payload self-contained: actor, affected members, from/to status, resulting strength
- [ ] `table/sos/consumer.go` appends one `sos_activity` row per summary and advances
      `lastActivityAt`
- [ ] New `type` values only — no `sos_activity` schema change
- [ ] Replay idempotent (test handles the same message twice)
- [ ] Consumer tests cover each new subject
- [ ] No `nathejk.dk/...` import in the `sos` package (its `lift_test.go` still passes)
- [ ] `go test ./nathejk/table/sos/`, `go vet`, `gofmt -l` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006. Write-side bodies and read-side projection kept in one
  task: they are the same package, and an event nothing projects is not verifiable.
