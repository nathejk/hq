# 178 — Wire searchperson into the projections slice and confirm live tokens

**Status:** open
**Priority:** high
**Created:** 2026-09-14
**Picked up by:**
**Started:**
**Completed:**

## Description

Depends on task 177. Adds `searchpersontable` to the `projections` slice in
`go/cmd/api/main.go`.

It must go in **that slice**, not straight onto the mux: the slice is what `live.NotifyAll`
wraps, and a consumer added outside it silently emits no live signals and never learns it is
caught up.

The second half of this task is verification, and it is the point of splitting it out. The
projection now subscribes to entity tokens from six event families, and the frontend
(task 182) will declare dependencies on them. Confirm against the **advertised** set —
`live.EntitiesFrom(projections...)`, logged at boot as "Live entities advertised" — not
against a hand-written list. The expected tokens are `spejder`, `senior`, `gøgler`, `friend`,
`crewmember`, `crew`, `patrulje`, plus whatever the wildcard `signedup` subscription yields.

There is deliberately no `bandit` token — task 175 established that a bandit is a senior with
an arm number, not a population, so nothing here subscribes to a bandit subject.

There is no `personnel` token, and `scan` is really `qr`. Both mistakes are documented in
`go/internal/live/entities.go` because both have been made before, and both fail *silently*:
the page looks live and simply never updates.

Note that adding this consumer makes one more pass over the same event families on every
restart. Marginal, but real — sanity-check boot time.

## Acceptance Criteria

- [ ] `searchpersontable` present in the `projections` slice in `main.go`
- [ ] Boot log's advertised entity set contains every token the search view will depend on
- [ ] The token list is recorded in the progress log, for task 182 to use verbatim
- [ ] Boot completes and the gate opens (no consumer left never reporting caught-up)
- [ ] `go build ./...` and `go test ./...` pass

## Progress Log

- 2026-09-14 22:08 — Task created from PRD 014 §10.
