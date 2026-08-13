# PRD 003 — Automatic patrulje number assignment on acceptance

**Status:** doing
**Author:** agent session
**Created:** 2026-07-31
**Last updated:** 2026-08-13
**Approved:** 2026-08-13
**Shipped:**
**Status note:** approved for implementation — tasks 057–060. The trigger now
fires: the order Pay saga is live in tilmelding (hq deliberately does not mount
it — see §8 Ownership), so `order.paid` is emitted.
**Target users:** organizer (HQ) — indirectly benefits patruljer/participants

---

## 1. Summary

When a patrulje has paid for at least **3 seats** (members), it is considered
**accepted** and is automatically assigned a **team number**. The number is
always **`max assigned number for the year + 1`** — so the first accepted
patrulje in a fresh year gets 1, and if a number was assigned manually (say
300), the next auto-assignment is 301. The sequence **resets every year**.
Assignment is a **side effect** of another domain event and is
realised by publishing `NATHEJK.{year}.patrulje.{teamId}.numberassigned` on the
stream. A patrulje that already has a number keeps it (assignments are
idempotent), and numbers must **not** be re-issued or reshuffled when the
service restarts and replays history.

## 2. Problem & Motivation

- **What problem does this solve?** Team numbers are currently only populated
  when a `patrulje.numberassigned` event exists (the projector in
  `go/nathejk/table/patrulje/consumer.go` writes `teamNumber` from
  `messages.NathejkPatrolNumberAssigned`). But **nothing publishes that event**
  today, so numbers are never assigned automatically. Organizers need accepted
  patruljer to get a stable, gap-free running number as soon as they've paid for
  a viable team (≥3 seats).
- **Why now?** With the order/payment layer in place (orders, `order.paid`,
  computed paid amounts), we can finally detect "paid for ≥3 seats"
  deterministically and drive acceptance from it.
- **Evidence.**
  - `messages.NathejkPatrolNumberAssigned { TeamID, TeamNumber string }` already
    exists in `github.com/nathejk/shared-go/messages` (team.go).
  - `patrulje/consumer.go` already consumes
    `NATHEJK:*.patrulje.*.numberassigned` and runs
    `UPDATE patrulje SET teamNumber=? WHERE teamId=?`.
  - No producer of that subject exists anywhere in the repo.

## 3. Goals

- Automatically assign a team number to a patrulje once it has **paid for at
  least 3 seats** (members). A patrulje is 3–7 seats.
- The assigned number is **`max assigned number in the year + 1`** (so it jumps
  past any manually/legacy-assigned numbers, e.g. a manual 300 → next 301). The
  sequence **resets each year** (first accepted patrulje in a fresh year = 1).
- Assignment is published as `NATHEJK.{year}.patrulje.{teamId}.numberassigned`
  so the existing projector materialises `patrulje.teamNumber`.
- **Idempotent:** a patrulje that already has a number is never reassigned.
- **Numbers are never reused.** If an accepted patrulje later cancels, its
  number is obsoleted (burned), not recycled.
- **Replay-safe:** restarting the service (which replays the whole event log)
  must not publish new/duplicate assignments or renumber existing teams.
- Assignment events are only **published in live mode**, never during catch-up.

## 4. Non-Goals

- Manual override / admin reassignment of team numbers (could be a follow-up).
  Note: manually-assigned numbers *are* respected as the `max` when computing
  the next auto number.
- Un-assigning or recycling numbers. Once assigned, a number is never reused,
  even if the patrulje later cancels/drops below 3 paid seats (the number is
  simply obsoleted).
- Numbering klaner, seniors, or personnel — patruljer only.
- Changing how `teamNumber` is displayed (it already shows in the patrulje list
  as `#`).
- Defining/ъchanging the payment or order flow itself.

## 5. User Stories & Scenarios

- As an **organizer**, I want each patrulje that has paid for a full-enough team
  (≥3 seats) to automatically receive the next running number, so I don't have
  to assign numbers by hand and the numbering reflects payment order.
- As an **organizer**, I want numbers to stay put across deploys/restarts, so a
  patrulje keeps the same number for the whole event.

### Primary happy path

1. A patrulje pays; its order is paid (`order.paid`) and covers ≥3 seats.
2. The acceptance saga (running live) confirms the patrulje has no number yet
   and is now eligible, computes the next number (max assigned number in the
   year + 1), and publishes
   `NATHEJK.{year}.patrulje.{teamId}.numberassigned` with that number.
3. The existing `patrulje` projector writes `teamNumber`; the patrulje list now
   shows the number.

### Edge cases

- **Already numbered:** triggering event arrives for a patrulje that already has
  a number → no-op (no new event).
- **Service restart / replay:** on startup the saga replays all historical
  `numberassigned` events to rebuild "who is numbered" and "highest number
  used", and replays triggering events, but publishes **nothing** until it is
  live. No renumbering, no duplicates.
- **Not yet eligible:** `order.paid` for a patrulje whose total paid seats < 3
  → no assignment (may qualify later on a subsequent event).
- **Concurrent qualifiers:** two patruljer qualify close together → because the
  saga processes messages sequentially, they get consecutive distinct numbers.
- **Manual / legacy numbers:** a manually- or legacy-assigned number (e.g. 300)
  is counted as the current `max`, so the next auto-assignment is 301. Existing
  numbers are never overwritten.
- **Cancellation after acceptance:** the patrulje keeps its (now obsolete)
  number; the number is not recycled and `max` does not decrease.

## 6. Requirements

### Functional

- [ ] Detect when a patrulje has **paid for ≥3 seats** (from `order.paid`).
- [ ] Publish `NATHEJK.{year}.patrulje.{teamId}.numberassigned`
      (`messages.NathejkPatrolNumberAssigned`) with the next number.
- [ ] Next number = **max assigned number in that year + 1** (including manual /
      legacy numbers); sequence resets per year.
- [ ] Never assign a second number to an already-numbered patrulje.
- [ ] Never reuse a number (cancellation obsoletes it; no recycling).
- [ ] Only publish in **live mode** (post-catch-up).
- [ ] Survive restarts without renumbering or duplicate assignment.

### Non-Functional

- Consistent with platform: a `superfluids/streaminterface` consumer wired into
  the `xstream.Mux` in `cmd/api/main.go`; events built with the existing
  subject/publisher helpers; year-scoped.
- Deterministic ordering: numbers follow the order in which patruljer qualify
  (i.e., the order of triggering events on the stream).
- No blocking of the mux: the saga's work per message must be cheap (a lookup +
  at most one publish).

## 7. UX / UI Notes

No new UI. The number surfaces through the existing `teamNumber` column on the
patrulje list (`PatruljeListView.vue`, column `#`) and anywhere `teamNumber` is
already shown. N/A beyond that.

## 8. Technical Considerations

### Ownership — exactly one service mounts this saga

hq owns it. A projector may be mounted in several services because each writes
only its own read model, but a saga publishes to the **shared** event log:
subscriptions are ephemeral ordered consumers with no queue group, so every
process receives every message rather than sharing them out. Two mounts would
both find a patrulje unnumbered and both publish `numberassigned` — and the
projector's `UPDATE patrulje SET teamNumber=?` is unconditional, so the two
numbers would fight rather than converge. It is also why this saga cannot be
scaled horizontally: two replicas of hq would duplicate identically.

hq rather than tilmelding because team numbering is an organizer concern, and hq
already owns the patrulje read model that the eligibility check and the seeding
read from. **It must not also be added to tilmelding.** (The mirror-image
decision was taken for the order Pay saga, which tilmelding owns because it owns
the payment lifecycle; hq's `cmd/api/main.go` records why it is absent there.)

### BFF (Go)

- **New saga/consumer** (e.g. `go/nathejk/table/patruljenumber/` or a saga under
  `patrulje/`), modelled on the existing order Pay saga
  (`go/nathejk/table/order/saga.go`) and implementing:
  - `Consumes()` → the **triggering** subject plus its **own** event subject
    (`NATHEJK.*.patrulje.*.numberassigned`) so it can rebuild state on replay.
  - `HandleMessage(msg)` → routes by subject.
  - `CaughtUp()` (`streaminterface.CatchupListener`) → flip an internal
    `live` flag; only publish when `live` is true.
- **State the saga maintains (in memory, rebuilt on replay), per year:**
  - `assigned map[TeamID]bool` — populated from replayed `numberassigned`
    events (and never re-published).
  - `maxNumber int` — the highest number issued/known **for the year**. The
    next number handed out is `maxNumber + 1`. Seed it at `CaughtUp` from the
    max of (a) replayed `numberassigned` events **and** (b) any existing
    `patrulje.teamNumber` values for the year (so manual/legacy numbers like
    300 are respected → next 301). Numbers are never reused, so `maxNumber`
    only ever increases.
- **On a `numberassigned` message (live or replay):** mark the team assigned and
  raise `maxNumber` to at least its number. Publish nothing.
- **On a triggering message (only act/publish when `live`):**
  1. Resolve the affected patrulje `teamId`.
  2. If `assigned[teamId]` → return (idempotent).
  3. If the patrulje has paid for ≥3 seats → publish
     `NathejkPatrolNumberAssigned{TeamID, TeamNumber: strconv.Itoa(maxNumber+1)}`
     on `NATHEJK.{year}.patrulje.{teamId}.numberassigned`, then optimistically
     mark `assigned[teamId]=true` and bump `maxNumber` (the saga's own
     subscription will also confirm it on the way back).
- **Triggering event — `NATHEJK.*.order.*.paid`:** when an order is paid, if its
  owner is a patrulje, compute the patrulje's total paid **seatCount** and
  compare to 3. The event body (`messages.NathejkOrderPaid`) carries only
  `OrderID`, so the saga must read the order to learn its owner and year.
- **Projection lag is a first-class case.** `PaidQuantityBySKU` counts lines on
  orders whose *projected* status is `paid`, and the order projector is an
  independent consumer, so immediately after `order.paid` the saga can read
  seatCount 0 for an order that is in fact paid. Nothing would re-trigger it, so
  the patrulje would never be numbered. The saga must therefore re-read a bounded
  number of times until the order shows `paid`, exactly as the order saga does
  for an unprojected order (`shared-go/tables/order/saga.go`, `resultUnprojected`
  / `waitBeforeRetry`) — including waiting between reads during replay, since
  back-to-back reads give the other projector no chance to advance.
- **Eligibility rule — "paid for ≥3 seats":** seats are the participation
  product the patrulje buys (a patrulje is 3–7 seats). `seatCount` = sum of
  `quantity` across **paid** participation lines for the patrulje owner, read via
  `order.Queries.PaidQuantityBySKU(ctx, year, ownerType, ownerID)`; eligible when
  `seatCount >= 3`. (A seat *count*, not a paid-amount threshold and not a
  distinct-`memberId` count.)
- **Which SKUs are seats:** resolved from the product catalogue rather than
  hardcoded — `product.Queries.ListEligibleFor(ctx, year, types.TeamTypePatrulje)`
  filtered to `Kind == product.KindParticipation`. For 2026 that is the single SKU
  `participation.patrulje`, but reading the catalogue means renaming or splitting
  the participation product does not silently break acceptance. Merchandise
  (t-shirts) must not count toward seats.
- **`CaughtUp()` reaches the saga only because the live decorator forwards it.**
  Every consumer in hq is wrapped by `live.Notify`, and the jetstream Subscribe
  path discovers `CatchupListener` by asserting on the handler it is given — the
  wrapper. `internal/live/notify.go` forwards `CaughtUp` for exactly this reason.
  Without that, `live` would never flip and the saga would publish nothing, ever.
  Any test of the live-only gate should exercise the wrapped consumer, not just
  the bare one.
- **Wiring:** construct the saga in `cmd/api/main.go`, add it to
  `mux.AddConsumer(...)`. The read/projection side already exists in
  `patrulje/consumer.go` — no change needed there.
- **Message/shared-go:** `NathejkPatrolNumberAssigned` already exists in the
  pinned shared-go; no shared-go change required.

### Data / storage

- No new tables required. `patrulje.teamNumber` is written by the existing
  projector. The saga's state is in-memory and rebuilt from the event log.

### Dependencies & risks

- **The triggering event now fires.** The order Pay saga emits `order.paid` and
  is live in tilmelding. hq must not mount it as well — see Ownership above.
  Should tilmelding's saga ever be removed, this PRD's feature stops triggering
  silently, since no `order.paid` would be published at all.
- **Legacy/pre-existing `teamNumber`s.** Some patruljer already have numbers set
  via other paths (e.g. `klan/consumer.go` inserts patrulje rows with
  `teamNumber`). These are handled by seeding `maxNumber` from the existing
  `patrulje.teamNumber` values (per year) at `CaughtUp`, so auto-assignment
  continues at max+1 and never collides. They are treated as already-assigned
  where a `teamId`→number is known.
- **Live-only publish is essential.** Publishing during catch-up would spam
  duplicate `numberassigned` events on every restart. The `CaughtUp()` gate is
  the crux and must be verified.

## 9. Success Metrics

- Every patrulje that has paid for ≥3 seats has a non-empty
  `teamNumber`.
- Numbers are **unique** per year and never reused; each new auto-assignment is
  strictly greater than every previously assigned number for that year (gaps are
  acceptable, e.g. around manual numbers).
- Restarting the API does not emit new `numberassigned` events for
  already-numbered teams (verify: event count for a team stays at 1) and does
  not change any existing `teamNumber`.

## 10. Rollout / Task Breakdown

- **Phase 1 — Trigger readiness:** ensure a suitable triggering event fires
  (wire the order Pay saga so `order.paid` is emitted, or select an alternative
  trigger).
- **Phase 2 — Acceptance saga:** implement + wire the live-only, replay-safe
  number-assignment saga.
- **Phase 3 — Backfill/verification:** confirm existing paid patruljer get
  numbers and legacy numbers don't collide.

Tasks created in `roadmap/tasks/open/` on approval:

- [ ] 057 — BFF: patrulje number-assignment saga (state from replay, eligibility
      from paid seatCount, live-only publish, lag-tolerant reads). Unwired.
- [ ] 058 — BFF: seed `assigned`/`maxNumber` at `CaughtUp` from existing
      `patrulje.teamNumber` per year, so manual/legacy numbers are respected.
- [ ] 059 — BFF: wire the saga into hq's mux (and only hq's).
- [ ] 060 — Verify replay/restart behaviour end to end: no renumbering, no
      duplicate emits, no reuse after cancellation. Then ship this PRD.

Phase 1 (trigger readiness) is already done: tilmelding runs the Pay saga.

## 11. Open Questions

None outstanding. Both questions below were resolved from the code on
2026-08-13, at approval.

### Resolved (per product decision)

- Trigger = `order.paid`.
- Eligibility = paid **seatCount ≥ 3** (seats = what a patrulje buys; a patrulje
  is 3–7 seats). Not a paid-amount threshold, not distinct `memberId`s.
- Next number = **max assigned in the year + 1** (respects manual/legacy
  numbers; e.g. 300 → 301).
- Sequence **resets per year** (first accepted in a fresh year = 1).
- Numbers are **never reused**; a later cancellation obsoletes the number.
- `TeamNumber` stored as a plain decimal string (no prefix/padding — any prefix
  is a display concern, not storage).

### Resolved at approval (2026-08-13, from the code)

- **seatCount source.** There *is* a dedicated seat product: participation
  products are `Kind == product.KindParticipation`, one per owner type
  (`participation.patrulje` for 2026, 250 DKK/member). seatCount is the paid
  quantity across those SKUs, resolved from the catalogue via `ListEligibleFor`
  rather than hardcoded. Merchandise (`tshirt.adult`) does not count.
- **Order Pay saga.** Wired and live in tilmelding, so `order.paid` is emitted.
  hq deliberately does not mount it (Ownership above).
