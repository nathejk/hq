# 114 — task dialog with the place picker

**Status:** done
**Priority:** high
**Created:** 2026-08-27
**Picked up by:** agent session (Zed)
**Started:** 2026-08-27
**Completed:** 2026-08-27

## Description

PRD 009 §7. `components/DispatchTaskDialog.vue` — create and edit a task.

Fields: kind (`pickup` / `transport` / `collection` / `delivery`, defaulting the places
differently), pick-up and drop-off place, description, priority (grøn/gul/rød via task 112),
space needs in words, `tidligst` and `skal leveres`.

**Place picker**: one control offering checkpoints, loks and HQ as groups and accepting free
text — free text is the normal case for "på Slangerupvej ved skovbrynet", not a fallback.
`composables/personnelTree.ts`'s grouped picker is the model.

Almost everything is optional: the mitigation for the desk-discipline risk (§8) is that the
written path is the fastest path.

## Acceptance Criteria

- [x] Dialog creates and edits tasks against `POST`/`PATCH /api/dispatch/task`
- [x] Kind selection defaults the places sensibly (delivery from HQ, collection to HQ, …)
- [x] Place picker offers grouped checkpoints/loks/HQ and accepts arbitrary text
- [x] Priority uses the shared severity vocabulary
- [x] Only kind and a description are required
- [x] Danish labels; day-and-time entry for `tidligst` / `skal leveres`

## Progress Log

- 2026-08-27 — Task created from PRD 009 §10.
- 2026-08-27 — **The picker needed a vocabulary, so the board payload now carries one.** Added
  `places` to `GET /api/dispatch` — HQ, then the year's checkpoints, then its loks, each as
  kind + refId + label. Sent with the board rather than fetched by the dialog: it must open
  instantly at 3am, and a picker that loads its own options is a picker somebody types around.
  HQ is present even on a fresh year with nothing configured, which is what keeps the control
  usable from the first night.
- 2026-08-27 — The picker is an **AutoComplete, not a Select**. Free text is not a fallback here:
  "på Slangerupvej ved skovbrynet" is the normal way to say where a scout is standing, and a
  control that only offered known locations would be worked around by typing the road name into
  the description — where nothing can read it as a place. Typing emits a `text` place on every
  change rather than on blur, so a dialog saved with Enter cannot lose the last thing typed.
- 2026-08-27 — Kind defaults: delivery leaves HQ, collection and pickup return to it, transport
  has neither end fixed. Applied only when the operator *changes* the kind, and never while
  editing an existing task — rewriting the places of something somebody already filled in is how
  a form loses work.
- 2026-08-27 — The draft is reset on `visible` rather than on mount: PrimeVue keeps the component
  alive between openings, so a stale half-typed task would be waiting the next time the dialog
  appeared.
- 2026-08-27 — 422 responses are rendered **beside the field they are about**, because the API
  already answers with a field → Danish message map, and a toast makes the operator hunt for which
  box is wrong. Anything not field-shaped still falls back to a toast.
- 2026-08-27 — Opening the dialog also **pauses live updates**, along with the two confirmation
  dialogs added in task 113. That required moving the `cancelling`/`boarding` refs above `paused`:
  `useDeferredApply` evaluates the condition immediately, so a later `const` is a
  temporal-dead-zone error on mount — which shows up as a blank screen, not a warning.
- 2026-08-27 — ✅ Verified: `vue-tsc` clean for the three files after fixing one real find of its
  own (`forceSelection="false"` passed the *string* "false", which is truthy — the picker would
  have refused free text, defeating the whole point of it); repo baseline unchanged at 109. The
  dev server compiles view, dialog and picker. Against the running stack, `GET /api/dispatch` now
  returns 14 places — HQ plus the year's real checkpoints and loks.
- 2026-08-27 — All criteria met. Moving to done.
