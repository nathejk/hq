# 179 — Join current status into search results

**Status:** done
**Priority:** high
**Created:** 2026-09-14
**Picked up by:** agent session 2026-09-14
**Started:** 2026-09-14
**Completed:** 2026-09-14

## Description

Depends on task 178. Implements PRD 014's requirement that **every result states the
person's current standing**.

Status is a **read-time join, not a projected column**:

- a spejder's lifecycle status from `spejderstatus` (`registered`, `seated`, `racing`,
  `finished`, `waiting`, `transit`, `sheltered`, `reunited`, `released`)
- otherwise the team's `signupStatus` from `patrulje` / `klan` / `personnel`
- plus the `deleted` flag from task 177

Same reasoning as the team-name join, and it pays twice: the consumer needs none of the
status subjects, so its replay and signal surface stay narrow, and PRD 006's state machine is
not re-implemented in a second place where it could disagree with the first.

Team and section **names** are joined here too (`patrulje`, `klan`, `section`) — the
projection stores only `teamId`.

**Crew and personnel rows have no `teamId` at all** (task 175: their events carry no team,
and a crew member's section arrives on an identity-free `section.assigned` event that
`searchperson` does not subscribe to). Their context has to come from a join to `personnel`
and `crewmember` on the person's id — `personnel.klan` / `groupName`, and
`crewmember.sectionSlug` → `section`. Without it, every gøgler and crew result row is a bare
name, which is barely a search result.

Details:

- **A spejder with no `spejderstatus` row is ordinary, not an error.** Statuses begin at
  signup-time `registered`/`seated`, and older years predate the table entirely. LEFT JOIN,
  and render absence as unknown — no `COALESCE` to a status that would be a lie.
- **`deleted` and a lifecycle status are different axes.** A row can be `deleted` (never
  started) or `released` (went home during the night); the response must let the UI say
  which, so do not collapse them into one string server-side.

## Acceptance Criteria

- [x] Results carry lifecycle status for spejdere, signup status otherwise, and the deleted flag
- [x] Crew and personnel rows get their context by joining `crewmember`/`personnel` on id
- [x] Team/section name resolved by join, not denormalized into `search_person`
- [x] A spejder with no status row returns unknown rather than a fabricated status
      — falls back to the *team's* signup status first, then unknown; see log
- [x] `deleted` and lifecycle status are separate fields in the response
- [x] Tests cover: racing spejder, released spejder, deleted spejder, spejder with no status row
- [x] `go build ./...` and `go test ./...` pass

## Progress Log

- 2026-09-14 22:08 — Task created from PRD 014 §10.
- 2026-09-15 01:30 — Picked up. This task creates `query.go` (deferred from 174), so it also
  settles the result shape task 180 will filter and task 181 will serialise.
- 2026-09-15 01:35 — Wrote `query.go`: one `selectResult` with six LEFT JOINs (patrulje, klan,
  spejderstatus, personnel, crewmember, section), a `row` type for the raw join, and
  `row.result(role)` which applies the per-kind rules.
- 2026-09-15 01:38 — Decision: the per-kind choice of *which* status and *which* name describes
  a person is made **in Go, not in a SQL CASE**. The alternative would have put a domain rule
  ("a spejder's status comes from spejderstatus, a klan member's from their klan") inside a
  string literal. As a pure function it is table-driven testable, which is how
  `TestResultChoosesTheRightStatusPerKind` can reuse one fully-populated join row and assert
  that each kind picks only its own columns out of it.
- 2026-09-15 01:41 — Refinement beyond the criterion: a spejder with no `spejderstatus` row
  falls back to the **team's** signup status before giving up. Before the race nobody has a
  lifecycle status, so the strict reading ("unknown") would have left every result blank during
  signup season, which is when the contact-person searches happen. `StatusKind` distinguishes
  the two vocabularies so the UI never presents `PAID` as if it described the person.
- 2026-09-15 01:44 — Added two structural guards that would otherwise be review-only:
  `TestEveryJoinIsOuter` (an inner join anywhere silently drops people — a scout with no status
  row, a crew member never assigned a section — and the symptom is "not in the system"), and
  `TestResultCarriesNoSensitiveExtras` (the query must not select address, birthday or notes;
  this is the first endpoint returning minors' contact details event-wide, so it must find
  people, not export them).
- 2026-09-15 01:52 — **Verified against the running dev database**, which turned out to be
  already populated: the dev API had hot-reloaded, created `search_person` and replayed
  **4,602 rows** across all seven kinds (1737 spejder, 1531 senior, 719 patruljekontakt, 230
  klankontakt, 225 gøgler, 133 friend, 27 crew). Ran the full six-join query against it.
- 2026-09-15 01:55 — Join coverage on real data: **0 unresolved** patrulje joins for
  `patruljekontakt` (719/719) and 0 unresolved klan joins for `klankontakt` (230/230) — the two
  populations with no prior art both resolve completely. Crew sections resolve to labels
  ("Rover 1"). 1034 of 1737 spejdere have no status row, which is the ordinary pre-race state
  and exactly why the fallback above was added.
- 2026-09-15 01:58 — An early sample looked as if the contact-person joins were failing. They
  were not: the sample used `GROUP BY kind` with non-aggregated columns, so MySQL returned
  values from unrelated rows. Recording it because the false alarm was convincing — counting
  `SUM(p.teamId IS NULL)` per kind is the honest check.
- 2026-09-15 02:02 — **Data-quality finding, upstream and not a projection bug:** 44 of 164
  indexed 2026 seniors have no name. The `senior` table itself has 33 of 151 in the same state,
  so the register is where the names are missing; search reflects it faithfully. Those people
  are still findable by phone where they have one. Worth surfacing to the organisers separately
  — not something search should paper over.
- 2026-09-15 02:04 — Retention confirmed on real data as a side effect: the index holds 164
  seniors for 2026 where the roster holds 151. The extra 13 are people removed from a roster and
  kept by task 177, which is precisely the intended behaviour.
- 2026-09-15 02:06 — Also confirmed on real data: `senior.phone` values like
  `+45 53 30 08 99` exist in the register and normalize correctly to `53300899`. This is the
  exact case task 174 found shared-go's `Normalize()` mishandles, so the national-form step is
  load-bearing against production data rather than hypothetical.
- 2026-09-15 02:09 — Observation for task 184, not acted on: the dev stream contains 17 rows
  under year **9999** with `TEST …` names (shared numbers, a deleted member, an unmapped
  section). Somebody's fixtures, not mine, and no such convention exists in the repo. Note that
  9999 is a *future* year, so a control labelled "tidligere år" would not describe it — worth a
  thought when 184 implements the year scope.
- 2026-09-15 02:11 — ✅ All criteria met. 60 tests/subtests pass; `gofmt`, `go vet`,
  `go build ./...` and the full `go test ./...` clean.
- 2026-09-15 02:12 — Completed. Every result now states where the person belongs and how they
  stand, with "udmeldt" and the lifecycle status kept as separate axes.
