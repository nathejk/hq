# 176 — Add contact-person sources to search_person

**Status:** open
**Priority:** high
**Created:** 2026-09-14
**Picked up by:**
**Started:**
**Completed:**

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
- **`signup` subscribes with a wildcard in the entity position** (`NATHEJK:*.*.*.signedup`),
  which is what makes `live.EntitySet.Exhaustive` false. Subscribe the same way and filter on
  `teamType` **in the handler** — do not try to enumerate entity types in the subject.
- `signup` holds both `phone` and `phonePending`. An unverified contact still needs finding,
  so index both. Prefer the verified one for display and record which matched.
- A patrulje contact person and a klan contact person can be the same human; two rows is the
  correct answer, not a bug (PRD 014 §5, "one number, several people").

## Acceptance Criteria

- [ ] `patruljekontakt` rows from patrulje signedup/updated, keyed by teamId
- [ ] `klankontakt` rows from signup, filtered by teamType in the handler
- [ ] Both `phone` and `phonePending` are findable, and the row records which matched
- [ ] Test: a klan contact who appears only in `signup` is findable by phone
- [ ] Test: the `signedup` subscription actually matches a real subject (guards the `Match` trap)
- [ ] `go build ./...` and `go test ./...` pass

## Progress Log

- 2026-09-14 22:08 — Task created from PRD 014 §10.
