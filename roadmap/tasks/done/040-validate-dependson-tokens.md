# 040 — Catch a `dependsOn` token that nothing can ever emit

**Status:** done
**Priority:** medium
**Created:** 2026-08-09
**Picked up by:** agent
**Started:** 2026-08-09
**Completed:** 2026-08-09

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

- [x] The API exposes the set of entity tokens derivable from the wired consumers
- [x] The set is derived from the wiring, not a hand-maintained list
- [x] `checkgroups` and `year` are present
- [x] A bogus `dependsOn` produces a console warning in dev, and nothing in prod
- [x] A test covers a token that is not in the set

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
  fragments of multi-line concatenations (`"NATHEJK:2026.payment."`,
  `".received"`). Runtime derivation excludes all of it by construction.
- 2026-08-09 — Go side done: `internal/live/entities.go` (`EntitySet`, `EntitiesFrom`)
  plus the `entities` SSE frame, announced **before** subscribing so it precedes the
  initial resync — a client cannot react to a signal it could not yet validate.
  Wired in `cmd/api/main.go` from the same `projections` slice fed to `NotifyAll`, and
  logged at boot.

  ✅ Verified against the running API rather than only by unit test. `wget` on
  `/api/stream` returns, as the first frame:

  ```
  event: entities
  data: {"entities":["bandit","checkgroup","checkgroups","checkpersonnel","checkpoint",
         "crew","crewmember","friend","gøgler","klan","lok","order","patrulje",
         "payment","qr","section","sections","senior","spejder","year"],
         "exhaustive":false}
  ```

  Three things worth reading off that: every token task 037 settled on is present;
  `scan` and `personnel` are absent, which is the whole point; and `exhaustive` is
  `false`, as the wildcard subscriptions predicted.
- 2026-08-09 — Frontend done: `plugins/live/entities.ts` holds the set and warns once
  per unknown token, `sse.ts` routes the frame to it (deliberately *not* onto the
  signal bus — it describes the stream rather than reporting a change), and
  `useLiveResource` validates on registration. Dependencies declared before the set
  arrives are re-checked when it lands, which is the normal case: pages mount before
  the stream connects.
- 2026-08-09 — **Proved the new test can fail.** Commenting out the single
  `validateDependencies` call in `useLiveResource` makes
  `useLiveResource.spec.ts` fail (1 of 13) while every test in `entities.spec.ts`
  still passes — which is exactly the hole the extra test exists to close. Restored
  afterwards.
- 2026-08-09 — **"Nothing in prod" turned into something checkable**, and it exposed a
  verification trap. Wanting the check *provably* dead rather than merely inert, I
  grepped the built bundle for the warning text. It was still there through three
  rewrites (optional chaining, a module `const`, an early return) — and the reason was
  not the code: **`docker compose run ui` sets `NODE_ENV=development`, and Vite derives
  its production mode from `NODE_ENV` before the build mode**, so `npm run build-only`
  in the dev container emits a *dev-mode* bundle in which retaining the code is
  correct. With `NODE_ENV=production` it is eliminated.

  The shipped artefact is unaffected: `docker/Dockerfile`'s `ui-builder` stage sets no
  `NODE_ENV`, and `node:20-alpine` leaves it unset (verified by running the image), so
  the real image builds in production mode. Final form wraps each body in
  `if (import.meta.env.DEV)`, which folds to `if (false)` and drops the code and its
  strings; the caveat is recorded in the file so the next person does not repeat the
  three rewrites. Worth knowing more generally: **`build-only` in the dev container is
  a compile check, not a production-equivalent build.**
- 2026-08-09 — Gates: 57 frontend tests (was 40 — 15 new for the checker, 2 for the
  transport seam, 2 for `useLiveResource`), Go tests/vet/staticcheck/gofmt clean,
  `vue-tsc` 106 (baseline), eslint clean on touched files, production build passes.
- 2026-08-09 — Done. Deliberately left for later: the check is advisory and cannot be
  exhaustive while five consumers subscribe with a wildcard entity. Narrowing those
  subscriptions would make the set complete and the warning definitive — worth doing
  if this ever produces a false positive, but not worth touching working projections
  speculatively.
