# 108 — `dispatchable` flag on sections

**Status:** open
**Priority:** high
**Created:** 2026-08-27
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] `dispatchable_section` table + schema, created on boot like `assignable.sql`
- [ ] Slugs exposed on the organisation/sections payload as `dispatchableSections` (`[]`, never `null`)
- [ ] `PUT /api/section/:slug/dispatchable` with OpenAPI annotations, mirroring the sos-assignable route
- [ ] Organisation page toggle per section, beside the existing sos-assignable toggle
- [ ] Tests: set, unset, idempotent set, year scoping

## Progress Log

- 2026-08-27 — Task created from PRD 009 §10.
