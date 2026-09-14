# 181 — GET /api/search/person endpoint

**Status:** doing
**Priority:** high
**Created:** 2026-09-14
**Picked up by:** agent session 2026-09-14
**Started:** 2026-09-14
**Completed:**

## Description

Depends on task 180. The HTTP surface: `go/cmd/api/search.go`, registered in
`cmd/api/routes.go`.

```
GET /api/search/person?q=&includePreviousYears=
```

`/api/search/person` rather than `/api/search`, so a later search over poster or kort is a
sibling and not a breaking change to this one.

`includePreviousYears` (default false) rather than `year=` or `allYears=`: the parameter names
the operator's *decision*, and its default is the primary behaviour, so the year scope cannot
widen through an omitted or malformed parameter. A caller that forgets it gets this year.
Implementation is task 184; this task wires the parameter and the default.

Response per result: `kind`, `id`, `name`, matched phone and its role (own / parent /
contact), `teamId`, team or section name, status, `deleted`, and `year`. Plus a top-level flag
for whether the cap was hit.

**Privacy.** This is the first endpoint that returns contact details for minors aggregated
across the whole event. It sits behind the same authentication as every other `/api` route —
confirm that, do not assume it. Deliberately **no** address, birthday or notes: those stay on
the detail pages the results link to, so search is a way to find a person, not a way to bulk
export the population.

**OpenAPI annotations are required.** Follow the style in `cmd/api/reassign.go`. The
description must state the cap, the year scoping, the phone-vs-name classification, and that
results include departed and removed people — a caller that assumed otherwise would present
them as current.

## Acceptance Criteria

- [ ] `GET /api/search/person` registered and serving
- [ ] OpenAPI annotations present, covering cap, year scope, classification, departed people
- [ ] `includePreviousYears` defaults to false; a malformed value does not widen the scope
- [ ] Response includes matched-phone role, status, deleted flag, year, and truncation flag
- [ ] Response carries no address, birthday or notes
- [ ] Endpoint requires authentication (asserted, not assumed)
- [ ] Handler tests for a phone hit, a name hit, an empty result and a too-short query
- [ ] `go build ./...` and `go test ./...` pass

## Progress Log

- 2026-09-14 22:08 — Task created from PRD 014 §10.
