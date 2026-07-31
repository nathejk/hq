# 010 — Add staticcheck to the go.mod tool section

**Status:** open
**Priority:** medium
**Created:** 2026-07-31
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] staticcheck is pinned in `go.mod` (tool directive, per chosen option).
- [ ] `go tool staticcheck ./...` runs.
- [ ] Go version / Dockerfile kept consistent if bumped.

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-07-31 14:50 — Task created. Flagged the Go 1.23 vs tool-directive (1.24) conflict; awaiting decision on the Go bump.
