# 056 — Nødtelefon detail: card for the case, PrimeVue Timeline for events

**Status:** done
**Priority:** medium
**Created:** 2026-08-11
**Picked up by:** agent session
**Started:** 2026-08-11
**Completed:** 2026-08-11

## Description

UX refinement to the case detail view built in task 050, requested by the product owner
after using it. Refines **PRD 001** §7 rather than changing its scope, so it is a task
rather than a PRD amendment.

Four changes:

1. The case's **headline and description are the prominent thing** on the screen, in a
   PrimeVue `Card` — they are what an operator reads when picking up the phone or taking
   over a shift.
2. The activity log uses the PrimeVue **`Timeline`** component, comments included, so
   everything that happened to the case is one chronological rail rather than a list of
   rows in the middle of the page.
3. **Priority, assignee and patrol association are dimmed and compact.** They are set
   once or twice per case; the case text and the timeline are read constantly.
4. **Creation and headline/description edits produce no timeline entry.** The card states
   the current title and description, so an entry saying they were set is noise on the one
   surface a handover depends on.

## Notes

Point 4 is a **display filter, not a change to what is recorded.** `created` and
`headline.updated` remain events (they are what create the case and update the row — the
projection cannot work without them) and they remain rows in `sos_activity`, so "who
renamed this case, and when" is still answerable. Only the rendered timeline is curated,
which is also the reversible direction: hiding a row can be undone, never writing it
cannot.

`description.updated` is filtered too, on the same reasoning as the headline, though the
request only named the title.

## Acceptance Criteria

- [x] Headline and description in a `Card`, headline at `text-3xl`, description in body text
- [x] Activity rendered with PrimeVue `Timeline` — marker icon, time in the opposite slot,
      content (including the inline comment editor) in the content slot
- [x] Comments appear on the same timeline as state changes
- [x] Priority / assignee / patrol column dimmed (full opacity on hover or focus-within)
      and narrowed from 1/3 to 1/4 of the grid
- [x] No timeline entry for `created`, `headline.updated`, `description.updated`
- [x] Events and `sos_activity` rows unchanged — audit trail intact
- [x] Frontend tests pass; eslint clean; no new type errors

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-11 — Created and completed in one sitting, from the product owner's feedback on
  the shipped detail view.
- 2026-08-11 — `SosActivityLine` became **content-only**: the icon moved to the Timeline's
  `#marker` and the timestamp to `#opposite`, so the rail stays aligned across entries of
  different heights. It kept the inline comment editor and the tolerance for unknown
  activity types, which PRD 006 depends on.
- 2026-08-11 — Narrowed the Timeline's opposite column to `3.5rem` in CSS: it holds `HH:MM`
  and the component's default split gives it half the width.
- 2026-08-11 — The aside is dimmed to 0.75 opacity, not hidden or collapsed. An operator
  taking over a shift still has to see who a case is assigned to without interacting with
  anything.
- 2026-08-11 — Verified against real data rather than by eye: a case with `created`,
  `team.associated`, `headline.updated`, `commented` and `severity.specified` rows renders
  three entries — the association, the comment and the priority — and its `sos_activity`
  table still holds all five.
- 2026-08-11 — `Card` and `Timeline` needed no imports: the PrimeVue resolver picked both
  up and `components.d.ts` regenerated accordingly.
- 2026-08-11 — ✅ All criteria met: 60 frontend tests pass, eslint clean on the five SOS
  files, type-error count unchanged at 106, Vite compiled without warnings.
