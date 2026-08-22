# 086 — spejderstatus.GetByStatuses query

**Status:** open
**Priority:** high
**Created:** 2026-08-23
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] `GetByStatuses` added to the `Queries` interface and implemented on `querier`
- [ ] Placeholders built from the slice; no string interpolation of status values
- [ ] Empty slice returns `([]SpejderStatus{}, nil)` without touching the database
- [ ] Results ordered by `updatedAt DESC`
- [ ] Unit tests: multiple statuses, empty slice, a status nobody has, and a year boundary
      (rows of another year are not returned)
- [ ] `go test ./...` passes, `lift_test.go` still green

## Progress Log

- 2026-08-23 09:30 — Task created from PRD 007 §10.
