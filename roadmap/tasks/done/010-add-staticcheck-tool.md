# 010 — Add staticcheck to the go.mod tool section

**Status:** done
**Priority:** medium
**Created:** 2026-07-31
**Picked up by:** agent (opus-4.8 session)
**Started:** 2026-07-31
**Completed:** 2026-07-31

## Description

Register `honnef.co/go/tools/cmd/staticcheck` as a `tool` directive in
`go/go.mod` so it is version-pinned and runnable via `go tool staticcheck`
(matching the `go-bff-layout` skill's stated dev-loop gate).

**Blocker / decision needed:** the `tool` directive is a **Go 1.24** feature.
This repo is currently on `go 1.23.0` (`go/go.mod`) with `toolchain go1.23.10`,
and the dev container is `FROM golang:1.23` (`docker/Dockerfile`). Adding a
`tool` directive therefore requires bumping the Go version — which the project
rules say to do "as its own task" and keep the Docker image in sync.

Options:
- (A) Bump `go.mod` go directive to `1.24` (+ toolchain) and the Dockerfile
  base to `golang:1.24`, then `go get -tool honnef.co/go/tools/cmd/staticcheck`.
- (B) Use the pre-1.24 `tools.go` blank-import pattern instead (not literally a
  go.mod `tool` section).

## Acceptance Criteria

- [x] staticcheck is pinned in `go.mod` (`tool honnef.co/go/tools/cmd/staticcheck`).
- [x] `go tool staticcheck ./...` runs.
- [x] Go version / Dockerfile kept consistent (see note on 1.25 below).

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-07-31 14:50 — Task created. Flagged the Go 1.23 vs tool-directive (1.24) conflict; awaiting decision on the Go bump.
- 2026-07-31 15:00 — Picked up. User chose Option A: bump Go 1.23→1.24 (go.mod + Dockerfile) and add the staticcheck tool directive.
- 2026-07-31 15:10 — `go get -tool honnef.co/go/tools/cmd/staticcheck` pulled staticcheck v0.7.0, which **requires Go >= 1.25**, so it auto-bumped `go.mod` to `go 1.25.0` (not 1.24). Aligned the Dockerfile to `golang:1.25` — which also matches the `go-bff-layout` skill's stated image. Deviation from my quoted "1.24" flagged for the user.
- 2026-07-31 15:15 — The upgrade left `go.sum` incomplete (build failed on missing entries for excelize/nkeys/go-mail transitive x/* pkgs). Ran `go mod tidy` to reconcile. `go build ./...`, `go vet ./...`, `go test ./...` all green afterwards.
- 2026-07-31 15:20 — `go tool staticcheck ./...` runs. My added/changed files (order/product/tablerow/cmd-api order.go/patrulje.go) are staticcheck-clean. BUT staticcheck reports **27 pre-existing findings** across other packages (cmd/api signup/mail/ctrlgrpcmd, nathejk/commands + several table pkgs). Since the dev loop gates on `go tool staticcheck ./...`, those now block it — logged as follow-up task 011. Completed.
