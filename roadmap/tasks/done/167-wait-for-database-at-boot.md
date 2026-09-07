# 167 — Wait for the database at boot instead of dying

**Status:** done
**Priority:** high
**Created:** 2026-09-07
**Picked up by:** agent session
**Started:** 2026-09-07
**Completed:** 2026-09-07

## Description

The API died at boot whenever MySQL was not already accepting connections:

```
{"level":"FATAL","message":"dial tcp 172.21.0.2:3306: connect: connection refused"}
exit status 1
```

`database.Open` pinged **once** with a 5-second deadline and `main` turned any failure into
`PrintFatal`. That makes the API's boot depend on winning a race it does not control: every
`docker compose up` starts it alongside MySQL, and MySQL is routinely slower.

**In dev nothing brings it back.** The hot-reload loop only rebuilds on a *file change*, so
after the fatal exit the process stays dead until somebody edits a file or restarts the
container. Meanwhile the ui container keeps serving the app shell perfectly well, so what an
operator sees is a page that loads and then fails every request — which reads as a broken
feature, not a missing database. That is exactly how it presented when it happened: it took
a walk back through vite proxy errors, container IPs and Traefik before the actual cause
(`connect: connection refused` to MySQL, four minutes earlier) turned up.

## Notes

- `Open` now retries the ping with exponential backoff to a 5s ceiling, over a **90s
  budget**, and logs every attempt. Bounded on purpose: retrying forever would turn a wrong
  password into a process that hangs while looking healthy.
- **Every error is retried, not just the ones that look transient.** "Connection refused" is
  the obvious case, but a first boot can also legitimately report an unknown database while
  the server's init scripts are still creating it — and hand-classifying driver errors is how
  a retry loop quietly stops covering the case it was written for. The budget is what makes
  that safe.
- The final wait is clamped to the remaining budget, so the promise about total time holds.
- The first wait is capped by `connectMaxWait` too, so the backoff cannot start above its own
  ceiling. That also makes the loop testable with a short budget — without it, a 250ms test
  budget clamped the initial 1s wait and produced exactly one attempt, which is how the first
  version of the test passed while proving nothing.
- Config errors (an unparseable `maxIdleTime`) still fail immediately: they are not
  connection problems and waiting 90s to report a typo would be worse than useless.
- The failure message names the attempt count and the budget, because
  `connect: connection refused` on its own is what made this expensive to diagnose.
- Not addressed here: **the process still exits if the database is unreachable for the whole
  budget, and the dev loop still will not restart it.** That is the right behaviour for a
  genuine misconfiguration, but a `restart: unless-stopped` (or an inotify loop that also
  reacts to the process dying) would make dev self-healing. Left alone as a compose/tooling
  decision rather than a code one.

## Acceptance Criteria

- [x] A boot with MySQL down waits instead of exiting
- [x] It connects by itself once MySQL becomes available
- [x] It still fails, with a clear message, if the database never appears
- [x] Config errors are not retried
- [x] Failure message names the attempt count and the budget
- [x] Tests covering: gives up after the budget, retries more than once, wraps the underlying
      error, does not retry a config error
- [x] Verified against the dev stack, not just in tests

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-09-07 — Found while diagnosing "webserver ui not responding": the ui was fine, the API
  had been dead for four minutes after the stack restarted and it lost the race with MySQL.
- 2026-09-07 — Implemented in `cmd/api/database.go` with four tests in `database_test.go`.
- 2026-09-07 — Verified end to end on the real stack: stopped MySQL, restarted the API, and
  watched it retry (`not ready (attempt 6) … retrying in 5s`) rather than exit; started MySQL
  and it logged `database: reachable after 8 attempts`, then `starting server`, and answered
  200 through the vite proxy. Before this change that same sequence killed the process.
- 2026-09-07 — Build, vet, fmt and the full Go suite green.
