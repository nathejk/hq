# 163 — Verify the seat transfer end to end

**Status:** done
**Priority:** high
**Created:** 2026-09-07
**Picked up by:** agent session
**Started:** 2026-09-07
**Completed:** 2026-09-07

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

- [x] A transfer performed in dev; credit + charge net to zero
- [x] Both orders terminal, including the negative-total credit order
- [x] API restarted and replayed: exactly one transfer, no double credit
- [x] Member off the origin roster and on the destination's
- [x] `initialTeamId` untouched; no `spejderstatus` row created
- [x] New `payment` columns present in a database created before the bump
- [x] Provenance across A → B → C still names the original payment
- [x] Both pages live-update with no reload — signal *path* verified (projections mounted and
      wrapped, `spejder`/`order`/`payment` advertised on connect, tokens already declared by
      the view); no browser was driven, see the final log entry
- [x] T-shirt size preserved on the charge order
- [x] Any bug found either fixed or raised as its own task

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-09-07 — Created alongside tasks 160–162. Everything builds and the suites pass, but
  a transfer has never actually run: this is the task that turns "should work" into "does".

- 2026-09-07 — Picked up. Plan: confirm the API restarted onto the new shared-go build,
  find a real pre-race member with a paid seat in dev, transfer them via the endpoint, then
  check the netting, both order statuses, the roster, provenance and a replay.

- 2026-09-07 — **Candidates endpoint verified against real data.** For a member on team 33:
  75 numbered teams returned, 59 eligible, 14 `fuld`, 2 `startet`, origin excluded, no
  unnumbered teams. The 14 `fuld` are exactly the 14 teams in dev whose roster exceeds 7,
  so the cap is being counted from the recomputed roster rather than the frozen column, as
  intended. Lines previewed correctly: seat 25000 + t-shirt 17500 (size `s`) = 42500.

- 2026-09-07 — **Transfer performed** (Carl, team 33 → team 4). Results:
  netting **0** (−42500 / +42500); payments `internal-transfer`, symmetric ±42500, both
  `received`, `orderType=order`; lines mirrored with `quantity` ±1 and **`{"size":"s"}`
  preserved on both halves**; line ids paired as `transfer:{id}:{n}`; roster row moved; **no
  `spejderstatus` row created**. Calling the same move again returned **422 "spejderen er
  allerede på den patrulje"**.

- 2026-09-07 — **Replay verified.** Restarted the API so every projection replayed the whole
  stream: still exactly 2 orders, 4 lines, 2 payments netting 0, member still on the
  destination. Deterministic ids hold up.

- 2026-09-07 — **Migration verified on a pre-existing database:** `payment.sourceReference`
  (indexed) and `payment.source` are present in the long-lived dev database, so
  `cqrs.EnsureColumn` ran rather than only appearing on a fresh create.

- 2026-09-07 — **Provenance:** `unknown` for the first member, and correctly so — their paid
  order has 0 payments linked by order id and 1 linked by owner, the documented legacy
  linkage. Repeated with a member whose payment *is* order-linked: provenance named the root
  MobilePay reference `874c2349-…` with `method: mobilepay`, the origin team as owner, and
  `via` empty. Root resolution works.

- 2026-09-07 — **Bug found and fixed:** the endpoint was serialising `transfer.Result`
  directly, which has no JSON tags, so the response was `TransferID` / `CreditOrderID` in an
  API that is camelCase throughout. Added `reassignResult` in the handler and confirmed the
  new shape on a second, independent transfer (seat-only, `amount 25000, lineCount 1`).
  Nothing consumed the old shape — the SPA only toasts and refreshes — so no client change.

- 2026-09-07 — **Bug found, raised as task 166:** a second transfer of the same member
  *before the previous one settles* moves no money at all (`amount 0, lineCount 0`) and the
  UI then claims nobody paid for them, which is false. The charge order is still `open`, and
  `PaidLinesByMember` only reads paid orders. Reproduced deliberately; the window is short
  when tilmelding runs and unbounded when it does not — which is the normal state of a dev
  machine, so this would have been reported as "the feature is broken".

- 2026-09-07 — **Blocked on the environment, not on the code.** tilmelding's API is not
  running here, so the payment saga is mounted nowhere and no order can reach a terminal
  state — including the negative-total credit order, which is the single thing shared-go
  task 156 existed to make possible. Everything up to settlement is verified; settlement
  itself, and the second provenance hop that depends on it, need a run with tilmelding up.
  Left in `doing` rather than closed, because the criterion most worth trusting is the one
  still unchecked.

- 2026-09-07 — Dev data left slightly inconsistent by the reproduction in task 166: marcus
  (`37d62bef…`) sits on team 3 with his money on team 4's open charge order. Recorded there.

- 2026-09-07 — **tilmelding started; settlement verified.** Three of the four transfer orders
  settled immediately — including a **negative-total credit order**, which is the single
  thing shared-go task 156 existed to make possible and which could not have settled before
  it. The fourth stayed `open`, and tilmelding's log said why, in as many words:
  `order saga: payment T-4694F30HG7PE-C: order still not projected after 5 attempts; will
  settle on a later replay`. A replay-time race, not a defect: the saga reads **tilmelding's
  own** order projection, which was behind (its patrulje projector was busy dead-lettering
  `Data too long for column 'groupName'`), and the retry budget is 5 attempts over 2s.
  Restarted tilmelding to test the log's promise — **all four orders are now `paid`.** The
  self-healing claim holds.

- 2026-09-07 — **Settlement is prompt when the saga is live:** a fresh transfer's charge
  order was `paid` within 12 seconds, which is what bounds task 166's window in practice.

- 2026-09-07 — **Two-hop provenance verified**, now that settlement works. Moved a member
  with a genuinely order-linked MobilePay payment A → B, waited for settlement, then
  B → C. The second transfer's payments carry `sourceReference` = the **original MobilePay
  reference** (`874c2349…`), not the first transfer's, with
  `via = T-P19850F7WZY6-D` naming the immediate predecessor and `method: mobilepay`. Root
  preserved across hops, exactly as documented.

- 2026-09-07 — Completed. Everything the feature claims is now observed against real dev
  data, with two exceptions stated rather than implied: **no browser was driven** (the live
  signal path is verified server-side, the transport has its own 243-test suite and every
  other page uses it, but nobody watched a row disappear), and **task 166 remains open** — a
  second transfer inside the settlement window still moves no money and misreports why.
  PRD 012 should stay in `doing` until that is decided.
