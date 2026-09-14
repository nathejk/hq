# 175 — Add senior, personnel and crewmember sources to search_person

**Status:** open
**Priority:** high
**Created:** 2026-09-14
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] Senior rows produced from `senior.updated`
- [ ] Gøgler and friend rows produced from their signedup/updated events, with distinct `kind`
- [ ] Crew rows produced from both `crewmember.*` and `crew.*.signedup`
- [ ] One test per source asserting a round trip from event to queryable row
- [ ] No subscription to subjects that carry no name or phone
- [ ] `go build ./...` and `go test ./...` pass

## Progress Log

- 2026-09-14 22:08 — Task created from PRD 014 §10.
