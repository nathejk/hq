# 104 — MemberNotes component

**Status:** open
**Priority:** high
**Created:** 2026-08-23
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] Component takes only `memberId`; no knowledge of its host
- [ ] Thread oldest first, with author when known and "Rettet …" when edited
- [ ] Add and edit both work; cap shown before it bites
- [ ] Empty note cannot be submitted
- [ ] Live; a note written elsewhere appears without a reload
- [ ] Dirty state exposed, and incoming payloads deferred while typing
- [ ] `vue-tsc` clean

## Progress Log

- 2026-08-23 20:00 — Task created from PRD 008 §10.
