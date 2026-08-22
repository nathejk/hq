# 088 — ShelterPlaced event + Placement on ShelterAccepted

**Status:** done
**Priority:** high
**Created:** 2026-08-23
**Picked up by:** agent session (Zed)
**Started:** 2026-08-23
**Completed:** 2026-08-23

## Description

PRD 007 needs one new lifecycle event and one new field, both in
`go/nathejk/table/spejderstatus/messages.go` beside their siblings — the status mapping
lives in one place, and every body resolves to exactly one `types.MemberStatus` via
`Status()`.

**`ShelterPlaced`** — the shelter records where a scout has been put, or moves them.

```go
type ShelterPlaced struct {
    MemberID  types.MemberID `json:"memberId"`
    TeamID    types.TeamID   `json:"teamId"`
    Placement string         `json:"placement"`
    Actor     Actor          `json:"actor"`
}

func (ShelterPlaced) Status() types.MemberStatus { return types.MemberStatusSheltered }
```

Subject `NATHEJK.{year}.spejder.{memberId}.shelter.placed`. Its own event rather than a
re-publish of `shelter.accepted`, because moving a sleeping child to another tent is a
distinct act and reads as one on the timeline.

**`Placement string \`json:"placement,omitempty"\`` on `ShelterAccepted`** — the crew types
the tent while accepting; two acts in one gesture should not need two events. Free to add:
nothing has ever published `ShelterAccepted`, so there is no compatibility question.

Add both subjects to the `spejderstatus` consumer's `Consumes()`. It already subscribes to
`shelter.accepted` and `handover.completed` — the comment there says these belong to "the
car and shelter interfaces", which is this PRD. Note the ordering trap the consumer's own
comment records: five-part subjects must be matched before the four-part `spejder.*.deleted`
patterns.

No `sosId` on either, deliberately — see the "No sosId, deliberately" note in that file.
The shelter may receive a scout nobody opened a case about.

## Acceptance Criteria

- [x] `ShelterPlaced` defined with a `Status()` of `sheltered`
- [x] `Placement` added to `ShelterAccepted` as `omitempty`
- [x] `shelter.placed` added to `Consumes()` in `spejderstatus/consumer.go`, matched before
      the four-part patterns
- [x] `spejderstatus` writes `sheltered` on `shelter.placed` (idempotent; the placering is
      the `shelter` table's business)
- [x] Message tests in `messages_test.go` covering both, following the existing style
- [x] No import of `nathejk.dk/...`; `lift_test.go` green

## Progress Log

- 2026-08-23 09:30 — Task created from PRD 007 §10.
- 2026-08-23 10:40 — Picked up. Taken before 087 because the projection consumes the event.
- 2026-08-23 10:50 — `ShelterPlaced` and `ShelterAccepted.Placement` added to `messages.go`.
  Also corrected the "published by the future shelter interface" comments and the
  `Consumes()` note that called the last three unpublished — only `pickup.accepted` is
  unpublished now, and a comment that lies about which events have producers is worse than
  no comment.
- 2026-08-23 10:55 — `shelter.placed` added to `Consumes()`, to `HandleMessage` (ahead of the
  four-part patterns, per the ordering trap the file's own comment records) and to
  `teamID`'s type switch — that last one is easy to miss and its absence would silently
  write an empty currentTeamId, making the member invisible to every per-team query.
- 2026-08-23 11:00 — Decision: `shelter.placed` goes through `setStatus` even though the
  status write is a no-op. Two reasons: it puts the move between tents on the member's
  timeline, which is exactly what somebody who cannot find a child asks for; and it is the
  one path that would heal a member whose acceptance event was lost.
- 2026-08-23 11:10 — ✅ All criteria met. Added two consumer tests: one asserting the
  placering never reaches `spejderstatus` (that table lifts to shared-go verbatim, and a
  placering column in it is how the lift stops being a file move), one asserting the move is
  recorded in history. Existing `TestEverySubjectDeclaredIsHandled` confirms the new subject
  is handled — it would have caught a subscription with no case.
- 2026-08-23 11:12 — Package tests green. Moving to done.
