# PRD 001 — Nødtelefon / SOS case management

**Status:** draft
**Author:** agent session (recreating legacy feature)
**Created:** 2026-07-29
**Last updated:** 2026-08-07
**Status note:** member lifecycle settled against shared-go
`v0.0.0-20260807180020-5ac2603c60ba`
**Target users:** organizer (HQ emergency-phone operators / nødtelefonvagter)

---

## 1. Summary

Recreate the legacy **Nødtelefon / SOS** feature in the current platform: a
lightweight "dispatch desk" that lets HQ operators log and manage emergency
calls that come in on the event's emergency phone (nødtelefon). Each call
becomes an SOS *case* (sag) with a headline, description, priority, assignee,
an activity timeline, and one or more associated patrols whose members can be
tracked. The interface's primary member actions are recording that a member
**wants to leave the race** (status `waiting`), letting them **carry on** if they
change their mind, and **reassigning** a member to another team. Once a member
steps into one of our cars they are no longer self-carrying, and from that point
they are handed along a chain of custody — car, then shelter — where each step is
confirmed by the party *receiving* the member, in their own interface, not here.
This UI shows where every member is; it sets only the steps where the member is
still on their own feet. A patrol is **required to have at least 3 racing
members**; when a member leaving would drop it below that, the interface surfaces
the breach and the operator on the nødtelefon handles it — collecting the whole
team, reassigning the survivors, or granting an explicit exception. A team with no
active members left is **discontinued**.

## 2. Problem & Motivation

> **Terminology:** what `shared-go/types/member.go` calls *trailside assistance*
> **is this interface**. The lifecycle documented there — a member leaving the
> route, being collected, sheltered at HQ and handed over — is the workflow this
> PRD builds the *first step and the overview* for, and that documentation is the
> canonical description of the member half of this feature. Read the two
> together. Note that only the `waiting` transition is produced here; see §3.

- **What problem does this solve?** During the event, participants in the field
  call the emergency phone for anything from a twisted ankle to a lost patrol.
  Operators need a single place to record each incident, track its priority and
  who is handling it, tie it to the affected patrol(s) and members, and keep a
  timestamped log of everything that happened — so a shift handover or a later
  review is possible and nothing is lost on paper notes.
- **Why now?** The feature existed and was used in the legacy platform (found in
  `_go/` and `_vue/`) but was not carried over when the platform was rebuilt.
  The current platform (`go/` + `vue/`) has no incident-handling capability at
  all, so operators currently have no digital tool for the emergency phone.
- **Evidence.** The legacy implementation is fully built out:
  - Frontend: `_vue/src/views/Sos/List.vue`, `_vue/src/views/Sos/View.vue`,
    the `dims` Vuex store (`_vue/src/store/dims.js`), the "Kontakt med
    nødtelefon" card on `_vue/src/views/Patrulje.vue`, and the `/sos` nav entry
    in `_vue/src/components/Navigation.vue`.
  - Backend: `_go/cmd/api/sos-routes.go`, `_go/cmd/api/soscmd.go`,
    `_go/nathejk/messages/sos.go`, `_go/nathejk/aggregate/sos/sos.go`,
    `_go/nathejk/table/sos.go`, `_go/nathejk/table/sosassoc.go`,
    `_go/nathejk/table/sos.sql`, `_go/internal/data/sos.go`.

## 3. Goals

- Operators can create an SOS case from an incoming call with a headline and a
  free-text description in a few seconds.
- Each case has a mutable **status** (open/closed), **priority/severity**
  (green/yellow/red) and an **assignee** — an organisation **section**
  (from the Organisation page) responsible for handling it.
- Every change to a case is captured on an append-only **activity timeline**
  (comments, status changes, priority/assignee changes, team association,
  member-status changes, member moves).
- Operators can **associate one or more patrols** with a case and see the
  patrol's members. This interface owns the transitions where the member is still
  **self-carrying** — on their own feet, whether walking or stopped by the
  trailside:
  - `racing` → `waiting`: the member wants to leave the race and awaits
    collection.
  - `waiting` → `racing`: they changed their mind and carry on under their own
    steam. Legitimate and expected, not a correction.
  Everything after that is a **handover accepted by the receiving party in their
  own interface**, and from the first of them the member is no longer
  self-carrying:
  - `waiting` → `transit`: the car accepts the member into the vehicle. **This is
    the point of no return** — once we are carrying them, there is no way back
    onto the route.
  - `transit` → `sheltered`: whoever receives them at the shelter accepts them
    in.
  - `sheltered` → `released` | `reunited`: the final handover to a guardian or
    back to their own team.
  Custody is therefore always *confirmed by the receiver*, never claimed by the
  party letting go — which is what makes the chain trustworthy.
- Operators can **see** the whole chain for every member on an associated patrol
  (current status, when it changed, who accepted them), even though only the
  first step is set here. Plus an override to correct out-of-sync data.
- **A patrol is required to have at least 3 racing members.** This is a rule of
  the event, not a preference — but the *handling* of a breach belongs to the
  operator on the nødtelefon, who is on the phone with the people involved. The
  interface therefore makes the breach unmissable and gives the operator the means
  to deal with it, without blocking the transition or acting on its own:
  1. The member carries on (`waiting` → `racing`) and the requirement is met
     again.
  2. **Collect the whole team** — the remaining members go `waiting` too and leave
     together.
  3. **Reassign the remaining members** to another patrol that can take them, if
     one is available.
  4. **Grant an exception** — the operator judges this particular patrol can
     continue short-handed. Permitted, but it is a deliberate departure from the
     rule and is recorded as one, not a silent dismissal.
  Options 2 and 3 both leave the original team with no active members, so it
  becomes discontinued — which is precisely what the legacy `merged`/`splited`
  events were encoding, and why the old `patruljemerged` table pointed at a
  `parentTeamId`: option 3 *is* a merge, expressed as member reassignment.
- **A team with no active members is discontinued.** "Active" here is `racing` —
  the same equivalence the legacy value mapping makes (`active` → `racing`). This
  is a separate threshold from the 3-member requirement: below 3 the team is in
  breach and needs handling; at zero it is out of the race.
- Operators can see, at a glance, how many members are currently **in our care**
  (`waiting`/`transit`/`sheltered`) — the count that must reach zero before the
  organisers can go home — and which members have been `waiting` too long, since
  a waiting member blocks their whole patrol.
- Operators can **reassign** a member to another team. Together with member
  status this replaces the old merge/split model: a team with no members left
  racing is **discontinued**.
- The patrol detail page surfaces the SOS cases the patrol is involved in.
- Cases are scoped to the current event year, matching the rest of the platform.

## 4. Non-Goals

- The car/driver interface and the shelter reception interface. **These will be
  specified in their own PRDs.** The `transit` and `sheltered` transitions are
  accepted by those parties in their own products; what this PRD owns is the
  `waiting` transition, the read-only overview of the rest, and the override. See
  Dependencies & risks for the seam this PRD fixes so those PRDs can be written
  against it — and note the withdrawal route does not complete end to end until
  they ship.
- Dispatching cars. Deciding which car goes where, and telling it to, is not part
  of this interface — it belongs with the car interface (see Open Questions for the
  one piece that may need to sit here).
- Public/participant-facing UI. This is an internal HQ tool only.
- Automated telephony/IVR integration — calls are still answered on a physical
  phone; this only records them.
- Position-request SMS / GPS location of members. The legacy feature could text
  a member a link to share their position (`member.positionsms.*`,
  `/api/sos/sms`); this is **not** ported.
- Real-time GPS map tracking of members.
- Rich-text or attachment support in comments (legacy was plain text; keep it).
- Migrating historical legacy SOS data. Cases start fresh for the current year.
- Merge/split of patrols. The terms *merged* and *splitted* are retired from the
  domain vocabulary and the legacy `patrulje.merged` / `patrulje.splited` events
  are **not** ported. They were an encoding of what §6 now models directly: a
  member is **reassigned** to another team, and a team with nobody racing is
  **discontinued**. The case that produced a legacy merge — a patrol dropping
  below minimum strength and its survivors joining another patrol — is a
  first-class flow in this PRD. The replacement must reproduce the observable
  behaviour of the old events, including their reversibility (legacy `.splited`
  deleted the `patruljemerged` row and thereby un-discontinued the team).

## 5. User Stories & Scenarios

- As an **operator**, I want to log an incoming emergency call as a new case so
  that it is tracked and not forgotten.
- As an **operator**, I want to set a case's priority and assign it to the
  responsible organisation section so the right team picks it up and urgent
  cases stand out.
- As an **operator**, I want to add comments over time so the case history is a
  complete, timestamped record for shift handover.
- As an **operator**, I want to record that a member wants to leave the race and
  is waiting to be collected (`waiting`), so that a car can be sent and the
  member is counted as being in our care from that moment.
- As an **operator**, I want to put a member who has changed their mind back into
  the race (`waiting` → `racing`) as long as no car has collected them yet, so a
  scout who gets their breath back can carry on and no car is sent needlessly.
- As an **operator**, I want to be told immediately when the member leaving would
  put their patrol below the required 3 racing members, and to have the means to
  handle it — collect them all, place the survivors with another patrol, or grant
  an exception — because I am the one on the phone and the rule needs a person to
  apply it.
- As an **operator**, I want my handling recorded, including an exception I grant,
  so the next shift and the post-event review see how the breach was dealt with.
- As an **operator**, I want to watch the member's progress — accepted into a
  car, accepted at the shelter, handed over — without having to update it myself,
  because the people receiving the member record it as it happens.
- As an **operator**, I want to see how many members are in our care right now,
  and be warned when somebody has been `waiting` too long — their patrol cannot
  continue until they are collected.
- As an **operator**, I want to **override** a member's status when it is wrong,
  to correct out-of-sync data.
- As an **operator**, I want to reassign a member from one team to another so
  that, e.g., a scout who continues with a different patrol is tracked
  correctly; when a team is left with nobody racing it is automatically
  considered **discontinued**.
- As an **operator viewing a patrol**, I want to see any SOS cases the patrol is
  part of so I have context.

### Primary happy path

1. Emergency phone rings. Operator opens **Nødtelefon** in the nav and clicks
   **Ny sag**.
2. They type a headline ("Forstuvet ankel ved post 4") and a short description,
   and save. The case appears under **Åbne sager**.
3. They set priority to **yellow** and assign it to the **Samarit** section.
4. They search and associate the caller's patrol; the patrol's members appear.
5. They mark the injured scout as **`waiting`** — the patrol is now blocked until
   a car reaches them, and the scout counts as in our care from this moment. The
   operator does nothing further to the status: when a car reaches them the driver
   accepts them aboard (**`transit`**), and on arrival the shelter accepts them
   (**`sheltered`**) — both appear on this case's timeline as they happen. A parent
   collects them that night: **`released`**. (Had their patrol reached the finish
   first, it would have been **`reunited`** instead — never `finished`, which is
   reserved for walking the route.)
6. As things develop they add comments. When resolved, they **Luk sag**; it
   moves to **Lukkede sager**. If it flares up again they **Genåbn sag**.

### Edge cases & error scenarios

- Creating a case with an empty headline or description is rejected.
- Closing an already-closed case (or reopening an open one) is a no-op, not an
  error.
- Associating a team already associated is idempotent (`INSERT IGNORE` in
  legacy `sosassoc`).
- Two operators editing the same case concurrently — last write wins per field;
  the timeline still records both actions in order.
- Reassigning a member *back* to a discontinued team makes it non-discontinued
  again (parity with the legacy `.splited` undo). Discontinuation must therefore
  be re-evaluated on every membership/status change, not set once.
- Retiring the last racing member of a team discontinues it; the case timeline
  records both the member change and the resulting team discontinuation.
- A team already below the requirement when a case is opened (two members left
  earlier in the event) is in breach from the start; the warning reflects the team's
  current strength, not only the transition that caused it.
- Reassigning the survivors when **no** target patrol is available leaves
  collecting them or granting an exception. The UI must not offer a reassignment
  flow that dead-ends with no candidates.
- An exception is granted per team, per breach — if strength changes again later
  (another member leaves), that is a new breach needing its own handling. An
  exception is not a permanent waiver.
- If the member resumes after the operator has already collected the rest of the
  team, the team is *not* automatically restored — those members are `waiting` and
  only they (or a car) can change that. Resolving one member does not unwind
  actions taken for others.
- A member in `waiting` who decides to carry on returns to `racing`; the case
  stays open and the timeline records both moves. This is valid only while they
  are still self-carrying — if a car has already accepted them (`transit`), the
  resume must be **rejected**, not silently applied. Race condition to handle
  explicitly: operator presses resume at the same moment the driver accepts the
  member aboard — the acceptance wins, since it reflects the member physically
  being in a car.
- A member who left the trail must never end up `finished`. `MemberStatus.CanFinish()`
  is true only for `racing`, so the finish-line flow cannot promote a `reunited`
  member — the SOS UI must not offer `finished` as an override target either. Note
  a member who resumed (`waiting` → `racing`) *can* finish, correctly: they walked
  the rest of the route themselves.
- Only cases for the current event year are shown (legacy filters messages by
  `msg.Time().Year()`; the current platform scopes by year slug).

## 6. Requirements

### Functional

- [ ] Create an SOS case with headline + description (`POST /api/sos`).
- [ ] Edit a case headline (`PATCH /api/sos/:id`).
- [ ] Close and reopen a case (`PATCH /api/sos/:id`, `status` field).
- [ ] Add a plain-text comment to a case (`POST /api/sos/:id/comment`).
- [ ] Set priority/severity green|yellow|red (`PATCH /api/sos/:id`).
- [ ] Assign a case to an organisation **section** (`PATCH /api/sos/:id`).
      The list of assignable sections comes from the sections defined on the
      Organisation page — possibly a curated subset rather than every section
      (see Open Questions).
- [ ] Associate / disassociate a patrol with a case
      (`PUT` / `DELETE /api/sos/:id/team/:teamId`). Only patrols can be associated
      with a case — clans (klaner) cannot.
- [ ] Mark a member as **`waiting`** — wanting to leave the race and awaiting
      collection. Recorded on the case timeline.
- [ ] Return a `waiting` member to **`racing`** when they choose to carry on
      under their own steam. Permitted **only** from `waiting`: the member must
      still be self-carrying, so the API must reject it once the status has moved
      to `transit` or beyond, with a message the operator can act on ("already
      collected") rather than a generic conflict.
- [ ] **Display** the rest of the chain read-only: current status, when it
      changed, and who accepted the member (`transit` from the car,
      `sheltered` from the shelter, `released`/`reunited` at handover). These
      arrive as events from other interfaces; this UI must never be the thing
      that sets them in normal operation.
- [ ] Show externally-produced member transitions **on the case timeline** of any
      case the member's team is associated with, so an operator watching a case
      sees the pickup and arrival without refreshing another screen. How those
      events are correlated to a case is a technical Open Question.
- [ ] Override a member's status to any valid value to correct out-of-sync data,
      excluding `finished` (reserved for walking the route — `CanFinish()` is
      true only for `racing`). This is the escape hatch for when a handover was
      not recorded, and it should be visibly distinct from the normal `waiting`
      action so it is not used as a shortcut for work another interface owns.
- [ ] Show a live **in our care** count (`MemberStatus.InOurCare()` — `waiting`,
      `transit`, `sheltered`) across the current year. This is the number that
      must reach zero before the organisers can go home, so it belongs on screen
      permanently, not per-case.
- [ ] Flag members who have been `waiting` beyond a threshold. A waiting member
      blocks their entire patrol, so this is the one state worth an alarm; the
      threshold is a config value (see Open Questions).
- [ ] **Warn, at the moment of setting `waiting`, when it would put the team below
      the required 3 `racing` members**, naming the resulting count. The operator
      must see the consequence *before* committing, since it changes the
      conversation they are having on the phone. The warning does not block the
      transition — the member is leaving whether or not the team is compliant — but
      it does require the operator to say how the breach is being handled.
- [ ] Show which teams on a case are below the requirement, and keep showing it
      while it is true, so an operator taking over a shift can see the state of
      play. A team in breach with no recorded handling is the one state this tool
      must never hide.
- [ ] **Collect the whole team** as one action: every remaining `racing` member on
      the team goes to `waiting` together, recorded as a single timeline entry
      rather than one per member. Operators are on the phone; three separate clicks
      invite two of them being forgotten.
- [ ] **Reassign the remaining members** to another patrol, with candidate targets
      offered by the backend (availability rules are an Open Question). This is the
      modern form of the legacy patrol merge.
- [ ] **Grant an exception** allowing a patrol to continue below 3, recorded on the
      timeline as a deliberate decision with the acting operator. This is the
      pressure valve that keeps the requirement honest rather than routinely
      ignored: it must be easy to do and impossible to do accidentally.
- [ ] The required minimum is **3** for patrols; it must be a configured value
      rather than a literal in code (see Open Questions for where it lives).
- [ ] Member statuses shown in the SOS panel originate from the platform, not from
      this UI: `racing` is derived from the team-started event
      (`NathejkTeamStarted`), and the later statuses come from the car and shelter
      interfaces. This interface writes only `waiting`, the return to `racing`, and
      the override.
- [ ] Team strength is the count of members whose `currentTeamId` is the team and
      whose status is `racing` — matching `MemberStatusRacing`'s documentation that
      it is "the only state in which a member counts towards their team's strength
      on the route". `waiting` members do not count.
- [ ] Reassign a member to another team (updates the member's `currentTeamId`,
      leaving `initialTeamId` untouched), recorded on the case timeline.
- [ ] A team with **no active (`racing`) members** is **discontinued** (udgået). `racing` is the
      only status that counts towards a team's strength on the route, per
      `shared-go/types/member.go`. A team that *finished* must not be swept up by
      this — see Open Questions for the exact predicate. Discontinuation is
      re-derived on every membership or status change and is **reversible**.
- [ ] Discontinuation must be observable through the existing
      `discontinuedTeamIds` surface (`data.TeamModel.GetDiscontinuedTeamIDs` /
      `patrulje.querier.GetDiscontinuedTeamIDs`, consumed by the checkgroup
      status view), so the checkpoint overview keeps working exactly as it did
      when it was fed by `patruljemerged`.
- [ ] List cases grouped into open / closed with columns: headline, created,
      last activity, priority, assignee (`GET /api/sos`).
- [ ] View a single case with its full activity timeline and associated teams
      (`GET /api/sos/:id`).
- [ ] Show SOS cases associated with a patrol on that patrol's detail page
      (query cases by team, as in legacy `data.SosModel.GetByTeam`).
- [ ] All cases and events are scoped to the current event year.

### Non-Functional

- **Consistency with platform:** REST + JSON via `app.*` helpers; MySQL
  projections rebuilt from JetStream on startup; frontend via the `http` module
  (`@/plugins/axios`) and PrimeVue Aura. No Bootstrap, no `vue-good-table`, no
  `b-popover`
  (all legacy-only).
- **OpenAPI:** every new/changed endpoint must carry OpenAPI annotations
  (repo `.rules` requirement).
- **Auth:** operator-only; behind the existing JWT cookie auth like the rest of
  `/api`.
- **Timeliness / freshness:** an operator's view reflects other operators' and
  other interfaces' changes within ~1 second, without the operator refreshing.
  Delivered by **PRD 004 (Live updates for the SPA)** — SSE push plus a shared
  client cache; see §8 "Live updates & perceived speed" for what this feature
  relies on.
- **Perceived speed — a first-class requirement, not polish.** This screen is used
  continuously through the event, often while talking on the phone, so:
  - Navigating away and back shows the current state **instantly** — no spinner, no
    refetch-then-render. Returning to a screen must never cost seconds.
  - Operator actions (comment, status, patch) feel immediate: the UI updates
    optimistically and reconciles with the server afterwards.
  - A cold load (hard refresh, new shift, new machine) paints usable content in
    well under a second, with no request waterfall.
  - Table state — scroll position, sort, filters — survives navigation. Losing an
    operator's place in a long list mid-call is a real cost.
- **Localization:** Danish UI text and `da-DK` date formatting, matching the
  rest of the SPA.
- **Auditability:** the activity timeline is append-only; capturing the acting
  user (`createdByUserId`) on each event is required (legacy carried `UserID`
  in message bodies but the API did not always populate it — this PRD requires
  it be populated from the authenticated user).

## 7. UX / UI Notes

New frontend surface (all inside the `ui` SPA, `vue/src`):

- **Nav entry:** add "Nødtelefon" (icon `fa-headset`) to `items` in
  `vue/src/components/Navigation.vue`, routing to a new `sos` route.
- **List view — `vue/src/views/SosListView.vue`** (`/sos`): a PrimeVue
  `DataTable` (Aura preset, as configured in `vue/src/main.ts`) with two groups,
  **Åbne sager** and **Lukkede
  sager**; columns Overskrift, Oprettet, Sidst opdateret, Prioritet, Tildelt; a
  **Ny sag** button; row click opens the detail view. Empty state: "Ingen
  nødråb fundet".
- **Detail view — `vue/src/views/SosView.vue`** (`/sos/:id`, `props: true`,
  plus a `/sos/new` route for creation):
  - Editable headline with pencil affordance.
  - Summary card: status badge (Åben/Afsluttet), created timestamp, priority,
    assignee.
  - **Activity timeline** rendering each activity type (comment, close, reopen,
    severity, assign, associate, disassociate, memberstatus, member-reassign,
    team-discontinued) with
    an icon and Danish label — port the legacy
    `ActivityLine` component to `vue/src/components/SosActivityLine.vue`.
  - Comment composer (headline required only when creating a new case).
  - Actions: Luk sag / Genåbn sag, Tilføj kommentar.
  - Right column cards:
    - **Tilknyttede patruljer:** team picker + per-team member list, showing the
      team's **strength** (racing members) beside its name, an **Under styrke**
      warning when it is below the required 3, and an **Udgået** badge when
      discontinued. Each member row shows the
      current status with its timestamp and, where known, who accepted them. Row
      actions depend on status: `racing` offers **Ønsker at udgå** (→ `waiting`);
      `waiting` offers **Fortsætter selv** (→ `racing`) as a normal, prominent
      action — not buried in an override menu, since a scout getting their breath
      back is an ordinary outcome and saves a car being sent. From `transit`
      onwards the row is **read-only**: it reflects what the car and shelter have
      recorded and offers no buttons to advance or reverse them. Secondary
      actions: a visibly-separate status override (for when a handover went
      unrecorded) and **Flyt til anden patrulje** (reassign). Members `waiting`
      past the threshold are highlighted.
    - **Below the 3-member requirement:** when a team on the case has fewer than 3
      racing members, the card shows a prominent warning stating the current
      strength and offering the three ways to handle it — **Hent hele patruljen**
      (all remaining racing members → `waiting`, one action), **Flyt de resterende**
      (reassign survivors, with backend-supplied candidate patrols) and **Tillad
      undtagelse** (grant an exception, requiring a short reason). Confirming
      `Ønsker at udgå` for a member whose departure causes the breach warns *before*
      committing, naming the resulting strength ("Patruljen har kun 2 tilbage"), and
      offers the same three actions plus proceeding and handling it next. Once
      handled, the warning becomes a settled note recording what was done and by
      whom — it stops demanding attention but does not disappear.
    - **Prioritet** select (green/yellow/red) and **Tildelt** select. The
      **Tildelt** (assignee) options are organisation sections loaded from the
      backend (the Organisation sections, optionally filtered to an assignable
      subset) — shown by section label, stored by section slug. Do **not**
      hardcode the legacy assignee list.
  - Legacy option lists to reuse (Danish): severities `green|yellow|red`.
    Member statuses come from `shared-go/types.MemberStatus` — `registered`,
    `seated`, `racing`, `finished`, `waiting`, `transit`, `sheltered`,
    `reunited`, `released` — served to the frontend from the backend, never
    hardcoded in the view, and rendered with Danish labels. `finished` is
    display-only here and is never offered as an action. This supersedes the
    legacy `active/waiting/transit/emergency/hq/out` list, whose mapping onto the
    new values is documented in `shared-go/types/member.go`. (The legacy
    hardcoded assignee list — guide/samarit/rover/… — is **replaced** by
    organisation sections.) Severities should be confirmed with organizers (see
    Open Questions).
- **List view header:** a permanent **I vores varetægt** counter
  (`InOurCare()`: waiting + transit + sheltered) with a breakdown per status, and
  a warning state when any member has been `waiting` past the threshold. This is
  the organisers' go-home number, so it should be visible without opening a case.
- **Patrol detail:** add a "Kontakt med nødtelefon" card to
  `vue/src/views/PatruljeView.vue` listing the patrol's SOS cases with created
  date, headline and open/closed badge; clicking navigates to the case.
- **State:** a Pinia store `vue/src/stores/sos.ts` holding the case list and
  current case, with actions wrapping the REST calls via the `http` module
  (replacing the legacy Vuex `dims` actions). Note the rest of the SPA keeps
  shared state in composables with module-level `ref()`
  (`vue/src/composables/globalstate.ts`) and has only one real store
  (`counter.ts`), so a composable `vue/src/composables/sos.ts` is the more
  idiomatic choice here — pick one and be consistent (see Open Questions).

## 8. Technical Considerations

### Frontend (Vue 3 / TS)

- New views `SosListView.vue`, `SosView.vue`; new component
  `SosActivityLine.vue`; new store/composable `sos.ts` (TypeScript — not the
  legacy `*.store.js` naming); routes in
  `vue/src/router/index.ts`; nav item in `Navigation.vue`.
- Replace legacy libs: use PrimeVue `DataTable`/`Select`/`Textarea`/`Badge`
  (auto-imported via `unplugin-vue-components`) on the existing **Aura** preset
  (`vue/src/main.ts:6,33`) plus Tailwind; use PrimeVue overlay/popover for the
  per-team and per-member action menus instead of `b-popover`. Do not introduce a
  second theme preset — the new UI uses Aura like the rest of the SPA.
- All requests go through the `http` module (`import { http } from
  '@/plugins/axios'`) to relative `/api/...` paths, as in
  `vue/src/views/PatruljeView.vue:5,20`. `http` is the project's fetch wrapper;
  no component or store may `import axios` directly. It already sets `baseURL:
  '/api/'` and attaches the `X-YearSlug` header via a request interceptor, so
  year scoping comes for free — SOS endpoints must **not** take the year as a
  path or query parameter.
- Live updates and caching: provided by PRD 004 — the SOS views compose its
  `useLiveResource` primitive rather than owning a bespoke store or transport.

### Live updates & perceived speed

**Delivered by PRD 004 — "Live updates for the SPA"** (`roadmap/prd/004-live-updates-spa.md`).
That capability is deliberately entity-agnostic and platform-wide: one SSE stream
of `entity.changed` signals, one reusable client cache primitive, and a generic
consumer decorator so every page — betalinger, patruljer, klaner, poster — becomes
live without per-page plumbing. This feature is its **first adopter and proving
ground**, chosen because it is the screen where staleness does the most damage.

Freshness and instant navigation are separate problems, and PRD 004 covers both:
no transport choice fixes navigation on its own, since a websocket app that
refetches on every mount still feels slow.

What this PRD relies on, and must not reimplement:

- `GET /api/stream` for change signals — **not** a SOS-specific stream endpoint.
- The `useLiveResource`-style cache primitive for the case list, the open case and
  the in-our-care counter. No bespoke SOS store.
- The `notify(hub, consumer)` decorator applied to the `sos`, `sos_team`,
  `sos_activity` and `spejderstatus` consumers, so their changes signal like any
  other entity's.
- The connection-state indicator.

SOS-specific consequences worth stating here:

- **The in-our-care counter and the `waiting` alarm are the reason freshness
  matters.** At 2–5s polling they are always slightly wrong, which is why polling
  is acceptable as PRD 004's day-one transport but not as the end state for this
  screen.
- **No staleness UI is needed — and that is a real safety property here.** The API
  does not serve until its projections are fully caught up (**PRD 005**, surfaced in
  PRD 004 §8), so the in-our-care count and patrol strengths are either correct or
  the screen cannot load at all. On most pages that distinction is cosmetic; on a
  dispatch desk, a plausible-but-wrong count of the people we are responsible for is
  the worst failure this tool could have. The operator-visible consequence is
  narrower: during an API restart the screen reports that it cannot reach the
  server, rather than showing numbers it cannot stand behind.
- **Timeline ordering.** The case timeline must tolerate signals for transitions
  this interface did not cause, arriving in any order, including for members on
  teams never associated with a case (see the withdrawal-chain seam above).
- **Optimistic writes** for comments, patches and member actions: an operator on
  the phone must never wait for a round trip.
- **Detail view seeded from the list row** (headline, status, severity, assignee
  are already known), with the timeline streaming in after.
- The case list is a heavily used table, so **scroll, sort and filter state must
  survive navigation** — `KeepAlive`, per PRD 004.

Legacy note: the `dims` websocket that served this screen before is dissected in
PRD 004 §2.1. Read it before reviving anything from `_go/cmd/api/dims.go` or
`_vue/src/store/dims.js` — the difficulty there was a duplicate in-memory read
model and a hand-rolled client cache, not the transport, so porting its shape
would reintroduce the problem under a new protocol.

### BFF (Go) — where the code lives

Entities are being lifted one by one out of `go/nathejk/table/` into
`github.com/nathejk/shared-go/tables/`, and shared-go is where they will all
end up eventually. Current state of that migration:

- **Already in shared-go/tables:** `crewmember`, `klan`, `order`, `patrulje`,
  `payment`, `product`, `section`, `senior`, `signup`, `spejder`.
- **Still local in `go/nathejk/table/`:** `checkgroup`, `checkpersonnel`,
  `checkpoint`, `lok`, `patruljemerged`, `patruljestatus`, `personnel`,
  `pincode`, `registrant`, `scan`, `spejderstatus`, `year` — plus
  still-live local duplicates of `order`, `patrulje`, `payment`, `senior` and
  `spejder` that the migration has not finished retiring (`internal/data/models.go`
  currently imports `patrulje`, `order` and `payment` from **local** while taking
  `klan`, `senior`, `spejder`, `section` and `crewmember` from **shared-go**).

Decisions for this feature:

- **Build the SOS package locally, in `go/nathejk/table/sos/`, but written to
  shared-go's guidelines so it can be lifted unchanged.** New entities are
  developed locally where the dev loop is fast (`docker compose up` +
  `inotifywait` hot-reload, no release-and-bump cycle), and lifted to
  `shared-go/tables/sos/` once the schema and events are stable. The same applies
  to the member-status/membership projection.
- **"Written to shared-go's guidelines" means, concretely:**
  - Follow the shared-go layout, not the varying local one (`lok` has no
    `commands.go`; `order` uses `commander.go`/`querier.go`/`saga.go`). The
    newest migration, `tables/signup`, is the reference: `table.go`,
    `consumer.go`, `querier.go`, `commands.go`, `repository.go`,
    `interfaces.go`, `table.sql`.
  - Respect the dependency-inversion rule documented in
    `shared-go/tables/interfaces.go`: the package declares what it needs from
    the application in its own `interfaces.go`, satisfied structurally by the
    consuming service, and never imports application code. This matters most
    for the acting-user context needed for `createdByUserId` — take it as a
    port, not by reaching into `nathejk.dk/internal/requestctx`.
  - No imports from `nathejk.dk/...` anywhere in the package. This is the single
    check that decides whether the lift is a file move or a rewrite, so it is
    worth enforcing in review from the first commit.
  - Depend only on shared-go types/messages (`shared-go/types`,
    `shared-go/messages`) for the domain vocabulary.
- **Handlers always stay local.** `go/cmd/api/sos.go` and the routes stay in hq
  permanently — they are not part of what gets lifted. Only the projection,
  queries, commands and schema move.
- **What must still go to shared-go up front:** the *types*, not the tables —
  the new `types.MemberStatus` constants, `SosCommentID`, the severity type and
  the SOS message structs. These are shared vocabulary that events are encoded
  with, so they cannot be prototyped locally without being redefined later
  (shared-go currently pinned at `v0.0.0-20260806204955-e7b46bb008f3`).

### BFF (Go)

- New resource handler file `go/cmd/api/sos.go` with one `<verb>SosHandler` per
  route, reading via `app.models.Sos` and writing via `app.commands.Sos`, using
  `app.ReadJSON`/`app.WriteJSON`/`app.ServerErrorResponse` etc. (Do **not**
  reuse the legacy `sos-routes.go` switch-on-`method+path` style or hand-rolled
  `json.NewEncoder`/`http.Error`.) The `PATCH` handler needs pointer or
  presence-tracked fields in its input struct so "field absent" is
  distinguishable from "field set to empty" — the same pattern as
  `updateYearHandler`/`patchKlanHandler`.
- New domain aggregate package `go/nathejk/table/sos/` written to shared-go's
  guidelines for a later lift (see placement section above), plus a `sos_team`
  association projection (legacy `sosassoc`). Port the two legacy read models:
  - `sos` table (id, year, headline, description, createdAt, createdBy, status,
    severity, assigneeSectionSlug) for the list/summary and by-team lookup.
    The assignee is stored as an organisation **section slug** (FK-style
    reference into the `section` table, year-scoped); list/detail queries join
    `section` to resolve the section label for display.
  - `sos_team` association table for team↔case links.
  - The rich **activity timeline** in legacy lived only in the in-memory
    `aggregate/sos` served over websocket — a symptom of the duplicate read model
    described in §8 "Lessons from the legacy `dims` channel", not a design. Since
    the current platform has one read model, the timeline must be persisted as a SQL
    projection too — add a
    `sos_activity` table (case id, seq/created-at, type, actor user id, value,
    status, comment text) so `GET /api/sos/:id` can return the full history.
- Assignee source: reuse the existing `section` projection
  (`github.com/nathejk/shared-go/tables/section`, already wired as
  `app.models.Section` / `app.commands.Section`) as the list of assignable
  sections. The frontend loads these via the existing `GET /api/organisation`
  endpoint, so no new endpoint is needed, and the `assigned` event/payload
  carries a section slug rather than the legacy free enum. How the assignable
  **subset** is determined is an Open Question.
- Write side: a `commands.Sos` command struct that publishes domain events.
  Port the legacy events but rename subjects to the current convention
  (`NATHEJK.{year}.sos.{sosId}.{event}` per `.rules`, built with
  `github.com/jrgensen/stream/subject`, rather than the legacy `nathejk:sos.*`
  channel strings):
  `created, headline.updated, description.updated, commented, comment.updated,
  severity.specified, assigned, deleted, closed, reopened, team.associated,
  team.disassociated`. Member events keep the **member** as their subject entity
  rather than the case. Two corrections to earlier assumptions here:
  - `messages.NathejkMemberStatusChanged` does **not** exist in shared-go.
    `go/nathejk/table/spejderstatus.go:49` references it, but only inside the
    commented-out block, and the type lives in the *legacy* local message
    packages (`_go`, and copies in `hjælper`/`scan-app`). It must be designed and
    added to `shared-go/messages/member.go`.
  - A single generic "status changed" event fits this model poorly. Each
    transition is a **distinct act by a distinct party** — a request to leave, a
    decision to carry on, an acceptance into a car, an acceptance at the shelter,
    a handover — so model them as separate events (e.g.
    `member.withdrawal.requested`, `member.withdrawal.cancelled`,
    `member.pickup.accepted`, `member.shelter.accepted`,
    `member.handover.completed`) that each carry the acting party and resolve to a
    `MemberStatus`. This makes the acceptor recordable, which a bare
    `{memberId, status}` payload cannot express — and it answers "who holds this
    member?" for free, because the car's acceptance event names the car.
  This interface publishes the withdrawal request and its cancellation, plus the
  override and `member.team.reassigned`. It **consumes** the rest.
  Populate `createdByUserId` from the authenticated user on every event.
- **The self-carrying boundary must be enforced on the write side, not just in the
  UI.** `member.withdrawal.cancelled` is valid only while the member is `waiting`;
  the command must dirty-check the current `spejderstatus` row and reject it
  otherwise. Hiding the button once the status advances is not sufficient — the
  operator's screen may be seconds stale (polling, §6), which is exactly when the
  car is accepting the member. If the acceptance and the cancellation race, the
  **acceptance wins**: it reflects a member physically sitting in a car, and the
  event log preserves both attempts in order.
- **Correlating externally-produced transitions onto a case timeline.** The car
  and shelter interfaces will not know a `sosId`, so the timeline cannot rely on
  one for the events it does not publish. Options: propagate the
  `correlationId` from the originating `waiting` event through the chain (clean,
  but requires every downstream producer to cooperate), or resolve at read time by
  matching the member to open cases associated with their team (no cross-repo
  coordination, but ambiguous when a member has two open cases). Decide before the
  timeline projection is written — see Open Questions.
- **Member status & team membership — revive `spejderstatus`.** The projection
  `go/nathejk/table/spejderstatus.go:13-18` already declares the right shape
  (`MemberID`, `InitialTeamID`, `CurrentTeamID`, `Status types.MemberStatus`) but
  is **inert**: `Consumes()` returns an empty subject list and the whole of
  `HandleMessage` is commented out. This feature revives it as the canonical
  member-status/membership projection — and `shared-go/types/member.go` now names
  it as such ("these strings live in the spejderstatus projection"), which
  settles where member status belongs: keep the projection, keep the name, do not
  fold it into `spejder`. Restructure it to shared-go's guidelines for a later
  lift.
  - Its schema needs extending: `spejderstatus.sql` is currently only
    `id, year, status, updatedAt` — the `initialTeamId` / `currentTeamId`
    columns the struct declares do not exist yet, and membership queries need
    an index on `(year, currentTeamId)`.
  - Reassignment updates `CurrentTeamID` only; `InitialTeamID` is immutable.
- **`racing` is derived from the team-started event, not set by this interface.**
  `messages.NathejkTeamStarted` on `NATHEJK.{year}.patrulje.{teamId}.started`
  (published by `commands.Team.StartPatrulje`, `go/nathejk/commands/team.go:125-148`,
  behind `PUT /api/patrulje/:id/start`) carries
  `Members []NathejkTeamStarted_Member` — precisely the members who actually
  started. So the revived `spejderstatus` consumer subscribes to
  `NATHEJK:*.patrulje.*.started` and, for each member in the body, writes
  `status = racing` with `initialTeamId = currentTeamId = teamId`. This closes the
  last gap in the member model: the withdrawal route now has a documented origin,
  and it needs no new event and no new producer.
  Details that matter for the consumer:
  - **Non-starters are deleted, not left behind.** `StartPatrulje` publishes
    `spejder.{memberId}.deleted` for every member who did *not* start, so the
    projection must handle that subject too or it will hold rows for no-shows and
    over-count team strength.
  - **Take the year from the subject**, not from `msg.Time()`. The body has no year
    field, the table is year-scoped, and the old commented-out code used
    `msg.Time().Year()` — which is wrong on replay across new year boundaries.
  - **Strength at start is `len(body.Members)`**, which the `patrulje` consumer
    already uses for `memberCount` (`table/patrulje/consumer.go:66`). Reusing the
    same source keeps the 3-member check consistent with the patrol's own member
    count rather than inventing a second definition.
  - This is the only status this feature consumes from an existing producer;
    `registered` and `seated` are irrelevant to the SOS panel, which only ever sees
    members who have started.
- **Legacy status values must be normalised on replay.** The projection is
  rebuilt from the full JetStream history, so it *will* encounter the superseded
  values that `shared-go/types/member.go` documents: `REGISTERED`/`STARTED` →
  `registered`/`racing`, `active` → `racing`, `emergency` → `waiting`,
  `hq` → `sheltered`, `out` → `released` (`waiting`/`transit` unchanged). The
  consumer must map these on the way in, otherwise the rebuilt table holds a
  mixture of old and new strings and every `InOurCare()`/`CanFinish()` check
  silently under-reports. This mapping belongs next to the constants in shared-go
  so all consumers share one implementation.
- **Reuse the lifecycle helpers rather than re-deriving them.**
  `MemberStatus.Valid()` gates what the API accepts, `CanFinish()` guards the
  finish-line flow, and `InOurCare()` is exactly the in-our-care count — no
  hand-rolled status lists in handlers, queries or the SPA.
- **Team strength & the 3-member requirement.** Strength is
  `COUNT(*) FROM spejderstatus WHERE year = ? AND currentTeamId = ? AND status = 'racing'`,
  which the revived projection can answer directly given the
  `(year, currentTeamId)` index. Two derived facts come from the same query and
  should share one implementation rather than being re-counted per caller: in
  breach (`< 3`) and discontinued (`= 0`). The threshold belongs in year
  configuration, not a constant — see Open Questions.
- **The requirement is real, but it is enforced by a person, not by a command.**
  No command may reject a withdrawal request because it would put a team below 3,
  and no consumer may auto-collect or auto-reassign in response. The member is
  leaving regardless — refusing to record that would only make the data wrong — and
  the remedy depends on things the projection cannot know: how far along the patrol
  is, how capable they are, whether a car is anywhere near. So the write side
  reports strength and records what the operator did; it never decides. The
  requirement is expressed as an unmissable breach plus a recorded handling, which
  is what makes it auditable after the event.
- **Model the handling explicitly.** A breach that has been dealt with must be
  distinguishable from one that has not, otherwise the UI cannot settle the warning
  and the post-event review cannot tell the two apart. Whether that is a
  first-class event or a case-scoped flag is an Open Question, but it needs to exist
  — "we notice breaches but do not record their resolution" is the failure mode to
  avoid.
- **Whole-team collection is one command, not N.** A single
  `sos.team.collected`-style command should publish a withdrawal request per
  remaining racing member atomically from the operator's point of view, sharing one
  `correlationId` so the timeline can render it as one entry ("hele patruljen hentes")
  while `spejderstatus` still sees per-member events. Publishing three independent
  requests from the frontend would risk a partial collection if one call fails —
  the worst possible outcome, since the team would then be split across states with
  nobody noticing.
- **Reassignment candidates need a query.** "Another team if such is available"
  requires the backend to offer targets; the rules are an Open Question, but note
  the shape: candidates are patrols in the same year, still racing, and
  — presumably — near enough to physically join. This is the query the legacy merge
  never had, because an operator picked the parent team by hand.
- **Discontinued teams replace `patruljemerged`.** Legacy encoded "no members
  left" as a `patruljemerged` row (`teamId → parentTeamId`), inserted on
  `.merged` and deleted on `.splited`, and derived discontinued teams from it in
  `go/internal/data/team.go:60`:
  `SELECT DISTINCT m.teamId FROM patruljemerged m JOIN patruljestatus s ...`.
  Under the new model, a team is discontinued when no member with
  `currentTeamId = team` is `racing` — `racing` being the only status that counts
  towards a team's strength on the route. **Careful:** a team that reached the
  finish also has nobody `racing` (its members are `finished`), so the naive
  predicate would report every finishing team as discontinued. The predicate must
  distinguish "left the route" from "completed it" — see Open Questions. Two
  viable shapes:
  1. **Derive on read** — query `spejderstatus` directly in
     `GetDiscontinuedTeamIDs`. Simplest, inherently reversible, no new event.
  2. **Explicit event** — a consumer watches membership and publishes
     `patrulje.discontinued` / a matching un-discontinue event, projected into a
     column on `patruljestatus`. Puts the fact on the log but needs both
     directions to stay reversible.
  Option 1 is the proposal; the decision is an Open Question. Either way, the
  discontinued query must actually be implemented —
  `go/nathejk/table/patrulje/query.go:121` `GetDiscontinuedTeamIDs` currently
  **returns an empty slice**, so discontinued teams are silently broken in the
  current platform. Note `shared-go/tables/patrulje` has no discontinued query at
  all yet, and is where it should end up. Once membership is the source of truth,
  the local `patruljemerged` projection and its consumer can be deleted.
- Wire the new projections into the `xstream.Mux` and `data.NewModels(...)` /
  `commands.New(...)` in `go/cmd/api/main.go`.

### API endpoints

New endpoints, following the resource conventions already in
`go/cmd/api/routes.go` — id in the path, `POST` to create, `PATCH` for
single-field updates, `PUT`/`DELETE` for sub-resources:

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/stream` | Platform SSE stream of `entity.changed` signals — **specified in PRD 004**, listed here only because this feature depends on it |
| GET | `/api/sos` | List cases (open/closed) for current year |
| GET | `/api/sos/:id` | Get one case with timeline + associated teams |
| POST | `/api/sos` | Create case (headline + description) |
| PATCH | `/api/sos/:id` | Update single fields: `headline`, `description`, `severity`, `assigneeSectionSlug`, `status` (open/closed) |
| DELETE | `/api/sos/:id` | Delete case (legacy `sos.deleted`) |
| POST | `/api/sos/:id/comment` | Add a plain-text comment |
| PATCH | `/api/sos/:id/comment/:commentId` | Edit a comment (legacy `comment.updated`) |
| PUT | `/api/sos/:id/team/:teamId` | Associate a patrol with the case |
| DELETE | `/api/sos/:id/team/:teamId` | Disassociate a patrol |
| POST | `/api/sos/:id/team/:teamId/collect` | Collect the whole team: every remaining `racing` member → `waiting`, one action |
| POST | `/api/sos/:id/team/:teamId/exception` | Grant an exception to the 3-member requirement (reason required) |
| GET | `/api/sos/:id/team/:teamId/reassign-candidates` | Patrols available to receive the remaining members |
| PUT | `/api/member/:memberId/waiting` | Record that the member wants to leave the race (→ `waiting`), `sosId` required |
| PUT | `/api/member/:memberId/racing` | Member carries on under their own steam (→ `racing`); rejected unless currently `waiting` |
| PUT | `/api/member/:memberId/status` | Override a member's status (correction path), optional `sosId` |
| PUT | `/api/member/:memberId/team` | Reassign member to another team (`currentTeamId`), optional `sosId` |

Notes on the shape:

- **`PATCH /api/sos/:id` carries all single-field case updates.** The legacy API
  had a separate verb-in-path endpoint per field (`/api/sos/headline`,
  `/severity`, `/assign`, `/close`, `/reopen`, each with the id in the body); those
  collapse into one partial update, matching `PATCH /api/year/:slug` and
  `PATCH /api/klan/:id`. Close/reopen are a `status` field like any other — §5
  already requires them to be idempotent, which a field assignment gives for free.
- **Each field still emits its own domain event.** The handler diffs the patch
  against current state and publishes only what changed (`headline.updated`,
  `severity.specified`, `assigned`, `closed`, `reopened`), so the write model and
  the timeline keep their granularity even though the transport is one endpoint.
  A patch that changes nothing publishes nothing.
- **Team actions are sub-resources of the case**, because associating a patrol,
  collecting it and granting it an exception are all facts *about this case's
  handling of that team* — unlike member status, which lives on the member.
- The withdrawal request and the override are **separate endpoints** rather than
  one parameterised status setter, because they are different acts: one is the
  normal workflow this interface owns, the other is an admission that another
  interface's handover went unrecorded. Splitting them keeps the override auditable
  and makes "how often are we correcting by hand?" answerable. `sosId` is required
  on the withdrawal request — a member should not leave the race without a case
  explaining why — and optional on the other two.

The `transit`, `sheltered`, `released` and `reunited` transitions have **no
endpoint here**: they belong to the car and shelter interfaces (§4 Non-Goals).
The boundary is *self-carrying* — while the member is on their own feet the
transitions are ours, and from the car door onwards they are the receiver's.

(Legacy had `/api/sos/merge` and `/api/sos/split`; these are **replaced** by
member reassignment plus derived discontinuation and are not ported. The legacy
`/api/sos/sms`
position-SMS endpoint is dropped entirely.) Note the legacy read model was delivered over a
websocket, so there were **no** legacy `GET /api/sos` endpoints — the two GETs
above are new and required by the polling approach.

Whether these need OpenAPI annotations is an Open Question — hq has no OpenAPI
tooling today.

### Data / storage

- New tables: `sos`, `sos_team`, `sos_activity` (all `CREATE TABLE IF NOT
  EXISTS`, embedded via `//go:embed`, MariaDB, year-scoped). Rebuilt from
  JetStream on startup like every other projection.

### Dependencies & risks

- **Shared types:** legacy used a local `nathejk/types` with `types.SosID`,
  `types.SosCommentID`, `types.Enum` and member statuses. In shared-go,
  `types.SosID` **already exists** (`types/types.go:51`) and
  `messages.NathejkTeamMerged` / `NathejkTeamSplited` already carry an optional
  `sosId` — so the precedent for tagging domain events with a case id is
  established. Still missing and needing a shared-go addition: `SosCommentID`,
  the severity type, the SOS message structs, and the new `MemberStatus`
  constants. That is a cross-repo change with its own workflow (currently pinned
  at `shared-go v0.0.0-20260806204955-e7b46bb008f3`), or defined locally if that
  is acceptable.
- **Member status model:** member status is the shared
  `github.com/nathejk/shared-go/types.MemberStatus` enum, which is now **fully
  defined and documented** in `shared-go/types/member.go` (commit "name and
  document the member lifecycle") — states, lifecycle diagram, terminal states,
  the `Valid()`/`CanFinish()`/`InOurCare()` helpers, and the mapping from the
  superseded values. This feature therefore **defines no member statuses of its
  own**; it is the first consumer of that vocabulary (nothing else in shared-go
  references the new constants yet), which makes it the feature that proves the
  model. Anything that does not fit should be raised as a change to
  `types/member.go`, not worked around locally.
  The SOS interface owns the transitions where the member is still
  **self-carrying**: `racing` → `waiting` and `waiting` → `racing`. The lifecycle
  documentation describes the route as starting when the member "or their team
  called trailside assistance", and trailside assistance *is* this interface. The
  subsequent transitions are acceptances by the receiving party in their own
  interface (car → `transit`, shelter → `sheltered`, handover →
  `released`/`reunited`), and the statuses before it belong to other flows —
  `racing` is derived from the existing team-started event (see below),
  `registered`/`seated` from signup and orders, `finished` at the finish. All of
  them write the same constants into the same projection.
- **The resume transition is now documented in shared-go** (commit `5ac2603`,
  "document that waiting is reversible, not a withdrawal"), which resolves the
  contradiction this PRD previously had to flag. The model is explicit: `racing`
  and `waiting` are reversible in both directions, `waiting` "is not yet a
  withdrawal" because the member is still self-propelled, and the irreversible
  step is **outside help** — getting into a car. `CanFinish()` stays
  `racing`-only, which is exactly right: a `waiting` member may still finish, but
  not from `waiting` — they have to actually carry on, which puts them back to
  `racing` first. The dependency is already bumped: `go/go.mod` now pins
  `v0.0.0-20260807180020-5ac2603c60ba`, so the lifecycle model and its
  documentation are available to build against today.
- **Team strength counts `racing` members only**, per
  `MemberStatusRacing`'s documentation. A `waiting` member does **not** count —
  deliberately, because the same documentation notes the team "may not continue
  until the member is either collected or back on their feet", so a patrol with a
  waiting member is halted regardless. Note two distinct thresholds follow from
  this and must not be conflated:
  - **In breach of the 3-member requirement:** fewer than 3 `racing` members —
    surfaced to the operator, who handles it (§6). Not enforced in code.
  - **Discontinued:** *no active members* — zero `racing` — the team is out,
    derived and reversible as described below.
- **The withdrawal route completes only once the car and shelter interfaces
  exist — they are separate PRDs, and this is a sequencing dependency rather than
  an open question here.** Nothing in the platform produces member status changes
  today: the `spejderstatus` consumer is commented out in every repo that has it
  (`hq`, `hjælper`, `skan`), and `NathejkMemberStatusChanged` exists only in legacy
  local message packages, not in shared-go. Until the downstream interfaces ship, a
  member put into `waiting` has no automatic way out, so `InOurCare()` will not
  drain on its own and the `waiting` alarm will fire for everybody.
  What follows for *this* PRD:
  - The **override** is the interim path. It is specified anyway (§6) as the
    correction route, and until the car and shelter tools exist it is how an
    operator records a pickup or an arrival by hand. Its cost is honest and
    already measured — §9 tracks override frequency precisely because a high count
    means handovers are not being recorded where they happen.
  - The **in-our-care counter and the `waiting` alarm should be read as provisional
    until then.** They are correct as specified; they will simply be dominated by
    manual bookkeeping in the interim. Do not tune the alarm threshold against that
    interim behaviour.
  - This PRD must therefore leave a **clean seam** for the later PRDs to build
    against, rather than assuming anything about how they work. See below.
- **The seam this PRD owns.** So the car and shelter PRDs can be written
  independently, this feature fixes the following and nothing more:
  - `spejderstatus` is the shared member status/membership projection. Downstream
    interfaces write the same table via their own events; they do not get their own
    copy.
  - Status values are `shared-go/types.MemberStatus` and the transition each
    interface is responsible for is fixed by the *self-carrying* boundary: ours up
    to and including `waiting`, theirs from the car door on.
  - Each downstream transition is an **acceptance by the receiver**, so its event
    carries the accepting party. This PRD does not define those events' payloads,
    only that they exist, are per-member, and resolve to a `MemberStatus`.
  - This interface **consumes** them for the timeline and the counter, so it must
    tolerate transitions it did not cause, arriving in any order, including for
    members on teams it never associated with a case.
  - How a downstream event is correlated to a case is still open (§11) and is the
    one part of the seam the later PRDs may need to influence.
- **Discontinuation is a derived fact, not a status:** "discontinued" describes a
  *team*, member statuses describe *members*, and the team fact follows from the
  members. Getting this wrong is the main modelling risk — do not add a
  `discontinued` flag that operators can set independently of membership, or it
  will drift out of sync the way the legacy `patruljemerged` rows could.
- **Blast radius of reviving `discontinued`:** `discontinuedTeamIds` feeds the
  checkgroup/checkpoint status view (`go/internal/data/models.go:239-240`
  computes `cg.Discontinued` and `cg.NotArrived` from it). Since
  `GetDiscontinuedTeamIDs` currently returns nothing, implementing it will change
  what that view shows — previously-"not arrived" teams will start appearing as
  discontinued. That is the correct behaviour but it is a visible change to an
  existing screen and should be verified with organizers.
- **Three overlapping query paths:** `go/internal/data/team.go:60` (legacy
  `patruljemerged` SQL), `go/nathejk/table/patrulje/query.go:121` (stub) and
  `shared-go/tables/patrulje` (no such query yet) all could host
  `GetDiscontinuedTeamIDs`, and `internal/data/models.go` still imports
  `patrulje` from the **local** package while taking neighbouring entities from
  shared-go. Per the placement rule the implementation goes in the **local**
  `table/patrulje` and travels with that entity when it is lifted; the legacy
  `internal/data` path should be retired rather than kept in parallel. Confirm
  which path is live at runtime before implementing, otherwise the fix may land
  in the unused one.
- **Cross-repo delivery on the critical path is limited to shared vocabulary.**
  Because the projections are built locally and lifted later, only the *types*
  block progress — and `MemberStatus` is already done **and already pinned**
  (`go/go.mod` at `v0.0.0-20260807180020-5ac2603c60ba`), so what remains is
  `SosCommentID`, the severity type, the SOS message structs, and the legacy
  status-value mapping. Each needs a shared-go commit, release and `go.mod` bump
  here. Risk if skipped: events get encoded with locally-defined types and every
  consumer has to be revisited when they move.
- **Lift-readiness can rot silently.** "Written to shared-go's guidelines" is
  only true if it stays true; a single convenience import of `nathejk.dk/...`
  turns the future lift from a file move into a rewrite, and nothing in the build
  will complain. Worth an explicit review check (see task list).
- **No websocket:** the entire legacy live-update UX depended on the `dims`
  websocket (`_go/cmd/api/dims.go`, `_go/pkg/sockethub`). The replacement is SSE
  rather than a websocket, which is a simplification rather than a regression:
  push is one-way here because writes go over REST. Read §8 "Lessons from the
  legacy `dims` channel" before reviving anything from it — the difficulty there
  came from a duplicate in-memory read model and a hand-rolled client cache, not
  from the transport, so porting its *shape* would reintroduce the problem under a
  new protocol.
- **Backwards compatibility:** none required — this is net-new in the current
  platform; legacy code is reference only.

## 9. Success Metrics

- Operators log ≥ 90% of emergency-phone calls as SOS cases during the event
  (qualitative: paper backup essentially unused).
- **The in-our-care count reaches zero before the organisers go home, and it did
  so because every member was genuinely accounted for — not because somebody
  cleared stale rows.** This is the metric the lifecycle documentation is written
  around, and it is the one that matters: a stale status here is a member nobody
  is looking for.
- No member ends the event `waiting` or `transit` — either would mean we lost
  track of somebody in our care.
- Median time from `waiting` to `transit` (how long a blocked patrol waits for a
  car), tracked from the timeline. No target for the first event; establish a
  baseline. Note this measures the **car fleet**, not this interface — it is here
  because this is where the data lands, and it is the number that tells organisers
  whether they have enough cars.
- **Every breach of the 3-member requirement has a recorded handling** — collected,
  reassigned, or an exception granted with a reason. Zero unhandled breaches at the
  end of the event is the target; *which* handling was chosen is not the measure,
  and a patrol that finished on an exception is a success.
- Exceptions are visible and reviewable afterwards. If most breaches were resolved
  by exception, either the requirement or the car capacity needs a conversation —
  which is only possible because the exception is recorded rather than implied.
- Overrides stay rare. A high override count means handovers are not being
  recorded where they happen, and the chain of custody is fiction.
- Resumes (`waiting` → `racing`) are tracked, not minimised. Each one is a car not
  sent and a patrol that kept going, so the count is a benefit of the feature
  rather than a defect — but a high *rate* may mean operators are recording
  `waiting` too eagerly during the call.
- Every case has a complete timeline (create → resolution) with no gaps;
  spot-check after the event shows shift handovers were possible from the tool
  alone.
- No incidents caused by the tool during the event (it must be dependable under
  stress).

## 10. Rollout / Task Breakdown

Phased so value lands early and the risky shared-type work is isolated. Only the
shared *types* need a shared-go release up front; the projections are built
locally to shared-go guidelines and lifted later, and handlers stay local
permanently.

- **Phase 1 — Core case management:** create/list/view, headline, close/reopen,
  comments, severity, assignee, timeline persistence. Delivers a usable
  dispatch log.
- **Phase 2 — Teams & members:** associate/disassociate patrols, the `waiting`
  transition on the revived `spejderstatus` projection, the read-only chain
  display, the in-our-care counter, member reassignment (`currentTeamId`) with
  derived team discontinuation, the 3-member breach handling, and the "Kontakt med
  nødtelefon" card on the patrol page. Shippable independently of the car and
  shelter PRDs — the withdrawal route simply relies on the override for its later
  steps until those land.
- **Phase 3 — Live updates:** adopt PRD 004's SSE transport (wiring the SOS and
  `spejderstatus` consumers through its notifier) plus the connection-state
  indicator. Separable, since PRD 004's cache layer delivers
  most of the perceived speed on its own.

Proposed tasks to create in `roadmap/tasks/open/` (not created yet):

- [x] Task: hq — bump `shared-go` to `5ac2603` or later (reversible-waiting docs + lifecycle) — **done**, `go.mod` pins `v0.0.0-20260807180020-5ac2603c60ba`
- [ ] Task: Local — team-strength query on `spejderstatus` (racing count) with shared in-breach / discontinued derivation
- [ ] Task: Local — record breach handling (collected / reassigned / exception granted, with reason & operator)
- [ ] Task: Local — whole-team collection as a single command (one `correlationId`, per-member events)
- [ ] Task: Local — reassignment-candidate query (available target patrols)
- [ ] Task: Frontend — breach warning + pre-commit warning on the `waiting` action, with collect-all / reassign / grant-exception actions
- [ ] Task: shared-go — add `SosCommentID`, severity type and SOS message structs; release + bump `go.mod` here
- [ ] Task: shared-go — add the legacy → current `MemberStatus` value mapping as shared code (documented in `types/member.go`, not yet implemented)
- [ ] Task: Local — `go/nathejk/table/sos/`: `sos` + `sos_team` + `sos_activity` projections & schemas, to shared-go guidelines
- [ ] Task: Local — `sos` write side (commands) + JetStream subjects (year-scoped)
- [ ] Task: shared-go — design & add the withdrawal-chain member events (request / cancel / pickup-accepted / shelter-accepted / handover) to `messages/member.go`, including the acting party
- [ ] Task: Follow-up PRDs — car-acceptance interface and shelter-acceptance interface (out of scope here; this PRD fixes the seam they build against)
- [ ] Task: Local — revive the `spejderstatus` projection: consume `patrulje.*.started` (→ `racing`) and `spejder.*.deleted`, legacy-value normalisation, `initialTeamId`/`currentTeamId` columns + index, year from subject
- [ ] Task: Local — in-our-care query (`InOurCare()` statuses) for the dashboard counter
- [ ] Task: Local — implement the discontinued-teams query from membership (excluding finished teams)
- [ ] Task: Local — wire SOS projections/commands into `cmd/api/main.go`
- [ ] Task: Local — SOS REST handlers (`go/cmd/api/sos.go`), stays local permanently
- [ ] Task: Local — member endpoints: `waiting` request, resume to `racing` (with self-carrying dirty-check), status override, reassign team
- [ ] Task: Local — consolidate the discontinued query paths; delete the `patruljemerged` projection
- [ ] Task: Review check — assert the `sos` and `spejderstatus` packages import nothing from `nathejk.dk/...` (lift-readiness)
- [ ] Task: Verify the checkgroup status view with discontinued teams re-enabled
- [ ] Task: PRD 004 — live-update capability (transport, cache primitive, `/api/stream`, notifier). Prerequisite for the live/instant behaviour below; tracked in its own PRD
- [ ] Task: Local — wire the `sos`, `sos_team`, `sos_activity` and `spejderstatus` consumers through PRD 004's `notify` decorator
- [ ] Task: Frontend — SOS adoption of `useLiveResource` (case list, open case, in-our-care counter)
- [ ] Task: Frontend — SOS-specific bits: KeepAlive on the list, route-chunk preload, detail seeded from list row, optimistic writes
- [ ] Task: Local — bootstrap endpoint (list + in-our-care counter + assignable sections) to avoid a cold-load waterfall
- [ ] Task: Frontend — `SosListView` + in-our-care counter + `/sos` route + nav item
- [ ] Task: Frontend — `SosView` detail with timeline + `SosActivityLine`
- [ ] Task: Frontend — team association, `waiting` / `Fortsætter selv` actions, read-only chain display & reassignment UI
- [ ] Task: Frontend — "Kontakt med nødtelefon" card on patrol detail
- [ ] Task: Confirm severity list and the `waiting` alarm threshold with organizers
- [ ] Task: Decide & implement which organisation sections are assignable to cases
- [ ] Task: Follow-up (post-stabilisation) — lift `sos` and `spejderstatus` into `shared-go/tables/`

## 11. Open Questions

- **Discontinued predicate vs finished teams:** a team that finished also has
  nobody `racing`, so "nobody racing" alone would mark every finishing team
  discontinued. Proposal: a team is discontinued when no member with
  `currentTeamId = team` is `racing` **and** none is `finished` — i.e. everybody
  left the route. Confirm, including the in-between case: a patrol where two
  members finished and one was `released` mid-route is finished, not
  discontinued — is that right?
- **`waiting` alarm threshold:** how long may a member be `waiting` before the
  dashboard warns? Their patrol is blocked for the whole duration, so this is the
  one number operators will feel. Fixed config value or per-severity?
- **Where the in-our-care count lives:** the nødtelefon list view (proposal), the
  HQ dashboard (`/api/home`), or both? It is an event-wide number rather than a
  SOS one.
- **Sequencing the withdrawal route:** should the API enforce the documented
  order (reject `racing` → `sheltered`, say) or accept any `Valid()` status and
  let the timeline show what happened? Strictness protects the data; leniency
  matters at 3am when the real world skipped a step.
- **Derived vs evented discontinuation:** derive discontinued on read from
  `spejderstatus` (proposal), or publish an explicit
  `patrulje.discontinued` event and project it? The evented version puts the fact
  on the log but needs a matching un-discontinue path to keep the legacy
  `.splited` reversibility. Related: should an operator be able to discontinue a
  team **directly**, independent of its membership?
- **Producer ownership of `registered` / `seated`:** which flows set them (signup
  + orders?) — not needed by this feature, since the SOS panel only ever sees
  members who have started, but worth knowing before `spejderstatus` is treated as
  complete. `racing` is settled: derived from `NathejkTeamStarted` (§8).
- **Reassignment target rules:** can a member be reassigned to any patrol, only
  to patrols in the same city/liga, or only to already-started patrols? And can a
  member be reassigned to a *klan*, or patrols only?
- **Existing `patruljemerged` data:** any historical `patruljemerged` rows encode
  past discontinuations that the new model cannot reconstruct (there are no
  per-member reassignment events behind them). Since §4 already rules out
  migrating legacy data, confirm we simply drop them and start fresh for the
  current year.
- **OpenAPI:** the `prd` skill mandates OpenAPI annotations on every endpoint,
  but hq has no OpenAPI tooling, spec or annotations anywhere in `go/`, and
  `.rules` does not require it. Do we introduce the tooling as part of this
  feature, or drop the requirement for this repo?
- **Frontend shared state:** a Pinia store (`vue/src/stores/sos.ts`) or a
  composable with module-level `ref()` (`vue/src/composables/sos.ts`)? The SPA
  has Pinia installed but keeps shared state in composables in practice, so the
  composable is the more idiomatic fit — confirm before the view work starts.
- **Live updates (PRD 004):** the transport, cache primitive and notifier are
  specified there, including the one unknown that could change the approach —
  whether SSE survives Traefik without buffering. Nothing
  in this PRD needs to answer it; Phase 3 here depends on that spike passing.
- **Seniors as well as spejdere:** `spejderstatus` is named for spejdere, but
  `MemberStatus` is documented as a *member* lifecycle and klan members are
  members too. Does the revived projection cover seniors, and if so does the name
  stay as-is (shared-go's documentation refers to "the spejderstatus
  projection")? §4 scopes SOS to patrols only, so this is not blocking, but it
  decides the shape of the eventual lift.
- **Shared types:** should `SosCommentID`/severity and the SOS message structs be
  added to `github.com/nathejk/shared-go`, or defined locally in this repo? Who
  owns the shared-go change and its release?
- **Assignable sections:** the assignee is an organisation section, but "maybe
  not all sections" — how do we determine the assignable subset? The `section`
  table (shared-go `tables/section`) has `slug, year, parentSlug, label,
  sortOrder` and **no** `assignable` flag, so a flag means a shared-go schema
  change, whereas "all descendants of a designated parent section" works today
  via `parentSlug`. Also: what happens to a case's assignee if that section is
  later renamed or deleted on the Organisation page?
- **Deferred to the car and shelter PRDs** (recorded here only so they are not
  lost): which product becomes the car interface and which the shelter one; who
  performs the final handover (`released` / `reunited`) — `reunited` happens at the
  finish line rather than the shelter, so it may not belong to either; and car
  dispatch, including whether the `waiting`-too-long alarm should *request* a
  pickup. The only piece that may need to sit in this interface is that request, so
  it is worth a decision when the car PRD is written rather than now.
- **Correlating downstream events to a case:** propagate the `correlationId` from
  the `waiting` event through the chain, or resolve at read time from the member's
  team and open cases? The first needs every downstream producer to cooperate — so
  it is the one seam decision the car and shelter PRDs may need to influence, and
  worth settling with whoever writes them; the second is self-contained here but
  ambiguous when a member has two open cases.
- **Where does the required minimum live?** 3 is the requirement for patrols, but
  it should be configurable rather than compiled in — year configuration (`year`
  entity) seems the natural home since it is a rule of the event. Confirm, and
  confirm whether it has ever differed between years or ligas.
- **What makes a patrol an available reassignment target?** Same year and still
  racing are obvious; beyond that, does it need to be near the stranded members
  (same checkpoint, same city, same liga), under some maximum size, or is it purely
  the operator's judgement from a list? And may survivors be split across two
  different target patrols, or must they stay together?
- **Does the requirement apply to klaner?** §4 scopes case association to patrols,
  so this may be moot for now, but the 3-member rule as stated is patrol-specific
  and klaner presumably have their own (or none).
- **What happens to a reassigned member's finish?** A survivor moved into another
  patrol is still `racing` and self-carrying, so they can still finish — but they
  finish with a team that is not the one they started with (`initialTeamId` ≠
  `currentTeamId`). Does that affect diplomas, results, or how the finish is
  recorded?
- **How is a granted exception modelled and scoped?** §6 requires it to be
  recorded with the acting operator. Is it a first-class event
  (`sos.understrength.exception.granted`) or a case-scoped record? And is it scoped
  to the breach (a later member leaving is a new breach needing new handling — the
  proposal) or to the team for the rest of the event?
