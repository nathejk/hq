# 073 — Whole-team collection as a single command

**Status:** done
**Priority:** medium
**Created:** 2026-08-17
**Picked up by:** agent session
**Started:** 2026-08-17
**Completed:** 2026-08-17

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

- [x] `POST /api/sos/:id/team/:teamId/collect` registered and wired
- [x] Publishes one withdrawal request per remaining `racing` member (see log re:
      `correlationId`)
- [x] Publishes exactly one summarising `sos` event naming the affected members
- [x] Members not currently `racing` are skipped
- [x] Renders as a single timeline entry
- [x] `activeMemberCount` reaches zero afterwards
- [x] Fails loudly rather than partially reporting success
- [x] Exercised against the dev stack on a team with three racing members — used four
- [x] `go build ./... && go vet ./...` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006. Depends on tasks 071 and 072.
- 2026-08-17 — Picked up. `CollectTeam` on the member commander plus
  `collectTeamHandler`; the loop is on the server so a partial failure is returned as a
  failure rather than leaving a team split across two states.
- 2026-08-17 — **⚠️ Could not honour the shared `correlationId` this task and PRD 006 §8
  both called for: `shared-go/messages.Metadata` has no `CorrelationID` field** — only
  `CorrelationSequence uint64`. Rather than force a cross-repo change, checked what the
  requirement was actually *for*: §8 wanted it "so the timeline can render it as one
  entry", and the summarising `sos` event from task 071 now does that outright. The
  grouping is also recoverable from the summary, which lists every member id. So it is
  dropped as superseded, and PRD 006 §8 amended to say so — not silently skipped, because
  it reads like an implementation detail and was really a rendering requirement.
- 2026-08-17 — Members already out of the race are **skipped, not re-requested**. This is
  what makes "collect the whole team" different from "set everyone to waiting": a member
  already in a car has left the route, and publishing a withdrawal for them would add a
  false line to their history and walk them backwards through the lifecycle. Tested with a
  team holding one racing, one transit, one waiting and one sheltered member.
- 2026-08-17 — A publish failure mid-loop **returns** rather than continuing, and discards
  the changes so far so no summary claims an operation that did not finish. Whatever was
  published stays published — the log is the record — but the operator is told.
- 2026-08-17 — `TeamStrength` is 0 by definition here rather than computed: every racing
  member is being taken out, so none is left. That zero is also what makes the patrol
  discontinued, with no event of its own.
- 2026-08-17 — Handler does **one** roster lookup for the team rather than one per
  collected member, which the shared `memberName` helper would have done.
- 2026-08-17 — ✅ Verified on the dev stack against a real 4-member 2025 patrol
  (Buddingedrengene):
  - collect → 4 changes, each `racing→waiting` with `TeamStrength: 0`
  - **one** `team.collected` timeline entry, not four — confirmed by grouping
    `sos_activity` by type
  - that single entry names all four members ("Villum Bækgaard", "Asmus Stengaard", "Magne
    Rundstrøm", "Marius Sletved") and records `teamStrength: 0`
  - `activeMemberCount` 0 with `signupStatus STARTED` → **discontinued** by the corrected
    predicate
  - in-our-care total 4, all `waiting`
  - collect again → `collected: []`, no second entry
  - restored afterwards: all four back to `racing`, `activeMemberCount` 4, in-our-care 0,
    case soft-deleted
- 2026-08-17 — ✅ All criteria met, 3 new tests. Full `go build`, `go vet`, `gofmt -l`,
  `go test ./...` clean. Moving to done — this closes Phase B.
- 2026-08-17 (later, during task 076) — **⚠️ Defect found and fixed: the endpoint did not
  check that the team was associated with the case.** Noticed by accident — a verification
  script's association call failed, and the collect afterwards emptied the patrol anyway.
  So a stale or copy-pasted `teamId` would take a patrol out of the race from a case that
  has nothing to do with it, and the summary would land on a timeline whose team card does
  not even list them — invisible exactly where it matters.

  Added the check in `collectTeamHandler`: the case is loaded and the team must be among
  its associations, else `"patruljen er ikke tilknyttet sagen"`. This is the one member
  operation that needs it, and the reason is worth keeping: the other three act on a member
  the operator is looking at, whereas this one acts on a **set** derived from a team id
  alone. That is the difference between a mistake affecting one row and one emptying a
  patrol.

  Verified: collecting a non-associated team is now rejected, an associated one still
  works. My own acceptance criteria did not ask for this, which is the lesson — the URL
  asserted a relationship the handler never verified.
