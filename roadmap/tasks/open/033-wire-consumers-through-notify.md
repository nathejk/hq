# 033 — Wire existing consumers through `notify`

**Status:** open
**Priority:** medium
**Created:** 2026-08-08
**Picked up by:**
**Started:**
**Completed:**

## Description

Make the whole SPA live in one wiring change: wrap the consumers already registered in
`cmd/api/main.go` with the `notify` decorator (031).

`main.go` currently registers a long `mux.AddConsumer(...)` list — signup, klan,
senior, patrulje, patruljestatus, spejderstatus, personnel, payment, spejder,
checkgroup, checkpoint, checkpersonnel, scan, patruljemerged, lok, year, section,
crewmember, order. Wrapping them is what turns betalinger, patruljer, klaner, poster
and the rest into live pages without a line of per-page backend code.

### Notes

- Keep the wiring readable: a helper that wraps a slice is better than nineteen
  `notify(hub, …)` call sites.
- **Consider what should *not* be wrapped.** Sagas and non-projection consumers
  (`order`'s saga, `confirm`) produce no read-model row a client would refetch;
  wrapping them costs a pointless signal per event. Decide deliberately and note it.
- High-volume subjects (`scan`, `qr`) will now generate signals during a checkpoint
  rush. Coalescing (030) plus the client's `?entities=` filter should absorb it, but
  this is the ticket where that assumption first meets reality — measure if possible.
- The hub must be constructed before the mux and passed in; keep it out of the
  `config` struct (it is not configuration).

## Acceptance Criteria

- [ ] Hub constructed in `main.go` and passed to the stream handler and the notifier
- [ ] Projection consumers registered through `notify`
- [ ] Non-projection consumers deliberately excluded, with a comment saying why
- [ ] No change to projection behaviour: the app still starts and projects as before
- [ ] `go test ./...`, `go vet ./...`, `go tool staticcheck ./...` pass
- [ ] Manual check: a change in one browser tab appears in another without reload

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-08 18:48 — Task created. Depends on 031, 032.
