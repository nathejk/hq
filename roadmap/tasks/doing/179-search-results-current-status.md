# 179 — Join current status into search results

**Status:** doing
**Priority:** high
**Created:** 2026-09-14
**Picked up by:** agent session 2026-09-14
**Started:** 2026-09-14
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

**Crew and personnel rows have no `teamId` at all** (task 175: their events carry no team,
and a crew member's section arrives on an identity-free `section.assigned` event that
`searchperson` does not subscribe to). Their context has to come from a join to `personnel`
and `crewmember` on the person's id — `personnel.klan` / `groupName`, and
`crewmember.sectionSlug` → `section`. Without it, every gøgler and crew result row is a bare
name, which is barely a search result.

Details:

- **A spejder with no `spejderstatus` row is ordinary, not an error.** Statuses begin at
  signup-time `registered`/`seated`, and older years predate the table entirely. LEFT JOIN,
  and render absence as unknown — no `COALESCE` to a status that would be a lie.
- **`deleted` and a lifecycle status are different axes.** A row can be `deleted` (never
  started) or `released` (went home during the night); the response must let the UI say
  which, so do not collapse them into one string server-side.

## Acceptance Criteria

- [ ] Results carry lifecycle status for spejdere, signup status otherwise, and the deleted flag
- [ ] Crew and personnel rows get their context by joining `crewmember`/`personnel` on id
- [ ] Team/section name resolved by join, not denormalized into `search_person`
- [ ] A spejder with no status row returns unknown rather than a fabricated status
- [ ] `deleted` and lifecycle status are separate fields in the response
- [ ] Tests cover: racing spejder, released spejder, deleted spejder, spejder with no status row
- [ ] `go build ./...` and `go test ./...` pass

## Progress Log

- 2026-09-14 22:08 — Task created from PRD 014 §10.
- 2026-09-15 01:30 — Picked up. This task creates `query.go` (deferred from 174), so it also
  settles the result shape task 180 will filter and task 181 will serialise.
