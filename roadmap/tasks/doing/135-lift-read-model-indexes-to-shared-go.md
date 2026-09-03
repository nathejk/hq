# 135 — Index the shared-go read models so list queries stop table-scanning

**Status:** open
**Priority:** high
**Created:** 2026-09-03
**Picked up by:**
**Started:**
**Completed:**

## Description

`GET /api/patrulje` began answering HTTP 500 in production. The cause was not the
handler: it was `context deadline exceeded` against the 3-second budget in
`patrulje.GetAll`, logged as

```
{"level":"ERROR","message":"context deadline exceeded",
 "properties":{"request_method":"GET","request_url":"/api/patrulje"}}
```

The patrol list computed three correlated subqueries **per patrol** over tables that
carry no index able to serve them. With a season's worth of rows that is
O(patruljer × rows) and it finally crossed three seconds. hq has since rewritten
`patrulje.GetAll` (`go/nathejk/table/patrulje/query.go`) to scan each table once and
join pre-grouped derived tables, which is the half of the fix this repo owns.

**The other half belongs in shared-go, and hq cannot fix it:** the schemas are
`shared-go/tables/*/table.sql`, and the identical query shape lives in
`shared-go/tables/klan/query.go`. Until the indexes exist, every rewrite here is
buying time on a full scan.

### Tables missing an index for how they are actually read

| Table | Current keys | Read by | Missing |
|---|---|---|---|
| `spejder` | PK `(year, memberId)` | `teamId` — patrol member count and t-shirt count | `KEY (teamId)`, or `(year, teamId)` |
| `senior` | PK `(year, memberId)` | `teamId` — klan member count | `KEY (teamId)`, or `(year, teamId)` |
| `payment` | PK `(reference)` | `orderForeignKey` + `status IN (…)` — every paid-amount sum | `KEY (orderForeignKey, status)`; consider `(year, status)` |
| `klan` | PK `(teamId)` | `year`, `signupStatus`, `lok` — the klan list | `KEY (year, signupStatus)` |
| `patrulje` | PK `(teamId)` | `year`, `signupStatus`, `teamNumber <> ''` — patrol list, started-teams, `AssignedNumbers` | `KEY (year, signupStatus)`, `KEY (year, teamNumber)` |
| `signup` | PK `(teamId)` | `(year, teamType)` — `internal/data/signup.go` | `KEY (year, teamType)` |

`orders` and `order_line` are already indexed (`idx_orders_owner`,
`idx_order_line_*`) and need nothing. Note `orders` is read by `ownerId` alone in
`personnel/query.go`, which cannot use `idx_orders_owner`'s leading columns — either
that query passes year and ownerType, or a narrower key is warranted.

### Queries to rewrite in shared-go

- `tables/klan/query.go` `GetAll` — worst of the set: a correlated subquery per klan
  that itself contains `LEFT JOIN orders`, plus a second correlated `COUNT(*)` over
  `senior`. Same fix as hq applied to `patrulje.GetAll`: group `senior` by `teamId`
  once, union the two ways a payment reaches a team (direct `teamId`, or via an order
  owned by it) into one keyed set, sum it once, and `LEFT JOIN` both.
- `tables/klan/query.go:59` — `SELECT COUNT(memberId) FROM senior WHERE year=?` is
  fine once `senior` is indexed on year (the PK already leads with it).

hq's `personnel/query.go` carries the same correlated payment sum against
`payment`/`orders`. It stays here for now, but the `payment` index is what makes it
cheap; re-measure it after the index lands (see also task 014).

### Important: an index added to `table.sql` does not appear on an existing table

`CREATE TABLE IF NOT EXISTS` never alters a table that already exists. Adding a `KEY`
to a `table.sql` therefore has **no effect** anywhere the table is already present —
harmless in stage/prod, where the database is cleared before deploy, but silent in
every existing dev database. So the schema edit alone is not verification: confirm
against a freshly created database, and state in the task log how existing
environments are expected to pick the indexes up.

### Do not simply raise the timeout

The 3-second budget in `GetAll` is what turned a slow query into a visible error
instead of a hung page, and it is also what once made the `patruljenumber` saga
dormant under replay load (see the comment on `AssignedNumbers`). It is the alarm,
not the fault.

## Acceptance Criteria

- [ ] `spejder`, `senior`, `payment`, `klan`, `patrulje` and `signup` in
      `shared-go/tables/*/table.sql` each carry a key matching how they are queried,
      per the table above
- [ ] `tables/klan/query.go` `GetAll` no longer uses correlated subqueries; member
      counts and paid amounts are joined as pre-grouped aggregates
- [ ] `paidAmount` and `memberCount` for a sample of klaner are byte-identical
      before and after the rewrite, including a klan paid directly by `teamId` and
      one paid via an order
- [ ] `EXPLAIN` on the patrol list, the klan list and the personnel list shows index
      use on `spejder`/`senior`/`payment` rather than `ALL`
- [ ] The patrol list and klan list answer well inside the 3-second budget against
      production-sized data, measured and recorded in the log
- [ ] Verified against a freshly created database, and the pickup path for existing
      dev databases is documented in the log
- [ ] shared-go pin bumped in `hq/go/go.mod`, `go build ./...` and `go test ./...`
      green, and the patrol list re-checked in hq

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-09-03 07:30 — Task created. Split out of the production 500 on `GET /api/patrulje`: hq fixed the handler's missing `return` (which emitted two JSON bodies) and rewrote `patrulje.GetAll` to join pre-grouped aggregates, but the missing indexes and the identical query shape in `tables/klan/query.go` live in shared-go.
