# 141 — track projection (track_latest + track_point)

**Status:** done
**Priority:** high
**Created:** 2026-09-03
**Picked up by:** agent session
**Started:** 2026-09-03
**Completed:** 2026-09-03

## Description

PRD 011 §4a, §8. Depends on task 140 (live signals) and, for deployment, task 139 (stream).

New read-only projection `go/nathejk/table/track/` following the house layout (`table.go`,
`consumer.go`, `query.go`, `filter.go`, `table.sql`) — no `commands.go`, hq never writes
telemetry. `Consumes()` returns the single subject `TELEMETRY.*.track.*.reported`.

Body (`hej-api`, pinned in PRD 011 §4a):

```json
{ "personId": "…", "userType": "gøgler", "year": "2026",
  "points": [ { "ts": 1788437919856, "lat": 55.709, "lng": 12.600, "accuracy": 18.7 } ] }
```

Two tables:

- `track_latest(personId PK, personType, year, latitude, longitude, accuracy, ts, updatedAt)`
- `track_point(personId, ts, personType, year, latitude, longitude, accuracy, PRIMARY KEY (personId, ts))`

**The key is `(personId, ts)`, not `(year, personId, ts)`** — that is the producer's own
definition of a point's identity, and the reason `ts` is an integer (ms) rather than RFC 3339:
a formatted date can be re-serialised into a different-but-equal form and read as two points.
`INSERT IGNORE` on that key makes both client retries and boot replay idempotent for free.

Points arrive **batched**, so the handler loops `body.points`, inserting each, and advances
`track_latest` only if the point is newer than the stored `ts` (an out-of-order point must not
regress it).

`personType` is stored **from the message** on both tables and never joined: roles change
while the stream is permanent, so a lookup against today's directory would silently
reinterpret last year's history.

Store coordinates as `DECIMAL(9,6)`/`DECIMAL(10,7)`, `ts` as `BIGINT` milliseconds — not the
`VARCHAR(99)` coordinates and `INT` seconds that `scan` uses. **Do not re-validate points**:
`track.Clean` at the producer already drops NaN, Null Island, impossible clocks and >100 km
accuracy, and deliberately keeps poor-but-real fixes. hq stores what it is given.

Not in scope: vehicle positions, which are a separate seam (task 120).

## Acceptance Criteria

- [x] `go/nathejk/table/track/` exists with the house file layout
- [x] `Consumes()` returns `TELEMETRY.*.track.*.reported`
- [x] Batched points all inserted; PK `(personId, ts)` with `INSERT IGNORE`
- [x] `track_latest` never regresses to an older point
- [x] `personType` persisted as received on both tables
- [x] Added to the `projections` slice in `cmd/api/main.go` (inside it, so `live.NotifyAll` wraps it)
- [x] `go build ./...` clean

## Progress Log

- 2026-09-03 — Task created from PRD 011 §10.
- 2026-09-03 — Picked up. Plan: `table/track` in the `kort` house style (goqu, documented `table.sql`, `querier` interface), messages struct local to the package since hej-app's `track` package is not in shared-go yet, then wire into `projections`.
- 2026-09-03 — Split the schema into `table.sql` (`track_latest`) and `point.sql` (`track_point`): `cqrs.Writer.Consume` takes **one** statement, so two `CREATE TABLE`s in one file would silently create only the first. `kort` does the same thing with `kortsaet.sql`, so this follows an existing precedent rather than inventing one.
- 2026-09-03 — `Reported` is a local struct, not an import. hej-app's `track` package is early work and not lifted to shared-go yet; field names and JSON tags are deliberately identical so that when it is lifted, this becomes a deletion rather than a translation. Noted in `messages.go`.
- 2026-09-03 — The out-of-order guard is `IF(VALUES(ts) > ts, VALUES(col), col)` per column, one statement, rather than read-then-write. Two reasons: a read-then-write is a race, and at one round trip per person per batch it would be the most expensive thing in the package. This matters more than it first looks — without it a boot replay would leave every person's "last seen" showing whichever message was applied last rather than the most recent position, and the glyph would then lie about staleness on every page in hq.
- 2026-09-03 — Chunked inserts at 500 rows. The Writer takes rendered SQL rather than arguments, so a 2,000-point backlog would otherwise be one very long statement.
- 2026-09-03 — Wrote `querier.go` with `Presence`, `Points` and `LatestFor` — the reads tasks 142 and 147 need — plus `filter.go` with millisecond bounds (no `time.Time` anywhere, so one representation of an instant runs from the phone to the map). `Points` deliberately does **not** filter on year: the time bounds select an event far more precisely, and adding year would only skip the `(personId, ts)` index the query exists to use.
- 2026-09-03 — ✅ All criteria. Wired as `tracktable` into the `projections` slice next to `scantable`. `go build ./...`, `go vet ./...`, `gofmt` all clean; 7 consumer tests pass, including the real PRD 011 §4a payload, newest-in-batch selection, the guard expressions, empty batch, chunking, and the subject's domain being `TELEMETRY` (which is what makes hq read a second stream at all).
- 2026-09-03 — Note for deployment: this is now the first consumer whose subject domain is not `NATHEJK`, so **task 139 gates deploying it**. If the `TELEMETRY` stream does not exist, `OrderedConsumer` fails, `mux.Run` returns an error and the api `PrintFatal`s — hq will not start, rather than merely lacking telemetry.
- 2026-09-03 — Completed. Positions now project into `track_latest`/`track_point`, and `track` is advertised as a live entity automatically via task 140.
