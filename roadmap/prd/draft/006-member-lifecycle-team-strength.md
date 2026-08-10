# PRD 006 — Member lifecycle, team strength & discontinuation

**Status:** draft
**Author:** agent session
**Created:** 2026-08-10
**Last updated:** 2026-08-10
**Approved:**
**Shipped:**
**Status note:** split out of PRD 001 on 2026-08-10; member lifecycle settled
against shared-go `v0.0.0-20260807180020-5ac2603c60ba`
**Target users:** organizer (HQ emergency-phone operators / nødtelefonvagter)

---

## 1. Summary

Make **where every participant is** a first-class, live fact in the platform, and
give the nødtelefon operator the actions that belong to them.

Concretely: revive the `spejderstatus` projection as the canonical member
status/membership read model; derive `racing` from the existing team-started event;
let an operator record that a member **wants to leave the race** (`waiting`) or has
decided to **carry on** (`waiting` → `racing`); show how many members are **in our
care** and warn when one has been waiting too long; count a patrol's **strength**
and surface a breach of the **3-racing-member requirement** with the means to handle
it (collect the team, reassign the survivors, or grant an exception); and derive
**team discontinuation** from membership, replacing the legacy `patruljemerged`
merge/split encoding.

The operator-facing surface for all of this is the SOS case detail view built by
**PRD 001**, which is therefore a prerequisite. The counters live on the nødtelefon
list view (or the dashboard — see Open Questions).

> **Terminology:** what `shared-go/types/member.go` calls *trailside assistance* is
> the nødtelefon interface. The lifecycle documented there — a member leaving the
> route, being collected, sheltered at HQ and handed over — is the workflow this PRD
> builds the *first step and the overview* for, and that documentation is the
> canonical description of the member half of the feature. Read the two together.

## 2. Problem & Motivation

- **What problem does this solve?** Nothing in the current platform records where a
  participant is. `spejderstatus` exists but is **inert**: `Consumes()` returns an
  empty subject list and the whole of `HandleMessage` is commented out
  (`go/nathejk/table/spejderstatus.go:39-70`). So when a scout leaves the route
  there is nowhere to write it, no way to count how many people the organisers are
  responsible for, and no way to know whether a patrol still has enough members to
  continue. The number that must reach zero before anyone goes home does not exist.
- **A second, quieter breakage:** team discontinuation is silently broken.
  `patrulje.querier.GetDiscontinuedTeamIDs` (`go/nathejk/table/patrulje/query.go:121`)
  **returns an empty slice**, and it feeds the checkgroup/checkpoint status view via
  `go/internal/data/models.go:166`. Discontinued teams therefore show as "not
  arrived" forever. The legacy mechanism behind it (`patruljemerged`, written by
  `patrulje.merged` / deleted by `patrulje.splited`) is not being produced either.
- **Why now?** PRD 001 gives operators a dispatch log, which immediately raises "and
  what happened to the scout?". The member lifecycle is also the seam the future car
  and shelter interfaces will build against, so fixing it once — properly, in one
  shared projection — avoids each of those products inventing its own.
- **Evidence.**
  - `shared-go/types/member.go` now defines and documents the full lifecycle:
    states, transitions, terminal states, the `Valid()` / `CanFinish()` /
    `InOurCare()` helpers, and the mapping from superseded values. Nothing in
    shared-go references the new constants yet, so this feature is its first
    consumer and its proving ground.
  - Legacy encoded the same needs crudely: `_go/nathejk/table/sos.go`, the
    `member.status.changed` events in the legacy message packages, and
    `patruljemerged`.
  - `MinMemberCount: 3` is currently hardcoded in a handler
    (`go/cmd/api/patrulje.go:99`), which shows the rule exists in the product but
    has no home.

## 3. Goals

- One canonical, live read model of **member status and team membership** for the
  event, written by whoever legitimately knows: the platform for `racing`, this
  interface for the self-carrying transitions, and later the car and shelter
  interfaces for the rest.
- Operators own the transitions where the member is still **self-carrying** — on
  their own feet, whether walking or stopped by the trailside:
  - `racing` → `waiting`: the member wants to leave the race and awaits collection.
  - `waiting` → `racing`: they changed their mind and carry on under their own
    steam. Legitimate and expected, not a correction.
- Operators can **see** the rest of the chain for every member on an associated
  patrol (current status, when it changed, who accepted them) without maintaining
  it, because each later step is a **handover accepted by the receiving party**:
  `waiting` → `transit` (the car accepts them aboard — the point of no return),
  `transit` → `sheltered`, `sheltered` → `released` | `reunited`. Custody is always
  *confirmed by the receiver*, never claimed by the party letting go, which is what
  makes the chain trustworthy. Plus an **override** to correct out-of-sync data.
- Operators can see at a glance how many members are **in our care**
  (`waiting`/`transit`/`sheltered`) — the count that must reach zero before the
  organisers can go home — and which members have been `waiting` too long, since a
  waiting member blocks their whole patrol.
- **A patrol is required to have at least 3 racing members.** This is a rule of the
  event, not a preference — but the *handling* of a breach belongs to the operator
  on the nødtelefon, who is on the phone with the people involved. The interface
  therefore makes the breach unmissable and gives the operator the means to deal
  with it, without blocking the transition or acting on its own:
  1. The member carries on (`waiting` → `racing`) and the requirement is met again.
  2. **Collect the whole team** — the remaining members go `waiting` too and leave
     together.
  3. **Reassign the remaining members** to another patrol that can take them, if
     one is available.
  4. **Grant an exception** — the operator judges this particular patrol can
     continue short-handed. Permitted, but it is a deliberate departure from the
     rule and is recorded as one, not a silent dismissal.
  Options 2 and 3 both leave the original team with no active members, so it becomes
  discontinued — which is precisely what the legacy `merged`/`splited` events were
  encoding, and why the old `patruljemerged` table pointed at a `parentTeamId`:
  option 3 *is* a merge, expressed as member reassignment.
- **A team with no active members is discontinued** (udgået), derived from
  membership and reversible — and observable through the existing
  `discontinuedTeamIds` surface so the checkpoint overview works again.
- Every member action appears on the timeline of the SOS case it was taken from, so
  a shift handover reads as one story.

## 4. Non-Goals

- **The car/driver interface and the shelter reception interface.** They will be
  specified in their own PRDs, **later**, and are deliberately not designed here.
  This PRD does not define their events' payloads, their screens, or their
  workflow — only that the transitions from the car door onwards are theirs, and
  that they write the same projection with per-member events resolving to a
  `MemberStatus`. §8 records the minimum seam so those PRDs can be written against
  it without renegotiating this one.
- **Dispatching cars.** Deciding which car goes where, and telling it to, is not
  part of this interface.
- SOS case management itself — cases, comments, severity, assignee, team
  association — which is **PRD 001** and a prerequisite here.
- Merge/split of patrols as a concept. The terms *merged* and *splitted* are retired
  from the domain vocabulary and the legacy `patrulje.merged` / `patrulje.splited`
  events are **not** ported. They were an encoding of what §6 models directly: a
  member is **reassigned** to another team, and a team with nobody racing is
  **discontinued**. The replacement must reproduce the observable behaviour of the
  old events, including their reversibility (legacy `.splited` deleted the
  `patruljemerged` row and thereby un-discontinued the team).
- Migrating historical `patruljemerged` data (see Open Questions).
- Position-request SMS / GPS location of members, and real-time map tracking.
- Klan (senior) participation in SOS cases. PRD 001 scopes case association to
  patrols; whether the projection covers seniors at all is an Open Question.

## 5. User Stories & Scenarios

- As an **operator**, I want to record that a member wants to leave the race and is
  waiting to be collected (`waiting`), so that a car can be sent and the member is
  counted as being in our care from that moment.
- As an **operator**, I want to put a member who has changed their mind back into
  the race (`waiting` → `racing`) as long as no car has collected them yet, so a
  scout who gets their breath back can carry on and no car is sent needlessly.
- As an **operator**, I want to be told immediately when the member leaving would
  put their patrol below the required 3 racing members, and to have the means to
  handle it — collect them all, place the survivors with another patrol, or grant an
  exception — because I am the one on the phone and the rule needs a person to apply
  it.
- As an **operator**, I want my handling recorded, including an exception I grant,
  so the next shift and the post-event review see how the breach was dealt with.
- As an **operator**, I want to watch the member's progress — accepted into a car,
  accepted at the shelter, handed over — without having to update it myself, because
  the people receiving the member record it as it happens.
- As an **operator**, I want to see how many members are in our care right now, and
  be warned when somebody has been `waiting` too long — their patrol cannot continue
  until they are collected.
- As an **operator**, I want to **override** a member's status when it is wrong, to
  correct out-of-sync data.
- As an **operator**, I want to reassign a member from one team to another so that,
  e.g., a scout who continues with a different patrol is tracked correctly; when a
  team is left with nobody racing it is automatically considered **discontinued**.
- As an **organizer at the checkpoints**, I want discontinued teams to stop showing
  as "not arrived", so the checkgroup overview means something again.

### Primary happy path

Continuing PRD 001's scenario, where a case exists and the caller's patrol is
associated with it:

1. The operator marks the injured scout as **`waiting`**. The patrol is now blocked
   until a car reaches them, and the scout counts as in our care from this moment.
2. The patrol had four racing members, so no breach — the card simply shows strength
   3 and no warning.
3. The operator does nothing further to the status: when a car reaches them the
   driver accepts them aboard (**`transit`**), and on arrival the shelter accepts
   them (**`sheltered`**) — both appear on the case timeline as they happen.
4. A parent collects them that night: **`released`**. (Had their patrol reached the
   finish first, it would have been **`reunited`** instead — never `finished`, which
   is reserved for walking the route.)
5. The **I vores varetægt** counter went from 0 to 1 and back to 0 without anybody
   maintaining a list.

### Edge cases & error scenarios

- A member in `waiting` who decides to carry on returns to `racing`; the timeline
  records both moves. This is valid only while they are still self-carrying — if a
  car has already accepted them (`transit`), the resume must be **rejected**, not
  silently applied. Race condition to handle explicitly: the operator presses resume
  at the same moment the driver accepts the member aboard — **the acceptance wins**,
  since it reflects the member physically being in a car, and the event log preserves
  both attempts in order.
- Retiring the last racing member of a team discontinues it; the case timeline
  records both the member change and the resulting team discontinuation.
- Reassigning a member *back* to a discontinued team makes it non-discontinued again
  (parity with the legacy `.splited` undo). Discontinuation must therefore be
  re-evaluated on every membership/status change, not set once.
- A team already below the requirement when a case is opened (two members left
  earlier in the event) is in breach from the start; the warning reflects the team's
  current strength, not only the transition that caused it.
- Reassigning the survivors when **no** target patrol is available leaves collecting
  them or granting an exception. The UI must not offer a reassignment flow that
  dead-ends with no candidates.
- An exception is granted per team, per breach — if strength changes again later
  (another member leaves), that is a new breach needing its own handling. An
  exception is not a permanent waiver.
- If the member resumes after the operator has already collected the rest of the
  team, the team is *not* automatically restored — those members are `waiting` and
  only they (or a car) can change that. Resolving one member does not unwind actions
  taken for others.
- A member who left the trail must never end up `finished`. `MemberStatus.CanFinish()`
  is true only for `racing`, so the finish-line flow cannot promote a `reunited`
  member — and the UI must not offer `finished` as an override target either. Note a
  member who resumed (`waiting` → `racing`) *can* finish, correctly: they walked the
  rest of the route themselves.
- A patrol that **finished** also has nobody `racing`. The naive discontinued
  predicate would mark every finishing team as discontinued — see Open Questions for
  the exact predicate, and note this is the single highest-risk detail in the PRD
  because it is wrong in a way that looks plausible.
- Replay encounters **legacy status values** (`active`, `emergency`, `hq`, `out`,
  `REGISTERED`, `STARTED`); un-normalised, every `InOurCare()` / `CanFinish()` check
  silently under-reports.
- `StartPatrulje` deletes non-starters; a projection that ignores those deletions
  holds rows for no-shows and over-counts team strength.
- Everything is scoped to the current event year.

## 6. Requirements

### Functional

**Member status projection**

- [ ] Revive `spejderstatus` as the canonical member status/membership projection,
      keyed by `MemberID`, holding `initialTeamId`, `currentTeamId`, `status`
      (`types.MemberStatus`) and `updatedAt`, year-scoped.
- [ ] Derive `racing` from the existing team-started event: for each member in
      `messages.NathejkTeamStarted` on `NATHEJK.{year}.patrulje.{teamId}.started`,
      write `status = racing`, `initialTeamId = currentTeamId = teamId`.
- [ ] Handle `spejder.{memberId}.deleted`, which `StartPatrulje` publishes for every
      member who did **not** start.
- [ ] Normalise superseded status values on the way in, per the mapping documented
      in `shared-go/types/member.go`.
- [ ] Take the year from the **subject**, not `msg.Time()`.

**Operator actions (self-carrying only)**

- [ ] Mark a member as **`waiting`** — wanting to leave the race and awaiting
      collection. Recorded on the SOS case timeline; `sosId` required.
- [ ] Return a `waiting` member to **`racing`** when they choose to carry on.
      Permitted **only** from `waiting`, enforced on the write side, with a message
      the operator can act on ("allerede hentet") rather than a generic conflict.
- [ ] Override a member's status to any valid value to correct out-of-sync data,
      **excluding `finished`**. Visibly distinct from the normal `waiting` action so
      it is not used as a shortcut for work another interface owns.
- [ ] Reassign a member to another team: updates `currentTeamId`, leaves
      `initialTeamId` untouched, recorded on the case timeline.

**Read-only chain**

- [ ] Display the rest of the chain read-only: current status, when it changed, and
      who accepted the member. These arrive as events from other interfaces; this UI
      must never be the thing that sets them in normal operation.
- [ ] Show externally-produced member transitions **on the case timeline** of any
      case the member's team is associated with, so an operator watching a case sees
      the pickup and arrival without switching screens. How those events are
      correlated to a case is an Open Question.
- [ ] Tolerate transitions this interface did not cause, arriving in any order,
      including for members on teams never associated with a case.

**Counters and alarms**

- [ ] Show a live **in our care** count (`MemberStatus.InOurCare()` — `waiting`,
      `transit`, `sheltered`) across the current year, permanently on screen rather
      than per-case. This is the number that must reach zero before the organisers
      can go home.
- [ ] Flag members who have been `waiting` beyond a threshold. A waiting member
      blocks their entire patrol, so this is the one state worth an alarm; the
      threshold is a config value (see Open Questions).

**Team strength and the 3-member requirement**

- [ ] Team strength is the count of members whose `currentTeamId` is the team and
      whose status is `racing` — matching `MemberStatusRacing`'s documentation that
      it is "the only state in which a member counts towards their team's strength on
      the route". `waiting` members do **not** count.
- [ ] **Warn, at the moment of setting `waiting`, when it would put the team below
      the required 3 racing members**, naming the resulting count, *before*
      committing — it changes the conversation the operator is having on the phone.
      The warning does not block the transition; the member is leaving whether or not
      the team is compliant.
- [ ] Show which teams on a case are below the requirement, and keep showing it
      while it is true, so an operator taking over a shift can see the state of play.
      A team in breach with no recorded handling is the one state this tool must
      never hide.
- [ ] **Collect the whole team** as one action: every remaining `racing` member goes
      to `waiting` together, recorded as a single timeline entry rather than one per
      member. Operators are on the phone; three separate clicks invite two of them
      being forgotten.
- [ ] **Reassign the remaining members** to another patrol, with candidate targets
      offered by the backend (rules are an Open Question). This is the modern form of
      the legacy patrol merge.
- [ ] **Grant an exception** allowing a patrol to continue below 3, recorded with the
      acting operator and a short reason. This is the pressure valve that keeps the
      requirement honest rather than routinely ignored: it must be easy to do and
      impossible to do accidentally.
- [ ] A handled breach is **distinguishable from an unhandled one**, so the UI can
      settle the warning and the post-event review can tell them apart.
- [ ] The required minimum (3) is a **configured value**, not a literal in code —
      note it is currently hardcoded at `go/cmd/api/patrulje.go:99`. Where it lives
      is an Open Question.

**Discontinuation**

- [ ] A team with **no active (`racing`) members** is **discontinued** (udgået),
      excluding teams that finished. Re-derived on every membership or status change
      and **reversible**.
- [ ] Discontinuation is observable through the existing `discontinuedTeamIds`
      surface (`data.TeamModel.GetDiscontinuedTeamIDs` /
      `patrulje.querier.GetDiscontinuedTeamIDs`, consumed by the checkgroup status
      view), so the checkpoint overview keeps working exactly as it did when it was
      fed by `patruljemerged`.
- [ ] The `patruljemerged` projection and its consumer are **deleted** once
      membership is the source of truth, and the duplicate query paths are
      consolidated.

**Cross-cutting**

- [ ] Member statuses and their Danish labels are served **from the backend**, never
      hardcoded in the view.
- [ ] Capture the acting user on every event this interface publishes, resolved from
      the request context and passed to the command by the handler — empty in practice
      until the planned auth service lands, exactly as in PRD 001 §6 (Auth).
- [ ] All queries and events are scoped to the current event year.

### Non-Functional

- **Consistency with platform:** REST + JSON via `app.*` helpers; MySQL projections
  rebuilt from JetStream on startup; frontend via the `http` module and PrimeVue
  Aura.
- **Timeliness / freshness — load-bearing here in a way it is not for PRD 001.**
  The in-our-care counter and the `waiting` alarm are the reason live updates
  matter: a count of the people we are responsible for that is quietly a minute out
  of date is worse than no count. Adoption uses PRD 004's shipped `useLiveResource`
  and SSE stream from the first commit.
- **Correctness over availability for the counters.** A plausible-but-wrong count of
  members in our care is the worst failure this tool could have. This is where the
  **boot gate (PRD 005)** stops being cosmetic: until it ships, a post-restart
  window can serve a partially rebuilt read model, and the honest interim behaviour
  is to say the screen cannot reach the server rather than to render a number.
  Treat PRD 005 as a dependency of this PRD, not of PRD 001.
- **Auditability:** every transition, every breach handling and every override is
  attributable to a **time**, and to a person once per-user identity exists (today
  authentication is perimeter-only — basic auth on stage/production, none in dev — so
  the actor is recorded but empty; see PRD 001 §6). This matters more here than for a
  textual case log: "who granted this patrol an exception?" is the question an
  incident review asks, so **an exception's reason text is doing the work the missing
  identity cannot** and should be required rather than optional.
- **Localization:** Danish UI text and `da-DK` date formatting.

## 7. UX / UI Notes

All of this lands **inside PRD 001's surfaces**; this PRD adds no new views.

- **`SosView.vue` → Tilknyttede patruljer card**, extended:
  - The team's **strength** (racing members) beside its name, an **Under styrke**
    warning when below the required 3, and an **Udgået** badge when discontinued.
  - Each member row shows the current status with its timestamp and, where known,
    who accepted them. Row actions depend on status: `racing` offers **Ønsker at
    udgå** (→ `waiting`); `waiting` offers **Fortsætter selv** (→ `racing`) as a
    normal, prominent action — not buried in an override menu, since a scout getting
    their breath back is an ordinary outcome and saves a car being sent. **From
    `transit` onwards the row is read-only**: it reflects what the car and shelter
    have recorded and offers no buttons to advance or reverse them.
  - Secondary actions: a visibly-separate status override, and **Flyt til anden
    patrulje** (reassign). Use PrimeVue overlay/popover for these menus, not
    `b-popover`.
  - Members `waiting` past the threshold are highlighted.
- **Below-strength panel** on the same card: when a team on the case has fewer than
  3 racing members, a prominent warning stating the current strength and offering
  the three ways to handle it — **Hent hele patruljen** (all remaining racing
  members → `waiting`, one action), **Flyt de resterende** (reassign survivors, with
  backend-supplied candidate patrols) and **Tillad undtagelse** (grant an exception,
  requiring a short reason). Confirming `Ønsker at udgå` for a member whose departure
  causes the breach warns *before* committing, naming the resulting strength
  ("Patruljen har kun 2 tilbage"), and offers the same three actions plus proceeding
  and handling it next. Once handled, the warning becomes a settled note recording
  what was done and by whom — it stops demanding attention but does not disappear.
- **`SosListView.vue` header:** a permanent **I vores varetægt** counter
  (`InOurCare()`: waiting + transit + sheltered) with a breakdown per status, and a
  warning state when any member has been `waiting` past the threshold. This is the
  organisers' go-home number, so it should be visible without opening a case.
- **New timeline entry types** in `SosActivityLine.vue`: memberstatus (per
  transition), member-reassign, team-collected, exception-granted,
  team-discontinued. PRD 001 requires the component to tolerate unknown types, so
  this is additive.
- **Drag-free, dirty-guard-aware:** the reassignment and exception dialogs hold
  unsaved state, so they must defer incoming payloads while open, as
  `KlanListView.vue` does, and say updates are paused.

## 8. Technical Considerations

### BFF (Go) — where the code lives

Same rule as PRD 001: build locally in `go/nathejk/table/spejderstatus/` (promoting
the current single file to a package) written to **shared-go's guidelines** so it can
be lifted to `shared-go/tables/spejderstatus/` unchanged — `table.go`, `consumer.go`,
`querier.go`, `commands.go`, `repository.go`, `interfaces.go`, `table.sql`, following
`tables/signup` as the reference, with **no imports from `nathejk.dk/...`**.
Handlers stay local permanently. `shared-go/types/member.go` explicitly names this
projection as the home of member status ("these strings live in the spejderstatus
projection"), which settles the question of whether to fold it into `spejder`: keep
the projection, keep the name.

### Member status projection

- The struct at `go/nathejk/table/spejderstatus.go:13-18` already declares the right
  shape (`MemberID`, `InitialTeamID`, `CurrentTeamID`, `Status types.MemberStatus`)
  but the projection is inert — `Consumes()` returns an empty list and
  `HandleMessage` is entirely commented out.
- **Schema needs extending:** `spejderstatus.sql` is currently only
  `id, year, status, updatedAt`. The `initialTeamId` / `currentTeamId` columns do not
  exist, and membership queries need an index on `(year, currentTeamId)`. Note
  `CREATE TABLE IF NOT EXISTS` never alters an existing table, so in dev the columns
  will not appear until the table is dropped — a known trap (task 038 covers the
  same class of drift for signup).
- **`racing` is derived from the team-started event, not set by this interface.**
  `messages.NathejkTeamStarted` (published by `commands.Team.StartPatrulje`,
  `go/nathejk/commands/team.go:125-148`, behind `PUT /api/patrulje/:id/start`)
  carries `Members []NathejkTeamStarted_Member` — precisely the members who actually
  started. This closes the last gap in the member model with no new event and no new
  producer. Details that matter:
  - **Non-starters are deleted, not left behind:** `StartPatrulje` publishes
    `spejder.{memberId}.deleted` for every member who did not start.
  - **Take the year from the subject**, not `msg.Time().Year()` — the body has no
    year field and the old commented-out code got this wrong, which breaks on replay
    across year boundaries.
  - **Strength at start is `len(body.Members)`**, which the `patrulje` consumer
    already uses for `memberCount` (`table/patrulje/consumer.go:66`). Reusing the
    same source keeps the 3-member check consistent with the patrol's own member
    count rather than inventing a second definition.
  - `registered` and `seated` are irrelevant here — the SOS panel only ever sees
    members who have started.
- **Legacy status values must be normalised on replay**, since the projection is
  rebuilt from full JetStream history: `REGISTERED`/`STARTED` →
  `registered`/`racing`, `active` → `racing`, `emergency` → `waiting`, `hq` →
  `sheltered`, `out` → `released` (`waiting`/`transit` unchanged). This mapping
  belongs next to the constants in shared-go so all consumers share one
  implementation — it is documented in `types/member.go` but not implemented.
- **Reuse the lifecycle helpers rather than re-deriving them.** `Valid()` gates what
  the API accepts, `CanFinish()` guards the finish-line flow, `InOurCare()` *is* the
  in-our-care count. No hand-rolled status lists in handlers, queries or the SPA.

### Events

- `messages.NathejkMemberStatusChanged` does **not** exist in shared-go. The
  reference at `go/nathejk/table/spejderstatus.go:49` is inside the commented-out
  block, and the type lives in the *legacy* local message packages (`_go`, and copies
  in `hjælper`/`scan-app`). It must be designed and added to
  `shared-go/messages/member.go`.
- **A single generic "status changed" event fits this model poorly.** Each transition
  is a **distinct act by a distinct party** — a request to leave, a decision to carry
  on, an acceptance into a car, an acceptance at the shelter, a handover — so model
  them as separate events (e.g. `member.withdrawal.requested`,
  `member.withdrawal.cancelled`, `member.pickup.accepted`,
  `member.shelter.accepted`, `member.handover.completed`) that each carry the acting
  party and resolve to a `MemberStatus`. This makes the acceptor recordable, which a
  bare `{memberId, status}` payload cannot express — and it answers "who holds this
  member?" for free, because the car's acceptance event names the car.
- **Subject entity: the member, not the case.** Proposal:
  `NATHEJK.{year}.spejder.{memberId}.withdrawal.requested` and so on, which means
  member changes emit the existing **`spejder`** live token. This must be decided
  before the frontend is written, because the token *is* the frontend contract (see
  below) — and it interacts with the seniors question in §11.
- This interface publishes the withdrawal request, its cancellation, the override
  and `member.team.reassigned`, plus whatever represents breach handling. It
  **consumes** the rest.
- **Whole-team collection is one command, not N.** A single
  `sos.team.collected`-style command publishes a withdrawal request per remaining
  racing member, atomically from the operator's point of view, sharing one
  `correlationId` so the timeline can render it as one entry ("hele patruljen
  hentes") while `spejderstatus` still sees per-member events. Publishing three
  independent requests from the frontend would risk a partial collection if one call
  fails — the worst possible outcome, since the team would then be split across
  states with nobody noticing.

### The self-carrying boundary is enforced on the write side

`member.withdrawal.cancelled` is valid only while the member is `waiting`; the
command must dirty-check the current `spejderstatus` row and reject it otherwise.
Hiding the button once the status advances is not sufficient — the operator's screen
may be a moment stale, which is exactly when the car is accepting the member. If the
acceptance and the cancellation race, the **acceptance wins**: it reflects a member
physically sitting in a car, and the event log preserves both attempts in order.

### The requirement is enforced by a person, not by a command

No command may reject a withdrawal request because it would put a team below 3, and
no consumer may auto-collect or auto-reassign in response. The member is leaving
regardless — refusing to record that would only make the data wrong — and the remedy
depends on things the projection cannot know: how far along the patrol is, how
capable they are, whether a car is anywhere near. So the write side reports strength
and records what the operator did; it never decides. The requirement is expressed as
an unmissable breach plus a recorded handling, which is what makes it auditable
afterwards.

**Model the handling explicitly.** A breach that has been dealt with must be
distinguishable from one that has not, otherwise the UI cannot settle the warning and
the review cannot tell the two apart. Whether that is a first-class event or a
case-scoped record is an Open Question, but it needs to exist — "we notice breaches
but do not record their resolution" is the failure mode to avoid.

### Queries

- **Team strength:**
  `COUNT(*) FROM spejderstatus WHERE year = ? AND currentTeamId = ? AND status = 'racing'`,
  answerable directly given the `(year, currentTeamId)` index. Two derived facts come
  from the same query and should share one implementation rather than being
  re-counted per caller: in breach (`< minimum`) and discontinued (`= 0`).
- **In our care:** count by status over `InOurCare()`, plus the oldest `waiting`
  timestamp for the alarm.
- **Reassignment candidates** need a query — "another team if such is available"
  requires the backend to offer targets. Rules are an Open Question, but the shape
  is: patrols in the same year, still racing, near enough to physically join. This
  is the query the legacy merge never had, because an operator picked the parent team
  by hand.
- **Discontinued teams replace `patruljemerged`.** Legacy encoded "no members left"
  as a `patruljemerged` row (`teamId → parentTeamId`), inserted on `.merged` and
  deleted on `.splited`, and derived discontinued teams from it at
  `go/internal/data/team.go:60`. Under the new model a team is discontinued when no
  member with `currentTeamId = team` is `racing`. **Careful:** a team that reached
  the finish also has nobody `racing`, so the naive predicate would report every
  finishing team as discontinued. Two viable shapes:
  1. **Derive on read** — query `spejderstatus` directly in
     `GetDiscontinuedTeamIDs`. Simplest, inherently reversible, no new event.
  2. **Explicit event** — a consumer watches membership and publishes
     `patrulje.discontinued` / a matching un-discontinue event, projected onto
     `patruljestatus`. Puts the fact on the log but needs both directions to stay
     reversible.
  Option 1 is the proposal; the decision is an Open Question.
- **Three overlapping query paths** could host `GetDiscontinuedTeamIDs`:
  `go/internal/data/team.go:60` (legacy `patruljemerged` SQL),
  `go/nathejk/table/patrulje/query.go:121` (the live stub) and
  `shared-go/tables/patrulje` (no such query yet). `internal/data/models.go` still
  imports `patrulje` from the **local** package while taking neighbouring entities
  from shared-go. Per the placement rule the implementation goes in the local
  `table/patrulje` and travels with that entity when it is lifted; the legacy
  `internal/data` path should be retired rather than kept in parallel. **Confirm
  which path is live at runtime before implementing**, otherwise the fix lands in
  the unused one.

### Correlating externally-produced transitions onto a case timeline

The car and shelter interfaces will not know a `sosId`, so the timeline cannot rely
on one for events this interface does not publish. Options: propagate the
`correlationId` from the originating `waiting` event through the chain (clean, but
requires every downstream producer to cooperate), or resolve at read time by matching
the member to open cases associated with their team (no cross-repo coordination, but
ambiguous when a member has two open cases). Decide before the timeline projection is
extended — see Open Questions. This is the one part of the seam the later PRDs may
need to influence.

### The seam this PRD fixes for the car and shelter PRDs

So those PRDs can be written independently and are not designed here, this feature
fixes exactly the following and nothing more:

- `spejderstatus` is **the** shared member status/membership projection. Downstream
  interfaces write the same table via their own events; they do not get their own
  copy.
- Status values are `shared-go/types.MemberStatus`, and the transition each interface
  owns is fixed by the *self-carrying* boundary: ours up to and including `waiting`,
  theirs from the car door on.
- Each downstream transition is an **acceptance by the receiver**, so its event
  carries the accepting party. This PRD does not define those payloads, only that
  they exist, are per-member, and resolve to a `MemberStatus`.
- This interface **consumes** them for the timeline and the counters, so it tolerates
  transitions it did not cause, in any order, including for members on teams it never
  associated with a case.
- Until they ship, the **override** is the interim path: it is how an operator records
  a pickup or an arrival by hand. Its cost is measured (§9 tracks override
  frequency), and the in-our-care counter and `waiting` alarm should be read as
  provisional until then — do not tune the alarm threshold against interim manual
  bookkeeping.

### Live updates

PRD 004 shipped; adoption is a few lines and belongs in the first commit of each
view, not a later phase. Specifics:

- Signals derive from the **event subject**, so the token for member changes is the
  subject entity decided above (proposal: `spejder`) — **not** `spejderstatus`, which
  is a projection name and can never appear in a signal.
- `dependsOn` sets, stated explicitly because a wrong token fails silently (PRD 004
  §12):
  - in-our-care counter → `['spejder']` (type: a member whose id was never seen must
    still move the number)
  - case detail's team card → `['sos:{id}', 'sos', 'spejder']`
  - checkgroup status view, once discontinuation is live → add `'spejder'` to its
    existing dependencies
- Use the SPA's dev-only dependency validation while building.
- Optimistic writes for member actions: an operator on the phone must never wait for
  a round trip. But **not** for the resume action, which the server may legitimately
  reject — show it as pending and let the server answer.

### API endpoints

| Method | Path | Purpose |
|--------|------|---------|
| PUT | `/api/member/:memberId/waiting` | Record that the member wants to leave the race (→ `waiting`), `sosId` required |
| PUT | `/api/member/:memberId/racing` | Member carries on under their own steam (→ `racing`); rejected unless currently `waiting` |
| PUT | `/api/member/:memberId/status` | Override a member's status (correction path), optional `sosId` |
| PUT | `/api/member/:memberId/team` | Reassign member to another team (`currentTeamId`), optional `sosId` |
| POST | `/api/sos/:id/team/:teamId/collect` | Collect the whole team: every remaining `racing` member → `waiting`, one action |
| POST | `/api/sos/:id/team/:teamId/exception` | Grant an exception to the 3-member requirement (reason required) |
| GET | `/api/sos/:id/team/:teamId/reassign-candidates` | Patrols available to receive the remaining members |
| GET | `/api/member/care` | In-our-care counts by status + oldest `waiting` timestamp |

Notes on the shape:

- **Member actions live on the member; breach handling lives on the case.**
  Associating, collecting and excepting a team are facts about *this case's handling
  of that team*; a member's status is a fact about the member, which is why the
  member endpoints are not nested under `/api/sos`.
- The withdrawal request and the override are **separate endpoints** rather than one
  parameterised status setter, because they are different acts: one is the normal
  workflow this interface owns, the other is an admission that another interface's
  handover went unrecorded. Splitting them keeps the override auditable and makes
  "how often are we correcting by hand?" answerable. `sosId` is required on the
  withdrawal request — a member should not leave the race without a case explaining
  why — and optional on the other two.
- `/api/member/...` uses the noun `member` while the projection is `spejderstatus`
  and no existing route uses `member`. That is deliberate (`MemberStatus` is a
  *member* lifecycle) but it collides with the seniors question in §11 — decide the
  noun once, since it leaks into subjects, live tokens and the eventual lift.
- Legacy `/api/sos/merge` and `/api/sos/split` are **replaced** by member
  reassignment plus derived discontinuation, and are not ported.
- OpenAPI: same open question as PRD 001 — hq has no OpenAPI tooling today.

### Data / storage

- `spejderstatus` gains `initialTeamId`, `currentTeamId` and an index on
  `(year, currentTeamId)`; the table must be recreated in dev for the columns to
  appear.
- Breach handling needs somewhere to live — a new table or additional
  `sos_activity` entry types (PRD 001 requires that table to be extensible for
  exactly this).
- `patruljemerged` table and projection are removed.

### Dependencies & risks

- **PRD 001 is a prerequisite**: this PRD renders inside its case detail view and
  writes to its timeline. It adds no views of its own.
- **PRD 005 (boot gate) is a real dependency** for the counters, as argued in §6.
  Not shipped; it is still a draft skeleton and there is no readiness gate in
  `go/cmd/api/main.go`.
- **Cross-repo work on the critical path:** the member events in
  `shared-go/messages/member.go` and the legacy status-value mapping. `MemberStatus`
  itself is already done and already pinned (`go/go.mod` at
  `v0.0.0-20260807180020-5ac2603c60ba`). Risk if skipped: events get encoded with
  locally-defined types and every consumer must be revisited later.
- **This is the first consumer of the documented member lifecycle** — nothing in
  shared-go references the new constants yet — so it is the feature that proves the
  model. Anything that does not fit should be raised as a change to
  `types/member.go`, not worked around locally.
- **Discontinuation is a derived fact, not a status.** "Discontinued" describes a
  *team*, member statuses describe *members*, and the team fact follows from the
  members. Getting this wrong is the main modelling risk — do not add a
  `discontinued` flag operators can set independently of membership, or it will
  drift the way legacy `patruljemerged` rows could.
- **Blast radius of reviving `discontinued`:** `discontinuedTeamIds` feeds the
  checkgroup/checkpoint status view (`go/internal/data/models.go:166`, which computes
  `cg.Discontinued` and `cg.NotArrived`). Since the query currently returns nothing,
  implementing it **will change what that view shows** — previously-"not arrived"
  teams start appearing as discontinued. That is correct behaviour, but it is a
  visible change to an existing screen and should be verified with organizers before
  the event.
- **The withdrawal route does not complete end to end until the car and shelter
  interfaces ship.** A member put into `waiting` has no automatic way out, so
  `InOurCare()` will not drain on its own and the `waiting` alarm will fire for
  everybody. The override is the interim path; see the seam section.
- **Lift-readiness can rot silently.** A single convenience import of
  `nathejk.dk/...` turns the future lift from a file move into a rewrite, and nothing
  in the build will complain. Worth an explicit review check.
- **Open questions outnumber the code.** Several of them (the discontinued
  predicate, the reassignment rules, where the minimum lives, how an exception is
  modelled) change the implementation rather than decorating it, so this PRD should
  not be approved into `doing/` until at least those four are answered.

## 9. Success Metrics

- **The in-our-care count reaches zero before the organisers go home, and it did so
  because every member was genuinely accounted for** — not because somebody cleared
  stale rows. This is the metric the lifecycle documentation is written around, and
  it is the one that matters: a stale status here is a member nobody is looking for.
- No member ends the event `waiting` or `transit` — either would mean we lost track
  of somebody in our care.
- Median time from `waiting` to `transit` (how long a blocked patrol waits for a
  car), tracked from the timeline. No target for the first event; establish a
  baseline. Note this measures the **car fleet**, not this interface — it is here
  because this is where the data lands, and it is the number that tells organisers
  whether they have enough cars.
- **Every breach of the 3-member requirement has a recorded handling** — collected,
  reassigned, or an exception granted with a reason. Zero unhandled breaches at the
  end of the event is the target; *which* handling was chosen is not the measure, and
  a patrol that finished on an exception is a success.
- Exceptions are visible and reviewable afterwards. If most breaches were resolved by
  exception, either the requirement or the car capacity needs a conversation — which
  is only possible because the exception is recorded rather than implied.
- Overrides stay rare. A high override count means handovers are not being recorded
  where they happen, and the chain of custody is fiction.
- Resumes (`waiting` → `racing`) are tracked, not minimised. Each one is a car not
  sent and a patrol that kept going, so the count is a benefit rather than a defect —
  but a high *rate* may mean operators are recording `waiting` too eagerly during the
  call.
- The checkgroup status view stops showing discontinued teams as "not arrived", and
  organizers confirm the new numbers match reality at the checkpoints.

## 10. Rollout / Task Breakdown

Sequenced so the projection is trustworthy before anything is built on it. Live
adoption is part of each frontend task, not a phase.

- **Phase A — the projection.** Revive `spejderstatus`, derive `racing`, normalise
  legacy values, extend the schema, add the strength / in-our-care / discontinued
  queries. Verifiable on its own by replaying the real stream and comparing member
  counts against `patrulje.memberCount`, with no UI at all.
- **Phase B — the operator's own transitions.** `waiting`, resume, override,
  reassign, plus the member rows and the in-our-care counter in the UI.
- **Phase C — the 3-member requirement.** Strength display, pre-commit warning,
  collect-all, reassignment candidates, exceptions, recorded handling.
- **Phase D — discontinuation.** Implement the query, retire `patruljemerged`,
  verify the checkgroup view with organizers.

Proposed tasks to create in `roadmap/tasks/open/` (not created yet):

- [ ] Task: shared-go — design & add the member lifecycle events to
      `messages/member.go` (withdrawal requested/cancelled, pickup accepted, shelter
      accepted, handover completed), each carrying the acting party
- [ ] Task: shared-go — implement the legacy → current `MemberStatus` value mapping
      as shared code (documented in `types/member.go`, not yet implemented)
- [ ] Task: Local — revive `spejderstatus` as a package to shared-go guidelines:
      consume `patrulje.*.started` (→ `racing`) and `spejder.*.deleted`, legacy-value
      normalisation, year from subject
- [ ] Task: Local — `spejderstatus` schema: `initialTeamId` / `currentTeamId` +
      index on `(year, currentTeamId)`
- [ ] Task: Local — team-strength query with shared in-breach / discontinued
      derivation
- [ ] Task: Local — in-our-care query (`InOurCare()` statuses + oldest `waiting`)
- [ ] Task: Local — member endpoints: `waiting` request, resume to `racing` (with
      self-carrying dirty-check), status override, reassign team
- [ ] Task: Local — whole-team collection as a single command (one `correlationId`,
      per-member events)
- [ ] Task: Local — record breach handling (collected / reassigned / exception
      granted, with reason & operator)
- [ ] Task: Local — reassignment-candidate query (available target patrols)
- [ ] Task: Local — implement the discontinued-teams query from membership
      (excluding finished teams)
- [ ] Task: Local — consolidate the discontinued query paths; delete the
      `patruljemerged` projection
- [ ] Task: Local — extend `sos_activity` with the member/breach entry types, and
      decide case correlation for externally-produced transitions
- [ ] Task: Local — move the 3-member minimum out of `go/cmd/api/patrulje.go:99`
      into configuration
- [ ] Task: Frontend — member rows in the case team card (status, timestamps,
      acceptor, `Ønsker at udgå` / `Fortsætter selv`, override, reassign)
- [ ] Task: Frontend — **I vores varetægt** counter + `waiting` alarm on the
      nødtelefon list view
- [ ] Task: Frontend — breach warning + pre-commit warning on the `waiting` action,
      with collect-all / reassign / grant-exception actions and dirty-guarded dialogs
- [ ] Task: Frontend — new `SosActivityLine` entry types
- [ ] Task: Review check — assert the `spejderstatus` package imports nothing from
      `nathejk.dk/...` (lift-readiness)
- [ ] Task: Verify the checkgroup status view with discontinued teams re-enabled
- [ ] Task: Confirm the `waiting` alarm threshold with organizers
- [ ] Task: Follow-up (post-stabilisation) — lift `spejderstatus` into
      `shared-go/tables/`
- [ ] Task: Follow-up PRDs — car-acceptance interface and shelter-acceptance
      interface (out of scope here; this PRD fixes the seam they build against)

## 11. Open Questions

The four marked **blocking** change the implementation rather than decorating it and
should be answered before this PRD moves to `doing/`.

- **(blocking) Discontinued predicate vs finished teams:** a team that finished also
  has nobody `racing`, so "nobody racing" alone would mark every finishing team
  discontinued. Proposal: a team is discontinued when no member with
  `currentTeamId = team` is `racing` **and** none is `finished` — i.e. everybody left
  the route. Confirm, including the in-between case: a patrol where two members
  finished and one was `released` mid-route is finished, not discontinued — is that
  right?
- **(blocking) Derived vs evented discontinuation:** derive on read from
  `spejderstatus` (proposal), or publish an explicit `patrulje.discontinued` event
  and project it? The evented version puts the fact on the log but needs a matching
  un-discontinue path to keep the legacy `.splited` reversibility. Related: should an
  operator be able to discontinue a team **directly**, independent of its membership?
- **(blocking) Reassignment target rules:** what makes a patrol an available target?
  Same year and still racing are obvious; beyond that, does it need to be near the
  stranded members (same checkpoint, same city, same liga), under some maximum size,
  or is it purely the operator's judgement from a list? May survivors be split across
  two different target patrols, or must they stay together? And may a member be
  reassigned to a *klan*, or patrols only?
- **(blocking) How is a granted exception modelled and scoped?** A first-class event
  (`sos.understrength.exception.granted`) or a case-scoped record? And is it scoped to
  the breach (a later member leaving is a new breach needing new handling — the
  proposal) or to the team for the rest of the event?
- **Correlating downstream events to a case:** propagate the `correlationId` from the
  `waiting` event through the chain, or resolve at read time from the member's team
  and open cases? The first needs every downstream producer to cooperate — so it is
  the one seam decision the future car and shelter PRDs may need to influence; the
  second is self-contained but ambiguous when a member has two open cases.
- **Where does the required minimum live?** 3 is the requirement for patrols, but it
  should be configurable rather than compiled in (it is currently hardcoded at
  `go/cmd/api/patrulje.go:99`). Year configuration seems the natural home since it is
  a rule of the event. Confirm, and confirm whether it has ever differed between years
  or ligas.
- **`waiting` alarm threshold:** how long may a member be `waiting` before the
  dashboard warns? Their patrol is blocked for the whole duration, so this is the one
  number operators will feel. Fixed config value or per-severity?
- **Where the in-our-care count lives:** the nødtelefon list view (proposal), the HQ
  dashboard (`/api/home`), or both? It is an event-wide number rather than a SOS one.
- **Sequencing strictness:** should the API enforce the documented order (reject
  `racing` → `sheltered`, say) or accept any `Valid()` status and let the timeline
  show what happened? Strictness protects the data; leniency matters at 3am when the
  real world skipped a step. Note the override endpoint is where this bites.
- **Seniors as well as spejdere:** `spejderstatus` is named for spejdere, but
  `MemberStatus` is documented as a *member* lifecycle and klan members are members
  too. Does the revived projection cover seniors, and if so does the name stay as-is
  (shared-go's documentation refers to "the spejderstatus projection")? This also
  decides the `member` vs `spejder` noun in the routes and the live token. PRD 001
  scopes cases to patrols, so it is not blocking, but it shapes the eventual lift.
- **Member event subject entity:** `spejder` (proposal — reuses the existing live
  token) or a new `member` entity? Affects `dependsOn` tokens across the SPA and is
  tangled with the seniors question above.
- **Producer ownership of `registered` / `seated`:** which flows set them (signup +
  orders?) — not needed by this feature, since the panel only sees members who have
  started, but worth knowing before `spejderstatus` is treated as complete. `racing`
  is settled: derived from `NathejkTeamStarted`.
- **Existing `patruljemerged` data:** historical rows encode past discontinuations
  the new model cannot reconstruct (there are no per-member reassignment events behind
  them). Since legacy data migration is out of scope, confirm we simply drop them and
  start fresh for the current year.
- **What happens to a reassigned member's finish?** A survivor moved into another
  patrol is still `racing` and self-carrying, so they can still finish — but with a
  team that is not the one they started with (`initialTeamId` ≠ `currentTeamId`). Does
  that affect diplomas, results, or how the finish is recorded?
- **Does the requirement apply to klaner?** The 3-member rule as stated is
  patrol-specific and klaner presumably have their own (or none).
- **Deferred to the car and shelter PRDs** (recorded here only so they are not lost):
  which product becomes the car interface and which the shelter one; who performs the
  final handover (`released` / `reunited`) — `reunited` happens at the finish line
  rather than the shelter, so it may not belong to either; and car dispatch, including
  whether the `waiting`-too-long alarm should *request* a pickup. The only piece that
  may need to sit in the nødtelefon interface is that request, so it is worth a
  decision when the car PRD is written rather than now.
