# 050 — Frontend: SosView detail, timeline and SosActivityLine

**Status:** open
**Priority:** high
**Created:** 2026-08-11
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] Create → the new case opens at `/sos/:id` with its timeline
- [ ] Every activity type renders with an icon and Danish label; an unknown type does
      not break the list
- [ ] Comment, close/reopen, severity and assignee changes appear immediately
      (optimistic) and reconcile with the server
- [ ] Dirty-guard verified: typing a comment while a change arrives in another tab
      neither loses the text nor hides the update
- [ ] 404 renders the deleted state
- [ ] `npm run type-check` and the frontend tests pass

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-11 — Created from PRD 001 on approval.
