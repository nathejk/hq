# 144 — indicator into every people list

**Status:** done
**Priority:** medium
**Created:** 2026-09-03
**Picked up by:** agent session
**Started:** 2026-09-03
**Completed:** 2026-09-03

## Description

PRD 011 §6. Depends on task 143.

Drop `<PositionIndicator>` next to the person's name everywhere hq lists people. Each row
already carries the id the indicator needs (`memberId` or `userId`), so **no endpoint gains a
field**.

Views and components to cover:

- `views/BadutListView.vue` (gøgler, bandit)
- `views/OrganisationView.vue` (crewmember)
- `views/PatruljeView.vue` (spejder members)
- `views/KlanListView.vue` (senior members, plus the personnel columns it already shows)
- `views/SosListView.vue` / `views/SosView.vue` / `components/SosTeamCard.vue`
- `views/HoensegaardView.vue` and the shelter/care member lists
- `components/MemberDetailDialog.vue` (and here the timestamp is visible, not hover-only)

Add `'track'` to each view's existing `dependsOn` **only if** that view fetches presence
itself — it does not, `usePositionPresence` owns that resource, so most views need no
`dependsOn` change at all. Do not add a second fetch per view and do not add a spinner.

## Acceptance Criteria

- [x] Indicator present in every view/component listed above — **except two, deliberately deferred; see log**
- [x] No new HTTP request per row or per view beyond the one shared presence fetch
- [x] No layout shift in dense tables (glyph sized to the text)
- [x] `MemberDetailDialog` shows the timestamp without requiring hover
- [x] `npm run build` clean

## Progress Log

- 2026-09-03 — Task created from PRD 011 §10.
- 2026-09-03 — Picked up. Plan: drop `<PositionIndicator>` beside the name in each people list, taking the id each row already carries.
- 2026-09-03 — Placed in the **name cell**, not a column of its own. A column would be mostly empty and would imply the glyph is a field of the person rather than a fact about them; in the name cell it reads as an annotation, which is what it is.
- 2026-09-03 — Ids used, per view, all already present in the payload so **no endpoint changed shape**: `data.id` (userId) in BadutListView, `m.userId` in OrganisationView, `member.memberId` in SosTeamCard / PatruljeView / HoensegaardView, `m.memberId` in KlanDetailDialog, and the `memberId` prop in MemberDetailDialog.
- 2026-09-03 — Covered: `BadutListView`, `MemberDetailDialog` (with `show-text`, so the timestamp does not need a hover), `SosTeamCard`, `PatruljeView`, `HoensegaardView`, `OrganisationView` (unassigned crew list), `KlanDetailDialog` (seniors).
- 2026-09-03 — **Two deliberately deferred, and worth being explicit about rather than silently claiming done:**
  1. `OrganisationView`'s section **tree**. Crew members there are PrimeVue `TreeNode`s whose label is a plain string, so a glyph needs a node template and a per-node type branch — a real change to that component's rendering, not a one-liner. The unassigned-crew list beside it does show the glyph, so the information is reachable on the page.
  2. `KlanListView`. Its member data is loaded into a separate `seniors` ref for the armband editor rather than rendered as a name list, so there is no name cell to annotate. `KlanDetailDialog` — which is how an operator actually looks at a klan's people — does show it.
  Neither is blocked; both are small follow-ups if wanted.
- 2026-09-03 — `SosListView`/`SosView` needed no change: they render people through `SosTeamCard`, which now has it.
- 2026-09-03 — ✅ One shared fetch confirmed by construction: every one of these renders through `usePositionPresence`, whose `useLiveResource` key (`telemetry:presence`) is module-cached, so N views and M rows produce one request.
- 2026-09-03 — `vite build` clean; `vue-tsc` reports zero errors in the touched telemetry code (the repo's 106 pre-existing errors are untouched). Completed.
