# 070 — An actor type for the member package

**Status:** done
**Priority:** medium
**Created:** 2026-08-17
**Picked up by:** agent session
**Started:** 2026-08-17
**Completed:** 2026-08-17

## Description

From **PRD 006** §8 ("The acting user needs its own type").

`app.actor` (`go/cmd/api/actor.go:23`) returns `sos.Actor` — a type owned by the `sos`
package. The `spejderstatus` package is written to be liftable to shared-go and therefore
**may not import `nathejk.dk/...`**, which includes `nathejk.dk/nathejk/table/sos`.

So the member commands need their own actor type, or a shared one if a second is one too
many. Small change; it is on the board because the obvious shortcut — importing `sos` for
the one type — is exactly the convenience import that turns the future lift from a file
move into a rewrite, and nothing in the build will complain.

Options:

1. `spejderstatus.Actor` mirroring `sos.Actor`, with a second `app.memberActor(r)` helper.
   Duplication, but each package stays self-contained and lifts independently.
2. A single `types.Actor` in shared-go that both packages use. Less duplication, but it is
   a cross-repo change and `sos.Actor` is already shipped and already lifted-ready.

Option 1 is the proposal: two tiny structs is a cheaper price than a cross-repo change on
the critical path, and it matches how `sos` solved the same problem.

## Notes

- Today the middleware puts an anonymous user with an empty id on every request —
  authentication is perimeter-only, basic auth on stage/production and nothing in dev, with
  a JWT service planned (PRD 001 §6 Auth). So the actor is empty in practice. Wire it
  anyway: when identity arrives, the events start carrying it with no change in the domain.
- The handler is the layer that knows about HTTP, so it resolves the actor and passes it
  in. `actor.go`'s own doc comment explains why, and is worth reading before choosing.

## Acceptance Criteria

- [x] Member commands take an actor without importing `sos`
- [x] Approach chosen and the reason recorded in the log
- [x] `cmd/api` resolves it from `requestctx` as `actor.go` does today
- [x] Task 081's lift-readiness check passes with it in place
- [x] `go build ./... && go vet ./...` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006. Needed before task 072 publishes anything.
- 2026-08-17 — **Option 1 chosen** (a package-local `Actor`), as proposed. Half of it was
  already done: `spejderstatus.Actor` was defined in `messages.go` during task 063, because
  the event bodies needed a field for it. This task added the missing half —
  `app.memberActor(r)` in `cmd/api/actor.go`.
- 2026-08-17 — Wrote the justification into the code rather than only here, because two
  structurally identical five-line structs are exactly what a later reader "tidies up" by
  making one import the other. The comment states the cost of that: `spejderstatus` and `sos`
  are each written to be lifted to shared-go **independently**, so a shared `sos.Actor` would
  make the member package depend on the SOS domain for no reason beyond saving five lines,
  and its lift would stop being a file move. Task 081's guard would fail — which is the point
  of having it, but a comment is cheaper than a failing test to interpret.
- 2026-08-17 — Recorded the rejected alternative (one `types.Actor` in shared-go) with the
  condition for revisiting it: a third occurrence. Three is a pattern, two is a coincidence,
  and it is not worth a cross-repo change on the critical path to remove two small structs.
- 2026-08-17 — ✅ All criteria met. `go build`, `go vet`, `gofmt` clean; task 081's
  lift-readiness test still passes. Moving to done.
