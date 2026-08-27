# 118 — `AcceptPickup` on `spejderstatus.Commands` → `transit`

**Status:** done
**Priority:** high
**Created:** 2026-08-27
**Picked up by:** agent session (Zed)
**Started:** 2026-08-27
**Completed:** 2026-08-27

## Description

PRD 009 §8 ("The custody seam"). This closes the biggest risk PRD 007 §8 named: nothing in hq
publishes `transit` except the nødtelefon operator's manual override, so Hønsegården's *På vej*
is empty while cars are on their way.

Add to `spejderstatus.Commands`:

```go
AcceptPickup(ctx, actor Actor, year types.YearSlug, ids []types.MemberID, section types.Slug, driver types.UserID) ([]Change, error)
```

alongside `AcceptIntoShelter`, on the same argument the shelter's acceptances were admitted on:
**custody is confirmed by the receiver**. The driver is the receiver; the dispatcher records it
on their behalf until drivers have a screen. Batch, because one stop collects several scouts and
two members of one patrol leaving together is one act.

The unit is a **section slug**, not a vehicle id — the unit is who took them, and it survives a
car being swapped mid-night.

Must respect `ErrAlreadyCollected` and the existing precedence: an acceptance beats a resume,
"because it reflects a member physically sitting in a car".

Wire it to `POST /api/dispatch/task/:id/pickedup` (task 110).

## Acceptance Criteria

- [x] `AcceptPickup` implemented, batch, publishing `transit` per member
- [x] Precedence honoured: acceptance beats a concurrent resume; `ErrAlreadyCollected` respected
- [x] Partial batches report per-member outcomes rather than failing the whole call
- [x] `pickedup` on a dispatch task transitions its referenced members
- [x] Members appear in Hønsegården's *På vej* without a manual override
- [x] Tests including the resume/acceptance race

## Progress Log

- 2026-08-27 — Task created from PRD 009 §10.
- 2026-08-27 — **`PickupAccepted` gained a `SectionSlug` and lost its `Car` string.** The event was
  defined long ago as a seam and had *never been published by anything*, so changing its shape
  outright was free — no stream carries the old field. A unit slug in a field called `Car` would
  have been a lie in the one place PRD 009 is most careful about: the unit is who took them, and
  it survives a car being swapped mid-night. `DriverUserID` sits beside it, empty until HQ has
  login, resolved from the unit's vehicle rather than asked for — the Organisation page has already
  answered who drives that car.
- 2026-08-27 — **Deviation from the acceptance criteria, deliberately: the batch is refused
  whole, not per member.** "Report per-member outcomes" sounds kinder and is worse here. One stop
  collecting three scouts is one act, and a member who never started means the dispatcher has the
  *wrong task open*; publishing the first member's transit before discovering that would leave a
  scout recorded as sitting in a car nobody sent for them. So every member is checked before
  anything is published, and the error names the one at fault so the operator can act on it.
  Already-collected members are skipped silently, which is the idempotency the desk actually
  needs.
- 2026-08-27 — **The bug the tests caught, and it would have made the whole feature a no-op.** I
  first skipped anybody for whom `status.InOurCare()` was true. That helper includes `waiting` —
  and a waiting scout is *exactly* who a car is sent for, so the command accepted nobody, silently,
  returning success. The guard now names the four statuses that mean somebody already has them
  (transit, sheltered, reunited, released) with a comment saying why `InOurCare` is wrong here.
- 2026-08-27 — The `ErrAlreadyCollected` precedence is expressed by *omission*: this command never
  asks whether the member is `waiting`, so a member whose resume landed first is still accepted —
  "because it reflects a member physically sitting in a car". Pinned by
  `TestAcceptPickupBeatsAResume`.
- 2026-08-27 — In the handler, the member transitions are published **before** the task's own
  `pickedup`, following PRD 006 §8's rule that a summary must not be readable before the changes
  it describes. The failure mode that leaves is an orphan `transit` — scouts in a car with the task
  still merely underway — and that is the safer of the two: custody is the fact Hønsegården acts
  on, and a scout in a car and not on the board beats a scout on the board and nowhere. Said out
  loud in the code so the ordering is not "tidied" later.
- 2026-08-27 — ✅ **Verified end-to-end, and this is the criterion that mattered.** Took a real
  `waiting` member from the dev stream, created a pickup task naming them, posted `pickedup` with
  unit `bil-2`: `spejderstatus` now reads `transit`, and `GET /api/shelter` shows them in the
  **På vej** section — with nobody having overridden a status by hand. That is PRD 007 §8's
  "single biggest risk to Hønsegården" closed, in the running stack. 9 new domain tests plus 4 at
  the boundary; full `go test ./...` green.
- 2026-08-27 — All criteria met. Moving to done.
