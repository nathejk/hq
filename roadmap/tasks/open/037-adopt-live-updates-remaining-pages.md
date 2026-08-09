# 037 — Adopt live updates on the remaining pages

**Status:** open
**Priority:** medium
**Created:** 2026-08-09
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] Betalinger, klaner and poster use `useLiveResource` with explicit `dependsOn`
- [ ] Each verified in two tabs: an edit in one appears in the other
- [ ] Navigating between adopted pages issues no refetch for cached ones
- [ ] No page depends on `scan` unless it demonstrably needs it
- [ ] `npm run test:unit` green; no new `vue-tsc` errors; lint clean on touched files
- [ ] `npm run build-only` passes

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-09 01:28 — Task created after 036 proved the pattern in two tabs.
