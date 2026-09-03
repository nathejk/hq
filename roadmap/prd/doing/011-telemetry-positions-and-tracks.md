# PRD 011 — Telemetry: who has reported a position, and where they went

**Status:** doing
**Author:** agent session (with knj)
**Created:** 2026-09-03
**Last updated:** 2026-09-03 (all code tasks complete; 139/152/153 outstanding)
**Approved:** 2026-09-03
**Shipped:**
**Target users:** organizer (HQ operators, løbsledelse, SOS/dispatch), and — indirectly — participants whose hej-app reports the positions

<!--
Status must match the folder this file is in: draft/, doing/ or done/.
Leave Approved blank until the PRD moves to doing/, and Shipped blank until it
moves to done/. See roadmap/prd/README.md for the lifecycle.
-->

---

## 1. Summary

The hej-app now posts positions to a new JetStream stream, `TELEMETRY`. This PRD
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
  `TELEMETRY` projection must not require a new mux, a new SSE channel, or a
  parallel live mechanism.

## 4. Non-Goals

- **Producing positions.** The hej-app owns collection, consent, batching and
  upload. HQ only consumes, and — since all inter-service communication goes
  through the stream — has no API access to hej-app and must not gain any.
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
- **Changing the hej-app.** If the subject shape or payload needs adjusting, that
  is work in the app's repo, tracked separately. As of 2026-09-03 nothing needs
  adjusting — see §4a.

## 4a. The contract (confirmed 2026-09-03)

Producer `hej-api` emits on the `TELEMETRY` stream, subject:

```
TELEMETRY.2026.track.f30793d2-5393-4d90-bbfa-cf224bbc131b.reported
```

— that is, `TELEMETRY.{year}.track.{personId}.reported`. The message carries the
standard envelope (`eventId`, `correlationId`, `causationId`, `version`, `time`,
`meta`) and a body of the shape:

```json
{
  "personId": "f30793d2-5393-4d90-bbfa-cf224bbc131b",
  "userType": "gøgler",
  "year": "2026",
  "points": [
    { "ts": 1788437919856, "lat": 55.70915145018305, "lng": 12.600336777419688, "accuracy": 18.739823543347217 }
  ]
}
```

The facts this pins down, each of which removed an open question:

- **`userType` is always present** on every track message. It is the same
  vocabulary as elsewhere (`gøgler`, `friend`, `bandit`, `crewmember`,
  `spejder`, `senior`, …), so HQ stores it verbatim as `personType` and never
  infers the id space. This also means the indicator can be scoped per kind for
  free if a staleness rule ever differs by kind.
- **Points are batched** — `points` is an array, many per message — so the
  consumer loops and inserts each, and one message can advance both the latest
  row and many history rows.
- **`ts` is Unix epoch milliseconds**, not seconds; `scan.uts` is seconds. The
  schema must store milliseconds (or a normalised precision chosen once) and the
  two must be reconciled when tracks and scans share a time axis on the map.
- **`lat`/`lng`/`accuracy` are JSON numbers** (floats), confirming `DECIMAL`
  storage over `scan`'s `VARCHAR(99)`. `accuracy` is in metres and is present on
  every point, so it is available for the implausible-point question.
- **`year` is in the body** *and* in the subject, matching every other
  projection's year scoping.
- **A point's identity is `(personId, ts)`**, and `ts` is an integer for exactly
  that reason — an RFC 3339 string could be re-serialised into a
  different-but-equal form and a consumer would see two points where there is
  one. HQ's primary key must therefore be `(personId, ts)`.
- **Junk is already filtered at the producer.** `track.Clean` drops NaN/±Inf,
  `ts == 0`, timestamps before 2020 or more than 24 h in the future,
  out-of-range coordinates, exactly `(0, 0)` (Null Island — a failed fix reported
  as a success), and accuracy outside `0 … 100 km`. Crucially it **drops the bad
  points and keeps the batch**, because the client retries until accepted and a
  rejected batch would poison every later point behind it. Two consequences for
  HQ: coordinates arriving here are already sane, and **gaps are normal** — a
  missing minute is not a bug to chase.
- **A poor fix is deliberately kept.** A multi-kilometre cell-tower fix is real
  and is passed through; only >100 km is rejected as "not a fix". So HQ receives
  low-confidence points on purpose and must decide how to *render* them — not
  whether to store them.
- **Batches are bounded at 2,000 points.** Note the producer sized that against a
  *12-hour* race (~1,440 points at 30 s sampling), while Nathejk runs closer to
  **30 hours**, whose theoretical ceiling is ~3,600 points. In practice a phone
  that records unbroken for 30 hours is unlikely, so the cap will rarely bind —
  but if an unchunked backlog ever does exceed it, the server returns
  `ErrBatchTooLarge`, the client retries the same oversized batch forever, and
  that is exactly the poison pill `Clean`'s drop-don't-reject design was built to
  avoid. Low likelihood, unbounded consequence, cheap to check: **worth raising
  with hej-app**. HQ can neither fix nor directly detect it — the symptom here is
  a person whose track never arrives after a long offline stretch.
- **The client retries until the server accepts**, so the same batch can be
  delivered more than once. Idempotent insert is not optional.
- **The entity token is `track`.** Not `position`, not `telemetry` — this is the
  live dependency token the whole frontend keys off (see §8).
- **The stream is `TELEMETRY`, uppercase**, since the library derives the stream
  name from the subject's domain verbatim.
- **The subject is per person because that is the erasure unit.** `nats stream
  purge --subject` can remove one individual's track and nothing else. The stream
  is **retained indefinitely** — which has a consequence for HQ that the producer
  cannot solve alone; see §6 and §8 on erasure.
- **`userType` is stamped at publish time, not looked up at read time**, because
  roles change while the stream is permanent: a spejder becomes a bandit, crew
  get reclassified. A consumer joining against today's directory would silently
  reinterpret last year's history. HQ therefore **stores the `userType` from the
  message and must never derive it by lookup** — and the same `personId` may
  legitimately carry different `userType` values over time.
- **HQ is a pure consumer.** There is no API between the services; the stream is
  the entire integration surface.

> **Note on cross-repo references.** The producer's comments cite "PRD 002 §11.1"
> and tasks 081–086; those are hej-app's numbers. HQ's PRD 002 is order-based
> payments. Do not follow those references into this repo.

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
- **Unknown personId.** A position may arrive for a `personId` HQ has no row
      for (a crew member registered elsewhere, a stale install from last year).
      Store it faithfully; it simply has no name to sit next to, so no indicator
      appears anywhere. Do not drop the point and do not error.
- **Member hard-deleted.** `NATHEJK.*.spejder.*.deleted` deletes the `spejder`
  row outright, so a withdrawn scout has no name to show. Their track must still
  appear on the patrol map, labelled from `spejderstatuslog` or as "tidligere
  medlem" if no name survives.
- **Person moved between two patrols.** Their track appears on **both** patrol
  maps, clipped to the interval during which they belonged to that team.
- **Clock skew / out-of-order points.** Points are ordered by the reported
  timestamp, not arrival; a point older than the stored latest must not
  overwrite the latest-known position.
- **Duplicate delivery.** The client retries a batch until accepted, and replay
  on API restart re-reads everything — so the same point will arrive twice in
  normal operation. The `(personId, ts)` key must absorb it.
- **A track is a set of segments, not a line.** Continuous recording for 30 hours
  is not realistic — phones are locked, apps are backgrounded and killed,
  batteries die, and nobody is watching their screen while walking at 3 a.m. **Gaps
  are the normal shape of this data, not an anomaly**, and they are frequently
  hours long. This is a modelling decision, not a rendering preference: HQ must
  represent a track as ordered *segments* split on a gap threshold, so that no
  consumer can accidentally draw a straight line across three hours of silence and
  present it as a walked route. A lie drawn confidently on a map is worse than a
  visible gap.
- **Volume is a ceiling, not an expectation.** ~3,600 points per person is the
  theoretical maximum at 30 s sampling; real coverage will be a fraction of it.
  Design the schema and the reduction path against the ceiling — they must not
  fall over on the one participant who did keep their phone awake — but do not
  build for a load that will not arrive.
- **Low-accuracy points.** The producer keeps cell-tower fixes of several
  kilometres on purpose, so a track can legitimately contain a point that is
  hundreds of metres off-route. HQ renders faithfully and may mark low-confidence
  points from the `accuracy` field; it must not drop them, since they are
  sometimes the only evidence of where someone was.

## 6. Requirements

### Functional

- [ ] HQ consumes the `TELEMETRY` stream and materialises reported positions
      into MySQL read models, rebuilt by replay on API start like every other
      projection.
- [ ] Two projections, deliberately split by access pattern:
      **latest position per person** (one row per person, read on every list) and
      **the point history** (append-only, read only when a track is requested).
- [ ] A presence endpoint returns, for the current year, every `personId` that
      has ever reported a position together with its last-reported timestamp — in
      one response, keyed by the raw `personId`, with no name resolution and no
      join against any people table.
- [ ] Every HQ list or card that shows a person's name renders the indicator
      from that one response: personnel (`friend`), badut (`gøgler`), bandit,
      crewmember, spejder and senior. Each row already carries the id the
      indicator needs (`memberId` or `userId`), so no endpoint gains a field and
      no lookup table is introduced. One shared component, one shared
      composable — not per-view fetching.
- [ ] Hovering the indicator shows the last-received timestamp, formatted
      `da-DK`, absolute plus relative.
- [ ] A per-person track endpoint returns the ordered point history for one
      person, bounded by an optional time window and reduced to a requested
      resolution.
- [ ] A per-patrulje track endpoint returns, for one team: the tracks of every
      person who has been a member of that team (each annotated with the member's
      name where known, and the interval of their membership), plus every scan
      registered for that team — reduced by default, never raw.
- [ ] Tracks are returned as **segments split on a gap threshold**, so a client
      cannot draw a continuous line through a period with no data. Each segment
      carries its own start and end time.
- [ ] Each track states its **coverage** over the requested window — how much of
      the interval actually has data — so an operator can tell a thin track from a
      well-recorded one before drawing conclusions from it.
- [ ] Reduction is honest: a reduced track states the resolution it was reduced
      to, so the UI can say so and can offer full fidelity for a narrow window.
      Scans are **never** reduced — they are few, exact, and the anchor points an
      operator reasons from.
- [ ] Membership history is derived from `spejderstatus` /
      `spejderstatuslog`, **not** from the `spejder` table, which does not retain
      removed members.
- [ ] `userType` is stored as received per message and never derived by lookup,
      so history keeps the role the person actually held at the time.
- [ ] **Erasure propagates.** Purging a person's subject from the stream must be
      matched by removing their rows from HQ's tables. HQ needs a deliberate
      mechanism for this — at minimum a documented, runnable delete keyed by
      `personId` — because an indefinitely-retained stream plus a
      replay-built read model means HQ would otherwise be the place erased data
      survives.
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
  The **ceiling** at ~30 s sampling over a ~30-hour race is ~3,600 points per
  person, but continuous recording for 30 hours is unrealistic — expect gaps, and
  expect real coverage to be a fraction of that. So: a thousand reporting
  participants has a ceiling around 3.6 M rows per event and will in practice sit
  well below it. The stream is **retained indefinitely**, so whatever the real
  figure is, it accumulates yearly unless HQ bounds it. Design against the ceiling
  for correctness — single idempotent insert, no read-modify-write, track query
  index-covered on (person, time) — without provisioning for it.
- **Honesty about coverage outranks completeness.** Because the data is sparse and
  irregularly sparse, every surface must distinguish *"was not here"* from *"we
  have no data"*. That applies to the indicator (muted means unknown, not
  missing), to the map (segments, not interpolated lines), and to the legend
  (coverage stated). An operator making a decision during an incident must not be
  able to mistake absence of data for evidence.
- **Map payload size is a real constraint, though softened by sparsity.** The
  ceiling for a six-member patrol over a full race is ~21,600 points; sparse
  recording will typically make it far less, but the endpoint cannot rely on that.
  Shipping raw points to Leaflet at the ceiling is megabytes of JSON and a janky
  map, and it is more detail than any screen can show — a 30-hour route at display
  zoom cannot resolve 30-second steps. The track endpoints must therefore support
  **server-side reduction** from the start: a `maxPoints` parameter or
  Douglas–Peucker simplification, applied **within** segments so reduction can
  never bridge a gap. Designing this in is cheap; retrofitting it after the UI
  assumes raw points is not.
- **Erasure.** The producer's per-person subject exists so one individual can be
  purged from the stream. That guarantee is void if HQ keeps a copy: a purge
  removes the source, but HQ's MySQL rows persist until something deletes them,
  and a replay-built read model will not notice their absence. HQ must be able to
  delete one person's telemetry on request, and that path must be as easy to run
  as the purge it accompanies.
- **Privacy.** Positions of named people, many of them minors, are personal data.
  Access is restricted to authenticated HQ users on the same footing as member
  contact details; retention is bounded (see Open Questions) and stated in the
  schema comment. No position data is exposed on any unauthenticated endpoint.

  > **Qualified during task 142.** "Authenticated" overstates what this repo does.
  > `app.authenticate` attributes every request to an anonymous user and enforces
  > nothing — authentication lives in an external service — so telemetry endpoints
  > are exactly as protected as `/api/patrulje/:id` already is, and no more. That
  > is the right *relative* position for this data, and it is what the PRD's
  > "same footing as member contact details" actually means. But if the intent was
  > that position history be held to a **higher** bar than the rest of the read
  > model, nothing in hq implements that today and it would need its own task.
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

**The muted state means "we do not know", not "something is wrong".** Since gaps
of hours are normal, a stale indicator is weak evidence of nothing at all — a
phone in a pocket on battery-saver looks identical to a phone at the bottom of a
lake. The wording must not imply alarm, and the indicator must never be presented
as a safety signal; it answers "can I expect location data from this person?" and
no more.

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
time.

A **time-window control is not optional** at 30 hours of route: the whole race on
one screen is a tangle, and the operator's real question is almost always "where
were they between 22:00 and 02:00?". The window doubles as the fidelity control —
zoom in on time, get more detail — which is why the endpoint takes `from`/`to`
and `maxPoints` from day one and the view opens on a sensible default window
rather than everything.

**Gaps are drawn as gaps.** Each member's track is rendered as one polyline **per
segment**, never a single line through the whole window, because most tracks will
be a handful of recorded stretches separated by hours of nothing. Segment ends get
a small terminator so it is visible that the data stops there rather than the
person stopping there, and the legend shows each member's coverage ("3 t 40 min
data af 12 t") next to their last-seen time. Where a gap is bridged for legibility
it must be visually distinct — dashed and dimmed — and never the default.

This is the difference between a map an operator can act on and one that misleads
them: a straight dashed line from a forest at 23:00 to a road at 02:00 says "we
don't know", while the same line drawn solid says "they walked this", and only one
of those is true.

**Empty and degraded states.** A patrol with no telemetry at all shows the scan
markers alone and says so ("Ingen positioner rapporteret — viser kun scanninger").
A patrol with neither shows the map centred on the race area with an empty state.

## 8. Technical Considerations

**Reading a second stream.** `xstream`/`jetstream` (`github.com/jrgensen/stream`)
resolves the JetStream stream name from the subject's **domain** — the part
before the first `.` — so a consumer declaring
`TELEMETRY.*.track.*.reported` subscribes to a stream named `TELEMETRY` with
no library change and no second mux. Three caveats must be handled explicitly:

- **The stream must already exist in NATS**, named `TELEMETRY` (uppercase, as the
  subject's domain). `stream.New()` has its `CreateStream` block commented out,
  so a missing or differently-cased stream makes `OrderedConsumer` fail and
  `mux.Run` fatal at boot. Stream provisioning is an infra prerequisite, and the
  API should fail loudly and legibly if it is absent.
- **The entity token is `track`, not `position`.** From
  `TELEMETRY.2026.track.{personId}.reported`, `live.SignalFromSubject` yields
  `Entity: "track"`, `ID: personId`, `Year: "2026"`, `Event: "reported"`. Every
  frontend `dependsOn` uses `track` / `track:{personId}`. This is exactly the
  trap the house rules call out (scans are `qr`, not `scan`), so it is worth
  stating twice: the UI concept and the table names may say position, the
  dependency token says `track`.
- **`live.SignalFromSubject` hardcodes `domain = "NATHEJK"`** and returns
  `ErrNotASignal` for anything else — so a telemetry projection would silently
  emit no live signals. `go/internal/live/signal.go` must be widened to accept a
  set of known domains (`NATHEJK`, `TELEMETRY`), keeping the existing shape rules.
  This is the only change to shared live plumbing, and it must not alter existing
  `NATHEJK` behaviour; `live.EntitiesFrom` then advertises `track` automatically.

**No service-to-service HTTP.** All communication between hej-api and HQ goes
through the stream — HQ has no API access to hej-app and must never acquire any.
Everything in this PRD is therefore derivable from the messages alone: there is
no fallback of "ask the app" for a missing name, a gap in a track, or a
backfill. If a fact is not on the stream, HQ does not have it, and the UI must
degrade rather than fetch.

**Person identity.** A message is keyed by a `personId` which is either a
**`memberID`** — a spejder or senior, PK `(year, memberId)` in `spejder` /
`senior` — or a **`crewmemberID`**, which is the `userId` of `crewmember`. Note
that `personnel` (gøgler, friend, bandit) is keyed by `userId` in the *same*
space, so one id space covers all of crew, gøgler, friend and bandit.

This is the cheapest possible outcome for HQ: both spaces are opaque
`VARCHAR(99)` ids that do not collide, and every people-list row already carries
its own id. So **no identity-mapping step, no resolution endpoint, and no name
join on the read path** — the presence response is a bare id-to-timestamp map,
and the frontend simply asks whether its own row's id is present. The payload's
`userType` (§4a) is stored verbatim as `personType` — the full vocabulary
(`gøgler`, `friend`, `bandit`, `crewmember`, `spejder`, `senior`), not a
two-value `member`/`crewmember` — for legibility and query scoping; correctness
does not depend on it, but it removes any need to infer the id space.

The consequence for the two track endpoints is that `:personId` is accepted from
either space without qualification, while the *patrulje* endpoint works purely in
`memberID` space, since only scouts have team membership.

**BFF (Go).**

- New `go/nathejk/table/track/` following the house layout (`table.go`,
  `consumer.go`, `query.go`, `filter.go`, `table.sql`), read-only — no
  `commands.go`, HQ never writes telemetry. `Consumes()` returns the single
  subject `TELEMETRY.*.track.*.reported`.
- Two tables: `track_latest(personId PK, personType, year, latitude, longitude,
  accuracy, ts, updatedAt)` and `track_point(personId, ts, personType, year,
  latitude, longitude, accuracy, PRIMARY KEY (personId, ts))`. **The key is
  `(personId, ts)` — not `(year, personId, ts)` — to match the producer's
  definition of a point's identity exactly**; `year` is a column, and the track
  query filters on `(personId, ts)` which the PK already covers. `INSERT IGNORE`
  then makes both client retries and boot replay idempotent for free. Latest is
  written with a guard so an older point cannot regress it.
- `personType` is stored on **both** tables from the message, never joined: roles
  change and the stream is permanent (§4a), so `track_point` records the role held
  when that batch was reported while `track_latest` carries the most recent one.
- `ts` is **epoch milliseconds** as sent (§4a); store it as `BIGINT`, not the
  `INT` seconds that `scan.uts` uses, and convert at the read boundary where the
  patrol map needs scans (`uts` seconds) and points (`ts` ms) on one axis.
- The consumer **iterates `body.points`**, inserting each; a single message
  therefore produces many history rows and at most one latest-row advance.
- Store coordinates as `DECIMAL(9,6)`/`DECIMAL(10,7)`, not the `VARCHAR(99)` that
  `scan` uses; the track query needs ordering and bounds, and `scan`'s string
  columns are a known wart we should not copy. `accuracy` is a metre float and is
  stored as sent — it is what lets the UI mark low-confidence points.
- **No re-validation of points.** `track.Clean` at the producer already rejects
  NaN, Null Island, impossible timestamps and >100 km accuracy. HQ duplicating
  that would drift from it and silently discard data the producer chose to keep;
  HQ stores what it is given.
- Note the standing constraint: `CREATE TABLE IF NOT EXISTS` never alters an
  existing table, so column changes after first deploy need care in dev.
- Add the new consumer to the `projections` slice in `cmd/api/main.go` — inside
  it, so `live.NotifyAll` wraps it.
- Track responses are shaped as `{ personId, personType, name?, membership?,
  coverage, segments: [{ from, to, points: […] }] }`. **Segments rather than a flat
  point array is the load-bearing choice**: it makes the sparse reality of the data
  impossible for a client to misrender, and it costs nothing to produce — the
  consumer walks points ordered by `ts` and starts a new segment when the delta
  exceeds the threshold. Reduction is applied within a segment, never across one.
- Patrol assembly lives in a handler that joins three sources: membership
  intervals from `spejderstatus`/`spejderstatuslog`, points from
  `track_point`, scans from `scan.GetAll(Filter{TeamID})`. Names come from
  `spejder` where the row still exists, and degrade gracefully where it does not.

**Frontend (Vue 3 / TS).**

- `composables/usePositionPresence.ts` — one `useLiveResource('telemetry:presence',
  …, { dependsOn: ['track'] })`, shared by every consumer, exposing
  `hasPosition(id)` and `lastSeenAt(id)`.
- `components/PositionIndicator.vue`, `components/TrackMapDialog.vue`, plus a
  `composables/useTrack.ts` wrapping the two track endpoints with instance-level
  dependencies (`patrulje:{id}` plus `track` for the patrol view, and
  `track:{personId}` for one person).
- The dependency token is the **subject's** entity (`track`), not the
  projection's or the UI's — the SPA warns in the dev console if it is wrong.
- The patrol view must depend on the **type** token `track` as well as the
  instance, because a point from a member it has never seen before carries an
  unknown `personId`; an instance-only dependency would miss exactly the newly
  joined member the operator is looking for.
- Track dialogs are read-only, so the three-line live adoption applies; no
  dirty-guard needed (unlike `KortView`).

**API endpoints.** All new, all `GET`, all requiring authentication, and **all
must carry OpenAPI annotations** in the same style as existing handlers:

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/telemetry/presence` | Every `personId` with at least one position this year + last-seen timestamp |
| GET | `/api/telemetry/person/:personId/track` | One person's ordered points, optional `from`/`to` and `maxPoints` |
| GET | `/api/telemetry/patrulje/:teamId/track` | All members' tracks (with membership intervals, reduced) + that team's scans (exact) |

**Dependencies & risks.**

- **Infra:** the `TELEMETRY` stream must exist, correctly cased, before deploy or
  the API will not boot. This is the largest operational risk and the only
  remaining prerequisite.
- **Volume × indefinite retention is the real risk.** Unlike every other stream HQ
  consumes, this one grows with wall time × participants and is never trimmed —
  ~3.6 M points per event, accumulating yearly. Replaying all of it on every API
  restart is the current model and is the plausible breaking point for boot time.
  It must be measured against real traffic during the season, with mitigations
  chosen on evidence: a year-scoped filter subject (`TELEMETRY.2026.>`) so HQ
  replays only the current event, a downsampled point table, or a consumer that
  does not replay. **A year-scoped subject is the cheapest and should be the
  default** — the consumer already declares its subjects, so scoping is a
  one-line change rather than an architecture.
- **Contract drift:** the subject and payload are pinned here (§4a and above) but
  live in another repo, and there is no API to ask. A silent shape change means a
  silently empty projection, so the consumer should log unhandled subjects the way
  `crewmember` does rather than fail closed.
- **Volume:** see above — the mitigation decision belongs in this PRD once
  measured.
- **Erasure:** HQ becomes a second copy of personal location data that the
  producer's per-person purge cannot reach. This is a compliance risk, not just a
  cleanup chore, and is the one requirement here that no amount of frontend work
  substitutes for.
- **Backwards compatibility:** additive only. Nothing existing changes except
  `live/signal.go`'s domain check, which is covered by tests today and must stay
  green.

## 9. Success Metrics

- Every HQ people list shows the indicator, and adding it costs exactly one
  extra HTTP request per session (verified in the network panel).
- A first-ever position for a person makes their indicator appear without a page
  reload, in under 2 seconds end to end.
- The patrol track endpoint returns for a full-race patrol in < 300 ms at p95,
  with a reduced payload under ~500 KB — measured against the worst realistic
  case, a well-recorded 30-hour six-member patrol, not a sparse or short track.
- No map in HQ ever draws a solid line across a gap. Verifiable by inspection of
  the response shape: if the API returns segments, the UI cannot.
- API boot time (replay included) stays within its current envelope; if it does
  not, the mitigation decision is recorded in this PRD.
- Qualitative: during the event, at least one SOS or dispatch case is resolved
  using the track view, and the operators say it was faster than the phone.

## 10. Rollout / Task Breakdown

Sequenced so value lands before the expensive part, and so the risky
infrastructure question is answered first.

1. Provision the stream (blocking, not code).
2. Projection + presence endpoint + indicator — broad, cheap, immediately useful.
3. Track endpoints + patrol assembly.
4. Track map dialog, with the time-window control included — at 30 hours of route
   it is part of the feature, not a refinement.
5. Erasure path and measured volume decision — before the event, not after.

Tasks created in `roadmap/tasks/open/` on approval:

- [x] 139 — Provision the `TELEMETRY` stream in NATS (dev + stage) — **dev confirmed, stage outstanding**
- [x] 140 — Widen `live.SignalFromSubject` to accept `TELEMETRY` alongside `NATHEJK`
- [x] 141 — Add `table/track` projection (`track_latest` + `track_point`), wired into the `projections` slice
- [x] 142 — `GET /api/telemetry/presence`
- [x] 143 — `usePositionPresence` composable + `PositionIndicator.vue`
- [x] 144 — Drop the indicator into every people list
- [x] 145 — Segment tracks on a gap threshold and report per-track coverage
- [x] 146 — Server-side track reduction, applied within segments
- [x] 147 — `GET /api/telemetry/person/:personId/track`
- [x] 148 — Derive patrulje membership intervals (current + former)
- [x] 149 — `GET /api/telemetry/patrulje/:teamId/track` — member tracks + scans
- [x] 150 — `TrackMapDialog.vue`
- [x] 151 — Per-person telemetry erasure (compliance) — `roadmap/api/telemetry-erasure.md`
- [ ] 152 — Raise the batch-cap sizing with hej-app
- [ ] 153 — Decide hq's telemetry scope and measure replay cost

### Found along the way

**A pre-existing bug in `scan`'s querier, fixed under task 149.** `GetAll` scanned `uts`
into a variable declared outside the row loop and never assigned it to the row, so every
scan it returned carried `uts=0` while the table held the real value. It had been silently
wrong for `GET /api/patrulje/:id/scans` too. It surfaced here because this PRD puts scans
and tracks on one time axis, where every marker landed in 1970 rather than merely being an
unused field — a good example of a new consumer exposing an old defect. `/api/patrulje/:id/scans`
now returns real timestamps, which is a visible behaviour change to an existing endpoint.

**Membership is year-scoped, `scan.GetAll` is not.** Requesting a previous year's team
under the current year slug returns its scans with zero members. Harmless today — the SPA
only navigates to current-year teams — but it is a seam somebody will trip over.

## 11. Open Questions

- ~~**Subject shape.**~~ **Answered (2026-09-03):**
  `TELEMETRY.{year}.track.{personId}.reported`. Entity token `track`, stream
  `TELEMETRY`. Fully closed — nothing in this PRD is now blocked on another repo.
- ~~**Person identity.**~~ **Answered (2026-09-03):** `personId` is either a
  `memberID` or a `crewmemberID`, and every message now carries a `userType`
  stating the kind explicitly (see §4a). HQ stores it and never infers. Fully
  closed.
- ~~**Payload fields.**~~ **Answered (2026-09-03):** `personId`, `userType`,
  `year`, and a batched `points[]` of `{ ts (ms), lat, lng, accuracy (m) }` — see
  §4a. No altitude/speed/heading/battery today; if any are added later the
  point schema can gain nullable columns (subject to the `CREATE TABLE IF NOT
  EXISTS` caveat).
- ~~**Reporting cadence.**~~ **Answered (2026-09-03):** ~30 s sampling over a
  ~30-hour race — **~3,600 points per person per event**, and ~21,600 for a
  six-member patrol. This is the volume input §8 sizes against, and it is what
  makes server-side reduction a requirement rather than an optimisation.
- ~~**Retention (stream).**~~ **Answered (2026-09-03):** the stream is retained
  **indefinitely**, and per-person subjects exist so an individual can be purged.
  What remains open is HQ's own copy, which is a different and sharper question:
  **does HQ keep every year's points in MySQL, or only the current event?** A
  year-scoped consumer subject bounds both the table and the boot-time replay in
  one move, and is my recommendation. Either way, HQ needs the per-person delete
  described in §6 — the stream purge does not reach it.
- **Staleness threshold.** After how long is the indicator muted rather than
  normal? Harder than it looks now that gaps are expected: at 30 s sampling a
  short silence is meaningful, but hour-long gaps are routine, so a tight
  threshold would leave most indicators muted most of the time and the state would
  stop carrying information. 15 minutes? An hour? Should it differ for a racing
  patrol and a gøgler? **Implemented as 30 min** (`STALE_AFTER_MS`, task 143) — a
  default in one place, not a decision.
- **Gap threshold.** What delta between consecutive points starts a new segment?
  This is the one number the whole track rendering hangs on. Too small and a
  normal track shatters into confetti; too large and we bridge a gap we should
  have shown. **Implemented as 5 minutes** (`GapThresholdMs`, task 145), i.e. ten
  samples, still to be checked against the real distribution of deltas.
- ~~**`crewmate`.**~~ **Answered (2026-09-03):** it is `crewmember`
  (`shared-go/tables/crewmember`). Residual: a gøgler/friend/bandit lives in
  `personnel`, not `crewmember`, though both are keyed by `userId` — confirm the
  producer treats those ids as one space, since the indicator relies on it.
- **Seniors and klan.** The spejder rule (all members + scans, per patrol) is
  specified. Should a klan behave the same way with its seniors, or is a klan
  member's own track sufficient?
- ~~**Implausible points.**~~ **Answered (2026-09-03):** the producer already
  drops what is not a position at all (NaN, Null Island, impossible clocks,
  >100 km accuracy) and deliberately keeps poor-but-real fixes. HQ therefore does
  **not** filter: it stores everything and renders faithfully, with `accuracy`
  available to mark low-confidence points. Residual UI question only: do we mark
  them, and how — thinner line, hollow vertex, accuracy circle on click?
- ~~**Gap rendering.**~~ **Answered (2026-09-03):** gaps are the normal shape of
  the data, so tracks are modelled and returned as **segments** and drawn as one
  polyline per segment; a bridged gap, if shown at all, is dashed and dimmed. The
  remaining variable is the threshold — see above.
