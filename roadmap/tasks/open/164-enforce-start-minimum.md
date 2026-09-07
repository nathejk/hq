# 164 — Enforce the 3-member minimum at the start gate

**Status:** open
**Priority:** medium
**Created:** 2026-09-07
**Picked up by:**
**Started:**
**Completed:**

## Description

From **PRD 012** §4 (X1). Related to, but deliberately not part of, the seat-transfer work.

**A patrulje below three members may not start.** That is a rule of the event, and the
server does not have it. `startPatruljeHandler` (`go/cmd/api/patrulje.go`) accepts whatever
starters it is given with no count check at all, and `MinMemberCount: 3` is a display value
served to the SPA in `TeamConfig`.

PRD 012 leans on this rule: it is *why* the transfer deliberately refuses to enforce a lower
bound on the team a member is moved out of. A team that has been hollowed out keeps its
number — `patruljenumber` has no unassign path — and the start gate is the safety net that
stops a stale number putting an ineligible team on the route. Right now that net does not
exist, so the reasoning in PRD 012 §4 is sound but unenforced.

## Notes

- The value has four call sites with three different pairs: `patrulje.go` uses 3/7,
  `klan.go`, `badut.go` and `mail.go` use 1/4, and `sos.go` builds its own 3/7 literal. Task
  074 (done) looked at this config; check what it settled before adding a fifth copy.
- Enforcing at the gate is the cheap half. The harder question is what the operator sees:
  a refusal is only useful if it says *why* and points at the way out — moving the remaining
  members to a team that has not started, which is exactly what
  `PUT /api/member/:memberId/reassign` now does. Wire the message to the remedy.
- Consider whether the check counts roster members or **paid seats**. They are different
  numbers, and `patruljenumber.MinSeats = 3` already counts paid seats for acceptance. A team
  with four members and two paid seats is a real case; decide deliberately which one blocks a
  start.
- `patrulje.memberCount` is written *by* the start event from `len(body.Members)`, so the
  check has to be on the request, not the read model.

## Acceptance Criteria

- [ ] Starting a patrulje with fewer than the minimum is refused server-side
- [ ] The refusal names the rule and points at the transfer as the remedy
- [ ] Decided and documented: roster members or paid seats
- [ ] No fifth hardcoded copy of the limits — reuse whatever task 074 settled on
- [ ] Test covering the refusal and the boundary (exactly 3 starts)

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-09-07 — Created from PRD 012 §4, which relies on this gate existing to justify not
  checking a lower bound when moving members out of a team.
