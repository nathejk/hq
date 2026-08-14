# 062 — Number assignment misses everyone on a cold start

**Status:** open
**Priority:** high
**Created:** 2026-08-14
**Picked up by:**
**Started:**
**Completed:**

## Description

Reported from production: after deploying the number saga, **no teamNumbers were
assigned**, even though the database holds many orders with `status='paid'`.

## Root cause

The catch-up backfill (061) decides who to number by reading the **read model**,
and `CaughtUp` fires when *this consumer's own* filtered backlog drains — which
says nothing about whether the `orders` projector has finished replaying. The two
are independent consumers.

Measured locally against a cleared database, which is what production does before
a deploy:

```
t=3s  patrulje=43   paid orders=43
t=4s  patrulje=408  paid orders=163
t=5s  patrulje=718  paid orders=177
t=6s  patrulje=718  paid orders=189   <-- sweep runs here
...
t=10s patrulje=718  paid orders=189   (settled)
```

The sweep fires ~6s in, while the orders projection is still filling. For any
patrulje whose paid order had not been projected yet, `paidSeats` returned 0, and
the sweep treats "fewer than 3 seats" as **terminal** — so it is skipped, and
nothing ever revisits it.

Dev looked fine because dev is never cleared: the sweep there read a
fully-populated table left over from the previous run. The bug only appears on a
cold start, which is exactly what production does.

This is the same lag the live path already handles — `attempt` distinguishes "not
projected yet" (retry) from "genuinely under-seated" (terminal) — but the sweep
bypassed that logic entirely.

## Fix: take the candidates from the stream, not the read model

During replay, record the `orderId` of every `order.paid` the saga sees instead of
discarding it. At `CaughtUp`, after seeding and opening the gate, process that
list through the **existing** `attempt` path.

Why this removes the race rather than narrowing it:

- The candidate list comes from the event log, which is complete by definition at
  catch-up — this consumer has read its whole backlog.
- Each candidate goes through the retry loop, which waits when the order is not
  yet projected as paid, instead of silently concluding "ineligible".
- Ordering is stream order, which *is* payment order — a better answer than
  061's string comparison of `changedAt`, which can go too.
- An owner whose seats are split across two paid orders is re-evaluated once per
  paid order, so an under-count from a partially-projected owner is corrected by
  the next entry rather than being final.
- It no longer depends on the `patrulje` projector either. That read stays only
  for seeding existing numbers, where a stale read means a too-low high-water
  mark, which the replayed `numberassigned` events already guard.

## Acceptance criteria

- [ ] `order.paid` seen during replay is recorded, not discarded.
- [ ] At `CaughtUp` the recorded orders are processed through the retry path, in
      stream order, and assign numbers.
- [ ] Test: an order that is not yet projected as paid when the drain starts is
      still numbered once the projection arrives (the prod scenario).
- [ ] Test: duplicate `order.paid` for the same order is recorded once.
- [ ] Test: replay assigns nothing before `CaughtUp`.
- [ ] Still idempotent: a team already numbered (event or read model) is skipped.
- [ ] The read-model candidate sweep, its `changedAt` sort and its `candidate`
      type are gone.
- [ ] The summary log says what it observed — how many deferred orders it
      processed and how many numbers it assigned — so a production deploy is
      diagnosable from one line.
- [ ] `gofmt`, `go vet`, `go test ./...` clean on both resolution paths, `-race`
      clean.
- [ ] Verified against a cleared database locally: a cold start numbers every
      qualifying patrulje.

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-14 15:20 — Task created. Race reproduced locally with a cleared DB and
  the timing measured; see Root cause. Note the current log line cannot confirm
  the diagnosis in production because it prints only the assigned count, so the
  fix must also report what the sweep observed.
