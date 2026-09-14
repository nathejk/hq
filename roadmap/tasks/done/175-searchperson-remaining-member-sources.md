# 175 — Add senior, personnel and crewmember sources to search_person

**Status:** done
**Priority:** high
**Created:** 2026-09-14
**Picked up by:** agent session 2026-09-14
**Started:** 2026-09-14
**Completed:** 2026-09-14

## Description

Depends on task 174. Extends the `searchperson` consumer to the three remaining
participant/staff populations. Mechanical once 174 established the shape, but the subject
literals differ and one of them is shared.

| Source | Subjects | kind |
|---|---|---|
| `senior` | `NATHEJK.*.senior.*.updated` | `senior` |
| `personnel` | `NATHEJK.*.gøgler.*.{signedup,updated}`, `NATHEJK.*.friend.*.{signedup,updated}` | `gøgler`, `friend` |
| `crewmember` | `NATHEJK.*.crewmember.*.{registered,updated}`, `NATHEJK.*.crew.*.signedup` | `crew` |

Notes:

- `NATHEJK.*.bandit.*.armNumber.assigned` is consumed by **both** `senior` and `personnel`
  today. It carries no name or phone, so this projection does not need it — do not subscribe
  just because the neighbours do.
- `crewmember` treats `NATHEJK.*.crew.*.signedup` as a crew registration; that is what makes
  a crew signup a crew member. Mirror it, or crew are unfindable until they are updated.
- `personnel` rows are keyed by `userId` and carry `userType`; keep `kind` as the event's
  entity token (`gøgler` / `friend`) rather than the table's name, so it lines up with the
  live entity tokens.
- `.deleted` handling is task 177.

Copy subject literals from the existing consumers. `senior`/`crewmember` are in shared-go;
`personnel` is `go/nathejk/table/personnel/consumer.go`.

## Acceptance Criteria

- [x] Senior rows produced from `senior.updated`
- [x] Gøgler and friend rows produced from their signedup/updated events, with distinct `kind`
- [x] Crew rows produced from both `crewmember.*` and `crew.*.signedup`
- [x] One test per source asserting a round trip from event to queryable row
- [x] No subscription to subjects that carry no name or phone
- [x] `go build ./...` and `go test ./...` pass

## Progress Log

- 2026-09-14 22:08 — Task created from PRD 014 §10.
- 2026-09-14 23:00 — Picked up. Read all three source consumers first. Plan: one handler per
  event shape, `kind` taken from subject part 2 so gøgler and friend stay distinct without
  duplicated code.
- 2026-09-14 23:10 — Four handlers added: `handleSeniorUpdated`, `handleSignedUp` (shared by
  gøgler, friend and crew, since the payload shape is identical and only the kind differs),
  `handlePersonnelUpdated`, `handleCrewMember` (registered and updated fold identically here —
  the distinction matters to the crewmember projection, which has more columns, not to an
  index of names and numbers).
- 2026-09-14 23:14 — Decision: `kind` is read from subject part 2 rather than from the body.
  The personnel payloads carry no notion of which population the person belongs to, and the
  subject is what the broker routed on — so an update also cannot move somebody between kinds.
- 2026-09-14 23:18 — **Finding: there is no `bandit` population.** The task description and
  PRD 014 both listed `bandit` as a kind. It is not one: `bandit.*.armNumber.assigned` is the
  only bandit-entity event, it carries neither name nor phone, and senior/personnel consume it
  only to stamp an arm number on an existing row. Banditter arrive as seniors. Indexing it
  would have produced empty rows and advertised a live dependency that never fires. Corrected
  PRD 014 (§6 kind list, §8 schema comment and token list) and task 178. Pinned by
  `TestBanditIsNotAPopulation`, which also asserts nothing subscribes to a bandit subject.
- 2026-09-14 23:24 — **Finding: crew and personnel rows have no `teamId`.** Their events
  carry no team, and a crew member's section arrives on `crewmember.*.section.assigned`, which
  is identity-free and therefore deliberately not subscribed to. Left the column empty rather
  than subscribing to an event that carries nothing this table indexes; their context must come
  from the read-time join, which is task 179's job. Noted in PRD 014 §8 and added a criterion
  to task 179 — without it every gøgler and crew result is a bare name.
- 2026-09-14 23:28 — Kind constants use ASCII identifiers with Danish values (`KindGoegler`
  = `"gøgler"`), matching the codebase: no exported Go identifier here carries an ø.
- 2026-09-14 23:30 — ✅ All criteria met. 28 tests/subtests pass, `gofmt`, `go vet`,
  `go build ./...` and the full `go test ./...` all clean.
- 2026-09-14 23:31 — Completed. Five of the six sources are indexed; only the contact persons
  remain, which is task 176 and the part with no prior art.
