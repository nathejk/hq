# 118 — `AcceptPickup` on `spejderstatus.Commands` → `transit`

**Status:** open
**Priority:** high
**Created:** 2026-08-27
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] `AcceptPickup` implemented, batch, publishing `transit` per member
- [ ] Precedence honoured: acceptance beats a concurrent resume; `ErrAlreadyCollected` respected
- [ ] Partial batches report per-member outcomes rather than failing the whole call
- [ ] `pickedup` on a dispatch task transitions its referenced members
- [ ] Members appear in Hønsegården's *På vej* without a manual override
- [ ] Tests including the resume/acceptance race

## Progress Log

- 2026-08-27 — Task created from PRD 009 §10.
