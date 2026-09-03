# 126 — `useKort` live resource composable

**Status:** doing
**Priority:** medium
**Created:** 2026-09-03
**Picked up by:** agent session (Zed)
**Started:** 2026-09-03
**Completed:**

## Description

PRD 010 §6 (non-functional), §8 (Frontend). Depends on 125.

`vue/src/composables/useKort.ts` wraps `useLiveResource` around `GET /api/kort` so
`KortView` and the settings modal share **one** cached source rather than fetching twice.

```ts
useLiveResource('kort:all', fetcher, {
  dependsOn: ['kort', 'kortsaet', 'checkpoint', 'checkgroup']
})
```

The dependency tokens are the event *subjects'* entities, per `.rules`. `checkpoint` and
`checkgroup` are included because the payload's usefulness depends on them: a renamed
checkpoint must show its new name in the picker.

No `onMounted` + `http.get` into a local `ref` — that is the regression `.rules` calls out.
Wire `pending` to any `:loading`; do not add a spinner, since `pending` is only true when
nothing is cached and a revisited page must not flash.

## Acceptance Criteria

- [ ] `useKort` returns data / error / pending / refresh from `useLiveResource`
- [ ] `dependsOn` as above, with no dev-console warning about unemittable tokens
- [ ] No component fetches `/kort` directly
- [ ] `npm run build` (or `vue-tsc`) clean

## Progress Log

- 2026-09-03 — Task created from PRD 010 §10.
- 2026-09-03 — Picked up. Plan: `composables/kort.ts` in the shape of `composables/sos.ts` and
  `dispatch.ts` — types, Danish labels, and one function that calls `useLiveResource` with the
  canonical key and dependency list, so the view and the dialog cannot disagree about either.
