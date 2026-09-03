# 127 — Settings button and `KortSettingsDialog` on `/kort`

**Status:** doing
**Priority:** high
**Created:** 2026-09-03
**Picked up by:** agent session (Zed)
**Started:** 2026-09-03
**Completed:**

## Description

PRD 010 §7. Depends on 126.

A gear button in the existing `.edit-toolbar` in `vue/src/views/KortView.vue`, beside the
edit-mode button, opening `vue/src/components/kort/KortSettingsDialog.vue`.

The map stays visible and is the primary feedback surface — a side `Dialog`/`Drawer` at
~420px, **not** a full-screen dialog that hides the thing being described.

Contents: the year's maps grouped by set, draggable to reorder (`vuedraggable`, already a
dependency), and an editor for the selected map — name, set, format (`SelectButton`), note.
Extents are task 130; the checkpoint picker is task 128.

**Marker dragging and map settings are mutually exclusive**: opening one disables the
other, since both own marker interaction (PRD §8, dependencies & risks).

## Acceptance Criteria

- [ ] Gear button in the existing toolbar, opening the dialog
- [ ] Maps listed grouped by set, reorderable by dragging, persisted via `/api/kort/sorted`
- [ ] Create / rename / edit / delete a map
- [ ] Selecting a map highlights its checkpoints on the map behind the dialog
- [ ] Edit mode and settings cannot be active at once
- [ ] Danish UI text
- [ ] `npm run build` clean

## Progress Log

- 2026-09-03 — Task created from PRD 010 §10.
- 2026-09-03 — Picked up. Plan: dialog in the shape of `DispatchTaskDialog` (PrimeVue `Dialog`,
  `emit('update:visible')`, writes then tells the owner to refresh). Scope held to the shell plus
  sheet CRUD, selection highlighting and reorder — checkpoints are 128, sets 129, extents 130.
