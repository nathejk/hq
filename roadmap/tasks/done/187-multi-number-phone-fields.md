# 187 — Find people whose number shares a field with another number

**Status:** done
**Priority:** high
**Created:** 2026-09-14
**Picked up by:** agent session 2026-09-15
**Started:** 2026-09-15
**Completed:** 2026-09-15

## Description

Discovered while verifying task 180 against the dev database, and it defeats PRD 014's headline
use case for a measurable slice of the population.

A guardian field frequently holds **two** numbers, as free text:

```
mor 22 79 01 52 eller Far 22110715
Mor: 24281097 eller Far: 22239313
```

`normalizePhone` strips the non-digits and yields a 16-digit string, so **neither parent is
findable**. The whole premise of the feature is "an unknown number rings — who is it?", and this
is precisely the case where the caller is a parent.

Measured on the 2026 dev data: **33 of 1459** guardian fields and **11** own-phone fields
normalize to something that is neither 8 nor 10 digits. Roughly 3% of rows, concentrated in
exactly the field an inbound call is most likely to match.

The same measurement turned up two other causes, which are **not** this task:

- Typos and OCR-ish damage: `2128151q`, `5O401639` (letter O for zero), 7-digit numbers.
- Junk: `00000000`, `112`, `45`.
- One doubled value, `+452244565222445652`, which looks like a number pasted twice.

Those should be left alone. Guessing at a malformed number risks matching the *wrong* person,
and in an emergency a confident wrong answer is worse than no answer. A data-quality report for
the organisers is the right home for them — see Open Questions in PRD 014.

Splitting a field that contains two well-formed 8-digit numbers is different: it is mechanical
and unambiguous, and both halves are real numbers belonging to real people.

## Approach

**Superseded by a simpler decision from the maintainer (2026-09-15): do not split the field at
all.** Collapse it into one 16-digit string, as normalization already does, and match phone
queries with `LIKE '%needle%'` so either half is found. Return the value as entered, which tells
the operator which parent is which better than any label we could derive. See the progress log
for what this cost and what it retired.

The original plan is kept below for the record, because the reasoning about a pair table is still
the right reasoning if substring matching ever stops being enough.

---

PRD 014 §11 already asks whether `search_person` should hold one row per person or whether a
`search_person_phone` pair table keyed `(year, phoneNormalized, kind, id, role)` would be
better, noting that the current schema "does not extend to a third number". **The data now
answers that question: there is a third number.** So this task is where that open question gets
resolved, and the pair table is the likely answer:

- Any number of numbers per person, each with its own role (`own`, `parent`, `parent2`,
  `contact`).
- One index on a single column serves every lookup, instead of one per phone column.
- The `role` travels with the number, so `PhoneRole` stops being derived by comparing the needle
  against two columns in `classification.roleOf`.

Consider carefully whether to keep the denormalized columns on `search_person` as well. Two
sources of truth for one number is how they drift.

Splitting rule: after normalization, if the digits are exactly 16 and both halves are plausible
Danish numbers, treat them as two numbers. Do not attempt three, and do not attempt to
associate the labels ("mor", "Far") with the halves — the *order* in the free text is not
reliable enough to promise which parent is which, and "en forælder" is an honest label where
"mor" would be a guess.

## Acceptance Criteria

- [x] A person whose guardian field holds two numbers is findable by **both**
- [x] The role shown for such a hit is honest about not knowing which parent it is — solved better
      than intended: the *entered text* is returned, so the operator reads "mor … eller Far …"
      and needs no label from us
- [x] ~~PRD 014 §11's one-row-vs-pair-table question is resolved~~ — resolved as **neither**: one
      row, one collapsed value, substring matching
- [x] Numbers that are merely malformed are left unmatched rather than guessed at, with a test
      naming the values from the description
- [x] Re-measure after the change: how many rows remain unfindable by any number
- [x] `go build ./...` and `go test ./...` pass

## Progress Log

- 2026-09-14 02:40 — Task created from a finding while verifying task 180 against the dev
  database. Split out rather than folded into 180 because it changes the schema and therefore
  revisits tasks 174–177.
- 2026-09-15 09:05 — Picked up with a **changed approach, from the maintainer**: do not split the
  field. Keep the collapsed 16-digit value and match with `LIKE '%needle%'`, which finds either
  half, and show the entered value. Much simpler than the pair table this task proposed — no
  schema change, so tasks 174–177 are untouched — and it is *better* on the point I had thought
  was the hard part: I was going to label such a hit "en forælder" because the order of "mor" and
  "Far" in free text is not reliable enough to promise which is which. Returning the text sidesteps
  the problem entirely — the operator reads it and knows.
- 2026-09-15 09:12 — Implemented: one substring branch in `predicate()` for both phone columns,
  replacing the equality/prefix split. `Exact()` no longer decides how matching is done and is now
  only a readability distinction; kept, with its doc corrected, because "eight digits identifies a
  person, four is part of one" still means something.
- 2026-09-15 09:16 — Added on my own initiative, and I think it is needed: **exact matches now sort
  above substring matches.** Substring matching cannot avoid false positives — an 8-digit needle can
  match across the join between two collapsed numbers, and a 4-digit fragment matches mid-number —
  so without this the person whose number was actually dialled could sit *below* a coincidence.
  Mitigating by ordering rather than by exclusion follows the instruction to return everything
  found. Pinned by `TestExactMatchesSortAboveSubstringMatches`.
- 2026-09-15 09:20 — **`roleOf` had to change too, and this would have been a silent bug.** It
  compared the needle by equality/prefix; against a collapsed 16-digit field neither holds, so every
  such hit would have fallen through to `nameRole` and been labelled as the person's *own* number —
  precisely the lie that function exists to prevent. Now containment, matching the predicate.
- 2026-09-15 09:28 — **Verified against the live register.** All four numbers inside the two real
  two-number fields now find their scout over HTTP, each labelled `phoneRole: parent`, and the
  returned `phone` is the entered text (`mor 22 79 01 52 eller Far 22110715`). The doubled value
  `+452244565222445652` is findable by `22445652` too — no rule of its own, same mechanism.
- 2026-09-15 09:34 — **Re-measured, and the cost is real but affordable.** EXPLAIN is now
  `type: ALL`: a leading wildcard across two columns cannot use an index, so the phone indexes are
  no longer used for the lookup. Over HTTP: **4.0ms p99 at 4.6k rows** (was 5.2ms with the index —
  no worse in practice at this size). At 73k rows, sixteen times current data: **13.6ms p50 /
  17.3ms p99**, against a 50ms target. So headroom falls from ~140x to ~3x and now grows linearly
  with the table. 4,602 rows is about two years of Nathejk, so 73k is roughly thirty — fine for a
  very long time, and the number to remember if this ever feels slow.
- 2026-09-15 09:38 — Decision: **kept the two phone indexes** even though they no longer serve the
  lookup. The write cost is trivial, and they are exactly what an exact-match fast path would need
  if the scan ever does get too slow (indexed equality first, scan only on no results). Dropping
  them means a DROP INDEX migration against production for no measured gain. Documented in
  table.sql so nobody reads their presence as a claim that lookups are indexed.
- 2026-09-15 09:42 — Corrected three now-false comments rather than leaving them: the package doc's
  "no index can serve REPLACE(…)" justification for the whole projection (speed is no longer on the
  list — the projection earns its place on findability and one-place normalisation), and both index
  comments in table.sql.
- 2026-09-15 09:50 — **Found and fixed a defect this change introduced in the frontend.** The
  results table rendered `<a :href="tel:${data.phone}">`, which for a two-number field produces
  `tel:mor 22 79 01 52 eller Far 22110715` — a link that dials nothing while looking as though it
  should. Added `dialable(phone)`, which offers a link only where the field resolves to exactly one
  number and renders text otherwise. Refusing to link is the honest outcome: we cannot know which
  of two numbers the operator meant to ring.
- 2026-09-15 09:56 — **A second divergence found by that test:** the frontend's `queryDigits`
  claimed to mirror the server's `normalizePhone` but did not strip the leading `00`, so
  `0045 12345678` normalized to 12 digits client-side and 8 server-side. Harmless for matching
  (the server does the real work) but the comment was a lie and the two would have drifted further.
  Fixed to match Go step for step.
- 2026-09-15 10:00 — Re-measured what remains unfindable, per the criterion: the collapsed
  two-number fields (33 rows) and the doubled value are now all findable. What remains are the
  genuinely damaged values — `2128151q`, `5O401639` (letter O for zero), 7-digit numbers,
  `00000000`, `112`, `45` — which stay unfindable **by design**. A `5O401639` normalizes to seven
  digits, so the correct 8-digit number will not match it. Guessing would risk pointing an operator
  at the wrong person mid-emergency; these belong in a data-quality report to the organisers, which
  PRD 014 §11 records.
- 2026-09-15 10:04 — ✅ All criteria met. Go: 92 tests/subtests in the package, full suite green,
  `gofmt`/`go vet` clean. Frontend: 346 tests pass, `npm run build` clean.
- 2026-09-15 10:05 — Completed. Both parents in a shared field are findable, and the operator sees
  the text that tells them which is which.
