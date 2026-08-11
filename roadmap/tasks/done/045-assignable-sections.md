# 045 — Assignable sections for SOS cases

**Status:** done
**Priority:** medium
**Created:** 2026-08-11
**Picked up by:** agent session
**Started:** 2026-08-11
**Completed:** 2026-08-11

## Description

From **PRD 001** §6 and §8 (amended 2026-08-11). A case is assigned to an organisation
**section**, but not every section should be offered — the operator picks from sections
explicitly flagged as able to take nødråb.

The flag is **owned by this feature**, in hq, not added as a column to shared-go's
`section` table: "can be assigned nødråb" is a fact about the nødtelefon, not a general
property of a section, and shared-go cannot be released mid-implementation.

- Projection `sos_assignable_section` keyed by `(year, sectionSlug)`
- Events `NATHEJK.{year}.sos.section.{slug}.assignable.set` / `.unset`
- Toggle command, off by default, so the assignee list starts empty and is opted into
- Exposed per section in the existing `GET /api/organisation` payload, so the SPA needs
  no new endpoint
- `PUT /api/section/:slug/sos-assignable` for the toggle

A case keeps the slug it was assigned even if the section is later renamed (new label
shows) or deleted (raw slug shown, marked "(slettet sektion)").

## Acceptance Criteria

- [x] Table created on startup; flag defaults to off
- [x] Toggle endpoint sets and unsets, idempotently
- [x] `GET /api/organisation` reports the flag per section
- [x] Assignee list in the API/UI offers only flagged sections
- [x] `go build ./... && go vet ./...` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-11 — Created from PRD 001 on approval.
- 2026-08-11 — Picked up after task 046, which had to land first for `commands.Sos`.
- 2026-08-11 — One subject with a boolean body
  (`NATHEJK.{year}.sos.section.{slug}.assignable`) rather than the `.set` / `.unset`
  pair the PRD sketched. One fact with two values, and a consumer matching two subjects
  can handle one and silently miss the other — which here would mean a section that can
  never be un-flagged.
- 2026-08-11 — The section case is matched **first** in `HandleMessage`: its subject also
  has `sos` in third position, so `NATHEJK.*.sos.*.assigned` would otherwise be a
  candidate match for it.
- 2026-08-11 — Schema in its own `assignable.sql` rather than appended to `table.sql`:
  the three tables there are a case and its history, this one is configuration of which
  sections the nødtelefon may route to.
- 2026-08-11 — The handler checks the section exists for the year before publishing.
  Without it a typo creates an assignable entry for a section nobody can see — invisible
  in the UI and impossible to turn off from it.
- 2026-08-11 — Exposed as `sosAssignableSections` (a list of slugs) beside `sections`
  rather than merged into each section object. The section entity belongs to shared-go
  and knows nothing about the nødtelefon; keeping them apart in the payload is what keeps
  that true.
- 2026-08-11 — ✅ Verified end to end against the dev stack: `PUT .../guides/sos-assignable`
  → event → projection row → `sosAssignableSections: ["guides"]` in `GET /api/organisation`;
  a repeat set is a no-op; unset removes the row; an unknown section is a 404.
- 2026-08-11 — The Organisation-page toggle itself is task 053. Moving to done.
