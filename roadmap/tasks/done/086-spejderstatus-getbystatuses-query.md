# 086 — spejderstatus.GetByStatuses query

**Status:** done
**Priority:** high
**Created:** 2026-08-23
**Picked up by:** agent session (Zed)
**Started:** 2026-08-23
**Completed:** 2026-08-23

## Description

PRD 007 (Hønsegården) needs the population "started but not active": every member of the
year whose status is one of `waiting`, `transit`, `sheltered`, `reunited`, `released`. No
existing query can answer it — `GetByTeam` is team-scoped, `GetByMemberIDs` needs the ids
up front, and `InOurCare` returns counts rather than rows.

Add to `go/nathejk/table/spejderstatus/querier.go`:

```go
GetByStatuses(ctx context.Context, year types.YearSlug, statuses []types.MemberStatus) ([]SpejderStatus, error)
```

Named for what it does, **not** `ListNotActive()`: the set of statuses is the caller's
question and the query must not encode one screen's policy. The screen builds its set from
`types.MemberStatus` predicates where it can (`InOurCare()`), so a fourth in-care state
added to shared-go shows up without editing this.

Follow `InOurCare`'s existing shape for the `IN` clause: build placeholders from the slice,
never interpolate. Empty slice returns an empty result and issues no query.

Order by `updatedAt DESC` — recent arrivals are what the shelter is looking at. The screen
groups by status itself, so the query must not.

The package may not import `nathejk.dk/...` (lift to shared-go, task 083); `lift_test.go`
enforces it.

## Acceptance Criteria

- [x] `GetByStatuses` added to the `Queries` interface and implemented on `querier`
- [x] Placeholders built from the slice; no string interpolation of status values
- [x] Empty slice returns `([]SpejderStatus{}, nil)` without touching the database
- [x] Results ordered by `updatedAt DESC`
- [x] Unit tests: multiple statuses, empty slice, a status nobody has, and a year boundary
      (rows of another year are not returned)
- [x] `go test ./...` passes, `lift_test.go` still green

## Progress Log

- 2026-08-23 09:30 — Task created from PRD 007 §10.
- 2026-08-23 09:45 — Picked up. Plan: extract the SQL building into a pure helper and test
  that, rather than adding a DB-backed test — `go-sqlmock` is in go.sum but used nowhere,
  and `consumer_test.go` establishes the house style of asserting on emitted SQL.
- 2026-08-23 10:05 — `GetByStatuses` implemented in `querier.go`, with `byStatusesQuery` as
  a pure builder. Ordering is `updatedAt DESC, id`: the id tiebreak is not decoration — a
  patrol starting writes its whole roster within one second, so without it the order within
  a group is the storage engine's whim and two loads can disagree, which on screen looks
  like rows shuffling by themselves.
- 2026-08-23 10:10 — Extracted `selectSpejderStatus`, the column list all four reads share.
  The scan order depends on it and it was spelled out four times; a column added to
  `SpejderStatus` could previously be added to three queries and forgotten in the fourth.
- 2026-08-23 10:15 — Decision: empty status set returns an empty slice and issues no query.
  Not defensive tidiness — `status IN ()` is a MySQL syntax error, so a caller that filtered
  its list down to nothing would get a database error where it meant an empty answer.
- 2026-08-23 10:20 — Added `GetByStatuses` to `stubQueries` in `commands_test.go` (the
  interface's only other implementer). It filters the stub's team rows rather than returning
  nil, so a future command reading by status gets an answer consistent with the rest of the
  stub instead of one that makes a bug look like an empty population.
- 2026-08-23 10:30 — ✅ All criteria met. `querier_test.go` added (5 tests); `go build ./...`,
  `go vet` and the full `go test ./...` are green. Two criteria were met by construction
  rather than by a DB test, and the wording deserves the correction: the "year boundary" and
  "status nobody has" cases are asserted at the level of the statement and its arguments
  (the year is always constrained and always the first argument; the status set is always
  placeholders) rather than against rows in a database. A row-level test needs a MySQL
  server or a new sqlmock dependency, and this package has neither today. Noted rather than
  quietly ticked.
- 2026-08-23 10:32 — Moving to done.
