# 029 — Subject → signal parser

**Status:** open
**Priority:** high
**Created:** 2026-08-08
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] `internal/live` package with a parser mapping a `cqrs.Subject` to a signal
- [ ] Extracts year, entity, optional id, and the full dotted event name
- [ ] Id-less shapes (year-level, collection-level) parse with no id rather than
      mis-assigning a token
- [ ] `NATHEJK:`-style subjects parse identically to `NATHEJK.`
- [ ] Non-ASCII entity tokens survive intact
- [ ] Unparseable subjects are rejected explicitly
- [ ] Table-driven tests covering every shape above, using real subjects from this
      repo
- [ ] `go test ./...`, `go vet ./...` and `go tool staticcheck ./...` all pass

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-08 18:40 — Task created. First of Phase 2 (029–035).
