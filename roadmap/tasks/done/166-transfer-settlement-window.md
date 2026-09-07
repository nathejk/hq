# 166 — A second transfer inside the settlement window silently moves no money

**Status:** done
**Priority:** high
**Created:** 2026-09-07
**Picked up by:** agent session
**Started:** 2026-09-07
**Completed:** 2026-09-07

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

### Chosen: option 1 — detect and refuse

Option 1 was chosen over 3 (widening what counts as paid in shared-go). Reasons, recorded
because the alternative is the one that would make the problem actually disappear:

- It is fixable **here**, in one place, with no cross-repo round trip.
- Option 3 weakens the "paid means paid" contract that several other readers depend on —
  including `patruljenumber`'s acceptance rule, which counts paid seats. Loosening that to
  fix a display would trade a visible, temporary refusal for an invisible, permanent change
  in what "paid" means.
- The wait is genuinely short: measured at **12–15 seconds** with the payment saga running.
  A refusal that clears by itself in seconds is a fair price.

The cost is accepted rather than hidden: a member is briefly immovable after being moved,
and the message has to be good enough that an operator waits instead of filing a bug.

### Options considered, none of the others taken

1. **Detect and refuse.** ← chosen
2. **Detect and warn.** Rejected: an operator who is told the truth and still allowed to
   proceed will occasionally proceed, and the outcome is money stranded on a team the member
   has left — a worse state than waiting ten seconds.
3. **Widen what counts as paid** (shared-go): let `PaidLinesByMember` include lines on an
   order fully covered by received payments but not yet stamped `paid`. Removes the window
   entirely, but weakens the contract as above. Worth revisiting only if the refusal proves
   annoying in practice.
4. **Mount the saga in hq too.** Rejected on sight: single-mount is deliberate, the
   transition is a read-then-publish with no compare-and-swap, and two mounts would both
   publish.

## Acceptance Criteria

- [x] A second transfer inside the settlement window either succeeds with the money or is
      refused with a message naming the real reason
- [x] The UI never claims "der er ikke betalt" for a member whose seat is paid but in flight
- [x] "Nobody ever paid" and "payment not yet settled" are distinguishable to the operator
- [x] Decision recorded, including whether it needed a shared-go change (it did not)
- [ ] The dev-data inconsistency created while reproducing this is cleaned up or knowingly
      left (marcus, team 3, money on team 4's open order) — **knowingly left**: it is dev
      data, the money is on a now-settled order and findable by its deterministic id, and
      rewriting an event-sourced history to tidy a test would be a worse habit than the mess

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-09-07 — Found and reproduced while verifying PRD 012 (task 163). Not a regression:
  the behaviour follows directly from the saga living in another service, which is why it
  was invisible until a transfer was actually run twice.

- 2026-09-07 — Implemented option 1. `pendingTransferOrder` in `cmd/api/reassign.go` looks
  for an **open** order carrying a `transfer:`-prefixed line for this member; the reassign
  handler refuses with 422 *"en tidligere flytning er ikke afregnet endnu — prøv igen om et
  øjeblik"*, and the candidates endpoint returns `pendingTransfer` so the dialog can say the
  truth. The SPA shows the warning **instead of** the "der er ikke betalt" line and disables
  **Flyt** — both were required: leaving the old line visible would have kept the lie on
  screen next to the new warning.

- 2026-09-07 — Deliberately narrow detection, with a unit test (`reassign_test.go`, 7 cases)
  pinning the three ways it could over-reach: a **settled** transfer is not pending (or the
  refusal would never clear), an ordinary **open** order is not a transfer (or every member of
  a team with an outstanding bill would be frozen), and **another member's** in-flight
  transfer does not block this one (teams share orders).

- 2026-09-07 — Verified against the dev stack by reproducing the race deliberately: two
  moves back to back → first **200** with `amount 25000`, second **422** with the new message
  rather than the silent `amount 0`. Waited 15 seconds and retried → **200 with `amount
  25000`**, so the "prøv igen om et øjeblik" promise is true and the refusal clears itself. A
  refusal that never cleared would have been worse than the bug it replaced.

- 2026-09-07 — Completed. `vite build` clean, Go build/vet/fmt clean, new test green.
