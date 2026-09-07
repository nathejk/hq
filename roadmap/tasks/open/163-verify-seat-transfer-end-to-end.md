# 163 — Verify the seat transfer end to end

**Status:** open
**Priority:** high
**Created:** 2026-09-07
**Picked up by:**
**Started:**
**Completed:**

## Description

From **PRD 012** §10 (H6) and §9. Tasks 160–162 landed the capability and everything
compiles and passes, but **no transfer has ever been performed against a real stream**. The
invariants worth trusting are the ones this task checks.

Run a real transfer in dev and confirm what the code claims:

1. **The pair nets to zero.** Credit total + charge total == 0. PRD 012 §9 makes this the
   headline metric; it should be a query that returns no rows.
2. **Both orders reach a terminal state.** In particular the **credit** order, whose
   negative total could never settle before shared-go task 156 — a row stuck at `open` is
   the regression to watch, and it fails silently.
3. **A full replay produces one transfer, not two.** Restart the API so every projection
   replays from JetStream and confirm nobody is double-credited. The ids are deterministic,
   so this should hold; it has not been observed.
4. **The roster moves.** The member leaves the origin's list and appears on the
   destination's — `spejder.teamId` actually changing is new behaviour and the single most
   invasive part of the shared-go change.
5. **`initialTeamId` was not written**, and no `spejderstatus` row appeared. A pre-race
   reassignment is not a lifecycle event.
6. **The two new `payment` columns exist in a pre-existing database.** They are applied by
   `cqrs.EnsureColumn` on start, so a database created *before* the bump is the only case
   that proves the migration ran. A fresh database would pass regardless of whether it works.
7. **Provenance survives two hops.** Move a member A → B → C and confirm the source still
   names the original MobilePay payment, not the B → C transfer.
8. **Live updates.** Both patrol pages update without a reload, from the `spejder`, `order`
   and `payment` signals alone.

## Notes

Also covers the two test gaps left by tasks 161 and 162, which are worth closing if this
uncovers anything:

- No handler tests for `reassign.go` — the preconditions (started origin, unnumbered /
  started / full destination, and the deliberate *absence* of a minimum) are rules, and a
  rule without a test is a rule somebody will "simplify" later. The full-destination check
  in particular reads the recomputed roster count rather than the frozen
  `patrulje.memberCount` column, which is subtle enough to deserve one.
- No component test for the dialog.

Worth checking while in there: **a transfer between two teams where the member has a
t-shirt**, since the size travels in `order_line.attributes` and the order is authoritative
for what gets packed. Losing it misdirects a physical object, and nothing else in the system
would notice.

## Acceptance Criteria

- [ ] A transfer performed in dev; credit + charge net to zero
- [ ] Both orders terminal, including the negative-total credit order
- [ ] API restarted and replayed: exactly one transfer, no double credit
- [ ] Member off the origin roster and on the destination's
- [ ] `initialTeamId` untouched; no `spejderstatus` row created
- [ ] New `payment` columns present in a database created before the bump
- [ ] Provenance across A → B → C still names the original payment
- [ ] Both pages live-update with no reload
- [ ] T-shirt size preserved on the charge order
- [ ] Any bug found either fixed or raised as its own task

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-09-07 — Created alongside tasks 160–162. Everything builds and the suites pass, but
  a transfer has never actually run: this is the task that turns "should work" into "does".
