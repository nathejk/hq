# 185 — Search box and keyboard shortcut in Navigation

**Status:** doing
**Priority:** medium
**Created:** 2026-09-14
**Picked up by:** agent session 2026-09-14
**Started:** 2026-09-14
**Completed:**

## Description

Depends on task 184. The entry point: a search field in `vue/src/components/Navigation.vue`,
present on every page, with a keyboard shortcut that focuses it from anywhere.

It is in the chrome rather than on a page of its own because the operator's hands are on the
phone, not on the navigation. The target interaction for an inbound call is: shortcut, eight
digits, one click.

Submitting navigates to `/search?q=…` (task 182). The field itself may show nothing; a
dropdown of live results in the navbar is explicitly not required here and should not be added
without asking — it competes with the results view for the same job.

Accessibility is a requirement, not a nicety: reachable and operable by keyboard alone, and the
result count announced. The shortcut must not fire while the operator is typing into another
input, and must not shadow a browser shortcut people rely on.

Last task before the latency check (186). Until this lands the feature is reachable only by
typing the URL, which is deliberate — it means an incomplete feature was invisible rather than
broken.

## Acceptance Criteria

- [ ] Search field in `Navigation.vue`, on every page
- [ ] Keyboard shortcut focuses it from anywhere, and does not fire from within another input
- [ ] Submitting navigates to `/search?q=…`
- [ ] Operable by keyboard alone; result count announced
- [ ] No navbar results dropdown (out of scope by decision)
- [ ] `npm run build` and unit tests pass

## Progress Log

- 2026-09-14 22:08 — Task created from PRD 014 §10.
- 2026-09-15 00:44 — Picked up. Plan: a small `components/NavSearch.vue` (its own file with
  `lang="ts"`, so it can be mounted and tested — `Navigation.vue` is plain JS and 300 lines of
  menu data), used from `Navigation.vue` so it is present on every page. A `<form role="search">`
  submitting to `/search?q=…`, a document-level shortcut that focuses it, and no dropdown. Shortcut
  candidate is `/`: Cmd/Ctrl+K shadows Firefox's search bar and Chrome's omnibox, `/` shadows
  nothing and is the established in-app convention.
