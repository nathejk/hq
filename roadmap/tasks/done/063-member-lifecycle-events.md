# 063 — Member lifecycle event bodies

**Status:** done
**Priority:** high
**Created:** 2026-08-17
**Picked up by:** agent session
**Started:** 2026-08-17
**Completed:** 2026-08-17

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

- [x] Event bodies added (in `go/nathejk/table/spejderstatus/messages.go` — see log), one
      per transition
- [x] Each carries the acting party and resolves to a `types.MemberStatus`
- [x] No `sosId` field on any of them
- [x] Subject noun (`spejder` vs `member`) confirmed and recorded in the log
- [x] Builds and vets clean; tests cover the status resolution, the self-carrying
      boundary and the in-our-care set

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006.
- 2026-08-17 — Picked up. **Decision: the bodies are defined locally in
  `go/nathejk/table/spejderstatus/` and lifted with the package (task 083), not added to
  shared-go first.** This is the same call PRD 001 recorded for the SOS vocabulary on
  2026-08-11 (`SosCommentID`, `Severity` and the twelve event structs all live in
  `table/sos/` and travel with it), and it applies here for the same reasons plus one
  more: hq would otherwise need a fresh shared-go pseudo-version pin on every iteration
  of a payload that is still being proved by its first consumer. The lift-readiness test
  (task 081) is what keeps this honest, and task 083 is what finishes it. Task title
  updated from "shared-go: …" accordingly; PRD 006 §8 and §10 updated to match.
- 2026-08-17 — **Subject noun decided: `spejder`.** Confirmed against the tree rather than
  guessed: `shared-go/tables/spejder/consumer.go` already consumes
  `NATHEJK.*.spejder.*.updated` and `.deleted`, so `spejder` is an **already-advertised**
  live token and the SPA's `dependsOn` validation will accept it on day one. Introducing a
  `member` entity would have advertised a brand-new token and left every existing
  `spejder`-dependent resource unaware of member changes. The seniors question (PRD 006
  §11) is deferred, not answered: klan members would publish on their own entity when that
  is decided.
- 2026-08-17 — Wrote `messages.go`: seven event bodies plus an `Actor` and a
  `MemberEvent` interface exposing `Status() types.MemberStatus`.
  **The interface is the design decision worth flagging.** The projection depends on it
  rather than on the concrete types, so each body declares the status it resolves to and
  the write path never switches on which event it is looking at. Adding a transition later
  — including one this repo does not publish — is a type plus a subject, not an edit to
  the consumer, and no event can invent a status the lifecycle does not define.
- 2026-08-17 — `TeamMoved` carries **both** `fromTeamId` and `toTeamId`. Needed so the
  projection can recompute `activeMemberCount` for the two affected teams without first
  reading the member's previous row — which keeps task 066's recompute order-independent
  on replay. It resolves to `racing` and does not touch `initialTeamId`: a moved survivor
  is still on the route and can still finish, with a team that is not the one they started
  with.
- 2026-08-17 — Defined `PickupAccepted`, `ShelterAccepted` and `HandoverCompleted` even
  though **nothing publishes them** — they are the car and shelter interfaces' events. That
  is deliberate and is the seam PRD 006 exists to fix: this package consumes them, so the
  shape has to be stated somewhere for those PRDs to be written against, without designing
  their screens or workflow here. `PickupAccepted.Car` names who holds the member, which is
  the question a dashboard needs answered while somebody is in transit and which a bare
  `{memberId, status}` payload could not express.
- 2026-08-17 — Four tests, three of which encode rules that are invisible at the call
  site: every event resolves to a status `Valid()` recognises, only the cancellation (and a
  move) leaves a member able to finish, and the in-our-care set spans exactly
  waiting→sheltered. A move deliberately does **not** count as in our care — counting a
  moved member would inflate the one number the night is judged by.
- 2026-08-17 — ✅ All criteria met. `gofmt`, `go vet` and `go test
  ./nathejk/table/spejderstatus/` clean. Moving to done.
