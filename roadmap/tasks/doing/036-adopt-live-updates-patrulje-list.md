# 036 — Adopt live updates on the patrulje list

**Status:** doing
**Priority:** high
**Created:** 2026-08-09
**Picked up by:** agent session (zed)
**Started:** 2026-08-09
**Completed:**

## Description

First adopter of the PRD 004 capability (Phase 3). Until a page composes
`useLiveResource`, the transport, hub and notifier are a pipe nobody drinks from:
signals reach the browser and are published onto the bus, and no resource is
registered to react to them.

That is exactly what a two-tab test showed — an edit in one tab appears in the other
only after navigating away and back, because the *old* `onMounted(load)` refetch is
still what fetches, and nothing is cached either.

`PatruljeListView.vue` is a good first adopter: one list, one obvious entity, and
heavily used during an event.

### What to change

Replace `onMounted(() => load())` plus the local `patruljer` ref with:

```ts
const { data, pending, error, refresh } = useLiveResource(
  'patrulje:list',
  async () => (await http.get('/patrulje')).data.teams.filter(p => p.name !== ''),
  { dependsOn: ['patrulje'] },
)
```

Dependency notes:

- **`'patrulje'` (the type, not an instance)** is what makes a newly signed-up patrol
  appear: a new row has an id this client has never seen, so an instance-keyed
  dependency could never catch it.
- The list also renders status and reward level, which are projected by
  `patruljestatus` and touched by `scan`. Whether those belong in `dependsOn` is the
  interesting question this ticket answers in practice — start with `patrulje` and add
  only what the page demonstrably needs, since each added type widens what invalidates
  the list.

### Notes

- The view is currently `<script setup>` **without** `lang="ts"`; do not convert it as
  part of this ticket — that is unrelated churn and would obscure the diff.
- Keep the existing error toast behaviour; `useLiveResource` exposes `error` rather
  than swallowing it.
- Verify with two browser tabs, which is the only test that proves this works.

## Acceptance Criteria

- [x] `PatruljeListView` uses `useLiveResource` with an explicit `dependsOn`
- [ ] Navigating away and back renders instantly with **no** network request
- [ ] An edit in one tab appears in the other without navigation or reload
- [ ] A newly created patrulje appears in the list without reload
- [x] Loading state only on the genuinely empty first load, not on revalidation
- [x] Errors still surface as they do today
- [x] `npm run test:unit` green; no new `vue-tsc` errors; lint clean on the file
- [x] `npm run build-only` passes

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-09 01:06 — Task created after a two-tab test showed no live update: the
  capability is built but unadopted, which is Phase 3 rather than a defect.
- 2026-08-09 01:08 — Picked up. Plan: swap the onMounted fetch for useLiveResource,
  keeping the file as plain JS and the toast behaviour intact.
- 2026-08-09 01:16 — Implemented. `onMounted(load)` and the local `patruljer` ref are
  gone; the list is now `useLiveResource('patrulje:list', fetcher, { dependsOn:
  ['patrulje'] })` with `patruljer` a computed over `data`.
  Notes:
  • Kept the file as plain `<script setup>` (no `lang="ts"`): converting it would be
    unrelated churn and would bury this diff.
  • `pending` is wired to the DataTable's `:loading`, which gives the intended
    behaviour for free — it is true only when there is nothing cached, so a
    revalidation never flashes a spinner over a populated table.
  • Errors are surfaced by the view rather than swallowed in the fetcher, via a watch
    on `error`, preserving the previous console log and adding a toast.
  • `dependsOn: ['patrulje']` only, for now. The list also renders status and reward
    level, which `patruljestatus` and `scan` project — but each added type widens what
    invalidates the whole list, so they should be added on evidence rather than
    speculation.
- 2026-08-09 01:18 — Verified what can be verified here: 40 frontend tests green
  (including the type-level-invalidation case this page relies on), lint clean apart
  from five pre-existing warnings in the file, `vue-tsc` unchanged at 107, and
  `build-only` passes. Backend end-to-end was already proven with a PATCH producing a
  signal in ~0.1s.
  **Outstanding: the two-tab check.** That is the only test that proves this works, and
  it needs a human with two browsers — left unticked rather than inferred from the unit
  tests.
