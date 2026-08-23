# 091 — Shelter write endpoints

**Status:** done
**Priority:** medium
**Created:** 2026-08-23
**Picked up by:** agent session (Zed)
**Started:** 2026-08-23
**Completed:** 2026-08-23

## Description

Three endpoints over the commands from task 089, in `go/cmd/api/shelter.go` (PRD 007 §8).

| Method | Path | Body | Purpose |
|---|---|---|---|
| PUT | `/api/member/:memberId/shelter` | `{placement?}` | Accept into the shelter |
| PUT | `/api/member/:memberId/placement` | `{placement}` | Set or change placering |
| PUT | `/api/member/:memberId/handover` | `{to}` | `released` or `reunited` |

Registered on the member, beside the existing `waiting`/`racing`/`status`/`team` routes: a
member's status is a fact about the member.

**None of them requires a `sosId`.** The shelter may receive a scout nobody opened a case
about, and these events are case-free by design. This depends on the helper relaxation in
task 089 — do not fake it with an empty id.

`placement`: trimmed, max 64 characters, may be empty on `/shelter` (accept now, place
later) but not on `/placement`. Free text is accepted by design — the vocabulary is
suggested, never enforced (PRD 007 §6).

`to` on `/handover` must be `released` or `reunited`; `400` otherwise, and `finished` is
rejected with an error that says why rather than a bare validation failure.

A no-op returns `200`, not an error: the second crew member to press **Modtaget** should see
the state they wanted, not a failure.

**OpenAPI annotations on all three**, following `cmd/api/order.go`.

## Acceptance Criteria

- [x] Three routes registered and handled
- [x] No `sosId` required by any of them
- [x] `placement` trimmed and length-capped; empty allowed on `/shelter`, rejected on
      `/placement`
- [x] `/handover` rejects anything but `released`/`reunited`, with a message naming the two
- [x] No-ops return `200`
- [x] OpenAPI annotations present and complete on all three
- [x] Handler tests for the happy paths and each rejection

## Progress Log

- 2026-08-23 09:30 — Task created from PRD 007 §10.
- 2026-08-23 16:00 — Picked up straight after 089, since the commands are useless unreachable.
- 2026-08-23 16:10 — The two status endpoints reuse `memberStatusOperation` under
  `caseOptional`. `/placement` deliberately does **not**: it changes no status, so there is no
  `Change` to summarise and nothing for the strength arithmetic to do — reusing that helper
  would have meant inventing a status transition to describe a scout staying exactly where they
  are in the lifecycle.
- 2026-08-23 16:15 — `shelterInput` buffers and restores the request body, because
  `memberStatusOperation` decodes it too and a body can only be read once. The alternative was
  adding `placement`/`to` to the shared `memberRequest`, which is the nødtelefon's shape — every
  member endpoint would then advertise a placering it has no use for. A malformed body is not
  reported twice: the shared helper answers 400 for it a moment later.
- 2026-08-23 16:25 — Refusals mapped to Danish messages phrased as *what to do next* rather
  than as rules broken — "spejderen er ikke i Hønsegården — modtag dem først" names the button
  that fixes it. A test asserts the raw domain error never reaches the client.
- 2026-08-23 16:35 — Blocker (self-inflicted, 10 minutes): the first handler test hung until the
  timeout instead of failing. Cause was `data.Models{Teams: nil}` — `memberStatusOperation`
  looks the patrol up unconditionally for the summary line. Fixed with a `fakeTeams`. Worth
  knowing that a nil model in this struct hangs rather than panics cleanly.
- 2026-08-23 16:50 — ✅ All criteria met. 10 endpoint tests; `cmd/api` at 26 passing; full Go
  suite green.
- 2026-08-23 17:00 — **Verified end to end against the running dev stack**, which is the part
  worth recording. A real scout (`Sofija`, in `transit`) was accepted with `"  Telt 4  "`:
  - the response returned `from: transit → to: sheltered`;
  - the `shelter` row landed with `placement = "Telt 4"` — trimmed — and both timestamps set;
  - `GET /api/shelter` moved her from the arrivals queue to *I Hønsegården*, and `placements`
    began suggesting `Telt 4`;
  - re-accepting answered `{"change": null}` — the no-op path, so a second crew member pressing
    Modtaget publishes nothing;
  - `/placement` moved her to `Telt 7`; the same call for a `waiting` scout was refused with
    the "modtag dem først" message;
  - `/handover` with `finished` was refused; with `released` it succeeded, **deleted the shelter
    row** (the bed is free) and dropped the in-our-care total from 3 to 2;
  - her `spejderstatuslog` now reads `… transit → shelter.accepted → shelter.placed →
    handover.completed`, which is the custody chain PRD 006 designed and nothing had ever
    written until now.
- 2026-08-23 17:02 — Moving to done.
