# 089 — Shelter commands: accept, place, hand over

**Status:** done
**Priority:** high
**Created:** 2026-08-23
**Picked up by:** agent session (Zed)
**Started:** 2026-08-23
**Completed:** 2026-08-23

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

- [x] Three commands implemented with dirty-checks
- [x] `AcceptIntoShelter` accepted from `waiting`, `transit` and `racing`; rejected from
      `registered`/`seated` (they never started)
- [x] `CompleteHandover` rejects any `to` but `released`/`reunited`, with a clear error
- [x] No-ops publish nothing and return no error
- [x] `memberStatusOperation` expresses "case optional" in its signature; the nødtelefon's
      existing handlers still require one
- [x] Command tests including the racing→sheltered jump and the finished rejection
- [x] `lift_test.go` green

## Progress Log

- 2026-08-23 09:30 — Task created from PRD 007 §10.
- 2026-08-23 15:00 — Picked up. `SetPlacement` needs to dirty-check the *placering*, which lives
  in hq's `shelter` table — a package `spejderstatus` may not import (it lifts to shared-go).
  Resolving that first; see the log below.
- 2026-08-23 15:10 — **Deviation from this task's plan, and the reason.** The task said all
  three commands go on the spejderstatus commander. `SetPlacement` cannot: it dirty-checks the
  current *placering*, which lives in the `shelter` table, and spejderstatus may not import hq
  packages (it lifts to shared-go verbatim) — nor could it import this one, since `shelter`
  already imports *it*, so the dependency would be a cycle. Split instead: the two status
  transitions stay in spejderstatus, `SetPlacement` moves to a new commander in `shelter`. That
  is also the better rule rather than a workaround: **a command belongs with the read model it
  dirty-checks against.** The handler composes the two, exactly as it already composes member
  events with the sos summary.
- 2026-08-23 15:20 — `AcceptIntoShelter` accepts every *started* status, including `reunited`
  and `released` (a scout handed back and then brought in again — rare, and refusing it would
  mean the shelter cannot record a child standing in front of them). Only `registered`/`seated`
  are refused: those members are at home, so an acceptance is a mistyped identity, and
  honouring it would invent a child in our care who is not on site.
- 2026-08-23 15:25 — Decision: an already-sheltered member is a no-op in `AcceptIntoShelter`
  **even when a placering is supplied**, and the handler is documented to use `SetPlacement`
  for that case. Publishing a second acceptance to carry a tent would claim custody was taken
  twice, which is the fiction these events exist to prevent.
- 2026-08-23 15:30 — Decision: `CompleteHandover` does **not** require `sheltered` first. A
  guardian can collect a scout from the roadside or straight out of the car, and insisting on
  an arrival at HQ would refuse to record a handover that actually happened — leaving a child
  counted as ours all night. `finished` keeps its own error, because somebody reaching for it is
  reaching for the wrong idea rather than making a typo.
- 2026-08-23 15:40 — Replaced `memberStatusOperation`'s `allowMinted bool` with a named
  `casePolicy` (`caseRequired` / `caseMinted` / `caseOptional`). The boolean was already the
  wrong shape for two behaviours and PRD 007 adds a third; passing an empty sosId through the
  minting path would have worked and been a lie, sending the next reader looking for cases that
  were never created. All four existing call sites updated; the nødtelefon still demands a case.
- 2026-08-23 15:45 — `MaxPlacementLength` counted in **runes**, not bytes: a Danish crew must
  not be told their tent name is too long because of an æ.
- 2026-08-23 15:55 — ✅ All criteria met. 9 new command tests in spejderstatus, 9 in shelter;
  full `go test ./...` green. Verified live against the running dev stack — see task 091's log
  for the end-to-end trace, since the endpoints are what exercise these.
- 2026-08-23 15:56 — Moving to done.
