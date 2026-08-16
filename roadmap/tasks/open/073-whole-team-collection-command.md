# 073 — Whole-team collection as a single command

**Status:** open
**Priority:** medium
**Created:** 2026-08-17
**Picked up by:**
**Started:**
**Completed:**

## Description

From **PRD 006** §6 and §8.

`POST /api/sos/:id/team/:teamId/collect` — every remaining `racing` member of the team goes
to `waiting` in **one** operator action.

**One command, not N calls.** It publishes a withdrawal request per remaining racing
member, sharing one `correlationId`, and then the one summarising `sos` event. Publishing
three independent requests from the frontend would risk a partial collection if one call
fails — the worst possible outcome, since the team would then be split across states with
nobody noticing. Operators are on the phone; three separate clicks invite two of them being
forgotten.

This is the general N+1 rule from task 071 applied to the case that motivated it: three
`spejder` events drive the projection, one `sos` event renders as a single timeline line
("hele patruljen hentes").

## Notes

- Members already `waiting` (or beyond) are skipped, not re-requested. Collecting a team
  where one member is already in a car must not touch that member.
- Atomic **from the operator's point of view** — one HTTP call, one summary. The underlying
  publishes are still individual events; if one fails mid-way the command should fail loudly
  rather than report success, and the log preserves what did get published.
- Resolving one member does not unwind actions taken for others: if a member resumes after
  the rest of the team has been collected, the team is *not* automatically restored. Those
  members are `waiting` and only they or a car can change that (PRD 006 §5).
- `activeMemberCount` falls to zero as a consequence, so the team becomes discontinued. No
  separate event for that (task 066).

## Acceptance Criteria

- [ ] `POST /api/sos/:id/team/:teamId/collect` registered and wired
- [ ] Publishes one withdrawal request per remaining `racing` member, one shared
      `correlationId`
- [ ] Publishes exactly one summarising `sos` event naming the affected members
- [ ] Members not currently `racing` are skipped
- [ ] Renders as a single timeline entry
- [ ] `activeMemberCount` reaches zero afterwards
- [ ] Fails loudly rather than partially reporting success
- [ ] Exercised against the dev stack on a team with three racing members
- [ ] `go build ./... && go vet ./...` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006. Depends on tasks 071 and 072.
