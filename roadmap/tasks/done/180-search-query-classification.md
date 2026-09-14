# 180 — Phone/name query classification and matching

**Status:** done
**Priority:** high
**Created:** 2026-09-14
**Picked up by:** agent session 2026-09-14
**Started:** 2026-09-14
**Completed:** 2026-09-14

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

- [x] Classification: 8 digits, 4 digits, 3 digits, `+45 12 34 56 78`, `Anders`, `Anders 42`
- [x] Query input normalized with `normalizePhone`, and a test that `+45 12 34 56 78` finds
      a person stored as `12 34 56 78`
- [x] 8-digit query is an equality match; `EXPLAIN` confirms the index is used
- [x] 4–7 digits prefix-matches
- [x] Parent number matches and is reported as the parent's
- [x] Name match is case-insensitive including æ/ø/å
- [x] Year is a required argument; a test asserts no cross-year leakage
- [x] Query layer returns deleted rows, flagged — it never filters them out
- [x] Cap enforced and truncation reported
- [x] Table-driven tests for all of the above
- [x] `go build ./...` and `go test ./...` pass

## Progress Log

- 2026-09-14 22:08 — Task created from PRD 014 §10.
- 2026-09-15 02:20 — Picked up. Plan: `search.go` with a `classify` step, a matcher that carries
  the phone role can be decided per row, and `Search` on top of task 179's join.
- 2026-09-15 02:26 — Refactored `query()` to take a `roleOf func(row) PhoneRole` instead of a
  fixed role. The predicate `phoneNormalized = ? OR phoneParentNormalized = ?` matches either
  way, so only a per-row comparison can say whose number it was — and that is the thing the
  operator needs told.
- 2026-09-15 02:30 — Decision on "both": the PRD said an ambiguous query is run as both a phone
  and a name search, and my first rule made that unreachable (letters → name, digits → phone).
  Kept "both" and gave it a real trigger instead: the number is used whenever there are ≥4
  digits, the name additionally whenever there are letters. That is what makes a pasted
  `"tlf. 20 68 00 11"` work — verified against live data, it finds the scout and his contact
  person. A letters-then-stop rule would have found nobody.
- 2026-09-15 02:34 — Renamed the flag `IncludeOtherYears`, not `IncludePreviousYears`: the dev
  stream carries fixtures under year **9999**, and nothing stops a future edition existing
  before the current one closes, so "previous" would be untrue. Flagged for task 184, whose UI
  label needs the same correction.
- 2026-09-15 02:36 — Added `escapeLike`. Without it a query of `%` becomes `LIKE '%%%'` and
  returns the entire population. Not an injection risk — the value is bound — but for a table of
  minors' contact details, "returns everybody" is not something to leave to chance.
- 2026-09-15 02:38 — `TooShort` is a distinct field from an empty result set, so the UI can say
  "keep typing" rather than "ingen match", which an operator would believe.
- 2026-09-15 02:44 — **Verified end to end against the dev database** (run inside `hq-api-1`,
  which has the source mounted and a Go toolchain; MySQL's port is not published). All of it
  behaves: `"20309696"` finds four people who share that number — a senior, her two klan
  contact rows and a patrol contact — which is the "one number, several people" case from PRD
  014 §5 working on real data. `"2030"` prefix-matches the same set. `"Koch"` finds nine.
  `"ab"` and `"%"` both return TooShort with no rows.
- 2026-09-15 02:48 — A scare worth recording: `"Koch"` returned what looked like the same
  spejder five times, which would have meant a join fan-out. It is not — eight *distinct*
  memberIds on one team are all registered as "Rakel A. Koch", the contact person's own name
  filled in for every seat, and the team has exactly one `patrulje` row. The query is correct;
  my throwaway's output simply omitted the ids. Noted in PRD §11, because it means result rows
  must show whatever distinguishing context exists.
- 2026-09-15 02:55 — **Finding, split out as task 187:** 33 of 1459 guardian fields hold *two*
  numbers as free text (`mor 22 79 01 52 eller Far 22110715`). These normalize to 16 digits, so
  **neither parent is findable** — defeating the feature's headline use case for ~3% of rows, in
  exactly the field an inbound call is most likely to match. Not fixed here: it needs a schema
  change and so revisits tasks 174–177, and PRD 014 §11 had already anticipated it as the
  one-row-vs-pair-table question. The data has now answered that question.
- 2026-09-15 02:58 — Deliberately **not** fixed: numbers that are merely damaged
  (`2128151q`, `5O401639`, 7-digit values, `00000000`, `112`). Guessing at a malformed number
  risks matching the wrong person, and in an emergency a confident wrong answer is worse than
  none. Recorded in PRD §11 as a data-quality report the organisers should get instead.
- 2026-09-15 03:00 — ✅ All criteria met. 79 tests/subtests pass; `gofmt`, `go vet`,
  `go build ./...` and the full `go test ./...` clean. Throwaway tool removed.
