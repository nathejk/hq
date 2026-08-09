# 040 — Catch a `dependsOn` token that nothing can ever emit

**Status:** doing
**Priority:** medium
**Created:** 2026-08-09
**Picked up by:** agent
**Started:** 2026-08-09
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
- 2026-08-09 — Picked up. First finding changes the design: **the set cannot be
  exhaustive.** Five wired subject patterns use `*` in the *entity* position —
  `NATHEJK:*.*.*.signedup` plus the `emailaddress.verified`, `phonenumber.verified`,
  `mail.validate.sent` and `sms.validate.sent` patterns — so hq listens to those
  events for *any* entity, and a token outside the enumerated set could legitimately
  arrive. Claiming completeness would make the check lie.

  So the contract becomes: the API reports the concrete tokens **and whether the set
  is exhaustive**, and the client's warning is worded as advisory when it is not. That
  still catches `scan` and `personnel` — the two real mistakes — while never asserting
  something it cannot know.

  Second decision: derive the tokens by feeding each pattern through
  `SignalFromSubject`, the very function that produces the signals, rather than a
  second parser. A pattern like `NATHEJK.*.checkgroup.*.created` yields
  `Entity: "checkgroup"`, and `NATHEJK.*.created` yields the synthesised `year` — both
  for free. If the parser ever changes, the advertised set changes with it, which is
  the property that makes this worth having at all.

  Third: derive from the **wired consumers at runtime**, not by grepping source. A
  grep of `FromStr("…")` also picks up commented-out patterns (`monolith:nathejk_team`,
  `nathejk`) and fragments of multi-line concatenations (`"NATHEJK:2026.payment."`,
  `".received"`). Runtime derivation excludes all of it by construction.
