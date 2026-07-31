# PRD 002 — Order-based payments in Betalinger & Patrulje views

**Status:** draft
**Author:** agent session
**Created:** 2026-07-31
**Last updated:** 2026-07-31
**Target users:** organizer (HQ finance/admin)

---

## 1. Summary

Payments recently gained an **order** layer: a payment is no longer tied
directly to a team but to an *order*, which in turn belongs to an owner (a team
or a personnel user). Update the HQ admin so the **Betalinger** page and the
payments section on the **Patrulje** page show **orders and their payment
status** rather than raw individual payments.

## 2. Problem & Motivation

- **What problem does this solve?** With the new order layer, the current UIs
  are wrong/misleading:
  - The **Betalinger** page (`vue/src/views/PaymentListView.vue`) lists
    individual payment transactions, not the orders they pay for. Operators
    can't see, per order, whether it is fully paid, partly paid, open, or
    cancelled.
  - The **Patrulje** page (`vue/src/views/PatruljeView.vue`) shows a
    "Betalinger" table of raw payments for the team.
  - Under the hood, both rely on `payment.orderForeignKey` being the **team
    id** (see `go/nathejk/table/payment/queries.go`: `where["orderForeignKey"]
    = f.TeamIDs`, `AmountPaidByTeamID`, and the `paidAmount` subquery in
    `go/nathejk/table/patrulje/query.go`). With the order layer,
    `orderForeignKey` now references an **order id**, so the team↔payment
    linkage must go through orders instead. Leaving it as-is will show wrong or
    empty payment data.
- **Why now?** The order layer is now implemented on the write/read side
  (`go/nathejk/table/order` and `go/nathejk/table/product`), but it is **not yet
  wired into the API** and the UI still shows raw payments — so the displays are
  drifting from reality.
- **Evidence.**
  - Order events defined in `github.com/nathejk/shared-go/messages/order.go`:
    `NathejkOrderCreated` (OrderID, Year, OwnerType, OwnerID, Currency),
    `NathejkOrderLinesChanged` (snapshot of lines + TotalAmount),
    `NathejkOrderPaid` (OrderID, PaidAmount), `NathejkOrderCancelled`.
  - Payment events reference an order via `orderForeignKey` / `orderType`
    (`shared-go/messages/payment.go`).
  - **The order + product read models now exist** in
    `go/nathejk/table/order/` (tables `orders`, `order_line`; projector
    consuming `NATHEJK.*.order.*.{created,lines.changed,cancelled,paid}`;
    command layer + Pay saga; read API `GetByID` / `FindOpenOrder` /
    `ListByOwner`) and `go/nathejk/table/product/` (table `product`, seeded via
    `seeds_2026.go`). They are **not** yet added to `mux.AddConsumer(...)` /
    `data.NewModels(...)` in `go/cmd/api/main.go`, and there are no order HTTP
    endpoints.

## 3. Goals

- The Betalinger page lists **orders** with their payment status (open / paid /
  cancelled, plus total vs. paid amount), for the current year.
- The Patrulje page shows the **team's orders** and their payment status instead
  of raw payments.
- The team↔payment relationship is computed correctly **through orders**
  (order owner = team; payments reference the order), fixing the current
  `orderForeignKey == teamId` assumption.
- Operators can still drill into an order to see its underlying payment
  transactions and line items.

## 4. Non-Goals

- Creating, editing, or cancelling orders from these views. This is a
  read/display change. The order command layer and Pay saga already live in
  `go/nathejk/table/order/` (driven by team/personnel handlers); order
  mutations are out of scope for this PRD.
- Initiating or refunding payments from these views.
- Changing the payment provider integration (MobilePay) or payment events.
- Personnel/badut order display — only if it falls out for free; the explicit
  target is team (patrulje/klan) orders (see Open Questions).
- Reworking the shared-go order/payment event schema (it already exists).

## 5. User Stories & Scenarios

- As an **organizer**, I want the Betalinger page to list orders with a clear
  paid/unpaid status so I can see who still owes money.
- As an **organizer**, I want to open an order and see its line items and the
  payment(s) applied to it, so I can reconcile a specific case.
- As an **organizer viewing a patrol**, I want to see that patrol's orders and
  whether they are paid, rather than a flat list of payment transactions.

### Primary happy path

1. Operator opens **Betalinger**. They see one row per order: owner (team),
   order total, amount paid, and a status tag (Åben / Betalt / Annulleret).
2. They filter/scan for unpaid orders.
3. They expand an order to see its lines (participation, t-shirts, …) and the
   payment transactions applied to it (reserved/received/…).
4. On a patrol's page, the "Betalinger" section lists that patrol's orders with
   the same status semantics.

### Edge cases

- Order with no payments yet → status **Åben** (`OPEN`), paid amount 0.
- Order partly covered by reserved (not yet received) payments → `paidAmount`
  already sums both `reserved` and `received` payments; the order stays `OPEN`
  until the Pay saga emits `order.paid`. A "delvist betalt" hint can be derived
  from `dueAmount > 0 && paidAmount > 0`.
- Cancelled order → on the wire this currently **collapses to `PAID`**
  (`Status.MarshalJSON`); see Open Questions if it must be shown as Annulleret.
- Payment referencing an unknown/old-style `orderForeignKey` (legacy team-keyed
  data) → the paid-amount subquery simply sums nothing; must not crash the view.
- Team with multiple orders (e.g. re-ordered after changes) → all shown
  (`ListByOwner` returns every order for the owner, newest first).

## 6. Requirements

### Functional

- [ ] Add a **list-all-orders-for-year** query to the order read API (the
      existing `Queries` has `GetByID` / `FindOpenOrder` / `ListByOwner` /
      `ReservedQuantity` / `PaidLineKeys` but no year-wide list).
- [ ] Wire the `order` (and `product`) projections into
      `go/cmd/api/main.go` (`mux.AddConsumer`, `data.NewModels`).
- [ ] `GET /api/orders` returns orders for the current year, each with owner,
      currency, total, paid amount, due amount, and status. The owner is
      resolved to a human-readable label by a **secondary lookup** against the
      patrulje/klan/personnel read models (grouped by `ownerType`, batched by
      `ownerId`), not a SQL join in the orders query.
- [ ] `GET /api/order/:id` returns a single order with its line items (reuse the
      existing `GetByID`, which already hydrates lines + paid/due amounts).
- [ ] Betalinger page (`PaymentListView.vue`) shows orders (+ status), with an
      expandable detail of lines (and, if desired, the underlying payments).
- [ ] Patrulje page (`PatruljeView.vue`) "Betalinger" section shows the team's
      orders (+ status) instead of raw payments (via `ListByOwner`).
- [ ] The patrol list `paidAmount` / paid-status computation is corrected to go
      through orders rather than `payment.orderForeignKey == teamId`.

### Non-Functional

- **Consistency:** REST + `app.*` helpers, MySQL projection rebuilt from
  JetStream, PrimeVue Lara + `fetchWrapper`, `da-DK` formatting and Danish
  labels — matching the rest of the app.
- **OpenAPI:** the new `/api/orders` and `/api/order/:id` endpoints must carry
  OpenAPI annotations (repo `.rules`).
- **No regressions** to the existing `/api/payments` endpoint if it is still
  used elsewhere; prefer additive endpoints.

## 7. UX / UI Notes

- **Betalinger — `vue/src/views/PaymentListView.vue`:** switch the primary
  `DataTable` from payments to **orders**. Suggested columns: Tidspunkt
  (created), Ejer (owner name/number, resolved by a secondary lookup on
  `ownerId`), Beløb (total), Betalt (paid amount), Mangler (due), Status tag
  (Åben/Betalt). Keep an expander row that shows:
  - order **lines** (product name, member, qty, unit price, line total), and
  - the **payment transactions** applied to the order (the current payment
    fields: method, amount, status, operations timeline).
  This reuses much of the existing expansion markup, moved down a level.
- **Patrulje — `vue/src/views/PatruljeView.vue`:** the "Betalinger"
  `DataTable` (currently `:value="payments"`) becomes `:value="orders"` with
  columns Tidspunkt, Beløb (total), Betalt, Status. Optionally expandable to the
  same order detail.
- Fetching: continue using `@/helpers`/`http`; the patrol view already loads its
  payload from `/patrulje/:id`, so the handler should return `orders` alongside
  (or instead of) `payments`.
- Status labels: the order read model exposes a **binary** status on the wire —
  `Status.MarshalJSON` renders `OPEN` or `PAID`, and cancelled collapses to
  `PAID`. The UI shows two states — **Åben** and **Betalt** — which is
  sufficient (no separate Annulleret state for now). Amounts to show:
  `totalAmount`, `paidAmount`, `dueAmount` (all already on the `Order` struct,
  in minor units — divide by 100 for `da-DK` currency formatting).

## 8. Technical Considerations

### BFF (Go)

- **The `order` + `product` read models already exist** and largely cover the
  read side:
  - `go/nathejk/table/order/` projects `orders` + `order_line` from
    `NATHEJK.*.order.*.{created,lines.changed,cancelled,paid}` (see
    `consumer.go`). `orders` holds `orderId`, `year`, `ownerType`, `ownerId`,
    `status` (open/paid/cancelled), `currency`, `totalAmount`, `cancelReason`,
    timestamps; `order_line` holds the line snapshot.
  - The `Order` read struct already computes `PaidAmount` (subquery summing
    `payment` rows where `orderForeignKey = orderId` and status in
    `reserved`/`received`) and `DueAmount`, and clamps them on terminal `paid`
    orders. `Status.MarshalJSON` emits binary `OPEN`/`PAID`.
  - `product` (`go/nathejk/table/product/`) holds the catalogue (seeded via
    `seeds_2026.go`) used by the order command layer.
- **Remaining backend work:**
  - **Add a year-wide list query** to `order.Queries` (e.g. `ListByYear(ctx,
    year)` or `GetAll(ctx, Filter{Year, OwnerType, OwnerID})`). The current API
    only has `GetByID`, `FindOpenOrder`, `ListByOwner`, `ReservedQuantity`,
    `PaidLineKeys` — none list all orders for a year for the Betalinger page.
  - **Wire the projections into `go/cmd/api/main.go`:** construct
    `product.New(...)` and `order.New(p, writer, db, year, products)`, add both
    to `mux.AddConsumer(...)`, and expose the order read API through
    `data.NewModels(...)` (e.g. `app.models.Order`). (Neither is wired yet.)
  - **Endpoints & handlers** (new `go/cmd/api/order.go`):
    - `listOrdersHandler` (`GET /api/orders`) — orders for `app.YearSlug(r)`.
      Enrich each row with an owner label via a **secondary lookup** in the
      handler: group the returned orders by `ownerType`, then batch-fetch names
      from the corresponding read model (patrulje/klan/personnel/…) through
      `app.models`. Keep the `order` querier free of cross-table joins on
      `ownerId`.
    - `showOrderHandler` (`GET /api/order/:id`) — via the existing `GetByID`
      (already returns lines + paid/due).
    - Update `showPatruljeHandler` to include the team's orders via
      `ListByOwner(year, "patrulje", teamId)` in the JSON envelope (and
      `showKlanHandler` if klan orders should show).
  - **Fix** `go/nathejk/table/patrulje/query.go` `GetAll`: the `paidAmount`
    subquery currently joins `payment.orderForeignKey = p.teamId`; change it to
    resolve payments through orders (`orders.ownerId = p.teamId` →
    `payment.orderForeignKey = orders.orderId`), or reuse the order read model,
    so the list's derived signup/paid status stays correct.
  - Keep `/api/payments` as-is (or repoint it) depending on whether anything
    still needs the flat payment list.
- **All new endpoints require OpenAPI annotations** (`.rules`).

### Frontend (Vue)

- `PaymentListView.vue`: fetch `/api/orders`; restructure the table to orders
  with an order-detail expansion (lines + payments). Extend the status-severity
  mapping for order statuses.
- `PatruljeView.vue`: consume `orders` from the `/patrulje/:id` payload; update
  the Betalinger `DataTable`.

### Data / storage

- `orders`, `order_line` (`go/nathejk/table/order/table.sql`) and `product`
  (`go/nathejk/table/product/table.sql`) tables already exist, `CREATE TABLE IF
  NOT EXISTS`, year-scoped, rebuilt from JetStream on startup. No new tables are
  needed for display; the only schema-adjacent work is the new year-wide list
  query.

### Dependencies & risks

- **Order projector location — resolved:** the projector runs in **this** repo
  (`order/consumer.go` consuming `NATHEJK.*.order.*`), so HQ owns the order read
  model. It just needs wiring into the API mux.
- **Mixed/legacy data:** existing `payment.orderForeignKey` values may be a mix
  of team ids (old model) and order ids (new). Orders whose id never matches a
  payment row simply show `paidAmount = 0`; confirm whether a migration/reset is
  needed or whether both must be tolerated during transition.
- **shared-go version pin:** the order/payment messages must be present in the
  shared-go version pinned in `go.mod` for prod/CI (`GOWORK=off`), not only in
  the dev workspace checkout.

## 9. Success Metrics

- Betalinger and Patrulje pages show order-level paid/unpaid status that matches
  the actual order state (spot-checked against a few known orders).
- Patrol list paid-status/`paidAmount` matches order-derived totals (no more
  team-keyed payment mismatch).
- No runtime errors from orphan/legacy `orderForeignKey` values.

## 10. Rollout / Task Breakdown

- **Phase 1 — Wire up + list query:** add the year-wide order list query, wire
  `order` + `product` into `main.go`/`data.Models`, add `GET /api/orders` +
  `GET /api/order/:id`.
- **Phase 2 — Betalinger UI:** switch `PaymentListView` to orders with detail
  expansion.
- **Phase 3 — Patrulje UI + list fix:** team orders in `/patrulje/:id`, update
  the Patrulje Betalinger table, and correct the patrol-list `paidAmount`
  computation.

Proposed tasks to create in `roadmap/tasks/open/` (not created yet):

- [ ] Task: BFF — add year-wide order list query to `order.Queries`
- [ ] Task: BFF — wire `order` + `product` into `main.go` mux & `data.Models`
- [ ] Task: BFF — `GET /api/orders` & `GET /api/order/:id` (OpenAPI)
- [ ] Task: BFF — include team orders in `showPatruljeHandler` (+ klan?)
- [ ] Task: BFF — fix patrulje list `paidAmount` to resolve payments via orders
- [ ] Task: Frontend — `PaymentListView` order-centric table + detail expansion
- [ ] Task: Frontend — `PatruljeView` Betalinger shows orders
- [ ] Task: Confirm shared-go version pin includes the order/payment messages

## 11. Open Questions

- **Owner scope:** Betalinger — team (patrulje/klan) orders only, or also
  personnel/other `OwnerType` orders? (`orders.ownerType` supports both.)
- **Legacy data:** are there existing payments keyed by team id that must keep
  working, or will data be reset so `orderForeignKey` is always an order id?
- **Keep raw payments view?** Should the flat payment transaction list (and
  `/api/payments`) remain available anywhere, or be fully superseded by orders?
  If kept, should the order detail expansion also list the underlying payments?
- **Naming:** keep the page label "Betalinger" and route `/betalinger`, or
  rename to "Bestillinger"/orders to match the new model?
