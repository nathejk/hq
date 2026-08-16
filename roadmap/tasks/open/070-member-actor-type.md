# 070 — An actor type for the member package

**Status:** open
**Priority:** medium
**Created:** 2026-08-17
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] Member commands take an actor without importing `sos`
- [ ] Approach chosen and the reason recorded in the log
- [ ] `cmd/api` resolves it from `requestctx` as `actor.go` does today
- [ ] Task 081's lift-readiness check passes with it in place
- [ ] `go build ./... && go vet ./...` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006. Needed before task 072 publishes anything.
