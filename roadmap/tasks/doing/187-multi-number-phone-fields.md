# 187 — Find people whose number shares a field with another number

**Status:** doing
**Priority:** high
**Created:** 2026-09-14
**Picked up by:** agent session 2026-09-15
**Started:** 2026-09-15
**Completed:**

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

- [ ] A person whose guardian field holds two numbers is findable by **both**
- [ ] The role shown for such a hit is honest about not knowing which parent it is
- [ ] PRD 014 §11's one-row-vs-pair-table question is resolved and the PRD updated
- [ ] Numbers that are merely malformed are left unmatched rather than guessed at, with a test
      naming the values from the description
- [ ] Re-measure after the change: how many rows remain unfindable by any number
- [ ] `go build ./...` and `go test ./...` pass

## Progress Log

- 2026-09-14 02:40 — Task created from a finding while verifying task 180 against the dev
  database. Split out rather than folded into 180 because it changes the schema and therefore
  revisits tasks 174–177.
