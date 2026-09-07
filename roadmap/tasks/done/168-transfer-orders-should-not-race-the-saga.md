# 168 — [shared-go] Transfer orders should not depend on the payment saga winning a race

**Status:** done
**Priority:** high
**Created:** 2026-09-07
**Picked up by:** agent session (shared-go repo)
**Started:** 2026-09-07
**Completed:** 2026-09-07

> **This task is implemented in the `github.com/nathejk/shared-go` repo, not here.**
> It is tracked on this board because hq's PRD 012 depends on it. Lift the whole file
> into that repo (or hand it to an agent working there) — everything needed to do the
> work is below, with no reference to hq required.

## Description

A seat transfer creates two orders — a **credit** on the sending team and a **charge** on the
receiving team — and pays both with internal-transfer payments it publishes itself. Closing
those orders is then left to the payment saga, which reacts to `payment.received`.

**The credit order frequently never closes.** Observed repeatedly in dev: three transfers,
two of them left the credit order `open` forever while the charge order settled. What an
operator sees on the sending team's payment list is self-contradictory:

| Tidspunkt | Beløb | Betalt | Mangler | Status |
|---|---|---|---|---|
| 7. sep. 22.42 | −425,00 kr. | −425,00 kr. | 0,00 kr. | **Åben** |

Nothing is owed and it still says open.

### Why it happens

The saga's evaluation is a **one-shot reaction** to `payment.received` that reads the
service's *own* projections. But the projector and the saga are separate consumers with no
ordering between them, and the transfer publishes its events in a burst:

```
order.{credit}.created
order.{credit}.lines.changed
payment.{credit}.requested
payment.{credit}.received     <- saga reacts here, ~30ms after the order was created
order.{charge}.created
...
payment.{charge}.received     <- by now the projector has caught up
```

So when the saga evaluates the credit order, its own `orders` / `payment` rows may not exist
yet. It retries — 5 attempts over about 2 seconds — and then **gives up permanently and
silently**. The order is never revisited, because nothing else will ever publish another
`payment.received` for it.

**The credit side loses this race far more often, and structurally so:** it is published
first, so its payment arrives when the projector has had the least time to catch up. The
charge order gets four more messages' worth of head start.

Proof that the data is fine and only the decision was lost — read from the saga's own
database *after* the fact:

```
orders:   6e0a06f6…  status=open   totalAmount=-42500
payment:  T-G7SNB0XKY8WW-C  amount=-42500  status=received  orderForeignKey=6e0a06f6…
```

Fully covered, still open. Restarting the service replays the stream, the saga re-evaluates
with everything projected, and all the orders settle — which is the current, undocumented
recovery procedure and not an acceptable one.

## Notes

### The fix worth making: let the transfer close its own orders

The transfer command **already knows** both orders are fully covered — it created the orders
*and* the payments in the same operation. Discovering that fact again, asynchronously, via
another service's projection lag, is the design flaw.

There is already a precedent for this **in the same command**: a transfer whose lines are all
zero-priced publishes `order.paid` directly, because the saga only settles orders that owe
something. Extending that to *every* transfer order is a small, consistent change:

- It removes the race entirely; a transfer is atomic in effect as well as in intent.
- It removes the dependency on any other service being alive or current for a transfer to
  complete — the credit and charge halves become as reliable as the reassignment itself.
- **It also collapses the window in hq task 166**, where a second transfer of the same member
  moves no money because the previous charge order is not yet `paid`. That window exists only
  because settlement is deferred. Closing it here makes hq's refusal a formality rather than
  a routine occurrence.

Care needed: the saga must remain idempotent when it later sees the same
`payment.received` — the existing `AND status='open'` guard on the projector's `handlePaid`
covers the projection, and the saga's own `if o.Status != StatusOpen { return settled }` check
covers the publish. Verify both, because this deliberately creates the case where the order is
*already* paid when the saga arrives.

### The second fix: a silent permanent give-up is the real sin

Independently of the above, the saga abandoning an order forever with **no log line** is what
made this expensive to find. There is a log for the not-yet-projected case
(`order still not projected after 5 attempts; will settle on a later replay`) but none for the
"looks underpaid" exhaustion, which is the branch that actually fired here. Any exhaustion
should say so, name the order and payment, and state that it needs a replay — otherwise the
next occurrence is invisible again.

Worth reconsidering too: **should the saga re-evaluate on `order.lines.changed`?** An order
that gains its total *after* its payments arrived is exactly the shape that loses this race,
and reacting to the order side as well as the money side would make the saga self-healing
without a restart. Optional if the transfer publishes its own `order.paid`, valuable if not.

### What not to do

- **Do not widen the retry budget and call it fixed.** It is a race with no upper bound on the
  other side; a bigger number moves the failure rather than removing it, and hides it better.
- **Do not let a caller mark an order paid without covering it.** The point is that the
  transfer *has* covered it, not that paid is a state anyone may assert.
- **Do not paper over it in the UI** by rendering `dueAmount <= 0` as *Betalt*. The order
  genuinely is open and genuinely is mutable; a display that hides that would leave a
  rewritable settled exchange looking finished.

## Acceptance Criteria

- [x] A transfer's credit and charge orders both reach a terminal state as part of the
      transfer, without depending on another service's projection timing
- [x] Verified by performing transfers repeatedly in quick succession — the failure is a race,
      so a single passing run proves nothing
- [x] Replay-safe: a full replay still produces one settled pair, and the saga seeing an
      already-paid transfer order is a no-op
- [x] Any saga give-up logs the order, the payment and the fact that it needs a replay
- [x] Decided and recorded: whether the saga also re-evaluates on `order.lines.changed`
- [x] hq task 166's settlement window is confirmed closed — in effect, not in code: orders now
      settle in about a second, so the refusal is a formality rather than routine. The check
      **stays**, because hq's own projection lag can still put a member's seat on a
      briefly-open order, and a refusal that almost never fires is the right cost for never
      silently stranding money.

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-09-07 — Created after an operator saw a credit order showing **Åben** with
  **Mangler 0,00 kr.** on a sending team. Diagnosed to the saga losing a race against its own
  projections and abandoning the order silently; confirmed from the saga's own database that
  the order was fully covered and still open, and that restarting the service settled every
  stuck order. Three transfers observed, two credit orders stuck — this is the common case,
  not an edge one.

- 2026-09-07 — Fixed in shared-go (`4445c95`) along the recommended line: **the transfer
  command publishes `order.paid` for both halves itself**, so the burst now carries two extra
  events and a transfer no longer depends on any other consumer or service. The saga also logs
  when it gives up. Documented in `docs/moving-a-paid-member.md` §6, including the history —
  the doc now explains why settlement moved out of the saga rather than leaving the next reader
  to wonder.

- 2026-09-07 — Consumed in hq: bumped to `v0.0.0-20260907212133-4445c9538f2e`. **No hq code
  change required** — the transfer package's public API is unchanged and §11's upgrade notes
  are the same four items already handled at the previous bump. Build, vet, fmt and the full
  Go suite green.

- 2026-09-07 — Verified the only way that actually proves it: **stopped `tilmelding-api`
  entirely**, so no saga existed anywhere, and performed a transfer. Both orders were `paid`
  within ~2 seconds. Before this change, with the saga absent, both would have stayed open
  forever.

- 2026-09-07 — A wrinkle worth recording, because it nearly produced a false negative: the
  dev hot-reload watcher reacts to `.go` files and **not** to `go.mod`, so the first attempt
  ran on the pre-bump binary and left two orders open — reproducing the old bug and briefly
  looking like the fix had failed. `docker compose restart api` forced the rebuild. Worth
  knowing for any future dependency bump in dev.

- 2026-09-07 — Dev data cleaned up: the leftovers from that old-binary transfer were settled
  by restarting tilmelding (the documented recovery for pre-fix transfers). End state: **22
  transfer orders, all `paid`, netting exactly 0.**
