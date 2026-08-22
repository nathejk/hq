# 092 — HoensegaardView: the grouped list

**Status:** open
**Priority:** high
**Created:** 2026-08-23
**Picked up by:**
**Started:**
**Completed:**

## Description

The screen itself, read-only in this task (PRD 007 §7). New
`vue/src/views/HoensegaardView.vue`, route `/hoensegaard` name `hoensegaard` in
`router/index.ts` (lazy-loaded, as every route but Home), nav entry in
`components/Navigation.vue` labelled **Hønsegården** between *Nødtelefon* and *Betalinger* —
it is a race-night screen and belongs beside the other one. Icon `fas fa-kiwi-bird`
(FontAwesome Free has no chicken).

Sections, in this order and not merged:

1. **I bil — på vej hertil** (`transit`) — the arrivals queue, first because it is the only
   section with somebody standing in front of it. Collapses to one quiet line ("ingen på
   vej") when empty rather than an empty table, so it does not push the screen down all
   night.
2. **I Hønsegården** (`sheltered`) — with placering.
3. **Afventer afhentning** (`waiting`).
4. **Afsluttet** (`reunited`, `released`) — collapsed by default.

`transit` and `waiting` stay apart deliberately: they look alike on a status list and mean
opposite things here — a scout in a car is acted on within minutes, a scout by a road must
not be acted on. Merging them buries the actionable rows.

Header strip with the group counts and the in-our-care total, so the shelter sees the same
number the organisers are waiting on.

Rows show name, patrol (linking to the patrol page), status with its Danish label **from the
server's `memberStatuses`** (no local label map — PRD 006 §6), "siden 21:40 (2t 14m)" in
`da-DK` from `updatedAt`, phones, and a link to the open case. Waiting longer than the alarm
threshold (task 082) is highlighted.

**Live, as required of every new page:**

```js
useLiveResource('shelter', fetcher, { dependsOn: ['spejder', 'patrulje'] })
```

`spejder` is the entity in the lifecycle subjects — *not* `spejderstatus`, which is only the
projection's name, and the SPA warns in the dev console about a dependency nothing can emit.
Type-level, not instance-level: a newly withdrawn scout's id has never been seen before.
Wire `pending` to each table's `:loading` and add no spinner of your own — `pending` is true
only when nothing is cached, so a revisited page must not flash.

3am legibility: large targets, high contrast, no destructive action behind an unguarded
click.

## Acceptance Criteria

- [ ] View, route and nav entry added; nav icon renders
- [ ] Four sections in the order above; `transit` first; `waiting` separate from `transit`
- [ ] Empty *I bil* renders as one line, not an empty table
- [ ] *Afsluttet* collapsed by default
- [ ] Status labels come from the server payload
- [ ] Durations in `da-DK`, both clock time and elapsed
- [ ] `useLiveResource` with `dependsOn: ['spejder', 'patrulje']`; no dev-console warning
      about an unemittable dependency
- [ ] `pending` drives `:loading`; revisiting the page does not flash
- [ ] Verified live in two browser tabs: a status change in one appears in the other

## Progress Log

- 2026-08-23 09:30 — Task created from PRD 007 §10.
