# 174 — search_person table and projection skeleton (spejder only)

**Status:** done
**Priority:** high
**Created:** 2026-09-14
**Picked up by:** agent session 2026-09-14
**Started:** 2026-09-14
**Completed:** 2026-09-14

## Description

First task of PRD 014. Establishes the `search_person` read projection with a single
source — `spejder` — so the table shape, phone normalization and test harness are proven
before the other five sources are added mechanically (task 175).

New package `go/nathejk/table/searchperson/` following the established shape (`table.go`,
`consumer.go`, `query.go`, `table.sql`, `filter.go`).

Schema per PRD 014 §8. `kind` is part of the primary key because the id spaces do not
merge: `memberId` for scouts/seniors, `userId` for personnel/crew, `teamId` for contact
persons who have no id of their own.

Two constraints that are easy to get wrong:

- **The consumer must read no other projection's table.** Every field comes from the event
  payload. Reading `spejder` to build the row would make this projection depend on the
  order in which two consumers happen to see the same event, which is not guaranteed.
- **Normalization uses `types.PhoneNumber.Normalize()`** from shared-go — the same function
  the SMS gateway uses, so a number that can be texted is a number that can be found.

Subjects for this task (copy the literals from `shared-go/tables/spejder/consumer.go`):

```
NATHEJK.*.spejder.*.updated
NATHEJK.*.spejder.*.reassigned
```

`NATHEJK:*.patrulje.*.started` also carries member phone numbers and is how a phone first
reaches the roster for some teams — worth handling, but it belongs with the other multi-member
events; note it and move on if it complicates this task.

**Subject separator.** `subject.FromStr` replaces the first `:` with `.`, so subscribing with
`NATHEJK:` or `NATHEJK.` is the same subscription. But `Subject.Match` escapes `.` without
touching `:`, so a `Match("NATHEJK:…")` pattern can never match — spell every `Match` pattern
with dots. `Match` is case-insensitive.

`spejder.deleted` retention is task 177; ignore the subject here rather than hard-deleting.

Do **not** wire it into the `projections` slice yet — that is task 178.

`CREATE TABLE IF NOT EXISTS` never alters an existing table, so add the `ensureColumn`
pattern already in `patrulje/table.go` from the start rather than after being bitten.

## Acceptance Criteria

- [x] `go/nathejk/table/searchperson/` created with table.sql matching PRD 014 §8
- [x] Table created on startup, with `ensureColumn`-style guards for later additions
- [x] `spejder.updated` and `.reassigned` produce/refresh a row with kind `spejder`
- [x] `phoneNormalized` and `phoneParentNormalized` populated via `types.PhoneNumber.Normalize()`
      — **plus a national-form step; see the log, Normalize() alone was not enough**
- [x] Consumer reads no other table
- [x] Unit tests cover upsert, re-upsert (no duplicate rows), and normalization of
      `+45 12 34 56 78`, `12 34 56 78`, `12345678` to the same digits
- [x] `go build ./...` and `go test ./...` pass

## Progress Log

- 2026-09-14 22:08 — Task created from PRD 014 §10.
- 2026-09-14 22:20 — Corrected the subject-separator note in the description and in PRD 014
  §8 before starting: `subject.FromStr` normalizes the first `:` to `.`, so the inconsistency
  between consumers is cosmetic. The real trap is `Subject.Match`, which escapes `.` but not
  `:`, so a `Match("NATHEJK:…")` pattern silently matches nothing.
- 2026-09-14 22:22 — Picked up. Plan: model the package on `table/photo` (newest style —
  exported `Table`, `New` returns an error, `cqrs` interfaces rather than `stream`), write
  table.sql, consumer for `spejder.updated`/`.reassigned`, a querier stub, and tests on
  `cqrstest.Writer`.
- 2026-09-14 22:35 — Package written: table.sql, table.go, sql.go, consumer.go. Decided
  `New(w, r)` takes **no Publisher** — search is read-only, there is no command side, and a
  nil publisher field would just invite one.
- 2026-09-14 22:38 — Decision: `spejder.updated` is decoded **twice**, as
  `NathejkScoutUpdated` and again as the legacy `NathejkMemberAdded`, purely to recover
  `teamId`. The scout-updated shape does not carry a team (the roster projection
  deliberately leaves teamId alone on that event so reassignment is the only way a team
  changes), but the published payload does, and `shared-go/tables/spejder` reads it the same
  way. Without it a scout would have no patrol until reassigned, and a result row with no
  patrol is barely a search result.
- 2026-09-14 22:40 — Decision: `teamId` is the one column protected in the upsert
  (`teamId=IF(VALUES(teamId)='', teamId, VALUES(teamId))`). Everything else is overwritten
  unconditionally, because for those the event *is* the source of truth — a scout who
  removes their phone number should stop being findable by it.
- 2026-09-14 22:44 — **Blocker, and the main finding of this task.**
  `types.PhoneNumber.Normalize()` does not do what PRD 014 assumed. It keeps *every* digit,
  so `+45 12 34 56 78` → `4512345678` while `12 34 56 78` → `12345678`: the two forms that
  most need to match still do not. shared-go is no help — its `IsValid()` requires exactly 8
  digits and therefore calls the `+45` form invalid, and `InternationalNumber()` prepends an
  empty country code to it. The test caught this immediately, which is the argument for
  having written it before wiring anything.
- 2026-09-14 22:48 — Blocker resolved: `normalizePhone` now delegates to shared-go for step
  one (so we agree with the SMS gateway about what a digit is) and adds a national-form step
  — strip a leading `00`, then strip a leading `45` **only from a 10-digit result**. The
  length guard matters: `45` is a real Danish prefix, so `45123456` is somebody's actual
  number, and stripping it unconditionally would have made every 45-prefixed subscriber in
  the event silently unfindable. Pinned by `TestNormalizationKeeps45PrefixedNationalNumbers`.
- 2026-09-14 22:50 — Corrected PRD 014 §8, which claimed shared-go's Normalize was
  sufficient, and added a warning to task 180: the query side must use *this* package's
  `normalizePhone`, or the halves disagree and it looks like "not in the system".
- 2026-09-14 22:52 — Deferred `query.go` and `filter.go` to task 180 rather than committing
  empty files: nothing reads the table yet, and the query shape is settled by tasks 179/180.
  Noted so the next task does not go looking for them.
- 2026-09-14 22:55 — ✅ All criteria met. 15 tests, `go build ./...`, `go vet` and the full
  `go test ./...` all pass. Not yet wired into the `projections` slice — that is task 178, so
  nothing in the running API touches this code yet.
- 2026-09-14 22:56 — Completed. `search_person` exists, spejdere are indexed by name and by
  both their own and their guardian's number, and the format problem that motivated the
  projection is actually solved rather than assumed.
