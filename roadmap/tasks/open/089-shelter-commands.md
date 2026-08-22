# 089 — Shelter commands: accept, place, hand over

**Status:** open
**Priority:** high
**Created:** 2026-08-23
**Picked up by:**
**Started:**
**Completed:**

## Description

The write side of PRD 007. Three commands on the `spejderstatus` commander
(`go/nathejk/table/spejderstatus/commands.go`), mirroring `RequestWithdrawal`'s shape:
dirty-check the current row, publish, return `*Change`.

- `AcceptIntoShelter(ctx, actor, year, id, placement string) (*Change, error)` — publishes
  `ShelterAccepted`. **Valid from `waiting`, `transit` and `racing`.** The last two need
  saying out loud: the shelter is the receiving party and its word is the better evidence,
  so a scout who arrives with no pickup recorded must still be acceptable (PRD 007 §5). A
  strict state machine here would refuse precisely the case the screen exists to handle —
  the same argument `OverrideStatus`'s comment already makes.
- `SetPlacement(ctx, actor, year, id, placement string) (*Change, error)` — publishes
  `ShelterPlaced`. Valid only while `sheltered`: placing somebody who is not in the shelter
  is a bug in the caller, not a correction. Dirty-checks the placering too, via the
  `shelter` table, so re-submitting the same tent publishes nothing.
- `CompleteHandover(ctx, actor, year, id, to types.MemberStatus) (*Change, error)` —
  publishes `HandoverCompleted`. `to` must be `released` or `reunited`; reject anything
  else, and `finished` especially — `CanFinish()` is true only for `racing` and no
  correction may confer a finish.

A no-op (already in the target status) publishes nothing and returns no error, exactly as
the existing commands do. That is what makes two crew members on two laptops pressing
**Modtaget** harmless, and it is why the screen shows no error for the second press.

**Relax the case requirement.** `memberStatusOperation` in `cmd/api/member.go` currently
requires a `sosId` — "nothing changes a member's status without a case explaining why".
That rule belongs to the nødtelefon: the shelter's acceptances are case-free by design
(`messages.go`, "No sosId, deliberately") because the shelter may receive a scout nobody
opened a case about. Make the optionality **explicit in the helper's signature** rather than
passing an empty id through a code path that assumes one — a silent empty string here is how
the next reader concludes cases are optional everywhere.

`spejderstatus` may not import `nathejk.dk/...` (task 083).

## Acceptance Criteria

- [ ] Three commands implemented with dirty-checks
- [ ] `AcceptIntoShelter` accepted from `waiting`, `transit` and `racing`; rejected from
      `registered`/`seated` (they never started)
- [ ] `CompleteHandover` rejects any `to` but `released`/`reunited`, with a clear error
- [ ] No-ops publish nothing and return no error
- [ ] `memberStatusOperation` expresses "case optional" in its signature; the nødtelefon's
      existing handlers still require one
- [ ] Command tests including the racing→sheltered jump and the finished rejection
- [ ] `lift_test.go` green

## Progress Log

- 2026-08-23 09:30 — Task created from PRD 007 §10.
