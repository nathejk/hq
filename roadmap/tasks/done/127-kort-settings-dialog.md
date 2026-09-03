# 127 — Settings button and `KortSettingsDialog` on `/kort`

**Status:** done
**Priority:** high
**Created:** 2026-09-03
**Picked up by:** agent session (Zed)
**Started:** 2026-09-03
**Completed:** 2026-09-03

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

- [x] Gear button in the existing toolbar, opening the dialog
- [x] Maps listed grouped by set, reorderable by dragging, persisted via `/api/kortsaet/:id/kort`
- [x] Create / rename / edit / delete a map
- [x] Selecting a map highlights its checkpoints on the map behind the dialog
- [x] Edit mode and settings cannot be active at once
- [x] Danish UI text
- [x] `npm run build` clean

## Progress Log

- 2026-09-03 — Task created from PRD 010 §10.
- 2026-09-03 — Picked up. Plan: dialog in the shape of `DispatchTaskDialog` (PrimeVue `Dialog`,
  `emit('update:visible')`, writes then tells the owner to refresh). Scope held to the shell plus
  sheet CRUD, selection highlighting and reorder — checkpoints are 128, sets 129, extents 130.
- 2026-09-03 — The dialog is **not modal** (`:modal="false"`, `position="right"`). That is the whole
  design: selecting a sheet fades every marker that is not on it, which is how a mis-assigned
  checkpoint is spotted *before* the sheets are printed. A modal mask over the map would hide the
  only feedback surface this screen has.
- 2026-09-03 — Highlighting **fades** non-member markers to 0.35 rather than hiding them. "This
  checkpoint is not on this sheet" is precisely the mistake being hunted, and a hidden marker
  cannot be noticed. It is applied by walking the existing markers, not by rebuilding them — a
  rebuild would drop the popups and, in edit mode, the drag handlers.
- 2026-09-03 — Called `applyHighlight()` at the end of `applyPayload` too. Without it, a live payload
  arriving while the dialog is open silently un-fades the map, because the markers it faded no
  longer exist. Easy to miss, and it would have looked like the highlight "randomly stops working".
- 2026-09-03 — The form edits a **copy**, not the cached row. The cached value is the same object the
  markers are drawn from, so binding an input straight to it would rename a marker letter by letter
  as the operator types — and a live payload mid-edit would rewrite the field under the cursor.
- 2026-09-03 — Switching sheets, and closing the dialog, are **refused while dirty** rather than
  silently discarding. In a list of fifteen one-line rows a mis-click is easy, and losing a typed
  name to one would be the kind of small betrayal that stops an operator trusting the screen.
- 2026-09-03 — Mutual exclusion with position editing is enforced on **both** entry points, not just
  the button's `:disabled`: `enterEditMode` returns early while the dialog is open and
  `enterSettings` returns early while editing. A disabled attribute is a hint, and these two both
  own marker interaction — more so once corner-picking lands in task 130.
- 2026-09-03 — An empty format is **omitted** from the PUT rather than sent as `""`: the API's four
  values do not include the empty string, and a sheet whose format is not yet decided is the normal
  state of a sheet just created. Sending it would turn "not decided" into a 422.
- 2026-09-03 — The reorder refresh runs in a `finally`, so a failed save also refreshes: otherwise
  the list would keep showing an order that was never persisted, which is worse than snapping back.
- 2026-09-03 — Verified through the running stack: created two sets and two sheets, assigned
  checkpoints, and sent the reorder exactly as the dialog does — `Kort 2` came back first with
  `sortOrder` 0. `vite build` succeeds and `vue-tsc` reports no errors in the new files (the repo
  has 107 pre-existing ones elsewhere).
- 2026-09-03 — **Not verified in a browser**: no way to drive one from here, so the dialog's layout
  and drag handle are unconfirmed visually. The Vite HMR log applied the view with no error, which
  is as far as I can take it.
- 2026-09-03 — Left two sets ("Patruljer" marked `patrulje`, and "Crew" unmarked) and two sheets in
  the dev data on purpose: set management is task 129, so without a set the dialog has nothing to
  add a sheet to and the feature cannot be tried at all yet.
- 2026-09-03 — Completed. `emit('update:dirty')` is already in place for task 131 to hang the
  deferral on.
