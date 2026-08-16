# 068 — In-our-care query

**Status:** open
**Priority:** high
**Created:** 2026-08-17
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] Query in the `spejderstatus` package returning per-status counts, total and oldest
      `waiting` timestamp
- [ ] Status set derived from `InOurCare()`, not enumerated at the call site
- [ ] `GET /api/member/care` returns it, year-scoped from `X-YearSlug`
- [ ] Returns zeroes rather than an error when nobody is in care
- [ ] No `nathejk.dk/...` import in the package
- [ ] Exercised against the dev stack
- [ ] `go build ./... && go vet ./...` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006. Depends on task 065.
