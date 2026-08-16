# 064 — shared-go: legacy MemberStatus value mapping

**Status:** open
**Priority:** high
**Created:** 2026-08-17
**Picked up by:**
**Started:**
**Completed:**

## Description

From **PRD 006** §8. Cross-repo work in `nathejk/shared-go`.

`types/member.go` **documents** the mapping from superseded status values but does not
implement it:

```
REGISTERED, STARTED   → registered, racing
active                → racing
emergency             → waiting
waiting, transit      → unchanged
hq                    → sheltered
out                   → released
```

Implement it as shared code next to the constants, so every consumer normalises
identically. Add a parse/normalise function (e.g. `ParseMemberStatus(string)
MemberStatus` or `MemberStatus.Normalised()`) with table-driven tests.

**Why this cannot be skipped:** the `spejderstatus` projection is rebuilt from full
JetStream history on every API restart, so replay *will* encounter these values. Left
un-normalised, every `InOurCare()` and `CanFinish()` check silently under-reports — a
member sitting in `hq` is in our care but would not be counted, which is the exact
failure the in-our-care number exists to prevent.

## Notes

- Belongs in shared-go rather than hq precisely so a second consumer cannot drift: this
  is the same reasoning that keeps `Valid()` / `CanFinish()` / `InOurCare()` there.
- Note there are also two values invented by hq's own SQL — `started` and `paid`, from
  `internal/data/member.go:42` — which are **not** in the documented mapping and are not
  real statuses. Do **not** add them here; they are deleted in task 067.
- Unknown values: decide and document whether they map to `MemberStatusNone` or are
  returned as-is with `Valid()` reporting false. The projection needs to be able to tell
  "not recorded" from "recorded as something I don't understand".

## Acceptance Criteria

- [ ] Mapping implemented in `shared-go/types/member.go` (or a sibling file in `types`)
- [ ] Every documented legacy value covered, plus the pass-through cases
- [ ] Behaviour for unrecognised values decided, documented and tested
- [ ] Table-driven tests covering the whole mapping
- [ ] shared-go builds and vets clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006.
