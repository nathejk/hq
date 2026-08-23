# PRD 008 — Member notes: documenting what was agreed

**Status:** draft
**Author:** agent session (with the Hønsegården crew and the nødtelefon owners)
**Created:** 2026-08-23
**Last updated:** 2026-08-23
**Approved:**
**Shipped:**
**Target users:** organizer (Hønsegården crew), organizer (nødtelefon operator)

<!--
Status must match the folder this file is in: draft/, doing/ or done/.
Leave Approved blank until the PRD moves to doing/, and Shipped blank until it
moves to done/. See roadmap/prd/README.md for the lifecycle.
-->

---

## 1. Summary

A prose comment trail on each scout — what was agreed with a guardian, what was said on the
phone, anything the next shift needs to know — written and read wherever that scout appears:
Hønsegården, the nødtelefon's case card, and the patrol page. Delivered as one component over
one live endpoint, which also means extracting the member detail modal out of `SosTeamCard.vue`
and making it live.

## 2. Problem & Motivation

**What problem does this solve?** The shelter's real work is not recorded anywhere. A crew
member rings a scout's mother at 01:20, agrees she will come at 06:00, agrees the scout may
sleep in the hall rather than a tent, and learns the father must not be called. None of that
fits in a status, a placering, or a case — so today it lives on paper, in one person's head, and
is gone at shift handover. The scout's status says `sheltered` and the screen says `Telt 4`,
which is true and answers none of the questions the next crew member will have at 04:00.

The nødtelefon has the same gap from the other side: an operator who spoke to a patrol leader
about a specific scout has nowhere to put what was said *about that scout*. Case comments exist,
but they are about the call, and the shelter has no case at all — PRD 007 made the shelter's
write path deliberately case-free, because the shelter may receive a scout nobody opened a case
about.

**Why now?** PRD 007 shipped the shelter's screen and closed the custody chain, and the first
thing the crew asked for on seeing it was somewhere to write down what they had arranged.
Everything technical this needs already exists — the member endpoint, the event conventions, the
live plumbing — so this is a small feature sitting on a lot of finished groundwork.

**Evidence.** Requested directly by the Hønsegården crew after using the screen (2026-08-23).
Two facts from the codebase that shape it: the member detail modal is **not a component** — it is
a `detail` ref and a `<Dialog>` inside the 956-line `SosTeamCard.vue` — and it is **not live**:
`loadDetail()` is a plain `http.get` into a local ref, so two crew members looking at the same
scout cannot see each other's work. For a shared trail that is disqualifying.

## 3. Goals

- Anything agreed about a scout is written down where the next person will look for it, without
  needing a case to hang it on.
- One trail, readable and writable from every screen that shows a member — the shelter, the case
  card, the patrol page.
- Notes are visible *as existing* without opening anything, so nobody has to click through forty
  scouts to discover that one has instructions.
- Two crew members working at once see each other's notes.
- Shift handover stops depending on one person's memory.

## 4. Non-Goals

- **Case comments.** Member notes do **not** appear on an SOS timeline (decided 2026-08-23). The
  case card will *read* a scout's notes because it shows that scout, but nothing is copied onto
  the case, so there is one text and one place it can be edited.
- **A structured pickup time.** Considered and dropped: an "expected pickup 06:00" field could be
  sorted and alarmed on, but the crew wants to write what was agreed in words, and half the
  agreements are not a time. Prose only.
- **Notifications.** Nothing is sent to anybody. A note is a record, not a message.
- **Attachments, photos, formatting.** Plain text.
- **Per-note visibility.** Every note is visible to everybody with HQ access. There is no
  private-note concept and should not be one: a note nobody else can read fails the only purpose
  this has.
- **Deletion.** See §6; corrections are edits, and there is no delete in v1.
- **Notes on klaner, gøglere or vehicles.** Scouts (`spejder`) only, matching the lifecycle. The
  event shape does not preclude a second entity later.

## 5. User Stories & Scenarios

- As a **shelter crew member**, I want to write down what I agreed with a guardian, so the next
  shift honours it and nobody rings the same mother twice.
- As a **shelter crew member**, I want to see at a glance which scouts have instructions, so I do
  not have to open forty of them to find out.
- As a **nødtelefon operator**, I want the notes about a scout on the case card, so I know what
  the shelter has already arranged before I ring anybody.
- As **whoever takes over at 04:00**, I want to read the trail in order, so I know what has
  happened rather than only what is currently true.

**Happy path.** A scout arrives, is accepted and placed in `Telt 4`. The crew member opens her by
clicking her name, and writes: *"Ringet til mor 01.20. Hun henter kl. 06. Må sove i hallen, hun
er bange for edderkopper. Far skal ikke kontaktes."* The note appears in the thread, and her row
in *I Hønsegården* now shows a note count and the first line of the latest note. At 04:00 the
relieving crew member sees the badge, opens her, reads the trail, and knows not to ring the
father. At 06:00 she is handed over with **Hentet af forældre** and moves to *Afsluttet* — with
her notes intact, because notes belong to the scout and not to her current status.

**Edge cases and errors.**

- **Two crew members writing at once.** Both notes land; the thread is append-only, so neither
  overwrites the other. The screen is live, so each sees the other's note appear.
- **A typo.** Editing is offered and expected to be used for exactly that. It is not a way to
  rewrite history — see §6 — and the event stream keeps every version whether or not the UI
  shows them.
- **A note on a scout with no case.** The ordinary case, and why notes are member-scoped. No case
  is created.
- **A note on a scout who is still racing.** Allowed. The nødtelefon takes calls about scouts who
  have not dropped out, and refusing a note until somebody withdraws would be arbitrary.
- **A scout moved to another patrol.** Notes follow the scout, not the patrol. They are keyed by
  member id, so a move changes nothing.
- **Unsigned notes.** Until login lands, every note is attributed to nobody (PRD 001 §6). Accepted
  on race day if login slips (decided 2026-08-23) — an unsigned trail is worth much more than no
  trail. See §8 for what this costs.
- **A very long note.** Capped (§6), with the limit shown as it is approached rather than the
  submit silently failing.

## 6. Requirements

### Functional

- [ ] A note is prose attached to one member, with an author (empty until login), a creation time
      and an edit time.
- [ ] Any member of the current year may have notes — not only sheltered scouts.
- [ ] Notes are **added** by anybody with HQ access, and require no SOS case.
- [ ] Notes are **editable**, in place, by anybody with HQ access. Ownership is not enforced
      because there is no identity to enforce it with; revisit when login lands.
      Editing is presented as a correction ("Ret") rather than as the normal way to add
      information, and the UI nudges toward a new note instead.
- [ ] Notes are **not deletable** in v1. A note about a child that turned out to be wrong is
      corrected by a further note; removing the record is a different decision and needs its own
      argument.
- [ ] The thread reads **oldest first** — a trail is a story and reads in the order it happened.
- [ ] Trimmed, and non-empty: an empty note is refused rather than published.
- [ ] Capped at **2000 characters**. Long enough for a phone call's worth of detail, short enough
      that a row snippet and a projection column stay sane.
- [ ] Every screen showing a member gives access to the same thread: Hønsegården, the nødtelefon
      case card, the patrol page.
- [ ] A member's row shows **how many notes exist and the first line of the most recent one**, so
      notes are discoverable without opening anything.
- [ ] Notes survive every status change, team move and handover, because they belong to the
      member.

### Non-Functional

- **Live.** The thread loads through `useLiveResource` and depends on the `spejder` entity type.
  No new live token is needed — the note events are spejder subjects, so the existing advertised
  set covers them, and Hønsegården's current `dependsOn: ['spejder', 'patrulje']` already
  invalidates its list when a note is written.
- **Unsaved text is never lost.** The note form is unsaved state, so the host must defer incoming
  payloads while it is dirty, as `HoensegaardView` already does for the placering field, and say
  on screen that updates are paused.
- **Privacy.** Notes will contain guardians' arrangements and children's fears. Same perimeter as
  the rest of HQ; nothing in a URL beyond the existing ids; no new sharing surface. Worth stating
  plainly because the content is more sensitive than anything else the platform stores.
- **Performance.** Tens of notes per event, not thousands. One query for a thread, one aggregate
  for the row summaries. No query per row.

## 7. UX / UI Notes

**`MemberNotes.vue`** — the thread and the form, host-agnostic, taking a `memberId`. This is the
piece that makes the host question cheap and reversible: modal today, an expanded row tomorrow if
the crew prefers it, with no second implementation.

**`MemberDetailDialog.vue`** — extracted from `SosTeamCard.vue` (the `detail` ref and the
`<Dialog>` at line 725), converted to `useLiveResource`, and given `MemberNotes` as a section. It
already shows the guardian's phone, address, birthday and full status history, which is exactly
what somebody has in front of them while ringing a parent — and note that PRD 007 removed phone
numbers from the shelter table on the grounds that this is where they belong.

**Hosts.** The scout's **name** becomes the affordance in all three places: Hønsegården's rows,
the case card's member rows (already the case), and the patrol page's roster.

**The row summary.** A note-count badge plus the first line of the latest note, in a column of its
own. This is deliberately the answer to "expandable rows" as a host: it gives the *context*
benefit — notes visible while scanning the list — without duplicating the editing UI on one
screen. Long snippets truncate with the full text in the tooltip.

**Wording.** Danish throughout: "Noter", "Tilføj note", "Ret", "Rettet 01.24". Timestamps use the
shelter's existing format (weekday, clock, elapsed) so one screen does not date things two ways.

## 8. Technical Considerations

**Data model — a new local table package `go/nathejk/table/spejdernote/`.**

Not on `spejderstatus`: that package owns status and team membership, is queued for lifting to
shared-go verbatim (task 083), and a notes table in it would make the lift a rewrite. Not on
`sos`: notes are not case-scoped, which is the whole point (§4).

- `spejdernote` table, keyed by `noteId`, holding `memberId`, `year`, `note`, `actorUserId`,
  `createdAt`, `updatedAt`. Indexed `(year, memberId, createdAt)` for the thread and
  `(year, memberId)` for the counts.
- **The projection holds current text; the event stream holds history.** An edit updates the row.
  Every version remains in JetStream, so showing an edit history later is a UI decision rather
  than a migration (agreed 2026-08-23). This is simpler than sos's `refActivityId` chaining and
  loses nothing we have decided we want.

**Events**, following the sos comment pattern:

- `NATHEJK.{year}.spejder.{memberId}.commented` — `{noteId, memberId, note, actor}`
- `NATHEJK.{year}.spejder.{memberId}.comment.updated` — `{noteId, memberId, note, actor}`

Note ids are minted server-side (as `sos.NewCommentID` does), so a client cannot collide with one
it has not seen. The entity in the subject is `spejder`, which means **no new live token** and no
client-side dependency warning — the existing `spejder` subscribers, including the shelter list,
invalidate correctly for free.

**Commands** on a `spejdernote` commander: `Comment` (returns the new id) and `UpdateComment`
(dirty-checks the text, so re-submitting an unchanged note publishes nothing). `UpdateComment`
must verify the note belongs to the member named in the request — without it, a client could
amend any note by id, which is the same check `sos.UpdateComment` makes.

**API endpoints** (year-scoped via `X-YearSlug`; **all need `@Summary` / `@Description` / `@Tags`
/ `@Router` OpenAPI annotations**, per `cmd/api/order.go`):

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/member/:memberId/notes` | The thread, oldest first |
| POST | `/api/member/:memberId/notes` | Add a note |
| PATCH | `/api/member/:memberId/notes/:noteId` | Correct a note |

Served on the member, beside the lifecycle routes. The thread could have been folded into
`GET /api/member/:memberId`, and is not: that endpoint feeds a modal opened one scout at a time,
while the thread is also wanted for the row summaries — and a separate resource is separately
cacheable and separately invalidated.

**Row summaries** need an aggregate, not the thread: `GET /api/shelter` gains `noteCount` and a
truncated `latestNote` per member, from one grouped query over `spejdernote` — the same shape as
the placering lookup, batched over the members already fetched. No query per row.

**Frontend.** `MemberNotes.vue`; `MemberDetailDialog.vue` extracted from `SosTeamCard.vue` and
made live; hosts wired in `HoensegaardView.vue`, `SosTeamCard.vue`, `PatruljeView.vue`.

**Dependencies & risks.**

- *Unsigned notes.* Accepted (§5), and worth being clear-eyed about the cost: "aftalt at mor
  henter kl. 06" without an author cannot be followed up — nobody knows who to ask. Login is
  expected before the race; if it slips, the trail is still useful for handover but weak as
  accountability. The `actorUserId` column is written regardless, so notes start being attributed
  the day accounts exist, with no change here.
- *Editable by anybody.* A consequence of the same gap. Acceptable for a crew of colleagues on one
  site; it should not survive login without being revisited.
- *Extraction touches the nødtelefon's most sensitive component.* `SosTeamCard.vue` is 956 lines
  and carries the collection and move flows. The extraction must be behaviour-preserving and is
  therefore its own task, ahead of any note work, rather than being done in passing.
- *The modal is not live today.* Making it live is in scope, and is a prerequisite for the trail
  being shared rather than per-browser.

## 9. Success Metrics

- Notes exist for most sheltered scouts by the end of race night. Target: a note on more than half
  of the scouts who spent the night, and on **every** scout collected by a guardian — the pickup
  arrangement is the note that matters most.
- No scout's guardian is rung twice about the same arrangement. Qualitative, from the crew.
- The paper list does not come back for notes (PRD 007's own closing metric, extended).
- Notes are read on the case card, not only written in the shelter: evidence that one trail in two
  places was the right shape rather than two audiences with one tool.

## 10. Rollout / Task Breakdown

Backend first, then the extraction, then the hosts — so the trail exists before anything tries to
show it, and the risky refactor lands on its own.

- [ ] Task: `spejdernote` table + `Commented`/`CommentUpdated` events + projection, wired into the
      `projections` slice (099)
- [ ] Task: `Comment` / `UpdateComment` commands, with the note-belongs-to-member check and the
      unchanged-text dirty-check (100)
- [ ] Task: the three note endpoints with OpenAPI annotations (101)
- [ ] Task: `noteCount` + truncated `latestNote` on the `GET /api/shelter` rows, batched (102)
- [ ] Task: extract `MemberDetailDialog.vue` from `SosTeamCard.vue`, behaviour-preserving, and
      make it live via `useLiveResource` (103)
- [ ] Task: `MemberNotes.vue` — thread, add form, edit-in-place, dirty-defer (104)
- [ ] Task: wire the dialog and the note summary column into `HoensegaardView.vue` (105)
- [ ] Task: wire the dialog into `SosTeamCard.vue` and `PatruljeView.vue` (106)

No feature flag: additive, and reachable only from screens that gain an affordance.

**Relation to PRD 007 task 097.** They do not collide and both stand. 097 summarises shelter
*operations* onto an associated open case, which is a machine-written line about a status change.
This is human prose about a scout, and it stays off the case timeline entirely (§4).

## 11. Open Questions

**Resolved 2026-08-23 (with the crew):**

- ~~Attribution before login?~~ An unsigned trail on race day is acceptable if login slips.
- ~~Prose or a structured pickup time?~~ Prose only; the timestamp field is dropped.
- ~~Editable or append-only?~~ Editable, mainly for typos, with the UI encouraging a new note
  instead. Every version is in the event stream either way, so showing an edit history remains a
  changeable decision.
- ~~Do notes appear on the SOS timeline?~~ No. The case card reads a scout's notes because it
  shows that scout; nothing is copied.

**Still open:**

- **Modal or expanded row, finally?** The plan hosts the thread in the extracted modal and gives
  the row a count-plus-snippet, on the grounds that the modal reaches all three screens with one
  implementation. If the crew tries it and wants inline editing in Hønsegården, dropping the same
  component into an expander is cheap — but somebody should decide after using it rather than now.
- **Does the patrol page want the notes too, or only the two race-night screens?** Listed in 106,
  and easy to drop if it clutters a page used mostly before the race.
- **Should a note be possible on a member who never started?** Allowed by the plan (any member of
  the year). The alternative would refuse notes for a scout who cancelled the week before, which
  is probably exactly when somebody wants to write down why.
- **Do we ever want an edit history on screen?** Not built; the events make it possible whenever
  somebody asks.
