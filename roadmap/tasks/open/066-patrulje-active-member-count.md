# 066 — patrulje.activeMemberCount, maintained by the member projection

**Status:** open
**Priority:** high
**Created:** 2026-08-17
**Picked up by:**
**Started:**
**Completed:**

## Description

From **PRD 006** §8 ("`activeMemberCount` is owned by the member projection") and §11
Decisions.

Add `activeMemberCount` to the `patrulje` table: the number of members whose
`currentTeamId` is the team and whose status is `racing`. It is the **one number** behind
both derived facts the PRD needs — team strength (`< minimum` means in breach) and
discontinuation (`== 0` means udgået) — so neither is ever computed ad hoc by a caller.

**There is no `team.discontinued` event and no discontinued flag.** The count is the
fact. Nothing may set discontinuation independently of membership, and no operator action
discontinues a team directly — they retire or move its members.

**The member projection writes it**, in the same `HandleMessage` that writes the member's
own row, by recomputing:

```sql
UPDATE patrulje SET activeMemberCount = (
  SELECT COUNT(*) FROM spejderstatus
  WHERE year = ? AND currentTeamId = ? AND status = 'racing'
) WHERE teamId = ?
```

Moving a member touches **two** teams and both must be recomputed.

## Notes

- **Why the member projection and not the `patrulje` consumer**, even though it means one
  projection writing another's table: if `patrulje` maintained the count by reading
  `spejderstatus`, it would be **racing its sibling consumer over the same message**. The
  mux hands the event to both with no ordering guarantee between them, so the recompute
  could read the row before the member projection has written it and land a count that is
  quietly one out. A count of who is still on the route that is plausibly wrong is worse
  than no count.
- **Recompute, do not increment.** A `±1` needs the member's previous status and makes
  replay order-dependent; a recompute converges regardless of arrival order and needs
  nothing on the event.
- `patrulje.*.started` still sets `memberCount` as it does today — that is the frozen
  count of who started and stays. `activeMemberCount` is the live one.
- **Expose it wherever the team is served**, including the patrulje **list** payload: the
  move picker in task 077 filters the SPA's live `patrulje:list` on
  `activeMemberCount > 0` rather than calling a candidate endpoint, so the column has to
  be on the list.
- Live updates: the count changes only in response to a member event, so the `spejder`
  token already announces it. Frontend consumers depend on `spejder`, **not** `patrulje`
  — the intuitive choice is wrong and fails silently (PRD 004 §12).
- `CREATE TABLE IF NOT EXISTS` will not add the column to an existing table; drop it in
  dev.

## Acceptance Criteria

- [ ] `activeMemberCount` column on `patrulje`
- [ ] Recomputed by the member projection on every membership or status change
- [ ] A move recomputes **both** the origin and destination teams
- [ ] Correct after a full replay in any message order (test)
- [ ] Zero for a team whose last racing member left; non-zero again when one is moved in
- [ ] Exposed on both the single-team and the list payloads
- [ ] No `patrulje.discontinued` event, no discontinued column, no way to set it directly
- [ ] `go build ./... && go vet ./...` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006. Depends on task 065.
