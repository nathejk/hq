# 126 — `useKort` live resource composable

**Status:** done
**Priority:** medium
**Created:** 2026-09-03
**Picked up by:** agent session (Zed)
**Started:** 2026-09-03
**Completed:** 2026-09-03

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

- [x] `useKort` returns data / error / pending / refresh from `useLiveResource`
- [x] `dependsOn` as above, with no dev-console warning about unemittable tokens
- [x] No component fetches `/kort` directly
- [x] `npm run build` (or `vue-tsc`) clean

## Progress Log

- 2026-09-03 — Task created from PRD 010 §10.
- 2026-09-03 — Picked up. Plan: `composables/kort.ts` in the shape of `composables/sos.ts` and
  `dispatch.ts` — types, Danish labels, and one function that calls `useLiveResource` with the
  canonical key and dependency list, so the view and the dialog cannot disagree about either.
- 2026-09-03 — Landed as `vue/src/composables/kort.ts`. It is a plain function over
  `useLiveResource`, **not** a store and holding no state: the value already lives in the module
  cache, and duplicating it in Pinia is how the legacy dims channel ended up with two read models
  (PRD 004 §2.1). `composables/sos.ts` makes the same point.
- 2026-09-03 — The reason for centralising the *load* rather than just the types: the cache is keyed
  by string, so two components inlining `useLiveResource` are two chances to write a different key
  or a different `dependsOn`. Both fail silently — a mismatched key gives each component its own
  copy that the other's writes never reach, and a missing token gives a page that looks live and
  never updates. `KORT_KEY` and `KORT_DEPENDENCIES` are exported and pinned by a test, because the
  tokens are not guessable from the UI's vocabulary.
- 2026-09-03 — Confirmed all four tokens against the API's own boot log rather than assuming them:
  `kort`, `kortsaet`, `checkpoint`, `checkgroup` all appear in the advertised entity set, so the
  dev-console dependency warning stays quiet.
- 2026-09-03 — Put `someMapContainsAll` and `checkpointsWithoutMap` here now, with tests, rather
  than leaving them for tasks 128 and 133. They encode the two rules easiest to get subtly wrong,
  and both are far cheaper to pin as functions than to verify later through a dialog: the
  containment test is **existential, not partitioning** (two overlapping sheets that each hold the
  whole checkgroup are fine, because overlap is designed in), and "unassigned" is **per set**,
  since a checkpoint missing from the crew maps is a different mistake from one missing from the
  patrol maps.
- 2026-09-03 — `import http from '@/plugins/axios'` was wrong — the module has no default export.
  `vue-tsc` caught it. Worth noting the repo has **107 pre-existing type errors** (mostly PrimeVue
  `Date | null` mismatches in `PostmandskabModal.vue`), so "typecheck is clean" is not available as
  a signal here; I checked instead that zero errors mention this feature's files.
- 2026-09-03 — Completed. 13 tests pass (`vitest run src/composables/kort.spec.ts`), no type errors
  in the new files. Ran both inside the `hq-ui` container, since node is not on the host.
