# 129 — Map set management UI

**Status:** done
**Priority:** medium
**Created:** 2026-09-03
**Picked up by:** agent session (Zed)
**Started:** 2026-09-03
**Completed:** 2026-09-03

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

- [x] Create / rename / reorder / delete sets
- [x] Team type settable and clearable, empty presented as an ordinary choice
- [x] Refused deletion of a non-empty set shows a clear Danish message
- [x] Maps list regroups when a map moves between sets
- [x] `npm run build` clean

## Progress Log

- 2026-09-03 — Task created from PRD 010 §10.
- 2026-09-03 — Picked up. Plan: inline set editing in the dialog's existing list rather than a second
  nested dialog — a set has two fields, and a modal over a modal over the map would bury the map
  the whole screen is about. This is also what makes the feature usable without curl.
- 2026-09-03 — The team type on a new set deliberately **starts empty**. Defaulting it to `patrulje`
  would be the convenient choice and a bad one: an operator creating the crew set and not touching
  the field would silently mark it as the spejder set, and the hej-app would then serve crew sheets
  to patrols. Empty is also the commonest correct answer.
- 2026-09-03 — The empty option is labelled „Ingen bestemt holdtype” and sits first, so it reads as a
  deliberate choice rather than an unfilled field — which is what a bare blank line in a `Select`
  looks like. Added a hint under it saying what the field decides, since "holdtype" alone does not
  convey that it is the thing the hej-app matches on.
- 2026-09-03 — The set PUT sends the **whole record** (name *and* team type), matching the API's
  event: a partial body could not express clearing the team type, because an absent field and "no
  team type" would be indistinguishable. Verified against the real API — `teamType: null` clears it
  and sending it back re-marks the set.
- 2026-09-03 — A refused delete is surfaced as a **toast carrying the API's own Danish message**
  („sættet indeholder stadig kort — flyt eller slet dem først”), not a field error: the problem is
  the set's contents, not anything the operator typed, so there is no field to put it beside.
  Checked the actual 422 body shape rather than assuming it — `{"error":{"kortsaetId":"…"}}` — which
  is what the extraction relies on.
- 2026-09-03 — Set reorder posts to `PUT /api/kortsaet`, the collection. Worth restating in the code
  comment because it looks like an oversight: `/api/kortsaet/sorted` cannot exist (httprouter panics
  on a static segment beside `/:id`), and Danish gave no plural to move the route to.
- 2026-09-03 — Set edits join the same `anyDirty` guard as sheet edits, so a half-typed set name also
  pauses live updates and blocks closing the dialog. Three sources of unsaved state with one guard,
  rather than three guards.
- 2026-09-03 — Moving a sheet between sets needs no new code: the sheet editor already has a
  Kortsæt `Select`, and the list regroups on the next payload because the API nests by set.
- 2026-09-03 — Verified against the real API: clear and restore a team type, reorder the two sets
  (Crew came back first), and the refused delete. `vite build` clean, no type errors in the feature.
- 2026-09-03 — Completed. The feature is now self-sufficient in the UI — an operator can create a set
  and its sheets without a single API call by hand, which was not true after task 128.
