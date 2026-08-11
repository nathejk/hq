# 046 — Wire SOS into cmd/api

**Status:** done
**Priority:** high
**Created:** 2026-08-11
**Picked up by:** agent session
**Started:** 2026-08-11
**Completed:** 2026-08-11

## Description

From **PRD 001** §8. Wire the new SOS projections, queries and commands into
`go/cmd/api/main.go`: `data.NewModels(...)`, `commands.New(...)`, the `xstream.Mux`, and
crucially the **`projections` slice** — a consumer added to the mux outside that slice is
silently not live, because `live.NotifyAll` wraps only what is in the slice.

Depends on 042, 043, 044, 045.

## Acceptance Criteria

- [x] `app.models.Sos` and `app.commands.Sos` available to handlers
- [x] All SOS consumers appear in the `projections` slice (so they emit live signals)
- [x] `live.EntitiesFrom(projections...)` advertises the `sos` token
- [x] API starts, creates the tables, and replays without error
- [x] `go build ./... && go vet ./...` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-11 — Created from PRD 001 on approval.
- 2026-08-11 — Pulled forward ahead of task 045: 045's toggle endpoint needs
  `app.commands.Sos`, so wiring had to land first. The task order in the PRD had the
  dependency backwards.
- 2026-08-11 — Wired `sostable` into `data.NewModels`, `commands.New` (as `cmds.Sos`) and
  the `projections` slice in `cmd/api/main.go`. The slice is the part that matters: a
  consumer added to the mux outside it is silently not live.
- 2026-08-11 — Added `cmd/api/actor.go`: one helper turning the request context's user
  into a `sos.Actor`. This is the seam that keeps `requestctx` out of the sos package —
  every other local table package imports it directly, which is exactly what makes those
  packages hard to lift.
- 2026-08-11 — ✅ Verified against the running dev stack rather than by inspection: the
  hot-reloaded API created `sos`, `sos_team`, `sos_activity` and
  `sos_assignable_section`, replayed the stream, answers the healthcheck, and now
  advertises `sos` in its live entity set — which is what the SPA's dev-only `dependsOn`
  validation checks against.
- 2026-08-11 — Noted in passing, **not** fixed (unrelated, pre-existing): the patrulje
  projection logs `Error 1406: Data too long for column 'groupName'` on replay for one
  2025 team whose group name is ~160 chars against a `VARCHAR(99)`. Worth its own task —
  that team's row is silently stale in the read model.
- 2026-08-11 — All criteria met. Moving to done.
