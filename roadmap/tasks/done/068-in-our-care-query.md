# 068 — In-our-care query

**Status:** done
**Priority:** high
**Created:** 2026-08-17
**Picked up by:** agent session
**Started:** 2026-08-17
**Completed:** 2026-08-17

## Description

From **PRD 006** §6 and §8. The read side of the number the whole feature is built
around: **how many members are currently in our care** — the count that has to reach zero
before the organisers can go home.

In the `spejderstatus` package, add a query returning, for the current year:

- counts per status over `MemberStatus.InOurCare()` — `waiting`, `transit`, `sheltered`
- the total
- the **oldest `waiting` timestamp**, which drives the alarm

Served by `GET /api/member/care` (PRD 006 §8 endpoint table).

## Notes

- Use `InOurCare()` from shared-go. Do **not** spell the three statuses out at the call
  site — that is exactly what the helper exists to prevent, and a fourth in-care status
  added later must not need this query edited.
- The oldest `waiting` timestamp rather than a count of stale ones: the threshold is
  configuration (open question in PRD 006 §11) and may change, so the query returns the
  fact and the caller applies the rule.
- **Correctness matters more than availability here.** A plausible-but-wrong count of the
  people we are responsible for is the worst failure this tool could have. Until PRD 005's
  boot gate ships, a post-restart window can serve a partially rebuilt read model — the
  honest interim behaviour is for the screen to say it cannot reach the server rather than
  to render a number (PRD 006 §6, Non-Functional).
- Year-scoped like everything else.

## Acceptance Criteria

- [x] Query in the `spejderstatus` package returning per-status counts, total and oldest
      `waiting` timestamp
- [x] Status set derived from `InOurCare()`, not enumerated at the call site
- [x] `GET /api/member/care` returns it, year-scoped from `X-YearSlug`
- [x] Returns zeroes rather than an error when nobody is in care
- [x] No `nathejk.dk/...` import in the package
- [x] Exercised against the dev stack
- [x] `go build ./... && go vet ./...` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006. Depends on task 065.
- 2026-08-17 — Picked up. `querier.InOurCare` + `Care` type + `GET /api/member/care`
  (`cmd/api/member.go`), wired through `data.Models.SpejderStatus`.
- 2026-08-17 — The status set is built by `InOurCareStatuses()`, which filters a local
  `allMemberStatuses` list through shared-go's `InOurCare()`. **shared-go cannot enumerate
  its own statuses** — `Valid()` is a switch — so the list has to exist somewhere; keeping
  it here and *deriving* the in-care subset means a fourth in-care state added upstream
  starts counting with no query edit. Two tests guard it: the derived set must agree with
  the helper for every status, and every listed status must be `Valid()`. Neither can catch
  an *omission* from the list, so it carries a comment saying so plus a count assertion as
  the nearest available substitute.
- 2026-08-17 — `ByStatus` includes every in-care status **at zero** rather than omitting
  empties. A breakdown that dropped them would make the counter flicker between three rows
  and one as the night went on, and "transit: 0" is information: no car is carrying anybody.
- 2026-08-17 — Returns `oldestWaitingAt` rather than a "somebody waited too long" boolean.
  The threshold is configuration and still unsettled (task 082), so the query returns the
  fact and the rule lives where it can change without redeploying this endpoint.
- 2026-08-17 — Measured only over `waiting`, not all in-care statuses: a member in a car or
  asleep at HQ is accounted for and their patrol has stopped waiting for them.
- 2026-08-17 — ✅ Verified on the dev stack. Empty case first (total 0, all three statuses
  present at 0, `oldestWaitingAt: null`) for both 2025 and 2026. Then probed the non-zero
  path with three temporary rows inserted **directly into MySQL rather than published to
  JetStream** — the stream retains, and a fabricated member would have replayed on every
  restart forever. Got total 3, `waiting: 2`, `transit: 1`, and `MIN` correctly choosing
  21:00 over 22:30.
- 2026-08-17 — That first probe did not actually prove the timestamp was **waiting-only**,
  because the transit row happened to be later than both waiting rows. Moved it to 18:00 —
  earlier than either — and re-checked: still 21:00, so the filter is right. Worth the extra
  step; the assertion would have passed just as happily with a bug in it. Probe rows deleted,
  0 left.
- 2026-08-17 — ✅ All criteria met. Full `go build`, `go vet`, `gofmt -l`, `go test ./...`
  clean. Moving to done.
