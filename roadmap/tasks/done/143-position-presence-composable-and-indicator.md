# 143 — usePositionPresence + PositionIndicator.vue

**Status:** done
**Priority:** high
**Created:** 2026-09-03
**Picked up by:** agent session
**Started:** 2026-09-03
**Completed:** 2026-09-03

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

- [x] `usePositionPresence` with `dependsOn: ['track']`, one shared live resource
- [x] `PositionIndicator.vue` with absent / normal / muted states
- [x] Nothing rendered while presence is loading
- [x] Tooltip shows absolute + relative `da-DK` time
- [x] Screen-reader label; timestamp reachable without hover (`showText` prop)
- [x] Staleness threshold in a single named const (`STALE_AFTER_MS`)
- [x] `npm run build` (or type-check) clean

## Progress Log

- 2026-09-03 — Task created from PRD 011 §10.
- 2026-09-03 — Picked up. Plan: `usePositionPresence.ts` modelled on `composables/kort.ts` (single owner of the fetch, so eight views cannot write eight cache keys), then a small `PositionIndicator.vue` using PrimeIcons + `v-tooltip` like `SosTeamCard`.
- 2026-09-03 — **Added a shared ticking clock (`useNow`), which was not in the plan and is necessary.** Staleness is a function of *now*, not of the data, so a page left open overnight with no new positions would keep rendering every glyph as fresh — claiming recency it does not have, which is the one failure this feature must not have. Live signals refresh data, but *silence produces no signal*, and silence is precisely what is being displayed. One interval for the whole app, reference-counted, so a 200-row table starts one timer rather than 200.
- 2026-09-03 — Staleness default set to **30 minutes**, and the reasoning recorded in the const because both directions are tempting. Sampling is ~30 s so a tight threshold looks defensible — but hour-long gaps are normal here, and a tight threshold would leave most glyphs muted most of the time, at which point the state stops carrying information. Still open in PRD 011 §11; this is a defensible default in one place, not a decision.
- 2026-09-03 — `hasPosition` returns false both for "never reported" and "not loaded yet", so the component checks `loading` separately and renders nothing until presence arrives. Without that, every first paint would assert "never reported" for everybody — and since absence *is* the information here, that would be stating something false rather than merely being unstyled.
- 2026-09-03 — Used PrimeIcons `pi-map-marker` + `v-tooltip`, following `SosTeamCard`'s inline-glyph pattern, rather than pulling FontAwesome into this component. `aria-label` carries the whole sentence so a screen reader gets the timestamp, not the existence of an icon, and `showText` renders it inline for detail views where a hover-only timestamp would be unreachable by touch.
- 2026-09-03 — ✅ 12 unit tests for the pure helpers (`isStale`, `formatRelative`, `formatClock`), including the threshold boundary and a **future** timestamp — a phone with a fast clock would otherwise render "for -1 minutter siden".
- 2026-09-03 — Type-check note: `vue-tsc` reports **106 pre-existing errors** in this repo (`PostlinjeModal`, `PostmandskabModal`, and others). None are in the files this task adds — verified by grepping the output. Not fixing them here; flagging that the project type-check is not currently a usable gate.
- 2026-09-03 — `vite build` clean. Completed.
