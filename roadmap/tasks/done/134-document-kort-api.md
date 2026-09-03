# 134 — Document `GET /api/kort` for the hej-app team

**Status:** done
**Priority:** medium
**Created:** 2026-09-03
**Picked up by:** agent session (Zed)
**Started:** 2026-09-03
**Completed:** 2026-09-03

## Description

PRD 010 §8, §9. Depends on 125.

The hej-app lives in another repo and is the reason this feature exists. Hand its team a
short written contract for `GET /api/kort`, covering the things they cannot infer from the
JSON and would otherwise get wrong:

- **Two reveal units, not one.** A map's checkpoints are revealed when its QR is linked or
  scanned; a checkgroup is revealed as a whole when any of its checkpoints is scanned. A
  skitse has no QR — its checkpoints are revealed via the previous checkpoint.
- **Find patrol maps via the set's `teamType`, never by name.** Set names are Danish free
  text an organizer may rename mid-season.
- **`teamType` is a filter, not a key.** Several sets may share one. It is nullable, and an
  unmarked set is the general/crew set — which klaner draw from. So filtering by `klan`
  usually returns nothing, and that is **not** an error: fall back to the unmarked set.
- Year-scoped by `X-YearSlug`; arrays are `[]`, never `null`.
- Maps are per year and never carried forward — the event is in a different area each year.

## Acceptance Criteria

- [x] Written contract committed in the repo (`roadmap/api/kort.md`) — **still needs sending to the
      hej-app team**, which I cannot do from here
- [x] Covers both reveal units, the `teamType` semantics and the klan fallback
- [x] Sample response payload included — captured from the running API, not written by hand
- [x] OpenAPI annotations verified to match the documented shape

## Progress Log

- 2026-09-03 — Task created from PRD 010 §10.
- 2026-09-03 — Picked up. Plan: a document in the repo, with a real captured payload rather than a
  hand-written one — an invented example is where a contract first drifts from the code.
- 2026-09-03 — Landed as `roadmap/api/kort.md`, in a new `roadmap/api/` folder. It documents *meaning*
  and defers shape to the annotations, so the two cannot contradict each other as the payload evolves.
- 2026-09-03 — The sample payload is **copied from the running dev API**, including its slightly odd
  reality (a set with no sheets, `format: ""`, `sortOrder` not starting at 0). A tidied-up example
  would have quietly promised a shape the API does not produce.
- 2026-09-03 — Led with the two reveal units, because that is the thing most likely to be implemented
  as one rule. Spelled out that a skitse has no QR and is revealed off the *previous* checkpoint's
  scan — which is also the reason a sheet with no extent and no QR still matters.
- 2026-09-03 — Gave the `teamType` semantics three numbered consequences rather than a sentence,
  because each is a bug waiting to happen: `null` is ordinary and not missing data; the value is not
  unique so *all* matching sets must be collected; and filtering by `klan` usually returns nothing,
  which is **not** an error — fall back to the unmarked set.
- 2026-09-03 — Also documented what is deliberately **absent**: which QR sits on which sheet, and where
  a sheet is handed out. The first is the immediate follow-up, and until it exists this endpoint gives
  *candidate* sheets rather than the one a team holds. Leaving that unsaid is how another team builds
  on an assumption we never made.
- 2026-09-03 — Verified the annotations mechanically: extracted every `@Router` line from
  `cmd/api/kort.go` and every registered `/api/kort*` route from `routes.go` and compared them — ten
  and ten, matching method for method. Worth doing because the repo has **no OpenAPI tooling at all**
  (found and flagged back in task 003), so nothing else would ever catch a drifted annotation.
- 2026-09-03 — Completed. The document needs a human to actually send it to the hej-app team.
- 2026-09-03 — **Superseded, same day.** knj: the hej-app must not fetch from HQ over REST — all
  cross-service communication goes over the stream. So this document was wrong in premise, not in
  detail: `GET /api/kort` serves HQ's own SPA and is not an integration point. Replaced with
  `roadmap/api/kort-events.md`, which documents the **event** contract for a consumer building its
  own projection. The semantics carried over unchanged (two reveal units, the `teamType` filter and
  its klan fallback, extents, per-year); what changed is the transport, plus two things that only
  matter over the stream: `kort.updated` is a **patch** while `kortsaet.updated` is a **whole
  record**, and a consumer must resolve `checkpointIds` against its own checkpoint projection
  because HQ's read-time filtering does not travel. `roadmap/api/kort.md` is deleted; PRD 010
  corrected in place; the shared-go lift is now blocking work as task 138.
