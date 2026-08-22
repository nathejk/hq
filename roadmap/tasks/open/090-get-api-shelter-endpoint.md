# 090 — GET /api/shelter

**Status:** open
**Priority:** high
**Created:** 2026-08-23
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] `GET /api/shelter` registered in `routes.go` and handled in `shelter.go`
- [ ] Response grouped as above, with counts and the in-our-care total
- [ ] `memberStatuses` and `placements` in the envelope
- [ ] Timestamps sent raw; no server-side duration formatting
- [ ] Members moved between teams resolve to a named patrol
- [ ] No per-member queries in a loop
- [ ] OpenAPI annotations present and complete
- [ ] Router boots (no wildcard/static conflict) — covered by an existing-style routes test

## Progress Log

- 2026-08-23 09:30 — Task created from PRD 007 §10.
