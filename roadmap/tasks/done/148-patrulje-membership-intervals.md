# 148 — patrulje membership intervals, current and former

**Status:** done
**Priority:** medium
**Created:** 2026-09-03
**Picked up by:** agent session
**Started:** 2026-09-03
**Completed:** 2026-09-03

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

- [x] A query returning, for a `teamId`: `{ memberId, name?, from, to? }` per membership interval
- [x] Includes members whose `spejder` row no longer exists
- [x] A member moved between two teams yields one interval per team, correctly bounded
- [x] Current members have an open-ended interval
- [x] Nothing read from `spejder` that would exclude deleted members
- [x] `go build ./...` clean

## Progress Log

- 2026-09-03 — Task created from PRD 011 §10.
- 2026-09-03 — Picked up. Plan: `TeamMemberships` in the `spejderstatus` package — it owns the log, so the query belongs with it rather than in the telemetry package.
- 2026-09-03 — The query reads **every** log row of every member who has ever touched the team, not only the rows naming it. Rows naming *other* teams are what close an interval, so filtering them out in SQL would leave every membership looking open-ended and a moved scout's track would never be clipped. Ordered by `(id, seq)`, not `createdAt` — events inside one operation share a timestamp to the second, and the sequence is the order the platform actually applied them (same reasoning as `GetHistory`).
- 2026-09-03 — Names come from a **separate** query against `spejder`, not a LEFT JOIN. The asymmetry is the point and is worth seeing in the code: the log is the source of truth for *who was here*, `spejder` is a best-effort lookup for what to call them, and its rows disappear.
- 2026-09-03 — Extracted the interval walk into `intervalsFor` so the decisions could be tested without a database, and the query calls the same function — a lifted copy would stop meaning anything the moment the two drifted. Three cases I would otherwise have got wrong, now pinned:
  1. **An event carrying no team must not end a membership.** `consumer.teamID` returns `""` for events that do not name one, and reading that as a departure would cut a member's track short at, say, a withdrawal request — losing exactly the positions somebody is looking for.
  2. **A returning member yields two intervals**, not one spanning their absence, or the other patrol's movement would appear on this patrol's map.
  3. **Repeated events while already on the team do not restart the interval.**
- 2026-09-03 — Members whose patrol never started have no log rows, so they are picked up from `spejderstatus` and given an **open interval with a zero `From`**. There is no event to date the start, and inventing one from the signup time would be a guess presented as a fact; zero means "do not clip", which is honest — every position held for them was recorded while they belonged to this team.
- 2026-09-03 — ⚠️ Adding one method to `spejderstatus.Queries` **broke three unrelated test stubs** (`shelter/commands_test.go`, `spejderstatus/commands_test.go`, `cmd/api/shelter_test.go`). That is the standing cost of this repo's one-fat-interface-per-table convention; I added the method to each stub with a note rather than working around it with a narrow consumer-side interface, since `data.Models` types the field as `spejderstatus.Queries` and a type assertion at the call site would be worse. Worth knowing that any future addition here pays the same tax.
- 2026-09-03 — ✅ All criteria. 6 interval sub-tests + the full repo suite pass; `go vet ./...` clean. Completed.
