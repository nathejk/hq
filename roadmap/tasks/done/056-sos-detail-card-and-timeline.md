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
- [x] Comments rendered in a `Card` on the rail, state changes as plain one-liners
- [x] Comment card carries a slight tint so it does not dissolve into the white panel
- [x] State-change **text** dimmed; marker and timestamp left as they are
- [x] Vertical spacing between entries halved via the `timeline.event.minHeight` token
- [x] Case summary is a note card in the *same* outer panel as the timeline, full width
- [x] Case card and comment cards share one style definition, so they cannot drift
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
- 2026-08-11 — Follow-up from the product owner after seeing it: **comments now sit in a
  `Card` on the rail**, state changes stay plain one-liners. A card each would put the wall
  of boxes back; a card only for comments is what makes somebody's words look like content
  rather than another status line. The card is flat and tightly padded — a raised panel per
  comment reads as a stack of floating boxes on a vertical rail. The "Kommentar" label is
  gone with it: the marker icon and the card already say what it is, so the label was
  spending a line to repeat them.
- 2026-08-11 — **Found while styling that card:** this project's Tailwind config maps
  `surface-*` to `rgb(var(--surface-N))` (`tailwind.config.js:22-33`), but PrimeVue 4 emits
  `--p-surface-N`, and nothing defines the unprefixed ones. So every `text-surface-500` in
  the SPA compiles to an invalid colour and is dropped — the dimming those classes intend
  has never rendered. My four SOS files now use the stock palette (`text-gray-500` and so
  on) so the hierarchy the owner asked for actually appears. **Not** fixed repo-wide: that
  is a deliberate, separate decision (define the variables, or sweep the classes) and it
  would change the look of pages nobody asked me to touch.
- 2026-08-11 — Second round of owner feedback, and the two halves of it work together: the
  comment card is **tinted** (`#f4f6f8`) because the page's own cards are white and a white
  card inside a white panel dissolves; and the state changes are **dimmed across all three
  of their parts** — text `gray-400`, detail `gray-500`, marker `gray-300` with no fill,
  timestamp `gray-400`. Dimming only the text would have left the markers as a column of
  dark dots pulling the eye down the rail, which is the opposite of the intent.
- 2026-08-11 — The tint is set in scoped CSS rather than as a `bg-*` class: PrimeVue's own
  `.p-card` background rule would otherwise win on equal specificity, and which one applied
  would depend on stylesheet order. A scoped rule carries the extra attribute selector and
  lands on top deterministically.
- 2026-08-11 — Third round: **only the text of a state change is dimmed.** Reverted the
  faded markers and timestamps from the previous round — the owner was right, those are the
  rail itself, and fading them makes the timeline harder to follow rather than quieter.
- 2026-08-11 — Halved the vertical spacing, and the fix was not where the previous round
  assumed. Aura sets `timeline.event.minHeight: 5rem`
  (`@primeuix/themes/aura/timeline`), which is what spaces the entries out; the content
  padding I had been adjusting is a rounding error beside it. Set
  `--p-timeline-event-min-height: 2.5rem` on the timeline instead, so the connector, marker
  and content shorten together and stay aligned — padding-only tweaks would have moved the
  content off its own marker.
- 2026-08-11 — Fourth round: **one outer panel**. The case summary, the timeline and the
  comment box now sit on a single `.card`, with the case rendered as the same tinted note
  card as a comment — full width, like the composer beneath it. Two stacked white panels
  made the case and its history read as separate screens; they are one story.
- 2026-08-11 — The note surface (`.sos-note`: tint, border, flatness, padding) is defined
  **once**, in this view's unscoped style block, and used by both the case card and
  `SosActivityLine`'s comment card. The requirement is that the two look the same, and two
  copies of those declarations would have drifted the first time either was nudged. The
  component keeps only its own hover-pencil rule and a comment pointing at where the
  surface lives.
- 2026-08-11 — Raised the shared padding from `0.6rem/0.75rem` to `0.75rem/1rem`: the
  original was tuned for one-line comments and read as cramped under the case's 3xl
  headline. One value that suits both, rather than a special case that would defeat the
  point of sharing.
- 2026-08-11 — Fifth round: the case's own buttons are `size="small"` to match "Tilføj
  kommentar", and "Genåbn sag" is `secondary` while "Luk sag" stays primary. Closing is the
  ordinary end of a case; reopening is a correction and should not look like the thing to
  do next.
- 2026-08-11 — Moved the live-update indication onto the navigation bar's moon (task
  answered a question rather than just a request: the badge was **already global**, in
  `App.vue`, not SOS-specific). The moon is yellow when healthy — which is how it has
  always looked — red while reconnecting, dark grey when offline, with the state and its
  explanation in the title/aria label. The labelled badge now appears **only** when nothing
  is arriving: a colour cannot tell an operator that what they are looking at may be
  missing changes, and that is the one case worth words.
- 2026-08-11 — Broke `Navigation.vue` doing it and caught it by asking Vite rather than
  trusting the linter: my edit left a premature `</script>`, so the file had two, and the
  SFC compiler returned 500 on every request for it. eslint reported it as an "invalid end
  tag" 150 lines away, which is why the transform check (`wget` the module from the dev
  server) is the check worth running — the page would simply have been blank.
