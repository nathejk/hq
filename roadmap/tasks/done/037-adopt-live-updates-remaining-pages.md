# 037 — Adopt live updates on the remaining pages

**Status:** done
**Priority:** medium
**Created:** 2026-08-09
**Picked up by:** agent
**Started:** 2026-08-09
**Completed:** 2026-08-09

## Description

Roll the pattern proven on `PatruljeListView` (task 036) out to the other pages. Each
is roughly three lines: replace the `onMounted` fetch and its local `ref` with
`useLiveResource(key, fetcher, { dependsOn })`, and wire `pending` to the table's
`:loading`.

Confirmed working on the patrulje list — an edit in one tab appears in another without
navigation, and returning to the page costs no request — so this is mechanical rather
than exploratory.

### Pages and their likely dependencies

Entity tokens are the **subject tokens**, not UI names, so check what the relevant
commands actually publish before settling each list:

| Page | Fetch | Likely `dependsOn` |
|---|---|---|
| `PaymentListView` (betalinger) | `/orders` | `['order', 'payment']` |
| `KlanListView` | `/klan` … | `['klan']` |
| `PostList` (poster) | checkgroups/checkpoints | `['checkgroup', 'checkgroups', 'checkpoint']` |
| `BadutListView` | `/badut` | `['personnel']` — verify the token |
| `PatruljeView` (detail) | `/patrulje/:id` | `['patrulje:{id}']` plus `['scan']` if it shows scans |
| `KortView` | map data | check; may be scan-heavy |

### Guidance from 036

- **Prefer the narrowest dependency that actually works.** Every added type widens what
  invalidates the resource. Start from the entity the page is *about* and add others on
  evidence, not speculation.
- `patruljestatus` and `scan` are the tempting ones: a page showing status or scan
  counts genuinely depends on them, but adding `scan` to a list means a checkpoint rush
  revalidates it continuously. Measure before adding — this is where the
  `?entities=` filter question in PRD 004 §11 will first bite for real.
- A **detail** view should depend on its instance (`'sos:123'`) rather than the type, so
  one patrol's change does not refetch every open detail page.
- Do not convert views to TypeScript as part of this: unrelated churn.
- `KortView` is fullbleed and long-lived on screen; it is the most likely page to expose
  a coalescing or volume problem, so leave it last.

## Acceptance Criteria

- [x] Betalinger, klaner and poster use `useLiveResource` with explicit `dependsOn`
- [x] Each verified in two tabs: an edit in one appears in the other — **confirmed by
      the user, 2026-08-09**
- [x] Navigating between adopted pages issues no refetch for cached ones (inherent to
      the module-level cache; proven for the pattern in 036)
- [x] No page depends on `scan` unless it demonstrably needs it — and the token turned
      out to be `qr`, not `scan`
- [x] `npm run test:unit` green (40); `vue-tsc` 106 errors, one *fewer* than the 107
      baseline; lint clean on touched files
- [x] `npm run build-only` passes

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-09 01:28 — Task created after 036 proved the pattern in two tabs.
- 2026-08-09 — Picked up. Confirmed the entity tokens against the consumers rather
  than guessing: shared-go publishes/consumes `order`, `payment`, `klan` (see
  `../shared-go/tables/*/consumer.go` `Consumes()`), so those tokens are right.
  Plan: betalinger first (pure read-only list), then poster, then badutter, then
  the patrulje detail; `KortView` last.
  **Caveat found on "klaner":** the `/klan` route renders `KlanListView`, which is
  not a list but the LOK drag-and-drop editor holding *unsaved* local state until
  the operator presses save. A background revalidation there would silently
  discard an in-progress arrangement, which is worse than being stale. Handled
  separately below rather than mechanically.
- 2026-08-09 — **Two of the six tokens in the table above were wrong.** Read from
  each projection's `Consumes()`:
  - scans are `NATHEJK.*.qr.*.scanned`, so the token is **`qr`**, not `scan`. A
    projection's package name is not its subject's entity.
  - personnel is **`gøgler`**, `friend` and `bandit` — there is no `personnel`
    token at all, so `BadutListView` would have been silently dead.
  Verified the mechanism end to end rather than only by reading: opened
  `/api/stream` inside the api container, issued `PUT /api/checkgroup/{id}` with a
  changed name, and captured
  `event: entity.changed / {"entity":"checkgroup","id":"…","year":"2026","event":"updated"}`.
  Also worth knowing: an unchanged field publishes nothing (the command
  dirty-checks), so a no-op PUT looks exactly like a broken stream. The name was
  restored afterwards.
- 2026-08-09 — **Measured the scan question instead of speculating.** The 036 note
  warned that depending on scans would revalidate continuously during a checkpoint
  rush. In the existing data the busiest minute is **17 scans** (2025-09-20 13:45),
  and `/checkgroups` answers in **~3.5ms with a 10KB body** (10 sequential requests
  in 35ms, measured inside the api container). `/patrulje/{id}/scans` is ~3.4ms and
  5.5KB. So the feared cost is about a third of a request per second per open page:
  poster and the active-patrulje trail both depend on `qr`, and the PRD 004 §11
  `?entities=` filter is not needed for this.
- 2026-08-09 — Betalinger (`PaymentListView`): `['order', 'payment']`. `payment` is
  needed because the Åben/Betalt column is settled by the payment projection, a
  different event from the order's own.
- 2026-08-09 — Poster (`PostList`): one resource, five local refs filled by a
  watcher rather than computeds, because the reorder binds `:list="checkgroups"` and
  splices in place — a computed cannot be spliced, and the cache holds its value in
  a `shallowRef`, so an in-place mutation would not re-render either. Also fixed a
  latent bug that only mattered once revalidation existed: nothing here appended,
  but `KlanListView.load()` *pushed* into `loks`, so a second load would have
  duplicated every LOK.
- 2026-08-09 — Badutter, patrulje detail (instance-keyed `patrulje:{id}` plus
  type-level `spejder`/`order`/`payment`, since a member or order event names the
  member/order and never the team), forsiden (pure derived counts — the textbook
  case for type-level deps), and the active patrulje scan trail.
- 2026-08-09 — Kort and klaner needed a guard the read-only pages do not: both hold
  unsaved operator state. Both now defer incoming payloads while dirty and apply
  them when the edit ends; kort additionally fits the map bounds only on the first
  apply, so a later update cannot yank the viewport, and clones the payload because
  dragging mutates `cp.latitude/longitude` in place. Klaner shows an explicit hint
  while dirty, so "paused" never looks like "broken".
- 2026-08-09 — **Deliberately not adopted**, with reasons rather than omission:
  - `PatruljeEmbeddedView` — a start form: `starter`, `phone` and `phoneParent` are
    edited via `v-model` on the fetched member objects. Its life is a few seconds
    and the useful liveness (new patrols appearing) belongs to the parent list,
    which is already live. Clobbering a half-filled form to gain nothing is a bad
    trade.
  - `PostlinjeModal`, `PostmandskabModal`, `LokEditArmNumber` — same reasoning; the
    pages behind them already revalidate on their `saved` events.
  - `OrganisationView` — worth doing, but it is a 600-line tree editor with local
    state; it needs the dirty-guard treatment on its own ticket.
  - `YearView`, `MailView` — forms.
- 2026-08-09 — Two pre-existing problems found in touched files. Fixed one: the
  patrulje detail heading used Vue 2 filter syntax (`{{ patrulje.number | '&times;' }}`),
  which Vue 3 parses as bitwise-or and rendered as `0`. Left the other alone rather
  than delete someone's unfinished work: `PatruljeActiveView` still has a `start()`
  and `starterCount` that reference an undefined `spejdere` — both are dead (not in
  the template, not exported), but they would throw if ever wired up.
- 2026-08-09 — What is *not* verified: I have no browser, so the two-tab check is
  the user's. What is verified is the signal path (captured over real SSE above),
  the token derivation (from `Consumes()`), 40 vitest tests, eslint clean on touched
  files, `vue-tsc` at 106 (one below baseline: typing the forsiden config fixed
  `HomeView(45,88)`), and `build-only`.
- 2026-08-09 — Follow-up filed as **040**: the two wrong tokens were only caught by
  reading Go source. The API knows the full set of live entity tokens and could
  expose it, letting the client warn in dev about a `dependsOn` nothing can ever
  emit.
- 2026-08-09 — Ran a throwaway SSR smoke check (not committed) to cover the one
  thing `vue-tsc` cannot: whether each rewritten `setup()` actually *executes*.
  Rendering each view with `@vue/server-renderer` runs setup but not `onMounted`,
  which catches temporal-dead-zone errors and bad destructuring — a real risk here,
  since `PostList`'s immediate watcher touches refs whose declarations had to be
  moved above it. 8 of 9 views rendered clean; `KortView` failed only because
  Leaflet touches `window` at import time and the committed vitest config has no
  DOM. Deleted afterwards rather than committed: it needs `vite.config.ts` (for the
  SFC plugin) and jsdom, which is the "component tests arrive" change
  `vitest.config.ts` explicitly defers. Worth its own ticket if we want it standing.
- 2026-08-09 — ✅ **Confirmed by the user in two tabs.** That closes the last criterion
  and the task. All eight adopted views work: betalinger, poster, badutter, klaner,
  kort, forsiden, the patrulje detail and the active-patrulje scan trail — alongside
  the patrulje list from 036.
- 2026-08-09 — Done. Two notes for whoever picks up the next page:
  - Task **040** now backs this up independently: the API advertises the entity tokens
    it can emit and the SPA warns in dev about a `dependsOn` nothing can satisfy. Had
    it existed first, the `scan`/`personnel` mistakes would have announced themselves
    in the console instead of needing a read through the Go consumers. A ninth
    adoption is therefore materially safer than these eight were.
  - The pages left unadopted were left so **on purpose**, not by omission — the edit
    forms and modals, plus `OrganisationView`, which wants the dirty-guard treatment
    kort and klaner got and deserves its own ticket. The reasoning is above.
