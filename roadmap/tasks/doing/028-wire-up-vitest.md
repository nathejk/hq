# 028 — Wire up the unit test harness (vitest)

**Status:** doing
**Priority:** high
**Created:** 2026-08-08
**Picked up by:** agent session (zed)
**Started:** 2026-08-08
**Completed:**

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

- [ ] `vitest` in `devDependencies`; `npm run test:unit` runs and passes
- [ ] Runs headless in one shot in CI/container (no watch mode hanging a container)
- [ ] `@/` path alias resolves in tests, matching the app config
- [ ] Tests for 024 covering: cache hit issues no second request; `resync`
      revalidates all held entries; a type-level signal revalidates (proving new rows
      appear); an instance-level signal revalidates only its own entry; a 404 evicts;
      concurrent revalidations collapse into one in-flight request
- [ ] `roadmap/tasks/done/024-*.md` criteria can be honestly ticked afterwards

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-08 16:50 — Task created mid-024, on discovering the test script has no
  runner behind it. 024 stays in `doing/` until this lands so its behavioural criteria
  can be verified rather than assumed.
- 2026-08-08 16:52 — Picked up. Plan: add vitest + a vitest config reusing the Vite
  alias config, then tests for the 024 behaviours. Node environment (no jsdom) since
  Vue reactivity works without a DOM.
