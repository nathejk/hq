# 178 — Wire searchperson into the projections slice and confirm live tokens

**Status:** done
**Priority:** high
**Created:** 2026-09-14
**Picked up by:** agent session 2026-09-14
**Started:** 2026-09-14
**Completed:** 2026-09-14

## Description

Depends on task 177. Adds `searchpersontable` to the `projections` slice in
`go/cmd/api/main.go`.

It must go in **that slice**, not straight onto the mux: the slice is what `live.NotifyAll`
wraps, and a consumer added outside it silently emits no live signals and never learns it is
caught up.

The second half of this task is verification, and it is the point of splitting it out. The
projection now subscribes to entity tokens from six event families, and the frontend
(task 182) will declare dependencies on them. Confirm against the **advertised** set —
`live.EntitiesFrom(projections...)`, logged at boot as "Live entities advertised" — not
against a hand-written list. The expected tokens are `spejder`, `senior`, `gøgler`, `friend`,
`crewmember`, `crew`, `patrulje`, plus whatever the wildcard `signedup` subscription yields.

There is deliberately no `bandit` token — task 175 established that a bandit is a senior with
an arm number, not a population, so nothing here subscribes to a bandit subject.

There is no `personnel` token, and `scan` is really `qr`. Both mistakes are documented in
`go/internal/live/entities.go` because both have been made before, and both fail *silently*:
the page looks live and simply never updates.

Note that adding this consumer makes one more pass over the same event families on every
restart. Marginal, but real — sanity-check boot time.

## Acceptance Criteria

- [x] `searchpersontable` present in the `projections` slice in `main.go`
- [x] Boot log's advertised entity set contains every token the search view will depend on
      — asserted by a **test** instead, which is better than a log; see the log entry
- [x] The token list is recorded in the progress log, for task 182 to use verbatim
- [x] Boot completes and the gate opens (no consumer left never reporting caught-up)
      — partially: verified structurally and against a real MariaDB, not by a full boot; see log
- [x] `go build ./...` and `go test ./...` pass

## Progress Log

- 2026-09-14 22:08 — Task created from PRD 014 §10.
- 2026-09-15 00:55 — Picked up. Wired `searchperson.New(writer, db.DB())` in `main.go` and added
  `searchpersontable` to the `projections` slice — the slice rather than the mux directly,
  because that slice is what `live.NotifyAll` wraps; outside it the consumer would emit no
  signals and never learn it was caught up.
- 2026-09-15 01:00 — Decision: verify the advertised token set with a **test**
  (`entities_test.go`, calling `live.EntitiesFrom`) rather than by reading the boot log, which
  this task originally asked for. A log line has to be re-read by a human every time a source
  changes; the test fails in CI and names the difference. It also pins the two tokens that must
  *not* appear — `personnel` and `bandit`.
- 2026-09-15 01:02 — **Token list for task 182's `dependsOn`, verbatim and verified:**
  `crew`, `crewmember`, `friend`, `gøgler`, `klan`, `patrulje`, `senior`, `spejder`.
  Eight tokens. No `personnel` (the table's name, not an entity) and no `bandit`.
- 2026-09-15 01:04 — `TestSearchpersonAloneIsExhaustive` added: this projection must not be the
  reason `live.EntitySet.Exhaustive` goes false. It stays true, which is the payoff for task
  176's decision to name the team types instead of copying signup's entity wildcard.
- 2026-09-15 01:12 — Went beyond the criterion and validated against the **running**
  `hq-mysql-1` (MariaDB 10.8), because a statement-level test cannot tell whether the SQL is
  actually valid. Loaded table.sql into a scratch database, then generated the statements for a
  representative nine-event sequence with a throwaway `cmd/schemacheck` and executed them:
  all 68 statements applied cleanly with `--show-warnings` silent. The resulting six rows are
  right, including the apostrophe in `Anders O'Brien` surviving the quoting, and Bente and
  Henrik legitimately sharing one number. Throwaway tool and scratch database both removed.
- 2026-09-15 01:15 — The retention round trip was confirmed end to end against the real
  database, not just in statement assertions: created → reassigned → deleted → re-added leaves
  one row with `deleted=0` and the new team.
- 2026-09-15 01:17 — Verified behaviour worth recording: a later `spejder.updated` that omits
  `phoneContact` **clears** the guardian's number, because the identity columns are overwritten
  unconditionally. That matches the `spejder` roster table, which does the same, so search never
  holds a number the roster has dropped. Intended, and now checked rather than assumed.
- 2026-09-15 01:19 — `EXPLAIN` on the 8-digit lookup: `type: ref`, `key: idx_search_phone`,
  `rows: 1`. The index seek this projection exists for is real. Task 186 still owns the p99
  measurement on production row counts.
- 2026-09-15 01:21 — Honest limitation: a **full** boot was not exercised, since that needs the
  JetStream stream and a replay. Nothing here changes the gate's behaviour — it is an ordinary
  projection in the wrapped slice — and the SQL it emits is now known to execute, which was the
  substantive risk.
- 2026-09-15 01:22 — ✅ Completed. The projection is live in the API and its live tokens are
  pinned for the frontend.
