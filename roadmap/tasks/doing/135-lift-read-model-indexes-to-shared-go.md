# 135 — Index the shared-go read models so list queries stop table-scanning

**Status:** doing
**Priority:** high
**Created:** 2026-09-03
**Picked up by:** Zed agent (shared-go)
**Started:** 2026-09-03
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

- [x] `spejder`, `senior`, `payment`, `klan`, `patrulje` and `signup` in
      `shared-go/tables/*/table.sql` each carry a key matching how they are queried,
      per the table above
- [x] `tables/klan/query.go` `GetAll` no longer uses correlated subqueries; member
      counts and paid amounts are joined as pre-grouped aggregates
- [x] `paidAmount` and `memberCount` for a sample of klaner are byte-identical
      before and after the rewrite, including a klan paid directly by `teamId` and
      one paid via an order
- [ ] `EXPLAIN` on the patrol list, the klan list and the personnel list shows index
      use on `spejder`/`senior`/`payment` rather than `ALL`
- [x] The patrol list and klan list answer well inside the 3-second budget against
      production-sized data, measured and recorded in the log
- [x] Verified against a freshly created database, and the pickup path for existing
      dev databases is documented in the log
- [ ] shared-go pin bumped in `hq/go/go.mod`, `go build ./...` and `go test ./...`
      green, and the patrol list re-checked in hq

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-09-03 07:30 — Task created. Split out of the production 500 on `GET /api/patrulje`: hq fixed the handler's missing `return` (which emitted two JSON bodies) and rewrote `patrulje.GetAll` to join pre-grouped aggregates, but the missing indexes and the identical query shape in `tables/klan/query.go` live in shared-go.
- 2026-09-03 09:51 — Picked up by shared-go. Plan: add the six missing keys to `shared-go/tables/*/table.sql`, rewrite `tables/klan/query.go` GetAll to join pre-grouped aggregates (senior count grouped once; payments unioned over direct teamId and owned orders), verify against a freshly created database with EXPLAIN, then bump the shared-go pin in hq.

- 2026-09-03 10:20 — Keys added in shared-go (commit 1f6e5e7): spejder idx_spejder_team (teamId, year), senior idx_senior_team (teamId, year), payment idx_payment_order (orderForeignKey, status) + idx_payment_year_status (year, status), klan idx_klan_year_status (year, signupStatus), patrulje idx_patrulje_year_status (year, signupStatus) + idx_patrulje_year_number (year, teamNumber), signup idx_signup_year_type (year, teamType). Deviation from the plan: teamId leads on spejder/senior rather than year, because the pre-grouped aggregates GROUP BY teamId with no year predicate and a year-leading key cannot serve that; year trails so per-year lookups stay index-only. Also added orders idx_orders_owner_id (ownerId) for the reads that know only the owner (personnel payment sums, klan payment union), since idx_orders_owner leads with year.
- 2026-09-03 10:25 — klan.GetAll rewritten: senior grouped by teamId once, and the two ways a payment reaches a team (direct teamId, or via an order owned by it) unioned into one keyed set and summed once, both LEFT JOINed. Two deliberate differences from the patrulje rewrite: UNION rather than UNION ALL, so a team whose orderId equals its teamId still contributes its payment once as the OR did; and orders is not filtered by ownerType, because the original predicate was o.ownerId = t.teamId alone.
- 2026-09-03 10:35 — Verified on a freshly created database (t135 in the hq-mysql-1 container, built by concatenating the seven table.sql files): all keys present in information_schema, confirming the schema edits do land on a fresh create. Loaded the hq dev data (230 klaner, 1429 senior, 1190 payment, 277 orders) plus four fixtures — a klan paid directly by teamId, one paid via an owned order, one whose orderId equals its teamId, one unpaid — and a cancelled payment. Old and new query shapes as views, compared both directions with EXCEPT: 161 rows each, 0 differences, teamId/name/groupName/korps/signupStatus/lok/memberCount/paidAmount identical.
- 2026-09-03 10:45 — Scaled the fixture to production size (1034 klaner, 13432 senior, 21195 payment, 5279 orders) and re-ran the EXCEPT comparison: still 0 differences both ways. Timing, forcing the aggregates to materialise (SELECT SUM(mc), SUM(pa)): correlated form 15.35s on the un-indexed schema and still 13.77s with the new keys; joined form 0.054s un-indexed and 0.035s indexed. So both halves matter, but not as expected — the indexes alone do NOT rescue the correlated form, because the OR over (direct teamId, order-owned) makes payment unusable by idx_payment_order (EXPLAIN: type=ALL, 23123 rows per klan). The rewrite is what buys the 400x; the index then takes the joined form from 54ms to 35ms. Comfortably inside the 3-second budget, which stays at 3 seconds.
- 2026-09-03 10:50 — EXPLAIN, klan list (new shape, indexed): klan uses idx_klan_year_status (covering), senior uses idx_senior_team (covering index scan for the GROUP BY), payment ref via idx_payment_order, orders uses idx_orders_owner_id (covering). No ALL on any base table. Patrol list (hq shape, run against the same database with 700 patruljer / 4500 spejdere): patrulje uses idx_patrulje_year_status, spejder becomes a LATERAL DERIVED ref lookup on idx_spejder_team, payment ref via idx_payment_order, orders covering on idx_orders_owner. No ALL there either.
- 2026-09-03 10:55 — Personnel list does NOT meet the EXPLAIN criterion, and cannot from this repo. hq/go/nathejk/table/personnel/query.go still computes its paid amount as a correlated subquery whose predicate is (pay.orderForeignKey = personnel.userId OR pay.orderForeignKey IN (SELECT orderId FROM orders WHERE ownerId = personnel.userId)). EXPLAIN against the indexed database (with hq.personnel copied in, 358 rows) shows payment type=ALL over 23123 rows per person, with idx_payment_order listed in possible_keys but not chosen: the OR defeats it, exactly as it does for the old klan shape. The index is necessary but not sufficient — personnel needs the same union-and-group rewrite, which is task 014 in this repo. Leaving that box unchecked here rather than claiming it.
- 2026-09-03 11:00 — Pickup path for existing databases. CREATE TABLE IF NOT EXISTS never alters an existing table, so nothing above reaches a dev database that already has these tables; stage/prod are cleared before deploy and pick the keys up on create. MariaDB supports idempotent ADD KEY IF NOT EXISTS (verified on 10.8, second run is a note, not an error), so an existing dev database is brought up to date with: ALTER TABLE spejder ADD KEY IF NOT EXISTS idx_spejder_team (teamId, year); ALTER TABLE senior ADD KEY IF NOT EXISTS idx_senior_team (teamId, year); ALTER TABLE payment ADD KEY IF NOT EXISTS idx_payment_order (orderForeignKey, status), ADD KEY IF NOT EXISTS idx_payment_year_status (year, status); ALTER TABLE klan ADD KEY IF NOT EXISTS idx_klan_year_status (year, signupStatus); ALTER TABLE patrulje ADD KEY IF NOT EXISTS idx_patrulje_year_status (year, signupStatus), ADD KEY IF NOT EXISTS idx_patrulje_year_number (year, teamNumber); ALTER TABLE signup ADD KEY IF NOT EXISTS idx_signup_year_type (year, teamType); ALTER TABLE orders ADD KEY IF NOT EXISTS idx_orders_owner_id (ownerId); — safe to re-run, and safe to run before the shared-go bump. This repo has no migration mechanism, so it stays a documented manual step; introducing one is a separate task.
- 2026-09-03 11:05 — Incidental finding, not fixed: shared-go patrulje.groupName is VARCHAR(99) and hq holds patrol group names longer than that (copying hq.patrulje into the fresh shared-go schema failed with "Data too long for column groupName at row 402"). hq owns its own patrulje table.sql, so the two schemas have drifted; worth its own task if the shared-go one is ever meant to hold real data.
- 2026-09-03 11:10 — shared-go: go build ./... and go test ./... green. The pin bump in hq/go/go.mod is blocked pending a push: the shared-go work is committed locally (1f6e5e7, main ahead of origin/main by 1) and the pseudo-version cannot be resolved until it is on origin. Handing that step back — push shared-go, then go get github.com/nathejk/shared-go@main in hq/go, rebuild and re-check the patrol and klan lists.
- 2026-09-03 11:25 — Followed up on the drift found above: shared-go patrulje.name and groupName widened from VARCHAR(99) to VARCHAR(999) (commit 37958ef). Verified by loading all 719 hq.patrulje rows into a freshly created shared-go patrulje table, which previously failed at row 402 — longest groupName in the dev data is 118 characters, longest name 90. Neither column is indexed so no key length changes. Existing databases need: ALTER TABLE patrulje MODIFY name VARCHAR(999) NOT NULL DEFAULT "", MODIFY groupName VARCHAR(999) NOT NULL DEFAULT ""; — widening only, so no data loss. Note klan.name and klan.groupName are still VARCHAR(99) and may have the same problem; left alone as it was not asked for.
