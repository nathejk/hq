# 144 — indicator into every people list

**Status:** open
**Priority:** medium
**Created:** 2026-09-03
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] Indicator present in every view/component listed above
- [ ] No new HTTP request per row or per view beyond the one shared presence fetch
- [ ] No layout shift in dense tables (glyph sized to the text)
- [ ] `MemberDetailDialog` shows the timestamp without requiring hover
- [ ] `npm run build` clean

## Progress Log

- 2026-09-03 — Task created from PRD 011 §10.
