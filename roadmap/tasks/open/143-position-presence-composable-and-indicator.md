# 143 — usePositionPresence + PositionIndicator.vue

**Status:** open
**Priority:** high
**Created:** 2026-09-03
**Picked up by:**
**Started:**
**Completed:**

## Description

PRD 011 §7, §8. Depends on task 142.

Two pieces, used everywhere afterwards by task 144:

`vue/src/composables/usePositionPresence.ts` — one shared
`useLiveResource('telemetry:presence', …, { dependsOn: ['track'] })` exposing
`hasPosition(id)` and `lastSeenAt(id)`. The dependency token is **`track`** — the subject's
entity — not `position`, not `telemetry`. Getting this wrong means the indicator never
updates; the SPA warns in the dev console about a dependency nothing can emit.

`vue/src/components/PositionIndicator.vue` — a small location pin after a person's name,
taking a person id. Three states only: **absent** (never reported — no glyph, no tooltip, no
empty state in the row), normal, and muted beyond a staleness threshold. Tooltip
`Sidst set 14:32 · for 6 minutter siden`, `da-DK`.

Two rules that are easy to get wrong:

- While presence is still loading, render **nothing** rather than a wrong negative. Absence of
  a glyph is meaningful, so it must not appear before the data does.
- The muted state means **"we do not know"**, not "something is wrong". Hour-long gaps are
  normal (phones locked, backgrounded, dead), so this must not read as a safety signal. It
  answers "can I expect location data from this person?" and no more.

Accessibility: a text label for screen readers, and the timestamp reachable without hover.

Staleness threshold is an open question in PRD 011 §11 — pick a defensible default (suggest
30 min), keep it in one exported const, and note the choice in the log.

## Acceptance Criteria

- [ ] `usePositionPresence` with `dependsOn: ['track']`, one shared live resource
- [ ] `PositionIndicator.vue` with absent / normal / muted states
- [ ] Nothing rendered while presence is loading
- [ ] Tooltip shows absolute + relative `da-DK` time
- [ ] Screen-reader label; timestamp reachable without hover
- [ ] Staleness threshold in a single named const
- [ ] `npm run build` (or type-check) clean

## Progress Log

- 2026-09-03 — Task created from PRD 011 §10.
