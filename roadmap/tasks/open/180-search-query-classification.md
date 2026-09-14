# 180 — Phone/name query classification and matching

**Status:** open
**Priority:** high
**Created:** 2026-09-14
**Picked up by:**
**Started:**
**Completed:**

## Description

Depends on task 179. The query logic, in the `searchperson` query layer, with table-driven
tests. Separated from the endpoint (task 181) because this is where the real behaviour lives
and it is worth testing without HTTP in the way.

**Classification.** A query is a phone query when its normalized digits number at least 4 and
it contains no letters; otherwise a name query. A query that could plausibly be both is run
as both and the results merged.

**Phone matching.**

**Normalize the operator's input with `searchperson`'s own `normalizePhone`, not with
`types.PhoneNumber.Normalize`.** Task 174 found that shared-go's function keeps every
digit, so `+45 12 34 56 78` and `12 34 56 78` reduce to different strings; `normalizePhone`
adds the national-form step that makes them equal. Using a different function on the query
side than the write side would produce a mismatch that looks exactly like "this person is
not in the system".

- exactly 8 digits → equality on `phoneNormalized` / `phoneParentNormalized`. This is the
  case the whole projection exists for, and it must be an index seek.
- 4–7 digits → prefix match, for a number read back badly over the phone.
- Match against a scout's own number **and** their parent's, and record which one hit — the
  operator must know who will answer.

**Name matching.** Case-insensitive substring, including for æ/ø/å. This scans; that is
accepted (PRD 014 §8) and no index can serve a leading wildcard anyway.

**Year scope.** Active year only, unless the caller opts in (task 184). The query layer must
take the year as a required argument, not default it — a widened scope must be impossible to
reach by omission.

**Ordering and cap.** Fixed, explainable ordering (exact phone match first, then current
before departed, then name). Cap at 50 and report whether the cap was hit, so the UI can say
so rather than silently truncating.

## Acceptance Criteria

- [ ] Classification: 8 digits, 4 digits, 3 digits, `+45 12 34 56 78`, `Anders`, `Anders 42`
- [ ] Query input normalized with `normalizePhone`, and a test that `+45 12 34 56 78` finds
      a person stored as `12 34 56 78`
- [ ] 8-digit query is an equality match; `EXPLAIN` confirms the index is used
- [ ] 4–7 digits prefix-matches
- [ ] Parent number matches and is reported as the parent's
- [ ] Name match is case-insensitive including æ/ø/å
- [ ] Year is a required argument; a test asserts no cross-year leakage
- [ ] Query layer returns deleted rows, flagged — it never filters them out (deferred here
      from task 177, which had no query layer to put it in)
- [ ] Cap enforced and truncation reported
- [ ] Table-driven tests for all of the above
- [ ] `go build ./...` and `go test ./...` pass

## Progress Log

- 2026-09-14 22:08 — Task created from PRD 014 §10.
