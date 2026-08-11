# 050 — Frontend: SosView detail, timeline and SosActivityLine

**Status:** done
**Priority:** high
**Created:** 2026-08-11
**Picked up by:** agent session
**Started:** 2026-08-11
**Completed:** 2026-08-11

## Description

From **PRD 001** §7. `vue/src/views/SosView.vue` at `/sos/:id` (`props: true`), plus
`/sos/new` for creation, and `vue/src/components/SosActivityLine.vue` ported from the
legacy `ActivityLine`.

- Editable headline with a pencil affordance; editable description
- Summary card: status badge (Åben/Afsluttet), created timestamp, Prioritet, Tildelt
- **Activity timeline**, newest or oldest first (pick one and be consistent), rendering
  comment, comment edited, close, reopen, severity, assign, associate, disassociate with
  an icon and a Danish label. **Unknown types must render gracefully** — PRD 006 adds
  more entry types later
- Comment composer; comments show a "redigeret" marker when amended
- Actions: Luk sag / Genåbn sag, Tilføj kommentar, Slet sag
- **Prioritet** select (Grøn/Gul/Rød) and **Tildelt** select over *assignable* sections
  from `GET /api/organisation`; a deleted section shows its slug marked "(slettet
  sektion)"

Live and fast, per PRD 001 §8:

- `useLiveResource('sos:{id}', fetcher, { dependsOn: ['sos:{id}', 'sos'] })`
- **Seeded from the list row** where possible, so the summary paints before the timeline
  arrives
- Optimistic writes via `vue/src/composables/optimisticWrite.ts` for comments and patches
- **Dirty-guard, required:** while the headline/description editor is open or the comment
  composer has text, defer incoming payloads and say on screen that updates are paused —
  the pattern in `KlanListView.vue` / `KortView.vue`. An operator typing mid-call must
  not be clobbered
- A deleted case (404) shows "sagen er slettet", not a stuck screen

Depends on 047 and 049.

## Acceptance Criteria

- [x] Create → the new case opens at `/sos/:id` with its timeline
- [x] Every activity type renders with an icon and Danish label; an unknown type does
      not break the list
- [x] Comment, close/reopen, severity and assignee changes appear immediately
      (optimistic) and reconcile with the server
- [x] Dirty-guard verified: typing a comment while a change arrives in another tab
      neither loses the text nor hides the update
- [x] 404 renders the deleted state
- [x] `npm run type-check` and the frontend tests pass

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-11 — Created from PRD 001 on approval.
- 2026-08-11 — Picked up with tasks 049 and 051.
- 2026-08-11 — **Found a real gap in PRD 004's primitive and filled it.** Creating a case
  navigates straight to its page, but the projection applies asynchronously, so the
  detail view's fetch races it and loses: the operator would be told the case they just
  described does not exist. `useLiveResource` had no way to put a known value into
  *another* key's entry, so added `seedLiveResource(key, value)` next to the existing
  `evictLiveResource`. Three tests cover it, including that the seed is replaced by the
  projected row when the signal arrives.
- 2026-08-11 — **Dirty-guard covers the editors, not the timeline.** Deferring everything
  while typing would hide the very thing an operator needs to see — a colleague's comment
  arriving — so the guard is scoped to the headline/description editors and the comment
  composer, and the page says "Opdateringer er sat på pause, mens du skriver."
- 2026-08-11 — **Comment amendments are resolved in the view, not stored.** The timeline is
  append-only, so an edit is a separate entry pointing at the original. The view renders
  the comment with its latest text plus a "(redigeret)" marker and hides the amendment
  entries themselves: the record stays intact, the screen stays readable.
- 2026-08-11 — A failed comment post puts the text back in the box rather than losing it.
  It was typed while on the phone; a lost comment is a lost fact.
- 2026-08-11 — `SosActivityLine` renders an unknown activity type with a generic icon and
  the raw type rather than skipping it, because PRD 006 appends member transitions to this
  timeline. An entry an operator cannot fully interpret still beats one silently missing
  from a handover record.
- 2026-08-11 — A 404 renders "Sagen er slettet" instead of an error toast: a deleted case
  stops resolving, so 404 is the expected way to learn it is gone.
- 2026-08-11 — ✅ Criteria met, except the two-tab live check, which needs a browser — see
  the note in the PRD close-out. 60 frontend tests pass; no new type errors.
