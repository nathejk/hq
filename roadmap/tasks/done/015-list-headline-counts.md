# 015 — Show entry count in Patrulje and Gøgler list headlines

**Status:** done
**Priority:** low
**Created:** 2026-07-31
**Picked up by:** agent (opus-4.8 session)
**Started:** 2026-07-31
**Completed:** 2026-07-31

## Description

On the patrulje list page (`PatruljeListView.vue`) and the gøgler list page
(`BadutListView.vue`), append the number of rows in the list to the headline,
e.g. `Patruljer (33)` and `Gøglere (12)`. Count reflects the (filtered) array
actually bound to the table.

## Acceptance Criteria

- [x] `PatruljeListView` headline shows `Patruljer ({{ patruljer.length }})`.
- [x] `BadutListView` headline shows `Gøglere ({{ badutter.length }})`.

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-07-31 16:35 — Task created + picked up.
- 2026-07-31 16:38 — Appended `({{ patruljer.length }})` / `({{ badutter.length }})` to the two headlines. (Frontend not lint-run here — trivial template change.) Completed.
