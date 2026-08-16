# 064 — Legacy MemberStatus value mapping

**Status:** done
**Priority:** high
**Created:** 2026-08-17
**Picked up by:** agent session
**Started:** 2026-08-17
**Completed:** 2026-08-17

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

- [x] Mapping implemented (in `go/nathejk/table/spejderstatus/status.go` — see log)
- [x] Every documented legacy value covered, plus the pass-through cases
- [x] Behaviour for unrecognised values decided, documented and tested
- [x] Table-driven tests covering the whole mapping
- [x] Builds and vets clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006.
- 2026-08-17 — Picked up. Implemented **locally** in
  `go/nathejk/table/spejderstatus/` for the same reason as task 063, and lifted with the
  package (task 083). Noting the cost honestly, because PRD 006 §8 argued for shared-go
  specifically so "all consumers share one implementation": until the lift, a *second*
  consumer of these legacy values in another repo would have to duplicate the mapping. In
  practice there is no such consumer — the values only exist in hq's own replayed history
  and nothing in shared-go references them — so the risk is deferred rather than taken.
  Recorded here so task 083 knows the mapping is part of what must move.
- 2026-08-17 — `ParseMemberStatus(string) (types.MemberStatus, bool)` in `status.go`.
  Order matters and is deliberate: **current values are checked with `Valid()` before the
  legacy map is consulted.** `registered`, `waiting` and `transit` are simultaneously
  current values and legacy ones, and the documentation says they are unchanged — checking
  the map first would have worked by luck for these three and broken the moment a future
  status collided with a legacy spelling.
- 2026-08-17 — **Unknown values return `MemberStatusNone` with `ok=false` rather than
  passing through as-is.** Passing through was the alternative and is worse: the row would
  be written first and rejected by `Valid()` only afterwards, i.e. the read model would
  already hold a status nothing can reason about. Refusing at the boundary also lets the
  caller distinguish "not recorded" (a member who has not started — normal) from "recorded
  as something I do not understand" (a bug), which are different problems that would
  otherwise look identical.
- 2026-08-17 — Deliberately did **not** map hq's invented `paid` (from
  `internal/data/member.go:42`). It has never been published as an event and is not a
  status — just a query papering over a missing row. Mapping it would legitimise it and
  leave task 067 with nothing to catch. There is a test asserting it is rejected. `started`
  *is* mapped, because that one is a genuine documented legacy value.
- 2026-08-17 — Four tests. Two are the obvious ones (the full table, case folding); two
  encode consequences that are invisible locally: every mapping **target** must satisfy
  `Valid()` — otherwise the fix reintroduces the silent under-reporting it exists to
  prevent — and no legacy outcome (`emergency`, `hq`, `out`) may map to something
  `CanFinish()` accepts, or replaying old history would hand a finish to a member who was
  driven in.
- 2026-08-17 — ✅ All criteria met. `gofmt`, `go vet` and `go test` clean. Moving to done.
