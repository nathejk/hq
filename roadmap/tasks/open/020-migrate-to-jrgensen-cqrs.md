# 020 — Migrate sqlpersister/tablerow → github.com/jrgensen/cqrs

**Status:** open
**Priority:** high
**Created:** 2026-07-31
**Picked up by:**
**Started:**
**Completed:**

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

## Acceptance Criteria

- [ ] `github.com/jrgensen/cqrs` added to go.mod; nothing imports
      `nathejk.dk/pkg/sqlpersister` or `nathejk.dk/pkg/tablerow`.
- [ ] `go/pkg/sqlpersister` and `go/pkg/tablerow` removed (or reduced to a
      documented remainder cqrs does not cover).
- [ ] Locally-written `EnsureColumn`/`EnsureIndex` replaced by the cqrs versions.
- [ ] `go build ./...`, `go vet ./...`, `go test ./...`,
      `go tool staticcheck ./...` all green.

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-07-31 18:45 — Task created as stage 2, queued after task 019 (stream migration) completed.
