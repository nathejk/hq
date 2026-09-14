# 174 — search_person table and projection skeleton (spejder only)

**Status:** open
**Priority:** high
**Created:** 2026-09-14
**Picked up by:**
**Started:**
**Completed:**

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

Subjects for this task (copy the literals from `shared-go/tables/spejder/consumer.go`
rather than retyping — the separator is not uniform across the codebase):

```
NATHEJK.*.spejder.*.updated
NATHEJK.*.spejder.*.reassigned
```

`spejder.deleted` retention is task 177; ignore the subject here rather than hard-deleting.

Do **not** wire it into the `projections` slice yet — that is task 178.

`CREATE TABLE IF NOT EXISTS` never alters an existing table, so add the `ensureColumn`
pattern already in `patrulje/table.go` from the start rather than after being bitten.

## Acceptance Criteria

- [ ] `go/nathejk/table/searchperson/` created with table.sql matching PRD 014 §8
- [ ] Table created on startup, with `ensureColumn`-style guards for later additions
- [ ] `spejder.updated` and `.reassigned` produce/refresh a row with kind `spejder`
- [ ] `phoneNormalized` and `phoneParentNormalized` populated via `types.PhoneNumber.Normalize()`
- [ ] Consumer reads no other table
- [ ] Unit tests cover upsert, re-upsert (no duplicate rows), and normalization of
      `+45 12 34 56 78`, `12 34 56 78`, `12345678` to the same digits
- [ ] `go build ./...` and `go test ./...` pass

## Progress Log

- 2026-09-14 22:08 — Task created from PRD 014 §10.
