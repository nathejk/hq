# 090 — GET /api/shelter

**Status:** done
**Priority:** high
**Created:** 2026-08-23
**Picked up by:** agent session (Zed)
**Started:** 2026-08-23
**Completed:** 2026-08-23

## Description

The read endpoint behind the Hønsegården screen (PRD 007 §8). New file
`go/cmd/api/shelter.go`, year-scoped from `X-YearSlug` like everything else.

Returns the started-but-not-active population, grouped as the screen renders it:

- `transit` — the arrivals queue
- `sheltered` — who is here, with placering
- `waiting` — still out on the trail
- `reunited` + `released` — closed, for the shift handover

Per member: name, patrol ref (id, number, name), status, `updatedAt` (the screen derives
"siden 21:40 (2t 14m)" from it — send the timestamp, not a formatted duration), placering
where sheltered, own phone, guardian phone, and the open sos case id if the patrol has one.

Also in the envelope: the group counts, the in-our-care total, `memberStatuses`
(`MemberStatuses()`, already served on the patrol and case payloads — the screen must not
carry a second label map, PRD 006 §6), and `placements` from `DistinctPlacements` (task 087)
for the combobox. All in one response: the screen wants them at the same moment, and they
are small.

Composes existing models — `SpejderStatus.GetByStatuses` (086), `Members.GetSpejdere` for
names and phones, `Teams.GetPatrulje` for team refs, the `sos` querier for the case link.
Watch the roster asymmetry the member modal already handles: a member moved between teams is
not on their current team's roster, so fall back to `initialTeamId`.

Path is `/api/shelter`, singular — a screen's view of a place. **Not** `/api/member/shelter`:
httprouter builds one tree per method and cannot hold a static segment where a sibling has a
wildcard, which is exactly what produced the plural `/api/members/care`. It would panic the
router at boot.

**OpenAPI annotations are required** — `@Summary`, `@Description`, `@Tags`, `@Produce`,
`@Success`, `@Failure`, `@Router` — following `cmd/api/order.go`.

Budget: p95 under 50ms. The population is tens of rows; avoid a query per member.

## Acceptance Criteria

- [x] `GET /api/shelter` registered in `routes.go` and handled in `shelter.go`
- [x] Response grouped as above, with counts and the in-our-care total
- [x] `memberStatuses` and `placements` in the envelope
- [x] Timestamps sent raw; no server-side duration formatting
- [x] Members moved between teams resolve to a named patrol
- [x] No per-member queries in a loop
- [x] OpenAPI annotations present and complete
- [x] Router boots (no wildcard/static conflict) — covered by an existing-style routes test

## Progress Log

- 2026-08-23 09:30 — Task created from PRD 007 §10.
- 2026-08-23 12:20 — Picked up. Added `shelter.Queries` to `data.Models` and threaded it
  through `NewModels` (positional, 19 params now — ugly but consistent; refactoring that
  signature is not this task's business).
- 2026-08-23 12:40 — Handler written. Sections are served as an **ordered array with Danish
  labels** rather than a map of statuses, so the crew's working order and the section copy are
  the server's to decide. Same argument PRD 006 §6 settled for status labels: two places
  holding the same Danish copy is how one of them ends up saying something else at 3am.
- 2026-08-23 12:45 — Everything bulk-fetched before the loop: the year's roster (one query,
  indexed by member — cheaper than one per patrol, since this screen's scouts are spread
  thinly across many patrols), the year's patrols, and the placeringer. The SOS case is the
  one exception: no bulk "cases for these teams" query exists, so it is looked up per
  *distinct patrol* and memoised. Bound is affected patrols, not scouts, and
  `TestOpenCaseIsLinkedOncePerPatrol` pins it.
- 2026-08-23 12:50 — Decision: a member missing from the roster falls back to their id for a
  name rather than being skipped or rendered blank. Happens when a signup row is removed after
  the scout started; a child the crew cannot act on is worse than an ugly row.
- 2026-08-23 12:55 — Decision: `startTeam` is sent only when it differs from the current team,
  so the ordinary row does not carry the same patrol twice and the SPA can treat its presence
  as meaning "this one did not start where you think".
- 2026-08-23 13:00 — Decision: the in-our-care total comes from `InOurCare()`, not from
  counting the rows just assembled. Two independent counts of the children we are responsible
  for is one more than the night can afford.
- 2026-08-23 13:05 — Removed a package-level `var _ = func(){...panic...}()` self-check I had
  written to assert the status set. Fail-fast at init is the wrong trade here: it would crash
  the whole API at boot over one screen's invariant. It is a test instead
  (`TestShelterStatusesAreDerivedAndExcludeTheRoute`).
- 2026-08-23 13:15 — **Blocker, and a finding worth recording.** The first version of the test
  helper built a server from `app.routes()` per test, and the second call panicked:
  `routes()` installs `app.Metrics`, which calls `expvar.NewInt`, and expvar panics on a
  duplicate name. **`app.routes()` can therefore be called at most once per process**, and
  `stream_test.go` already calls it. Resolved by invoking the handler directly with
  `httptest.NewRecorder`. Note this is not a loss of coverage for route registration: that one
  existing call is what would panic on the httprouter static-vs-wildcard conflict this task
  was warned about (`/api/shelter` beside `/api/member/:memberId`), so the constraint is
  verified — just not by a test of mine. Anyone adding routes should know about the expvar
  limit; it is a trap with a confusing error message.
- 2026-08-23 13:25 — ✅ All criteria met. 11 handler tests, all green; `cmd/api` at 16 passing
  tests; `go build ./...` and full `go test ./...` green.
- 2026-08-23 13:26 — Moving to done.
