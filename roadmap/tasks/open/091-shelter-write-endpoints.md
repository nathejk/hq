# 091 — Shelter write endpoints

**Status:** open
**Priority:** medium
**Created:** 2026-08-23
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] Three routes registered and handled
- [ ] No `sosId` required by any of them
- [ ] `placement` trimmed and length-capped; empty allowed on `/shelter`, rejected on
      `/placement`
- [ ] `/handover` rejects anything but `released`/`reunited`, with a message naming the two
- [ ] No-ops return `200`
- [ ] OpenAPI annotations present and complete on all three
- [ ] Handler tests for the happy paths and each rejection

## Progress Log

- 2026-08-23 09:30 — Task created from PRD 007 §10.
