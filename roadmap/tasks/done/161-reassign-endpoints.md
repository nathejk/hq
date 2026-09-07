# 161 — Reassign endpoint and transfer candidates

**Status:** done
**Priority:** high
**Created:** 2026-09-07
**Picked up by:** agent session
**Started:** 2026-09-07
**Completed:** 2026-09-07

## Description

From **PRD 012** §10, phases H1 and H2. Depends on task 160.

Expose the pre-race member transfer:

- `PUT /api/member/:memberId/reassign` — body `{teamId}`, **no `sosId`**.
- `GET /api/member/:memberId/transfer-candidates` — every accepted team annotated with
  whether this member may move there and why not, plus the lines that would move.

Two endpoints rather than one because they answer different questions, but they must share
one definition of every rule: the list that offers a destination and the call that accepts
it cannot be allowed to disagree about whether a team is full.

**Preconditions belong here, not in the domain.** The shared-go command refuses only an
unknown member and a move to the team they are already on; hq owns the team read models and
the Danish wording, so it enforces the rest. Deliberately absent: any lower bound on the
origin's member count — a team below three may not start and the way out is to move its
last members away, so refusing to empty a team would block the case the feature exists for.

## Notes

- New file `go/cmd/api/reassign.go`. Kept out of `member.go` because every handler there
  runs through `memberContext`/`casePolicy`, and this operation deliberately has no case at
  all — putting it in that file would invite somebody to "tidy" it onto the same helper.
- Named `reassign`, not `/team`: `PUT /api/member/:memberId/team` is the race-time move and
  keeps the member on their starting roster. Sharing a path would let the two be confused,
  and confusing them makes `initialTeamId` a lie.
- **`maxPatruljeMemberCount = 7` is now enforced, not just displayed.** The same 7 that
  `TeamConfig` serves the SPA. First place the server actually applies it.
- **The member cap is counted from the roster, not `patrulje.memberCount`.** That column is
  frozen at who started and is 0 before that — so it cannot answer "is this team full" for
  a team that has not started, which is every team this endpoint deals with. Uses
  `patrulje.GetAll`'s recomputed count instead.
- The origin is read from the roster (`spejder.RosterReader`) rather than taken from the
  request, for the same reason the shared-go command does: it is the only thing that knows,
  and a stale browser tab must not be able to name a team the member already left.
- `data.Models` gained `Roster` and `Order` became `data.OrderInterface` (the order read
  API plus `MemberLineReader`) so the **preview and the transfer read the same lines**. A
  second definition of "what has been paid for this member" is exactly the drift worth
  avoiding here.
- Three distinct refusals — `ikke optaget i løbet`, `startet`, `fuld` — rather than one
  generic 422, because they are three different things for an operator to do something
  about.
- Ineligible teams are **returned, not filtered**: an operator looking for a specific
  patrol needs to see that it is full or started, not wonder whether they misremembered.
- hq's `patrulje.GetByID` returns `nathejk.dk/nathejk/table`.ErrRecordNotFound while the
  roster returns shared-go's `tables.ErrRecordNotFound`. Two distinct sentinels with the
  same name and message; the file imports both under separate aliases. Worth knowing before
  writing an `errors.Is` against the wrong one — it fails silently as a 500.

## Acceptance Criteria

- [x] `PUT /api/member/:memberId/reassign` moves the member and their money
- [x] No `sosId` required or accepted
- [x] `GET /api/member/:memberId/transfer-candidates` returns all numbered teams with
      `eligible` + `reason`, plus `lines`, `amount` and `maxMemberCount`
- [x] Origin refused when started, pointing at the nødtelefon's move instead
- [x] Destination refused when unnumbered, started, or full — each with its own message
- [x] No minimum-member check on the origin
- [x] `ErrSameTeam` → 422 with a field message; `ErrMemberNotFound` → 404
- [x] OpenAPI annotations on both endpoints
- [x] Build, vet, fmt and tests green

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-09-07 — Implemented in `cmd/api/reassign.go`, routed beside the member lifecycle
  block with a comment on why it is *not* part of it. Added `data.Models.Roster` and
  `data.OrderInterface` so the candidate preview and the transfer command share one read of
  the member's paid lines. Enforced the 7-member cap from the recomputed roster count
  rather than the frozen column.
- 2026-09-07 — No handler tests written. The two handlers are mostly composition over read
  models that the existing test fakes do not cover, and the rules they enforce are worth
  testing — recorded as a gap in task 163 rather than left implied.
