# 065 — Revive spejderstatus as the canonical member projection

**Status:** done
**Priority:** high
**Created:** 2026-08-17
**Picked up by:** agent session
**Started:** 2026-08-17
**Completed:** 2026-08-17

## Description

From **PRD 006** §8. The foundation task: everything else in the PRD reads what this
writes.

`go/nathejk/table/spejderstatus.go` is **inert** — `Consumes()` returns an empty slice
and the whole of `HandleMessage` is commented out. Promote it from a single file in the
root `table` package to a package at `go/nathejk/table/spejderstatus/`, written to
shared-go's guidelines so it can later be lifted unchanged (`table.go`, `consumer.go`,
`querier.go`, `commands.go`, `repository.go`, `interfaces.go`, `table.sql`, following
`shared-go/tables/signup` as the layout reference).

`shared-go/types/member.go` explicitly names this projection as the home of member
status ("these strings live in the spejderstatus projection"), which settles the question
of folding it into `spejder`: keep the projection, keep the name.

**Schema** (the struct at `spejderstatus.go:13-18` already declares the right shape, the
`.sql` does not):

- `id`, `year`, `status`, `updatedAt` — exist today
- add `initialTeamId`, `currentTeamId`
- add an index on `(year, currentTeamId)` — every membership query needs it

**What it consumes:**

- `NATHEJK.*.patrulje.*.started` → `messages.NathejkTeamStarted`. For each member in
  `body.Members`, write `status = racing`, `initialTeamId = currentTeamId = teamId`.
  This is where `racing` comes from — **derived from an existing event, with no new
  producer.** `body.Members` is precisely the members who actually started, which is the
  same source `table/patrulje/consumer.go:66` already uses for `memberCount`, so the
  3-member check stays consistent with the patrol's own count instead of inventing a
  second definition.
- `NATHEJK.*.spejder.*.deleted` — `commands.Team.StartPatrulje`
  (`go/nathejk/commands/team.go:116`) publishes this for every member who did **not**
  start. A projection that ignores it holds rows for no-shows and over-counts strength.
- The member lifecycle events from task 063.

## Notes

- **Take the year from the subject, not `msg.Time()`.** The body has no year field, and
  the old commented-out code used `msg.Time().Year()`, which breaks the moment history is
  replayed across a year boundary. `table/sos/consumer.go:290-294` already documents this
  exact trap and is the reference implementation.
- Normalise legacy values on the way in using task 064's shared mapping. Do not
  hand-roll it here.
- **Reuse the lifecycle helpers rather than re-deriving them**: `Valid()` gates what the
  API accepts, `CanFinish()` guards a finish flow that does not exist yet, `InOurCare()`
  *is* the in-our-care count. No hand-rolled status lists anywhere.
- `registered` and `seated` are out of scope: the SOS panel only ever sees members who
  have started, and which flows own those two is an open question in PRD 006 §11.
- `CREATE TABLE IF NOT EXISTS` never alters an existing table, so **the table must be
  dropped in dev** for the new columns to appear. Same class of drift as task 038.
- Must be replayable and idempotent — upserts only.
- **No imports from `nathejk.dk/...`** (lift-readiness, task 081 asserts it).
- Add it to the `projections` slice in `cmd/api/main.go`, not just to the mux: only that
  slice is wrapped by `live.NotifyAll`, and a consumer added elsewhere silently emits no
  live signals. Delete the six-line inline `table.NewSpejderStatus(writer)` shim it
  replaces.

## Acceptance Criteria

- [x] Package at `go/nathejk/table/spejderstatus/` following the shared-go layout
- [x] `table.sql` gains `initialTeamId`, `currentTeamId` and the `(year, currentTeamId)` index
- [x] `racing` derived from `patrulje.*.started` for every member in `body.Members`
- [x] `spejder.*.deleted` removes the row
- [x] Legacy status values normalised via task 064's mapping
- [x] Year taken from the subject; a test covers a cross-year replay
- [x] Replay is idempotent (a test handles the same message twice)
- [x] Wired into the `projections` slice in `cmd/api/main.go`; the old inline shim deleted
- [x] No `nathejk.dk/...` import in the package
- [x] `go build ./... && go vet ./...` and `gofmt -l` clean
- [x] Verified against the real stream: row count and per-team totals match
      `patrulje.memberCount`

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006. Schema and consumer kept in one task deliberately:
  PRD 006 §10 listed them separately, but a consumer that writes columns another task has
  not added yet cannot be verified in isolation, so splitting would only produce a
  knowingly-broken intermediate state.
- 2026-08-17 — Picked up. `table.sql`, `table.go`, `consumer.go`, `querier.go` written to
  the shared-go layout; old `table/spejderstatus.{go,sql}` deleted and the inline
  `table.NewSpejderStatus(writer)` shim in `main.go` replaced by the package inside the
  `projections` slice (so `live.NotifyAll` wraps it and member changes emit signals).
- 2026-08-17 — **The upsert takes an explicit list of columns it may overwrite**, rather
  than updating everything. This is the design decision the projection rests on and it is
  not obvious: the start event is replayed on every restart and legitimately knows a
  member's *initial* team, but nothing about where they are now. If its upsert refreshed
  `currentTeamId`, **every restart would silently undo every move made during the event**,
  leaving two patrols with wrong strengths and no trace of why. So the start event updates
  `initialTeamId` only, and the move updates `currentTeamId` only. Both directions have a
  test, because neither is visible at the call site.
- 2026-08-17 — Statuses are written via the body's `Status()` (the `MemberEvent` interface
  from task 063) rather than named per case, so all six status-setting events share one
  write path and the three the car and shelter interfaces will publish need no new code —
  only a subject. Subscribed to those three already: being ready for them is cheaper than
  revisiting this file when they ship.
- 2026-08-17 — An unrecognised status is **refused and logged, not returned as an error**.
  Erroring would let one poison message wedge the replay that rebuilds the entire table;
  writing it would put a value in the read model that neither `InOurCare()` nor a strength
  query counts, so a member would vanish from the number that has to reach zero. Refusing
  is the only option that fails safe in both directions.
- 2026-08-17 — Events carry `teamId` so a lifecycle event alone can create a row — for a
  member whose start this projection never saw, because history was truncated or the car
  interface got there first. Without it such a member would have a status and no team, and
  would be invisible to every per-team query.
- 2026-08-17 — 12 tests, including one asserting **every subject declared in `Consumes()`
  is actually handled**. That pair drifts apart silently: a declared-but-unmatched subject
  looks exactly like an event nobody publishes. It caught a real gap while writing it.
- 2026-08-17 — Hit the documented `CREATE TABLE IF NOT EXISTS` trap on the dev stack
  exactly as the notes predicted: the existing table survived and every insert failed with
  `Unknown column 'currentTeamId'`. Confirms the note is worth keeping — the projection
  logic was already correct, and the log showed it deriving `racing` from real 2025 events
  while writing nothing. Dropped the table and restarted to force a full replay.
- 2026-08-17 — ✅ **Verified against the real stream.** 686 rows projected for 2025, against
  `SUM(memberCount)` of 686 over started patrols — and **zero teams** where the projected
  count disagrees with the patrol's own `memberCount`. That is the criterion that mattered:
  the two are derived from the same event, so any disagreement would have meant the
  projection was reading `body.Members` differently from the patrulje consumer.
- 2026-08-17 — Also updated a stale comment in `internal/live/entities_test.go` that cited
  `table/spejderstatus.go` returning an empty subject list — that file is gone.
- 2026-08-17 — ✅ All criteria met. Full `go build ./...`, `go vet ./...`, `gofmt -l` and
  `go test ./...` clean. Moving to done.
