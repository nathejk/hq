# 019 — Migrate superfluids → github.com/jrgensen/stream

**Status:** done
**Priority:** high
**Created:** 2026-07-31
**Picked up by:** agent (opus-4.8 session)
**Started:** 2026-07-31
**Completed:** 2026-07-31

## Description

Stage 1 of a two-stage migration away from the vendored infrastructure:
replace the internal `nathejk.dk/superfluids/*` packages with the external
`github.com/jrgensen/stream`. (Stage 2, task 020, replaces
`pkg/sqlpersister` + `pkg/tablerow` with `github.com/jrgensen/cqrs`.)

Scope: 58 Go files import `superfluids`; 39 of them call
`streaminterface.SubjectFromStr` / `SubjectFromParts`.

### API mapping (verified against stream@v0.1.2)

The root `stream` package exposes an identical interface set to
`streaminterface`: `Message`, `MutableMessage`, `MessageFunc`,
`MessageHandler(Func)`, `Publisher`, `PublisherFunc`, `Subscriber`,
`Subscription`, `Stream`, `Consumer`, `Producer`, `CatchupListener`, `Subject`.

| superfluids | jrgensen/stream |
|---|---|
| `nathejk.dk/superfluids/streaminterface` | `github.com/jrgensen/stream` (pkg `stream`) |
| `streaminterface.X` | `stream.X` |
| `streaminterface.SubjectFromStr(s)` | `subject.FromStr(s)` (`.../stream/subject`) |
| `streaminterface.SubjectFromParts(d,t)` | `subject.FromParts(d,t)` |
| `nathejk.dk/superfluids/jetstream` | `github.com/jrgensen/stream/jetstream` |
| `nathejk.dk/superfluids/xstream` | `github.com/jrgensen/stream/xstream` |
| `jetstream.New(url, meta)` | `jetstream.New(url)` + `metatagger.New(pub, defaults)` |

`StringSubject` is not referenced outside `superfluids`, so no mapping needed.
`xstream.NewMux/AddConsumer/Run` and `MuxBlockUntilLive` are unchanged.

### Notable behaviour change

superfluids' `jetstream.New(url, meta)` applied `meta` as *default* metadata on
every published message (`m.SetDefaultMeta(s.meta)`). jrgensen's `jetstream.New`
takes only a URL; the equivalent is wrapping the publisher in
`metatagger.New(js, defaults)`. main.go passes
`{"producer":"hq-api","version":"1234"}`, so the metatagger must be used for
publishing to preserve behaviour, while the raw stream is still used for
subscribing (mux).

## Acceptance Criteria

- [x] `github.com/jrgensen/stream` v0.1.2 added to go.mod (direct); nothing imports `superfluids`.
- [x] `go/superfluids/` removed.
- [x] Default publish metadata (producer/version) preserved via `metatagger`.
- [x] `go build ./...`, `go vet ./...`, `go test ./...`, `go tool staticcheck ./...` all green (staticcheck 0 findings).

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-07-31 18:00 — Task created + picked up. Verified both target modules are already in the module cache (stream@v0.1.2, cqrs@v0.1.0) and mapped the API surface (above).
- 2026-07-31 18:10 — Added `github.com/jrgensen/stream@v0.1.2`. Needed Go cache/proxy write+network access (sumdb/module/build caches live outside the project); chose that over disabling checksum verification with GOSUMDB=off. Also registered `golang.org/x/tools/cmd/goimports` as a go.mod tool to make the 58-file import rewrite safe.
- 2026-07-31 18:20 — Mechanical rewrite over the 51 non-superfluids files: `streaminterface.SubjectFromStr/FromParts` → `subject.FromStr/FromParts`, `streaminterface.` → `stream.`, and the three import paths remapped. No aliased superfluids imports existed, and `StringSubject` was unused outside the package, so no special cases. `goimports -w` then added the `stream/subject` imports and dropped any newly-unused ones.
- 2026-07-31 18:30 — Deleted `go/superfluids/`. Fixed `main.go`: `jetstream.New(dsn)` (one arg) + `metatagger.New(js, {producer,version})`; passed the tagged `publisher` to every Publisher-taking constructor (year/klan/checkgroup/checkpoint/checkpersonnel/section/crewmember/order/commands) while the raw `js` stream still feeds `xstream.NewMux`.
- 2026-07-31 18:40 — Behaviour gap closed: `app.jetstream` turned out to be used *only* for publishing (routes.go ctrlgrp + signup.go), so those paths would have lost the default metadata. Replaced the struct field `jetstream stream.Stream` with `publisher stream.Publisher` (the metatagger) and repointed both call sites — fully behaviour-preserving.
- 2026-07-31 18:45 — `go mod tidy`; build, vet, test, gofmt and staticcheck all clean. 49 files now import `github.com/jrgensen/stream`. Completed. Stage 2 (cqrs) queued as task 020.
