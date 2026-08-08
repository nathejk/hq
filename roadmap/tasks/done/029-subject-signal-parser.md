# 029 — Subject → signal parser

**Status:** done
**Priority:** high
**Created:** 2026-08-08
**Picked up by:** agent session (zed)
**Started:** 2026-08-08
**Completed:** 2026-08-08

## Description

First slice of PRD 004 Phase 2. A pure function turning an event subject into the
`entity.changed` signal the SPA already understands (shipped in 023).

This is what makes the whole design *zero per-entity code*: the subject convention
already carries everything a signal needs, so one parser serves every entity present
and future.

```
NATHEJK.{year}.{entity}.{id}.{event…}
```

Lives in `go/internal/live/` — infrastructure for this binary, not a domain
aggregate, so `internal/` per the layout skill's default.

### The shapes it must survive

Verified against the actual subjects in this repo:

- **The common case**, 5+ tokens: `NATHEJK.2026.patrulje.{id}.started`
- **Event names containing dots**: `status.changed`, `armNumber.assigned`,
  `lines.changed`, `checkpoints_sorted` — so the event is *everything after the id*,
  not the last token
- **Year-level, no id** (3 tokens): `NATHEJK.{year}.created|updated`
  (`table/year/consumer.go:21`)
- **Collection-level, no id** (4 tokens): `NATHEJK.{year}.checkgroups.sorted`
  (`table/checkgroup/consumer.go:25`) — note the plural entity token
- **Both separators**: `NATHEJK.` (83 uses) and `NATHEJK:` (45). No special handling
  needed — `subject.FromStr` already normalises the first `:` to `.`
  (`stream/subject/subject.go:22`) — but assert it, because it is load-bearing and
  invisible.
- **Non-ASCII entity tokens**: `gøgler`, `nødtelefon` — must pass through unmangled

A subject that cannot be parsed must be reported as such, not silently turned into a
signal for entity `""` — a bad signal is worse than none, because the client would
invalidate nothing while appearing to work.

## Acceptance Criteria

- [x] `internal/live` package with a parser mapping a `cqrs.Subject` to a signal
- [x] Extracts year, entity, optional id, and the full dotted event name
- [x] Id-less shapes (year-level, collection-level) parse with no id rather than
      mis-assigning a token
- [x] `NATHEJK:`-style subjects parse identically to `NATHEJK.`
- [x] Non-ASCII entity tokens survive intact
- [x] Unparseable subjects are rejected explicitly
- [x] Table-driven tests covering every shape above, using real subjects from this
      repo
- [x] `go test ./...`, `go vet ./...` and `go tool staticcheck ./...` all pass

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-08 18:40 — Task created. First of Phase 2 (029–035).
- 2026-08-08 18:55 — Picked up. Plan: `internal/live/signal.go` with the Signal type and
  `SignalFromSubject`, table-driven tests using subjects copied from this repo's
  consumers. Using `cqrs` aliases rather than importing `jrgensen/stream` directly.
- 2026-08-08 19:02 — Added `internal/live/signal.go`: `Signal` (type/entity/id/year/event),
  `SignalFromSubject`, `Resync()` and `Signal.Key()` for coalescing.
  Decisions:
  • **Reject rather than degrade.** An unparseable subject returns `ErrNotASignal`
    instead of a zero-valued Signal — a signal naming entity `""` would invalidate
    nothing on the client while looking like it worked.
  • **Year-level events (`NATHEJK.{year}.created`) report entity `"year"` with the year
    as the id.** They carry no entity token at all, so something had to be chosen; a
    client can then depend on `'year'` by name like any other entity, and the year value
    is genuinely what identifies which one changed.
  • **Event is `strings.Join(parts[4:], ".")`**, not the last token, so `status.changed`
    and `armNumber.assigned` survive whole.
  • **Domain match is case-insensitive** (`EqualFold`), since subjects appear as both
    `NATHEJK` and lowercase in the wild.
  • `Key()` is `entity:id`, which is the coalescing unit 030 needs: two events about
    one instance are interchangeable to a client.
- 2026-08-08 19:08 — Tests use subjects copied from this repo's consumers rather than
  invented ones — `checkgroups.sorted` (`table/checkgroup/consumer.go:25`),
  `NATHEJK.{year}.created` (`table/year/consumer.go:21`), `order.*.lines.changed`,
  `armNumber.assigned`, plus a `gøgler` case for non-ASCII. Added an explicit test that
  a `NATHEJK:`-style subject parses identically, because `subject.FromStr` normalising
  the first colon (`stream/subject/subject.go:22`) is invisible and load-bearing — if
  that upstream behaviour ever changed, 45 subjects in this repo would silently stop
  producing signals.
- 2026-08-08 19:12 — ✅ All gates pass in the `api` container: 15 test cases green,
  `gofmt` clean, `go vet` OK, `go tool staticcheck` OK, full `go test ./...` with no
  failures. This also confirmed the new `base` Dockerfile stage builds under compose.
- 2026-08-08 19:14 — Completed. Next: 030 (hub).
