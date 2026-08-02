# PRD 003 — Automatic patrulje number assignment on acceptance

**Status:** draft
**Author:** agent session
**Created:** 2026-07-31
**Last updated:** 2026-07-31
**Target users:** organizer (HQ) — indirectly benefits patruljer/participants

---

## 1. Summary

When a patrulje has paid for at least **3 members**, it is considered
**accepted** and is automatically assigned a sequential **team number**
(1, 2, 3, … 200+). The first patrulje to qualify gets number 1, the next gets 2,
and so on. Assignment is a **side effect** of another domain event and is
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
  a viable team (≥3 members).
- **Why now?** With the order/payment layer in place (orders, `order.paid`,
  computed paid amounts), we can finally detect "paid for ≥3 members"
  deterministically and drive acceptance from it.
- **Evidence.**
  - `messages.NathejkPatrolNumberAssigned { TeamID, TeamNumber string }` already
    exists in `github.com/nathejk/shared-go/messages` (team.go).
  - `patrulje/consumer.go` already consumes
    `NATHEJK:*.patrulje.*.numberassigned` and runs
    `UPDATE patrulje SET teamNumber=? WHERE teamId=?`.
  - No producer of that subject exists anywhere in the repo.

## 3. Goals

- Automatically assign a team number to a patrulje once it has **paid for ≥3
  members**.
- Numbers are **sequential and gap-free per year**, starting at 1 for the first
  accepted patrulje.
- Assignment is published as `NATHEJK.{year}.patrulje.{teamId}.numberassigned`
  so the existing projector materialises `patrulje.teamNumber`.
- **Idempotent:** a patrulje that already has a number is never reassigned.
- **Replay-safe:** restarting the service (which replays the whole event log)
  must not publish new/duplicate assignments or renumber existing teams.
- Assignment events are only **published in live mode**, never during catch-up.

## 4. Non-Goals

- Manual override / admin reassignment of team numbers (could be a follow-up).
- Un-assigning or recycling numbers when a patrulje later drops below 3 paid
  members or cancels. Once accepted, the number stays (see Open Questions).
- Numbering klaner, seniors, or personnel — patruljer only.
- Changing how `teamNumber` is displayed (it already shows in the patrulje list
  as `#`).
- Defining/ъchanging the payment or order flow itself.

## 5. User Stories & Scenarios

- As an **organizer**, I want each patrulje that has paid for a full-enough team
  (≥3 members) to automatically receive the next running number, so I don't have
  to assign numbers by hand and the numbering reflects payment order.
- As an **organizer**, I want numbers to stay put across deploys/restarts, so a
  patrulje keeps the same number for the whole event.

### Primary happy path

1. A patrulje pays; its order becomes paid and covers ≥3 members.
2. The acceptance saga (running live) sees the triggering event, confirms the
   patrulje has no number yet and is now eligible, computes the next number
   (highest assigned so far + 1), and publishes
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
- **Not yet eligible:** triggering event for a patrulje with < 3 paid members →
  no assignment (may qualify later on a subsequent event).
- **Concurrent qualifiers:** two patruljer qualify close together → because the
  saga processes messages sequentially, they get consecutive distinct numbers.
- **Pre-existing/legacy numbers:** patruljer that already carry a number from
  another path (e.g. imported via the klan consumer) must not collide with newly
  issued numbers (see Open Questions).

## 6. Requirements

### Functional

- [ ] Detect when a patrulje becomes "paid for ≥3 members" from a domain event.
- [ ] Publish `NATHEJK.{year}.patrulje.{teamId}.numberassigned`
      (`messages.NathejkPatrolNumberAssigned`) with the next sequential number.
- [ ] Assign the first accepted patrulje number 1, then increment.
- [ ] Never assign a second number to an already-numbered patrulje.
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

### BFF (Go)

- **New saga/consumer** (e.g. `go/nathejk/table/patruljenumber/` or a saga under
  `patrulje/`), modelled on the existing order Pay saga
  (`go/nathejk/table/order/saga.go`) and implementing:
  - `Consumes()` → the **triggering** subject plus its **own** event subject
    (`NATHEJK.*.patrulje.*.numberassigned`) so it can rebuild state on replay.
  - `HandleMessage(msg)` → routes by subject.
  - `CaughtUp()` (`streaminterface.CatchupListener`) → flip an internal
    `live` flag; only publish when `live` is true.
- **State the saga maintains (in memory, rebuilt on replay):**
  - `assigned map[TeamID]bool` — populated from replayed `numberassigned`
    events (and never re-published).
  - `next int` — the next number to hand out, seeded from the highest number
    seen in replayed `numberassigned` events + 1 (and see legacy note below).
- **On a `numberassigned` message (live or replay):** mark the team assigned and
  bump `next` past its number. Publish nothing.
- **On a triggering message (only act/publish when `live`):**
  1. Resolve the affected patrulje `teamId`.
  2. If `assigned[teamId]` → return (idempotent).
  3. If the patrulje is eligible (≥3 paid members — see rule below) → publish
     `NathejkPatrolNumberAssigned{TeamID, TeamNumber: strconv.Itoa(next)}` on
     `NATHEJK.{year}.patrulje.{teamId}.numberassigned`, then optimistically
     mark `assigned[teamId]=true` and increment `next` (the saga's own
     subscription will also confirm it on the way back).
- **Triggering event — recommended `NATHEJK.*.order.*.paid`:** when an order is
  paid, if its owner is a patrulje, count the paid **participation** members and
  compare to 3. Note: `order.paid` is emitted by the order Pay saga, which is
  **not yet wired** (flagged during PRD 002 / task 002) — this PRD depends on
  that saga being active, or on choosing a different trigger (see Open
  Questions).
- **Eligibility rule — "paid for ≥3 members":** recommended definition is the
  count of paid **participation** order lines (product kind `participation`,
  distinct `memberId`) for the patrulje owner ≥ 3. Alternative: paid amount ≥
  3 × member price (250 DKK = 75000 øre), reusing the `paidAmount` logic already
  in `patrulje/query.go`. Pick one (Open Question).
- **Wiring:** construct the saga in `cmd/api/main.go`, add it to
  `mux.AddConsumer(...)`. The read/projection side already exists in
  `patrulje/consumer.go` — no change needed there.
- **Message/shared-go:** `NathejkPatrolNumberAssigned` already exists in the
  pinned shared-go; no shared-go change required.

### Data / storage

- No new tables required. `patrulje.teamNumber` is written by the existing
  projector. The saga's state is in-memory and rebuilt from the event log.

### Dependencies & risks

- **Depends on a triggering event that actually fires.** If `order.paid` isn't
  emitted (Pay saga unwired), nothing triggers. Resolve the trigger before/with
  this work.
- **Legacy/pre-existing `teamNumber`s.** Some patruljer already have numbers set
  via other paths (e.g. `klan/consumer.go` inserts patrulje rows with
  `teamNumber`). If those weren't issued via `numberassigned` events, the saga's
  `next` counter won't know about them and could collide. Mitigation: also seed
  `next`/`assigned` from the current `patrulje.teamNumber` values at `CaughtUp`,
  or ensure all historical numbers came through `numberassigned` (Open
  Question).
- **Live-only publish is essential.** Publishing during catch-up would spam
  duplicate `numberassigned` events on every restart. The `CaughtUp()` gate is
  the crux and must be verified.

## 9. Success Metrics

- Every patrulje that has paid for ≥3 members has a non-empty `teamNumber`.
- Numbers are unique and contiguous (1..N) per year with no duplicates or gaps
  attributable to the saga.
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

Proposed tasks to create in `roadmap/tasks/open/` (not created yet):

- [ ] Task: Decide the triggering event + eligibility rule (order.paid vs payment; participation-lines vs amount)
- [ ] Task: BFF — implement patrulje number-assignment saga (live-only, replay-safe; consumes trigger + own numberassigned)
- [ ] Task: BFF — seed `next`/`assigned` from history (+ legacy teamNumber) to avoid collisions
- [ ] Task: BFF — wire the saga into `cmd/api/main.go` mux
- [ ] Task: Verify restart/replay behaviour (no renumber, no duplicate emits)
- [ ] Task: (dependency) wire the order Pay saga so `order.paid` is emitted, if chosen as the trigger

## 11. Open Questions

- **Trigger:** use `order.paid` (requires the order Pay saga to be wired) or
  react directly to payment events (`payment.*.received`) and recompute paid
  members? `order.paid` is cleaner but currently unemitted.
- **Eligibility definition:** count paid **participation lines** (≥3 distinct
  members) or paid **amount** ≥ 3 × member price? These can differ if a patrulje
  pays a partial/lump amount.
- **Sequence seeding vs legacy numbers:** should `next` start at 1 on a clean
  system only, and otherwise continue from the max of historical
  `numberassigned` events **and** any pre-existing `patrulje.teamNumber`? Are all
  existing numbers guaranteed to have come through `numberassigned`?
- **Number scope:** is the sequence per **year** (reset each year) or global?
  ("1..200+" and "first team gets 1" implies per-year starting at 1.)
- **Loss of eligibility:** if an accepted patrulje later cancels/refunds below 3
  members, do they keep the number? (Assumed yes — Non-Goal.)
- **Format:** `TeamNumber` is a string in the message/column. Plain decimal
  ("1", "2", …) assumed; confirm no prefix/padding is expected.
