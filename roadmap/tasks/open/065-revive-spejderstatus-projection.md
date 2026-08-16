# 065 — Revive spejderstatus as the canonical member projection

**Status:** open
**Priority:** high
**Created:** 2026-08-17
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] Package at `go/nathejk/table/spejderstatus/` following the shared-go layout
- [ ] `table.sql` gains `initialTeamId`, `currentTeamId` and the `(year, currentTeamId)` index
- [ ] `racing` derived from `patrulje.*.started` for every member in `body.Members`
- [ ] `spejder.*.deleted` removes the row
- [ ] Legacy status values normalised via task 064's mapping
- [ ] Year taken from the subject; a test covers a cross-year replay
- [ ] Replay is idempotent (a test handles the same message twice)
- [ ] Wired into the `projections` slice in `cmd/api/main.go`; the old inline shim deleted
- [ ] No `nathejk.dk/...` import in the package
- [ ] `go build ./... && go vet ./...` and `gofmt -l` clean
- [ ] Verified against the real stream: row count and per-team totals match
      `patrulje.memberCount`

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006. Schema and consumer kept in one task deliberately:
  PRD 006 §10 listed them separately, but a consumer that writes columns another task has
  not added yet cannot be verified in isolation, so splitting would only produce a
  knowingly-broken intermediate state.
