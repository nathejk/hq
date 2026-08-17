# 085 — Bulk move endpoint, so one dialog action is one timeline entry

**Status:** done
**Priority:** low
**Created:** 2026-08-17
**Picked up by:** agent session
**Started:** 2026-08-17
**Completed:** 2026-08-17

## Description

Found while building task 077. Not a defect — the recorded data is correct either way —
but an inconsistency with the reasoning task 073 used, and a mild footgun.

**Today:** the "Flyt de resterende" dialog moves N selected members by issuing **N
requests** to `PUT /api/member/:memberId/team`. Each publishes its own `member.moved`
summary, so moving two survivors to one patrol produces **two** timeline entries.

**Compare with collecting a team**, where exactly this shape was rejected (task 073):
three separate calls from the browser could half-succeed and leave a team split across
two states with nobody noticing, so the loop was moved to the server behind
`POST /api/sos/:id/team/:teamId/collect` and renders as one line.

The same argument applies here. If the second of two moves fails, one member has moved
and one has not, and the operator gets an error toast without being told which — from a
dialog whose selection has already been cleared.

**Proposed:** `POST /api/sos/:id/team/:teamId/move` taking `{ memberIds: [...], toTeamId }`,
looping server-side and publishing one `MembersMoved` summary. The event body already
supports it: `MembersMoved.Members` is a list, and each entry carries its own
`toTeamId` precisely so one operation can name several members.

## Notes

- **Why this was not done in 077:** moving is genuinely per member in a way collecting is
  not. Collecting sends every member to one state, so it is unambiguously one operation.
  Members being moved may go to **different** destinations, so per-member events are the
  honest grain — the dialog moving N to *one* target is one operation only in the common
  case. A bulk endpoint is therefore a refinement of the common case rather than a
  correction of the model.
- The per-member endpoint stays regardless: the row-level **Flyt til anden patrulje**
  action uses it, and the correction interface (task 084) will too.
- Keep the partial-failure behaviour explicit when implementing: the collect command
  returns the error and discards its changes so no summary claims an operation that did
  not finish. Do the same here rather than reporting a half-move as success.
- Low priority: the data is right either way, and two timeline lines for two moves is
  legible, just chattier than the design intends.

## Acceptance Criteria

- [x] `POST /api/sos/:id/team/:teamId/move` accepting a member list and a destination
- [x] Requires the origin team to be associated with the case, as collect does
- [x] Publishes one `spejder` event per member plus **one** `MembersMoved` summary
- [x] A partial failure returns an error and publishes no summary
- [x] `SosTeamCard`'s move dialog uses it instead of looping
- [x] The per-member endpoint still works for the single-row action
- [x] Renders as one timeline entry for a multi-member move

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created while verifying task 077, which shipped the looping version. Raised
  rather than fixed in place so the trade-off is visible: see Notes for why moving is a
  weaker case for a bulk command than collecting was.
- 2026-08-17 — Picked up. `MoveMembers` on the member commander, `moveMembersHandler`, and
  the dialog switched from N `PUT`s to one `POST`.
- 2026-08-17 — **Validates every member before publishing anything.** A loop that checked as
  it went would discover an illegal move half way through and leave the operation partly
  applied — which is the exact failure this command exists to prevent, so doing it that way
  would have been theatre. There is a test.
- 2026-08-17 — **Strengths are computed for the whole operation, not per member.** This is
  the part that would have been wrong if I had simply looped the existing per-member
  command: moving three members out one at a time reports the origin at 3, then 2, then 1 —
  three different "resulting strengths" for a single step to 0, with the timeline naming
  whichever came last. `strengthWithout` excludes the whole moving set at once. Two tests
  cover it, including that every move in one operation reports the *same* pair.
- 2026-08-17 — Extracted `teamOnCase` and `moveTarget` while wiring this up: the association
  guard was needed identically here, and the destination validation was duplicated. Both
  carry the reasoning that earned them — the association check exists because these two
  endpoints act on a **set** derived from a team id alone, and `moveTarget` deliberately does
  *not* require the destination to still have racing members, which is what makes
  discontinuation reversible (task 077).
- 2026-08-17 — A stub artifact nearly gave a false pass: `stubQueries.GetByMemberID` returned
  the same member for every id, so the strength assertions were satisfied for the wrong
  reason — the code looked like it was handling one member three times. Fixed the stub to
  resolve the requested id against the team rows. Worth noting because the test failed
  *correctly* and my first instinct was to doubt the code.
- 2026-08-17 — ✅ Verified end to end on two real 2025 patrols:
  - moving with an origin **not** on the case → rejected, "patruljen er ikke tilknyttet
    sagen"
  - two members in one request → both moves report the same strengths (`from 2`, `to 6`),
    the whole-operation semantics
  - `sos_activity` holds **one** `member.moved` entry, not two
  - it renders as: *"Deltagere flyttet: Ørkenrotternes bande — Peter Norsker → Enhjørning,
    Augusta Holm Steenstrup → Enhjørning, 2 tilbage i løbet"*
  - projections landed: origin 4→2, destination 4→6
- 2026-08-17 — Dev data restored by moving both back (which also re-exercised the reversal
  path): both patrols at 4 racing, nobody in care, no open probe cases.
- 2026-08-17 — ✅ All criteria met. Moving to done — this closes the last implementation task
  derived from PRD 006; only the post-stabilisation lift (083) remains.
