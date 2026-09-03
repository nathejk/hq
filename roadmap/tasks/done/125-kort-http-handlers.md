# 125 — Kort CRUD and read handlers with OpenAPI annotations

**Status:** done
**Priority:** high
**Created:** 2026-09-03
**Picked up by:** agent session (Zed)
**Started:** 2026-09-03
**Completed:** 2026-09-03

## Description

PRD 010 §8 (API endpoints). Depends on 121, 122, 124. **This is what unblocks the hej-app**,
so it lands before any frontend work.

| Method | Path |
|---|---|
| GET | `/api/kort` — the year's sets with their maps nested |
| POST | `/api/kort` |
| PUT | `/api/kort/:id` |
| DELETE | `/api/kort/:id` |
| PUT | `/api/kort/:id/checkpoints` |
| PUT | `/api/kortsaet/:id/kort` — reorder the set's maps |

Note the reorder route: **not** `PUT /api/kort/sorted`, which task 122 established cannot exist —
httprouter panics at startup on a static segment beside `/:id` at the same level, taking the whole
API down. Sheet order lives under the set, which is where handout order is meaningful anyway.

Deliberately **no `GET /api/kort/:id`** and no `GET /api/kortsaet`: the whole year is a
handful of records, `GET /api/kort` returns all of it, and both the modal and the hej-app
work from that one cached response. A single-record read would be a second code path with
no caller.

`GET /api/kort` nests maps under sets so a consumer gets the `teamType` marking and the
maps in one round trip. Year-scoped via the existing `X-YearSlug` header.

Arrays are `[]`, never `null` — the hej-app parses this.

Every endpoint needs **OpenAPI annotations** (repo convention, PRD §8).

## Acceptance Criteria

- [x] All six endpoints registered in `cmd/api/routes.go`
- [x] OpenAPI annotations on every one
- [x] `GET /api/kort` returns sets with maps nested, `checkpointIds` and `extents` as `[]`
      when empty
- [x] Year-scoped by `X-YearSlug`
- [x] Manually exercised end to end (create a set, a map, assign checkpoints, read back)

## Progress Log

- 2026-09-03 — Task created from PRD 010 §10.
- 2026-09-03 — Picked up. Plan: handlers alongside the set handlers in `cmd/api/kort.go`, reusing
  `kortCommandError`. `GET /api/kort` nests maps under sets from one query of each table, and every
  array is `[]` rather than null because the hej-app parses this.
- 2026-09-03 — **End-to-end testing against the running dev stack found a bug the unit tests could
  not see.** `ErrDegenerateExtent` and `ErrTooManyExtents` were missing from `kortCommandError`, so
  they fell through to `ServerErrorResponse`: an operator drawing a rectangle with two clicks on the
  same spot got a **500 with a stack trace** instead of "vælg to forskellige hjørner". The command
  tests passed — they assert the right error comes *back* — and the handler then mistranslated it.
  Fixed, and verified 422 through the real stack.
- 2026-09-03 — Added `cmd/api/kort_errors_test.go` as an **exhaustive table over the package's
  exported errors**, rather than a case for the two that were missing. Adding an error to the kort
  package and forgetting to map it now fails here. Also pinned the *wrapped* `ErrSetNotEmpty` (it
  carries the sheet count), because a handler switching on `==` would 500 on the only form of that
  error an operator ever hits.
- 2026-09-03 — Dropped a clever negative case from that test that fed a string-wrapped error to prove
  `errors.Is` was in use: it takes the 500 path, and `application{}` has no logger, so it panicked
  and hung the package's tests for ten minutes. The wrapped-error case proves the same thing without
  touching the 500 path.
- 2026-09-03 — `Kortsaet.Maps` lost its `omitempty`, caught by a test asserting the JSON shape: with
  it, a set with no sheets sent **no `kort` key at all**, forcing every client to handle absence as
  well as emptiness. A newly created set is exactly that case.
- 2026-09-03 — `Nest` returns orphans (`orphanKort`) rather than dropping sheets whose set is
  unknown. That state is reachable during replay — events arrive in stream order, so a sheet may
  precede its set — and after a bad edit, and silently omitting them would make a map invisible in
  the one screen that exists to find such mistakes. Normally empty.
- 2026-09-03 — Verified through the real stack (NATS + MariaDB, hot-reloaded container): the live
  entity advertisement now includes `kort,kortsaet`; corners sent bottom-right-first came back
  normalised to a true north-west/south-east pair; a bogus checkpoint id was filtered out of the
  read; the crew set serialised `teamType: null` and `kort: []`; deleting a non-empty set answered
  422. Test data cleaned up afterwards, leaving `{"kortsaet":[],"orphanKort":[]}`.
- 2026-09-03 — **Known consequence of CQRS, observed while cleaning up:** deleting a sheet and then
  immediately deleting its set is refused, because the set's emptiness is checked against the read
  model, which lags the write by one projection round trip. The second attempt succeeds. This is
  correct rather than a bug — the check must read committed state — and the UI is well placed to
  absorb it, since the live signal empties the set's list before an operator can click again. Worth
  knowing for task 129, which surfaces that refusal.
- 2026-09-03 — Completed. `go build ./...`, `go vet`, `gofmt` clean; 49 tests pass across
  `nathejk/table/kort` and `cmd/api`. `GET /api/kort` is live and ready for the hej-app (task 134
  documents it).
