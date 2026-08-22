# PRD 007 — Hønsegården: the shelter interface

**Status:** doing
**Author:** agent session (with the Hønsegården crew's request)
**Created:** 2026-08-23
**Last updated:** 2026-08-23
**Approved:** 2026-08-23
**Shipped:**
**Target users:** organizer (Hønsegården crew), organizer (nødtelefon operator, as a reader)

<!--
Status must match the folder this file is in: draft/, doing/ or done/.
Leave Approved blank until the PRD moves to doing/, and Shipped blank until it
moves to done/. See roadmap/prd/README.md for the lifecycle.
-->

---

## 1. Summary

A screen for the crew running the scout shelter at HQ — internally *Hønsegården* —
listing every scout who started the race and is no longer on the route: who is in the
shelter and where they have been bedded down, who is in a car on the way, and who is
sitting by the trail waiting for one. It is also the interface that finally *publishes*
`spejder.*.shelter.accepted` and `spejder.*.handover.completed`, the two acceptances
PRD 006 defined, wired up and deliberately left without a producer.

## 2. Problem & Motivation

**What problem does this solve?** The shelter crew's job is custody of children: they
receive scouts off the cars, put them to bed, and hand them back to a guardian or to
their own patrol at the finish. Today they have no screen at all. The information they
need is spread across the nødtelefon's case cards (one case at a time, and only for
members somebody opened a case about) and a paper list at the door. Two concrete
failures follow from that:

- **Nobody knows who is coming.** A car arrives with three scouts the shelter has never
  heard of. The crew cannot prepare beds, and cannot tell a caller "she is on her way,
  ETA twenty minutes" because they do not know she left the trail.
- **Custody is recorded nowhere.** `MemberStatusSheltered` exists, `ShelterAccepted`
  exists, and nothing publishes it — so a scout handed over at the door stays `transit`
  in the read model, and the in-our-care count (PRD 006 §, `/api/members/care`) says a
  car is still holding somebody who has been asleep for two hours. The number that must
  reach zero before the organisers go home is therefore wrong in exactly the direction
  that hides a missing child.
- **Where they physically are is not tracked at all.** "Which tent is she in?" is asked
  at 3am by a parent standing at the door, and answered today by walking the site.

**Why now?** PRD 006 shipped the lifecycle, the projection and the nødtelefon's half of
the write surface. `spejderstatus`'s `Consumes()` already subscribes to the shelter
events, with a comment saying they "belong to the car and shelter interfaces" and that
subscribing now is what makes the projection ready for them. This is that interface. It
is also the cheaper half of the two — it needs no mobile-first driver UX — and it closes
the custody chain from the shelter end, which is where the count actually settles.

**Evidence.** Requested directly by the Hønsegården crew. `PickupAccepted`,
`ShelterAccepted` and `HandoverCompleted` in
`go/nathejk/table/spejderstatus/messages.go` document the seam and name this screen;
`roadmap/prd/done/006-member-lifecycle-team-strength.md` scopes it out explicitly.

## 3. Goals

- The shelter crew can see, without asking anybody, every scout who has left the route:
  in the shelter, in a car, or waiting for one.
- The physical whereabouts of every scout in the shelter's care is recorded and visible
  on screen, so a parent at the door is answered from the screen rather than by a walk
  round the site.
- Custody changes are recorded by the party taking custody, at the moment it happens:
  the shelter records receiving a scout and records handing them on.
- The in-our-care count becomes trustworthy, because `sheltered` starts being reached by
  an acceptance rather than by a manual correction.
- The screen is usable by a tired volunteer on a laptop at 3am: it updates itself, and
  it never loses what somebody was typing.

## 4. Non-Goals

- **The car/driver interface.** Publishing `pickup.accepted` remains out of scope; the
  shelter screen *reads* transit but cannot create it. Until that interface exists,
  `transit` is reached by the nødtelefon's manual override, as today.
- **Klaner and gøglere.** The lifecycle is scouts (`spejder`) only. Seniors are not
  handled through the nødtelefon and have no status rows.
- **Bed inventory / capacity management.** We record where somebody was put, not what
  space is free. No bed map, no occupancy limits, no allocation logic.
- **Replacing the nødtelefon case flow.** Cases stay where they are; this screen links
  to them and does not manage them.
- **Notifying guardians.** No mail or SMS is sent from this screen. Phone numbers are
  displayed so a human can dial them.
- **Marking anybody `finished`.** Impossible by construction (`CanFinish()`), and this
  screen must not appear to offer it.

## 5. User Stories & Scenarios

- As a **shelter crew member**, I want one list of every scout who is no longer racing,
  so I know who is here, who is coming and who is still out there.
- As a **shelter crew member**, I want to record that a scout has been handed to me, so
  the organisation knows the car no longer holds them and I am now answerable for them.
- As a **shelter crew member**, I want to note where I put a scout, so anybody on the
  crew — including whoever relieves me — can find them.
- As a **shelter crew member**, I want to record that a guardian has collected a scout,
  or that their patrol reached the finish and took them back, so we stop counting them.
- As a **nødtelefon operator**, I want to see that the scout I sent in a car has arrived,
  so I can close the case.

**Happy path.** A car radios in with two scouts. The crew opens Hønsegården and sees them
at the top of the screen under *I bil — på vej hertil*, with the patrol they started with,
how long they have been in the car, and the case number. The car arrives; the crew selects
both rows, presses **Modtaget**, and types `Telt 4` — offered as a suggestion, because two
other scouts are already there — and both move to *I Hønsegården*.
`spejder.*.shelter.accepted` is published for each, the projection writes `sheltered`, the
in-our-care breakdown shifts from transit to sheltered, and the nødtelefon's case card
updates on its own. At 02:40 a mother arrives for one of them; the crew finds her in
`Telt 4` from the screen, presses **Hentet af forældre**, and the scout leaves the list
into *Afsluttet*. The other's patrol crosses the line at 06:10; the crew presses
**Genforenet med patruljen**. The in-our-care count reaches zero.

**Edge cases and errors.**

- **A scout arrives whom nobody logged as picked up.** They are on the screen as
  `waiting` (or, if the nødtelefon never got as far as recording anything, not on the
  screen at all). Pressing **Modtaget** on a `waiting` scout must work: the shelter is
  the receiving party and their word is the better evidence. The transition
  `waiting → sheltered` skips `transit`, which the lifecycle permits (`OverrideStatus`'s
  comment already argues why a strict state machine here would refuse the correction it
  exists to make). It is recorded as an acceptance, not as a correction — because it *is*
  one.
- **A scout arrives who is not on the screen at all** (never recorded as withdrawn, still
  `racing`). Same button, same result. The screen must offer a way to reach a scout who
  is not in the list — see §7, the search field. This is the case that most needs to be
  possible at 3am and is the one a "only members already in our care" filter would block.
- **Double-press / two crew members on two laptops.** Both acceptances resolve to the
  same status; the command dirty-checks and publishes nothing the second time (and
  therefore emits no live signal). No error is shown for a no-op: the screen already
  says what the operator wanted it to say.
- **Handover to the wrong ending.** `released` and `reunited` are not interchangeable and
  cannot be silently swapped. A mistake is fixed through the existing manual override on
  the patrol/case screens, which records itself as a correction. This screen does not
  offer an "undo".
- **A patrol is discontinued while its scouts sit in the shelter.** Nothing special:
  `reunited` requires a patrol that finished, so those scouts end at `released`. The
  screen must not imply reunited is available for a patrol with no active members — see
  §7.
- **A scout is moved to another patrol while sheltered.** Cannot happen in practice
  (moves apply to racing survivors) but the screen must render `currentTeamId` differing
  from `initialTeamId` without confusion, as the member modal already does.

## 6. Requirements

### Functional

- [ ] A new route `/hoensegaard` (name `hoensegaard`) and a nav entry labelled
      **Hønsegården** with a chicken icon (`fas fa-kiwi-bird` — FontAwesome Free has no
      chicken; see §11).
- [ ] The screen lists every scout of the active year who **started and is not active**:
      status ∈ {`waiting`, `transit`, `sheltered`, `reunited`, `released`}. `registered`
      and `seated` are excluded (they never started); `racing` and `finished` are excluded
      (they are or were on the route).
- [ ] The list is grouped, in this order, because the groups answer different questions
      and are read at different rates:
      - **I bil — på vej hertil** (`transit`) — the arrivals queue. This is the section a
        crew member opens when a car pulls up: the scouts getting out of it are found here
        and accepted from here. It is first because it is the only section with somebody
        standing in front of it, waiting.
      - **I Hønsegården** (`sheltered`) — who is here, with placering.
      - **Afventer afhentning** (`waiting`) — still out on the trail. Not yet the
        shelter's responsibility to act on, but its warning of what is coming.
      - **Afsluttet** (`reunited`, `released`) — collapsed by default, kept for the record
        and for the handover between shifts.

      `transit` and `waiting` are deliberately **not** one "På vej" section. They look
      alike on a status list and mean opposite things to this crew: a scout in a car is
      about to be their problem and is acted on within minutes, a scout by a road is
      somebody else's problem for now. Merging them buries the actionable rows among rows
      that must not be acted on.
- [ ] **Accepting several scouts at once.** A car arrives with three, and they arrive
      together. The *I bil* section supports multi-select with a single **Modtaget** action
      over the selection and one placering applied to all of them — scouts off one car are
      normally bedded down together. One HTTP request per member is fine: unlike
      whole-team collection there is no atomicity requirement, because each acceptance is
      an independent fact about one child, and a half-succeeded batch leaves the rest
      visibly still in the queue rather than in a wrong state.
- [ ] Each row shows: name, patrol (number and name, linking to the patrol), current
      status with its Danish label, **how long they have been in it** (from
      `spejderstatus.updatedAt`), placering (sheltered only), own phone, guardian phone,
      and a link to the open case if there is one.
- [ ] **Who accepted them, by name.** Custody is a person, not a vehicle: the crew needs
      the driver who took the scout aboard — they are who you ring when a car is overdue,
      and who you ask what state the child was in. That is the **actor of the event that
      put the scout in their current status**, which the platform already records
      (`spejderstatuslog.actorUserId`, and `Actor` on every lifecycle event body). Shown
      for `transit` as "i bil hos <navn>" and for `sheltered` as "modtaget af <navn>",
      which also gives the crew its own shift handover for free.
      No car, licence plate or vehicle id is shown. `PickupAccepted.Car` is not used by
      this screen.
- [ ] **The column appears by itself when authentication does.** Actors are anonymous until
      HQ has real login (§11), so the server omits the field when it has no name and the
      section hides the column when no row in it has one. Nothing is deployed to turn it on:
      the first attributable acceptance makes the column appear. A column of em-dashes is
      noise on a screen somebody reads at 3am, and worse, it reads as a bug — the crew would
      reasonably conclude the screen is broken rather than that nobody is logged in.
- [ ] Header counts per group, and the in-our-care total, so the shelter sees the same
      number the organisers are waiting on.
- [ ] Action: **Modtaget i Hønsegården** — valid from `waiting`, `transit` and `racing`;
      publishes `shelter.accepted`; may carry a placering in the same action.
- [ ] Action: **Sæt/ret placering** — a short label (max 64 chars), settable and
      changeable while sheltered, published as its own event.
- [ ] **The zone vocabulary defines itself as the night goes on.** The zones are not known
      until race start, so they are neither configured nor hardcoded: the placering field
      is an editable combobox whose suggestions are the distinct placeringer already
      recorded this year, most-used first, with free text still accepted. The first scout
      into a zone is typed; every one after that is a pick. That is what stops "Telt 4",
      "telt4" and "t4" becoming three places, and it needs no zone entity, no admin screen
      and no setup step at race start — which is the only kind of solution that survives
      contact with a night where the layout is decided an hour beforehand.
- [ ] Actions: **Hentet af forældre** (`released`) and **Genforenet med patruljen**
      (`reunited`) — publish `handover.completed` with `to`.
- [ ] None of these actions requires a `sosId`. The events are explicitly case-free
      (`messages.go`, "No sosId, deliberately"); the shelter may receive a scout nobody
      opened a case about.
- [ ] Where an *open* case is associated with the scout's patrol, the operation appends a
      summarising activity entry to that case, so an operator watching the case sees the
      arrival without being told. Where there is none, no case is created.
- [ ] A search field finds any scout of the year by name or patrol number, so somebody
      not in the list can still be accepted.
- [ ] Every listed scout links to the existing member detail modal/endpoint
      (`GET /api/member/:memberId`) for address, birthday and full status history.

### Non-Functional

- **Live.** The view loads through `useLiveResource` with `dependsOn: ['spejder',
  'patrulje']` — `spejder` is the entity in the lifecycle subjects (*not* `spejderstatus`,
  which is only the projection's name), `patrulje` for team names, numbers and the
  finish. Type-level, not instance-level, because a newly withdrawn scout's id has never
  been seen before. `pending` drives the table's `:loading`; no bespoke spinner.
- **Unsaved state is never overwritten.** The placering field is an editor: while it is
  dirty, incoming payloads are deferred and applied when the edit ends, and the screen
  says updates are paused — as `KlanListView.vue` and `KortView.vue` do.
- **3am legibility.** Large touch/click targets, high contrast, no destructive action
  behind a single unguarded click on a row. Waiting durations past the alarm threshold
  (task 082) are highlighted.
- **Privacy.** The screen shows children's names, phone numbers and whereabouts. It sits
  behind the same perimeter authentication as the rest of HQ; no new sharing surface, no
  public link, nothing in a URL that identifies a child beyond the existing id scheme.
- **Performance.** One query set per load; the population is tens of rows, not thousands.
  Budget: p95 under 50ms server-side.

## 7. UX / UI Notes

New view `vue/src/views/HoensegaardView.vue`, routed at `/hoensegaard`, added to the nav
in `components/Navigation.vue` between **Nødtelefon** and **Betalinger** — it is a
race-night screen and belongs beside the other one.

Layout: a header strip with the group counts and the in-our-care total, then PrimeVue
`DataTable`s under Danish headings. **I bil — på vej hertil** first, because it is the
arrivals queue and the only section with somebody standing in front of it; **I
Hønsegården** second, the section a parent at the door is answered from. When *I bil* is
empty it collapses to a single quiet line ("ingen på vej") rather than an empty table, so
it does not push the rest of the screen down all night.

Row actions are buttons in the row, not a hidden menu — a tired volunteer should not have
to discover them. The primary action per section differs: *I bil* rows offer **Modtaget**,
both per row and over a multi-selection; *I Hønsegården* rows offer **Hentet af
forældre** / **Genforenet med patruljen** and an inline placering field; *Afventer
afhentning* rows offer no primary action (accepting one is possible but lives behind the
same confirm as accepting a `racing` scout — it asserts an arrival the platform has no
pickup for); *Afsluttet* rows offer nothing.

The placering combobox suggests the placeringer already in use tonight and accepts
anything typed. Placing a multi-selection asks once and applies to all.

**Genforenet med patruljen** is disabled, with a tooltip, for a scout whose patrol has no
active members — that patrol is discontinued and will not cross a line to be reunited at.

Status labels come from the server (`MemberStatuses()`, already served on the patrol and
case payloads), not from a label map in this view. PRD 006 §6 makes the point: a second
label map is how one screen ends up saying "waiting" to an operator at 3am.

Durations render as "siden 21:40 (2t 14m)" in `da-DK` — both the clock time and the
elapsed span, because the first is what gets written on paper and the second is what
triggers a decision.

## 8. Technical Considerations

**Frontend (Vue 3 / TS).**
- New: `views/HoensegaardView.vue`; route in `router/index.ts`; nav item in
  `components/Navigation.vue`.
- Likely extracted component: a placering editor cell with the dirty-defer behaviour, and
  a shared "status chip + duration" cell reusable by the case card.
- Data via `useLiveResource('shelter', ...)`; actions via `http.put`, then rely on the
  live signal for the refresh rather than mutating local state — the write path and the
  read path must not disagree.

**BFF (Go).**
- New handler file `cmd/api/shelter.go`: the read endpoint plus three write endpoints.
- Reads compose existing models: `SpejderStatus` (needs one new query — see below),
  `Members.GetSpejdere` for names and phones, `Teams.GetPatrulje` for team refs, and the
  `sos` querier for the open-case link.
- New query `spejderstatus.Queries.GetByStatuses(ctx, year, []types.MemberStatus)`. Not
  `ListNotActive()`: the set of statuses is the caller's question, and the query should
  not encode a screen's policy. The screen derives its set from
  `types.MemberStatus`-level predicates where it can (`InOurCare()`), so a fourth in-care
  state added to shared-go appears here without an edit.
- New query on the `shelter` table: `DistinctPlacements(ctx, year)`, ordered by use count
  descending, feeding the combobox. It is derived from the placeringer already recorded
  rather than from a zone table, which is what lets the vocabulary come into existence at
  race start without anybody configuring it. Served in the `/api/shelter` envelope, not on
  its own endpoint: it is small, it changes only when a placering is set, and the screen
  wants it at the same moment it wants the list.
- New query `spejderstatus.Queries.GetLatestActors(ctx, year, []types.MemberID)` — the
  actor of each member's most recent lifecycle event. **No new column and no new event
  field:** `spejderstatuslog` already stores `actorUserId` per event, so "who accepted this
  scout" is a fact the platform has been recording since PRD 006 and has simply never
  read back. One query over the log with a `MAX(seq)` per member, batched with an `IN`
  clause over the members already fetched — the `(year, id, seq)` key covers it. Names are
  resolved through `app.models.CrewMember.GetByID`, as `vehicle.go` does for a custodian.

  Deriving it beats storing an `acceptedBy` on the `shelter` table, because the same
  derivation answers it for `transit` — a status the shelter table knows nothing about,
  since nothing about a car is the shelter's to project. It is also the truer statement:
  the accepting party *is* whoever published the acceptance.

  **Caveat, and it is not small:** authentication is perimeter-only today, so
  `authenticate` puts an anonymous user with an empty `UserID` on every request (PRD 001
  §6 Auth). Real login is coming soon but is not here yet, so every actor recorded up to
  that point is blank — and, because the actor is stored on each log row as it happens,
  **authentication does not fill in the past**. Any race run before it lands has no drivers
  recorded, permanently. That is a reason to get login in before the next race, not a
  reason to build something else here.

  So the column is built and left to fill itself in (§6): no driver-name text field for the
  shelter or the operator to type. That workaround would be a second source of truth for
  something the event model already expresses, it would have to be migrated or ignored once
  accounts exist, and it would outlive by years the few weeks of gap it was built for. The
  `Actor` struct carries a `Name` beside the id precisely so a producer can attribute an
  act before HQ has user accounts — a future car interface can populate it without one.
- Writes go through **new commands on the `spejderstatus` commander** —
  `AcceptIntoShelter`, `CompleteHandover` — mirroring `RequestWithdrawal`'s shape
  (dirty-check, publish, return `*Change`). They reuse the existing `memberStatusOperation`
  plumbing in `member.go` but with the `sosId` requirement relaxed: the case is optional
  here, which is a change to that helper's contract and should be made explicit rather
  than by passing an empty id through a code path that assumes one.
- `spejderstatus` may not import `nathejk.dk/...` (it is queued for lifting to shared-go,
  task 083). New commands must respect that; `lift_test.go` enforces it.

**Placering: a separate table, not a column on `spejderstatus`.**
New local package `go/nathejk/table/shelter/` with its own projection, consuming
`spejder.*.shelter.accepted`, `spejder.*.shelter.placed` and
`spejder.*.handover.completed` (clearing the row on handover). Two reasons, and the
second is the load-bearing one:

1. `spejderstatus` owns *status and team membership*. Where a bed is is a fact about the
   shelter, not about the lifecycle.
2. `spejderstatus` is about to be lifted to shared-go verbatim. Adding an
   hq-specific column now makes that lift a rewrite.

And a practical one: `CREATE TABLE IF NOT EXISTS` never alters an existing table, so a new
column on `spejderstatus` would silently not exist in any dev database — a new table does.

**New event.** `spejder.*.shelter.placed`, body `{memberId, teamId, placement, actor}`,
`Status() → MemberStatusSheltered`. Defined in `spejderstatus/messages.go` beside its
siblings so the status mapping stays in one place, consumed by both projections (the
status write is idempotent; the placement write is the point). `ShelterAccepted` gains an
optional `Placement string` — free, since nothing has ever published it.

**API endpoints** (all year-scoped via `X-YearSlug`; **all four need `@Summary` /
`@Description` / `@Tags` / `@Router` OpenAPI annotations**, as `cmd/api/order.go` and
`internal/live/http.go` have):

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/shelter` | The started-not-active population, grouped, with counts |
| PUT | `/api/member/:memberId/shelter` | Accept into the shelter; optional `placement` |
| PUT | `/api/member/:memberId/placement` | Set or change placering |
| PUT | `/api/member/:memberId/handover` | `{to: "released"\|"reunited"}` |

`/api/shelter` singular, not `/api/members/shelter`: it is a screen's view of a place. Note
the httprouter constraint that produced `/api/members/care` — a static segment cannot sit
where a sibling has a wildcard — so this must not be `/api/member/shelter`.

**Data / storage.** One new table (`shelter`), no migrations (projections rebuild from
JetStream on boot). No changes to `spejderstatus`'s schema.

There is also no data to preserve: no sos cases and no member status history exist yet, so
the `shelter` table's shape can still be changed by editing `table.sql` and dropping the
table. That freedom expires the moment the first race night runs — `CREATE TABLE IF NOT
EXISTS` never alters an existing table, so a column added later is silently absent from
every database that already has the table. Worth spending now rather than discovering in
November: get the columns right while they are free.

**Live wiring.** The new `shelter` consumer must go in the `projections` slice in
`cmd/api/main.go`, not straight into the mux — outside that slice it is wrapped by nothing
and emits no signal, and the screen would look live and never update. The entity token
stays `spejder`, since the subjects are spejder subjects.

**Dependencies & risks.**
- *The arrivals queue is only as good as its input, and its input is not ours.* **The
  crew's primary section is `transit`, and nothing in hq publishes `transit` except the
  nødtelefon operator's manual override.** Until the car interface exists, a car whose
  pickup nobody recorded arrives with scouts who are still `waiting` — or still `racing` —
  and *I bil* is empty while the crew is receiving people. This is the single biggest risk
  to the feature, and it is a process risk rather than a code one, so the design absorbs
  it in two ways rather than pretending it away: accepting from `waiting` and from
  `racing` is a supported action, and the search field (§6) reaches a scout in no section
  at all. It is also the strongest argument for the car interface being the next PRD.
- *Two writers on one status.* The nødtelefon can override a status while the shelter is
  accepting. Last write wins, both are on the timeline, and the override records itself as
  a correction; no locking. Worth a note in the code, not a mechanism.
- *Reunited depends on the patrol finishing*, which the platform does not currently record
  as a patrol-level event — the crew judges it. Open question below.
- *A typo becomes a permanent suggestion.* The combobox's vocabulary is derived from what
  was typed, so "Telt 4" and a fat-fingered "Telt 44" both appear in the list. Ordering by
  use count keeps the real one at the top and the mistake at the bottom, and correcting
  the affected scout's placering removes it. No rename tool in v1.

## 9. Success Metrics

- **The in-our-care count reaches zero on race night without manual overrides.** Measured
  by the ratio of `status.overridden` events to `shelter.accepted` +
  `handover.completed` events: the corrections are the admission that the chain of custody
  is fiction (`messages.go` says as much), so a falling ratio is the whole point. Target:
  fewer than 1 override per 10 acceptances.
- Every scout in `sheltered` has a non-empty placering. Target: 100% by the end of the
  night; anything less means the field is in the wrong place on screen.
- Median time from a car's arrival to `shelter.accepted` — proxy: the gap between
  `pickup.accepted` (once cars exist) or the case comment and the acceptance. Target under
  5 minutes.
- Qualitative, and the one that decides whether this shipped: the crew stops keeping the
  paper list.

## 10. Rollout / Task Breakdown

Sequenced so the screen is useful after the first frontend task and complete after the
last. Backend read path first, because the write actions are worthless without the list.

- [ ] Task: `spejderstatus.GetByStatuses` query + tests (086)
- [ ] Task: `shelter` projection (table, consumer, querier incl. `DistinctPlacements`) for
      placering, wired into the `projections` slice (087)
- [ ] Task: `ShelterPlaced` event + `Placement` on `ShelterAccepted` in
      `spejderstatus/messages.go` (088)
- [ ] Task: `AcceptIntoShelter` / `CompleteHandover` / `SetPlacement` commands, with
      dirty-checks and the optional-case relaxation of `memberStatusOperation` (089)
- [ ] Task: `GET /api/shelter` handler with OpenAPI annotations (090)
- [ ] Task: the three write endpoints with OpenAPI annotations (091)
- [ ] Task: `HoensegaardView.vue` — grouped read-only list with *I bil* first, live via
      `useLiveResource`, route + nav icon (092)
- [ ] Task: row actions (modtaget / handover) wired to the endpoints (093)
- [ ] Task: multi-select acceptance in the *I bil* section, one placering for the selection
      (094)
- [ ] Task: placering combobox — suggestions from `DistinctPlacements`, free text accepted,
      dirty-defer and the "updates paused" affordance (095)
- [ ] Task: search field for accepting a scout not in any section (096)
- [ ] Task: sos case summarising entry for shelter operations (reuses PRD 006's
      `RecordMemberStatusChanged`) (097)
- [ ] Task: `GetLatestActors` query + "i bil hos / modtaget af" column (098)

No feature flag: the screen is additive, reachable only from a new nav entry, and its write
path publishes events the projection already consumes.

**Timeline.** The next race is roughly a month out (from 2026-08-23), so the full scope is
reachable rather than needing a cut-down first version. The read path is worth landing
early regardless: it is the half that can be exercised against a quiet system, and the
write path is only truly testable with people arriving.

## 11. Open Questions

**Resolved 2026-08-23 (crew):**

- ~~Placering: free text or a named set?~~ **Neither.** The zones are not known until race
  start, so the vocabulary is derived from what has already been recorded and offered as
  suggestions — see §6. No configuration, no zone entity, and it converges on consistent
  names by itself.
- ~~Is `transit` worth showing?~~ **It is the primary section.** A car arriving is the
  crew's main event, and its passengers must be findable and acceptable from that section.
  §6 and §7 are rewritten around it, and the risk that nothing publishes `transit` is
  promoted to the top of §8's risks.
- ~~Which car holds them, and can the queue be grouped by car?~~ **The driver, not the
  car.** Custody is a person: the driver who accepted the scout is who you ring when a car
  is overdue. No vehicle is shown and `PickupAccepted.Car` goes unused by this screen. It
  is derived from the actor of the event that set the status, which `spejderstatuslog`
  already stores — so no new column and no new event field. What remains is not a design
  question but an identity gap, resolved below.
- ~~Does the blank driver name need an interim workaround?~~ **No.** HQ gets real
  authentication soon, though not tonight, so the column is built now, shows nothing in the
  meantime, hides itself while it is empty, and starts working on its own the day accounts
  exist (§6). Specifically the nødtelefon's override dialog is *not* being extended to ask
  the operator to type a driver's name — §8 has the argument for why a temporary second
  source of truth is the expensive option.

  One consequence, now defused: the actor is written to the log as each event happens, so
  login does **not** backfill. There is nothing to backfill — no sos cases and no member
  status history exist yet — and the next race is a month out, so login is expected to land
  first and the column will most likely ship already working. If it slips, the cost is one
  race with no driver attribution rather than a hole in existing data.

  Sequenced accordingly: `GetLatestActors` is the last task in §10, being the only piece
  that produces nothing visible on the day it is delivered.

**Still open:**

- **Does the crew need to see who is coming *before* a car has them?** *Afventer
  afhentning* is the answer proposed, but if the practical warning is "a car has been sent
  for four scouts", that is a dispatch fact the platform does not record either.
- **`reunited` — who decides the patrol finished?** There is no patrol-finish event. Is the
  crew's judgement enough (proposed), or should this wait on a finish signal?
- **Should the shelter be able to accept a `racing` scout?** §5 argues yes, because it
  happens, and §8 now leans on it as the mitigation for an empty arrivals queue. It means
  the screen can create a lifecycle jump with no withdrawal recorded. Confirm with the
  nødtelefon owners.
- **Nav icon.** FontAwesome Free has no chicken. `fa-kiwi-bird` is the nearest bird,
  `fa-egg` the nearest joke. Confirm, or supply an SVG.
- **Waiting alarm threshold** is still unsettled (task 082); this screen is a second
  consumer of whatever it becomes.
- **Does the crew need an offline/paper fallback?** If HQ's wifi drops at 3am, the paper
  list is what they have. Out of scope, but worth knowing whether the screen must be
  printable.
