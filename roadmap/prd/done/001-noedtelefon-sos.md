# PRD 001 — Nødtelefon / SOS case management

**Status:** done
**Author:** agent session (recreating legacy feature)
**Created:** 2026-07-29
**Last updated:** 2026-08-11
**Approved:** 2026-08-11
**Shipped:** 2026-08-11
**Status note:** live updates confirmed in two browser tabs by the product owner
2026-08-11. Split on 2026-08-10 — the member half (withdrawal chain, team strength,
discontinuation) moved to **PRD 006**, which is sequenced after this one
**Target users:** organizer (HQ emergency-phone operators / nødtelefonvagter)

---

## 1. Summary

Recreate the legacy **Nødtelefon / SOS** feature in the current platform: a
lightweight "dispatch desk" that lets HQ operators log and manage emergency calls
that come in on the event's emergency phone (nødtelefon). Each call becomes an SOS
*case* (sag) with a headline, description, priority, an assigned organisation
section, an append-only activity timeline, and one or more associated patrols. The
patrol's detail page shows the cases it is involved in.

**What this PRD does not cover:** anything that changes a *member's* state. Marking
a member as wanting to leave the race, team strength, the 3-member requirement,
reassignment and team discontinuation are **PRD 006 — Member lifecycle, team
strength & discontinuation**. This document delivers the dispatch log those actions
will later hang off; it is deliberately shippable on its own, because a written,
timestamped, handover-able record of every call is most of the value and needs none
of PRD 006's open decisions.

### Why this is two PRDs

The original PRD 001 covered both halves and grew to ~1,200 lines and 18 open
questions, most of which belonged to the member half: what the discontinued
predicate is, what makes a patrol a valid reassignment target, how an exception is
modelled, what happens to a reassigned member's diploma. None of those block the
dispatch log. Worse, the member half is not really an SOS feature at all — it is a
platform-wide member lifecycle that the future car and shelter interfaces depend on
just as much as this screen does, and that touches the checkgroup status view and
retires the `patruljemerged` projection.

So: this PRD is the case management. PRD 006 is the member lifecycle. They share a
seam (PRD 006 renders member rows inside this PRD's case detail view, and writes
member transitions onto this PRD's timeline), documented in §8.

## 2. Problem & Motivation

- **What problem does this solve?** During the event, participants in the field
  call the emergency phone for anything from a twisted ankle to a lost patrol.
  Operators need a single place to record each incident, track its priority and who
  is handling it, tie it to the affected patrol(s), and keep a timestamped log of
  everything that happened — so a shift handover or a later review is possible and
  nothing is lost on paper notes.
- **Why now?** The feature existed and was used in the legacy platform (found in
  `_go/` and `_vue/`) but was not carried over when the platform was rebuilt. The
  current platform (`go/` + `vue/`) has no incident-handling capability at all, so
  operators currently have no digital tool for the emergency phone.
- **Evidence.** The legacy implementation is fully built out:
  - Frontend: `_vue/src/views/Sos/List.vue`, `_vue/src/views/Sos/View.vue`, the
    `dims` Vuex store (`_vue/src/store/dims.js`), the "Kontakt med nødtelefon" card
    on `_vue/src/views/Patrulje.vue`, and the `/sos` nav entry in
    `_vue/src/components/Navigation.vue`.
  - Backend: `_go/cmd/api/sos-routes.go`, `_go/cmd/api/soscmd.go`,
    `_go/nathejk/messages/sos.go`, `_go/nathejk/aggregate/sos/sos.go`,
    `_go/nathejk/table/sos.go`, `_go/nathejk/table/sosassoc.go`,
    `_go/nathejk/table/sos.sql`, `_go/internal/data/sos.go`.

## 3. Goals

- Operators can create an SOS case from an incoming call with a headline and a
  free-text description in a few seconds.
- Each case has a mutable **status** (open/closed), **priority/severity**
  (green/yellow/red) and an **assignee** — an organisation **section** (from the
  Organisation page) responsible for handling it.
- Every change to a case is captured on an append-only, **persisted** activity
  timeline (comments, status changes, priority/assignee changes, team
  association/disassociation), so a shift handover is possible from the tool alone.
- Operators can **associate one or more patrols** with a case, and see the patrol's
  members listed for context.
- The patrol detail page surfaces the SOS cases the patrol is involved in.
- The screen is **live and instant**: it reflects other operators' changes without
  a refresh, and returning to it never costs a spinner.
- Cases are scoped to the current event year, matching the rest of the platform.

## 4. Non-Goals

- **Everything that changes a member's state → PRD 006.** The `waiting`
  transition, the resume to `racing`, the status override, member reassignment,
  team strength, the 3-member requirement and its handling, team discontinuation,
  the "in our care" counter and the `waiting` alarm are specified there. This PRD
  lists a patrol's members for context only, and offers no actions on them.
- The car/driver interface and the shelter reception interface. They will be
  specified in their own PRDs, **later and separately**; nothing here anticipates
  them, and PRD 006 owns the seam they will build against.
- **Notifying the assigned section.** Assignment is a label on the case, not a
  dispatch mechanism: no SMS, no mail, no push. Operators tell the section by phone or
  radio, as they do today. If that ever changes it is its own PRD.
- Public/participant-facing UI. This is an internal HQ tool only.
- Automated telephony/IVR integration — calls are still answered on a physical
  phone; this only records them.
- Position-request SMS / GPS location of members. The legacy feature could text a
  member a link to share their position (`member.positionsms.*`, `/api/sos/sms`);
  this is **not** ported.
- Rich-text or attachment support in comments (legacy was plain text; keep it).
- Migrating historical legacy SOS data. Cases start fresh for the current year.
- Merge/split of patrols. The terms *merged* and *splitted* are retired from the
  domain vocabulary and the legacy `patrulje.merged` / `patrulje.splited` events
  are not ported; what replaces them is specified in PRD 006. Legacy
  `/api/sos/merge` and `/api/sos/split` therefore have no counterpart here.

## 5. User Stories & Scenarios

- As an **operator**, I want to log an incoming emergency call as a new case so
  that it is tracked and not forgotten.
- As an **operator**, I want to set a case's priority and assign it to the
  responsible organisation section so the right team picks it up and urgent cases
  stand out.
- As an **operator**, I want to add comments over time so the case history is a
  complete, timestamped record for shift handover.
- As an **operator**, I want to associate the caller's patrol with the case so the
  next person reading it knows who it is about.
- As an **operator**, I want the case I have open to show what my colleague just
  added, without me reloading.
- As an **operator viewing a patrol**, I want to see any SOS cases the patrol is
  part of so I have context.

### Primary happy path

1. Emergency phone rings. Operator opens **Nødtelefon** in the nav and clicks **Ny
   sag**.
2. They type a headline ("Forstuvet ankel ved post 4") and a short description, and
   save. The case appears under **Åbne sager**.
3. They set priority to **yellow** and assign it to the **Samarit** section.
4. They search and associate the caller's patrol; the patrol's members are listed
   for context.
5. As things develop they add comments. A colleague on another machine adds one
   too, and it appears on the timeline without either of them reloading.
6. When resolved, they **Luk sag**; it moves to **Lukkede sager**. If it flares up
   again they **Genåbn sag**.

### Edge cases & error scenarios

- Creating a case with an empty headline or description is rejected.
- Closing an already-closed case (or reopening an open one) is a no-op, not an
  error.
- Associating a team already associated is idempotent (`INSERT IGNORE` in legacy
  `sosassoc`).
- Two operators editing the same case concurrently — last write wins per field; the
  timeline still records both actions in order.
- An operator with the headline editor open, or half a comment typed, must **not**
  have it clobbered by an incoming live update. See §8.
- A case is deleted while another operator has it open: they get a clear "sagen er
  slettet" state, not a stuck screen.
- Only cases for the current event year are shown (legacy filtered messages by
  `msg.Time().Year()`; the current platform scopes by year slug).

## 6. Requirements

### Functional

- [x] Create an SOS case with headline + description (`POST /api/sos`). **Both are
      required.** The server mints the `SosID` and returns the created case; the SPA
      then replaces `/sos/new` with `/sos/:id`.
- [x] Edit a case headline and description (`PATCH /api/sos/:id`).
- [x] Close and reopen a case (`PATCH /api/sos/:id`, `status` field).
- [x] **Soft-delete** a case (`DELETE /api/sos/:id`, legacy `sos.deleted`), for one
      created in error. The projection sets `deletedAt` and keeps the row and its
      timeline; the case disappears from both lists and `GET /api/sos/:id` answers
      404, so an operator holding it open gets the "sagen er slettet" state from §5.
      Recovery is possible because nothing is destroyed — the row is still there and
      the event log is authoritative — but there is **no restore endpoint or UI in the
      first slice**; an accidental deletion is undone by an operator with database
      access. **Any operator may delete**, as with every other write here.
- [x] Add a plain-text comment to a case (`POST /api/sos/:id/comment`). The server
      mints the `SosCommentID` and returns it, so the comment has a stable target
      for a later edit.
- [x] Edit an existing comment (`PATCH /api/sos/:id/comment/:commentId`, legacy
      `comment.updated`). **The timeline stays append-only:** the edit writes a new
      `sos_activity` row referencing the original comment id, and the original row is
      left untouched. The detail view renders the current text with an "redigeret"
      marker rather than hiding that it changed. **Any operator may edit any
      comment** — there is no per-user identity to restrict it to the author, and the
      append-only trail is what keeps that safe. Revisit when the auth service lands.
- [x] Set severity to **`green` | `yellow` | `red`** (`PATCH /api/sos/:id`), Rendered as a coloured badge labelled **Grøn / Gul /
      Rød**; it does not filter or sort the list in the first slice, which is ordered
      by last activity.
- [x] Assign a case to an organisation **section** (`PATCH /api/sos/:id`). The
      selectable sections are those flagged **assignable** for the year — a flag owned
      by this feature (`sos_assignable_section`), toggled per section on the
      Organisation page and exposed through the existing `GET /api/organisation`. A
      case keeps the slug it was assigned; if that section is later renamed the new
      label simply shows, and if it is deleted the case displays the raw slug marked
      "(slettet sektion)" rather than dropping the assignment.
- [x] Associate / disassociate a patrol with a case (`PUT` /
      `DELETE /api/sos/:id/team/:teamId`). Only patrols can be associated with a
      case — clans (klaner) cannot.
- [x] The team picker is **searchable by team number, name and group**, since a
      caller reads out their number. It filters the year's patrol list already held
      in the SPA's live cache (`GET /api/patrulje`, as used by `PatruljeListView`) —
      no new search endpoint.
- [x] Show each associated patrol's **identity and contact** — team number, name,
      group, korps and the contact person's phone — which is what an operator needs
      mid-call. **No member list, no member status, no member actions.** The member
      rows are introduced by PRD 006 together with the status and actions that make
      them useful; a list of names with nothing next to them would read as a broken
      feature.
- [x] List cases grouped into open / closed with columns: headline, created, last
      activity, priority, assignee (`GET /api/sos`), sorted by **last activity
      descending** within each group so the case that just moved is at the top.
- [x] View a single case with its full activity timeline and associated teams
      (`GET /api/sos/:id`).
- [x] Show SOS cases associated with a patrol on that patrol's detail page. Delivered
      by **extending `GET /api/patrulje/:id`**, which already assembles members,
      payments and orders (`go/cmd/api/patrulje.go:85-96`), rather than by a second
      request — port legacy `data.SosModel.GetByTeam` (`_go/internal/data/sos.go:33`)
      as the query.
- [x] The timeline is **persisted as a SQL projection**, not held in memory, and
      renders every activity type with an icon and a Danish label.
- [x] Capture the acting user on every event as `createdByUserId`, resolved from the
      request context (`requestctx.UserFrom`) and passed to the command by the
      handler. Note this is **empty in practice** until the planned auth service
      lands (see Non-Functional → Auth); the field and the plumbing exist so that
      nothing has to change when it does.
- [x] All cases and events are scoped to the current event year.
- [x] The case list, the open case and its timeline are **live**, and the case
      timeline tolerates entries produced by events this interface did not publish
      (which is what PRD 006 will add).

### Non-Functional

- **Consistency with platform:** REST + JSON via `app.*` helpers; MySQL
  projections rebuilt from JetStream on startup; frontend via the `http` module
  (`@/plugins/axios`) and PrimeVue Aura. No Bootstrap, no `vue-good-table`, no
  `b-popover` (all legacy-only).
- **Auth — perimeter today, per-user identity later.** Access is protected at the
  **perimeter**: HTTP basic auth in front of the service on stage and production
  (deployment configuration, outside this repo), and **no auth in dev**. The API
  itself does not authenticate: `app.authenticate`
  (`go/cmd/api/routes.go:127-193`) has its body commented out and injects
  `requestctx.User{ID: "", Name: "anonymous"}` on every request, and nothing reads
  `AUTH_BASEURL` (set at `docker-compose.yml:70`, unused). A proper auth service
  issuing JWTs for signed-in users is planned but not scheduled.
  **The consequence this PRD accepts deliberately:** the feature is written *as if*
  an authenticated user is present — commands take an actor and events carry
  `createdByUserId` — so nothing needs restructuring when the auth service lands.
  Until then that id is empty in practice and the timeline is attributable by
  **time, not by person**. That is a known, recorded limitation rather than a
  surprise, and it must not be presented in the UI as though it were an audit trail:
  do not render an "af <bruger>" byline that would always be blank.
- **Authorisation:** none beyond the perimeter. Every operator who can reach the
  service can do everything here. Acceptable for an internal HQ tool behind basic
  auth; revisit when the auth service exists (`internal/data/permissions.go` already
  models permissions and is unused by routes).
- **Timeliness / freshness:** an operator's view reflects other operators' changes
  within ~1 second, without refreshing. **The capability already exists** — PRD 004
  shipped 2026-08-09 (SSE via `GET /api/stream`, `go/internal/live/`,
  `vue/src/composables/useLiveResource.ts`), so this is adoption, not new
  infrastructure, and it is part of the first slice rather than a later phase.
- **Perceived speed — a first-class requirement, not polish.** This screen is used
  continuously through the event, often while talking on the phone, so:
  - Navigating away and back shows the current state **instantly** — the
    module-level cache renders immediately and revalidates behind it. No spinner on
    a revisit.
  - Operator actions (comment, patch) feel immediate: `optimisticWrite` applies the
    change and reconciles when the signal arrives.
  - A cold load paints usable content in well under a second, with no request
    waterfall.
- **Localization:** Danish UI text and `da-DK` date formatting, matching the rest
  of the SPA.
- **Auditability:** the activity timeline is append-only, and every entry carries a
  timestamp and the acting user as resolved from the request context. Legacy carried
  a `UserID` in message bodies but the API did not always populate it; here it is
  always populated — with the caveat above that its value is empty until the auth
  service exists, which is why the timeline's *timestamps* are what a handover
  currently relies on.

## 7. UX / UI Notes

New frontend surface (all inside the `ui` SPA, `vue/src`):

- **Nav entry:** add "Nødtelefon" (icon `fa-headset`) to `items` in
  `vue/src/components/Navigation.vue`, routing to a new `sos` route.
- **List view — `vue/src/views/SosListView.vue`** (`/sos`): a PrimeVue `DataTable`
  (Aura preset, as configured in `vue/src/main.ts`) with two groups, **Åbne sager**
  and **Lukkede sager**; columns Overskrift, Oprettet, Sidst opdateret, Prioritet,
  Tildelt; a **Ny sag** button; row click opens the detail view. Empty state: "Ingen
  nødråb fundet". `:loading` bound to the resource's `pending`, not a bespoke
  spinner.
- **Detail view — `vue/src/views/SosView.vue`** (`/sos/:id`, `props: true`, plus a
  `/sos/new` route for creation):
  - Editable headline with pencil affordance.
  - Summary card: status badge (Åben/Afsluttet), created timestamp, priority,
    assignee.
  - **Activity timeline** rendering each activity type (comment, comment edited,
    close, reopen, severity, assign, associate, disassociate) with an icon and a
    Danish label — port the legacy `ActivityLine` component to
    `vue/src/components/SosActivityLine.vue`. The component must render **unknown
    activity types gracefully**, because PRD 006 adds more.
  - Comment composer. (Both headline and description are required when *creating* a
    case; a comment needs only its text.)
  - Actions: Luk sag / Genåbn sag, Tilføj kommentar.
  - Right column cards:
    - **Tilknyttede patruljer:** a team picker searchable by number, name and group
      over the SPA's cached patrol list, then one row per associated patrol showing
      its number, name, group, korps and contact phone, with a link to the patrol's
      own page and a remove action. **No member rows** — PRD 006 adds them, along with
      each member's status and actions and the team's strength/breach warnings. Leave
      vertical room for that rather than designing a layout it has to fight.
    - **Prioritet** select (Grøn/Gul/Rød, coloured badge) and **Tildelt** select. The
      **Tildelt** options are the **assignable** organisation sections loaded from the
      backend via the existing `GET /api/organisation` — shown by section label,
      stored by section slug. Do **not** hardcode the legacy assignee list
      (guide/samarit/rover/…).
  - **Dirty-guard.** While the headline editor is open or the comment composer has
    text, incoming payloads are deferred and applied when the edit ends, and the UI
    says updates are paused — as `KlanListView.vue` and `KortView.vue` do. This is
    required, not optional: it is a page holding unsaved state, and the operator is
    typing while on the phone.
- **Organisation page:** the section rows gain an **assignable** toggle ("kan tildeles
nødråb"), which is the only change this feature makes to an existing screen besides
the patrol card. Off by default, so the assignee list starts empty and is opted into
deliberately.
- **Terminology:** the field and its events are `severity`; the UI label is
  **Prioritet**. Do not let the two drift into a third name.
- **Patrol detail:** add a "Kontakt med nødtelefon" card to
  `vue/src/views/PatruljeView.vue` listing the patrol's SOS cases with created date,
  headline and open/closed badge; clicking navigates to the case. The data arrives in
  the existing `GET /api/patrulje/:id` payload, so the card is a render change plus
  one token added to that view's `dependsOn`.
- **State:** no store. The views compose `useLiveResource` from
  `vue/src/composables/`, with shared bits (if any) in a composable with
  module-level `ref()` like the rest of the SPA. The Pinia-vs-composable question
  the earlier draft carried is **closed** by PRD 004: there is one cache primitive
  and pages compose it.

## 8. Technical Considerations

### Frontend (Vue 3 / TS)

- New views `SosListView.vue`, `SosView.vue`; new component
  `SosActivityLine.vue`; routes in `vue/src/router/index.ts`; nav item in
  `Navigation.vue`.
- Use PrimeVue `DataTable`/`Select`/`Textarea`/`Badge` (auto-imported via
  `unplugin-vue-components`) on the existing **Aura** preset (`vue/src/main.ts`)
  plus Tailwind. Do not introduce a second theme preset.
- All requests go through the `http` module (`import { http } from
  '@/plugins/axios'`) to relative `/api/...` paths. It sets `baseURL: '/api/'` and
  attaches `X-YearSlug` via a request interceptor, so year scoping comes for free —
  SOS endpoints must **not** take the year as a path or query parameter.

### Live updates (adoption, not construction)

PRD 004 shipped on 2026-08-09; all ~20 projections are already wrapped by
`live.NotifyAll` in `cmd/api/main.go`, so a new projection becomes live by being
added to that slice. PRD 004 §12 explicitly asks this feature to compose
`useLiveResource` **from the start** rather than treating live updates as a later
phase, and the repo `.rules` now say the same. What that means concretely:

- **Signals are derived from the event subject, not the projection name.** All
  three new projections (`sos`, `sos_team`, `sos_activity`) consume
  `NATHEJK.{year}.sos.{sosId}.*`, so they all emit the **one** token `sos`. Do not
  expect a token per projection or per table.
- **`dependsOn` sets, stated explicitly** — PRD 004 §12 records that bare-string
  `dependsOn` tokens were the one recurring defect (two of six wrong, failing
  silently), so they belong in the spec rather than being guessed per view:
  - case list → `['sos']` (type, so newly created cases appear)
  - case detail + timeline → `['sos:{id}', 'sos']`
  - patrol page's "Kontakt med nødtelefon" card → the card has **no resource of its
    own**: cases arrive in `GET /api/patrulje/:id`, so add `'sos'` to
    `PatruljeView.vue`'s existing `dependsOn`
    (`['patrulje:{id}', 'spejder', 'order', 'payment']`)
  - PRD 006 will add a member token to the detail view; that is its change to
    make, not a placeholder to add now.
- The SPA's dev-only dependency validation warns about tokens nothing can emit —
  use it while building rather than reasoning about it.
- **Optimistic writes** via `vue/src/composables/optimisticWrite.ts` for comments
  and patches: an operator on the phone must never wait for a round trip.
- **Detail view seeded from the list row** (headline, status, severity, assignee
  are already known), with the timeline arriving after.
- `KeepAlive` on the list and route-chunk preload are **not** inherited from PRD
  004 — its §12 records them as never started and probably unnecessary, since the
  module-level cache delivers most of the perceived speed. If the case list still
  feels slow, that is a follow-up task here, not an assumed dependency.
- **A caveat this PRD must not overstate.** PRD 004 §8 argues no staleness UI is
  needed because the API does not serve until its projections are caught up. That
  boot gate is **PRD 005, still a draft** — there is no readiness gate in
  `go/cmd/api/main.go` today. So during an API restart this screen can briefly
  serve a partially-rebuilt read model. For a dispatch log of textual cases that is
  tolerable; it is much less tolerable for PRD 006's counters, which is where the
  dependency should be recorded.

Legacy note: the `dims` websocket that served this screen before is dissected in
PRD 004 §2.1. Read it before reviving anything from `_go/cmd/api/dims.go` or
`_vue/src/store/dims.js` — the difficulty there was a duplicate in-memory read
model and a hand-rolled client cache, not the transport, so porting its shape would
reintroduce the problem under a new protocol. Note the legacy SOS **activity
timeline existed only in that in-memory model**, which is precisely why it must be
a SQL projection here.

### BFF (Go) — where the code lives

Entities are being lifted one by one out of `go/nathejk/table/` into
`github.com/nathejk/shared-go/tables/`, and shared-go is where they will all end up
eventually. Current state of that migration:

- **Already in shared-go/tables:** `crewmember`, `klan`, `order`, `patrulje`,
  `payment`, `product`, `section`, `senior`, `signup`, `spejder`.
- **Still local in `go/nathejk/table/`:** `checkgroup`, `checkpersonnel`,
  `checkpoint`, `lok`, `patruljemerged`, `patruljestatus`, `personnel`, `pincode`,
  `registrant`, `scan`, `spejderstatus`, `year` — plus still-live local duplicates
  of `order`, `patrulje`, `payment`, `senior` and `spejder` that the migration has
  not finished retiring.

Decisions for this feature:

- **Build the SOS package locally, in `go/nathejk/table/sos/`, but written to
  shared-go's guidelines so it can be lifted unchanged.** New entities are
  developed locally where the dev loop is fast (`docker compose up` +
  `inotifywait` hot-reload, no release-and-bump cycle), and lifted to
  `shared-go/tables/sos/` once the schema and events are stable.
- **"Written to shared-go's guidelines" means, concretely:**
  - Follow the shared-go layout, not the varying local one (`lok` has no
    `commands.go`; `order` uses `commander.go`/`querier.go`/`saga.go`). The newest
    migration, `tables/signup`, is the reference: `table.go`, `consumer.go`,
    `querier.go`, `commands.go`, `repository.go`, `interfaces.go`, `table.sql`.
  - Respect the dependency-inversion rule documented in
    `shared-go/tables/interfaces.go`: the package declares what it needs from the
    application in its own `interfaces.go`, satisfied structurally by the consuming
    service, and never imports application code.
  - **The acting user is passed in, not fetched.** Every other local command reaches
    for `requestctx.UserFrom(ctx)` directly (`table/year/commands.go:28`,
    `table/checkgroup/commands.go:54`, `table/checkpoint/command.go:31`), which is an
    import of `nathejk.dk/...` and therefore off-limits here. The simplest resolution
    needs no port at all: the **handler** resolves the actor and passes it to the
    command as an explicit argument. Keep it that way when the auth service lands —
    the actor becomes non-empty, and no package boundary moves.
  - **No imports from `nathejk.dk/...` anywhere in the package.** This is the single
    check that decides whether the lift is a file move or a rewrite, and nothing in
    the build will complain if it rots, so it is worth enforcing in review from the
    first commit.
  - Depend only on shared-go types/messages for the domain vocabulary.
- **Handlers always stay local.** `go/cmd/api/sos.go` and the routes stay in hq
  permanently — they are not part of what gets lifted. Only the projection,
  queries, commands and schema move.
- **What must go to shared-go up front:** the *types*, not the tables —
  `SosCommentID`, the severity type and the SOS message structs. These are shared
  vocabulary that events are encoded with, so they cannot be prototyped locally
  without being redefined later. `types.SosID` **already exists**
  (`shared-go/types/types.go:51`), and `messages.NathejkTeamMerged` /
  `NathejkTeamSplited` already carry an optional `sosId`, so the precedent for
  tagging domain events with a case id is established. `go/go.mod` currently pins
  `github.com/nathejk/shared-go v0.0.0-20260807180020-5ac2603c60ba`.

  **Amended 2026-08-11, at the start of implementation.** The types are defined
  **inside `go/nathejk/table/sos/`** instead, and lifted to shared-go with the package.
  Reason: a shared-go type change is only consumable here after commit, push, release
  and a `go.mod` bump — and the dev container mounts only `./go`, so a `replace`
  directive does not help either. Putting the vocabulary in shared-go first would have
  put a cross-repo release cycle in front of every other task, which is exactly what
  §8's "build locally, lift later" rule exists to avoid. The package still imports
  nothing from `nathejk.dk/...`, so the lift stays a file move, and the SOS events have
  no consumer outside hq today — nothing else can be broken by the vocabulary living
  here for now.

### BFF (Go)

- New resource handler file `go/cmd/api/sos.go` with one `<verb>SosHandler` per
  route, reading via `app.models.Sos` and writing via `app.commands.Sos`, using
  `app.ReadJSON`/`app.WriteJSON`/`app.ServerErrorResponse` etc. Do **not** reuse
  the legacy `sos-routes.go` switch-on-`method+path` style or hand-rolled
  `json.NewEncoder`/`http.Error`. The `PATCH` handler needs pointer or
  presence-tracked fields in its input struct so "field absent" is distinguishable
  from "field set to empty" — the same pattern as
  `updateYearHandler`/`patchKlanHandler`.
- New aggregate package `go/nathejk/table/sos/` with three projections:
  - **`sos`** (id, year, headline, description, createdAt, createdBy, status,
    severity, assigneeSectionSlug, **lastActivityAt**) for the list/summary and
    by-team lookup. `lastActivityAt` is maintained by the projection on every event
    for that case, so the list's "Sidst opdateret" column and its default sort are a
    single-row read rather than an aggregate over `sos_activity`. The assignee is
    stored as an organisation **section slug** (FK-style reference into the
    year-scoped `section` table); list/detail queries join `section` to resolve the
    label for display.
  - **`sos_team`** association table for team↔case links (legacy `sosassoc`).
  - **`sos_activity`** (case id, activity id, seq/created-at, type, actor user id,
    value, status, comment text, **refActivityId**) so `GET /api/sos/:id` can return
    the full history. New relative to legacy, where the timeline lived only in the
    in-memory aggregate. `refActivityId` is what makes a comment edit append rather
    than overwrite — it points at the comment being amended. Design the type column
    and payload so PRD 006 can append member-related entry types without a schema
    change.
- Assignee source: reuse the existing `section` projection
  (`github.com/nathejk/shared-go/tables/section`, already wired as
  `app.models.Section` / `app.commands.Section`), filtered to sections flagged
  **assignable**.

  **Amended 2026-08-11:** the flag lives in **hq, owned by the SOS domain**
  (`sos_assignable_section`, keyed by year + section slug, written by
  `NATHEJK.{year}.sos.section.{slug}.assignable.set|unset`), not as a column on
  shared-go's `section` table. Two reasons, one practical and one about modelling:
  shared-go cannot be released from here mid-implementation (see above); and "can be
  assigned nødråb" is a fact about **this feature**, not a general property of an
  organisation section — a section does not become a different thing because the
  nødtelefon can route to it. Keeping it here means the Organisation page renders the
  toggle but does not own the concept, and the earlier objection to deriving the list
  from `parentSlug` still stands: reorganising the tree must not change who can be
  assigned. `GET /api/organisation` exposes the flag per section by joining this
  table.

  Rejected alternative: "all descendants of a designated parent section", which works
  today via `parentSlug` but couples the assignee list to the shape of the
  organisation tree. The `assigned` event carries the section slug rather than the
  legacy free enum.
- Write side: a `commands.Sos` command struct that publishes domain events with
  subjects on the current convention — `NATHEJK.{year}.sos.{sosId}.{event}`, built
  with `github.com/jrgensen/stream/subject`, rather than the legacy `nathejk:sos.*`
  channel strings:
  `created, headline.updated, description.updated, commented, comment.updated,
  severity.specified, assigned, deleted, closed, reopened, team.associated,
  team.disassociated`. `createdByUserId` comes from the actor the handler passes in
  (empty until the auth service lands), and commands dirty-check so a patch that
  changes nothing publishes nothing (and therefore emits no live signal).
- **Extend the patrulje read path** rather than adding an endpoint: `GET
  /api/patrulje/:id` gains the team's cases, alongside the members, payments and
  orders it already assembles (`go/cmd/api/patrulje.go:85-96`).
- Wire the new projections into the `projections` slice and `data.NewModels(...)` /
  `commands.New(...)` in `go/cmd/api/main.go`. The slice matters: a consumer added
  to the mux outside it is silently not live.

### API endpoints

New endpoints, following the resource conventions already in
`go/cmd/api/routes.go` — id in the path, `POST` to create, `PATCH` for partial
updates, `PUT`/`DELETE` for sub-resources:

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/sos` | List cases (open/closed) for current year, ordered by last activity |
| GET | `/api/sos/:id` | Get one case with timeline + associated teams |
| POST | `/api/sos` | Create case (headline + description, both required); returns the created case including its id |
| PATCH | `/api/sos/:id` | Update single fields: `headline`, `description`, `severity`, `assigneeSectionSlug`, `status` (open/closed) |
| DELETE | `/api/sos/:id` | Soft-delete a case (legacy `sos.deleted`); no restore endpoint in the first slice |
| POST | `/api/sos/:id/comment` | Add a plain-text comment |
| PATCH | `/api/sos/:id/comment/:commentId` | Edit a comment (legacy `comment.updated`) |
| PUT | `/api/sos/:id/team/:teamId` | Associate a patrol with the case |
| DELETE | `/api/sos/:id/team/:teamId` | Disassociate a patrol |

One **existing** endpoint changes: `GET /api/patrulje/:id` gains the patrol's cases,
which is how the "Kontakt med nødtelefon" card is fed. No new by-team endpoint.

`GET /api/stream` is **not** listed: it exists already (PRD 004) and needs no
change for this feature.

Notes on the shape:

- **`PATCH /api/sos/:id` carries all single-field case updates.** The legacy API had
  a separate verb-in-path endpoint per field (`/api/sos/headline`, `/severity`,
  `/assign`, `/close`, `/reopen`, each with the id in the body); those collapse into
  one partial update, matching `PATCH /api/year/:slug` and `PATCH /api/klan/:id`.
  Close/reopen are a `status` field like any other — §5 already requires them to be
  idempotent, which a field assignment gives for free.
- **Each field still emits its own domain event.** The handler diffs the patch
  against current state and publishes only what changed (`headline.updated`,
  `severity.specified`, `assigned`, `closed`, `reopened`), so the write model and
  the timeline keep their granularity even though the transport is one endpoint.
- **Team association is a sub-resource of the case**, because associating a patrol
  is a fact about *this case*. PRD 006's member actions will live on the member (or
  on `/api/sos/:id/team/:teamId/...` where the action is about the case's handling
  of that team) — that is its decision to make.
- **No search endpoint for the team picker.** The SPA already holds the year's patrol
  list live (`GET /api/patrulje`, cached by `PatruljeListView`), so association
  filters that cache client-side on number, name and group.
- The legacy `/api/sos/merge`, `/api/sos/split` and `/api/sos/sms` endpoints have
  no counterpart. The first two are replaced by PRD 006; the third is dropped.
- The legacy read model was delivered over a websocket, so there were **no** legacy
  `GET /api/sos` endpoints — the two GETs above are new.

The endpoints above are specified here rather than in an OpenAPI document: hq has no
OpenAPI tooling, and the mandate to annotate was dropped for this repo (see
Decisions).

### Data / storage

- New tables: `sos`, `sos_team`, `sos_activity` (all `CREATE TABLE IF NOT EXISTS`,
  embedded via `//go:embed`, MariaDB, year-scoped). Rebuilt from JetStream on
  startup like every other projection.
- `sos` carries `deletedAt` for the soft delete; every read path filters it out, and
  the row and its timeline are retained.
- `section` (shared-go) gains an `assignable` boolean.
- Note `CREATE TABLE IF NOT EXISTS` never alters an existing table, so getting the
  `sos_activity` shape roughly right up front matters more than usual in dev: a
  column added later silently will not appear where the table already exists. The
  same trap applies to `section.assignable`, which lands in a table that already
  exists everywhere.

### Dependencies & risks

- **Shared types are no longer on the critical path** (§8 amendment, 2026-08-11): the
  SOS vocabulary lives in the `sos` package and is lifted with it, so no shared-go
  release blocks this work. The risk moves to lift discipline instead — if the package
  acquires a `nathejk.dk/...` import, the lift becomes a rewrite.
- **PRD 006 sequencing.** This PRD is deliberately independent: it ships and is
  useful with no member functionality at all. The reverse is not true — PRD 006
  needs this PRD's case, timeline and association surfaces to hang its actions off,
  so it is sequenced after. The interfaces PRD 006 relies on are listed in §8
  above; the risk to manage is designing `sos_activity` and the team card so
  extension is additive.
- **Boot gate (PRD 005) is not shipped**, so a brief post-restart window can serve
  a partially rebuilt read model. Tolerable for textual cases (see §8); record it
  as a real dependency for PRD 006's counters instead of assuming it here.
- **Live-update tokens fail silently when wrong** (PRD 004 §12). Mitigated by
  stating them in §8 and by the dev-only validation.
- **No per-user identity yet** (see §6 Auth): the timeline is attributable by time
  only until the planned auth service lands, and the perimeter is basic auth on
  stage/production with nothing in dev. The risk is presentational as much as
  technical — if the UI implies attribution it does not have, operators will trust
  the log further than they should.
- **Backwards compatibility:** none required — this is net-new in the current
  platform; legacy code is reference only.

## 9. Success Metrics

- Operators log ≥ 90% of emergency-phone calls as SOS cases during the event
  (qualitative: paper backup essentially unused).
- Every case has a complete timeline (create → resolution) with no gaps; a
  post-event spot-check shows shift handovers were possible from the tool alone.
  Note this is a claim about **times and content**, not about who did what — there is
  no per-user identity yet.
- No operator reports having to reload the screen to see a colleague's change, and
  none reports losing typed text to an incoming update.
- No incidents caused by the tool during the event (it must be dependable under
  stress).
- Case deletions are rare and explicable — a high count means cases are being used
  as scratch notes, or that creation is too easy to do by accident.

## 10. Rollout / Task Breakdown

One slice, in roughly this order. Live-update adoption is **not** a separate phase:
per PRD 004 §12 the views compose `useLiveResource` from their first commit.

Proposed tasks to create in `roadmap/tasks/open/` (not created yet):

- [x] Task: Local — SOS domain vocabulary (`SosCommentID`, severity, message structs)
      inside `go/nathejk/table/sos/`, to be lifted with the package
- [x] Task: Local — `sos_assignable_section` projection + toggle command, exposed in
      `GET /api/organisation`
- [x] Task: Local — `go/nathejk/table/sos/`: `sos` + `sos_team` + `sos_activity`
      projections & schemas, to shared-go guidelines
- [x] Task: Local — `sos` write side (commands) + year-scoped JetStream subjects,
      with per-field dirty-checking
- [x] Task: Local — wire SOS projections/commands into `cmd/api/main.go`, including
      the `projections` slice so they are live
- [x] Task: Local — SOS REST handlers (`go/cmd/api/sos.go`), stays local permanently
- [x] Task: Local — extend `GET /api/patrulje/:id` with the patrol's cases (port
      legacy `data.SosModel.GetByTeam`)
- [x] Task: Frontend — `SosListView` + `/sos` route + nav item, on
      `useLiveResource(['sos'])`
- [x] Task: Frontend — `SosView` detail with timeline + `SosActivityLine`
      (tolerant of unknown activity types), seeded from the list row, optimistic
      comment/patch writes, dirty-guard on the headline editor and composer
- [x] Task: Frontend — team association card: searchable picker over the cached
      patrol list plus per-team identity and contact (no member rows — PRD 006)
- [x] Task: Frontend — "Kontakt med nødtelefon" card on patrol detail (render only —
      data comes from the extended patrulje payload; add `'sos'` to that view's
      `dependsOn`)
- [x] Task: Frontend — `assignable` toggle on the Organisation page section rows
- [x] Task: Review check — assert the `sos` package imports nothing from
      `nathejk.dk/...` (lift-readiness)
- [ ] Task: Follow-up (post-stabilisation) — lift `sos` into `shared-go/tables/`
      — **not done, deliberately**; carried as task 055

## 11. Open Questions

None outstanding. Everything that was open has been decided — see Decisions below.
The questions that were holding the original document up moved to PRD 006 with the
work they belong to.

### Decisions

Recorded so they are not reopened.

- **Authentication (2026-08-10):** perimeter-only — basic auth on stage/production,
  none in dev — with a JWT-issuing auth service planned but unscheduled. The feature
  is written as though an authenticated user is present, so the actor plumbing and
  `createdByUserId` are built now and carry an empty value until that service exists
  (§6 Auth).
- **Severity (2026-08-10):** `green` | `yellow` | `red`, labelled Grøn / Gul / Rød.
  Display only in the first slice — no filtering or sorting by it.
- **Assignable sections (2026-08-10):** an `assignable` flag on the section, toggled
  on the Organisation page. Not derived from the organisation tree's shape, so
  reorganising the tree cannot silently change who can be assigned. **Amended
  2026-08-11:** the flag is owned by this feature in hq
  (`sos_assignable_section`) rather than added as a column to shared-go's `section`
  — see §8 for why.
- **Domain vocabulary (2026-08-11):** `SosCommentID`, the severity type and the SOS
  message structs are defined in `go/nathejk/table/sos/` and lifted with the package,
  rather than going to shared-go first — see §8.
- **Case deletion (2026-08-10):** soft. `deletedAt` on the row, hidden from the
  lists, 404 on the detail, nothing destroyed, no restore UI in the first slice.
- **Who may write (2026-08-10):** every operator may create, edit, comment, edit
  anybody's comment, and delete. There is no identity to scope permissions to, and
  the append-only timeline is what makes that acceptable. Revisit with the auth
  service.
- **Members on the case card (2026-08-10):** not shown before PRD 006. The card lists
  each associated patrol's identity and contact only; member rows arrive with the
  status and actions that give them a purpose.
- **Assignee notification (2026-08-10):** none. Assigning a case to a section is a
  **label** that operators see on screen; the tool does not SMS or email the section,
  and the operator tells them by phone or radio as they do today. Same as legacy.
  Notification would need its own decisions (SMS or mail, whole section or a lead,
  what happens on reassignment, what happens at 3am) and belongs in its own PRD if it
  is ever wanted.
- **OpenAPI (2026-08-10):** dropped for this repo. hq has no OpenAPI tooling, spec or
  annotations, and `.rules` does not ask for any; the endpoint table in §8 is the API
  documentation. Note the `prd` skill still states the mandate
  (`.agents/skills/prd/SKILL.md`), so it will keep surfacing on future PRDs until
  somebody relaxes it there — this decision governs hq regardless.

## 12. Closure

**Closed as completed 2026-08-11**, after tasks 042–054.

### What shipped

| Half | Delivered |
|---|---|
| BFF (`go/nathejk/table/sos/`) | domain vocabulary (ids, severity, status, twelve event bodies); three projections — `sos`, `sos_team`, `sos_activity` — plus `sos_assignable_section`; commands with per-field dirty-checking on year-scoped subjects; queries for list, single-case-with-timeline, by-team and assignable sections; a lift-readiness test |
| BFF (`go/cmd/api/`) | nine SOS routes, the section-assignable toggle, `GET /api/patrulje/:id` extended with the patrol's cases, `actor.go` resolving the acting user for the domain, wiring into the `projections` slice so every SOS change is live |
| Frontend (`vue/src/`) | `SosListView`, `SosView`, `SosActivityLine`, `SosTeamCard`, `composables/sos.ts`, routes and the nødtelefon nav entry; the "Kontakt med nødtelefon" card on the patrol page; the assignable toggle on the Organisation page; `seedLiveResource` added to PRD 004's cache primitive |

Verification: 22 Go tests in the sos package plus the existing suites, `go vet` and
`gofmt` clean, 60 frontend tests, no new TypeScript errors, and **every endpoint
exercised against the running dev stack** — create, validation failures, list, timeline
ordering, patch, assignee rejection, comment, comment edit, association, klan rejection,
close, close-again idempotency, reopen, delete, 404-after-delete.

And — the part PRD 004 §12 called the one that mattered — **live updates confirmed in two
browser tabs by the product owner, 2026-08-11**. That closes the loop the agent session
could not: the backend halves were verifiable (the `sos` token advertised at
`/api/stream`, the consumers inside the `projections` slice, the views declaring
`dependsOn`), but whether an operator actually sees a colleague's change appear was not,
and a wrong `dependsOn` token fails silently rather than loudly.

### Deliberately not done

Recorded rather than ticked, because none of it is finished:

1. **Lifting the package to shared-go** — task 055, explicitly post-stabilisation. Its
   whole purpose is that the schema has stopped changing.
2. **`OrganisationView` is still not live.** It loads by hand and needs a dirty guard
   first (PRD 004 §12). This feature added a toggle to it without changing that, so the
   flag needs a reload to reflect another operator's change. Its own task, not smuggled
   into a PRD 001 one.
3. **`KeepAlive` and route-chunk preload** on the case list: not inherited from PRD 004,
   not needed so far. Revisit only if the list feels slow with a real event's worth of
   cases.
4. **SSE on the production path** remains unverified — stage and production have basic
   auth and one proxy hop fewer than dev, and that is task 041, not this PRD's to close.
   The two-tab confirmation above was on the dev path.

### After closure

**Task 056** refined the detail view on the product owner's feedback, over five rounds on
the same day: the case moved into a tinted note card sharing one style definition with the
comment cards, both sitting on a single panel with the timeline and the composer; the
activity log became a PrimeVue `Timeline` with comments on the same rail as state changes;
creation and headline/description edits stopped producing timeline entries (a display
filter — the events and rows are untouched, so the audit trail is intact); the handling
controls were dimmed and narrowed; and the live-update indicator moved onto the navigation
bar's moon — yellow when healthy, red while reconnecting, dark grey when offline.

Recorded here rather than folded into the sections above, because the PRD is the record of
what was agreed and shipped, and this was agreed after it shipped. Nothing in the
requirements changed.

### What this PRD got wrong, for the next one

- **The task order had a dependency backwards.** 045 (assignable sections) needed 046
  (wiring) to exist first. Cheap to fix mid-flight, but the lesson is that "wire it up"
  belongs early, not after the things that call into it.
- **Asynchronous projections change what a write endpoint may answer**, and the PRD did
  not say so anywhere. Both `POST` and `PATCH` were written to read the row back, and both
  were wrong: the read races the projection and loses. `POST` now synthesises the case it
  just published and `PATCH` answers 202 with an echo, because returning pre-patch values
  to an optimistic UI makes it flicker backwards — the same "stale value that looks live"
  trap PRD 004 spent its whole implementation on. Worth stating up front in any PRD that
  adds write endpoints to this platform.
- **Verifying against the running stack found things tests did not.** Both of the above,
  plus a response leaking Go field names. The dev loop is cheap here; use it per endpoint
  rather than at the end.
- **The auth claim in an early draft was simply false** ("behind the existing JWT cookie
  auth"), and it survived several revisions because it was plausible. Reviewing a PRD's
  factual claims against the code is worth as much as reviewing its design.
- **The UX section was too thin to build from.** §7 described the surfaces but not their
  hierarchy, so the first implementation gave the case, its history and the handling
  controls roughly equal weight — and every one of task 056's five rounds was about that
  hierarchy rather than about behaviour. A sentence naming what an operator reads *most*
  on each screen would have saved all of it.
