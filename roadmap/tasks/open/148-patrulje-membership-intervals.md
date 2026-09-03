# 148 — patrulje membership intervals, current and former

**Status:** open
**Priority:** medium
**Created:** 2026-09-03
**Picked up by:**
**Started:**
**Completed:**

## Description

PRD 011 §6, §8. Prerequisite for task 149.

For a patrol map to show "everyone who has been on this team", hq needs, per team: every
`memberId` that has belonged to it, and **the interval during which they did**.

**This cannot come from the `spejder` table.** `NATHEJK.*.spejder.*.deleted` hard-deletes the
row (`DELETE FROM spejder WHERE memberId=…` in shared-go), so a withdrawn or removed scout
leaves no row at all. The history lives in:

- `spejderstatus(year, id, initialTeamId, currentTeamId, status, updatedAt)` — the "was here,
  now there" fact
- `spejderstatuslog(seq, id, year, teamId, status, event, actorUserId, createdAt)` — the
  append-only per-event history, including `team.moved`, `withdrawal.requested`,
  `handover.completed`, `shelter.placed`

Derive intervals by walking `spejderstatuslog` per member: each row's `teamId` from
`createdAt` until the next row that changes team, open-ended for the current team. A member
moved away mid-race yields a closed interval; a current member an open one.

Return a name where one survives (`spejder`), and degrade to "tidligere medlem" where it does
not — a track with no name is still evidence and must not be dropped.

Query only; no new table, no new projection.

## Acceptance Criteria

- [ ] A query returning, for a `teamId`: `{ memberId, name?, from, to? }` per membership interval
- [ ] Includes members whose `spejder` row no longer exists
- [ ] A member moved between two teams yields one interval per team, correctly bounded
- [ ] Current members have an open-ended interval
- [ ] Nothing read from `spejder` that would exclude deleted members
- [ ] `go build ./...` clean

## Progress Log

- 2026-09-03 — Task created from PRD 011 §10.
