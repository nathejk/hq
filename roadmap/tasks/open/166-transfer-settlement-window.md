# 166 — A second transfer inside the settlement window silently moves no money

**Status:** open
**Priority:** high
**Created:** 2026-09-07
**Picked up by:**
**Started:**
**Completed:**

## Description

Found while verifying PRD 012 end to end (task 163).

A transfer's **charge** order is closed by the payment saga, which is mounted in
**tilmelding**, not hq. Until it closes, that order is `open` — and
`order.PaidLinesByMember`, which is what the transfer reads to decide what a member has
paid for, deliberately only considers **paid** orders.

So for as long as a transfer's charge order is unsettled, the member's seat is invisible to
a second transfer. Moving them on again in that window:

- transfers **no money** (`amount: 0`, `lineCount: 0`),
- still reassigns the member,
- and makes the UI say *"Der er ikke betalt for denne deltager, så der overføres ingen
  betaling"* — which is **false**. Somebody did pay; the money is in flight, or stranded on
  the previous team's open order.

The result is a member on a team whose seat was never paid for, with the money left behind
on a team they are no longer on, and no error anywhere.

### Reproduced

In dev, with tilmelding's API not running (so nothing settles anything):

1. Moved `37d62bef-0688-48e1-8e42-090f93128777` (marcus, real MobilePay-funded seat) from
   team 75 to team 4 → `amount 25000, lineCount 1`, provenance correctly naming the root
   payment `874c2349-af4f-445a-a5bc-4586157df1af`.
2. Moved the same member from team 4 to team 3 → **`amount 0, lineCount 0`**.

The money is now on team 4's open charge order and the member is on team 3.

## Notes

- **Normally the window is short.** The transfer publishes `payment.received` itself, so a
  healthy tilmelding settles the charge order within moments and a second move behaves
  correctly. The window is only long when tilmelding is down, lagging, or replaying.
- **It is unbounded when tilmelding is not running**, which is exactly the state a dev
  machine is usually in — so this will be hit in dev far more often than in production, and
  it looks like the feature is broken rather than like a service is missing.
- The order ids are deterministic per `(year, memberId, fromTeamId, toTeamId)`, so the
  stranded money is findable and the situation is repairable by hand. Nothing is lost.
- Whatever the fix, it should not make an *unpaid* member look paid: the honest distinction
  is between "nobody ever paid for this member" and "their seat is paid for but the transfer
  that moved it has not settled yet". Today both render as the former.

### Options, none yet chosen

1. **Detect and refuse.** The candidates endpoint and the command look for an unsettled
   transfer charge order naming this member and say so — *"en tidligere flytning er ikke
   afregnet endnu, prøv igen om et øjeblik"*. Honest, cheap, and blocks a real foot-gun. It
   does mean a member can be temporarily immovable.
2. **Detect and warn.** Allow it, but replace the "nobody paid" line with the truth so the
   operator can decide. Keeps the tool unblocking, risks them proceeding anyway.
3. **Widen what counts as paid** (shared-go): let `PaidLinesByMember` include lines on an
   order that is fully covered by received payments but not yet stamped `paid`. Fixes it at
   the root and removes the window entirely — but it weakens the "paid means paid" contract
   that several other readers rely on, and would need shared-go's agreement.
4. **Mount the saga in hq too.** Rejected on sight: single-mount is deliberate, the
   transition is a read-then-publish with no compare-and-swap, and two mounts would both
   publish.

Option 1 or 3. Worth deciding with whoever owns shared-go's order package, since 3 is the
only one that makes the problem actually go away.

## Acceptance Criteria

- [ ] A second transfer inside the settlement window either succeeds with the money or is
      refused with a message naming the real reason
- [ ] The UI never claims "der er ikke betalt" for a member whose seat is paid but in flight
- [ ] "Nobody ever paid" and "payment not yet settled" are distinguishable to the operator
- [ ] Decision recorded, including whether it needed a shared-go change
- [ ] The dev-data inconsistency created while reproducing this is cleaned up or knowingly
      left (marcus, team 3, money on team 4's open order)

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-09-07 — Found and reproduced while verifying PRD 012 (task 163). Not a regression:
  the behaviour follows directly from the saga living in another service, which is why it
  was invisible until a transfer was actually run twice.
