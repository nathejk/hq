# 040 — Catch a `dependsOn` token that nothing can ever emit

**Status:** open
**Priority:** medium
**Created:** 2026-08-09
**Picked up by:**
**Started:**
**Completed:**

## Description

`dependsOn` is a list of bare strings, and a wrong one fails in the worst possible
way: the page looks live, never errors, and simply never updates. This is not
hypothetical — task 037 found that two of the six tokens written down in its own
plan were wrong:

- scans are published as `NATHEJK.*.qr.*.scanned`, so the token is `qr`. A
  projection's package name (`scan`) is not its subject's entity.
- personnel is `gøgler` / `friend` / `bandit`. There is no `personnel` token, so
  `BadutListView` would have been silently dead.

Both were caught only by reading Go source. That is not a repeatable safeguard.

The API already knows the answer: the union over every registered consumer's
`Consumes()` of the entity token in each subject pattern *is* the set of tokens the
stream can ever emit. Expose it, and the client can complain about a dependency that
can never fire.

### Sketch

- Go: derive the set from the same `projections` slice that `live.NotifyAll` wraps,
  so it cannot drift from what is actually wired. `live.SignalFromSubject` already
  knows how to pull the entity out of a subject; wildcard ids (`*`) are fine because
  only `parts[2]` is needed. Serve it on the stream endpoint (e.g. as an initial
  `event: entities` frame, which costs no extra request and no extra route) or as
  `GET /api/stream/entities`.
- Frontend: in dev only, `useLiveResource` warns once per unknown token. Never throw,
  and never in production — a stale allow-list must not break a page.
- Watch the collection-level tokens: `checkgroups` (from `checkgroups.sorted`) is a
  legitimate token that is not any table's name, and `year` is synthesised by the
  parser rather than appearing in a subject. Both must be in the set.

## Acceptance Criteria

- [ ] The API exposes the set of entity tokens derivable from the wired consumers
- [ ] The set is derived from the wiring, not a hand-maintained list
- [ ] `checkgroups` and `year` are present
- [ ] A bogus `dependsOn` produces a console warning in dev, and nothing in prod
- [ ] A test covers a token that is not in the set

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-09 — Created from 037, after two of six planned tokens turned out wrong.
