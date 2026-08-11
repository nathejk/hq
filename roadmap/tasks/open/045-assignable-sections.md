# 045 — Assignable sections for SOS cases

**Status:** open
**Priority:** medium
**Created:** 2026-08-11
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] Table created on startup; flag defaults to off
- [ ] Toggle endpoint sets and unsets, idempotently
- [ ] `GET /api/organisation` reports the flag per section
- [ ] Assignee list in the API/UI offers only flagged sections
- [ ] `go build ./... && go vet ./...` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-11 — Created from PRD 001 on approval.
