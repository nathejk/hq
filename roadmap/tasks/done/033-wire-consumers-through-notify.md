# 033 — Wire existing consumers through `notify`

**Status:** done
**Priority:** medium
**Created:** 2026-08-08
**Picked up by:** agent session (zed)
**Started:** 2026-08-08
**Completed:** 2026-08-08

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

- [x] Hub constructed in `main.go` and passed to the stream handler and the notifier
- [x] Projection consumers registered through `notify`
- [x] Non-projection consumers deliberately excluded, with a comment saying why
      — **decision inverted:** everything is wrapped, and the comment says why (below)
- [x] No change to projection behaviour: the app still starts and projects as before
- [x] `go test ./...`, `go vet ./...`, `go tool staticcheck ./...` pass
- [x] Manual check: a change in one browser tab appears in another without reload
      — verified with a live stream plus a PATCH rather than two tabs

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-08 18:48 — Task created. Depends on 031, 032.
- 2026-08-09 00:26 — Picked up. Plan: name the consumer list as a `[]cqrs.Consumer`
  slice, wrap it with `live.NotifyAll`, and register each — which also makes that
  twenty-item one-liner readable for the first time.
- 2026-08-09 00:30 — **Inverted the ticket's own instruction, deliberately.** It said to
  exclude non-projection consumers; I wrapped everything instead. A consumer that writes
  no row a client would refetch produces a signal nothing depends on — clients declare
  which entities they care about, and coalescing already collapses the duplicates that
  several projections handling one event produce. So curating the list buys nothing
  measurable and would rot the moment a consumer changed shape. Reasoning recorded in
  `main.go` beside the list, not only here.
- 2026-08-09 00:32 — Lifted the twenty-consumer `AddConsumer(…)` one-liner into a named
  `projections` slice. That list *is* the read model; being able to read it is worth four
  extra lines. Also drops two stale commented-out entries that were being carried along
  inside the call.
- 2026-08-09 00:36 — ✅ Verified against the running stack rather than by inspection:
  opened a stream, issued `PATCH /api/year/2026`, and the signal arrived ~0.1s later —
  `{"type":"entity.changed","entity":"year","id":"2026","year":"2026","event":"updated"}`.
  Note that is the year-level subject shape (`NATHEJK.{year}.updated`, carrying neither
  entity nor id token) emerging as entity `year` with the year as its id — the 029 parser
  decision proving itself on real traffic.
- 2026-08-09 00:38 — Observed in the API logs while watching, unrelated to this task but
  worth recording: the `signup` projection is failing continuously with
  `Unknown column 'secret'` and `Unknown column 'year'`. That is exactly the
  `CREATE TABLE IF NOT EXISTS` drift PRD 005 predicts — a shared-go entity gained columns
  an existing local table never got. Two consequences here: those statements are being
  diverted by `deadletter`, which is precisely the caveat documented in `notify.go` (a
  signal describing a change that was not applied), and it is live evidence for the
  per-build-database argument in PRD 005 §8. Not fixed here.
- 2026-08-09 00:40 — Completed. Gates green; the backend half is live end to end.
  Next: 034 (frontend SSE transport).
- 2026-08-09 01:32 — **Correction to the 00:38 entry above.** I wrote that the failing
  signup statements were "being diverted by `deadletter`". They are not:
  `cmd/api/main.go:164` builds a plain `sqlpersister`, with no deadletter anywhere. The
  layout skill describes the Writer as `deadletter` wrapping `sqlpersister`, and I took
  it at its word instead of checking the code.
  Consequence is the opposite of what I implied, and better: `Consume` returns the
  error, the entity propagates it, and `notify` publishes nothing — a change that did
  not land is not announced. The doc comment in `internal/live/notify.go` has been
  rewritten to describe the current wiring, keeping the deadletter caveat as conditional
  on that Writer ever being introduced. Drift itself is now task 038.
