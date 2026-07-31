# PRD 001 — Nødtelefon / SOS case management

**Status:** draft
**Author:** agent session (recreating legacy feature)
**Created:** 2026-07-29
**Last updated:** 2026-07-29
**Target users:** organizer (HQ emergency-phone operators / nødtelefonvagter)

---

## 1. Summary

Recreate the legacy **Nødtelefon / SOS** feature in the current platform: a
lightweight "dispatch desk" that lets HQ operators log and manage emergency
calls that come in on the event's emergency phone (nødtelefon). Each call
becomes an SOS *case* (sag) with a headline, description, priority, assignee,
an activity timeline, and one or more associated patrols whose members can be
tracked. The interface's primary member action is marking a member as
**resigned** (udgået); the remaining statuses are normally set automatically by
other parts of the system as real events happen, with a manual override here to
correct drift.

## 2. Problem & Motivation

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
  patrol's members. The interface's primary member action is marking a member
  as **resigned**; operators can also **override** a member's status to any
  valid value to correct out-of-sync data. Member statuses are: **Active,
  Resigned, Transit, Stationed, Departed**.
- The patrol detail page surfaces the SOS cases the patrol is involved in.
- Cases are scoped to the current event year, matching the rest of the platform.

## 4. Non-Goals

- Public/participant-facing UI. This is an internal HQ tool only.
- Automated telephony/IVR integration — calls are still answered on a physical
  phone; this only records them.
- Position-request SMS / GPS location of members. The legacy feature could text
  a member a link to share their position (`member.positionsms.*`,
  `/api/sos/sms`); this is **not** ported.
- Real-time GPS map tracking of members.
- Rich-text or attachment support in comments (legacy was plain text; keep it).
- Migrating historical legacy SOS data. Cases start fresh for the current year.
- Merge/split of patrols. This never actually existed as a real operation — the
  legacy `team.merged` / `team.splited` events are **not** ported. The real
  need is **moving a member between teams** (see Requirements), which supersedes
  it.

## 5. User Stories & Scenarios

- As an **operator**, I want to log an incoming emergency call as a new case so
  that it is tracked and not forgotten.
- As an **operator**, I want to set a case's priority and assign it to the
  responsible organisation section so the right team picks it up and urgent
  cases stand out.
- As an **operator**, I want to add comments over time so the case history is a
  complete, timestamped record for shift handover.
- As an **operator**, I want to mark a member of an affected patrol as
  **resigned** so we know they have withdrawn from the event; and, when a
  member's tracked status is wrong, I want to **override** it to the correct
  value (Active, Resigned, Transit, Stationed, Departed).
- As an **operator**, I want to move a member from one team to another so that,
  e.g., a scout who continues with a different patrol is tracked correctly; when
  a team is left with no active members it is automatically considered resigned.
- As an **operator viewing a patrol**, I want to see any SOS cases the patrol is
  part of so I have context.

### Primary happy path

1. Emergency phone rings. Operator opens **Nødtelefon** in the nav and clicks
   **Ny sag**.
2. They type a headline ("Forstuvet ankel ved post 4") and a short description,
   and save. The case appears under **Åbne sager**.
3. They set priority to **yellow** and assign it to the **Samarit** section.
4. They search and associate the caller's patrol; the patrol's members appear.
5. They mark the injured scout as **Resigned**.
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
- Only cases for the current event year are shown (legacy filters messages by
  `msg.Time().Year()`; the current platform scopes by year slug).

## 6. Requirements

### Functional

- [ ] Create an SOS case with headline + description (`PUT /api/sos`).
- [ ] Edit a case headline (`POST /api/sos/headline`).
- [ ] Close and reopen a case (`POST /api/sos/close`, `POST /api/sos/reopen`).
- [ ] Add a plain-text comment to a case (`PUT /api/sos/comment`).
- [ ] Set priority/severity green|yellow|red (`POST /api/sos/severity`).
- [ ] Assign a case to an organisation **section** (`POST /api/sos/assign`).
      The list of assignable sections comes from the sections defined on the
      Organisation page — possibly a curated subset rather than every section
      (see Open Questions).
- [ ] Associate / disassociate a patrol with a case
      (`POST` / `DELETE /api/sos/team`). Only patrols can be associated with a
      case — clans (klaner) cannot.
- [ ] Mark a member of an associated patrol as **resigned** — the interface's
      primary member action (`POST /api/sos/member`), recorded on the timeline.
- [ ] Override a member's status to any valid value (Active, Resigned, Transit,
      Stationed, Departed) to correct out-of-sync data. The non-resigned
      transitions are normally produced by other functions/events when the real
      action happens; this UI only sets them as a manual correction.
- [ ] Move a member from one team to another (updates the member's
      `currentTeamId`), recorded on the case timeline.
- [ ] When a team has no remaining active members, it is automatically
      considered **resigned** (udgået), marked by a dedicated team-resigned
      domain event (e.g. `team.resigned`) emitted when the last active member
      leaves.
- [ ] List cases grouped into open / closed with columns: headline, created,
      last activity, priority, assignee (`GET /api/sos`).
- [ ] View a single case with its full activity timeline and associated teams
      (`GET /api/sos/:id`).
- [ ] Show SOS cases associated with a patrol on that patrol's detail page
      (query cases by team, as in legacy `data.SosModel.GetByTeam`).
- [ ] All cases and events are scoped to the current event year.

### Non-Functional

- **Consistency with platform:** REST + JSON via `app.*` helpers; MySQL
  projections rebuilt from JetStream on startup; frontend via `fetchWrapper`
  and PrimeVue Lara. No Bootstrap, no `vue-good-table`, no `b-popover`
  (all legacy-only).
- **OpenAPI:** every new/changed endpoint must carry OpenAPI annotations
  (repo `.rules` requirement).
- **Auth:** operator-only; behind the existing JWT cookie auth like the rest of
  `/api`.
- **Timeliness / freshness:** an operator's view should reflect other operators'
  changes within a few seconds. The legacy platform used a websocket "dims"
  channel; the current platform has none. See Technical Considerations + Open
  Questions — default proposal is short-interval polling of the REST endpoints,
  with SSE/websocket as a possible later enhancement.
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
  `DataTable` (unstyled Lara) with two groups, **Åbne sager** and **Lukkede
  sager**; columns Overskrift, Oprettet, Sidst opdateret, Prioritet, Tildelt; a
  **Ny sag** button; row click opens the detail view. Empty state: "Ingen
  nødråb fundet".
- **Detail view — `vue/src/views/SosView.vue`** (`/sos/:id`, `props: true`,
  plus a `/sos/new` route for creation):
  - Editable headline with pencil affordance.
  - Summary card: status badge (Åben/Afsluttet), created timestamp, priority,
    assignee.
  - **Activity timeline** rendering each activity type (comment, close, reopen,
    severity, assign, associate, disassociate, memberstatus, member-move) with
    an icon and Danish label — port the legacy
    `ActivityLine` component to `vue/src/components/SosActivityLine.vue`.
  - Comment composer (headline required only when creating a new case).
  - Actions: Luk sag / Genåbn sag, Tilføj kommentar.
  - Right column cards:
    - **Tilknyttede patruljer:** team picker + per-team member list. Primary
      per-member action is a **Marker som udgået** (resign) button; a secondary
      status override control exposes all statuses (Active, Resigned, Transit,
      Stationed, Departed).
    - **Prioritet** select (green/yellow/red) and **Tildelt** select. The
      **Tildelt** (assignee) options are organisation sections loaded from the
      backend (the Organisation sections, optionally filtered to an assignable
      subset) — shown by section label, stored by section slug. Do **not**
      hardcode the legacy assignee list.
  - Legacy option lists to reuse (Danish): severities `green|yellow|red`.
    Member statuses are **Active, Resigned, Transit, Stationed, Departed**,
    sourced from the shared `github.com/nathejk/shared-go/types.MemberStatus`
    constants (not hardcoded in the view; served to the frontend from the
    backend). This replaces the legacy `active/waiting/transit/emergency/hq/out`
    list. (The legacy hardcoded assignee list — guide/samarit/rover/… — is
    **replaced** by organisation sections.) Severities should be confirmed with
    organizers (see Open Questions).
- **Patrol detail:** add a "Kontakt med nødtelefon" card to
  `vue/src/views/PatruljeView.vue` listing the patrol's SOS cases with created
  date, headline and open/closed badge; clicking navigates to the case.
- **State:** a Pinia store `vue/src/stores/sos.store.js` holding the case list
  and current case, with actions wrapping the REST calls via `@/helpers`
  `fetchWrapper` (replacing the legacy Vuex `dims` actions).

## 8. Technical Considerations

### Frontend (Vue 3 / TS)

- New views `SosListView.vue`, `SosView.vue`; new component
  `SosActivityLine.vue`; new store `sos.store.js`; routes in
  `vue/src/router/index.ts`; nav item in `Navigation.vue`.
- Replace legacy libs: use PrimeVue `DataTable`/`Select`/`Textarea`/`Badge`
  (auto-imported) and Tailwind; use PrimeVue overlay/popover for the per-team
  and per-member action menus instead of `b-popover`.
- All requests through `@/helpers` `fetchWrapper` to relative `/api/...` paths;
  no bare axios (legacy used axios directly).
- Live updates: implement polling in the store (e.g. refetch the open list and
  the open case on an interval) as the default; keep it isolated so it can be
  swapped for SSE/websocket later.

### BFF (Go)

- New resource handler file `go/cmd/api/sos.go` with one `<verb>SosHandler` per
  route, reading via `app.models.Sos` and writing via `app.commands.Sos`, using
  `app.ReadJSON`/`app.WriteJSON`/`app.ServerErrorResponse` etc. (Do **not**
  reuse the legacy `sos-routes.go` switch-on-`method+path` style or hand-rolled
  `json.NewEncoder`/`http.Error`.)
- New domain aggregate package `go/nathejk/table/sos/` following the standard
  per-aggregate layout (`table.go`, `consumer.go`, `query.go`, `commands.go`,
  `table.sql`, `filter.go`), plus a `sos_team` association projection
  (legacy `sosassoc`). Port the two legacy read models:
  - `sos` table (id, year, headline, description, createdAt, createdBy, status,
    severity, assigneeSectionSlug) for the list/summary and by-team lookup.
    The assignee is stored as an organisation **section slug** (FK-style
    reference into the `section` table, year-scoped); list/detail queries join
    `section` to resolve the section label for display.
  - `sos_team` association table for team↔case links.
  - The rich **activity timeline** in legacy lived only in the in-memory
    `aggregate/sos` served over websocket. Since the current platform has no
    websocket, the timeline must be persisted as a SQL projection too — add a
    `sos_activity` table (case id, seq/created-at, type, actor user id, value,
    status, comment text) so `GET /api/sos/:id` can return the full history.
- Assignee source: reuse the existing `section` projection
  (`go/nathejk/table/section`) as the list of assignable sections. The frontend
  loads these (via the existing organisation/sections read, or a small
  dedicated "assignable sections" endpoint) and the `assigned` event/payload
  carries a section slug rather than the legacy free enum. How the assignable
  **subset** is determined is an Open Question.
- Write side: a `commands.Sos` command struct that publishes domain events.
  Port the legacy events but rename subjects to the current convention
  (`NATHEJK.{year}.sos.{sosId}.{event}` per `.rules`, built with
  `github.com/jrgensen/stream/subject`, rather than the legacy `nathejk:sos.*`
  channel strings):
  `created, headline.updated, description.updated, commented, comment.updated,
  severity.specified, assigned, deleted, closed, reopened, team.associated,
  team.disassociated`, plus the member-status event (`member.status.changed`
  with SOS metadata) and the member-move event (e.g. `member.team.changed`).
  Populate `createdByUserId` from the authenticated user on every event.
- Wire the new projections into the `xstream.Mux` and `data.NewModels(...)` /
  `commands.New(...)` in `go/cmd/api/main.go`.

### API endpoints

New endpoints (all require **OpenAPI annotations**):

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/sos` | List cases (open/closed) for current year |
| GET | `/api/sos/:id` | Get one case with timeline + associated teams |
| PUT | `/api/sos` | Create case |
| POST | `/api/sos/headline` | Edit headline |
| POST | `/api/sos/close` | Close case |
| POST | `/api/sos/reopen` | Reopen case |
| PUT | `/api/sos/comment` | Add comment |
| POST | `/api/sos/severity` | Set priority |
| POST | `/api/sos/assign` | Assign responder |
| POST | `/api/sos/team` | Associate team |
| DELETE | `/api/sos/team` | Disassociate team |
| POST | `/api/sos/member` | Mark member resigned / override member status |
| POST | `/api/sos/member/move` | Move a member to another team (updates `currentTeamId`) |

(Legacy had `/api/sos/merge` and `/api/sos/split`; these are **replaced** by
the member-move flow above and are not ported. The legacy `/api/sos/sms`
position-SMS endpoint is dropped entirely.) Note the legacy read model was delivered over a
websocket, so there were **no** legacy `GET /api/sos` endpoints — the two GETs
above are new and required by the polling approach.

### Data / storage

- New tables: `sos`, `sos_team`, `sos_activity` (all `CREATE TABLE IF NOT
  EXISTS`, embedded via `//go:embed`, MariaDB, year-scoped). Rebuilt from
  JetStream on startup like every other projection.

### Dependencies & risks

- **Shared types:** legacy used `types.SosID`, `types.SosCommentID`,
  `types.Enum`, and member-status types from a local `nathejk/types`. The
  current platform uses `github.com/nathejk/shared-go/types` +
  `.../messages`. `types.MemberStatus` already exists in shared-go and must be
  **extended** with the new operational constants (Active/Resigned/Transit/
  Stationed/Departed). The SOS-specific types (`SosID`, `SosCommentID`,
  severity) and SOS message structs likely do **not** exist in shared-go yet and
  will need to be added there (a cross-repo change with its own workflow), or
  defined locally if that is acceptable. **This is the biggest unknown.**
- **Member status model:** member field status is the shared
  `github.com/nathejk/shared-go/types.MemberStatus` enum. All member-status
  constants **must** be defined in shared-go (`types/member.go`) — this feature
  does not define its own. shared-go currently has `MemberStatusNone`,
  `MemberStatusRegistered`, `MemberStatusStarted`; the new operational values
  **Active, Resigned, Transit, Stationed, Departed** need to be added there
  (e.g. `MemberStatusActive`, `MemberStatusResigned`, `MemberStatusTransit`,
  `MemberStatusStationed`, `MemberStatusDeparted`) as a shared-go change with
  its own workflow/release, then consumed here. The SOS interface is only one
  producer of `member.status.changed`: its main job is emitting **Resigned**,
  plus manual **override** to any value. The non-resigned transitions
  (Transit/Stationed/Departed/back-to-Active) are expected to be emitted by
  *other* functions when the real-world action happens (e.g. checkpoint scans,
  transport/logistics) — those producers are **out of scope** here, but they
  reference the same shared-go constants so this override and those producers
  converge on the same value. The current platform has `spejder`/`senior` +
  `patruljestatus`/`spejderstatus` projections but no per-member operational
  status lifecycle yet; we must confirm where the canonical member status lives
  and that the SOS override writes to the same projection the other producers
  use (not a SOS-local copy).
- **Member team membership (`startedTeamId` / `currentTeamId`):** instead of
  legacy merge/split, a member carries two team references: `startedTeamId` (the
  team they originally signed up / started with) and `currentTeamId` (the team
  they currently belong to). **Moving a member** between teams updates
  `currentTeamId` only. A team's active membership is the set of members whose
  `currentTeamId` points at it and whose field status is active; when that set
  becomes empty a dedicated **team-resigned event** (e.g. `team.resigned`) is
  emitted and the team's projection is marked resigned — resignation is an
  explicit event on the log, not merely derived on read. This requires either
  extending the existing `spejder`/`senior` projections with these two columns
  (and a move event, e.g. `member.team.changed`) or confirming they already
  exist, plus a consumer that watches active membership and publishes
  `team.resigned`. The SOS timeline should record both moves and resignations as
  activities.
- **No websocket:** the entire legacy live-update UX depended on the `dims`
  websocket. Replacing it with polling is a real behavioral change; acceptable
  for an internal, low-volume tool but worth confirming.
- **Backwards compatibility:** none required — this is net-new in the current
  platform; legacy code is reference only.

## 9. Success Metrics

- Operators log ≥ 90% of emergency-phone calls as SOS cases during the event
  (qualitative: paper backup essentially unused).
- Every case has a complete timeline (create → resolution) with no gaps;
  spot-check after the event shows shift handovers were possible from the tool
  alone.
- No P0/P1 incidents caused by the tool during the event (it must be dependable
  under stress).

## 10. Rollout / Task Breakdown

Phased so value lands early and the risky shared-type/member-status work is
isolated:

- **Phase 1 — Core case management:** create/list/view, headline, close/reopen,
  comments, severity, assignee, timeline persistence. Delivers a usable
  dispatch log.
- **Phase 2 — Teams & members:** associate/disassociate patrols, per-member
  status, moving a member between teams (`currentTeamId`) with automatic
  resigned derivation, and the "Kontakt med nødtelefon" card on the patrol page.
- **Phase 3 (optional) — Live updates:** replace polling with SSE/websocket if
  polling proves insufficient.

Proposed tasks to create in `roadmap/tasks/open/` (not created yet):

- [ ] Task: Extend `github.com/nathejk/shared-go/types.MemberStatus` with Active/Resigned/Transit/Stationed/Departed
- [ ] Task: Define/agree SOS shared types & message structs (shared-go vs local)
- [ ] Task: BFF — `sos` + `sos_team` + `sos_activity` projections & schemas
- [ ] Task: BFF — `commands.Sos` write side + JetStream subjects (year-scoped)
- [ ] Task: BFF — SOS REST handlers (`go/cmd/api/sos.go`) with OpenAPI annotations
- [ ] Task: BFF — wire SOS projections/commands into `cmd/api/main.go`
- [ ] Task: Frontend — `SosListView` + `sos.store.js` + `/sos` route + nav item
- [ ] Task: Frontend — `SosView` detail with timeline + `SosActivityLine`
- [ ] Task: Frontend — team association & per-member status UI
- [ ] Task: BFF+Frontend — move member between teams (`currentTeamId`) + resigned derivation
- [ ] Task: Frontend — "Kontakt med nødtelefon" card on patrol detail
- [ ] Task: Confirm severity list & member-status ownership with organizers
- [ ] Task: Decide & implement which organisation sections are assignable to cases

## 11. Open Questions

- **Live updates:** is short-interval polling acceptable, or do we need
  SSE/websocket parity with the legacy real-time experience from day one?
- **Shared types:** should `SosID`/`SosCommentID`/severity/member-status
  types and SOS message structs be added to `github.com/nathejk/shared-go`, or
  defined locally in this repo? Who owns the shared-go change?
- **Member status:** does the current platform already model a per-member
  field-status lifecycle (active/transit/out/…) and the `startedTeamId` /
  `currentTeamId` team references? If yes, SOS should write to it; if no, do we
  introduce them here (extending `spejder`/`senior` + a `member.team.changed`
  event) or keep status SOS-local?
- **Resigned event:** the platform emits a dedicated team-resigned event when
  the last active member leaves a team. Open points: what is the exact
  subject/name (`team.resigned`?) and payload, which component is responsible
  for detecting "no active members left" and publishing it, and can it also be
  emitted directly (e.g. an operator manually resigning a team) rather than only
  as a side effect of the last member moving/going out?
- **Assignable sections:** the assignee is an organisation section, but "maybe
  not all sections" — how do we determine the assignable subset? Options: a flag
  on the section (e.g. `assignable`/`isBeredskab`), all descendants of a
  designated parent section (e.g. a "Nødtelefon"/"Beredskab" root), a
  configurable allowlist, or simply all sections. Also: what happens to a case's
  assignee if that section is later renamed or deleted on the Organisation page?
- **Member status ownership:** member statuses are the shared-go
  `types.MemberStatus` constants (Active, Resigned, Transit, Stationed,
  Departed — to be added to shared-go). Which existing/other functions own the
  non-resigned transitions (Transit/Stationed/Departed), and where does the
  canonical member status live so the SOS override and those producers write to
  the same projection?
