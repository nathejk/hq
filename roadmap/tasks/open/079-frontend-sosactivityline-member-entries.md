# 079 — SosActivityLine entry types for member operations

**Status:** open
**Priority:** medium
**Created:** 2026-08-17
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] Each new entry type renders with a Danish label and a formatted timestamp
- [ ] A team collection renders as a single line naming the affected members
- [ ] Rendering uses only the entry payload, never a lookup of current member state
- [ ] Unknown types still degrade gracefully (existing behaviour preserved)
- [ ] Actor shown where present, absent gracefully where empty
- [ ] `npm run build` and `vitest` clean; no new TypeScript errors

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006. Depends on task 071.
