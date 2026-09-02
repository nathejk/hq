# PRD 011 — Telemetry: who has reported a position, and where they went

**Status:** draft
**Author:** agent session (with knj)
**Created:** 2026-09-03
**Last updated:** 2026-09-03
**Approved:**
**Shipped:**
**Target users:** organizer (HQ operators, løbsledelse, SOS/dispatch), and — indirectly — participants whose hej-app reports the positions

<!--
Status must match the folder this file is in: draft/, doing/ or done/.
Leave Approved blank until the PRD moves to doing/, and Shipped blank until it
moves to done/. See roadmap/prd/README.md for the lifecycle.
-->

---

## 1. Summary

The hej-app now posts positions to a new JetStream stream, `telemetry`. This PRD
consumes that stream in HQ and delivers two things: a small **position indicator
next to every person's name** across HQ — present only if that person has ever
reported a position, with the last-received timestamp on hover — and the data
model plus read endpoints needed to **show a person's full track**, so the map
work that follows is a UI task and not a data task.

For a spejder the track is not a single line. The unit that matters during the
race is the patrulje, so the patrol view must show the tracks of **all its
members, current and former**, together with **every QR scan registered for that
patrulje** — one picture of where that patrol has been, regardless of which body
carried which phone.

## 2. Problem & Motivation

**What problem does this solve?**

HQ currently answers "where is this patrol?" from QR scans alone
(`go/nathejk/table/scan/`): one coordinate per checkpoint scan, minutes or hours
apart, and only at the places we put a post. `go/nathejk/table/dispatch/tours.go`
says so explicitly — a tour is what makes "when?" answerable *without GPS*. That
was the right stance while no positions existed. Positions now exist, and three
concrete gaps open up:

- **Nobody can tell whether a person is reachable by telemetry at all.** When a
  SOS case comes in, or dispatch needs to find a patrol, the first question is
  "does this person's phone report?" Today that question has no answer anywhere
  in HQ, so an operator has to try and find out. A single glyph next to the name
  turns a guess into a fact, and its hover timestamp says how stale the fact is.
- **A trail is qualitatively different from a scan.** Scans say a patrol was at
  posts 4 and 6; a trail says which way they walked between them, whether they
  stopped for two hours, and where to look when they stop answering. This is the
  difference between reconstructing an incident afterwards and finding someone
  during it.
- **Membership churn breaks per-person views.** Scouts move between patrols
  mid-race (`NATHEJK.*.spejder.*.team.moved`), withdraw, and are handed over;
  `spejderstatus` tracks `initialTeamId` vs `currentTeamId` for exactly that
  reason. A per-person track is therefore not enough on its own — the patrol's
  history is spread across the people who have passed through it.

**Why now?**

The stream is live and producing; the data is being written whether or not HQ
reads it. Consuming it early in the season means the retention and volume
questions are answered against real traffic rather than at the event. The two
halves also carry very different cost: the indicator is cheap, broad and useful
immediately, while the map is narrow and expensive — so shipping the indicator
plus a clean read model first is what lets the map be built without guesswork.

**Evidence.** Nothing named telemetry, position or gps exists anywhere in `go/`
or `vue/src/` today (`types.Position` is used only for checkpoints and kort
corners). PRD 001 (nødtelefon/SOS) and PRD 009 (dispatch) both resolve
"where?" by asking a human on the phone.

## 3. Goals

- An operator can see, anywhere a person's name is listed in HQ, whether that
  person has ever reported a position, and how recently.
- The last-known position and the full point history of a person are available
  over stable read endpoints, live-updating, without HQ having to talk to the
  hej-app or to NATS directly.
- For a patrulje, HQ can retrieve one combined picture of the patrol's movement:
  the tracks of everyone who has been a member, plus that patrol's scans.
- Reading a second stream is a wiring exercise, not a rewrite: adding a
  `telemetry` projection must not require a new mux, a new SSE channel, or a
  parallel live mechanism.

## 4. Non-Goals

- **Producing positions.** The hej-app owns collection, consent, batching and
  upload. HQ only consumes.
- **Live-tracking UI on `/kort`.** Drawing every participant's current position
  on the main map is the obvious next feature and is deliberately out of scope;
  this PRD delivers the endpoints it would need. `/kort` stays checkpoint-focused
  (PRD 010).
- **Geofencing, alerts, off-route detection, or ETA prediction.** No derived
  intelligence in this PRD — only faithful storage and display.
- **Positions for non-people.** Vehicles (PRD 009) are not in this stream yet;
  the schema should not preclude them, but no vehicle work is included.
- **Backfilling scans into the track model.** Scans stay in `scan`; the patrol
  view joins the two at read time rather than merging the stores.
- **Changing the hej-app.** If the subject shape or payload needs adjusting (see
  Open Questions), that is work in the app's repo, tracked separately.

## 5. User Stories & Scenarios

- As an **HQ operator** looking at the badut list, I want a position glyph next
  to each gøgler so I can see at a glance which of them I can locate.
- As a **SOS operator** on a call about a patrol, I want to open the patrol and
  see immediately which members' phones are reporting, and when each last did,
  so I know whose phone to ask them to check.
- As **løbsledelse** reconstructing an incident, I want to see the patrol's full
  movement — every member who has been on the team, plus the scans — on one map
  so I can say where they actually were at a given hour.
- As an **organizer**, I want the indicator to be honest: absent, not merely
  greyed, when the person has never reported, so its presence carries meaning.

**Happy path (indicator).** Operator opens `/badut`. The list renders from cache
instantly (PRD 004). A small pin appears next to 43 of 61 names. Hovering one
shows "Sidst set 14:32 (for 6 minutter siden)". A gøgler's phone reports for the
first time; within a second the pin appears next to their name without a
refresh.

**Happy path (patrol track).** Operator opens a patrulje, clicks the track
action. A Leaflet map opens showing one coloured polyline per member — including
the scout who was moved off the team at 11:00, whose line ends there — with the
patrol's scan positions as markers along the way, and a legend naming each
member and whether they are still on the team.

**Edge cases.**

- **Never reported.** No glyph. No tooltip. No empty state in the row.
- **Reported once, long ago.** Glyph present; tooltip must make the staleness
  obvious (absolute time plus relative), and the glyph is visually muted beyond
  a staleness threshold rather than hidden.
- **Member hard-deleted.** `NATHEJK.*.spejder.*.deleted` deletes the `spejder`
  row outright, so a withdrawn scout has no name to show. Their track must still
  appear on the patrol map, labelled from `spejderstatuslog` or as "tidligere
  medlem" if no name survives.
- **Person moved between two patrols.** Their track appears on **both** patrol
  maps, clipped to the interval during which they belonged to that team.
- **Clock skew / out-of-order points.** Points are ordered by the reported
  timestamp, not arrival; a point older than the stored latest must not
  overwrite the latest-known position.
- **Duplicate delivery.** Replay on API restart must be idempotent — the same
  point twice must not double the track.
- **Implausible jumps.** A single wild coordinate should not silently draw a line
  across the country; see Open Questions on filtering.

## 6. Requirements

### Functional

- [ ] HQ consumes the `telemetry` stream and materialises reported positions
      into MySQL read models, rebuilt by replay on API start like every other
      projection.
- [ ] Two projections, deliberately split by access pattern:
      **latest position per person** (one row per person, read on every list) and
      **the point history** (append-only, read only when a track is requested).
- [ ] A presence endpoint returns, for the current year, every person who has
      ever reported a position together with their last-reported timestamp — in
      one response, keyed by person id.
- [ ] Every HQ list or card that shows a person's name renders the indicator
      from that one response: personnel (`friend`), badut (`gøgler`), bandit,
      crewmember, spejder and senior. One shared component, one shared
      composable — not per-view fetching.
- [ ] Hovering the indicator shows the last-received timestamp, formatted
      `da-DK`, absolute plus relative.
- [ ] A per-person track endpoint returns the ordered point history for one
      person, bounded by an optional time window.
- [ ] A per-patrulje track endpoint returns, for one team: the tracks of every
      person who has been a member of that team (each annotated with the member's
      name where known, and the interval of their membership), plus every scan
      registered for that team.
- [ ] Membership history is derived from `spejderstatus` /
      `spejderstatuslog`, **not** from the `spejder` table, which does not retain
      removed members.
- [ ] The patrol map view renders those tracks as one polyline per member with a
      legend, and the scans as markers, on the same Leaflet base layers as
      `/kort`.
- [ ] Everything above is live: new points make indicators appear and an open
      track view update, via the existing SSE mechanism.

### Non-Functional

- **Live-first (PRD 004).** All new reads go through `useLiveResource` with a
  correct `dependsOn`; the presence response is a type-level dependency so
  newly-reporting people appear, and a track is instance-level.
- **List cost is O(1) requests.** Adding the indicator must not add a request per
  row, nor widen ten existing handlers. A list page pays one extra small fetch,
  cached across views.
- **Volume.** Point history is the first genuinely high-cardinality table in HQ.
  The write path must be a single idempotent insert with no read-modify-write,
  and the track query must be index-covered on (year, person, time).
- **Privacy.** Positions of named people, many of them minors, are personal data.
  Access is restricted to authenticated HQ users on the same footing as member
  contact details; retention is bounded (see Open Questions) and stated in the
  schema comment. No position data is exposed on any unauthenticated endpoint.
- **Honest absence.** The indicator must distinguish "never reported" from "we
  have not asked yet": while presence is still loading, show nothing rather than
  a wrong negative.
- **Accessibility.** The indicator carries a text label for screen readers and
  the timestamp is reachable without hover (focus, and visible in the member
  detail dialog).

## 7. UX / UI Notes

**The indicator.** A single small FontAwesome location pin, inline after the
name, sized to the text and coloured from the theme's muted foreground so it
reads as metadata rather than status. Three states only: absent (never
reported), normal (reported recently), muted (stale beyond the threshold).
Tooltip: `Sidst set 14:32 · for 6 minutter siden`. Implemented once as
`components/PositionIndicator.vue` taking a person id, and dropped into
`BadutListView`, `PatruljeView`, `KlanListView`, `OrganisationView`,
`SosTeamCard`, `MemberDetailDialog` and the shelter/care lists.

Clicking the indicator is the entry point to the track — for a spejder it opens
the patrol map (all members + scans), for personnel and crew it opens that
person's own track. This keeps one affordance and makes the spejder rule
discoverable rather than hidden in a menu.

**The track view.** A dialog over the current page rather than a new route, so
the operator does not lose their place in a list they are working through:
`components/TrackMapDialog.vue`, a Leaflet map reusing `/kort`'s base layers
(Dataforsyningen topo, ortofoto, OSM), `fitBounds` on the data, one colour per
member from a fixed palette, scan markers distinguishable from track vertices,
and a legend listing each member with their membership interval and last-seen
time. A time-window control is desirable but may land in a follow-up; the
endpoint accepts the bounds from day one.

**Empty and degraded states.** A patrol with no telemetry at all shows the scan
markers alone and says so ("Ingen positioner rapporteret — viser kun scanninger").
A patrol with neither shows the map centred on the race area with an empty state.

## 8. Technical Considerations

**Reading a second stream.** `xstream`/`jetstream` (`github.com/jrgensen/stream`)
resolves the JetStream stream name from the subject's **domain** — the part
before the first `.` — so a consumer declaring
`telemetry.*.position.*.reported` subscribes to a stream named `telemetry` with
no library change and no second mux. Three caveats must be handled explicitly:

- **The stream must already exist in NATS.** `stream.New()` has its
  `CreateStream` block commented out, so a missing stream makes `OrderedConsumer`
  fail and `mux.Run` fatal at boot. Stream provisioning is an infra prerequisite,
  and the API should fail loudly and legibly if it is absent.
- **Casing.** The publish path uppercases the domain
  (`strings.ToUpper(...Domain())`), so producers emit `TELEMETRY.…`. HQ only
  consumes here, but the subject casing must be pinned in one place and matched
  against what hej-app actually publishes before any code is written.
- **`live.SignalFromSubject` hardcodes `domain = "NATHEJK"`** and returns
  `ErrNotASignal` for anything else — so a telemetry projection would silently
  emit no live signals. `go/internal/live/signal.go` must be widened to accept a
  set of known domains, keeping the existing shape rules. This is the only change
  to shared live plumbing, and it must not alter existing `NATHEJK` behaviour;
  `live.EntitiesFrom` then advertises the new token automatically.

**BFF (Go).**

- New `go/nathejk/table/position/` (or `telemetry/`) following the house layout
  (`table.go`, `consumer.go`, `query.go`, `filter.go`, `table.sql`), read-only —
  no `commands.go`, HQ never writes telemetry.
- Two tables: `position_latest(year, personId PK-part, personType, latitude,
  longitude, accuracy, uts, updatedAt)` and `position_point(year, personId, uts,
  latitude, longitude, accuracy, PRIMARY KEY (year, personId, uts))` — the PK
  giving idempotent `INSERT IGNORE` on replay, and the same index serving the
  track query. Latest is written with a guard so an older point cannot regress
  it.
- Store coordinates as `DECIMAL(9,6)`/`DECIMAL(10,7)`, not the `VARCHAR(99)` that
  `scan` uses; the track query needs ordering and bounds, and `scan`'s string
  columns are a known wart we should not copy.
- Note the standing constraint: `CREATE TABLE IF NOT EXISTS` never alters an
  existing table, so column changes after first deploy need care in dev.
- Add the new consumer to the `projections` slice in `cmd/api/main.go` — inside
  it, so `live.NotifyAll` wraps it.
- Patrol assembly lives in a handler that joins three sources: membership
  intervals from `spejderstatus`/`spejderstatuslog`, points from
  `position_point`, scans from `scan.GetAll(Filter{TeamID})`. Names come from
  `spejder` where the row still exists, and degrade gracefully where it does not.

**Frontend (Vue 3 / TS).**

- `composables/usePositionPresence.ts` — one `useLiveResource('telemetry:presence',
  …, { dependsOn: ['position'] })`, shared by every consumer, exposing
  `hasPosition(id)` and `lastSeenAt(id)`.
- `components/PositionIndicator.vue`, `components/TrackMapDialog.vue`, plus a
  `composables/useTrack.ts` wrapping the two track endpoints with instance-level
  dependencies (`patrulje:{id}` and `position:{personId}`).
- The dependency token is the **subject's** entity (`position`), not the
  projection's or the UI's — the SPA warns in the dev console if it is wrong.
- Track dialogs are read-only, so the three-line live adoption applies; no
  dirty-guard needed (unlike `KortView`).

**API endpoints.** All new, all `GET`, all requiring authentication, and **all
must carry OpenAPI annotations** in the same style as existing handlers:

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/telemetry/presence` | Every person with ≥1 position this year + last-seen timestamp |
| GET | `/api/telemetry/person/:personId/track` | One person's ordered points, optional `from`/`to` |
| GET | `/api/telemetry/patrulje/:teamId/track` | All members' tracks (with membership intervals) + that team's scans |

**Dependencies & risks.**

- **Infra:** the `telemetry` stream must exist with adequate retention before
  deploy, or the API will not boot. This is the largest operational risk.
- **Payload contract:** the hej-app's subject and JSON payload are not yet pinned
  in this repo; a mismatch means a silently empty projection.
- **Volume:** unlike every other stream HQ consumes, this one grows with wall
  time × participants. Replaying it on every API restart is the current model and
  may become the binding constraint on boot time; this must be measured against
  real traffic during the season, and mitigations (JetStream retention limits,
  a downsampled point table, or a non-replaying consumer) chosen on evidence.
- **Backwards compatibility:** additive only. Nothing existing changes except
  `live/signal.go`'s domain check, which is covered by tests today and must stay
  green.

## 9. Success Metrics

- Every HQ people list shows the indicator, and adding it costs exactly one
  extra HTTP request per session (verified in the network panel).
- A first-ever position for a person makes their indicator appear without a page
  reload, in under 2 seconds end to end.
- The patrol track endpoint returns for a full-race patrol in < 300 ms at p95.
- API boot time (replay included) stays within its current envelope; if it does
  not, the mitigation decision is recorded in this PRD.
- Qualitative: during the event, at least one SOS or dispatch case is resolved
  using the track view, and the operators say it was faster than the phone.

## 10. Rollout / Task Breakdown

Sequenced so value lands before the expensive part, and so the risky
infrastructure question is answered first.

1. Pin the contract and provision the stream (blocking, mostly not code).
2. Projection + presence endpoint + indicator — broad, cheap, immediately useful.
3. Track endpoints + patrol assembly.
4. Track map dialog.
5. Measure volume and decide retention.

Proposed tasks for `roadmap/tasks/open/`:

- [ ] Task: Pin the telemetry subject and payload contract against hej-app, and provision the `telemetry` stream in NATS (dev + stage)
- [ ] Task: Widen `live.SignalFromSubject` to accept multiple stream domains, keeping `NATHEJK` behaviour and tests green
- [ ] Task: Add `table/position` projection with `position_latest` + `position_point`, wired into the `projections` slice
- [ ] Task: `GET /api/telemetry/presence` handler with OpenAPI annotations
- [ ] Task: `usePositionPresence` composable + `PositionIndicator.vue`
- [ ] Task: Drop the indicator into every people list (badut, personnel, klan, organisation, patrulje, SOS, shelter/care, member dialog)
- [ ] Task: `GET /api/telemetry/person/:personId/track` with time-window params and OpenAPI annotations
- [ ] Task: Derive patrulje membership intervals (current + former) from `spejderstatus`/`spejderstatuslog`
- [ ] Task: `GET /api/telemetry/patrulje/:teamId/track` combining member tracks + scans, with OpenAPI annotations
- [ ] Task: `TrackMapDialog.vue` — Leaflet tracks + scan markers + member legend
- [ ] Task: Measure telemetry volume and replay cost; decide retention/downsampling and record the decision in PRD 011

## 11. Open Questions

- **Subject shape.** Is it `TELEMETRY.{year}.position.{personId}.reported`, and
  is the entity token `position` — or something else? Everything in the live
  layer keys off this.
- **Person identity.** Positions arrive keyed by what? A `UserID` (personnel,
  crew), a `MemberID` (spejder, senior), a phone number, or a hej-app account id
  that maps to none of the above? If the last, HQ needs a resolution step and
  this PRD grows an identity-mapping section.
- **Payload fields.** Beyond lat/lng/timestamp: accuracy, altitude, speed,
  heading, battery, foreground/background? Are points batched (many per message)
  or one per message?
- **Reporting cadence.** Seconds or minutes? This determines whether
  `position_point` holds thousands or millions of rows per event and therefore
  whether replay-on-boot survives.
- **Retention.** How long do we keep points — the event, the season, forever?
  Personal-data minimisation argues for the shortest useful window, and this is a
  decision for knj, not the implementation.
- **Staleness threshold.** After how long is the indicator muted rather than
  normal? 15 minutes? An hour? Should it differ for a racing patrol and a gøgler?
- **`crewmate`.** The request mentions crewmates; in this repo the organisation
  crew is `crewmember` (`shared-go/tables/crewmember`, token `crewmember`) and
  there is no `crewmate` entity. Confirming these are the same people is enough
  to close this.
- **Seniors and klan.** The spejder rule (all members + scans, per patrol) is
  specified. Should a klan behave the same way with its seniors, or is a klan
  member's own track sufficient?
- **Implausible points.** Do we filter server-side on accuracy or jump distance,
  render everything faithfully, or render everything and mark suspect segments?
  Filtering hides data; not filtering draws lies.
