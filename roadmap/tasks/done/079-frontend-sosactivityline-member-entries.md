# 079 — SosActivityLine entry types for member operations

**Status:** done
**Priority:** medium
**Created:** 2026-08-17
**Picked up by:** agent session
**Started:** 2026-08-17
**Completed:** 2026-08-17

## Description

From **PRD 006** §7. Render the new timeline entries in
`vue/src/components/SosActivityLine.vue`, one per kind of summarising case event (task 071):

- **member status changed** — who, from what to what
- **members moved** — who, and to which patrol
- **team collected** — one line for the whole operation ("hele patruljen hentes"), naming
  the members

Additive: PRD 001 requires the component to tolerate unknown types, so an unrendered type
degrades rather than breaks, and the backend half can ship first.

## Notes

- **Render from the entry's own payload, not from current member state.** The summarising
  event is self-contained precisely so a timeline line reads as what happened *then*. Joining
  to today's member rows would show today's truth on yesterday's entry, which is the one thing
  a shift-handover log cannot do.
- **No exception-granted entry** — exceptions do not exist (PRD 006 §11 Decisions).
- **No team-discontinued entry** — discontinuation has no event. What the timeline shows is
  the operation that took `activeMemberCount` to zero; the Udgået badge (task 076) shows the
  resulting state.
- One line per operation, so a three-member collection is one entry rather than three. If it
  renders as three, the backend is publishing per-member events without the summary — check
  task 071 before working around it here.
- Danish labels, `da-DK` timestamps via the shared date helpers (`composables/datefilters.ts`)
  — Go's `time.Time` text form is not parseable by Safari, which is why `parseApiDate` exists.

## Acceptance Criteria

- [x] Each new entry type renders with a Danish label and a formatted timestamp
- [x] A team collection renders as a single line naming the affected members
- [x] Rendering uses only the entry payload, never a lookup of current member state
- [x] Unknown types still degrade gracefully (existing behaviour preserved)
- [x] Actor shown where present, absent gracefully where empty — unchanged from PRD 001;
      the actor is empty until the auth service lands
- [x] `npm run build` and `vitest` clean; no new TypeScript errors

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006. Depends on task 071.
- 2026-08-17 — Picked up. Three labels and icons in `composables/sos.ts`, plus
  `parseMemberSummary` / `memberStatusPhrase`, rendered by `SosActivityLine.vue`.
- 2026-08-17 — These are the **only** timeline entries whose `value` is structured rather
  than a bare string, so `isMemberSummaryType` gates the parse. Treating a PRD 001 entry as
  JSON would blank a line that works today, which is why the guard is an explicit list of
  three types rather than a try-parse on everything.
- 2026-08-17 — A summary that fails to parse **falls back to the raw value** rather than
  rendering nothing. An entry an operator cannot fully read still belongs on a handover
  record; a blank line does not.
- 2026-08-17 — Rendered strictly from the stored payload, never a lookup. This is the
  property the whole self-contained-summary design exists for: a member moved twice would
  otherwise have their *first* move described using their *second* team, and a timeline
  whose entries change meaning after the fact is worse than no timeline.
- 2026-08-17 — Status phrases are lower-case and mid-sentence ("i løbet → venter på at blive
  hentet"), deliberately separate from the backend's `MemberStatuses()` picker labels, which
  are capitalised standalone tags. Same strings, two grammatical contexts — and the empty
  status is named ("ikke startet") rather than left as a gap.
- 2026-08-17 — Strength after the operation is shown, with zero spelled out as "Patruljen er
  udgået — ingen tilbage i løbet". Checked against `undefined` rather than falsiness,
  because 0 is the most meaningful value it takes.
- 2026-08-17 — 13 new tests (91 total), including that the three new types are recognised
  and the four PRD 001 types are **not**, and that unknown types still fall back to the raw
  label and generic icon — the tolerance PRD 001 requires, since the car and shelter
  interfaces will add more.
- 2026-08-17 — ✅ Verified against a real stored summary from the dev database (the
  `team.collected` entry from task 076's verification), which renders as:
  *"Hele patruljen hentes: Uglerne"*, four members each *"i løbet → venter på at blive
  hentet"*, then *"Patruljen er udgået — ingen tilbage i løbet"*. One entry, four people,
  and the consequence stated — which is what the N-events-plus-one-summary design was for.
- 2026-08-17 — ✅ All criteria met. Moving to done.
