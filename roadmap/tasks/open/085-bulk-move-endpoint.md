# 085 — Bulk move endpoint, so one dialog action is one timeline entry

**Status:** open
**Priority:** low
**Created:** 2026-08-17
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] `POST /api/sos/:id/team/:teamId/move` accepting a member list and a destination
- [ ] Requires the origin team to be associated with the case, as collect does
- [ ] Publishes one `spejder` event per member plus **one** `MembersMoved` summary
- [ ] A partial failure returns an error and publishes no summary
- [ ] `SosTeamCard`'s move dialog uses it instead of looping
- [ ] The per-member endpoint still works for the single-row action
- [ ] Renders as one timeline entry for a multi-member move

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created while verifying task 077, which shipped the looping version. Raised
  rather than fixed in place so the trade-off is visible: see Notes for why moving is a
  weaker case for a bulk command than collecting was.
