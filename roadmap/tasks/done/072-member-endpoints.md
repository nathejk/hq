# 072 — Member endpoints: waiting, resume, override, move

**Status:** done
**Priority:** high
**Created:** 2026-08-17
**Picked up by:** agent session
**Started:** 2026-08-17
**Completed:** 2026-08-17

## Description

From **PRD 006** §6 and §8. The four operator actions this interface owns — everything up
to and including `waiting`, the **self-carrying** boundary. From the car door onwards the
transitions belong to interfaces that do not exist yet.

| Method | Path | Purpose |
|--------|------|---------|
| PUT | `/api/member/:memberId/waiting` | Member wants to leave the race (→ `waiting`). `sosId` **required** |
| PUT | `/api/member/:memberId/racing` | Member carries on under their own steam (→ `racing`). Rejected unless currently `waiting` |
| PUT | `/api/member/:memberId/status` | Override to any valid status **except `finished`**. `sosId` required; minted-and-closed when the caller has none |
| PUT | `/api/member/:memberId/team` | Move to another team (`currentTeamId`). `sosId` **required** |

Each publishes one `spejder` event per affected member, then one summarising `sos` event
when a `sosId` is present (task 071).

**The self-carrying boundary is enforced on the write side.** The resume is valid only
while the member is `waiting`; the command must dirty-check the current `spejderstatus`
row and reject otherwise, with a message the operator can act on ("allerede hentet")
rather than a generic conflict. Hiding the button is not sufficient — the operator's screen
may be a moment stale, which is exactly when a car is accepting the member. If the
acceptance and the cancellation race, **the acceptance wins**: it reflects a member
physically sitting in a car, and the log preserves both attempts in order.

**The 3-member requirement is not enforced here.** No command may reject a withdrawal
because it would put a team below three, and no consumer may auto-collect or auto-move in
response. The member is leaving regardless; refusing to record it would only make the data
wrong. The write side reports strength and records what happened — it never decides.

## Notes

- **`sosId` is required on all four.** Nothing changes a member's status or team without a
  case explaining why (PRD 006 §11 Decisions, 2026-08-17). For the correction path, where
  the operator is on the patrol page and has no case, **the backend mints one and closes it
  immediately** — see task 084, which owns that behaviour and its headline convention.
- **Move** updates `currentTeamId` and leaves `initialTeamId` untouched. A valid target is
  any patrol in the same year still racing (started, `activeMemberCount > 0`) that is not
  the team being left — nothing further, because the destination is agreed by crew in the
  field and the operator is *recording* it, not choosing it. **Always a patrol, never a
  klan:** klaner are not handled through the nødtelefon.
- **Moving is per member.** Two survivors may end up in two different patrols. Do not model
  the command as "move this group to that team".
- **The override excludes `finished`.** `CanFinish()` is true only for `racing`, and a
  member who took a lift must never end up finished. There is no finish producer anywhere
  in hq today, so this is a guard for a flow that arrives later.
- The override is a **separate endpoint** from the withdrawal request rather than one
  parameterised status setter, because they are different acts: one is the normal workflow
  this interface owns, the other is an admission that another interface's handover went
  unrecorded. It is also **not reachable from the case card** — it is the patrol page's
  correction tool (task 084). Two endpoints on two screens keeps it from becoming a
  shortcut, and keeps "how often are we correcting by hand?" answerable (§9 tracks it).
- Member actions live on the member, not nested under `/api/sos`: a member's status is a
  fact about the member. Breach handling — what little of it exists — is a fact about the
  case.
- Note `/api/member/...` uses a noun no existing route uses. That is settled and
  deliberate: `MemberStatus` is a *member* lifecycle, while the events and the live token
  are `spejder`, and with klaner out of scope there is no second population to generalise
  for.
- Commands dirty-check before publishing, so a no-op write publishes nothing and emits no
  live signal — the house pattern.
- Actor comes from task 070.

## Acceptance Criteria

- [x] Four routes registered and wired to commands in the `spejderstatus` package
- [x] All four reject a missing `sosId` (the override's is minted by task 084's handler)
- [x] Resume rejected unless the member is currently `waiting`, with an actionable message
- [x] Override rejects `finished` and any status failing `Valid()`
- [x] Move rejects a target that is not a racing patrol in the same year, and rejects the
      member's current team
- [x] Move rejects a klan as a target — by construction: the target is looked up as a
      patrulje, so a klan id is "ukendt patrulje"
- [x] Each action publishes per-member `spejder` events plus one summarising `sos` event
      when `sosId` is given
- [x] No-op writes publish nothing
- [x] Every endpoint exercised against the running dev stack, including the rejections
- [x] No `nathejk.dk/...` import in the package
- [x] `go build ./... && go vet ./...` and `gofmt -l` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006. Depends on tasks 063, 065, 070, 071.
- 2026-08-17 — Amended before pickup by the decisions of 2026-08-17: `sosId` is now
  **required on all four** endpoints (was optional on the override and the move — the one
  combination nobody would choose, since it made a correction auditable only if the operator
  happened to be in a case). The override also moves off the case card entirely; task 084
  owns its surface and the mint-and-close behaviour.
- 2026-08-17 — Picked up. `spejderstatus/commands.go` (four commands), three
  `Record*` methods on the sos commander, and `cmd/api/member.go` composing them.
- 2026-08-17 — **Hit a structural problem the task did not anticipate: the member command
  must cause an `sos` event, but `spejderstatus` may not import `sos`** — both are written
  to be lifted independently, and task 081's guard enforces it. Resolved by making the
  **handler** the composition point: the member command publishes the member events and
  returns a `Change`/`Move` describing what it did, and the handler enriches that with
  names and publishes the summary. That is what a backend-for-frontend handler is *for*,
  and it keeps both packages movable. Same reasoning for move-target validation, which
  needs the patrulje entity: the handler checks it before calling.
- 2026-08-17 — **The resulting team strength is computed in the command, not read from
  `patrulje.activeMemberCount`.** Not an optimisation: at command time the projection has
  not consumed the event yet, so the column still holds the *old* number — recording it
  would put a stale strength on a permanent timeline entry. `strengthAfter` reads the team
  and substitutes the one member's pending status in memory. `MemberStatusNone` is how
  "leaving the team altogether" is expressed for a move.
- 2026-08-17 — Withdrawal is refused from `transit` onwards (`ErrNotSelfCarrying`): once a
  car has them, a member has not "asked to leave", they have left. Resume distinguishes
  `ErrAlreadyCollected` from `ErrNotWaiting`, because the first is a fact about where the
  scout physically is ("allerede hentet") and the second means the screen was wrong.
- 2026-08-17 — Override implemented **lenient** as decided: `racing → sheltered` is accepted
  with no pickup logged, because that is exactly the out-of-sync case the tool exists for.
  Only `finished` stays shut.
- 2026-08-17 — A failed summary publish is **logged, not returned**. The member events are
  already in the log and cannot be recalled, so telling the operator the withdrawal failed
  when it did not would be worse than losing a timeline line.
- 2026-08-17 — 12 command tests. Writing them turned up that `stream.MutableMessage` needs
  `SetSubject`/`SetTime` as well as the read methods — worth noting for the next fake
  publisher in this repo, since the compiler reveals them one at a time.
- 2026-08-17 — ✅ **Verified end to end against the dev stack — the first time the projection
  has had a producer.** 15 steps on a real 2025 patrol (SISMO, 4 members), scripted so it
  was reproducible:
  - missing `sosId` → rejected with the explanatory message
  - withdrawal → `racing→waiting`, **TeamStrength 3**, one timeline entry
  - same request again → `change: null`, nothing published
  - `/api/member/care` → total 1, `waiting: 1`, `oldestWaitingAt` set
  - resume → `waiting→racing`, **TeamStrength back to 4**
  - override to `finished` → "gennemført kan ikke sættes manuelt"
  - override `racing→sheltered` → accepted (leniency confirmed on live data)
  - resume while sheltered → "allerede hentet"
  - move to a never-started 2026 team → "patruljen er ikke i løbet"
  - move to their own team → "deltageren er allerede i patruljen"
  - timeline carries self-contained summaries including the team name "SISMO"
  - member restored to `racing` and `activeMemberCount` back to **4**
- 2026-08-17 — These events are permanent dev JetStream history, so a 2025 team (past event)
  was used deliberately. The two verification cases were soft-deleted through the API
  afterwards so they do not clutter the case list; the throwaway script was removed.
- 2026-08-17 — ✅ All criteria met. Full `go build`, `go vet`, `gofmt -l`, `go test ./...`
  clean. Moving to done.
