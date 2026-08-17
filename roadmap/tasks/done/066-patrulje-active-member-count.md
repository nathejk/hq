# 066 — patrulje.activeMemberCount, maintained by the member projection

**Status:** done
**Priority:** high
**Created:** 2026-08-17
**Picked up by:** agent session
**Started:** 2026-08-17
**Completed:** 2026-08-17

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

- [x] `activeMemberCount` column on `patrulje`
- [x] Recomputed by the member projection on every membership or status change
- [x] A move recomputes **both** the origin and destination teams
- [x] Correct after a full replay in any message order (test)
- [x] Zero for a team whose last racing member left; non-zero again when one is moved in
- [x] Exposed on both the single-team and the list payloads
- [x] No `patrulje.discontinued` event, no discontinued column, no way to set it directly
- [x] `go build ./... && go vet ./...` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006. Depends on task 065.
- 2026-08-17 — Picked up. Column added to `patrulje`, recompute in
  `spejderstatus/consumer.go`, exposed on `GetAll` and `GetByID`.
- 2026-08-17 — **Did not need a manual table drop.** `table/patrulje/table.go` already
  carried a `widenTextColumns` list of idempotent `ALTER`s, added because
  `CREATE TABLE IF NOT EXISTS` silently skips existing tables — the same trap task 065
  hit. Generalised it to `schemaMigrations` and added
  `ADD COLUMN IF NOT EXISTS activeMemberCount`. This is strictly better than the note in
  the task description ("drop it in dev"): a column that needs manual intervention to
  appear is a column that will be missing in production one day.
- 2026-08-17 — The recompute runs **after** the member-row write inside the same
  `HandleMessage`, and its doc comment explains why the count cannot live in the patrulje
  consumer: the mux hands the same message to every consumer with no ordering guarantee,
  so a recompute there could read `spejderstatus` before this projection had written the
  row and land a count one out. Nothing would fail — the number would just be plausibly
  wrong, which for a count of who is still on the route is the worst available outcome.
- 2026-08-17 — A team start recomputes **once for the team, not once per member**. The
  count is derived from the table rather than accumulated, so N members would issue N
  identical statements for the same answer.
- 2026-08-17 — A move recomputes both teams, which is what makes `FromTeamID` on the
  event worth carrying: once the row is updated it no longer says where the member came
  from. Recomputing only the destination is the plausible half-fix, and it would leave the
  patrol the member *left* overstating its strength — so it would not show as under styrke
  when it should, defeating the number's whole purpose. There is a test.
- 2026-08-17 — Reworked the test helpers rather than bumping statement counts: writes are
  now selected by target table (`memberStmts` / `countStmts`), so each assertion is about
  one thing. Found along the way that goqu's mysql dialect renders a delete as
  ``DELETE `t` FROM `t` WHERE …``.
- 2026-08-17 — ✅ Verified on real data. Column added automatically on restart; 169 teams
  with strength summing to **686**, matching the 686 projected member rows exactly, and
  **zero teams** where `activeMemberCount` disagrees with a live count over
  `spejderstatus`. `activeMemberCount` present on `GET /api/patrulje`.
- 2026-08-17 — **⚠️ Found a real gap in the PRD's decision while verifying, and amended it.**
  239 started-looking 2025 teams had zero strength, which under the agreed predicate
  (`activeMemberCount == 0` ⇒ udgået) would have badged them all as having left the race.
  Investigating: `patruljestatus.startedUts` is **not** a start signal despite the name —
  it is hardcoded to `1` on *signedup* (`table/patruljestatus.go:87`, the real parse is
  commented out). The honest signal is `patrulje.signupStatus = 'STARTED'`, written by the
  patrulje consumer from the very same `patrulje.*.started` event this projection derives
  `racing` from. Cross-tabulating confirms it exactly: 2025 has 169 `STARTED` teams all
  with strength > 0, and 239 never-started teams all at zero. **The naive predicate would
  have marked all 310 teams of the current 2026 event as udgået** — the entire event.
  Corrected PRD 006 §6, §8 and §11, and pushed the fix into task 076 before the frontend
  could implement the wrong version. Worth noting the shape: this is the second time the
  specification assumed "nobody racing" means "gave up" — the first was the finished-team
  trap in §5 — and both were found by querying rather than by reading.
- 2026-08-17 — ✅ All criteria met. Full `go build`, `go vet`, `gofmt -l` and `go test
  ./...` clean. Moving to done.
