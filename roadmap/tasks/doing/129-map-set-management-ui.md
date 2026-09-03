# 129 — Map set management UI

**Status:** doing
**Priority:** medium
**Created:** 2026-09-03
**Picked up by:** agent session (Zed)
**Started:** 2026-09-03
**Completed:**

## Description

PRD 010 §6, §7 (set editor). Depends on 122, 127.

Manage sets from the settings dialog: create, rename, reorder, delete, and set or clear the
optional team type.

The team type is a `Select` **with an empty option**, and the empty option is the normal
case — it is the crew set, which klaner also draw from. Label it so it is clear the field is
what marks a set as the spejder set, not decoration.

Sets are created by the operator, never chosen from a fixed enum: most years there are two,
but a year may have three, and that must need no code change.

Deleting a set that still holds maps is refused by the API (task 122) — surface that as a
readable message rather than a generic error.

## Acceptance Criteria

- [ ] Create / rename / reorder / delete sets
- [ ] Team type settable and clearable, empty presented as an ordinary choice
- [ ] Refused deletion of a non-empty set shows a clear Danish message
- [ ] Maps list regroups when a map moves between sets
- [ ] `npm run build` clean

## Progress Log

- 2026-09-03 — Task created from PRD 010 §10.
- 2026-09-03 — Picked up. Plan: inline set editing in the dialog's existing list rather than a second
  nested dialog — a set has two fields, and a modal over a modal over the map would bury the map
  the whole screen is about. This is also what makes the feature usable without curl.
