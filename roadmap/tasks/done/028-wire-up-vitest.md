# 028 — Wire up the unit test harness (vitest)

**Status:** done
**Priority:** high
**Created:** 2026-08-08
**Picked up by:** agent session (zed)
**Started:** 2026-08-08
**Completed:** 2026-08-08

## Description

`vue/package.json` already has `"test:unit": "vitest"`, but **`vitest` is not in
`devDependencies`** and there are no test files — so the script cannot run and the
frontend has no way to assert behaviour.

This surfaced while implementing 024: that ticket's acceptance criteria are
behavioural (a revisited key issues zero requests, `resync` fans out to every held
entry, a 404 evicts, overlapping revalidations collapse) and none of them are provable
by `vue-tsc` or eslint. Type-checking a cache tells you nothing about whether it
caches.

Pulled ahead of 025–027 deliberately: the remaining Phase 1 tickets all describe
behaviour that wants asserting, so the harness should exist before they are built
rather than after.

Scope is the harness plus the tests that verify 024. Not a coverage drive, and no
component/DOM testing yet — everything in Phase 1 is plain reactive logic, so a jsdom
environment is not needed and should not be added on spec.

## Acceptance Criteria

- [x] `vitest` in `devDependencies`; `npm run test:unit` runs and passes
- [x] Runs headless in one shot in CI/container (no watch mode hanging a container)
- [x] `@/` path alias resolves in tests, matching the app config
- [x] Tests for 024 covering: cache hit issues no second request; `resync`
      revalidates all held entries; a type-level signal revalidates (proving new rows
      appear); an instance-level signal revalidates only its own entry; a 404 evicts;
      concurrent revalidations collapse into one in-flight request
- [x] `roadmap/tasks/done/024-*.md` criteria can be honestly ticked afterwards

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-08 16:50 — Task created mid-024, on discovering the test script has no
  runner behind it. 024 stays in `doing/` until this lands so its behavioural criteria
  can be verified rather than assumed.
- 2026-08-08 16:52 — Picked up. Plan: add vitest + a vitest config reusing the Vite
  alias config, then tests for the 024 behaviours. Node environment (no jsdom) since
  Vue reactivity works without a DOM.
- 2026-08-08 16:58 — Installed `vitest@2.1.9` as a devDependency (container is still
  Node 18 until `docker compose build ui` picks up the Node 20 Dockerfile; vitest 2
  supports both).
- 2026-08-08 17:04 — Added `vue/vitest.config.ts`. Decisions: standalone from
  `vite.config.ts` rather than merged, since these tests need neither the Vue SFC
  plugin nor jsdom — adding either now would be speculative; and `watch: false` so a
  container run cannot hang on a watcher, which is what `npm run test:unit` would
  otherwise do.
- 2026-08-08 17:12 — Wrote 11 tests for `useLiveResource`. Used `effectScope()` to
  mount/unmount as a component would, so `onScopeDispose` is valid and the
  remount-from-cache case is exercised honestly rather than by calling the composable
  twice in one scope.
- 2026-08-08 17:18 — ✅ All 11 pass. `npm run test:unit` exits 0 headless. `vue-tsc`
  still 108 errors, none in the new files (the spec type-checks under the app config,
  which pulls in vitest's types cleanly).
- 2026-08-08 17:20 — Completed. Harness in place; 024's behavioural criteria are now
  verifiable and verified. Remaining Phase 1 tickets (025–027) can assert behaviour
  from the start.
