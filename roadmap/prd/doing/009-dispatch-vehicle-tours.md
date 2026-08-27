# PRD 009 — Dispatch (Kørsel): vehicle tours and tasks

**Status:** doing
**Author:** agent session (with the product owner's request)
**Created:** 2026-08-27
**Last updated:** 2026-08-27 (revised three times during drafting as the product owner answered — tours, dispatchable subsections, logistics as the operator, desk-only in this repo, English names in source, SOS priority vocabulary, one allowance for all vehicles — see §11)
**Approved:** 2026-08-27
**Shipped:**
**Target users:** organizer (logistics crew — the dispatch desk and the drivers), organizer (nødtelefon operator, as the originator of pickups), organizer (Hønsegården crew, as a reader)

<!--
Status must match the folder this file is in: draft/, doing/ or done/.
Leave Approved blank until the PRD moves to doing/, and Shipped blank until it
moves to done/. See roadmap/prd/README.md for the lifecycle.
-->

---

## 1. Summary

A dispatch desk for the cars, run by the **logistics** section. Two things are recorded:

- a **task** — something that needs moving: collect a scout from the roadside, carry maps from
  Post 2A to 2B, clear the materials out of a closed Start, get dinner to the loks before seven;
- a **tour** — a run by one car: from A to B with as many stops as it takes. Tasks are put into
  tours.

A tour is assigned to a **dispatchable subsection** of logistics — a small unit that holds a
vehicle, a driver and possibly a co-driver — which is a shape the organisation tree can already
express.

The screen exists to answer the question the desk cannot answer today: **when?** When can the
scout expect to be collected, and is dinner going to make 19:00. The tour is what makes that
answerable without GPS: a planned run with ordered stops *is* an estimate, made by a human who
knows the roads.

**Naming.** `dispatch`, `tour`, `task` and `stop` in the source; **"Kørsel"** in the interface.
English in code wherever a clean English word exists — which it does here, unlike `patrulje`,
`klan` and `lok`.

**What this PRD does not do:** compute arrival times from vehicle positions, or give the driver
a screen. There is a tracker on each car and no access to its feed yet; the driver's own screen
is a separate app in a separate repo. Everything here works without either and gets better when
they arrive — see §4 and §8.

## 2. Problem & Motivation

**What problem does this solve?** Cars are dispatched by phone and memory. Nothing is written
down, so the desk cannot answer the three questions it is asked all night:

- *When is someone coming for my scout?* The nødtelefon operator has no idea. A patrol stood
  at a roadside with a member who cannot walk is blocked until a car arrives (see
  `MemberStatusWaiting` — "the team may not continue"), and the honest answer today is a shrug.
- *Where is the car I sent out an hour ago, and what did I ask it to do?* Held in one person's
  head, and lost at shift handover — the problem PRD 001 solved for phone calls and this
  leaves unsolved for cars.
- *Is dinner going to be served on time?* Nobody knows until it is late.

**It also leaves a hole in the custody chain.** PRD 007 §8 names it as the single biggest risk
to Hønsegården:

> **The crew's primary section is `transit`, and nothing in hq publishes `transit` except the
> nødtelefon operator's manual override.** Until the car interface exists, a car whose pickup
> nobody recorded arrives with scouts who are still `waiting` — or still `racing` — and nobody
> in *På vej* carries the `transit` status while the crew is receiving people. […] It is also
> the strongest argument for the car interface being the next PRD.

**Why now?** Three PRDs have deferred this interface and each made the case for it. PRD 001 §4
listed "the car/driver interface" as a later PRD; PRD 006 §4 repeated it and reserved the seam
(*"from the car door onwards the events belong to the car and shelter interfaces"*); PRD 007
shipped the receiving end and left the sending end missing. The shelter screen now has a *På
vej* section that, in a real race, would be empty while cars are actually on their way.

**Evidence.**

- The product owner's request: dispatching vehicles to specific tasks; pickups originating from
  the nødtelefon; maps between checkpoints; materials from a closed Start; dinner to the
  bandits and the posts; GPS present but unavailable; drivers who must sleep; and *"one
  question we want to be able to answer is: what is the wait time for a specific task"*.
- `MemberStatusTransit`'s own documentation already assumes this interface: *"The car is not
  necessarily driving to HQ: it may be collecting members from other teams first"* — a car
  running a tour with several stops is the modelled reality, and nothing records it.
- `spejderstatus.ErrAlreadyCollected` exists to guard a race — an operator pressing resume
  *"in the same moment the driver accepts the member aboard"* — that nothing can currently
  trigger, because no driver acceptance is recorded anywhere.
- The vehicles are already in the platform: the `vehicle` entity (plate, custodian, driver,
  section, seat count) is managed on the Organisation page. They are enrolled and idle.

## 3. Goals

- **Every task and every tour is written down** — what, where from, where to, which unit is
  doing it — so a handover loses nothing and the desk can look up what it promised an hour ago.
- **The desk can answer "when", with a number it can defend.** Not a guess dressed as an ETA:
  a time derived from a planned tour, or a time a human committed to, or an honest "not
  planned yet, and here is how long they have already waited".
- **Deadline-bound tasks surface before they are late.** Dinner at 19:00 is visible as at risk
  while there is still time to act.
- **The custody gap closes.** Recording a pickup puts the scout in `transit`, so Hønsegården's
  *På vej* fills from the cars rather than from an operator remembering to override a status.
- **Dispatch works with the capacity that actually exists.** Units are on duty by agreement,
  and the board shows who is on now and who comes on next.

## 4. Non-Goals

- **Vehicle positions and a live map.** There is a tracker on each car and no access to the
  feed. This PRD defines the seam it will arrive through (§8) and depends on nothing from it.
  No map view, no distance-based ETA, no geofencing.
- **Route optimisation and automatic assignment.** A human builds the tour and orders the
  stops. The platform records the plan and shows its consequences. Suggesting an order is a
  later and much harder question.
- **A driver-facing screen.** The drivers *will* get one, and it is **another app in another
  repo** — not this one. What this PRD owes it is an API it can call and events it can read,
  which §8 records as a seam. There is precedent: the scanner at the posts is already a separate
  app whose scans arrive as `NATHEJK.*.qr.*.scanned`.

  Until then the desk is the only screen: the dispatcher rings the driver and ticks the stops
  off as they report in. **Not** for security reasons — there are fewer than ten cars and the
  desk knows every driver and their phone number — but because it is out of scope here.
- **Turn-by-turn or distance-derived arrival times.** Without positions there is nothing to
  derive them from, and a fabricated ETA is worse than none — it gets quoted to a patrol
  standing in the dark.
- **Vehicle, crew and section management.** Registering cars, assigning drivers and building
  the organisation tree are existing features on the Organisation page. This PRD consumes
  them and adds one flag.
- **Cargo inventory.** What is in the car is a line of text. We are not tracking how many maps
  exist or how much food was loaded.
- **Fuel, mileage, expenses, damage reports, driver hour compliance.** Not a race-night desk.
- **The shelter's own workflow.** PRD 007 owns everything from the handover at HQ onwards.

## 5. User Stories & Scenarios

- As a **logistics dispatcher**, I want to build a tour out of the tasks waiting, so that one
  car's run is one plan I can ring a driver about.
- As a **logistics dispatcher**, I want every task written down with the tour it belongs to,
  so that a shift handover does not lose what was promised to whom.
- As a **nødtelefon operator**, I want to turn a case into a task without leaving the case, so
  that a waiting scout enters the logistics queue the moment I hear about them.
- As a **nødtelefon operator**, I want to read off when the car will reach them, so that a
  patrol can decide whether to wait where they are.
- As a **kitchen/gøgler lead**, I want dinner deliveries visible with their serving time, so
  that being late is something we see coming.
- As the **Hønsegården crew**, I want scouts to appear in *På vej* when a car takes them
  aboard, so that we know who is coming without being told twice.

### Primary scenario — a scout pickup joins a tour

1. A patrol rings the nødtelefon: one member cannot continue, on the road by Post 2B.
2. The operator marks the member as wanting to leave (PRD 006 — the member becomes `waiting`)
   and presses **Bestil kørsel** on the case.
3. A task is created: kind *pickup*, pick-up place "ved Post 2B", drop-off HQ, carrying the
   case, the patrol and the member. It lands in the logistics board's **Ikke planlagt** queue
   with a clock running.
4. The logistics dispatcher sees unit **Bil 2** on duty and already running a tour with two
   stops. They add the task to that tour as a third stop, ordered after the stop at 2A.
5. The tour's plan now says: 2A at 22:10, 2B at 22:35, HQ at 23:00. The task inherits **22:35**
   as its expected time.
6. The nødtelefon operator, still on the phone, reads 22:35 off the case and tells the patrol.
7. The driver collects the scout and rings in. The dispatcher presses **Hentet** on the task;
   the member becomes `transit` and appears in Hønsegården's *På vej*.
8. The car reaches HQ, the shelter accepts the scout (PRD 007), and the task is **Færdig**.
   When its last task is done, so is the tour.

### Scenario — maps from Post 2A to Post 2B

A checkpoint rings: wrong maps. The dispatcher creates a *transport* task, from Post 2A to
Post 2B, "kort til postlinje 2", normal priority, no deadline. It waits in the queue until a
tour passes both — which is exactly the judgement a tour makes visible, because the board can
show which planned tours already stop at 2A.

### Scenario — Start has closed

Start's last patrol has left. A *collection* task: from Start to HQ, "materiel og skilte",
needs most of the boot. It carries **tidligst** — there is no point sending a car while
patrols are still departing — and no deadline. It will most likely be a tour of its own.

### Scenario — dinner at 19:00

At 16:00 the kitchen says dinner is ready at 18:00 and must be served at 19:00. The dispatcher
creates four *delivery* tasks, one per lok, each **skal leveres 19:00** and **tidligst 18:00**,
and builds one tour: HQ → Lok 1 → Lok 2 → Lok 3 → Lok 4 → HQ, departing 18:00. The plan shows
Lok 4 at 18:50 — inside the deadline, so the tour is viable. Had it shown 19:20, the desk would
know at 16:00 that it needs two cars, which is the entire point.

### Edge cases and failures

- **No unit on duty.** The queue still accepts tasks; the board says the next unit comes on at
  22:00, and an unplanned task's estimate is built from that rather than from nothing.
- **The scout carries on after all.** PRD 006 lets a `waiting` member resume. If a car has
  already taken them aboard, `ErrAlreadyCollected` wins — and the task is what recorded it. If
  not, the task is cancelled with a reason and removed from its tour.
- **A tour is re-planned mid-run.** Normal, not exceptional: a task is added to a tour already
  underway, or moved to another tour, or pulled back to the queue. Stops that have been
  visited are fixed; the rest can be reordered.
- **A task is dropped from a tour.** It returns to the queue **with its original waiting clock
  intact** — the scout has been waiting since the call, not since the re-plan.
- **A plan goes stale.** A planned stop time in the past with the stop not visited is shown as
  overdue. That is the point of recording plans: it makes broken ones visible.
- **The car breaks down.** No special modelling: the unit goes off duty and its unvisited stops
  return to the queue, each keeping its own clock.
- **A unit with no vehicle, or no driver.** A dispatchable subsection missing either is not
  capacity. The board says so rather than silently offering it.

## 6. Requirements

### Functional

**The dispatch unit — a dispatchable subsection**

- [ ] A tour is assigned to a **section**, not to a car or a person. The unit is a subsection
      of logistics that holds a vehicle, a driver and possibly a co-driver.
- [ ] Sections carry a **`dispatchable`** flag, opted into per section per year, exactly as
      `sos_assignable_section` works for nødråb assignees. Only dispatchable sections are
      offered as tour owners.
- [ ] A unit's **vehicle** is the vehicle whose `sectionSlug` is that subsection; its **people**
      are the crew members in it. The vehicle's existing `driverUserId` names the driver;
      anyone else in the subsection is a co-driver.
- [ ] The board shows a unit as **not ready** when it has no vehicle or nobody in it, with
      which is missing.
- [ ] More than one vehicle in a dispatchable subsection is a **configuration mistake, flagged
      not forbidden** — the desk can still work, and the Organisation page is where it is fixed.

**Duty windows**

- [ ] A unit has **duty windows** — from/to — agreed in advance with the logistics crew and
      recorded per unit, not per person. The unit is what is available or asleep.
- [ ] The board shows **which units are on duty now**, and **when the next one comes on** when
      none are.
- [ ] Planning a tour outside its unit's window is **allowed but warned about**. The race does
      not stop for a roster, and a system that refuses the real world gets worked around.

**Tasks**

- [ ] A task has a **kind**: `pickup` (people), `transport` (A to B), `collection` (fetch to
      HQ), `delivery` (take out from HQ). Four kinds because they read differently on a board
      and default their places differently — not because their lifecycles differ.
- [ ] A task has a **pick-up place** and a **drop-off place**, each of which is a checkpoint, a
      lok, HQ, or free text. Free text is the normal case for "på Slangerupvej ved skovbrynet",
      not a fallback for missing data.
- [ ] A task has a **description**, a **priority**, and optional **space needs** in words.
- [ ] Priority uses the **same vocabulary as an SOS case**: grøn / gul / rød. Two race-night
      desks should not have two ways of saying urgent, and a task that came from a red case
      should be able to arrive carrying that word. Same values, same Danish labels, same theme
      colours — see §8 for how that is shared rather than copied.
- [ ] A `pickup` task may reference **the members being collected** and the SOS case it came
      from.
- [ ] Times, all optional except the first: **oprettet** (the waiting clock), **tidligst** (not
      before), **skal leveres** (hard deadline).
- [ ] **State**: `queued` → `planned` (in a tour) → `underway` → `done`, plus `cancelled` from
      any state. A `pickup` additionally records **hentet** — people aboard — on the way to
      `done`, because that is when custody changes and it is not when the task finishes.
- [ ] Dropping a task from a tour returns it to `queued` **without resetting `oprettet`**.
- [ ] Cancelling requires a **reason**.

**Tours**

- [ ] A tour belongs to **one unit**, has a **planned departure**, an **ordered list of stops**,
      and a state: `planned` → `underway` → `completed`, plus `cancelled`.
- [ ] A **stop** is a place, a position in the order, an optional planned time, and the tasks
      actioned there. A task that moves something occupies **two** stops — where it is loaded
      and where it is unloaded — and the board must not let the unload be ordered before the
      load.
- [ ] Stops can be **reordered, added and removed** while the tour is `planned` or `underway`;
      **visited stops are fixed**.
- [ ] Marking a stop **visited** advances the tour and completes or progresses the tasks at it.
- [ ] A tour with no remaining unvisited stops offers to **complete**.
- [ ] **Warn when a tour's pickups exceed the vehicle's seats.** `seatCount` is already on the
      vehicle, and a car sent for five scouts with four seats is a wasted run and a patrol left
      standing. A warning rather than a refusal: seats get folded down, a member sits with a
      leader, and the desk knows things the platform does not. Counted across the tour's
      unvisited pickup stops, since scouts already dropped at HQ have left the car.

**Answering "when?"**

- [ ] Every task shows **how long it has waited**, always, from `oprettet`. The number that
      needs no model and is never wrong.
- [ ] A task in a tour takes its expected time from **its stop's planned time**. This is the
      primary answer, and it is a human's plan rather than a computation.
- [ ] Planned stop times are **derived** from the tour's departure plus a per-leg allowance, and
      **any stop can be overridden by hand**. Deriving keeps planning fast — a dispatcher
      building a six-stop tour at 3am should not type six times — and the override is what makes
      the derivation acceptable: the moment the desk knows better, it can say so. An overridden
      stop is marked as such, and the stops after it re-derive from it.
- [ ] A queued task with no tour shows an **estimate**, visibly marked *anslået*:
      `max(now, tidligst, next unit on duty) + allowance(kind)`. Deliberately crude — see §8.
- [ ] The plan **beats** the estimate on screen. A dispatcher who has built a tour knows more
      than the queue does.
- [ ] Deadline tasks show **time until skal leveres**, and are flagged when the plan lands
      after the deadline, or when they are still queued inside a configurable window of it.
- [ ] The board answers at a glance: **how many tasks are unplanned, how long the oldest has
      waited, how many tours are out, and which deadlines are at risk.**

**Seams with the rest of the platform**

- [ ] From an SOS case, **Bestil kørsel** creates a pickup task pre-filled with the case, its
      patrol and the waiting members.
- [ ] Marking a pickup **hentet** transitions those members to `transit`, filling Hønsegården's
      *På vej* from the cars. This closes PRD 007 §8's biggest risk.
- [ ] The SOS case shows its tasks and their expected times, so the operator on the phone does
      not need the dispatch board open.
- [ ] Every state change is **appended to a timeline** with a timestamp and a note, as SOS
      cases are. A dispatch desk is a log first.
- [ ] The whole screen is **live** (PRD 004), with correct `dependsOn` tokens.

### Non-Functional

- **Live and cached**, per PRD 004: `useLiveResource`, dependencies declared, `pending` wired
  to loading state, nothing that fetches once on mount.
- **Unsaved state is protected.** The board is an editor — a half-built tour or a drag mid-flight
  must not be replaced by an incoming payload. Use `useDeferredApply` and say on screen when
  updates are paused.
- **Two desks, one board.** Logistics dispatch and the nødtelefon are different people at
  different screens writing the same tasks. Last write wins, everything is on the timeline, no
  locking — the answer PRD 007 reached for the same situation.
- **Usable at 3am by a tired volunteer.** Danish throughout. The oldest waiting task and the
  nearest deadline visible without scrolling.
- **Usable on a phone.** The dispatcher is not always at a desk. Not a driver app; the same
  board, narrow.
- **No authentication dependency.** Attribution is by explicit selection — see §8.
- **Times carry a weekday** wherever they cross midnight, as the shelter screen does: a race
  runs through a night and "21.40" alone is ambiguous.

## 7. UX / UI Notes

**New route:** `/koersel`, "Kørsel" in the navigation, beside Nødtelefon and Hønsegården — the
race-night screens. A separate screen rather than a panel inside the nødtelefon, because
logistics runs it and the nødtelefon only originates tasks and reads times back.

The path is Danish while the source is English, matching what is already there: the routes are
the user-facing surface (`/poster`, `/hoensegaard`, `/klan`) and the components behind them are
not.

**The board** is two panes:

| Ikke planlagt | Ture |
|---|---|
| queued tasks, oldest first, deadline-at-risk pinned to the top | one card per tour, grouped by unit, stops in order |

Tasks are moved into a tour by drag-and-drop or by a "Læg i tur" action; a tour card lists its
stops with planned times and the tasks at each. Reordering stops is drag-and-drop within the
card. The precedent is `KlanListView`'s LOK arrangement and `OrganisationView`'s tree — both
hold unsaved arrangements and both defer live updates while dirty; this must do the same.

**The capacity strip**, above both panes: each dispatchable unit with its vehicle, driver, duty
window, and whether it is on duty now; when none are, *"Næste enhed på vagt 22:00"*. This is
what makes an estimate legible instead of magic, and what makes a not-ready unit visible.

**A task card** carries: kind icon, pick-up → drop-off, what is being moved, the waiting clock
or the planned time, priority, and its tour. A pickup card also shows the scout's name and
links to `MemberDetailDialog` (PRD 008), so the guardian's number is one click away.

**Deadline banner.** Reuse the pattern from the checkgroup teams dialog: when a deadline is
inside the next hour and the task is unplanned or its plan lands late, a `Message` says so with
a shortcut that filters the board to it. Deliberately the same vocabulary — two race-night
screens should not invent two ways to say "you are about to be late".

**Place picker.** One control that offers checkpoints, loks and HQ as groups and accepts free
text. The grouped-picker work from `composables/personnelTree.ts` is the model.

**From a case.** `SosView` / `SosTeamCard` gain a **Bestil kørsel** action on a waiting member
and on the case, plus a list of the case's tasks with expected times.

## 8. Technical Considerations

### The dispatch unit needs no new entity

This is the cheapest part of the design and worth stating plainly: a subsection holding a
vehicle, a driver and a co-driver is **already expressible**. Sections form a tree
(`parentSlug`), crew members carry a `sectionSlug`, and `vehicle` carries a `sectionSlug` too.
So "Bil 2" as a subsection of "Logistik", with one vehicle and two crew members in it, is a
thing the Organisation page can build today with no changes.

What is missing is only the flag. Add **`dispatchable`** per section per year, following
`sos_assignable_section` exactly: a `dispatchable_section` table, a list of slugs returned
beside the sections, and `PUT /api/section/:slug/dispatchable` mirroring
`/api/section/:slug/sos-assignable`. Not a column on the section entity — `section` lives in
shared-go and knows nothing about kørsel, and PRD 001 already took this decision for the same
reason.

**Rejected alternative:** a new `dispatchunit` entity pairing a vehicle with drivers. Rejected
because it duplicates the organisation tree, needs its own editor, and would immediately
disagree with `vehicle.sectionSlug` about which car belongs where.

### Frontend (Vue 3 / TS)

- New view `vue/src/views/DispatchView.vue`, lazy-loaded at `/koersel`.
- `components/DispatchTaskDialog.vue` (create/edit), `components/DispatchTourCard.vue`.
- English identifiers, Danish strings. The labels an operator reads are Kørsel, Tur, Opgave,
  "Ikke planlagt", "På vej" — and nothing in the code is named after them.
- Live through `useLiveResource`; deferral through `useDeferredApply`, because the board holds
  an unsaved tour arrangement.
- **Clocks must advance.** Waiting times and countdowns are wrong a minute after render and
  Vue will not recompute them — nothing they depend on changed, only time passed. Use the
  shared `useNow()` from `composables/shelter.ts`, one interval for the screen.
- Danish formatting via `composables/datefilters.ts`.

### BFF (Go)

- **A new entity in hq**: `go/nathejk/table/dispatch/` with the repo's package layout
  (`table.go`, `consumer.go`, `query.go`, `commands.go`, `filter.go`, `table.sql`), owning both
  tasks and tours — they are one aggregate in practice, since a stop is meaningless without
  its tour and a task's state is driven by its stops. hq-owned rather than shared-go, following
  `sos`, `shelter` and `spejdernote`: tilmelding has no use for it, and tasks 055 and 083 exist
  for lifting entities once they have settled.
- **Duty windows** as a small `dispatchduty` table keyed by section slug, the same shape as
  `checkpersonnel` (`startUts`, `endUts`) and deliberately not the same table: a shift on a
  post and a shift behind a wheel are different facts, and one table would make "which units
  are driving now" a query with a checkpoint join in it.
- **Events** on the house convention: `NATHEJK.{year}.dispatch.{taskId}.{created|updated|
  planned|unplanned|underway|pickedup|completed|cancelled}` and
  `NATHEJK.{year}.tour.{tourId}.{created|stops.changed|underway|stop.visited|completed|
  cancelled}`, plus `NATHEJK.{year}.dispatchduty.{id}.{set|removed}`.
- **The projection must go in the `projections` slice in `cmd/api/main.go`**, not straight onto
  the mux. Outside that slice it is wrapped by nothing, emits no live signal, and the board
  would look live and never update. The client's `dependsOn` tokens are the subjects' entities
  — `dispatch`, `tour`, `dispatchduty` — not the projection's name. A board showing tasks, tours,
  units and members needs `dispatch`, `tour`, `dispatchduty`, `section`, `crewmember`, `crew`,
  `vehicle`, `spejder` and `sos`; get them from the projections that own the events, and expect
  the dev-console warning to catch a wrong one.
- **Commands dirty-check** before publishing, as the repo does. Note the consequence, learned
  on the klan status override: a command that publishes nothing emits no signal, so a UI
  relying solely on the signal to confirm a save must also refresh.

### The custody seam — the one interface change outside this feature

Marking a pickup **hentet** must transition its members to `transit`. Per PRD 006 §8 those
transitions belong to this interface, and `spejderstatus.Commands` deliberately omits them
today (*"A driver's pickup is still not ours to publish"*).

Add to `spejderstatus.Commands`:

```go
AcceptPickup(ctx, actor Actor, year types.YearSlug, ids []types.MemberID, section types.Slug, driver types.UserID) ([]Change, error)
```

alongside `AcceptIntoShelter`, on the same argument the shelter's acceptances were admitted on:
**custody is confirmed by the receiver**. The driver is the receiver; the dispatcher records it
on their behalf until drivers have a screen. A batch signature because one stop collects
several scouts, and two members of one patrol leaving together is one act.

Note the unit is passed as a **section slug** rather than a vehicle id: the unit is who took
them, and it survives a car being swapped mid-night. It must respect `ErrAlreadyCollected` and
the existing precedence — an acceptance beats a resume, *"because it reflects a member
physically sitting in a car"*.

### The driver's screen is another app, and this is its API

The drivers will get their own screen and it will not live in this repo. That makes the
endpoints in this PRD an integration surface rather than a private back end for one Vue view,
and it changes two things about how they are built:

- **The state transitions stay first-class endpoints.** `POST /api/dispatch/tour/:id/stop/:stopId/visited`
  is exactly what a driver app would call when a stop is done. Folding these into a general
  `PATCH` on the tour, or into UI-only actions, would mean rebuilding them later.
- **OpenAPI annotations are not paperwork here.** They are how the other repo learns the shape
  without reading Go.

There is precedent for the split: the scanner used at the posts is already a separate app, and
its scans reach hq as `NATHEJK.*.qr.*.scanned` events. A driver app can take the same route —
REST for its reads and writes, or events if it turns out to need offline queueing, which a car
in a forest plausibly does.

**Authentication is not the blocker it was for PRD 007.** There are fewer than ten cars and the
desk knows every driver and their number, so the product owner has ruled the identity question
out of scope rather than solved: whoever builds the driver app decides how a phone says which
unit it belongs to. Nothing in this PRD depends on the answer, because the desk records
everything in Phase 1.

### Attribution without authentication

hq has no user identity: `authenticate` sets every request's user to `anonymous`. PRD 007 hit
this and chose to build the driver column and let it fill itself in when authentication lands,
rather than adding a text field that would later have to be removed.

This PRD needs attribution now, because "which car has my scout" is the question. It takes the
other available route: **the unit is selected, not inferred.** The dispatcher picks the
dispatchable subsection, and the driver comes from that unit's vehicle. When authentication
arrives it can fill in the *actor*; the *unit* stays an explicit choice, because the person at
the keyboard is not the person in the car.

### Priority: mirrored from SOS, shared not copied

A task's priority is the SOS severity vocabulary — `green` / `yellow` / `red`, labelled Grøn /
Gul / Rød, rendered with the theme's `success` / `warn` / `danger` rather than hex. Two desks
working the same night should not have two words for urgent, and a pickup created from a red
case should be able to arrive red.

**Front end.** `composables/sos.ts` already holds `severityLabel`, `severityTagSeverity` and
`severityOptions`, with a comment saying they are kept in one place *"so the list badge and the
detail select cannot drift apart"* — exactly the drift a second copy in a dispatch view would
reintroduce. But importing `sos.ts` from a delivery task is the wrong dependency to read. So:
move those three helpers to a neutral `composables/severity.ts` and re-export them from
`sos.ts`, leaving the nødtelefon's call sites untouched. Small task, listed in §10.

**Back end.** `dispatch` defines its own three-value `Priority` rather than importing the `sos`
package: a delivery of dinner must not depend on the emergency-phone entity to know what "rød"
means. Three duplicated string constants is the cheaper wrong thing. When `sos` is lifted to
shared-go (task 055) a shared `types.Severity` becomes the obvious home for both, and that is
the moment to converge them — noted here so it is a decision rather than a discovery.

### The estimate, and why it stays crude

For a task **in a tour**, the expected time is the tour's planned time at its stop. That is the
good answer and it exists because a human planned the run — which is why the tour model earns
its complexity.

For a task **not yet in a tour**:

```
estimate = max(now, tidligst, next unit on duty) + allowance(kind)
```

with `allowance` a small configured table (pickup 30 min, transport 20, …). It ignores distance,
traffic and where the car is.

**One allowance for every vehicle.** A minibus and an estate do not drive alike, and we are
ignoring that on purpose: the difference between two cars is far smaller than the difference
between an accurate ETA and the guess we are actually making, so encoding it would add a column
and a maintenance burden while moving the number by less than its own error. If the desk ever
finds itself planning around one particular slow vehicle, that is the evidence to revisit — and
the measured gap between planned and visited stop times (§9) is where it would show up.

That crudeness is honesty about the inputs, not laziness. There are no positions, so a distance
term would be invented; and an estimate that looks precise gets quoted down a phone to a patrol
in the dark, who then stop making their own plans. So it is coarse, labelled *anslået*, and
always shown beside the fact that needs no model — how long they have already waited.

**Rejected alternative:** showing nothing until GPS exists. Rejected because the feature is
needed for this race, and "nothing" is what the desk has now.

### GPS, when it comes

The seam is a vehicle position, not a dispatch concern:
`NATHEJK.{year}.vehicle.{id}.position.reported` carrying lat/long/uts, projected onto the
existing `vehicle` read model as `lastLat/lastLng/lastSeenUts`. Nothing here reads it. When it
lands: a unit's last position and its age can be shown on the capacity strip, and the leg
allowance can gain a distance term. No change to `dispatch` is needed for that, which is the
point of keeping position off the task.

Note that **loks have no coordinates** (`lok` is `lokId, name, sortOrder, userIds, teamIds`), so
"how far to Lok 3" will not be answerable even with the tracker feed. Checkpoints do have
`latitude`/`longitude`. Worth knowing before promising a map.

### API endpoints

All new, all under `/api`, and **every one needs OpenAPI annotations** — a repo requirement,
called out here so it is not discovered at review:

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/dispatch` | the board: queued tasks, tours, units, duty |
| POST | `/api/dispatch/task` | create a task |
| GET | `/api/dispatch/task/:id` | one task with its timeline |
| PATCH | `/api/dispatch/task/:id` | edit fields (times, description, priority, places) |
| POST | `/api/dispatch/task/:id/pickedup` | people aboard → `transit` |
| POST | `/api/dispatch/task/:id/cancelled` | cancel, with reason |
| POST | `/api/dispatch/tour` | create a tour for a unit |
| PATCH | `/api/dispatch/tour/:id` | departure, unit, notes |
| PUT | `/api/dispatch/tour/:id/stops` | set the ordered stops and their tasks |
| POST | `/api/dispatch/tour/:id/underway` | tour has set off |
| POST | `/api/dispatch/tour/:id/stop/:stopId/visited` | stop reached |
| POST | `/api/dispatch/tour/:id/completed` | tour done |
| POST | `/api/dispatch/tour/:id/cancelled` | cancel, with reason |
| GET/PUT/DELETE | `/api/dispatchduty…` | duty windows per unit |
| PUT | `/api/section/:slug/dispatchable` | mark a section dispatchable |
| GET | `/api/sos/:id/dispatch` | a case's tasks |

`PUT …/stops` sets the whole ordered list in one call, rather than per-stop add/remove/move
endpoints: a reorder is one operator action and one intent, and three endpoints would make a
half-applied reorder possible. `/api/sections/sorted` already takes this shape.

Collections must serialise as `[]`, never `null` — this has bitten the repo three times (the
shelter sections, the note trail, and the klan dialog, where `orders: null` threw during render
and took the dialog's own close button with it, trapping the operator in a modal).

### Data / storage

- New tables: `dispatch_task`, `dispatch_tour`, `dispatch_stop`, `dispatch_activity` (the
  timeline), `dispatch_duty`, `dispatchable_section`.
- **Get the columns right before the first race.** `CREATE TABLE IF NOT EXISTS` never alters an
  existing table, so a column added later is silently absent from every database that already
  has the table. PRD 007 §8 made this argument while its table was still free to change; the
  same window is open here and closes the first night this runs.
- Places are stored as **type + reference + label** (`kind`, `refId`, `label`), not a foreign
  key: a place may be free text, and a checkpoint's name should be preserved on the task even
  if the checkpoint is later renamed.
- A task's link to a tour lives on the **stop**, not on the task: a task can occupy two stops
  (load and unload), so a `tourId` column on the task would be a lie half the time. The task's
  state is derived from its stops.

### Dependencies & risks

- *The board is only as good as the desk's discipline.* If cars are dispatched by phone without
  being written down, the wait times are fiction. Same class of risk as PRD 007's `transit`
  gap, and the same mitigation: make the written path the fastest path — one click from a case,
  and a create dialog where almost everything is optional.
- *Two desks on one task.* Logistics plans while the nødtelefon edits the same task. Last write
  wins, both on the timeline, no locking — but the board must defer incoming payloads while a
  tour is being rearranged, or an operator loses a plan mid-drag.
- *Duty windows will not be maintained.* If the roster goes stale, "next unit on duty" is
  wrong. Mitigation: the strip shows how stale it is, and the estimate degrades to
  `now + allowance` rather than to nonsense. Also why planning outside a window warns rather
  than refuses.
- *Units drift from the organisation.* A car moved to another subsection on the Organisation
  page silently changes who owns a tour. Tours record the section slug at planning time and the
  board flags a mismatch rather than rewriting history.
- *Scope creep toward a logistics system.* Inventory, fuel, expenses and driver hours all
  suggest themselves. §4 refuses them; a race-night desk is the product.

## 9. Success Metrics

- **The custody gap closes.** Share of `transit` transitions published by this interface rather
  than by the nødtelefon's manual override — target: the override becomes the exception, not
  the mechanism. Measurable from the timeline's actor, and the clearest signal the feature works.
- **Every dispatched job is recorded.** After the race, tasks created versus the desk's own
  account of what the cars did. Target: no "we also sent a car to…" the board never saw.
- **The desk can answer "when".** Share of pickup tasks that are in a tour — and therefore
  carry a planned time — within 15 minutes of being created.
- **Plans are kept, or visibly broken.** Median difference between a stop's planned time and
  when it was marked visited. The target is not zero; it is *known*, so the desk learns what
  its allowances should be.
- **Deadline tasks land on time.** Share of `delivery` tasks completed before `skal leveres`.
  Dinner at 19:00 is the worked example.
- **Nobody waits unseen.** Longest wait for an unplanned pickup during the race, reviewed after.

## 10. Rollout / Task Breakdown

Phased so each phase is shippable and the risky part is not last. **Phase 1 alone is worth
running a race on** — a written board with tours beats a phone and a memory even with no
estimates and no seams.

**Phase 1 — tasks and tours.** The entity, `dispatchable` sections, create a task, build a
tour, order its stops, mark stops visited, complete. Waiting clocks and planned times. No duty
windows, no estimates, no member transitions.

**Phase 2 — capacity and wait time.** Duty windows per unit, the capacity strip, the queued
estimate, deadline warnings and the at-risk filter.

**Phase 3 — the seams.** `Bestil kørsel` from a case; `AcceptPickup` publishing `transit`; the
case showing its tasks and their times. This closes PRD 007's risk, and is third only because
it needs Phase 1's state machine to hang off.

**Phase 4 — positions.** When the tracker feed is available: the vehicle position event, last
known position on the capacity strip, a distance term in the leg allowance.

Tasks created in `roadmap/tasks/` on approval (2026-08-27):

- [ ] 108 — `dispatchable` flag on sections — table, endpoint, Organisation page toggle
- [ ] 109 — `dispatch` entity — tasks, tours, stops, timeline, projection, live wiring
- [ ] 110 — dispatch API — board, task create/edit/cancel (with OpenAPI annotations)
- [ ] 111 — tour API — create, stops (whole-list PUT), underway, stop visited, complete
- [ ] 112 — extract `composables/severity.ts` from `sos.ts` and share the grøn/gul/rød vocabulary
- [ ] 113 — `DispatchView` — queue + tour panes, live + deferred while rearranging
- [ ] 114 — task dialog with the place picker (checkpoint / lok / HQ / free text)
- [ ] 115 — `dispatchduty` entity + duty window editor per unit
- [ ] 116 — capacity strip, unit readiness, and the queued wait-time estimate
- [ ] 117 — deadline warnings and the at-risk filter
- [ ] 118 — `AcceptPickup` on `spejderstatus.Commands` → `transit`
- [ ] 119 — `Bestil kørsel` from an SOS case, and the case's task list
- [ ] 120 — vehicle position seam (Phase 4, blocked on tracker access)

## 11. Open Questions

Answered by the product owner 2026-08-27:

1. ~~Who runs the board?~~ **The logistics section**, not the nødtelefon operators. Hence a
   screen of its own (§7), and two different desks writing the same tasks (§6 non-functional).
2. ~~Is a job assigned to a car, a driver, or both?~~ **Neither: to a dispatchable subsection**
   holding a vehicle, a driver and possibly a co-driver. The organisation tree already
   expresses this; only the `dispatchable` flag is new (§8).
3. ~~Where do duty windows come from?~~ **Agreed in advance with the logistics crew** — a
   planned roster, recorded per unit.
4. ~~How is a batch represented?~~ **Tours.** A car is assigned tours; a tour runs A to B with
   several stops; tasks are put into tours. This is now the backbone of the design and of the
   answer to "when".
5. ~~Does the driver get a screen?~~ **Yes, but not here.** It is another app in another repo,
   so it is a non-goal (§4) with an API seam (§8). In this repo the desk ticks the stops off.
   The identity problem is ruled out of scope rather than solved: under ten cars, all drivers
   known by name and number, so there is no security concern to design around.
6. ~~Are planned stop times entered or derived?~~ **Derived, with any stop overridable.** Fast
   to plan, and correctable the moment the desk knows better.
7. ~~Naming.~~ **English in source, Danish in the interface.** `dispatch`, `tour`, `task`,
   `stop` in code; "Kørsel" on screen. Route stays `/koersel`, matching `/poster` and
   `/hoensegaard`.
8. ~~Seats versus headcount.~~ **Warn.** Counted across a tour's unvisited pickups, and a
   warning rather than a refusal — seats fold down and the desk knows things we do not.
9. ~~Priority scheme.~~ **Mirror the SOS vocabulary**, grøn / gul / rød. Shared through a
   neutral `composables/severity.ts` on the front end rather than copied; a separate
   three-value type in Go until `sos` is lifted to shared-go (§8).
10. ~~Where do the per-leg allowances live?~~ **One constant for every vehicle.** Per-vehicle
    speed is deliberately ignored: the difference between two cars is smaller than the error in
    the estimate itself (§8).

Still open:

11. **Can a tour serve two units?** Assumed no — one tour, one unit. A car handing over mid-tour
    would be two tours.
12. **Is a co-driver explicit?** Proposed: no — the unit's people are whoever is in the
    subsection, and the vehicle's `driverUserId` names the driver. If who drove matters after
    the fact, it needs recording per tour.
13. **Recurring deliveries.** Dinner happens every day at the same time. Templates, or fresh
    tasks each evening? Phase 1 assumes fresh.
14. **Should a pickup's drop-off ever be somewhere other than HQ?** A scout handed back to
    their own leaders at a lok, for instance.
15. **What closes a delivery — arrival, or handover to a named person?** The shelter models
    handover explicitly; dinner probably does not need it.
