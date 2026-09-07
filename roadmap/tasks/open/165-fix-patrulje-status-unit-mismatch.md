# 165 — Fix the øre/kroner comparison in the patrulje list status

**Status:** open
**Priority:** medium
**Created:** 2026-09-07
**Picked up by:**
**Started:**
**Completed:**

## Description

From **PRD 012** §8 (X2). A pre-existing bug, found while researching the seat transfer and
recorded so it is not later blamed on it.

`go/nathejk/table/patrulje/query.go` derives a display `signupStatus` for a team whose column
is empty by comparing a paid amount against what the team owes:

```go
payableAmount := p.TshirtCount*175 + p.MemberCount*250
if p.SignupStatus != "" {
} else if p.PaidAmount == 0 {
	p.SignupStatus = types.SignupStatusPay
} else if p.PaidAmount >= payableAmount {
	p.SignupStatus = types.SignupStatusPaid
} else {
	p.SignupStatus = types.SignupStatusSemipaid
}
```

**The two sides are in different units.** `PaidAmount` is summed from `payment.amount`, which
is **øre**; `payableAmount` is built from `175` and `250`, which are **kroner**. So the
comparison is off by a factor of 100 and `PAID` is true for very nearly any payment — a team
that has paid 25 kr of 1 750 kr shows as fully paid.

Note task 005 (done) fixed a *different* problem in the same computation — routing the paid
amount through orders — and left the unit mismatch in place.

## Notes

- The prices are hardcoded in **three** places in three units: `TeamConfig` in kroner (four
  handlers, three different limit pairs), the product catalogue in øre
  (`shared-go/tables/product/seeds_2026.go`), and this SQL-adjacent arithmetic. The
  catalogue is the only one that is authoritative — prefer reading it over adding a fourth
  copy with a `*100`.
- Seat transfers change `memberCount` and `tshirtCount` on both teams, so this will start
  being wrong more visibly and on teams somebody is actively looking at.
- Check whether the derived status is still needed at all. It only fires when the stored
  `signupStatus` is empty, and orders now carry a real paid/due amount — the honest fix may
  be to delete the derivation rather than repair its arithmetic.

## Acceptance Criteria

- [ ] Paid and payable compared in the same unit
- [ ] Prices read from the product catalogue, or a written reason why not
- [ ] Decided whether the derived status is still needed; deleted if not
- [ ] A team that has paid a fraction of what it owes no longer shows as Betalt

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-09-07 — Created from PRD 012 §8, which flagged it as adjacent to the seat transfer and
  deliberately out of its scope.
