# 181 — GET /api/search/person endpoint

**Status:** done
**Priority:** high
**Created:** 2026-09-14
**Picked up by:** agent session 2026-09-14
**Started:** 2026-09-14
**Completed:** 2026-09-14

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

- [x] `GET /api/search/person` registered and serving — verified over HTTP against the running API
- [x] OpenAPI annotations present, covering cap, year scope, classification, departed people
- [x] `includeOtherYears` defaults to false; a malformed value does not widen the scope
      (renamed from `includePreviousYears`; see task 180's log)
- [x] Response includes matched-phone role, status, deleted flag, year, and truncation flag
- [x] Response carries no address, birthday or notes
- [ ] ~~Endpoint requires authentication (asserted, not assumed)~~ — **cannot be met inside hq;
      raised as task 188.** `app.authenticate` attaches an anonymous user and does not
      authenticate. The route is on the same middleware as every other `/api` route, which is
      all that is achievable here; what that is worth is task 188's question.
- [x] Handler tests for a phone hit, a name hit, an empty result and a too-short query
- [x] `go build ./...` and `go test ./...` pass

## Progress Log

- 2026-09-14 22:08 — Task created from PRD 014 §10.
- 2026-09-15 03:05 — Picked up. `cmd/api/search.go` plus a route, and `SearchPerson` added to
  `data.Models` — assigned after `NewModels` the way `PhotoCover` is, rather than threaded
  through a parameter list that is already 24 arguments long.
- 2026-09-15 03:08 — Decision: the year flag is read as `== "true"` and nothing else. `"1"`,
  `"yes"`, `"TRUE"` and a trailing space all leave the scope at the active year. Parsing
  leniently would mean a typo could silently reach across every edition of the event, and this
  endpoint returns minors' contact details. Pinned by a table of eight inputs — including
  `includePreviousYears=true`, the name the parameter nearly had, which must do nothing.
- 2026-09-15 03:10 — Decision: a nil projection is a **500**, not an empty result. An empty
  result reads to an operator as "this person does not exist", which is the one answer this
  feature must never give wrongly.
- 2026-09-15 03:14 — An empty `q` is answered 200 with `tooShort`, not rejected with 400: the SPA
  holds the box open with nothing in it, and a 400 per keystroke would be noise.
- 2026-09-15 03:22 — **Two gotchas hit while testing, both worth recording.**
  (1) `&application{}` with a nil Logger does not panic cleanly on the error path — it hangs, and
  cost this task a 600-second test timeout before I noticed. Tests that exercise
  `ServerErrorResponse` need a real `jsonlog.Logger` writing to `io.Discard`.
  (2) **`app.routes()` can only be called once per test binary.** It calls `app.Metrics`, which
  calls `expvar.NewInt`, and `expvar.Publish` panics on a duplicate name. My route-registration
  test made the pre-existing `TestStreamRouteBypassesMetricsAndStreams` fail while passing in
  isolation — a nasty shape of failure. Dropped that test with a comment explaining why, and
  verified the route against the running API instead, which is better evidence anyway since it
  also proves the wiring in `main.go`.
- 2026-09-15 03:30 — **Verified over HTTP against the running dev API** (hot-reloaded, so this is
  the real binary): `q=20309696` returns Sophie Wedenborg with `phoneRole: own` and her three
  duplicate contact rows; `q=Koch` returns the nine Kochs; `q=ab` returns
  `tooShort: true`; `q=notfoundxyz` returns `tooShort: false, matchedName: true, results: []` —
  the "we looked and found nobody" case, which is exactly the distinction the UI needs.
  `/api/search/nonsense` is a 404, so nothing wider than the intended path is routed.
- 2026-09-15 03:36 — **Could not meet the authentication criterion, and did not pretend to.**
  `app.authenticate` does not authenticate: it attaches an anonymous user and calls through,
  because authentication lives in an external service. Every `/api` route is in the same
  position, so this endpoint is no worse than its neighbours — but "the same as everything else"
  is a weaker statement than the PRD's non-functional requirement assumed. Pre-existing and not
  caused by this feature; what this feature changes is the stake, since one request now returns
  fifty named minors with phone numbers where before it took knowing which list to open. Raised
  as **task 188**, and PRD 014 §6 corrected to say what is actually true. Also noted there that
  §11's auditing question is currently unanswerable for the same reason: there is no identity to
  log.
- 2026-09-15 03:40 — ✅ Completed, with the authentication criterion explicitly unmet and
  delegated rather than ticked. `go build ./...`, `gofmt`, and the full `go test ./...` clean.
