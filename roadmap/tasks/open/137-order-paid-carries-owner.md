# 137 — Put the owner on NathejkOrderPaid so the numbering saga stops chasing crew orders

**Status:** open
**Priority:** medium
**Created:** 2026-09-03

## Description

A startup log full of alarming lines about crew orders:

```
patruljenumber: order 35451330-…-31c29079490a still not projected as paid after 5
attempts; its patrulje may be unnumbered until the next restart
```

That order is `ownerType = crew`. It has no patrulje, and the message is false.

`patruljenumber` triggers on `NATHEJK:*.order.*.paid`, which is emitted for all four
owner types — in dev's data **136 of 217 paid orders are not patruljer** (80 gøgler,
66 klan, 21 crew, against 110 patrulje). The saga *does* filter, in
`attempt()` (`go/nathejk/table/patruljenumber/saga.go`):

```go
o, err := s.orders.GetByID(ctx, orderID)
case errors.Is(err, tables.ErrRecordNotFound):
    return unprojected, false, nil   // retryable
...
if o.OwnerType != types.TeamTypePatrulje {
    return settled, false, nil
}
```

but the filter sits *below* the read, and it has to, because the event says nothing
about who owns the order:

```go
type NathejkOrderPaid struct {
	OrderID    string
	PaidAmount int
	Timestamp  time.Time
}
```

During catch-up the orders projection is an independent consumer still replaying, so
the row is often absent or not yet `paid`. The saga cannot tell "crew order, ignore"
from "patrulje order, projector is behind", so it retries — the right call for a
patrulje, but it means every crew, gøgler and klan order burns the full budget:
`DefaultAttempts` 5 reads across a `DefaultSettle` of 2s, so ~1.6s of naps each,
which is the spacing visible between the log lines.

### What is and is not broken

Nothing is mis-assigned. A number needs `ownerType == patrulje` *and* ≥ `MinSeats`
paid seats, so a crew order can never receive one. Of the 25 warned orders in the
observed run, 9 were patruljer and all 9 already held numbers.

The real risk is the budget. `catchupTimeout` is 5 minutes for seed and drain
together; at ~1.6s per unresolvable order, a few hundred deferred orders can spend
it, and then `catch-up budget spent after N of M` leaves genuinely paid patruljer
unnumbered until the next restart. That is task 062's failure mode reached from a
different direction. The observed run survived it — 433 deferred, only 25 hitting the
full budget — but the margin is thinner than it looks, and it shrinks as the season
fills up.

### The fix

Add `ownerType` and `ownerId` to `NathejkOrderPaid` in shared-go, published by
`tables/order/saga.go`. Both are known where the event is raised. Then hq can decide
before reading anything:

- not a patrulje → settled immediately, no read, no naps, no false warning;
- a patrulje already holding a number → `isAssigned` short-circuits, also without a
  read, which today is equally unreachable until the row appears.

The drain then shrinks to the orders that could actually earn a number.

### Constraint: old events on the stream have no owner

The stream is permanent and hq rebuilds from it on every start, so replay will keep
delivering `order.paid` events published before this change, with the field empty.
The saga must therefore treat an **absent** owner as "unknown, fall back to reading
the read model" — exactly today's behaviour — and only take the fast path when the
field is populated. A blanket `if ownerType != patrulje { skip }` would silently stop
numbering every patrulje whose paid event predates the change, which is most of them.
That distinction is the whole risk in this task and is worth a test of its own.

### Not the fix

- Do not widen `catchupTimeout`: it is the alarm, and the noise is what should go.
- Do not make an unprojected order terminal to save the budget. It is retryable for a
  good reason (task 062): concluding "not a full team" from a half-filled projection
  is what left production unnumbered.
- Do not have the saga wait for the orders projection to finish. They are independent
  consumers by design, and coupling them re-introduces an ordering dependency the
  deferred-drain exists to avoid.

## Acceptance Criteria

- [ ] `NathejkOrderPaid` carries `ownerType` and `ownerId`, populated wherever the
      event is published in shared-go
- [ ] `patruljenumber` skips a non-patrulje order using the event alone — no call to
      `orders.GetByID`, asserted with a reader that fails the test if consulted
- [ ] An order whose event carries no owner still resolves through the read model
      exactly as today, with a test standing in for a replayed pre-change event
- [ ] An event naming a patrulje that already holds a number settles without a read
- [ ] No `still not projected as paid` line is logged for an order the event itself
      shows is not a patrulje
- [ ] Startup measured before and after on the fuller dataset (~433 deferred):
      catch-up wall time and the number of warnings recorded in the log
- [ ] shared-go pin bumped in `hq/go/go.mod`, `go build ./...` and `go test ./...`
      green, and a real startup re-checked

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-09-03 15:10 — Task created from a startup log showing the saga spending its retry budget on crew and gøgler orders. Diagnosed: the trigger subject covers every owner type, the ownerType check is necessarily below the read because the event carries only orderId/paidAmount/timestamp, and an unprojected row is indistinguishable from a non-patrulje one. Verified against dev data — of 25 warned orders, 2 crew, 6 gøgler, 9 patrulje (all already numbered), 8 absent — and confirmed 136 of 217 paid orders in that season are not patruljer.
