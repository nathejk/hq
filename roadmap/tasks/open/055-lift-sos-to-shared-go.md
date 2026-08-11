# 055 — Follow-up: lift the sos package into shared-go

**Status:** open
**Priority:** low
**Created:** 2026-08-11
**Picked up by:**
**Started:**
**Completed:**

## Description

From **PRD 001** §8. Once the SOS schema and events have been through an event and are
stable, move `go/nathejk/table/sos/` to `shared-go/tables/sos/`, together with the domain
vocabulary defined in it by task 042 (`SosCommentID`, severity, the message structs —
which may also belong in `shared-go/messages` at that point).

Deliberately **not** done up front: PRD 001 §8 (amended 2026-08-11) records why — a
shared-go change is only consumable here after commit, push, release and a `go.mod` bump,
and the dev container mounts only `./go`, so doing it first would put a cross-repo release
cycle in front of every other task.

Handlers (`go/cmd/api/sos.go`) and routes stay in hq permanently; only the projection,
queries, commands, schema and vocabulary move.

**Do not start this until the feature has been used in anger.** Its whole purpose is that
the shape has stopped changing.

## Acceptance Criteria

- [ ] Package moved to `shared-go/tables/sos/` unchanged (a file move, per task 054)
- [ ] shared-go released; `go/go.mod` bumped; hq imports the shared package
- [ ] Local copy deleted, not left in parallel
- [ ] API replays and serves identically afterwards

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-11 — Created from PRD 001 on approval. Not to be picked up until the SOS
  schema and events have been exercised during an event.
