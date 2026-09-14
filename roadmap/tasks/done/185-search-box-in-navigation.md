# 185 — Search box and keyboard shortcut in Navigation

**Status:** done
**Priority:** medium
**Created:** 2026-09-14
**Picked up by:** agent session 2026-09-14
**Started:** 2026-09-14
**Completed:** 2026-09-14

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

- [x] Search field in `Navigation.vue`, on every page
- [x] Keyboard shortcut focuses it from anywhere, and does not fire from within another input
- [x] Submitting navigates to `/search?q=…`
- [x] Operable by keyboard alone; result count announced
- [x] No navbar results dropdown (out of scope by decision)
- [x] `npm run build` and unit tests pass

## Progress Log

- 2026-09-14 22:08 — Task created from PRD 014 §10.
- 2026-09-15 00:44 — Picked up. Plan: a small `components/NavSearch.vue` (its own file with
  `lang="ts"`, so it can be mounted and tested — `Navigation.vue` is plain JS and 300 lines of
  menu data), used from `Navigation.vue` so it is present on every page. A `<form role="search">`
  submitting to `/search?q=…`, a document-level shortcut that focuses it, and no dropdown. Shortcut
  candidate is `/`: Cmd/Ctrl+K shadows Firefox's search bar and Chrome's omnibox, `/` shadows
  nothing and is the established in-app convention.
- 2026-09-15 00:50 — Built as its own component, `components/NavSearch.vue`, used from
  `Navigation.vue`. Two reasons: `Navigation.vue` is plain JS and mostly a 150-line menu-item array,
  so a focus-managing form inside it would be untestable; and the shortcut owns a document-level
  listener, which wants an explicit mount/unmount boundary. ✅ It renders inside the persistent nav,
  so it is on every page.
- 2026-09-15 00:54 — ✅ Shortcut is `/`, unmodified. Rejected Cmd/Ctrl+K explicitly: Firefox uses it
  to focus its search bar and Chrome to put the omnibox in search mode, and the criterion says not to
  shadow a shortcut people rely on. `/` also needs no modifier, which is the right property when the
  operator's other hand is holding a phone. `preventDefault` is called **only after** the keystroke
  has been established as ours, so Firefox's quick-find still works for anyone who did not want our
  search.
- 2026-09-15 00:58 — ✅ The shortcut is suppressed for `INPUT`, `TEXTAREA`, `SELECT` and
  `contenteditable`. Blocker: `isContentEditable` is not implemented in jsdom, so the test for the
  Quill case passed vacuously. Rather than delete the test, the check now also walks
  `closest('[contenteditable="true"], [contenteditable=""]')` — which is more correct anyway, since a
  keystroke inside Quill lands on a child node of the editable region rather than on the region
  itself. Tests now cover input, textarea, contenteditable, and the three modifier combinations.
- 2026-09-15 01:02 — ✅ Submit navigates to `/search?q=…`. `push` here, deliberately unlike the
  debounced `replace` inside `SearchView`: this is a navigation the operator chose, so Back should
  return them to the page they came from. An all-whitespace query re-focuses the field instead of
  pushing an empty search.
- 2026-09-15 01:05 — Decision: the box mirrors `route.query.q`. Without it, arriving on
  `/search?q=20309696` from a shared link leaves an empty box above a full page of results, and the
  operator's next keystroke silently starts a new search. The consequence is that on `/search` there
  are two boxes holding the same text — the nav one for starting a search, the page one for refining
  it. Worth questioning; noted for the requester rather than resolved unilaterally, since the
  alternative (hiding the nav box on `/search`) breaks "on every page" and leaves the shortcut with
  nothing to focus.
- 2026-09-15 01:08 — ✅ Keyboard-only operation: a real `<form role="search">`, so Enter submits with
  no key handler and the box is reachable by landmark navigation rather than by tabbing the whole icon
  bar; a visually-hidden `<label>`; an `aria-label` that states the shortcut; Escape clears. The
  result count is announced by `SearchView`'s `role="status" aria-live="polite"` region (task 182),
  which also names how many rows came from other years — the count belongs where the results are, not
  in the chrome.
- 2026-09-15 01:10 — ✅ No dropdown, and a test asserts the absence rather than just the code lacking
  one, so adding it later is a deliberate act.
- 2026-09-15 01:12 — ✅ `npm run test:unit` 341 passed (22 files), `npm run build` clean, `vue-tsc`
  clean on every touched file.
- 2026-09-15 01:14 — Completed. Person search is now reachable from anywhere with one keystroke; the
  feature is no longer URL-only.
