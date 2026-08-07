# 021 — Adopt shared-go tables entities

**Status:** done
**Priority:** high
**Created:** 2026-08-07
**Picked up by:** agent (opus-4.8 session)
**Started:** 2026-08-07
**Completed:** 2026-08-07

## Description

`github.com/nathejk/shared-go` now ships entity packages under `tables/`
(crewmember, klan, order, patrulje, payment, product, section, senior, signup,
spejder + shared `errors.go`/`interfaces.go`). These are the upstream versions of
packages we duplicate in `go/nathejk/table/`. Switch to the shared ones where
they are drop-in compatible.

Local-only (no shared counterpart, stay put): `checkgroup`, `checkpersonnel`,
`checkpoint`, `lok`, `patruljemerged`, `personnel`, `scan`, `year`.

### Divergence found (both directions)

Shared is **ahead** of local:
- `patrulje` — `paidAmount` resolved via `LEFT JOIN orders` (`orderForeignKey =
  teamId OR orders.ownerId = teamId`), i.e. upstream converged on the task-005 fix.
- `payment` — same order-aware resolution in its queries.
- `spejder` — `initialTeamId` **and** `currentTeamId` json tags (task 009 only
  renamed the duplicate; upstream also renamed the first).
- Generally: `*sql.DB` → `cqrs.Reader`, and errors come from
  `shared-go/tables` instead of the local `nathejk/table`.

Shared is **behind** local:
- `order` — **missing** `ListByYear`, `linesForYear` and the
  `NOT (status='open' AND totalAmount=0)` filter added in tasks 001/013/017,
  which the Ordrehistorik page + its line-item summary depend on. The querier
  type is unexported upstream, so these cannot be added from hq — they must be
  upstreamed to shared-go before `order` can switch.

## Outcome

**Adopted (7)** — local duplicates deleted, only import paths changed (plus two
small call-site tweaks):

| package | notes |
|---|---|
| `product` | drop-in, incl. `seeder`/`Seeds2026` |
| `spejder` | drop-in. Shared renames `teamId`→`initialTeamId`; safe because before task 009 both fields shared the `teamId` tag so Go emitted *neither* — nothing could depend on it |
| `senior` | querier methods identical to `data.SeniorInterface` |
| `crewmember` | shared Commands is a superset (adds `Update`); Queries identical |
| `section` | Commands **and** Queries identical |
| `klan` | strict superset (querier adds `RequestedMemberCount`/`RequestedSeniorCount`; Commands adds member ops). `Klan` struct is a JSON superset (adds `requestedMemberCount`, `reservedMemberCount`) so no frontend regression. Needed `klan.Filter.YearSlug` to be a `string` — 2 call sites wrapped in `string(...)` |
| `signup` | Queries identical; shared `New` takes a publisher first, so main.go now passes `publisher` |

**Deferred (3)** — each would regress functionality; all need upstreaming first:

- **`order`** — shared lacks `ListByYear`, `linesForYear` and the
  `NOT (status='open' AND totalAmount=0)` filter (tasks 001/013/017) that the
  Ordrehistorik list + line-item summary depend on. The `querier` is unexported
  upstream so these can't be added from hq.
- **`patrulje`** — shared lacks `GetStartedTeamIDs`, `GetContact` and
  `GetDiscontinuedTeamIDs`, all used by `data.Models`/handlers. (Shared does add
  `GetLastWithNumber` + `Commands.AssignNumber`, which are exactly what PRD 003
  needs — worth revisiting together.)
- **`payment`** — shared read API is different/older: `Query.GetAll(teamID)` with
  no ctx and no `Filter` type at all (returns `Metadata`), so it cannot express
  the year-wide `/api/payments` listing or the `Filter{TeamIDs}` usage.

Local-only, no shared counterpart (unchanged): `checkgroup`, `checkpersonnel`,
`checkpoint`, `lok`, `patruljemerged`, `personnel`, `scan`, `year`.

## Acceptance Criteria

- [x] Drop-in-compatible shared entities adopted; local duplicates deleted (7).
- [x] No functional regression — Ordrehistorik/order queries untouched by keeping
      `order` local; klan/spejder JSON shapes verified as supersets.
- [x] `order` explicitly deferred with reason (plus `patrulje`, `payment`).
- [x] `go build`/`go vet`/`go test`/`staticcheck`/`gofmt` all green.

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-07 10:00 — Task created + picked up. Confirmed the pinned shared-go version (v0.0.0-20260806204955) contains `tables/` and matches the sibling checkout exactly. Mapped divergence in both directions (above).
- 2026-08-07 10:15 — Compared exported surfaces per package rather than trusting the file-level diff (every file "differs" purely from import paths). Confirmed `cqrs.Publisher` ≡ `stream.Publisher` and `*sql.DB` satisfies `cqrs.Reader`, so most constructors are call-compatible as-is.
- 2026-08-07 10:25 — Migrated `product` + `spejder`; build clean with only import-path changes.
- 2026-08-07 10:35 — Migrated `senior`, `crewmember`, `section` (verified method sets identical / superset); build clean.
- 2026-08-07 10:45 — Migrated `klan` + `signup`. Fixed `klan.Filter.YearSlug` being `string` (2 call sites) and added `publisher` to `signup.New`. Build clean.
- 2026-08-07 10:55 — Assessed and DEFERRED `payment` (shared read API lacks ctx/Filter and is teamID-only — would regress `/api/payments` and the by-team query), `patrulje` (missing 3 querier methods in use) and `order` (missing this session's additions).
- 2026-08-07 11:00 — goimports/gofmt, then build, vet, test and staticcheck all clean. 7 duplicate packages removed from `nathejk/table`. Completed.
