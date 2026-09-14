# 186 — Verify phone-query latency on production-sized data

**Status:** done
**Priority:** medium
**Created:** 2026-09-14
**Picked up by:** agent session 2026-09-14
**Started:** 2026-09-14
**Completed:** 2026-09-14

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

- [x] p99 for an 8-digit phone query measured and recorded
- [x] `EXPLAIN` confirms index usage for the equality case
- [x] Name-search timing recorded at the same row count
- [x] Test pinning the year default
- [x] Coverage test across all six sources, including a removed person
- [x] Findings logged in this task and PRD 014 updated if a conclusion changes

## Progress Log

- 2026-09-14 22:08 — Task created from PRD 014 §10.
- 2026-09-15 03:50 — Picked up. Measuring against the running dev stack, which holds 4,602 real
  indexed rows for 2025–2026.
- 2026-09-15 03:54 — `EXPLAIN` on the real predicate (an OR across both phone columns, not the
  single-column form task 178 checked): `type: index_merge`, `key:
  idx_search_phone,idx_search_phone_parent`, `Using union(…)`, **5 rows examined out of 4,602**.
  Both indexes are used, so searching the guardian's number alongside the scout's own costs
  nothing.
- 2026-09-15 03:58 — **p99 over HTTP against the running API**, 200 requests per query from
  inside the container (so this is close to server time):

  | query | p50 | p95 | p99 | max |
  |---|---|---|---|---|
  | `20309696` (exact phone) | 0.6ms | 1.6ms | **5.2ms** | 13.7ms |
  | `2030` (prefix) | 0.6ms | 1.3ms | 1.7ms | 3.6ms |
  | `Koch` (name scan) | 2.0ms | 2.6ms | 3.6ms | 10.8ms |
  | `Anders` (common name) | 2.2ms | 3.0ms | 5.3ms | 6.2ms |
  | `tlf. 20 68 00 11` (both) | 2.2ms | 3.2ms | 3.5ms | 4.5ms |

  Target was p99 under 50ms for the exact phone case. **5.2ms — roughly 10x headroom.**
  (`an` measured 0.1ms because it is below the minimum and is never searched, which is worth
  knowing when reading these numbers rather than mistaking it for a fast path.)
- 2026-09-15 04:06 — Went further than the criteria and measured at **16x scale**: copied the
  index into a scratch database and doubled it four times to 73,632 rows. Indexed phone lookup
  **0.27ms p50 / 0.36ms p99** — flat, as an index seek should be. Name scan 4.67ms p50, about 17x
  slower, and that ratio grows linearly with the row count. Scratch database dropped.
- 2026-09-15 04:10 — **The honest conclusion, which cuts against the choice this PRD made.**
  PRD 014 chose a projection over a query-time `UNION` primarily on the grounds that a phone
  lookup must be an index seek because search sits on the critical path of an emergency call.
  The measurements do confirm the index works — it is ~17x faster than a scan at 73k rows and
  stays flat as the table grows. But they also show that **a scan would have met the 50ms target
  comfortably at 16x current scale**, so latency alone did not justify the projection. What does
  justify it is everything else: contact persons becoming findable *at all*, normalisation done
  once at write time rather than in an unindexable expression per query, one place to query
  instead of six, and — discovered along the way — klan contact corrections that no other table
  keeps. Recorded in PRD 014 §8 so the next search feature is scoped on the real reasons.
- 2026-09-15 04:14 — A comment in `table.sql` was **wrong and is now corrected**. It claimed
  `idx_search_name` does not serve the name match because a leading-wildcard LIKE cannot use an
  index. In fact MariaDB uses it as a *covering* index — `type: ref`, `Using index` — seeking on
  the year prefix and scanning name entries inside it rather than touching the table. Still a
  scan, but over an index and considerably cheaper than the comment implied.
- 2026-09-15 04:18 — Extracted `Query.scope()` so the year predicate is testable without a
  database. It is the one default in this feature with a privacy dimension, and "the server never
  widens the scope on its own" deserved to be a pinned assertion rather than a property of an
  inline `if`. Together with the handler's eight-input table in `cmd/api/search_test.go`, the
  default is now covered on both sides.
- 2026-09-15 04:22 — Added `TestEverySourceIsIndexedAndFindable`: all seven kinds from event to
  row, each findable by name *and* number, then the removal case asserting the row is kept and
  flagged. This is the standing guard on the requirement PRD 014 names as most likely to be
  quietly broken later.
- 2026-09-15 04:26 — Unrelated but worth recording, since it could have invalidated a lot of this
  work: `go/go.work` replaces `shared-go` with a **local checkout** at
  `/Users/knj/Development/nathejk/shared-go`, not the module cache version in `go.mod`. Every
  finding in tasks 174–181 was read from the cache. I diffed the parts that mattered — consumed
  subjects for all five shared-go sources, `types.PhoneNumber.Normalize`, and the message shapes —
  and they are identical, so the conclusions hold. gopls reports a phantom "undefined:
  messages.NathejkMemberReassigned" in `consumer.go` from a stale index; `go build`, `go vet` and
  `go test` are all clean.
- 2026-09-15 04:28 — ✅ All criteria met. 89 tests/subtests in the package, whole Go suite green.
