# PRD 010 — Kort: defining the maps we hand out

**Status:** done
**Author:** agent session (with knj)
**Created:** 2026-09-02
**Last updated:** 2026-09-03
**Approved:** 2026-09-03
**Shipped:** 2026-09-03
**Target users:** organizer (kortansvarlig, postansvarlig), and — indirectly — participants using the hej-app

<!--
Status must match the folder this file is in: draft/, doing/ or done/.
Leave Approved blank until the PRD moves to doing/, and Shipped blank until it
moves to done/. See roadmap/prd/README.md for the lifecycle.
-->

---

## 1. Summary

Give HQ a place to define the physical maps we print and hand out during the
event — their name, which set they belong to, their format, the geographic
rectangle(s) they cover, and **which checkpoints are drawn on them** — managed
from a settings modal on the `/kort` route.

The checkpoint-to-map relation is the part with reach beyond HQ: together with
the checkgroup it is what lets the hej-app reveal exactly the checkpoints a
patrol is holding a map for, and no more.

## 2. Problem & Motivation

**What problem does this solve?**

The maps exist today only as physical artefacts and as knowledge in the head of
whoever drew them. Nothing in the system knows that "Kort 3, bagside" shows
posts 5A–5C, or that the crew map covers the whole area while a patrol map
covers one corner of it. Three consequences follow:

- **The hej-app cannot reveal checkpoints correctly.** Not knowing where they
  are going is the race dynamic; a patrol must see what its current map shows
  and nothing beyond it. Reveal happens at two granularities today — the whole
  checkgroup at once, and the map that was handed over — and only one of them is
  modelled. Without the map, the app has to approximate the other.
- **Nobody can check the sheets line up before printing.** The maps in a set are
  meant to cover the race area between them, overlapping slightly at the edges.
  Today that is verified by laying paper on a table, late, and it is easy to
  leave a seam between two adjacent sheets — a strip of ground no sheet shows.
- **Nothing knows which sheet a patrol is holding.** We do not track — and do not
  need to track — which post hands out which map; that varies and is the crew's
  business. The sheet becomes known the moment its **QR code is linked to the
  team**, which already happens today. But a QR can only be linked to a *map* if
  maps exist as records, so the app currently learns nothing from the link beyond
  the team. Defining the sheets is what turns that scan into knowledge.

**Why now?**

`/kort` already renders every checkpoint on a Leaflet map with positions
editable in place (`vue/src/views/KortView.vue`), and checkpoints already carry
`latitude`/`longitude` (`go/nathejk/table/checkpoint/table.sql`). The one thing
missing between what HQ knows and what the hej-app needs is the map itself.
Defining it now, in the season before the print deadline, means the app work can
proceed against real data rather than a hardcoded table.

**Evidence.** The reveal rule in the hej-app is currently derived from
checkgroup membership alone. That is correct as far as it goes — every
checkpoint in a checkgroup is revealed together — but it cannot express a
skitse, which shows the next group of checkpoints on a slip of paper with no QR
code at all. Every year the printed maps and the app's idea of them drift.

## 3. Goals

- HQ can describe every map handed out during the event, and that description is
  the single source of truth for what is printed.
- The set of checkpoints drawn on a given map is explicit, so a consumer can
  answer "what may this team see now?" without inference.
- An organizer can see, before printing, whether the maps in a set leave a seam.
- The data is available to the hej-app over a stable read endpoint.

## 4. Non-Goals

- **Generating or rendering the printable maps.** Drawing happens in external
  cartography tooling; this PRD describes maps, it does not produce PDFs.
- **Uploading map artwork.** No file storage in this PRD. It may follow once the
  definitions exist.
- **Changing the hej-app's reveal logic.** This PRD delivers the data and the
  endpoint; consuming it is work in the app's own repo, tracked separately.
- **QR codes and printed copies.** Every printed map carries a unique QR code,
  and scanning an unknown one prompts for mapping it to a patrulje
  (`NATHEJK.*.qr.*.scanned`, `go/nathejk/table/scan/`). Recording *which map* a
  QR sits on is the natural follow-up to this PRD and is deliberately not part
  of it — this PRD only makes the list of candidate maps exist for it to pick
  from. Mentioned here only as context.
- **Recording which post hands out which map.** Not merely out of scope — not
  wanted. Which post hands out which sheet is not reliably known and varies in
  practice, and it is not needed: the sheet a team holds is established when its
  QR code is linked to the team, not by inferring it from where they were.
- **Per-team map issue tracking.** Who received which copy is not tracked.

## 5. User Stories & Scenarios

- As a **kortansvarlig**, I want to define each map we print, so that the app and
  the post crews work from the same list I send to the printer.
- As a **kortansvarlig**, I want to see which checkpoints are on which map, so
  that no checkpoint is missing from every map, or on the wrong one.
- As a **kortansvarlig**, I want to see the maps in a set drawn together, so I
  discover a seam at my desk and not on the course.

### Primary scenario — defining the patrol maps

The kortansvarlig opens `/kort` and clicks the settings button in the toolbar. A
modal lists the maps defined for the year; it is empty the first time. She adds
one, names it `Kort 1 — Start til Post 2`, puts it in the **Patruljer** set, in
**A4** format. She draws its extent by picking the top-left and bottom-right
corners on the map behind the modal.

She then ticks the checkpoints drawn on it. The map behind the modal highlights
exactly those, so a mistake is visible rather than merely saved. She drags the
map to its place in the handout order. She repeats this for the remaining maps,
and for the crew set.

With a set defined, she toggles the set's overlay: every extent in it is drawn
at once, and the space between them shaded. She sees a thin unshaded seam
between `Kort 3` and `Kort 4`, nudges a corner, and it closes.

### Scenario — a skitse

Post 3 hands out a small hand-drawn slip showing the three posts of group 4. She
defines it as format **skitse** with no extent and three checkpoints. It has no
QR code, so it is never scanned — those checkpoints are revealed when the
*previous* checkpoint is scanned. The hej-app can still use the skitse's
checkpoint list to know what to reveal, because the reveal comes from the
checkpoint list, not the geometry and not a scan of the map itself.

### Scenario — a double-sided A3

`Kort 5` is printed on both sides: posts 6A–6C on the front, 7A–7B on the back.
It is **one** map — one sheet handed over, one QR code, one reveal — so it is one
record with five checkpoints. But its two sides show two different rectangles, so
it carries **two extents** rather than one. Extents are therefore a list, not a
pair of corners, and a map may have zero, one or two of them.

The two extents are simply two areas. Nothing records which is the front and
which the back, and the checkpoints are not split per side — both sides are
handed over together, so the distinction has no consumer.

### Scenario — the same checkpoint on two maps

Post 2B is on the patrol map for group 2 and again on the crew map that covers
the whole area. Both are correct. Adjacent maps in the same set also overlap a
little by design, so the same checkpoint appearing twice within one set is
normal and not flagged.

### Edge cases and failures

- **A checkpoint on no map.** Not an error — a checkpoint may be defined before
  its map is drawn — but it is surfaced in the modal as an unassigned list, so it
  cannot be forgotten silently.
- **A map with no checkpoints.** Allowed; a pure overview map for drivers
  legitimately has none, and a half-finished map should be saveable.
- **A checkpoint with no position.** It can still be assigned to a map; it just
  cannot be drawn. Flagged, not blocked.
- **Corners drawn backwards.** Whichever two corners the operator picks, they are
  normalised to a true north-west/south-east pair on save.
- **Deleting a checkpoint** removes it from every map that referenced it, the
  same way `checkgroup.*.deleted` already cascades in the checkpoint consumer.
- **Deleting a map** does not touch its checkpoints.
- **Two operators editing at once.** The modal is a dirty-state editor and must
  defer live updates while open — see Technical Considerations.

## 6. Requirements

### Functional

- [ ] A **kortsæt** entity exists per year: a named set with a sort order and an
      **optional `teamType`**. Sets are created by the operator, not chosen from a
      fixed enum — most years there are two (a spejder set and a crew set), but a
      year may have three.
- [ ] `teamType` marks a set as belonging to a team type (`patrulje`, `klan`),
      using the existing `TeamType` from `nathejk/shared-go`. It is how a consumer
      identifies the spejder maps without matching on a set's name. It is optional:
      the crew/gøgler/bandit set has no team type. More than one set may carry the
      same team type.
- [ ] A **kort** entity exists per year, belonging to exactly **one** set, with:
      name, format, a list of zero to two extents, an optional note, sort order,
      and a list of checkpoint ids.
- [ ] Format is one of `a4`, `a3`, `skitse`, `andet`.
- [ ] Sets can be created, renamed, reordered and deleted, and a set's `teamType`
      can be set or cleared.
- [ ] A settings button on `/kort` opens a modal for managing the year's maps.
- [ ] Maps can be created, renamed, edited, reordered by dragging, and deleted.
- [ ] Sort order is the handout order along the route, within a set.
- [ ] An extent can be set by picking two corners on the underlying Leaflet map,
      added a second time for the reverse side, and removed.
- [ ] Checkpoints are assigned to a map by selection, grouped by checkgroup,
      matching the existing context-menu grouping in `KortView.vue`.
- [ ] Selecting a map highlights its checkpoints and draws its extents on the map
      behind the modal.
- [ ] A checkpoint may belong to any number of maps, including several in one set.
- [ ] The modal lists checkpoints belonging to **no** map in a given set.
- [ ] A per-set overlay draws every extent in the set at once and shades the gaps
      between them, so a seam is visible.
- [ ] The modal warns when **no single map in a set contains all of a checkgroup's
      checkpoints**. Since a checkgroup reveals as a whole, a patrol would
      otherwise be shown checkpoints it holds no sheet for. Note the shape of the
      test: it is satisfied by *any one* map containing them all, so overlapping
      sheets that each contain the whole group are fine, and the checkpoints may
      sit in different areas (extents) of that map. It is about map membership,
      never geometry.
- [ ] Deleting a set is refused while it still holds maps.
- [ ] A read endpoint exposes maps with their checkpoint ids for the hej-app.
- [ ] Changes propagate live to other open HQ sessions (see Non-Functional).

### Non-Functional

- **Live updates are mandatory.** The projection is registered in the
  `projections` slice in `cmd/api/main.go` so `live.NotifyAll` signals it, and the
  frontend loads maps through `useLiveResource` with
  `dependsOn: ['kort', 'kortsaet', 'checkpoint', 'checkgroup']`. `KortView` is
  already a deferred-apply page; the settings modal extends the same rule — while
  it is open and dirty, incoming payloads are held and applied on close, and the
  UI says updates are paused. See `.rules` → Live updates.
- **Print deadline is the hard constraint.** The data must be enterable well
  before print.
- **Danish UI text**, English code, per repo convention.
- **Read path must be cheap.** The hej-app polls the read endpoint on a race day
  for every team; the response is small and year-scoped.

## 7. UX / UI Notes

Everything lives on the existing `/kort` route — no new route. The map stays
visible and is the primary feedback surface; the modal is a side panel over it,
not a full-screen dialog that hides the thing being described.

- **Entry point:** a gear button in the existing `.edit-toolbar` in
  `KortView.vue`, beside the current edit-mode button. Map editing and marker
  dragging are mutually exclusive — opening one disables the other, since both
  own marker interaction.
- **Layout:** a PrimeVue `Dialog` positioned right, or a `Drawer`, at ~420px: the
  year's maps grouped by set and draggable to reorder (using the `vuedraggable`
  already in the project), and an editor for the selected map.
- **Editor fields:** name; the set it belongs to; format as a `SelectButton`; the
  extent list with a "Vælg på kort" button per extent and an "Tilføj område"
  action; a note field; and the checkpoint picker grouped by checkgroup with
  per-group select-all.
- **Set editor:** a set's name, its sort order, and an optional team type. The
  team type is a `Select` with an empty option, labelled to make clear it is what
  marks the set as the spejder set rather than cosmetic.
- **On the map:** the selected map's extents draw as translucent `L.rectangle`s;
  its checkpoints render highlighted while all others fade. This is what makes an
  error obvious before it is printed.
- **Seam check:** a per-set toggle draws all extents in the set together with the
  gaps between them hatched. Deliberately visual — see Technical Considerations
  for why there is no percentage.
- **Unassigned checkpoints:** a collapsible list at the bottom of the modal, each
  entry clicking through to select it on the map.
- **Split-checkgroup warning:** an inline warning beside the set, naming the
  checkgroup and the maps it is spread across, with the offending checkpoints
  highlighted on the map. A warning, never a block — see below.

## 8. Technical Considerations

### Sets are an entity, because `teamType` lives on them

An earlier draft made the set a free-text `setName` column on each map, derived
by grouping. That does not survive `teamType`: the team type is a property of the
set as a whole, so storing it per map means five maps in one set each carry a copy
that can disagree, and "which set is the spejder set?" becomes a question with
five possibly-conflicting answers. It also let `Patruljer` and `patruljer` drift
into two sets.

So `kortsaet` is its own small entity and a map references it. Sets stay fully
dynamic — the operator creates them, and a third set in some future year needs no
code change — but each one exists once, with one name and one team type.

`teamType` reuses `TeamType` from `nathejk/shared-go` (`patrulje`, `klan`) rather
than a new enum, so the marking means the same thing here as everywhere else in
the system. It is nullable, and stays nullable: the crew/gøgler/bandit set covers
people who are not a team type at all, and forcing a value on it would invent a
fictional one.

The payoff is for the hej-app and for QR linking: the maps a patrol may be holding
are those in the sets marked `patrulje`, found by team type and never by matching
a Danish set name that an organizer is free to rename mid-season.

**A map belongs to exactly one set** — a sheet is printed for one audience — so
the reference is a single `kortsaetId`, not a list.

**Several sets may share a team type**, and that is deliberately not constrained.
The team type is a filter, not a key: it yields the *candidate* maps for a team
type, which is precisely what QR linking needs — when an unknown QR is scanned,
every map across every `patrulje` set is offered as a possibility. Enforcing one
set per team type would buy a uniqueness nobody consumes and would block a year
that legitimately splits its patrol maps into two sets.

**Klaner normally use the crew set**, which carries no `teamType` at all. So
`teamType` is best read as *"this set is specifically for this team type"*, not
*"only this team type uses it"* — an unmarked set is the general one, and klaner
draw from it. In practice that means a typical year has one set marked `patrulje`
and one unmarked, and `klan` may never be used.

The consequence for consumers: **filtering by `teamType == klan` will usually
return nothing, and that is not an error.** A caller looking for a klan's candidate
maps must fall back to the unmarked set rather than concluding there are none. The
field is kept nullable and unconstrained precisely so the year that does print
dedicated klan sheets needs no code change.

### Maps and checkgroups are both reveal units

The checkgroup already is a reveal unit and stays one: every checkpoint in a
checkgroup is revealed at the same time. The map is a *second*, independent one,
and the two do not nest — a skitse shows the next group of checkpoints on a slip
with no QR, and a double-sided A3 spans two groups.

So this PRD adds a unit rather than replacing one. The checkgroup keeps its
existing meaning; the map carries the handout. The consuming app combines them.

### What reveal looks like, and why it is not implemented here

Two triggers exist, and the map only participates in the first:

- A **map's QR is linked or scanned** → that map's checkpoints become visible.
  This is also the only way the system learns which sheet the team is holding;
  there is no handout record to consult.
- A **checkpoint is scanned** → its whole checkgroup becomes visible, and the
  checkpoints on any skitse handed out there become visible, since a skitse has
  no QR of its own to scan.

The asymmetry is worth stating plainly: a sheet with a QR is known because it was
scanned, and a skitse is known only by inference from the previous checkpoint. That
is why the skitse's checkpoint list must be recorded even though it has no QR and
no extent — it is the only trace of it in the system.

This PRD makes both triggers expressible and implements neither. Keeping the rule
in the consuming app, over a plain read model, avoids inventing a reveal-policy
engine in HQ before we have a second caller for it.

### Why extents are a list

A double-sided sheet is one map — one handover, one QR, one reveal — but two
drawn rectangles. Modelling extent as a single pair of corners would force the
sheet to be split in two just to record its geometry, which would then
double-count the handover. A list of zero-to-two extents keeps the sheet whole.

### Why checkpoints are a column, not a join table

`kort_checkpoint` is not modelled as a table. The relation is read in exactly one
direction — given a map, which checkpoints — and we will never ask for all maps
containing a given checkpoint. A JSON array column on `kort` therefore costs one
row per map instead of one per assignment, and the read endpoint needs no join.

The tradeoff is the cascade: deleting a checkpoint must remove its id from every
map's array. For a single checkpoint that is `JSON_SEARCH` + `JSON_REMOVE`, keyed
off the id in the event's subject, which is order-independent and cheap.

Deleting a **checkgroup** is the hard half, and it is handled on *read* rather
than on write. That event names only the group — its checkpoints are removed by
the checkpoint projection in one `DELETE ... WHERE checkgroupId = ?`, with no
per-checkpoint events — so a write-side cascade would have to learn the group's
members from another projection's table. Both ways of doing that fail: a
correlated subquery is correct only if the two projections happen to run in the
right order, which nothing guarantees; and `JSON_TABLE` over the array is broken
for this on MariaDB 10.8, where a `JSON_TABLE` correlated with a column is not
re-evaluated per row and an `UPDATE` using it writes one row's result into others
(verified — a sheet with no checkpoints acquired one).

So `GET /api/kort` filters out ids that no longer resolve, at the cost of one
indexed query. That is order-independent, and it self-heals stale ids from any
cause — a checkpoint deleted while the API was down, a half-finished replay. A
stale id left in the column is inert: it names a checkpoint that does not exist,
and the next edit to the sheet rewrites the array anyway.

### A checkgroup must fit on one map

Because a checkgroup is revealed as a whole, a patrol that is shown a checkgroup
no single sheet covers sees checkpoints it holds no map for. That is a printing
mistake with a race-day cost, and it is cheap to detect: for each set, warn when
no single map in that set contains all of a checkgroup's checkpoints.

Note the test is existential, not partitioning — *some* map must contain the whole
group. Two overlapping sheets that both contain it are fine, which matters because
overlap is deliberate.

Note too what is *not* checked. A checkgroup's checkpoints may well sit in two
different areas of the same sheet — that is exactly what the two extents of a
double-sided A3 are for — so the check is about map membership, not geometry.
Comparing positions here would produce false alarms on every double-sided sheet.

It is a warning and never a block: a half-entered set trips it constantly, and a
save that refuses to complete during data entry is worse than a visible warning.

### Coverage: an overlay, not a percentage

An earlier draft computed a coverage *percentage* server-side over a grid. That
is rejected. A percentage needs a denominator, and the race area is simply "where
we are" — it has no recorded boundary, and all maps and checkpoints lie inside it
by construction. The denominator would end up being the bounding box of the maps
themselves, against which coverage is always ~100%.

More importantly it would measure the wrong thing. Adjacent maps overlap a little
by design, so the failure that matters is not a shortfall at the edge of the area
but a **seam between two sheets** — which barely moves a percentage while losing
a patrol. Drawing every extent in a set at once and shading the space between
them shows a seam immediately, needs no denominator, and lives entirely in
Leaflet with no geometry in Go and no endpoint.

### Frontend (Vue 3 / TS)

- `vue/src/views/KortView.vue` — add the settings button, corner-picking mode,
  extent rectangles, checkpoint highlighting, per-set overlay.
- `vue/src/components/kort/KortSettingsDialog.vue` — new; the modal.
- `vue/src/composables/useKort.ts` — wraps `useLiveResource` for the map list so
  the view and the modal share one cached source.
- Extend the existing deferred-apply mechanism (`applyDeferred`,
  `syncMapIfDeferred`) to cover modal dirty state rather than adding a second,
  parallel mechanism.

### BFF (Go)

- `go/nathejk/table/kort/` — new projection, following the established shape:
  `table.go`, `consumer.go`, `query.go`, `command.go`, `table.sql`, `filter.go`,
  mirroring `go/nathejk/table/checkpoint/`.
- Consumes `NATHEJK.*.kort.*.created|updated|deleted`,
  `NATHEJK.*.kortsaet.*.created|updated|deleted` and — to cascade —
  `NATHEJK.*.checkpoint.*.deleted` and `NATHEJK.*.checkgroup.*.deleted`.
- **The projection and its event message types both start local to this repo, and
  move to `nathejk/shared-go` independently.** The messages will very likely go
  first — the hej-app needs to *read* kort events long before anything outside HQ
  needs to materialise them — and the projection follows if and when a second
  service wants the same read model. Neither move is a rewrite: the messages carry
  no HQ-only types, and the consumer makes no HQ-only assumptions. Starting local
  keeps both churning in one place and removes the dependency bump from the
  critical path.
- Commands dirty-check against the read model before publishing, so a no-op save
  publishes nothing and emits no live signal.
- Handlers in `cmd/api/`, registered in `routes.go`; the projection is added to
  the `projections` slice in `main.go`.

### API endpoints

All new endpoints require **OpenAPI annotations**, per repo convention.

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/kort` | List the year's sets and maps with checkpoint ids and extents |
| POST | `/api/kort` | Create a map |
| PUT | `/api/kort/:id` | Update name, set, format, extents, note |
| DELETE | `/api/kort/:id` | Delete a map |
| PUT | `/api/kort/:id/checkpoints` | Replace the map's checkpoint list |
| POST | `/api/kortsaet` | Create a set |
| PUT | `/api/kortsaet` | Reorder sets (ids in order) |
| PUT | `/api/kortsaet/:id` | Update a set's name and team type |
| PUT | `/api/kortsaet/:id/kort` | Reorder the set's maps (ids in order) |
| DELETE | `/api/kortsaet/:id` | Delete a set; refused while it holds maps |

There is deliberately no `GET /api/kort/:id` and no `GET /api/kortsaet`: the whole
year is a handful of records, `GET /api/kort` returns all of it, and the modal and
the hej-app both work from that one cached response. A single-record read would be
a second code path serving no caller.

**Reordering is `PUT` on a collection, not `…/sorted`.** An earlier draft specified
`PUT /api/kort/sorted` and `PUT /api/kortsaet/sorted`, mirroring the existing
`/api/checkgroups/sorted`. Those cannot exist: httprouter panics at startup on a
static segment beside a wildcard at the same level, so registering them next to
`/api/kort/:id` would stop the whole API booting — not misroute, not 404. The
checkgroup routes escape it because `checkgroup`/`checkgroups` are two different
segments, and Danish gives us no such escape: "kort" and "kortsæt" are their own
plurals, and inventing "kortsaets" to satisfy a router is worse than putting the
order on the collection that has it. A map's order therefore lives under its set,
which is also where handout order is actually meaningful.

A consequence worth having: a set that happens to be *named* "sorted" stays an
ordinary set, reachable at `/api/kortsaet/sorted`. The rejected design could not
have expressed it.

A reorder is one event for the whole collection, not N single-field updates — one
operator gesture, and N events would let a replay observe orders that never
existed on screen.

`GET /api/kort` returns sets with their maps nested, so the hej-app gets the
`teamType` marking and the maps in one round trip. It is year-scoped by the
existing `X-YearSlug` header, like every other endpoint. There is no coverage
endpoint — the seam check is client-side.

Deleting a set that still holds maps is refused rather than cascading; losing a
season's map definitions to a mis-click is not worth the convenience.

### Data / storage

Two tables, created via `CREATE TABLE IF NOT EXISTS` from an embedded
`table.sql`:

- `kortsaet` — `id`, `year`, `version`, `name`, `teamType` (nullable),
  `sortOrder`.
- `kort` — `id`, `year`, `version`, `kortsaetId`, `name`, `format`, `note`,
  `sortOrder`, `checkpointIds` (JSON array of checkpoint ids), `extents` (JSON
  array of 0–2 objects, each with north-west and south-east lat/lng).

A set is a row rather than a string, so a third set needs no code change while
still existing exactly once. Coordinates inside the JSON are stored as numbers at
the precision a printed rectangle needs; the `FLOAT` columns on `checkpoint` are
the existing precedent.

Both are year-scoped and nothing is carried between years: **the event is in a
different area each year**, so last year's maps and extents have no meaning this
year and there is no cloning feature to build.

Read models rebuild from JetStream on restart, so there is no migration. Note the
standing caveat from `.rules`: `CREATE TABLE IF NOT EXISTS` never alters an
existing table, so a later column addition needs a dropped table in dev.

### Dependencies & risks

- **The hej-app is a separate repo.** This PRD is only useful once the app
  consumes it; the endpoint should ship early so app work can proceed in
  parallel, and its shape should not change afterwards without warning.
- **JSON cascade correctness.** Removing a deleted checkpoint's id from every
  map's array is the one piece of non-obvious SQL here and deserves a test — and
  it earned one: `JSON_SEARCH` must match whole values, or deleting `cp-1` also
  removes `cp-10`.
- **The two-extent cap is a UI convention, not a storage constraint.** A sheet has
  at most two sides, so the editor offers at most two. The column is a JSON array
  and would hold more, which is the right way round: a future three-panel fold
  needs no schema change.
- **Two owners of marker interaction.** Marker dragging and corner picking both
  claim the map; mutual exclusion is a requirement, not a nicety.
- **Data entry burden.** ~10–15 maps per year, each with a handful of
  checkpoints, and it starts from nothing every year since the area changes.
  Per-checkgroup select-all keeps it to minutes.
- **Shading the gaps is real geometry.** Drawing the extents is trivial; shading
  the *complement* of their union means computing that union, which Leaflet will
  not do. Options are a polygon-clipping dependency, or a canvas/grid
  approximation. This is why the overlay is sequenced last and marked cuttable:
  drawing the rectangles as outlines alone already lets a human spot a seam, and
  is the fallback if the shading proves expensive.
- **A set with no `teamType`** is the normal case — it covers crew, and klaner draw
  from it too — so a consumer must not treat the field as always present, and must
  not conclude a team type has no maps just because no set is marked with it.

## 9. Success Metrics

- Every checkpoint with a position is on at least one map in every set before the
  print deadline — measured directly by the unassigned list reaching zero.
- No seam is visible in any set's overlay before printing.
- No split-checkgroup warning is outstanding at the print deadline.
- The hej-app derives its map-based reveal from `GET /api/kort` with no hardcoded
  checkpoint list.
- The hej-app and QR linking find the candidate patrol maps via the sets'
  `teamType`, not by name.
- No map is reprinted during the event because a sheet was missing ground.

## 10. Rollout / Task Breakdown

Sequenced so the read endpoint — the thing another repo waits on — lands early,
and the seam overlay, the most speculative part, lands last and can be cut
without losing the feature.

1. Projection and local message types (backend foundation).
2. CRUD endpoints and the read endpoint (unblocks the hej-app).
3. The modal, without extents (checkpoint assignment is the valuable half).
4. Extents and corner picking on the map.
5. Per-set seam overlay.

Proposed tasks for `roadmap/tasks/open/`:

- [ ] Task: Add `nathejk/table/kort` projection with local message types (both
      shaped for a later, independent lift to shared-go) and register it in
      `projections`
- [ ] Task: Add `kortsaet` set entity with optional `teamType`, incl. commands and
      CRUD endpoints
- [ ] Task: Cascade checkpoint and checkgroup deletion into `kort.checkpointIds`
- [ ] Task: Add kort commands with dirty-checking
- [ ] Task: Add kort CRUD + read handlers and routes with OpenAPI annotations
- [ ] Task: Add `useKort` live resource composable
- [ ] Task: Add settings button and `KortSettingsDialog` to `/kort`
- [ ] Task: Checkpoint assignment UI with per-checkgroup select-all
- [ ] Task: Map set management UI — create/rename/reorder sets, set team type
- [ ] Task: Extent corner picking and rectangle rendering, incl. reverse side
- [ ] Task: Defer live payloads while the settings modal is dirty
- [ ] Task: Per-set extent overlay with gaps shaded
- [ ] Task: Warn when a checkgroup is not contained within a single map of a set
- [ ] Task: Document `GET /api/kort` for the hej-app team

## 11. Open Questions

None outstanding. Earlier questions — the race-area denominator, front/back
labelling, per-side checkpoints, carrying maps between years, set uniqueness,
whether a map can span sets, where maps are handed out, and how klaner are served
— have been resolved into the sections above.
