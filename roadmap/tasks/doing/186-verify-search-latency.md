# 186 — Verify phone-query latency on production-sized data

**Status:** doing
**Priority:** medium
**Created:** 2026-09-14
**Picked up by:** agent session 2026-09-14
**Started:** 2026-09-14
**Completed:**

## Description

Depends on task 185. The closing check on PRD 014, and the one that validates the decision to
build a projection at all.

PRD 014 chose a `search_person` projection over a query-time `UNION` across six tables
specifically so that a phone lookup is an index seek rather than a scan — on the grounds that
search sits on the critical path of an inbound emergency call. That claim is worth measuring
rather than asserting.

Target: **p99 server time under 50 ms for an 8-digit phone query**, on the production row
count.

Also confirm:

- `EXPLAIN` shows `idx_phone` / `idx_phone_parent` actually used for the 8-digit case. If the
  planner ignores them the projection has bought nothing and the finding is important.
- Name search, which scans by design, is still acceptable at the same row count.
- No search reaches beyond the active year without `includePreviousYears` — a
  privacy-adjacent default that would be easy to erode, so pin it with a test rather than a
  manual check.
- Coverage: for every one of the six sources, a known person is findable by name and by each
  of their stored numbers, and a removed person is still found and flagged. Assert by test,
  not by sampling — the retention requirement is the one most likely to be quietly broken by
  a later "clean up the deleted rows" change.

If the numbers come in far under target, record that too: it is the evidence for whether the
simpler `UNION` approach would have sufficed, which is worth knowing before the next search
feature is scoped.

## Acceptance Criteria

- [ ] p99 for an 8-digit phone query measured and recorded
- [ ] `EXPLAIN` confirms index usage for the equality case
- [ ] Name-search timing recorded at the same row count
- [ ] Test pinning the year default
- [ ] Coverage test across all six sources, including a removed person
- [ ] Findings logged in this task and PRD 014 updated if a conclusion changes

## Progress Log

- 2026-09-14 22:08 — Task created from PRD 014 §10.
