# 125 — Kort CRUD and read handlers with OpenAPI annotations

**Status:** open
**Priority:** high
**Created:** 2026-09-03
**Picked up by:**
**Started:**
**Completed:**

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
| PUT | `/api/kort/sorted` |

Deliberately **no `GET /api/kort/:id`** and no `GET /api/kortsaet`: the whole year is a
handful of records, `GET /api/kort` returns all of it, and both the modal and the hej-app
work from that one cached response. A single-record read would be a second code path with
no caller.

`GET /api/kort` nests maps under sets so a consumer gets the `teamType` marking and the
maps in one round trip. Year-scoped via the existing `X-YearSlug` header.

Arrays are `[]`, never `null` — the hej-app parses this.

Every endpoint needs **OpenAPI annotations** (repo convention, PRD §8).

## Acceptance Criteria

- [ ] All six endpoints registered in `cmd/api/routes.go`
- [ ] OpenAPI annotations on every one
- [ ] `GET /api/kort` returns sets with maps nested, `checkpointIds` and `extents` as `[]`
      when empty
- [ ] Year-scoped by `X-YearSlug`
- [ ] Manually exercised end to end (create a set, a map, assign checkpoints, read back)

## Progress Log

- 2026-09-03 — Task created from PRD 010 §10.
