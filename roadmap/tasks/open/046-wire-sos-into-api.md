# 046 — Wire SOS into cmd/api

**Status:** open
**Priority:** high
**Created:** 2026-08-11
**Picked up by:**
**Started:**
**Completed:**

## Description

From **PRD 001** §8. Wire the new SOS projections, queries and commands into
`go/cmd/api/main.go`: `data.NewModels(...)`, `commands.New(...)`, the `xstream.Mux`, and
crucially the **`projections` slice** — a consumer added to the mux outside that slice is
silently not live, because `live.NotifyAll` wraps only what is in the slice.

Depends on 042, 043, 044, 045.

## Acceptance Criteria

- [ ] `app.models.Sos` and `app.commands.Sos` available to handlers
- [ ] All SOS consumers appear in the `projections` slice (so they emit live signals)
- [ ] `live.EntitiesFrom(projections...)` advertises the `sos` token — verifiable at
      `GET /api/stream`, which the SPA's dev-only dependency validation reads
- [ ] API starts, creates the tables, and replays without error
- [ ] `go build ./... && go vet ./...` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-11 — Created from PRD 001 on approval.
