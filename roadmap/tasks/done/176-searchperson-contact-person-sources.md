# 176 — Add contact-person sources to search_person

**Status:** done
**Priority:** high
**Created:** 2026-09-14
**Picked up by:** agent session 2026-09-14
**Started:** 2026-09-14
**Completed:** 2026-09-14

## Description

Depends on task 175. This is the part of PRD 014 with the most value and the least prior
art: **a contact person is not a row anywhere today.**

- For a **patrulje** they are three columns on the team (`contactName`, `contactPhone`,
  `contactEmail`), so they are projected from `NATHEJK:*.patrulje.*.{signedup,updated}` with
  kind `patruljekontakt`, keyed by `teamId`.
- For a **klan** there are no contact columns at all — `klan` has none — so the only source
  is `signup` (`name`, `phone`, `phonePending`), from `NATHEJK:*.*.*.signedup`, with kind
  `klankontakt`.

Traps:

- **Spell `Match` patterns with dots.** `subject.FromStr` normalizes the first `:` to `.`, so
  it makes no difference which form `Consumes()` uses — but `Subject.Match` escapes `.` and
  leaves `:` alone, so `Match("NATHEJK:*.*.*.signedup")` matches nothing at all. The subject
  holds a dot by the time the handler sees it.
- ~~**`signup` subscribes with a wildcard in the entity position**~~ — **superseded, see the
  log.** Subscribing entity-by-entity keeps `live.EntitySet.Exhaustive` true, which is worth
  more than mirroring signup's wildcard.
- ~~`signup` holds both `phone` and `phonePending`, so index both.~~ — **wrong, see the log.**
  Verification does `SET phone = phonePending`: one number, two states.
- A patrulje contact person and a klan contact person can be the same human; two rows is the
  correct answer, not a bug (PRD 014 §5, "one number, several people").

## Acceptance Criteria

- [x] `patruljekontakt` rows from patrulje signedup/updated, keyed by teamId
- [x] `klankontakt` rows from klan signedup **and klan updated** — subscribed entity by entity
      rather than via signup's wildcard, and so needing no teamType filter (see log)
- [x] ~~Both `phone` and `phonePending` are findable~~ — they are one number in two states, so
      one indexed pair covers both, verified or not (see log)
- [x] Test: a klan contact who appears only in `signup` is findable by phone
- [x] Test: the `signedup` subscription actually matches a real subject (guards the `Match` trap)
- [x] `go build ./...` and `go test ./...` pass

## Progress Log

- 2026-09-14 22:08 — Task created from PRD 014 §10.
- 2026-09-14 23:40 — Picked up. Read the patrulje, klan and signup consumers before writing
  anything; three of the assumptions in this description turned out to be wrong, logged below.
- 2026-09-14 23:48 — **Correction 1: do not mirror signup's wildcard subscription.** This task
  said to subscribe `NATHEJK:*.*.*.signedup` and filter on `teamType` in the handler. Doing so
  would put a wildcard in the *entity* position, which is precisely what makes
  `live.EntitySet.Exhaustive` false — and that flag is what lets the SPA warn about a
  dependency nothing can satisfy. We already know which team types have a contact person, so
  there is nothing to discover at runtime. Subscribed to `patrulje`/`klan` explicitly instead,
  and pinned it with `TestNoWildcardInTheEntityPosition`. The advertised token set stays
  complete.
- 2026-09-14 23:52 — **Correction 2: `phone` and `phonePending` are not two numbers.** This
  task said to index both. They are one number in two *states*: signup's verification handler
  does `UPDATE signup SET phone = phonePending`. So a single indexed pair finds the contact
  whether or not they ever verified, which is the behaviour wanted, and a second pair of
  columns would have been dead weight with a misleading name.
- 2026-09-14 23:58 — **Finding: `klan.updated` carries a contact person that nothing projects.**
  The klan table has no contact columns, and signup only keeps what the *signup* event carried,
  so a corrected klan contact currently exists nowhere in the read model. Subscribed to it, so
  after this the corrected version exists here. This is search holding a fact no other table
  does — worth knowing when task 179 wires the joins.
- 2026-09-15 00:04 — Decision: contact fields are read from the **prefixed** members of
  `NathejkTeamUpdated` (`ContactName`, not `Name`). The unprefixed ones describe the team, and
  reading them would have filed "Ørnene" as a human being. Covered by the `patrulje update`
  case.
- 2026-09-15 00:08 — Decision: a team update carrying no contact fields at all is **ignored**
  rather than written. It is an event about something else on the team (a liga, a group name),
  and upserting it would blank a perfectly good phone number — the operator would then find
  nobody. Same guard on signup, where a row with no name and no number could be found by
  neither and would only pad the table. Two tests cover this.
- 2026-09-15 00:10 — Two existing tests had to change, both correctly:
  `TestConsumedSubjectsReachAHandler`'s probe body needed contact fields (otherwise it would
  have passed by matching a handler that then declined to write), and
  `TestUnhandledSubjectIsNotAnError` had been using `klan.signedup`, which this task makes a
  handled subject — switched it to a `qr` scan, which carries no person at all.
- 2026-09-15 00:12 — Noted for task 179: patrulje's own projection writes its `year` from
  `msg.Time().Year()` while everything here uses the subject's year token. They agree in
  practice (the publisher derives both from the same clock, and a replayed event keeps its
  original time), but the join should be on `teamId` alone, which is unique, rather than on
  `teamId` *and* `year`.
- 2026-09-15 00:14 — ✅ All criteria met. 37 tests/subtests pass; `gofmt`, `go vet`,
  `go build ./...` and the full `go test ./...` clean.
- 2026-09-15 00:15 — Completed. All six sources are now indexed, including the two populations
  that were previously unfindable by any means.
