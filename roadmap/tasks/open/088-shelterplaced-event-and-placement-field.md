# 088 — ShelterPlaced event + Placement on ShelterAccepted

**Status:** open
**Priority:** high
**Created:** 2026-08-23
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] `ShelterPlaced` defined with a `Status()` of `sheltered`
- [ ] `Placement` added to `ShelterAccepted` as `omitempty`
- [ ] `shelter.placed` added to `Consumes()` in `spejderstatus/consumer.go`, matched before
      the four-part patterns
- [ ] `spejderstatus` writes `sheltered` on `shelter.placed` (idempotent; the placering is
      the `shelter` table's business)
- [ ] Message tests in `messages_test.go` covering both, following the existing style
- [ ] No import of `nathejk.dk/...`; `lift_test.go` green

## Progress Log

- 2026-08-23 09:30 — Task created from PRD 007 §10.
