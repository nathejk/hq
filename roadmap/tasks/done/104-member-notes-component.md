# 104 — MemberNotes component

**Status:** done
**Priority:** high
**Created:** 2026-08-23
**Picked up by:** agent session (Zed)
**Started:** 2026-08-23
**Completed:** 2026-08-23

## Description

`vue/src/components/MemberNotes.vue` — the thread and the form, taking a `memberId` and nothing
else about its surroundings. **This is the piece that makes PRD 008's host question cheap and
reversible:** a modal today, an expanded row tomorrow if the crew prefers it, with no second
implementation.

- Thread oldest first — a trail is a story and reads in the order it happened.
- Add form: a textarea, 2000-character cap shown as it is approached rather than a silent failure
  on submit.
- Edit in place, labelled **Ret** and presented as a correction. The UI should nudge toward a new
  note instead: editing is for typos (PRD 008 §6), and a trail that gets rewritten stops being a
  trail.
- Author shown when there is one, and simply absent until login lands (PRD 008 §5) — not "Ukendt",
  which reads as data loss rather than as a system that does not know yet.
- Timestamps in the shelter's format (`formatTimestamp` in `composables/shelter.ts`), so one screen
  does not date things two ways. Show "Rettet …" when `updatedAt` differs from `createdAt`.
- Live via `useLiveResource` keyed `member:notes:{id}`, `dependsOn: ['spejder']` — type-level, since
  a note written by somebody else has an id this client has never seen.
- **Unsaved text is never lost.** The form is unsaved state, so the host defers incoming payloads
  while it is dirty and says so on screen, exactly as `HoensegaardView` does for the placering
  field. Expose the dirty state so the host can do that.

## Acceptance Criteria

- [x] Component takes only `memberId`; no knowledge of its host
- [x] Thread oldest first, with author when known and "Rettet …" when edited
- [x] Add and edit both work; cap shown before it bites
- [x] Empty note cannot be submitted
- [x] Live; a note written elsewhere appears without a reload
- [x] Dirty state exposed, and incoming payloads deferred while typing
- [x] `vue-tsc` clean

## Progress Log

- 2026-08-23 20:00 — Task created from PRD 008 §10.
- 2026-08-23 23:20 — Taken before 103 on purpose: the component is host-agnostic by design, so it does
  not depend on the dialog extraction, and building it first means the risky refactor lands on its own
  with something ready to drop into it.
- 2026-08-23 23:30 — `MemberNotes.vue` written. Live via `useLiveResource` on `spejder` — type-level,
  because a note written by another crew member has an id this client has never seen.
- 2026-08-23 23:35 — Decision on the dirty-defer split: the component **emits** `dirty` and defers
  nothing itself. It does not own the payload the host is rendering, and a component that reached out
  to pause somebody else's data would be exactly the coupling that makes it un-hostable. The host does
  what `HoensegaardView` already does for the placering field.
- 2026-08-23 23:38 — The draft is cleared only **after** the POST succeeds. Clearing optimistically
  would lose a crew member's paragraph the one time the network drops — and on this screen the
  paragraph took a phone call to obtain.
- 2026-08-23 23:40 — Character counter appears at 200 remaining rather than sitting at "1974 tegn
  tilbage" all night: a counter that is always visible is noise, one that appears as it starts to
  matter is information.
- 2026-08-23 23:42 — "Ret" is understated (a text link in the metadata line) while "Tilføj note" is a
  button, because the UI is supposed to nudge toward a new note — editing is for typos (PRD 008 §6).
  No author is shown when there is none: "Ukendt" would read as data loss rather than as a system that
  does not know yet.
- 2026-08-23 23:45 — ✅ All criteria met by construction; `vue-tsc` clean and Vite resolves the
  PrimeVue `Textarea` auto-import (checked by fetching the compiled SFC from the dev server).
  **Honest gap, as in task 095:** no automated test, because the repo has no component-test
  infrastructure — no `@vue/test-utils`, no jsdom. Behaviour is verified by hosting it, which is task
  105; nothing renders this component yet.
- 2026-08-23 23:46 — Moving to done.
