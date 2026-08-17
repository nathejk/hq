# PRD 006 — Member lifecycle, team strength & discontinuation

**Status:** doing
**Author:** agent session
**Created:** 2026-08-10
**Last updated:** 2026-08-17
**Approved:** 2026-08-17
**Shipped:**
**Status note:** split out of PRD 001 on 2026-08-10; member lifecycle settled
against shared-go `v0.0.0-20260815075712-35c10e0f6942`. Revised 2026-08-17: the
merged-team concept is deprecated in favour of a member-focused model, team
discontinuation becomes an `activeMemberCount` on the team with **no**
`team.discontinued` event, breaches of the 3-member requirement are **recorded rather
than handled** (no exception mechanism), every operation publishes one `spejder` event
per member plus one summarising `sos` event, and everything checkgroup-related is out of
scope. **All four originally-blocking questions are now decided** — see Decisions in §11.
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
it (collect the team or move the survivors); and maintain an
**`activeMemberCount` on the team**, so that a team with none is discontinued
(udgået) — retiring the legacy `patruljemerged` merge/split encoding.

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
- **A second, quieter gap:** nothing tracks how many members a team still has on the
  route. `patrulje.memberCount` is frozen at whatever started
  (`table/patrulje/consumer.go:66`), so it cannot answer "does this patrol still have
  three?", and there is no team-level fact for *discontinued* at all. The legacy
  mechanism that encoded the same need — `patruljemerged`, written by
  `patrulje.merged` and deleted by `patrulje.splited` — is not being produced any
  more, and **the concept it encoded is deprecated**: teams are not merged and split,
  members are **moved**. A team is discontinued because nobody is left in it, which is
  a consequence of membership rather than an act performed on the team.
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
    has no home. The same `TeamConfig` literal is built with `MinMemberCount: 1` in
    three other handlers (`klan.go:170`, `badut.go:48`, `mail.go:59`), so the value
    has four call sites, not one.

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
  event, not a preference — and the tool's job is to make a breach **visible**, not to
  adjudicate it. The interface therefore surfaces the breach unmissably and offers the
  actions an operator actually takes, without blocking the transition, deciding
  anything, or asking anyone to justify themselves:
  1. The member carries on (`waiting` → `racing`) and the requirement is met again.
  2. **Collect the whole team** — the remaining members go `waiting` too and leave
     together.
  3. **Move the remaining members** to another patrol. The nødtelefon does not
     *choose* the destination — in practice crew in the field contact a nearby patrol
     and agree it, and the operator records the outcome. Any patrol from the same year
     still racing is a legitimate target.

  And a patrol may simply continue below three. That happens, it is a judgement made by
  people in the field, and **there is nothing to grant and nothing to approve**: the
  record is the member changes themselves plus the strength they leave behind. No
  exception object, no reason text, no approval step — see §11 Decisions.
  Options 2 and 3 both leave the original team with no active members, so it becomes
  discontinued — which is what the legacy `merged`/`splited` events were reaching for,
  and why the old `patruljemerged` table pointed at a `parentTeamId`. Option 3 is that
  same outcome expressed the way the domain actually works: a member is moved.
- **Every team carries an `activeMemberCount`**, maintained from member status, so
  team strength and discontinuation are the same number rather than two independent
  derivations: a team that started and whose count is zero is **discontinued** (udgået),
  and moving a
  member into it makes it active again. There is **no `team.discontinued` event** —
  the count is the fact, and a team cannot be discontinued except by losing its
  members.
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
  member is **moved** to another team, and a team with nobody racing is
  **discontinued**. The replacement must reproduce the observable behaviour of the
  old events, including their reversibility — legacy `.splited` deleted the
  `patruljemerged` row and thereby un-discontinued the team, which
  `activeMemberCount` gets for free by being recomputed rather than set.
- **The checkgroup / checkpoint overview.** Whether and how a discontinued team is
  reflected at the checkpoints is not this PRD's concern. `activeMemberCount` makes
  the fact available on the team; consuming it is the checkgroup view's own work.
  Nothing in `cmd/api/checkgroup.go`, `table/checkgroup/` or `PostList.vue` changes
  here, and `patrulje.querier.GetDiscontinuedTeamIDs` — which has no callers — is left
  where it is.
- **An "Udgået" list page.** `Navigation.vue:160` links to `/ude`, a route that does
  not exist, and shared-go's `spejder.GetInactive` is commented out waiting for this
  projection (`tables/spejder/querier.go:90-93`). Both become buildable once this
  ships; neither is built here. Recorded in §11 so the dead link is not mistaken for
  something this PRD forgot.
- Migrating historical `patruljemerged` data (see Open Questions).
- Position-request SMS / GPS location of members, and real-time map tracking.
- **Klan (senior) members entirely.** Decided 2026-08-17: klaner are **not handled
  through the nødtelefon**, so this projection covers spejdere only. That settles the
  seniors question outright rather than deferring it — the subject noun is `spejder`, the
  projection keeps its name, and a klan member has no lifecycle here. If seniors ever need
  one it is a separate entity with a separate token, not a widening of this.

## 5. User Stories & Scenarios

- As an **operator**, I want to record that a member wants to leave the race and is
  waiting to be collected (`waiting`), so that a car can be sent and the member is
  counted as being in our care from that moment.
- As an **operator**, I want to put a member who has changed their mind back into
  the race (`waiting` → `racing`) as long as no car has collected them yet, so a
  scout who gets their breath back can carry on and no car is sent needlessly.
- As an **operator**, I want to be told immediately when the member leaving would
  put their patrol below the required 3 racing members, and to have the actions I
  actually take — collect them all, or place the survivors with another patrol — to
  hand, because I am the one on the phone.
- As an **operator**, I want what happened recorded — the member changes and the
  strength they left behind — so the next shift and the post-event review can see it
  without me writing an explanation.
- As an **operator**, I want to watch the member's progress — accepted into a car,
  accepted at the shelter, handed over — without having to update it myself, because
  the people receiving the member record it as it happens.
- As an **operator**, I want to see how many members are in our care right now, and
  be warned when somebody has been `waiting` too long — their patrol cannot continue
  until they are collected.
- As an **operator**, I want to **override** a member's status when it is wrong, to
  correct out-of-sync data.
- As an **operator**, I want to move a member from one team to another so that,
  e.g., a scout who continues with a different patrol is tracked correctly; when a
  team is left with nobody racing its `activeMemberCount` reaches zero and it is
  **discontinued**.

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
- Retiring the last racing member of a team takes its `activeMemberCount` to zero and
  the team is thereby discontinued. The case timeline records the member change; the
  team fact follows from it and has no event of its own.
- Moving a member *back* into a team with nobody left makes it active again — the same
  reversibility legacy `.splited` had, for free: `activeMemberCount` is recomputed on
  every membership or status change rather than set once.
- A team already below the requirement when a case is opened (two members left
  earlier in the event) is in breach from the start; the warning reflects the team's
  current strength, not only the transition that caused it.
- Members are moved **one at a time**, so two survivors may end up in two *different*
  patrols. Uncommon — most often they stay together — but the flow must not assume a
  single destination for the whole group, and "move these two to that patrol" must not
  be the only shape available.
- There is effectively always a target: any racing patrol from the same year qualifies,
  so the flow cannot dead-end for lack of candidates. What it can dead-end on is nobody
  in the field having agreed one yet — in which case the operator has not been told a
  destination, and the breach stays visible until they are.
- A team that stays below three keeps showing as under styrke for the rest of the
  event. Nothing settles it, because nothing can be granted — the warning is a
  statement of fact, not an open item awaiting sign-off.
- If the member resumes after the operator has already collected the rest of the
  team, the team is *not* automatically restored — those members are `waiting` and
  only they (or a car) can change that. Resolving one member does not unwind actions
  taken for others.
- A member who left the trail must never end up `finished`. `MemberStatus.CanFinish()`
  is true only for `racing`, so a finish-line flow cannot promote a `reunited`
  member — and the UI must not offer `finished` as an override target either. Note a
  member who resumed (`waiting` → `racing`) *can* finish, correctly: they walked the
  rest of the route themselves. **No producer sets `finished` today** — there is no
  finish flow anywhere in hq (no finish event, no `endedUts`; `patruljestatus.sql`
  holds only `startedUts`), and this PRD does not add one. `CanFinish()` is therefore
  a guard for a flow that arrives later.
- Because nothing sets `finished`, members stay `racing` after their team reaches the
  end, so **a finishing team's `activeMemberCount` does not fall to zero and it is not
  mistaken for discontinued**. That is luck, not design: the day a finish producer
  lands, every finishing team's count drains and the naive reading of the number turns
  every finisher into an udgået patrol. Recorded here as the trap that work must
  handle — see §11 — rather than pre-solved with a predicate nothing can exercise.
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
- [ ] **Rework `GetSpejdere` so it stops inventing statuses.**
      `go/internal/data/member.go:42` currently synthesises a status when no
      `spejderstatus` row exists:
      `IF(ss.status IS NULL, IF(ps.startedUts > 0, 'started', 'paid'), ss.status)`.
      Neither `started` nor `paid` is a valid `types.MemberStatus` and `paid` is not
      even in the legacy mapping, so the endpoint behind `GET /api/patrulje/:id`
      (`cmd/api/patrulje.go:85`) serves values no lifecycle helper recognises. Status
      comes from messages: serve `ss.status`, and `MemberStatusNone` where there is no
      row yet. Same for the identical join in `data/member.go:55`'s sibling queries and
      the export path (`cmd/api/export.go:88`).
- [ ] Populate `currentTeamId` on the member payload. `data.Spejder.CurrentTeamID`
      exists with a JSON tag but is never selected or scanned, so it is always empty
      today.

**Operator actions (self-carrying only)**

- [ ] Mark a member as **`waiting`** — wanting to leave the race and awaiting
      collection. Recorded on the SOS case timeline; `sosId` required.
- [ ] Return a `waiting` member to **`racing`** when they choose to carry on.
      Permitted **only** from `waiting`, enforced on the write side, with a message
      the operator can act on ("allerede hentet") rather than a generic conflict.
- [ ] Override a member's status to any valid value to correct out-of-sync data,
      **excluding `finished`**. **Not on the case card** — it lives on the patrol page's
      member list (§7), because it is a correction rather than part of the call the
      operator is on. Being a different screen is what keeps it from becoming a shortcut
      for work another interface owns.
- [ ] **`sosId` is required on every member command**, including the override and the
      move. Nothing changes a member's status or team without a case explaining why. Where
      the operator has no case — the correction path — the backend **creates one and closes
      it immediately**, purely as the record (§8).
- [ ] Move a member to another team: updates `currentTeamId`, leaves
      `initialTeamId` untouched, recomputes `activeMemberCount` on **both** teams, and
      is recorded on the case timeline.

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

- [ ] **Team strength is `activeMemberCount` on the team**: the number of members
      whose `currentTeamId` is the team and whose status is `racing` — matching
      `MemberStatusRacing`'s documentation that it is "the only state in which a member
      counts towards their team's strength on the route". `waiting` members do **not**
      count. Maintained by the member projection on every membership or status change,
      never computed ad hoc by a caller.
- [ ] **Warn, at the moment of setting `waiting`, when it would put the team below
      the required 3 racing members**, naming the resulting count, *before*
      committing — it changes the conversation the operator is having on the phone.
      The warning does not block the transition; the member is leaving whether or not
      the team is compliant.
- [ ] Show which teams on a case are below the requirement, and keep showing it
      while it is true, so an operator taking over a shift can see the state of play.
      It is a statement of fact with nothing to acknowledge: it appears when strength
      drops below three and disappears when it does not, and nothing else clears it.
- [ ] **Collect the whole team** as one action: every remaining `racing` member goes
      to `waiting` together, recorded as a single timeline entry rather than one per
      member. Operators are on the phone; three separate clicks invite two of them
      being forgotten.
- [ ] **Move the remaining members** to another patrol. A valid target is any patrol
      in the same year that is **still racing** (started, `activeMemberCount > 0`) and
      is not the team they are leaving. Nothing further: no proximity, liga or size
      rule, because the destination is agreed by crew in the field and the operator is
      **recording** it, not choosing it — so a backend that second-guessed the choice
      would only be able to reject a decision already made on the ground.
- [ ] Moving is **per member**, with moving several to the same target as the
      convenient default. Two survivors may go to two different patrols, so the flow
      must allow that even though it is rare.
- [ ] **No exception mechanism.** A patrol continuing below three needs no grant, no
      reason and no approval; the record is the member changes and the resulting
      strength. Nothing in the schema, the events or the UI represents an exception, and
      there is no handled/unhandled distinction to track.
- [ ] The required minimum (3) is a **configured value**, not a literal in code —
      note it is currently hardcoded at `go/cmd/api/patrulje.go:99`. Where it lives
      is an Open Question.

**Discontinuation**

- [ ] A team that **started** and whose **`activeMemberCount` is zero** is
      **discontinued** (udgået). Derived from the count, recomputed on every membership
      or status change, and therefore reversible: moving a member in makes the team
      active again.
- [ ] **The "started" half is not optional.** A team that never started also has zero
      racing members, so the count alone does not distinguish *left the route* from
      *never on it*. Measured on the dev data (2026-08-17): the naive predicate marks 239
      abandoned 2025 signups **and all 310 teams of the current 2026 event** as udgået.
      The signal is `patrulje.signupStatus = 'STARTED'`, set by the patrulje consumer
      from the same `patrulje.*.started` event the member projection derives `racing`
      from. Note `patruljestatus.startedUts` is **not** that signal despite its name — it
      is hardcoded to 1 on *signup* (`table/patruljestatus.go:87`).
- [ ] **No `team.discontinued` event and no discontinued flag.** The count is the
      fact. Nothing may set discontinuation independently of membership, and no
      operator action discontinues a team directly — they retire or move its members.
- [ ] `activeMemberCount` is exposed on the team wherever the team is served, so any
      consumer (this PRD's case card now, others later) reads one number for both
      strength and discontinuation.
- [ ] The `patruljemerged` projection, its consumer and its table are **deleted**.
      Note two dead readers remain — `table/checkgroup/handler.go:48` and
      `table/year/handler.go:48` both `SELECT ... FROM patruljemerged` — but both sit
      behind `/api/cgstatus`, which is commented out at `routes.go:91`. They are
      unreachable; deleting the table leaves them referencing a table that is gone.
      Flagged for whoever revisits checkgroup rather than fixed here.

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
- **Auditability:** every transition and every override is attributable to a **time**,
  and to a person once per-user identity exists (today authentication is perimeter-only
  — basic auth on stage/production, none in dev — so the actor is recorded but empty;
  see PRD 001 §6). With no exception object to explain a short-handed patrol, the audit
  trail *is* the sequence of member changes and the summarising case entries: "what
  happened to this patrol?" is answered by reading them in order, which is why the
  summary event must be self-contained (§8) rather than re-derived from current state.
- **Localization:** Danish UI text and `da-DK` date formatting.

## 7. UX / UI Notes

All of this lands **inside PRD 001's surfaces**; this PRD adds no new views.

- **`vue/src/components/SosTeamCard.vue`** (the *Tilknyttede patruljer* card),
  extended. PRD 001 shipped this card with per-patrol identity and contact only — its
  header comment says so explicitly and reserves the space — so this PRD **introduces
  the member rows** rather than decorating existing ones; they arrive together with the
  status and actions that make them worth showing:
  - The team's **strength** (`activeMemberCount`) beside its name, an **Under styrke**
    warning when below the required 3, and an **Udgået** badge when the count is zero.
  - Each member row shows the current status with its timestamp and, where known,
    who accepted them. Row actions depend on status: `racing` offers **Ønsker at
    udgå** (→ `waiting`); `waiting` offers **Fortsætter selv** (→ `racing`) as a
    normal, prominent action — not buried in an override menu, since a scout getting
    their breath back is an ordinary outcome and saves a car being sent. **From
    `transit` onwards the row is read-only**: it reflects what the car and shelter
    have recorded and offers no buttons to advance or reverse them.
  - Secondary action: **Flyt til anden patrulje** (move the member). Use PrimeVue
    overlay/popover for the menu, not `b-popover`. **No override here** — corrections live
    on the patrol page, see below.
  - Members `waiting` past the threshold are highlighted.
- **Below-strength panel** on the same card: when a team on the case has fewer than
  3 racing members, a prominent warning stating the current strength and offering the
  two actions — **Hent hele patruljen** (all remaining racing members → `waiting`, one
  action) and **Flyt de resterende** (move survivors, picking the destination with the
  same filtered patrol picker the card already uses to associate a team — see §8).
  Confirming `Ønsker at udgå` for a member whose departure causes the breach warns
  *before* committing, naming the resulting strength ("Patruljen har kun 2 tilbage"),
  and offers the same two actions plus simply proceeding. The warning then stays for as
  long as the team is short-handed and needs no acknowledgement: there is nothing to
  grant, so there is nothing to settle. It states a fact and the timeline below it says
  how the team got there.
- **`PatruljeView.vue` → the members list's expanded row: the correction interface.**
  When reality and the record disagree — a member was driven in and nobody wrote it down,
  a status was set on the wrong person — this is where it is put right. The expanded row is
  currently a placeholder dumping `{{ data }}`, so there is room.

  It shows the member's current status with its timestamp and acceptor, and offers setting
  it to any valid value **except `finished`**. Deliberately *not* on the case card: a
  correction is not part of the call an operator is on, and putting it on a different
  screen is a stronger separation than a visually-distinct button on the same one — which
  is what §6 was reaching for when it asked for the override to be hard to use as a
  shortcut.

  **Every correction still produces a case.** The operator has none here, so the backend
  makes one and closes it immediately (§8). The patrol page already lists the patrol's
  cases in its **Kontakt med nødtelefon** card, so corrections surface in context, on the
  same page they were made, without an operator having to go looking — and without
  polluting a live case with unrelated bookkeeping.
- **`SosListView.vue` header:** a permanent **I vores varetægt** counter
  (`InOurCare()`: waiting + transit + sheltered) with a breakdown per status, and a
  warning state when any member has been `waiting` past the threshold. This is the
  organisers' go-home number, so it should be visible without opening a case.
- **New timeline entry types** in `SosActivityLine.vue`, one per kind of summarising
  case event (§8): member status changed, members moved, team collected. There is **no**
  team-discontinued entry — discontinuation has no event, so what the timeline shows is
  the operation that took the count to zero — and no exception entry, since exceptions do
  not exist. PRD 001 requires the component to tolerate unknown types, so this is
  additive.
- **Drag-free, dirty-guard-aware:** the move dialog holds unsaved state, so it must
  defer incoming payloads while open, as `KlanListView.vue` does, and say updates are
  paused.

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
  same class of drift for signup). `patrulje` gains `activeMemberCount` and hits the
  same trap.
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

### `activeMemberCount` is owned by the member projection

The count lives on the team (`patrulje.activeMemberCount`) but is a function of member
rows, so **the member projection writes it**, in the same `HandleMessage` that writes
the member's own row, by recomputing from `spejderstatus` for the affected team(s).
Moving a member touches two teams and both must be recomputed.

This breaks the usual "one projection owns one table" rule, deliberately, because the
alternative is worse: if the `patrulje` consumer maintained the count by reading
`spejderstatus`, it would be **racing its sibling consumer over the same message** —
the mux gives both consumers the event with no ordering guarantee between them, so the
recompute could read the row before the member projection has written it and land a
count that is quietly one out. Recomputing (rather than incrementing) also means a
replay converges regardless of the order events arrive in, and no event needs to carry
the member's previous status.

Consequences worth stating:

- **Discontinued is not a column.** It is `signupStatus = 'STARTED' AND
  activeMemberCount == 0`, read wherever the
  team is read. Nothing to keep in sync, nothing to un-set, no reverse event.
- **Live updates come free from the member signal.** Because the count changes only in
  response to a member event, the `spejder` token already announces it — a view showing
  team strength depends on `spejder`, not on `patrulje`. State this in the frontend
  task, because the intuitive-but-wrong choice (`patrulje`, since the number is on the
  team) fails silently.
- **A team's own events do not touch the count.** `patrulje.*.started` sets
  `memberCount` as it does today; `activeMemberCount` is set by the member projection
  when it derives `racing` from the same event.

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
  and `member.team.moved`, each per member, plus one summarising case event per
  operation (below). It **consumes** the rest. There is deliberately **no team-level
  event**: no `patrulje.discontinued`, no `patrulje.merged` replacement, and no
  exception event.

### Every operation is N member events plus one case event

**Decided.** A member-changing operation publishes **one event on the `spejder` entity
per updated member**, and then **one event on the `sos` entity summarising the whole
operation**. Uniformly — a single member going `waiting` is one `spejder` event plus one
`sos` event, exactly like a whole-team collection is three plus one. No special case for
the single-member path, which is what keeps the timeline and the projection from
disagreeing about what an "operation" is.

Each half has exactly one reader:

- The **`spejder` events** drive `spejderstatus` and `activeMemberCount`. They are
  per-member because that is the grain the projection works at, and they say nothing
  about cases — **no `sosId` in the payload**. That is what lets the car and shelter
  interfaces publish the same kind of event later without inventing a case they know
  nothing about.
- The **`sos` event** is the timeline entry. It is published last, so a reader that sees
  the summary is guaranteed the member events preceding it are already in the log, and
  it lands on `NATHEJK.{year}.sos.{id}.*` — which means `table/sos/consumer.go` needs no
  new subject pattern and no knowledge of member events. One row per operation, which is
  how "hele patruljen hentes" reads as one line while three members changed status.

`sosId` is therefore a parameter of the *command* (which case is this being done from?),
not a field on the member event. It is required for the withdrawal request and optional
for the override and the move; when it is absent, the member events are published and
the summary simply is not.

**The summary must be self-contained.** It carries who changed, from what to what, and
the team's resulting strength — enough to render the line without joining to current
state. A timeline entry that re-derives its text from today's member rows shows today's
truth on yesterday's line, which is precisely the failure a handover log cannot have.

Two consequences worth stating:

- **Live signals collapse.** N+1 events would be N+1 signals, but the hub coalesces
  within `DefaultCoalesceWindow` (75ms, `internal/live/hub.go`), so an operation emits
  effectively one `spejder` and one `sos` signal however many members it touched. The
  coalescing window was written with exactly this in mind.
- **Downstream transitions still have no case event.** The car and shelter interfaces
  will publish `spejder` events and no `sos` event, since they have no case. So their
  transitions update the projection and the counters but do **not** appear on a case
  timeline unless the timeline merges them at read time — see the remaining open question
  in §11. Their absence is not a bug in this decision; it is the price of not forcing a
  `sosId` onto interfaces that cannot know one.
- **Whole-team collection is one command, not N.** A single
  `sos.team.collected`-style command publishes a withdrawal request per remaining
  racing member, atomically from the operator's point of view, sharing one
  `correlationId`, and then the one summarising `sos` event. Publishing three
  independent requests from the frontend would risk a partial collection if one call
  fails — the worst possible outcome, since the team would then be split across
  states with nobody noticing. It is the general rule above applied to the case that
  motivated it.

### The self-carrying boundary is enforced on the write side

`member.withdrawal.cancelled` is valid only while the member is `waiting`; the
command must dirty-check the current `spejderstatus` row and reject it otherwise.
Hiding the button once the status advances is not sufficient — the operator's screen
may be a moment stale, which is exactly when the car is accepting the member. If the
acceptance and the cancellation race, the **acceptance wins**: it reflects a member
physically sitting in a car, and the event log preserves both attempts in order.

### The requirement is reported, never enforced

No command may reject a withdrawal request because it would put a team below 3, and
no consumer may auto-collect or auto-move members in response. The member is leaving
regardless — refusing to record that would only make the data wrong — and what happens
next depends on things the projection cannot know: how far along the patrol is, how
capable they are, whether a car is anywhere near. So the write side reports strength and
records what happened; it never decides.

**And nothing represents "dealing with" a breach.** There is no exception, no resolution
flag, no handled/unhandled distinction — an earlier draft of this PRD had all three. What
replaces them is simpler and truer: `activeMemberCount` says what the team's strength is
now, and the case timeline says what happened to get there. A patrol that continued with
two is visible as a patrol with two, which is the whole of what an incident review needs
and rather more reliable than a reason field filled in at 3am.

### Queries

- **Team strength is a column, not a query.** `patrulje.activeMemberCount` is read
  with the team; "in breach" is `< minimum` and "discontinued" is `== 0` **on a team
  that started**, both read off
  the same number. The recompute behind it is
  `COUNT(*) FROM spejderstatus WHERE year = ? AND currentTeamId = ? AND status = 'racing'`,
  answerable directly given the `(year, currentTeamId)` index.
- **In our care:** count by status over `InOurCare()`, plus the oldest `waiting`
  timestamp for the alarm.
- **Move targets need no endpoint.** A valid target is any patrol in the same year
  still racing, and `SosTeamCard.vue` **already holds that list live**: its team-association
  picker filters the SPA's `patrulje:list` cache (min two characters, capped at ten,
  matching number / name / group) rather than calling a search endpoint. The move picker
  reuses it, filtered additionally on `activeMemberCount > 0` — which the list payload
  carries once §6 exposes the column. So there is no candidate query, no
  `reassign-candidates` route, and opening the picker costs no request. This is the one
  place where "the operator records a decision made in the field" pays off directly: a
  backend rule set would have needed a query, an endpoint and a definition of *near*.
- **Nothing here touches the checkgroup surface.** `patrulje.querier.GetDiscontinuedTeamIDs`
  (`table/patrulje/query.go:122`) returns an empty slice and **has no callers** — the
  live `GET /api/checkgroups` computes its `Missing` count from started teams and scans
  (`cmd/api/checkgroup.go:69-124`) and never asks about discontinuation, and
  `checkgroup.Checkgroup.Discontinued` is declared but never populated. So there is no
  live view to fix and nothing to consolidate; the stub is left alone (§4).

### The acting user needs its own type

`app.actor` (`go/cmd/api/actor.go:23`) returns `sos.Actor`. A member package written to
be liftable cannot import `sos`, so it needs its own actor type — or one in shared-go if
a second is one too many. Trivial, and named here only because the obvious shortcut is
exactly the convenience import that turns the future lift into a rewrite.

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

- Signals derive from the **event subject**, so member changes emit `spejder` and the
  summarising case event emits `sos` — **not** `spejderstatus`, which is a projection
  name and can never appear in a signal.
- `dependsOn` sets, stated explicitly because a wrong token fails silently (PRD 004
  §12):
  - in-our-care counter → `['spejder']` (type: a member whose id was never seen must
    still move the number)
  - case detail's team card → `['sos:{id}', 'sos', 'spejder']`
  - **anything showing `activeMemberCount` or Udgået → `'spejder'`, not `'patrulje'`.**
    The number sits on the team but only ever changes in response to a member event, so
    `patrulje` is the intuitive and wrong answer — and it fails silently.
- Use the SPA's dev-only dependency validation while building.
- Optimistic writes for member actions: an operator on the phone must never wait for
  a round trip. But **not** for the resume action, which the server may legitimately
  reject — show it as pending and let the server answer.

### API endpoints

| Method | Path | Purpose |
|--------|------|---------|
| PUT | `/api/member/:memberId/waiting` | Record that the member wants to leave the race (→ `waiting`), `sosId` required |
| PUT | `/api/member/:memberId/racing` | Member carries on under their own steam (→ `racing`); rejected unless currently `waiting` |
| PUT | `/api/member/:memberId/status` | Override a member's status (correction path). `sosId` required; created-and-closed automatically when the caller has none |
| PUT | `/api/member/:memberId/team` | Move member to another team (`currentTeamId`), `sosId` required |
| POST | `/api/sos/:id/team/:teamId/collect` | Collect the whole team: every remaining `racing` member → `waiting`, one action |
| GET | `/api/member/care` | In-our-care counts by status + oldest `waiting` timestamp |

Notes on the shape:

- **Every member command requires a `sosId`.** Nothing moves a member or changes their
  status without a case explaining why, which is what makes the whole lifecycle auditable
  from one place. The earlier draft had it required for the withdrawal request and optional
  for the override and the move — the one combination nobody would choose deliberately,
  since it made a correction auditable only if the operator happened to be inside a case.
- **The correction path creates its own case and closes it at once.** An operator fixing an
  out-of-sync status from the patrol page has no case, so the backend opens one, records
  the correction on its timeline, and closes it in the same operation. Consequences, all of
  them wanted:
  - the timeline is uniform — there is no second, case-less way for a member's status to
    change, so "what happened to this member?" is always answered by reading cases;
  - it does not clutter the open-case list, because it is closed on arrival, and
    `SosListView` groups by status;
  - it is **countable**, which is what §9's "overrides stay rare" metric needs: one case per
    correction, distinguishable by its headline, makes "how often are we fixing this by
    hand?" a query rather than a guess;
  - it appears in the patrol page's **Kontakt med nødtelefon** card automatically.

  The headline must mark it as machine-created and name what was corrected, so nobody
  reading the case list mistakes it for a call. Reusing an existing open case was the
  alternative and is worse: a correction is rarely part of the story that case is telling,
  and "reuse if exactly one is open" needs a rule for when two are.

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
- Legacy `/api/sos/merge` and `/api/sos/split` are **replaced** by moving members
  plus the derived `activeMemberCount`, and are not ported.
- OpenAPI: not applicable — the mandate was dropped for this repo (PRD 001
  Decisions); endpoints are specified in the PRD instead.

### Data / storage

- `spejderstatus` gains `initialTeamId`, `currentTeamId` and an index on
  `(year, currentTeamId)`; the table must be recreated in dev for the columns to
  appear.
- `patrulje` gains `activeMemberCount`, maintained by the member projection. Same
  `CREATE TABLE IF NOT EXISTS` trap.
- The summarising `sos` events need entry types in `sos_activity` — one per kind of
  operation, with the summary in the existing `value TEXT` column. PRD 001 required that
  table to be extensible for exactly this, so no schema change beyond new `type` values.
- `patruljemerged` table and projection are removed. No table replaces it.

### Dependencies & risks

- **PRD 001 is a prerequisite**: this PRD renders inside its case detail view and
  writes to its timeline. It adds no views of its own.
- **PRD 005 (boot gate) is a real dependency** for the counters, as argued in §6.
  Not shipped; it is still a draft skeleton and there is no readiness gate in
  `go/cmd/api/main.go`.
- **Cross-repo work on the critical path:** the member events in
  `shared-go/messages/member.go` and the legacy status-value mapping. `MemberStatus`
  itself is already done and already pinned (`go/go.mod` at
  `v0.0.0-20260815075712-35c10e0f6942`). Risk if skipped: events get encoded with
  locally-defined types and every consumer must be revisited later.
- **This is the first consumer of the documented member lifecycle** — no shared-go code
  references the new constants yet (`tables/spejder` uses the *type* on a field it
  reads from this very table) — so it is the feature that proves the model. Anything
  that does not fit should be raised as a change to `types/member.go`, not worked
  around locally.
- **Discontinuation is a derived count, not a status and not an event.**
  "Discontinued" describes a *team*, member statuses describe *members*, and the team
  fact follows from `activeMemberCount`. Getting this wrong is the main modelling
  risk — do not add a `discontinued` flag operators can set independently of
  membership, or it will drift the way legacy `patruljemerged` rows could.
- **Reviving the projection changes an existing endpoint's payload.**
  `GET /api/patrulje/:id` currently serves invented member statuses (`started`,
  `paid`) from `data/member.go:42`; once rows exist, real values appear instead. The
  fallback must be **removed**, not left as a default, or members without a row keep
  reading `paid` while `InOurCare()` is asked about them. Harmless on screen today only
  because `PatruljeView.vue:116` renders a hardcoded `"ikke startet"` tag and ignores
  the value — which this PRD should fix while it is there.
- **The withdrawal route does not complete end to end until the car and shelter
  interfaces ship.** A member put into `waiting` has no automatic way out, so
  `InOurCare()` will not drain on its own and the `waiting` alarm will fire for
  everybody. The override is the interim path; see the seam section.
- **Lift-readiness can rot silently.** A single convenience import of
  `nathejk.dk/...` turns the future lift from a file move into a rewrite, and nothing
  in the build will complain. Worth an explicit review check.
- **No blocking open questions remain.** All four of the original ones are decided — two
  dissolved with `activeMemberCount`, and the move-target rules and the exception model
  were settled on 2026-08-17 (§11 Decisions), the latter by removing the concept. What is
  left in §11 is configuration and sequencing, answerable during implementation. The
  residual design risk is the one this PRD deliberately does not close: how, or whether,
  car and shelter transitions reach a case timeline.

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
- **Teams that continued below three are countable afterwards** — from the timeline and
  the strength history, not from an exception log, because there is none. If it turns out
  most short-handed patrols carried on regardless, that is a conversation about the rule
  or about car capacity, and the data supports having it without anybody having had to
  fill in a reason during the night.
- Overrides stay rare. A high override count means handovers are not being recorded
  where they happen, and the chain of custody is fiction.
- Resumes (`waiting` → `racing`) are tracked, not minimised. Each one is a car not
  sent and a patrol that kept going, so the count is a benefit rather than a defect —
  but a high *rate* may mean operators are recording `waiting` too eagerly during the
  call.
- **`activeMemberCount` matches reality when spot-checked.** The cheapest honest test:
  pick a handful of patrols mid-event, ask them how many they are, and compare. A count
  that is quietly one out is the failure this model is built to avoid, and no internal
  consistency check can catch it.

## 10. Rollout / Task Breakdown

Sequenced so the projection is trustworthy before anything is built on it. Live
adoption is part of each frontend task, not a phase.

- **Phase A — the projection.** Revive `spejderstatus`, derive `racing`, normalise
  legacy values, extend the schema, maintain `activeMemberCount`, add the in-our-care
  query, and rework `GetSpejdere` so it stops inventing statuses. Verifiable on its own
  by replaying the real stream and comparing `activeMemberCount` against
  `patrulje.memberCount`, with no UI at all.
- **Phase B — the operator's own transitions.** `waiting`, resume, override,
  move-to-team, plus the member rows and the in-our-care counter in the UI.
- **Phase C — the 3-member requirement.** Strength display, pre-commit warning,
  collect-all, moving members, and the Udgået badge on the case card. Nothing to build
  for exceptions: there are none.

Phase D (discontinuation) is gone: with `activeMemberCount` there is no separate
discontinuation mechanism to build, and the checkgroup work it existed to enable is out
of scope (§4). What remains of it — deleting `patruljemerged` — sits in Phase A.

Proposed tasks to create in `roadmap/tasks/open/` (not created yet; the board is at 062,
so these become 063+):

- [ ] Task: shared-go — design & add the member lifecycle events to
      `messages/member.go` (withdrawal requested/cancelled, pickup accepted, shelter
      accepted, handover completed), each per member, carrying the acting party and
      **no `sosId`**
- [ ] Task: Local — the summarising `sos` event per operation: message types, publish
      last, self-contained payload (who changed, from what to what, resulting strength)
- [ ] Task: shared-go — implement the legacy → current `MemberStatus` value mapping
      as shared code (documented in `types/member.go`, not yet implemented)
- [ ] Task: Local — revive `spejderstatus` as a package to shared-go guidelines:
      consume `patrulje.*.started` (→ `racing`) and `spejder.*.deleted`, legacy-value
      normalisation, year from subject
- [ ] Task: Local — `spejderstatus` schema: `initialTeamId` / `currentTeamId` +
      index on `(year, currentTeamId)`
- [ ] Task: Local — `patrulje.activeMemberCount`: column, recompute from the member
      projection on every membership/status change, both teams on a move
- [ ] Task: Local — rework `GetSpejdere` (`internal/data/member.go:42`) to serve
      `ss.status` instead of inventing `started`/`paid`, and to populate
      `currentTeamId`; check the export path (`cmd/api/export.go:88`) with it
- [ ] Task: Local — in-our-care query (`InOurCare()` statuses + oldest `waiting`)
- [ ] Task: Local — member endpoints: `waiting` request, resume to `racing` (with
      self-carrying dirty-check), status override, move to team — each publishing its
      per-member `spejder` event(s) then one summarising `sos` event
- [ ] Task: Local — whole-team collection as a single command (one `correlationId`,
      per-member events plus the one summarising `sos` event)
- [ ] Task: Local — delete the `patruljemerged` projection, consumer and table; note
      the two unreachable readers left behind (`table/checkgroup/handler.go:48`,
      `table/year/handler.go:48`)
- [ ] Task: Local — add the summarising entry types to `sos_activity` (new `type`
      values, summary in `value`; no schema change)
- [ ] Task: Local — an actor type for the member package that does not import `sos`
- [ ] Task: Local — move the 3-member minimum out of `go/cmd/api/patrulje.go:99`
      into configuration (note three sibling handlers pass 1)
- [ ] Task: Frontend — introduce member rows in `SosTeamCard.vue` (PRD 001 shipped the
      card without them): status, timestamps, acceptor, `Ønsker at udgå` /
      `Fortsætter selv`, move to team
- [ ] Task: Correction interface in `PatruljeView`'s expanded member row + the
      mint-and-close case behind it (task 084)
- [ ] Task: Frontend — **I vores varetægt** counter + `waiting` alarm on the
      nødtelefon list view
- [ ] Task: Frontend — breach warning + pre-commit warning on the `waiting` action,
      with collect-all / move-members actions and a dirty-guarded move dialog; the move
      picker reuses `SosTeamCard`'s live `patrulje:list` filter
- [ ] Task: Frontend — strength / Udgået on the team card, depending on `spejder`
- [ ] Task: Frontend — new `SosActivityLine` entry types
- [ ] Task: Frontend — replace the hardcoded `"ikke startet"` member status tag in
      `PatruljeView.vue:116` with the real status
- [ ] Task: Review check — assert the `spejderstatus` package imports nothing from
      `nathejk.dk/...` (lift-readiness)
- [ ] Task: Confirm the `waiting` alarm threshold with organizers
- [ ] Task: Follow-up (post-stabilisation) — lift `spejderstatus` into
      `shared-go/tables/`
- [ ] Task: Follow-up PRDs — car-acceptance interface and shelter-acceptance
      interface (out of scope here; this PRD fixes the seam they build against)

## 11. Open Questions

### Decisions

Recorded 2026-08-17 so they are not reopened.

- **The merged team is deprecated (2026-08-17).** The model is member-focused: a member
  is **moved** to another team, and a team with no active members is thereby
  discontinued. *Merge* and *split* are retired as concepts, not just as events, and
  `patruljemerged` goes with them. This closes the original question of whether an
  operator may discontinue a team directly: **no** — they retire or move its members.
- **Discontinuation is `activeMemberCount == 0` on a team that started, with no event
  (2026-08-17).** The team
  carries the count; nothing publishes `patrulje.discontinued` and there is no reverse
  event, because a count that is recomputed is reversible for free. This also settles
  the derived-vs-evented question: **neither** — it is a maintained column, read as a
  fact. Strength and discontinuation become the same number.

  **Amended 2026-08-17, during task 066:** the count alone is not the predicate. A team
  that never started has zero racing members too, so `activeMemberCount == 0` conflates
  *left the route* with *never on it* — on the dev data that is 239 abandoned 2025
  signups and **every one of the 310 teams in the current 2026 event**. The predicate is
  `signupStatus = 'STARTED' AND activeMemberCount = 0`. Found by querying the projection
  after building it rather than by reading the specification, which is worth recording:
  it is the same shape of mistake as the finished-team trap in §5, and the specification
  made it twice.
- **Klaner are out, so the seniors question is closed (2026-08-17).** Klan members are
  **not handled through the nødtelefon**, so `spejderstatus` covers spejdere only, keeps
  its name, and publishes on the `spejder` entity. This also settles the `member` vs
  `spejder` noun: the routes read `/api/member/…` because `MemberStatus` is a *member*
  lifecycle, but the events and the live token are `spejder`, and there is no second
  population to generalise for. If seniors ever need a lifecycle it is a separate entity
  with its own token, not a widening of this one.
- **Every member command requires a `sosId`, and the correction path mints its own
  (2026-08-17).** No member's status or team changes without a case explaining why. The
  asymmetry in the earlier draft — required for withdrawal, optional for the override and
  the move — is removed. Where the operator has no case, the backend **creates one and
  closes it immediately**, purely as the record. That keeps the timeline uniform, keeps the
  open-case list clean, makes overrides countable for §9's metric, and surfaces corrections
  in the patrol page's existing **Kontakt med nødtelefon** card.
- **The correction interface lives on the patrol page, not the case card (2026-08-17).**
  `PatruljeView.vue`'s member list, in the expanded row — which today is a placeholder
  rendering `{{ data }}`. Being a different screen from the case card is a stronger
  separation than a visually-distinct button beside the normal actions, and it matches what
  the two surfaces are for: the case card is the call in progress, the patrol page is the
  record of a patrol.
- **There is no exception mechanism (2026-08-17).** Breaches of the 3-member
  requirement are **not handled** — we record what happened. Nothing grants, approves or
  resolves a short-handed patrol: no exception event, no reason text, no acting-operator
  attribution for a decision that was never asked for, and no handled/unhandled
  distinction. `activeMemberCount` states the strength and the case timeline states how
  the team got there, which is the entire record. Removed accordingly: the
  `POST /api/sos/:id/team/:teamId/exception` endpoint, the breach-handling storage, the
  "settled note" UI, and two success metrics.
- **Event shape: N member events plus one case event (2026-08-17).** A member-changing
  operation publishes **one event on the `spejder` entity per updated member**, then
  **one event on the `sos` entity summarising the whole operation** — uniformly, with no
  special case for a single member. The member events carry **no `sosId`**, so the car
  and shelter interfaces can publish the same kind of event without knowing about cases;
  the case event is the timeline entry, published last so the changes it describes are
  already in the log, and on a subject `table/sos/consumer.go` already consumes. `sosId`
  is a parameter of the command, not a field on the member event. This closes the
  timeline-writing question and the whole-team-collection N-vs-1 problem in one rule —
  see §8.
- **Move targets are any racing patrol from the same year (2026-08-17).** No proximity,
  liga or size rule. In practice the destination is agreed by **crew in the field**, who
  contact a nearby patrol directly; the nødtelefon records the outcome rather than
  deciding it, so a backend rule could only reject a decision already made on the
  ground. Members are moved **one at a time** — usually all to the same patrol, but two
  survivors may go to two different ones, and the flow must allow it. Consequence: **no
  candidate query and no `reassign-candidates` endpoint** — the picker filters the live
  `patrulje:list` the SPA already holds, as team association does (§8).
- **Everything checkgroup-related is out of scope (2026-08-17).** The checkpoint
  overview, `GetDiscontinuedTeamIDs`, `Checkgroup.Discontinued` and `PostList.vue`'s
  *Mangler* count are not this PRD's work. `activeMemberCount` makes the fact
  available; consuming it belongs to whoever revisits that view. This removes Phase D
  and the organizer verification task, and with them the visible-change risk to an
  existing screen.
- **`finished` has no producer yet (2026-08-17).** There is no finish flow in hq at all,
  and this PRD does not add one; it will come later. So the finished-vs-discontinued
  collision cannot be exercised today (members stay `racing` past the end, so no
  finishing team's count drains), and no predicate is written against it now. The trap
  is recorded in §5 and below for the work that introduces the producer.
- **`GetSpejdere` must stop inventing statuses (2026-08-17).** Status is set by
  messages. The `IF(ss.status IS NULL, IF(ps.startedUts > 0, 'started', 'paid'), …)`
  fallback at `internal/data/member.go:42` is reworked to serve the projected value —
  see §6 and §8.

### Still open

None of these are blocking: they are configuration, sequencing and questions the car and
shelter PRDs will answer. All four originally-blocking questions are decided above.

- **How do downstream transitions reach a case timeline — or do they?** With the event
  shape decided, the car and shelter interfaces publish `spejder` events and no `sos`
  event, because they know no case. So their transitions update the projection and the
  counters but land on no timeline. Options: merge them at read time from the case's
  associated teams (self-contained, ambiguous when a member has two open cases), or
  accept that the timeline records only what the nødtelefon did and read the member's
  current status from the member row instead. The second is cheaper and arguably more
  honest; decide when the car PRD is written, since it is the one seam decision those
  PRDs may want to influence.
- **Where does the required minimum live?** 3 is the requirement for patrols, but it
- **When the finish producer arrives, what stops a finishing team reading as udgået?**
  Not this PRD's to solve, but it is this PRD's number that will be misread. Options:
  the finish sets a team-level fact that outranks the count, or consumers ask
  "`activeMemberCount == 0` **and** not finished". Worth a line in whichever PRD adds
  the producer.
- **Correlating downstream events to a case** is covered by the first question above;
  the `correlationId`-propagation option is the version that needs every downstream
  producer to cooperate, and the decided event shape does not require it.
- **Where does the required minimum live?** 3 is the requirement for patrols, but it
  should be configurable rather than compiled in (currently hardcoded at
  `go/cmd/api/patrulje.go:99`, with three sibling handlers passing 1). Year
  configuration seems the natural home since it is a rule of the event. Confirm, and
  confirm whether it has ever differed between years or ligas.
- **`waiting` alarm threshold:** how long may a member be `waiting` before the
  dashboard warns? Their patrol is blocked for the whole duration, so this is the one
  number operators will feel. Fixed config value or per-severity?
- **Where the in-our-care count lives:** the nødtelefon list view (proposal), the HQ
  dashboard (`/api/home`), or both? It is an event-wide number rather than a SOS one.
- **Sequencing strictness:** should the API enforce the documented order (reject
  `racing` → `sheltered`, say) or accept any `Valid()` status and let the timeline
  show what happened? Strictness protects the data; leniency matters at 3am when the
  real world skipped a step. Note the override endpoint is where this bites — and now that
  the override is the designated out-of-sync repair tool, leniency is the likelier answer:
  its whole purpose is recording a reality that did not follow the diagram.
- **Seniors as well as spejdere:** closed — klaner are not handled through the nødtelefon,
  and with them the `member` vs `spejder` noun. See Decisions above.
- **The Udgået page.** `Navigation.vue:160` links to `/ude`, which has no route, and
  shared-go's `spejder.GetInactive` is commented out waiting for exactly this
  projection — with a bug noted in its own doc comment ("two placeholders but passes
  four args"). Out of scope here (§4), but it becomes buildable the moment this ships:
  its own task, or its own small PRD?
- **Producer ownership of `registered` / `seated`:** which flows set them (signup +
  orders?) — not needed by this feature, since the panel only sees members who have
  started, but worth knowing before `spejderstatus` is treated as complete. `racing`
  is settled: derived from `NathejkTeamStarted`.
- **Existing `patruljemerged` data:** closed 2026-08-17 (task 069) — the table held **zero
  rows** when it was dropped, so there is no historical data to migrate or drop. The
  question was moot all along.
- **What happens to a moved member's finish?** A survivor moved into another
  patrol is still `racing` and self-carrying, so they can still finish — but with a
  team that is not the one they started with (`initialTeamId` ≠ `currentTeamId`). Does
  that affect diplomas, results, or how the finish is recorded? Deferred with the finish
  producer.
- **May a member be moved to a klan?** Closed: no. Klaner are not handled through the
  nødtelefon at all (Decisions above), so a move target is always a patrol.
- **Does the 3-member requirement apply to klaner?** Moot for this feature for the same
  reason; recorded only because the `MinMemberCount: 1` in the klan, badut and mail
  handlers will still be sitting there when task 074 moves the patrol minimum into
  configuration.
- **Deferred to the car and shelter PRDs** (recorded here only so they are not lost):
  which product becomes the car interface and which the shelter one; who performs the
  final handover (`released` / `reunited`) — `reunited` happens at the finish line
  rather than the shelter, so it may not belong to either; and car dispatch, including
  whether the `waiting`-too-long alarm should *request* a pickup. The only piece that
  may need to sit in the nødtelefon interface is that request, so it is worth a
  decision when the car PRD is written rather than now.
