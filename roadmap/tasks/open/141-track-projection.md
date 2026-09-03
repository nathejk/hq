# 141 — track projection (track_latest + track_point)

**Status:** open
**Priority:** high
**Created:** 2026-09-03
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] `go/nathejk/table/track/` exists with the house file layout
- [ ] `Consumes()` returns `TELEMETRY.*.track.*.reported`
- [ ] Batched points all inserted; PK `(personId, ts)` with `INSERT IGNORE`
- [ ] `track_latest` never regresses to an older point
- [ ] `personType` persisted as received on both tables
- [ ] Added to the `projections` slice in `cmd/api/main.go` (inside it, so `live.NotifyAll` wraps it)
- [ ] `go build ./...` clean

## Progress Log

- 2026-09-03 — Task created from PRD 011 §10.
