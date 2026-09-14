# 179 — Join current status into search results

**Status:** open
**Priority:** high
**Created:** 2026-09-14
**Picked up by:**
**Started:**
**Completed:**

## Description

Depends on task 178. Implements PRD 014's requirement that **every result states the
person's current standing**.

Status is a **read-time join, not a projected column**:

- a spejder's lifecycle status from `spejderstatus` (`registered`, `seated`, `racing`,
  `finished`, `waiting`, `transit`, `sheltered`, `reunited`, `released`)
- otherwise the team's `signupStatus` from `patrulje` / `klan` / `personnel`
- plus the `deleted` flag from task 177

Same reasoning as the team-name join, and it pays twice: the consumer needs none of the
status subjects, so its replay and signal surface stay narrow, and PRD 006's state machine is
not re-implemented in a second place where it could disagree with the first.

Team and section **names** are joined here too (`patrulje`, `klan`, `section`) — the
projection stores only `teamId`.

Details:

- **A spejder with no `spejderstatus` row is ordinary, not an error.** Statuses begin at
  signup-time `registered`/`seated`, and older years predate the table entirely. LEFT JOIN,
  and render absence as unknown — no `COALESCE` to a status that would be a lie.
- **`deleted` and a lifecycle status are different axes.** A row can be `deleted` (never
  started) or `released` (went home during the night); the response must let the UI say
  which, so do not collapse them into one string server-side.

## Acceptance Criteria

- [ ] Results carry lifecycle status for spejdere, signup status otherwise, and the deleted flag
- [ ] Team/section name resolved by join, not denormalized into `search_person`
- [ ] A spejder with no status row returns unknown rather than a fabricated status
- [ ] `deleted` and lifecycle status are separate fields in the response
- [ ] Tests cover: racing spejder, released spejder, deleted spejder, spejder with no status row
- [ ] `go build ./...` and `go test ./...` pass

## Progress Log

- 2026-09-14 22:08 — Task created from PRD 014 §10.
