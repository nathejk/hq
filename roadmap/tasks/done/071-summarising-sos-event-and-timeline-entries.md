# 071 — Summarising sos event per operation, and its timeline entries

**Status:** done
**Priority:** high
**Created:** 2026-08-17
**Picked up by:** agent session
**Started:** 2026-08-17
**Completed:** 2026-08-17

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

- [x] Summarising event bodies defined in `table/sos/messages.go`, one per operation kind
- [x] Payload self-contained: actor, affected members, from/to status, resulting strength
- [x] `table/sos/consumer.go` appends one `sos_activity` row per summary and advances
      `lastActivityAt`
- [x] New `type` values only — no `sos_activity` schema change
- [x] Replay idempotent (test handles the same message twice)
- [x] Consumer tests cover each new subject
- [x] No `nathejk.dk/...` import in the `sos` package (its `lift_test.go` still passes)
- [x] `go test ./nathejk/table/sos/`, `go vet`, `gofmt -l` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006. Write-side bodies and read-side projection kept in one
  task: they are the same package, and an event nothing projects is not verifiable.
- 2026-08-17 — Picked up. Three bodies (`MemberStatusChanged`, `TeamCollected`,
  `MembersMoved`), three `ActivityType` values, three consumer cases, three subjects.
- 2026-08-17 — **The summary is stored as JSON in the existing `value TEXT` column**, via a
  new `appendSummary` helper. The existing entries store a bare string because one event
  changed one thing; a member operation changes several members and the line has to name
  them. Marshalling keeps `sos_activity` a log of *what happened* rather than a widening
  union of every event's fields — which is what PRD 001 meant by requiring the table to be
  extensible without a schema change. Confirmed: no migration, only new `type` values.
- 2026-08-17 — `TeamCollected` is a **separate type from `MemberStatusChanged` despite an
  identical shape**, because it is a distinct act: the operator decided the patrol is done,
  not that three individuals each wanted to stop. Collapsing them would save a struct and
  lose the only thing the timeline needed to say.
- 2026-08-17 — `MembersMoved` carries the destination **per member**, not once per operation.
  Two survivors may go to two different patrols; making that representable in one entry is
  what stops the flow flattening it later. There is a test.
- 2026-08-17 — A marshalling failure logs and skips the entry rather than returning an
  error. The event is already in the log and cannot be un-published, so returning an error
  would wedge the replay on **every** restart from then on. A missing timeline line is
  recoverable; a projection that cannot finish is not.
- 2026-08-17 — Summaries `touch` the case rather than `update` it: a member changing status
  is activity *on* a case, not a change *to* it. But it must still advance
  `lastActivityAt`, or the list would show a case as untouched while somebody was leaving
  the race from it.
- 2026-08-17 — 4 new tests. The two that carry weight: one operation touching three members
  produces **exactly one** timeline entry, and the stored summary contains every field
  needed to render the line (names, from/to, strength). The second is the anti-regression
  for the subtlest property here — storing ids and joining at render time would describe a
  member's *first* move using their *second* team, i.e. an entry that changes meaning after
  the fact.
- 2026-08-17 — ✅ All criteria met. Full `go build`, `go vet`, `gofmt -l`, `go test ./...`
  clean; the sos package's own lift guard still passes. Moving to done.
