# 020 — Migrate sqlpersister/tablerow → github.com/jrgensen/cqrs

**Status:** done
**Priority:** high
**Created:** 2026-07-31
**Picked up by:** agent (opus-4.8 session)
**Started:** 2026-07-31
**Completed:** 2026-07-31

## Description

Stage 2 of the infrastructure migration (stage 1 = task 019, superfluids →
`github.com/jrgensen/stream`). Replace the internal `nathejk.dk/pkg/sqlpersister`
and `nathejk.dk/pkg/tablerow` with `github.com/jrgensen/cqrs` (v0.1.0, already in
the module cache).

Scope: ~45 Go files reference `sqlpersister` or `pkg/tablerow`.

### What cqrs offers (from a first inspection)

```
cqrs.go            → Publisher, Consumer, Writer, Reader interfaces;
                     SubjectFromStr(s) convenience (= subject.FromStr)
migrate.go         → EnsureColumn(r Reader, w Writer, table, column, ddl)
                     EnsureIndex(r Reader, w Writer, table, index, ddl)
sqlpersister/      → writer.go (replacement for pkg/sqlpersister)
deadletter/        → deadletter + dialect
cqrstest/          → test helpers
```

Notable: `cqrs.EnsureColumn` / `cqrs.EnsureIndex` are the upstream equivalents of
the helpers hand-written in `go/pkg/tablerow/migrate.go` during task 002 — those
can be deleted in favour of the library.

### Mapping to work out during implementation

- `tablerow.Consumer` (`Dialect() string`, `Consume(string) error`) → `cqrs.Writer`
  (confirm the method set matches, and how the dialect is handled).
- `sqlpersister.New(db, dialect)` → `cqrs/sqlpersister` writer constructor.
- `tablerow.SQLTableRow` / `SQLTableCreator` / `EntityChangedPublisher` → find
  cqrs equivalents or keep locally if there is none.
- `tablerow.EnsureColumn`/`EnsureIndex` → `cqrs.EnsureColumn`/`EnsureIndex`
  (note the arg order/types: cqrs takes a `Reader` + `Writer`).
- Decide whether `pkg/tablerow` and `pkg/sqlpersister` are deleted entirely or
  reduced to whatever cqrs does not cover.
- Consider adopting `deadletter` for failed projections (optional, separate).

### Mapping as implemented

| local | cqrs |
|---|---|
| `tablerow.Consumer` (`Dialect()`, `Consume(string)`) | `cqrs.Writer` (`Consume(string)` only) |
| `nathejk.dk/pkg/tablerow` | `github.com/jrgensen/cqrs` |
| `tablerow.EnsureColumn/EnsureIndex` | `cqrs.EnsureColumn/EnsureIndex` (identical signature) |
| `sqlpersister.New(db, dialect)` | `cqrs/sqlpersister.New(db)` |
| `SQLTableRow`, `SQLPrimaryKeys`, `SQLTableCreator`, `EntityChangedPublisher` | none needed — all were unused, deleted |

The dropped `Dialect()` method was safe: no consumer ever called it (only
`main.go` passed `db.Dialect()` into the local constructor; `database.go` has its
own unrelated `Dialect()` used for `sql.Open`).

## Acceptance Criteria

- [x] `github.com/jrgensen/cqrs` v0.1.0 added to go.mod (direct); nothing imports
      `nathejk.dk/pkg/sqlpersister` or `nathejk.dk/pkg/tablerow`.
- [x] `go/pkg/sqlpersister` and `go/pkg/tablerow` removed — nothing remained, so
      `go/pkg/` is gone entirely.
- [x] Locally-written `EnsureColumn`/`EnsureIndex` replaced by the cqrs versions.
- [x] `go build ./...`, `go vet ./...`, `go test ./...`,
      `go tool staticcheck ./...` all green (staticcheck 0 findings).

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-07-31 18:45 — Task created as stage 2, queued after task 019 (stream migration) completed.
- 2026-07-31 18:55 — Picked up. Inspected cqrs: `Writer` is just `Consume(statement) error`; `Reader` is the `*sql.DB` method set (so `*sql.DB` satisfies it directly); `Publisher`/`Consumer`/`Message`/`Subject` are aliases of the stream types (declared interchangeable in both directions); `EnsureColumn/EnsureIndex` have the same signature as the ones written in task 002.
- 2026-07-31 19:00 — Inventory: `tablerow.Consumer` used in 50 places; `EnsureColumn` 2, `EnsureIndex` 1; and `SQLTableRow`/`SQLPrimaryKeys`/`SQLTableCreator`/`EntityChangedPublisher` had **zero** external references. Confirmed no consumer calls `.Dialect()`, so narrowing to `cqrs.Writer` loses nothing.
- 2026-07-31 19:10 — Rewrote all call sites (`tablerow.Consumer` → `cqrs.Writer`, `tablerow.Ensure*` → `cqrs.Ensure*`, import path remapped), swapped `main.go` to `cqrs/sqlpersister.New(db.DB())`, deleted `pkg/tablerow` + `pkg/sqlpersister` (and the then-empty `pkg/`), ran goimports + `go mod tidy`.
- 2026-07-31 19:15 — build, vet, test, gofmt and staticcheck all clean. Both jrgensen modules are now direct deps. Completed — the two-stage migration off the vendored infrastructure is done.
