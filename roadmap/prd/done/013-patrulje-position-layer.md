# PRD 013 — The patrol position layer: where every patrulje was last seen

**Status:** done
**Author:** agent session (with knj)
**Created:** 2026-09-09
**Last updated:** 2026-09-09
**Approved:** 2026-09-09
**Shipped:** 2026-09-09
**Target users:** organizer (HQ operators, løbsledelse, SOS/dispatch)

<!--
Written after the fact. PRD 011 §4 listed a live-tracking layer on /kort as a
non-goal, and that non-goal was overtaken during the kort work: the layer was
asked for, built and shipped in one session. This document exists so the
roadmap describes the code rather than contradicting it, and so the decisions
taken under conversation pressure are written down where the next person will
look. It goes straight to done/ because the feature is in production behaviour;
the folder is the status, and the status is shipped.
-->

---

## 1. Summary

`/kort` gains an optional layer showing **each patrulje's last known position** as a
green dot: filled when the position is under five minutes old, hollow when it is
older. Overlapping dots group into one dot carrying a count, which lists its
patruljer when clicked.

## 2. Problem & Motivation

- **What problem does this solve?** During the race, løbsledelse repeatedly asks
  "where is everybody?" — and until now the answer required opening one patrol at a
  time. PRD 011 delivered per-patrol tracks (a *history*, one team per dialog) and a
  presence glyph (does this phone report *at all*). Neither answers the question the
  operations table actually asks, which is about the field as a whole, at a glance.
- **Why now?** PRD 011 shipped the two evidence sources and their endpoints, and PRD
  010 gave `/kort` the sheets, areas and checkpoints an operator judges positions
  *against*. The layer is the small remaining step that turns both into an
  operational picture, and it was requested directly during the kort work.
- **Evidence.** Requested in session by knj, in the same breath as the checkpoint
  circles and the crow-flies lines — i.e. as part of making `/kort` the screen the
  race is run from rather than the screen the course is drawn on.

## 3. Goals

- An operator can see, on one screen, roughly where the field is.
- An operator can tell **"this is current"** from **"this is the last thing we
  heard"** without reading a timestamp.
- Positions never mislead: a dot must not imply knowledge HQ does not have.
- The layer costs nothing when not wanted — it is off until asked for.
- Readable with ~200 patruljer, including where they bunch at the start and finish.

## 4. Non-Goals

- **Per-person positions on the map.** The unit is the patrulje. A screen with 800
  dots answers no question, and a person's own movement is what PRD 011's track
  dialog is for.
- **Interpolation, prediction, ETA or off-route detection.** No derived
  intelligence. A dot is an observation, not an estimate.
- **A safety or accountability signal.** A stale dot means "we do not know", never
  "something is wrong" — see §7.
- **Vehicles.** Dispatch tours (PRD 009) are a separate stream and a separate layer.
- **Klaner and crew.** Only patruljer, for now: the operational question is about the
  racing field. Extending to klaner is a decision, not an oversight — see §11.
- **Producing positions.** As PRD 011: the hej-app owns collection and consent; HQ
  consumes the stream and never calls hej.

## 5. User Stories & Scenarios

- As an **organizer**, I want to see where the patruljer are so that I can judge
  whether the field is spread as expected, without opening twenty pages.
- As an **organizer**, I want to know whether a position is current so that I do not
  send a driver to where a patrol was two hours ago.
- As an **organizer**, I want to know **how many** patruljer are in one spot so that
  a cluster at a post is not mistaken for a single team.

**Happy path.** During the race an operator opens `/kort`, ticks *Sidst kendte
position* in the layers control, and sees the field: filled dots where phones are
reporting now, hollow dots where the newest evidence is a scan or an older report.
Judging those against the sheet areas already drawn on the same map answers "is
anybody outside the ground we printed?".

**Edge cases.**

- **No position at all** for a patrulje (no scans, no telemetry): it is **absent**
  from the layer. Absence means "we know nothing", which is honest; a dot at 0,0 or a
  "last seen: never" marker would be worse than silence.
- **Everything stale.** Normal at night. The layer is then all hollow dots, which is
  the correct picture rather than an alarm.
- **Two patrols in one field.** Grouped into one dot with a count; clicking lists
  them newest-first.
- **A scan with no coordinates.** Skipped, not shown at 0,0 — `scan.latitude` is a
  VARCHAR that may be `""` or `"0"`.
- **A phone with a fast clock** reporting a position "in the future": treated as
  live, since it is certainly not stale.

## 6. Requirements

### Functional

- [x] One endpoint returns, for the year, the last known position of every patrulje
      that has one: team id, number, name, coordinates, timestamp, and which kind of
      evidence it came from.
- [x] The position is the **newer** of (a) the newest reported position among the
      team's members and (b) the team's newest QR scan that has usable coordinates.
- [x] Patruljer with no position are omitted rather than returned empty.
- [x] Timestamps are epoch **milliseconds** in the response, converted once at the
      read boundary — `scan.uts` is in seconds and `track_latest.ts` in milliseconds.
- [x] `/kort` offers the layer as a checkbox in the Leaflet layers control, off by
      default.
- [x] A position under five minutes old is drawn filled; older is drawn as an
      outline. Both are green.
- [x] Dots that would overlap on screen are grouped into a single dot showing the
      number of patruljer in it.
- [x] Clicking a group lists its patruljer, newest first, each with its age and
      whether the evidence was a scan or telemetry.
- [x] Dots re-style as they age, without new data arriving.
- [x] The layer updates live, from the same signals that drive the rest of `/kort`.
- [x] A development-only simulation renders ~200 fake patruljer, so density and
      grouping can be judged before an event exists.

### Non-Functional

- **Performance.** The endpoint is polled by a map layer and must cost a fixed
  number of queries regardless of team count — ~200 patruljer with several members
  each must not become a query per team.
- **Truthfulness over completeness.** Where the two evidence sources disagree, the
  newer wins and says which it is; nothing is averaged or smoothed.
- **Privacy.** The layer shows a *team's* position, not a person's, and names no
  individual — the coarsest granularity that answers the question. Position history
  is the most personal data HQ holds (see PRD 011 and
  `roadmap/api/telemetry-erasure.md`).
- **Live.** No page refresh; the layer follows the house live-update mechanism
  (PRD 004).

## 7. UX / UI Notes

- **Where.** `KortView.vue`, as an overlay in `L.control.layers` beside
  *Fugleflugslinjer*. Leaflet already provides that checkbox, and it is where an
  operator looks for things to switch on over the map.
- **The dot.** Green in both states, and **never red**. This is the most important
  UI decision here: a stale position means *we do not know*, and a phone in a pocket
  on battery-saver is indistinguishable from a phone at the bottom of a lake. Red
  would make an absence of data look like an emergency, on the one screen where that
  misreading is most expensive.
- **Filled vs hollow.** Filled = under five minutes = "act on this". Hollow = "this
  is the last thing we heard". Hollow is the *default* appearance during a night
  race, which is why the filled state is the one that stands out.
- **Five minutes**, deliberately much tighter than the 30 minutes
  `usePositionPresence` uses to mute its glyph, because the two answer different
  questions: there, "does this person report at all?", where hour-long gaps are
  routine; here, "can I act on this dot right now?", where a twenty-minute-old
  position is a patrol that could be two kilometres away in any direction.
- **Groups.** One glyph type for both cases, sized by how many are in it: a group is
  the same thing as a dot, only more of it, and a second icon design would suggest
  otherwise. The count is rendered rather than implied, because "bigger" cannot
  distinguish three from thirty.
- **A group is live if *any* member is.** With "all", one stale patrol in a group of
  ten would hollow out the dot and hide nine live ones.
- **Popup.** A single dot shows the patrol, its age, its clock time and its source. A
  group leads with the count, then up to twelve patruljer newest-first, then "… og N
  mere" — the popup is for orientation, and thirty names in a scrolling box is a
  table pretending to be a tooltip.

## 8. Technical Considerations

- **Frontend (Vue 3 / TS).** `composables/patruljePositions.ts` owns the resource,
  the five-minute rule and the clustering; `views/KortView.vue` owns the Leaflet
  layer and the popups. Loading goes through `useLiveResource` with
  `dependsOn: ['track', 'qr', 'patrulje']` — the event *subjects'* entity tokens, not
  the projections' names: scans arrive on `NATHEJK.*.qr.*.scanned`, so the token is
  `qr` and there is no `scan` token.
- **Clustering.** Hand-rolled as a pure function taking a projection, rather than
  adding `leaflet.markercluster`. The requirement is one gesture over a few hundred
  points in a small area, while the library brings an icon set, a stylesheet,
  spiderfy and click-to-zoom that would have to be configured back out. A fixed
  **pixel** grid, because overlap is a property of the screen — so grouping is
  recomputed on `zoomend`. It is O(n) and stable between redraws; its one artefact is
  that two dots either side of a cell boundary stay separate, which is harmless for
  "is there more than one here?".
- **Liveness is re-evaluated on a timer** (one minute). Liveness is a fact about
  *now*, so with no new data a dot filled at 21:58 would still be filled at 22:30 —
  a five-minute promise quietly becoming a half-hour lie.
- **BFF (Go).** `cmd/api/telemetry_positions.go`, with the merge rule
  (`mergePosition`) as a pure, table-tested function. Reads three projections:
  `patrulje` (identities), `track_latest` joined via `spejderstatus`, and `scan`.
  `spejderstatus` rather than `spejder`, because the latter hard-deletes withdrawn
  scouts and would omit exactly the people whose movement is being reconstructed.
- **API endpoints.** `GET /api/telemetry/positions` — new, **with OpenAPI
  annotations** in the house style, whose description states what a client cannot
  infer: that a position may come from a scan or from telemetry, that `ts` is
  milliseconds, and that absence means "no position known" rather than an error.
  Registered under `/api/telemetry/` because `httprouter` refuses a static segment
  beside `/api/patrulje/:id`; a test asserts the router builds rather than trusting
  it.
- **Query strategy.** Three queries per request, independent of team count. Each
  source is scanned once for the year in ascending time order and folded to one
  candidate per team in Go. Newest-per-team is deliberately **not** done in SQL: a
  bare-column `GROUP BY` beside `MAX(ts)` can return one member's coordinates with
  another's timestamp.
- **Data / storage.** No schema change. This is a read model over projections PRD
  011 and earlier PRDs already built.
- **Dependencies & risks.** Inherits PRD 011's telemetry risks, including the
  year-scoped consumer decision (task 153). The simulation is dev-only and must not
  reach production behaviour. `scan`'s VARCHAR coordinates are a known wart:
  unusable values are filtered in SQL *and* during the fold, so a fix-less newest
  scan cannot shadow an older good one.

## 9. Success Metrics

- An operator can answer "where is the field?" without opening a patrol page.
- Endpoint cost stays flat as teams are added: three queries at 20 teams and at 200.
- No dot on screen is older than the newest evidence HQ holds for that team.
- The layer is used during the race — if løbsledelse leaves it off, the dot design
  or the grouping is wrong, and that is the signal to revisit §7.

## 10. Rollout / Task Breakdown

Shipped in one session, without board entries — which is the process deviation this
document exists to record rather than to justify. Had it been planned, the tasks
would have been:

- [x] Task: `GET /api/telemetry/positions` — merge telemetry and scans per patrulje
- [x] Task: `patruljePositions` composable — resource, liveness rule, clustering
- [x] Task: `Sidst kendte position` overlay in `/kort`'s layers control
- [x] Task: dev-only simulation of ~200 patruljer

Follow-ups, created on the board as this PRD was written:

- [ ] 172 — Decide whether klaner get a position layer too
- [ ] 173 — Mark low-confidence positions (large `accuracy`) on the map

## 11. Open Questions

- **Is five minutes the right line?** Chosen by reasoning, not evidence, and
  correctable in one place (`LIVE_WITHIN_MS`). The real distribution of reporting
  gaps during a race will say; until then it is a defensible default, not a decision.
- **Is a 34 px grouping distance right?** It reads well with the simulated 200, but
  simulated clumps are not real clumps.
- **Does a group of thirty want a popup or a panel?** The popup caps at twelve. If
  operators routinely hit that cap, the answer is a side list rather than a longer
  popup.
- **Klaner.** Same layer, same rules? The endpoint would extend naturally; the
  question is whether the operational picture is helped or crowded (task 172).
- **Low-confidence fixes.** `accuracy` is stored and available. A multi-kilometre
  cell-tower fix currently looks exactly like a GPS fix, which slightly overstates
  what HQ knows (task 173).
