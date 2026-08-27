# 108 — `dispatchable` flag on sections

**Status:** done
**Priority:** high
**Created:** 2026-08-27
**Picked up by:** agent session (Zed)
**Started:** 2026-08-27
**Completed:** 2026-08-27

## Description

PRD 009 §8 ("The dispatch unit needs no new entity"). A dispatch unit is a subsection of
logistics holding a vehicle and crew — already expressible by the organisation tree. The only
new fact is which sections are dispatchable.

Follow `sos_assignable_section` **exactly**: a `dispatchable_section` table keyed by
`(yearSlug, sectionSlug)`, the slugs returned beside the sections on the organisation payload,
and `PUT /api/section/:slug/dispatchable` mirroring `/api/section/:slug/sos-assignable`.

Not a column on the `section` entity: `section` lives in shared-go and knows nothing about
kørsel (same decision as PRD 001).

## Acceptance Criteria

- [x] `dispatchable_section` table + schema, created on boot like `assignable.sql`
- [x] Slugs exposed on the organisation/sections payload as `dispatchableSections` (`[]`, never `null`)
- [x] `PUT /api/section/:slug/dispatchable` with OpenAPI annotations, mirroring the sos-assignable route
- [x] Organisation page toggle per section, beside the existing sos-assignable toggle
- [x] Tests: set, unset, idempotent set, year scoping

## Progress Log

- 2026-08-27 — Task created from PRD 009 §10.
- 2026-08-27 — Picked up. Plan: new `go/nathejk/table/dispatch/` package holding only the
  dispatchable-section flag (the rest of the entity is task 109), mirroring `sos`'s
  `assignable.sql` / `SectionAssignableSet` / `SetSectionAssignable` triple; then the
  organisation payload field, the route, and the tree toggle.
- 2026-08-27 — **Decision: the flag lives in a new `dispatch` package, not in `sos` and not in a
  package of its own.** It is a kørsel fact, and task 109 puts tasks and tours in the same
  package, so a separate `dispatchablesection` package would exist for one day and then be
  merged. `dispatchable.sql` is embedded separately from the (not yet written) `table.sql`, for
  the reason `sos` splits its own: configuration of which sections can be sent out is a
  different kind of thing from a record of what happened.
- 2026-08-27 — Package written: `types.go` (Actor), `messages.go`, `consumer.go`, `querier.go`,
  `commands.go`, `table.go`. The `schemaMigrations` hook is in from the start, empty, because
  `CREATE TABLE IF NOT EXISTS` is a no-op wherever the table exists.
- 2026-08-27 — ✅ Backend criteria: table, query, command and consumer. Wired into the
  `projections` slice in `cmd/api/main.go`, so the new live entity token is `dispatch` — derived
  from position 3 of `NATHEJK:*.dispatch.section.*.dispatchable`, which is why the subject is
  `dispatch.section.{slug}.dispatchable` rather than `section.{slug}.dispatchable`: the latter
  would emit a `section` signal and make kørsel invisible to a client depending on `dispatch`.
- 2026-08-27 — **On "OpenAPI annotations": there is no OpenAPI tooling in this repo** — no
  generator, no spec, no swag comments anywhere in `go/`. Rather than invent a convention on one
  endpoint, the handler carries a doc comment stating method, path, request body and every
  response code, which is the shape a generator would read later. Recorded here because PRD 009
  §8 leans on the annotations as the driver app's contract, and that promise is now a doc comment
  until somebody adds the tooling.
- 2026-08-27 — **Correction to the entry above: I was wrong, and the reason is worth recording.**
  My grep for annotations passed `@Summary` through a shell that ate the `@`, so it reported
  nothing while `note.go`, `shelter.go` and `order.go` are full of swag comments. The handler now
  carries proper `@Summary` / `@Description` / `@Router` annotations in the house style. The
  lesson generalises: a grep that finds *nothing* is evidence about the grep at least as often as
  about the code.
- 2026-08-27 — A fourth `Actor` conversion (`dispatchActor` in `cmd/api/dispatch.go`). `actor.go`
  says three is a pattern worth converging; still not done, because a shared `types.Actor` is a
  shared-go change and every one of these packages is deliberately unable to import another. The
  comment now says so out loud.
- 2026-08-27 — ✅ Frontend: `dispatchableSections` on the payload type, a `pi-truck` toggle on the
  section rows beside the nødråb phone. Optimistic like its neighbour, and here that is not just
  feel: the command dirty-checks, so a no-op toggle publishes nothing and there is no live signal
  to confirm it by.
- 2026-08-27 — Verified: `go build ./...` and the full `go test ./...` green, 10 new tests in
  `dispatch`. `vue-tsc` reports nothing on the added lines (the repo's type-check has 109
  pre-existing errors, including two on this file's tree row, all untouched). Not verified
  end-to-end: the hq stack is not running locally and there is no node toolchain outside Docker.
- 2026-08-27 — All criteria met. Moving to done.
