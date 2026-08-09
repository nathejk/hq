# 036 — Adopt live updates on the patrulje list

**Status:** open
**Priority:** high
**Created:** 2026-08-09
**Picked up by:**
**Started:**
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

- [ ] `PatruljeListView` uses `useLiveResource` with an explicit `dependsOn`
- [ ] Navigating away and back renders instantly with **no** network request
- [ ] An edit in one tab appears in the other without navigation or reload
- [ ] A newly created patrulje appears in the list without reload
- [ ] Loading state only on the genuinely empty first load, not on revalidation
- [ ] Errors still surface as they do today
- [ ] `npm run test:unit` green; no new `vue-tsc` errors; lint clean on the file
- [ ] `npm run build-only` passes

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-09 01:06 — Task created after a two-tab test showed no live update: the
  capability is built but unadopted, which is Phase 3 rather than a defect.
