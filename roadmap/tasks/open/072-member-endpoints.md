# 072 — Member endpoints: waiting, resume, override, move

**Status:** open
**Priority:** high
**Created:** 2026-08-17
**Picked up by:**
**Started:**
**Completed:**

## Description

From **PRD 006** §6 and §8. The four operator actions this interface owns — everything up
to and including `waiting`, the **self-carrying** boundary. From the car door onwards the
transitions belong to interfaces that do not exist yet.

| Method | Path | Purpose |
|--------|------|---------|
| PUT | `/api/member/:memberId/waiting` | Member wants to leave the race (→ `waiting`). `sosId` **required** |
| PUT | `/api/member/:memberId/racing` | Member carries on under their own steam (→ `racing`). Rejected unless currently `waiting` |
| PUT | `/api/member/:memberId/status` | Override to any valid status **except `finished`**. Optional `sosId` |
| PUT | `/api/member/:memberId/team` | Move to another team (`currentTeamId`). Optional `sosId` |

Each publishes one `spejder` event per affected member, then one summarising `sos` event
when a `sosId` is present (task 071).

**The self-carrying boundary is enforced on the write side.** The resume is valid only
while the member is `waiting`; the command must dirty-check the current `spejderstatus`
row and reject otherwise, with a message the operator can act on ("allerede hentet")
rather than a generic conflict. Hiding the button is not sufficient — the operator's screen
may be a moment stale, which is exactly when a car is accepting the member. If the
acceptance and the cancellation race, **the acceptance wins**: it reflects a member
physically sitting in a car, and the log preserves both attempts in order.

**The 3-member requirement is not enforced here.** No command may reject a withdrawal
because it would put a team below three, and no consumer may auto-collect or auto-move in
response. The member is leaving regardless; refusing to record it would only make the data
wrong. The write side reports strength and records what happened — it never decides.

## Notes

- **Move** updates `currentTeamId` and leaves `initialTeamId` untouched. A valid target is
  any patrol in the same year still racing (started, `activeMemberCount > 0`) that is not
  the team being left — nothing further, because the destination is agreed by crew in the
  field and the operator is *recording* it, not choosing it (PRD 006 §11 Decisions).
- **Moving is per member.** Two survivors may end up in two different patrols. Do not model
  the command as "move this group to that team".
- **The override excludes `finished`.** `CanFinish()` is true only for `racing`, and a
  member who took a lift must never end up finished. There is no finish producer anywhere
  in hq today, so this is a guard for a flow that arrives later.
- The override is a **separate endpoint** from the withdrawal request rather than one
  parameterised status setter, because they are different acts: one is the normal workflow
  this interface owns, the other is an admission that another interface's handover went
  unrecorded. Splitting them keeps it auditable and makes "how often are we correcting by
  hand?" answerable (§9 tracks it).
- Member actions live on the member, not nested under `/api/sos`: a member's status is a
  fact about the member. Breach handling — what little of it exists — is a fact about the
  case.
- Note `/api/member/...` uses a noun no existing route uses, and it collides with the
  seniors question in PRD 006 §11. Confirm alongside task 063's subject decision.
- Commands dirty-check before publishing, so a no-op write publishes nothing and emits no
  live signal — the house pattern.
- Actor comes from task 070.

## Acceptance Criteria

- [ ] Four routes registered and wired to commands in the `spejderstatus` package
- [ ] `waiting` rejects a missing `sosId`
- [ ] Resume rejected unless the member is currently `waiting`, with an actionable message
- [ ] Override rejects `finished` and any status failing `Valid()`
- [ ] Move rejects a target that is not a racing patrol in the same year, and rejects the
      member's current team
- [ ] Each action publishes per-member `spejder` events plus one summarising `sos` event
      when `sosId` is given
- [ ] No-op writes publish nothing
- [ ] Every endpoint exercised against the running dev stack, including the rejections
- [ ] No `nathejk.dk/...` import in the package
- [ ] `go build ./... && go vet ./...` and `gofmt -l` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006. Depends on tasks 063, 065, 070, 071.
