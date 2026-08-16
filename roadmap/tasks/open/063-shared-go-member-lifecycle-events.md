# 063 — shared-go: member lifecycle event bodies

**Status:** open
**Priority:** high
**Created:** 2026-08-17
**Picked up by:**
**Started:**
**Completed:**

## Description

From **PRD 006** §8 (Events). Cross-repo work in `nathejk/shared-go`, on the critical
path for everything else in this PRD.

`messages.NathejkMemberStatusChanged` does **not** exist in shared-go — the reference in
the old commented-out projection points at a type that lives only in the legacy local
message packages. Design and add the member lifecycle events to
`shared-go/messages/member.go`.

**A single generic "status changed" event fits this model poorly.** Each transition is a
distinct act by a distinct party — a request to leave, a decision to carry on, an
acceptance into a car, an acceptance at the shelter, a handover — so model them as
separate events that each carry the acting party and resolve to a
`types.MemberStatus`. That is what makes the *acceptor* recordable, which a bare
`{memberId, status}` payload cannot express, and it answers "who holds this member?" for
free because the car's acceptance event names the car.

Proposed set (names to be confirmed while implementing):

- `member.withdrawal.requested` — → `waiting`. Published by hq (this PRD).
- `member.withdrawal.cancelled` — → `racing`, the member carries on. Published by hq.
- `member.status.overridden` — the correction path. Published by hq.
- `member.team.moved` — `currentTeamId` changes. Published by hq.
- `member.pickup.accepted` — → `transit`. Published by the **future car interface**.
- `member.shelter.accepted` — → `sheltered`. Published by the **future shelter interface**.
- `member.handover.completed` — → `released` | `reunited`.

## Notes

- **One event per updated member, and no `sosId` in any payload** (PRD 006 §11
  Decisions). `sosId` is a parameter of the *command*, not a field on the event — that is
  what lets the car and shelter interfaces publish these same events later without
  inventing a case they know nothing about. The case-scoped summary is a separate `sos`
  event, task 071.
- Subject entity is the **member**: `NATHEJK.{year}.spejder.{memberId}.<event>`, which
  reuses the already-advertised `spejder` live token (shared-go's `spejder` consumer
  already consumes `spejder.*.updated` / `.deleted`). The `member` vs `spejder` noun is
  an open question in PRD 006 §11 tangled with seniors — **confirm before publishing**,
  because the token is the frontend contract and a wrong one fails silently.
- Every payload carries the acting party. In hq that resolves to an empty actor until
  the auth service lands (PRD 001 §6 Auth); wire it anyway.
- The last three are defined here but **published by nobody yet** — that is deliberate,
  it is the seam the car and shelter PRDs build against.
- `types.MemberStatus` itself is already done and pinned (`go/go.mod` at
  `v0.0.0-20260815075712-35c10e0f6942`); this task adds only messages.

## Acceptance Criteria

- [ ] Event bodies added to `shared-go/messages/member.go`, one per transition
- [ ] Each carries the acting party and resolves to a `types.MemberStatus`
- [ ] No `sosId` field on any of them
- [ ] Subject noun (`spejder` vs `member`) confirmed and recorded in the log
- [ ] shared-go builds and vets clean; new pin available for hq's `go.mod`

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006.
