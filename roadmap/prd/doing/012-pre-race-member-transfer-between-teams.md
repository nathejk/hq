# PRD 012 — Pre-race member transfer between patruljer (seat & merchandise follow the member)

**Status:** doing
**Author:** agent session
**Created:** 2026-09-07
**Last updated:** 2026-09-07
**Approved:** 2026-09-07
**Shipped:**
**Target users:** organizer (HQ admin, acting on a contact person's request)
**Progress:** shared-go S1–S4 delivered (tasks 156–159) and consumed at `8b51980`; hq
endpoints and SPA landed (tasks 160–162); **verified end to end against real dev data**
(task 163 — netting, settlement of both negative-total credit orders, replay idempotency,
roster move, t-shirt size, two-hop provenance). One open defect blocks shipping: **task
166** — a second transfer inside the settlement window moves no money and tells the operator
nobody paid. See §10.

<!--
Status must match the folder this file is in: draft/, doing/ or done/.
Leave Approved blank until the PRD moves to doing/, and Shipped blank until it
moves to done/. See roadmap/prd/README.md for the lifecycle.
-->

---

## 1. Summary

Let an HQ operator move a **registered, not-yet-started** spejder from one accepted
patrulje to another from the patrol page, and have the **money follow the member**: the
paid seat and any merchandise that member bought are credited on the sending team's
books and charged on the receiving team's, settled by an **internal transfer** rather
than a new MobilePay payment.

## 2. Problem & Motivation

- **What problem does this solve?** A contact person pays for two teams with a given
  number of seats in each, members occupy those seats, and some of them have bought a
  t-shirt. Then reality intervenes and the teams need reshuffling. Today there is **no
  way to do this at all** before the race:
  - `spejder.teamId` is **write-once**. The only branch that ever writes it is an
    `INSERT IGNORE` in `shared-go/tables/spejder/consumer.go:43-54`; the subsequent
    `UPDATE spejder SET name=…` (`consumer.go:63-75`) deliberately omits `teamId`. **No
    event in the system changes a roster row's team.**
  - The move that *does* exist — `spejder.*.team.moved`, `spejderstatus.MoveTeam`,
    `PUT /api/member/:memberId/team` — is **race-time only** and structurally
    unsuitable here (see below).
  - The paid seat cannot move either. A seat is `SUM(order_line.quantity)` over paid
    participation lines for the owner (`go/nathejk/table/patruljenumber/saga.go:564-585`);
    there is no command to move a line between owners and **hq has no order or payment
    command wired at all** (§8).
  - So the operator's only options today are wrong on purpose: delete the member and
    re-register them on the other team (destroying the payment trail and the seat), or
    leave the data lying.
- **Why the existing move is the wrong mechanism.** PRD 006's `team.moved` models a
  member who *starts with one patrol and continues with another mid-race*, which is why
  it keeps `initialTeamId`, why `GetSpejdere` still lists the member on their **origin**
  roster tagged "flyttet til anden patrulje" (`go/internal/data/member.go:57-108`), and
  why the destination must be `signupStatus == STARTED`
  (`go/cmd/api/member.go:350-365`). It also requires an `sosId` (`caseRequired`,
  `member.go:420-425`) and a `spejderstatus` row, and **`spejderstatus` rows only come
  into existence at `patrulje.started`** (`spejderstatus/consumer.go:39-63`). A pre-race
  reshuffle is the opposite in every respect: the member genuinely **belongs** to the
  new team from now on, must vanish from the old roster and appear on the new one, there
  is no emergency and no case, and neither team has started. Reusing `team.moved` would
  produce a roster that is a lie and a case log full of administrivia.

  **The two moves are different features that share a noun.** They overlap nowhere, and
  an implementer who treats this PRD as an extension of PRD 006 will get it wrong:

  | | In-race move (PRD 006) | Pre-race transfer (this PRD) |
  |---|---|---|
  | Meaning | Started with A, continues with B | Was never really on A; belongs to B |
  | Team state | Both teams **started** | Neither team started |
  | Origin roster | **Keeps** the member, tagged "flyttet" | **Loses** the member |
  | `initialTeamId` | Preserved — the point of the record | Must **not** be written |
  | Trigger | Emergency; `sosId` **required** | Administrative; **no case** |
  | Prerequisite | A `spejderstatus` row (exists only after start) | A roster row; no status row exists yet |
  | Money | None. Seats were settled long ago | **The whole hard half** — seat + merch transfer |
  | Reversible by | Moving back (status history grows) | Moving back (a second, offsetting transfer pair) |

  The one thing they must share is that a given member is only ever subject to one of
  them: §6 requires the pre-race action to be unavailable once a race-time move is
  recorded, so the two cannot interleave on one person.
- **Why now?** Both halves of the problem now have somewhere to land. PRD 002/003 gave
  us orders, products and paid-seat accounting; PRD 006 gave us a member-focused
  vocabulary in which teams are not merged and split, **members are moved**. This is the
  pre-race sibling of that move, and it is the last place where the platform still
  forces a destructive workaround.
- **Evidence.**
  - The type documentation already anticipates it: `shared-go/types/member.go:86-89`
    defines `MemberStatusSeated` — *"a paid seat is assigned to this member"* — and
    `member.go:13-31` explains that *"the gap between registered and seated is the
    team's outstanding order"*. **Nothing publishes `seated`**; it is reachable only
    through the manual status override.
  - `shared-go/tables/order/commander.go:132-135` already blesses signed quantities:
    *"A negative quantity is a credit: it reclaims a unit already paid for."*
    `ApplyPaidOffset` (`commander.go:684`) is the existing precedent, though it always
    pairs a credit with a charge **inside one order**; a transfer needs the pair split
    across two owners.
  - The refusal to invent payments is documented and must be respected on its own
    terms: `go/nathejk/commands/klan.go:15-31` — *"HQ inventing a payment that did not
    happen through the provider … corrupts the payment trail to fix a display."* §8
    argues why an internal transfer is not that.

## 3. Goals

- An operator can move one not-yet-started, registered member from their team to
  another **accepted** team, from the member's expanded row on `/patrulje/:teamId`, in
  one dialog and without inventing or destroying anything.
- **The roster is the truth afterwards.** The member is on the destination team's
  roster and off the origin's — no "flyttet" ghost row, because pre-race there is no
  origin story worth preserving.
- **The money follows the member, exactly and visibly.** The seat they occupied and
  any merchandise attributable to them are credited to the sending team and charged to
  the receiving team, both sides reconcile to the same amount, and the reconciliation is
  readable on each team's Betalinger section months later.
- **No money is invented and none is destroyed.** Nothing is charged to a card, nothing
  is refunded to one; the already-received funds are re-attributed. A transfer nets to
  zero across the two teams.
- **The money keeps its history.** A transferred seat still points back to the real
  payment that originally paid for it, however many times the member has been moved, so
  "who actually paid for this participant?" stays answerable.
- Both teams remain in a legal state, or the operator is told why not, **before** they
  commit: the destination is not started and has room; the origin has not started.
- The two teams need **no relationship whatsoever** — same contact person, different
  contact persons, different groups, different korps. The operator's authority is the
  authority.

## 4. Non-Goals

- **Self-service reshuffling by the contact person.** This is an HQ-operated action on
  their behalf. A tilmelding-side flow may follow; it is not this.
- **Moving a member after either team has started.** That is PRD 006's `team.moved`,
  it already works, and the two must not be conflated. This PRD adds a *second, disjoint*
  move and is responsible for keeping the boundary between them sharp (§6).
- **Refunds.** A sending team left with more paid than owed holds a credit, visible as
  an over-paid order. Whether, when and how HQ pays money back to a card is out of
  scope and unresolved on purpose (§11).
- **Moving members between team *types*** (patrulje ↔ klan). Seat products are
  eligibility-scoped per team type (`product/seeds_2026.go`), the member-count rules
  differ, and klaner have no member lifecycle at all (PRD 006 §4). Patrulje→patrulje only.
- **Bulk / whole-team reshuffling in one action.** One member per confirmation. The
  operator repeats it; a team of seven is six clicks, and each click is separately
  reversible.
- **Enforcing the 3-member minimum *on the move*.** Not enforced, because emptying a
  team out one member at a time is a thing this feature must be able to do — the last
  member cannot be moved if the rule is checked on the way out. **The minimum belongs at
  the start gate instead:** a team below three is not allowed to start, and the usual
  remedy is exactly this feature — transfer the remaining one or two participants into a
  team that has not started yet. Blocking the move would remove the only clean way to
  satisfy the rule it was trying to protect.

  Note this is **its own rule with its own reason**, not an inheritance from PRD 006:
  that one declines to enforce three *racing* members because a member leaving mid-race
  is a fact to record rather than a decision to adjudicate. Pre-race the numbers bite in
  two other places — `patruljenumber.MinSeats` (`saga.go:51-53`), which counts **paid
  seats** and gates acceptance, and the start gate above. Moving seats out can therefore
  breach a rule that matters; §11 asks what that should mean.

  **Building the start gate is not in this PRD's scope**, and today it does not exist:
  `startPatruljeHandler` (`go/cmd/api/patrulje.go:200-237`) accepts whatever starters it
  is given with no count check at all, and `MinMemberCount: 3` is a display value served
  to the SPA. The gap is recorded here because this feature is the remedy such a gate
  would send operators to — the two are worth landing in the same season, and a task is
  listed in §10.
- **Revoking a team number.** A sending team that drops below `MinSeats = 3` paid seats
  keeps its number. `patruljenumber` has no unassign path (`saga.go:474-503`,
  `isAssigned` guard) and this PRD does not build one — see §8 and §11.
- **Changing the MobilePay integration**, the payment provider, or the payment
  lifecycle owned by tilmelding.
- **Assigning seats to members as a first-class fact.** `MemberStatusSeated` stays
  unproduced. The seat↔member link remains `order_line.memberId`, which is what this
  feature reads and rewrites.

## 5. User Stories & Scenarios

- As an **organizer**, I want to move a paid-for member to another patrulje so a
  contact person's reshuffle is recorded correctly instead of being faked by deleting
  and re-registering them.
- As an **organizer**, I want the seat and the t-shirt they paid for to move with them,
  so neither team is asked to pay twice and neither is left short.
- As an **organizer**, I want to see which teams I am *allowed* to move someone into,
  and to see the ones I am not — and why — rather than discovering it from an error.
- As **HQ finance**, I want each side of a transfer to appear as its own order with a
  clearly non-card payment, so a reshuffle is auditable and I never wonder whether
  MobilePay was really involved.

### Primary happy path

1. Operator opens `/patrulje/{origin}` and expands the member's row.
2. They click **Flyt til anden patrulje**.
3. A dialog lists candidate teams — every patrulje in the year **with a team number**.
   Teams that are **started** or **already at 7 members** are listed but **disabled and
   struck through**, with the reason shown.
4. They pick a destination. The dialog shows exactly what will move: the seat
   (250 kr) and, if present, the t-shirt (175 kr, size L) — and states that the amount
   is credited to `{origin}` and charged to `{destination}` as an internal transfer.
5. They confirm. The member disappears from this roster; the origin's Betalinger gains
   a credit order for −425 kr marked paid by internal transfer, and the destination's
   gains a charge order for +425 kr, likewise paid.
6. Both pages update live without a refresh.

### Edge cases and error scenarios

- **Member bought nothing but their seat** → one participation line each side. Ordinary.
- **Member has no paid seat at all** (registered, team never paid, or seats fewer than
  members) → the member moves and **no orders are created**. There is nothing to
  transfer; the destination's obligation grows and the origin's shrinks, which the
  existing derived `signupStatus` already expresses. The dialog must say so plainly
  rather than silently doing nothing.
- **Team has fewer paid seats than members** → seats are a *quantity*, not an
  assignment (`saga.go:564-585` sums quantity, not distinct members), so "which seat is
  this member's" is genuinely ambiguous. Resolution: a member's seat is the
  participation line carrying their `memberId`; if none does, treat as the case above.
- **Team is down to 1–2 members and cannot start** → this is the *motivating* case, not
  an edge case. A team below three may not start, so the remedy is to move the survivors
  into another team that has not started yet, one at a time, until the team is empty and
  the members are racing with someone else. Their paid seats and shirts go with them, so
  the receiving team is not asked to pay for the newcomers. This is precisely why the
  move must not check a lower bound on the origin (§4) — the last member out is the whole
  point of the operation.
- **Destination started between the dialog opening and the confirm** → server rejects
  with a clear message; the operator sees it and the list refreshes (it is live).
- **Destination would exceed 7 members** → rejected server-side too, not only in the UI.
- **Origin has started** → the button is not offered at all; the operator is pointed at
  the SOS-based race-time move.
- **Member already moved by the race-time mechanism** (`currentTeamId != teamId`) →
  button hidden. The two mechanisms must not interleave on one member.
- **Move back / undo** → performed as another move in the opposite direction, producing
  a second, offsetting transfer pair. There is no delete, and the history reads as two
  moves, which is what happened.
- **Same team chosen** → refused (`ErrSameTeam` precedent, `member.go:620-621`).
- **Transfer succeeds, member move fails, or vice versa** → the ordering and the
  failure semantics are a real design constraint, not an afterthought; see §8.

## 6. Requirements

### Functional

- [ ] A **pre-race reassignment** exists as its own domain fact, distinct from
      `team.moved`: it changes the member's team of record, and the origin roster no
      longer lists them.
- [ ] `spejder.teamId` becomes mutable **by that event only**. The `INSERT IGNORE` /
      `UPDATE`-without-`teamId` behaviour on `spejder.*.updated` is unchanged.
- [ ] It requires **no SOS case**. An administrative reshuffle is not an incident and
      must not mint one.
- [ ] Preconditions, enforced server-side and reflected in the UI:
      - origin team `signupStatus != STARTED`
      - destination team has a **non-empty `teamNumber`** (accepted into the race)
      - destination team `signupStatus != STARTED`
      - destination roster count `< 7` (`MaxMemberCount`)
      - destination `!=` origin
      - member has no race-time move recorded against them
      - **no minimum-member check on the origin** — deliberately absent
- [ ] The candidate list returns **all numbered teams**, each annotated with whether it
      is selectable and why not, so the UI can show ineligible teams disabled rather
      than hiding them.
- [ ] **Transferable value** is every `order_line` on a **paid** order owned by the
      origin team whose `memberId` is the moving member — participation *and*
      merchandise, at the **price actually paid** (the snapshotted `unitPrice`, not
      today's catalogue price).
- [ ] The transfer produces **two orders**: a credit order owned by the origin
      (negative lines, negative total) and a charge order owned by the destination
      (the same lines, positive), each line preserving `productSku`, `productName`,
      `unitPrice`, `memberId` and `attributes` (t-shirt size must survive).
- [ ] Both orders are settled by an **internal transfer payment** — not MobilePay —
      whose amount matches its order exactly, so `paidAmount == totalAmount` on each
      and both reach a terminal state. Neither order may be left permanently open.
- [ ] A transfer **nets to zero**: credit total + charge total == 0.
- [ ] Both orders are **linked to each other and identifiable as a transfer**, so
      finance can pair them and neither is mistaken for a real sale or refund.
- [ ] Each transfer payment records the **provenance of the money it moves** — a
      reference to the original provider payment that brought those funds in, carried in
      the event payload so it survives replay. Where the source payment cannot be
      identified (many payments are reachable only through their owner, not their order),
      provenance is recorded as **explicitly unknown** rather than omitted, and the
      transfer still succeeds.
- [ ] Provenance **survives repeated moves**: a member moved A→B→C still points at the
      original payment on A, not at the B→C transfer.
- [ ] A transferred seat's origin is **visible to the operator**, not merely stored —
      e.g. *"betalt af Patrulje 12 (MobilePay, 4. juni)"*.
- [ ] The two orders and their transfer payments render sensibly in the existing
      Betalinger tables on both teams' pages and on `PaymentListView`, including the
      negative amounts, without a special case per screen.
- [ ] The member's expanded row on `/patrulje/:teamId` gains a **Flyt til anden
      patrulje** button, alongside the existing "Ret status manuelt", shown only when
      the preconditions allow it.
- [ ] Both affected pages update live; no manual refresh, and no full-page reload.

### Non-Functional

- **OpenAPI annotations on every new or changed endpoint** (repo `.rules`). Non-optional.
- **Live updates** per `.rules`: the reassignment projection is added to the
  `projections` slice in `cmd/api/main.go`, and the new/changed views load through
  `useLiveResource` with a mandatory `dependsOn`. The move publishes on the `spejder`
  entity and the transfer on `order` / `payment`, all three of which
  `PatruljeView.vue:33-46` already depends on — so the origin page updates for free and
  only the destination's dependency needs thought.
- **Replay-safe and idempotent.** Everything is rebuilt from JetStream on every API
  start; a replayed transfer must not double-credit anyone. Any in-memory
  high-water-mark or "have I done this" state is a bug.
- **Auditable.** After the event nobody should have to reconstruct what happened from
  amounts: who moved, from where, to where, when, by which operator, and which two
  orders paid for it.
- **Danish UI text**, `da-DK` amount and date formatting, PrimeVue + Tailwind, matching
  the existing patrol page.
- **No regression to the race-time move.** `PUT /api/member/:memberId/team` and
  `POST /api/sos/:id/team/:teamId/move` keep their current semantics, including
  `caseRequired`.

## 7. UX / UI Notes

**Where:** `vue/src/views/PatruljeView.vue`, the members `DataTable` `#expansion` block
(L210–254). The file's own comment (L110–113) explains why per-member actions live in
the expansion rather than as a column action; this follows that.

**Button:** `Flyt til anden patrulje`, `size="small"`, `outlined`, alongside `Ret
status manuelt`. Hidden — not disabled — when the origin has started or the member has
been moved race-time, since in those cases the action does not exist rather than being
temporarily unavailable. A one-line hint below, in the style of the existing
`"Brug kun når virkeligheden ikke passer…"`, states that the paid seat and merchandise
move with the member.

**Dialog:** modal, header `Flyt til anden patrulje`. The prior art to copy is the
"Skift patrulje" dialog in `vue/src/components/SosTeamCard.vue` (L779–815): an
`InputText` search over teams plus a filtered `v-for` list with a **Vælg** button per
row. That is this repo's team-picker idiom — **not** `AutoComplete`, **not** `Select`,
which is reserved for small fixed option sets. Differences from that prior art:

- The list must include **ineligible** teams, rendered `line-through text-gray-400`
  with the reason as a short suffix (`— startet`, `— fuld (7)`) and no **Vælg** button.
  This is explicit in the request and is the opposite of `SosTeamCard`'s `destinations()`
  (L383–397), which filters ineligible teams out.
- Each row shows `teamNumber · name · group` and the member count, e.g. `4/7`, so the
  operator can see room at a glance.
- Once a destination is chosen, the dialog shows a **what moves** summary before the
  confirm: the lines being transferred with their amounts, the total, and a sentence
  naming both teams — *"425,00 kr. krediteres Patrulje 12 og debiteres Patrulje 31 som
  intern overførsel."* If nothing is transferable, that sentence is replaced by *"Der er
  ikke betalt for denne deltager, så der overføres ingen betaling."*
- Footer: `Annuller` (secondary text) and `Flyt` (disabled until a destination is
  chosen, `:loading` while submitting).

**Data for the picker:** `useLiveResource('patrulje:list', …, { dependsOn: ['patrulje'] })`
hits the **same cache key** `PatruljeListView.vue:23-30` and `SosTeamCard.vue:58-65`
already use, so the picker costs no extra request on a warm cache. Note that
`GET /api/patrulje` already returns `teamNumber`, `memberCount`, `activeMemberCount` and
`signupStatus` (`nathejk/table/patrulje/table.go:14-45`) — nearly everything the picker
needs, which is why §8 prefers annotating server-side over inventing a second endpoint.

**Do not mix the two team shapes.** The list endpoint returns
`teamId`/`teamNumber`/`signupStatus`; the detail endpoint's `team` is `data.Patrulje`
with `id`/`number`/`status` (`go/internal/data/team.go:14-33`).

**After confirming:** close the dialog and let live updates do the work; an explicit
`refresh()` as belt-and-braces matches `saveCorrection` (L145). The member simply
leaves the table.

**Betalinger:** the existing table on the patrol page renders `totalAmount`,
`paidAmount`, `dueAmount` and a binary status tag. Two presentation problems need
solving there (§8): order `Status.MarshalJSON` collapses `cancelled` into `"PAID"`, so a
transfer order shows as *Betalt* whatever happens to it; and a **negative** amount must
read as a credit rather than as a mistake. A transfer order should be labelled as such —
*Intern overførsel* — rather than sitting in the list looking like an anonymous 0-kr or
negative sale. The order lines are already fetched and never rendered (there is no
expansion on that table), so showing *what* moved requires adding one.

## 8. Technical Considerations

### The event and the projection (Go, hq + shared-go)

- **A new event is required**; `team.moved` must not be reused for the reasons in §2.
  Proposed: `NATHEJK.{year}.spejder.{memberId}.reassigned`, payload
  `{memberId, fromTeamId, toTeamId, actor}` — deliberately parallel to
  `NathejkMemberTeamMoved` (`shared-go/messages/member.go:265-272`) so the difference in
  *meaning* is carried by the subject rather than by a flag on a shared struct. Note hq
  keeps its own copy of that struct (`spejderstatus/messages.go:172-180`); this PRD
  should not deepen that duplication without deciding which side owns the new one (§11).
- **`shared-go/tables/spejder/consumer.go` must consume it** and `UPDATE spejder SET
  teamId = ?`. This is the first mutation of that column and the single most invasive
  change in this PRD: `spejder` is the roster everything else joins against, and it is
  shared with tilmelding. `Consumes()` (`consumer.go:14-20`) currently lists only
  `updated`, `deleted` and `patrulje.started`.
- **`spejderstatus` should ignore it.** Pre-race there is no status row and no
  `activeMemberCount` to recompute, and `initialTeamId` — which exists precisely to
  record where a member *started* — would be actively wrong if a pre-race reassignment
  wrote to it. A reassigned member who later races has always been on their new team.
- **`patrulje.memberCount`:** no work needed for the list (`patrulje/query.go:56-126`
  recomputes it as `COUNT(*)` from `spejder`), but the **detail** endpoint reads the
  stored column, which is 0 until `patrulje.started` — so the destination's
  member count on its own page may look wrong. Worth checking whether the 7-member
  precondition is computed from the same source the UI displays; if it is not, they will
  disagree.

### The money (shared-go `order` + `payment`) — the hard part

Four separate obstacles. One is now settled; three remain.

1. **hq may create the payments it initiates — settled.** The earlier reading of
   `main.go:233-235` (*"hq … has no business transitioning them"*) was too broad, and the
   comment has been rephrased. Ownership is **per payment**: a service owns the payments
   it initiates. tilmelding owns the *provider* payments because it asks MobilePay for
   money and takes the callback; an internal transfer has no provider and no callback, so
   hq initiating one is not a transgression — it is hq owning its own payment. A manual
   registration is the same. The narrow rule that stands is that hq must not run a second
   copy of the **Pay saga**.

   That has a pleasant consequence for Phase 1: **settlement follows the money, not the
   initiator.** The saga reacts to `payment.received` regardless of who published it
   (`shared-go/tables/order/saga.go:110-124`, filtered only by season and
   `OrderType == order`), so tilmelding's single mount will close the **charge** order
   with no new settlement code. Only the credit side needs work — obstacle 2.

   This also relieves the pressure to invent an hq→tilmelding API, which does not exist.

2. **A negative-total order can never close.** The Pay saga bails on `TotalAmount <= 0`
   (`shared-go/tables/order/saga.go:289-301`) and `Settle` refuses a non-zero total with
   `ErrOrderNotFree` (`commander.go:342`), so the credit order would stay `open` — and
   therefore mutable — forever, silently, with nothing logged. This must be fixed, not
   worked around: either the saga learns that a credit order covered by a matching
   negative payment is settled, or `Settle` is generalised from "free" to "fully
   covered". **Task S1** (§10). Note `checkStock` counts only positive quantities
   (`commander.go:493`), so credits correctly do not release seat stock — that part is
   already right.
3. **There is no non-MobilePay payment method.** `payment.method` is a free
   `VARCHAR(99)` with exactly one literal ever written — `"mobilepay"`, hardcoded at
   `shared-go/tables/payment/commands.go:162`. No `PaymentMethod` type exists in
   `shared-go/types/`. Adding an `internal-transfer` method is schema-free but needs a
   producer, and every "has this been paid" sum filters `status IN ('reserved',
   'received')` — so a transfer payment must land in one of those statuses to count.
   Since hq is now the initiator of a payment kind, `method` should become a **typed enum
   in `shared-go/types`** rather than a second hardcoded string in a different repo.
   **Task S2**.
4. **A negative payment is unprecedented.** `payment.amount` is a signed `INT`, so it is
   representable, but nothing has ever written one and `paidAmount` subqueries
   (`order/querier.go:102-104`) will simply sum it. The alternative — a positive payment
   on the charge order and *no* payment on the credit order, closed some other way — is
   less symmetrical but avoids negative payments entirely. **Choose deliberately (§11 Q2)**;
   the symmetry is what makes the pair auditable, and asymmetry is what keeps the
   payment table conventional. Feeds S1 and S4.

### Provenance: where the money originally came from

An internal transfer moves money that a **real provider payment** once brought in, and
that origin must remain traceable — otherwise a transferred seat's audit trail dead-ends
at an internal bookkeeping entry and the question "who actually paid for this member?"
becomes unanswerable. Requirements and constraints:

- A transfer payment must carry a reference to the **payment(s) whose money it moves**,
  not merely to the counterpart order. Both are wanted: the counterpart pairs the two
  halves of *this* transfer, the provenance answers where the funds entered the system.
- **Follow the chain to its root, not just one hop.** A member moved A→B→C must still
  point at the original MobilePay payment on A, otherwise provenance degrades with every
  reshuffle. A transfer whose source is itself a transfer should inherit that transfer's
  root reference rather than naming its immediate predecessor — or record both, but the
  root is the one that must never be lost.
- **Two facts make this harder than it sounds.** First, `payment.orderForeignKey` is
  polymorphic: since the order entity landed it holds an order id with
  `orderType == "order"`, but older rows hold a team or user id
  (`shared-go/tables/payment/table.go:96-103`). Second, most real payments are **not**
  reachable from an order at all — `go/cmd/api/order.go:52-57` records that of 189 paid
  orders in live 2026 data, only 44 are linked by order id while 151 are reachable only
  through their owner. So "find the payment that paid for this seat" is a lookup that can
  legitimately fail, and the transfer must still be creatable when it does. Recording
  *unknown provenance* explicitly beats silently recording none.
- Candidate homes: a **real column** is now on the table (schema changes are in scope,
  delivered as a shared-go task), and for something finance will query — *show me every
  transfer whose money came from payment X* — an indexed column beats digging through JSON.
  `payment.operations` (already `JSON`, already an append-only narrative of what happened
  to a payment) remains the natural home for the human-readable chain, and
  `order_line.attributes` for per-line source. Decide per field rather than adopting one
  mechanism for all of it; the migration caveat is in §10, not a reason to avoid columns.
- The provenance must survive a **full JetStream replay**, which means it has to be in
  the event payload, not computed once at command time from a read model that will look
  different later.

What the operator should see, at minimum: on a transferred seat, *"betalt af Patrulje 12
(MobilePay, 4. juni)"* rather than *"intern overførsel"* full stop.

Given the above, the transfer wants to be **one command that does the whole thing** —
read the member's paid lines, create both orders, set their lines, pay both, publish
the reassignment — rather than the caller orchestrating five commands. `EnsureOpenOrder`
is explicitly *not* the entry point: it reuses an existing open order
(`commander.go:64-115`) and would contaminate a real, unpaid order with transfer lines.

**Failure semantics matter and cannot be hand-waved.** There is no transaction across
JetStream publishes. If the reassignment lands and the transfer does not, a member sits
on a team whose seat was never paid for while the origin still holds it — recoverable by
hand, and *visible*. If the transfer lands and the reassignment does not, the money has
moved for a member who has not, which is worse and much harder to spot. Therefore
**publish the money first, the reassignment last**, and make the whole operation
idempotent on replay. Precedent for partial-failure honesty:
`spejderstatus.MoveMembers` pre-validates everything before publishing anything
(`commands.go:334-390`).

### Candidate list (Go)

`GET /api/patrulje` already carries `teamNumber`, `signupStatus`, `memberCount` and
`activeMemberCount`, so the SPA *could* compute eligibility itself. It should not: the
same rules are enforced server-side on the move, and two copies of the rule will drift —
the exact failure mode §4 of PRD 003 and `patruljenumber`'s `paidSeatsFor` comment both
warn about. Prefer either a dedicated
`GET /api/member/:memberId/transfer-candidates` returning teams with an
`eligible` flag and a `reason`, or the same annotation added to the existing list. The
former keeps the rule adjacent to the command that enforces it and can also return the
**transferable lines** for the dialog's summary, which the team list has no business
knowing.

### API endpoints

| Method | Path | Purpose |
|---|---|---|
| `PUT` | `/api/member/:memberId/reassign` | Pre-race move + transfer. Body `{teamId}`. **No `sosId`.** |
| `GET` | `/api/member/:memberId/transfer-candidates` | Teams with `eligible` + `reason`, plus the lines that would transfer |

Named `reassign` rather than `team` so it cannot be confused with the existing
`PUT /api/member/:memberId/team`, which stays exactly as it is. **Both need OpenAPI
annotations** (`.rules`). Error mapping should follow `member.go:606-636`, adding
distinct validation messages for *started*, *fuld*, and *ikke optaget i løbet* — a
single generic 422 would leave the operator guessing which rule they hit.

### Data / storage

No new tables. Changed: `spejder.teamId` becomes mutable; `orders` / `order_line` /
`payment` gain transfer rows. New **columns** are acceptable where they earn their keep —
provenance is the likely case, since finance will want to query it — with the existing JSON
columns (`payment.operations`, `order_line.attributes`) for the narrative and per-line
source. The constraint is procedural, not architectural: `CREATE TABLE IF NOT EXISTS`
**never alters an existing table** (`.rules`), and shared-go's answer to that is
`cqrs.EnsureColumn` / `cqrs.EnsureIndex` called from the package's `New`
(`shared-go/tables/order/table.go:138-153`). Use it; adding a column to a `table.sql`
alone is silent in dev and only takes effect in stage/prod, where the database is cleared
before deploy — which is exactly how this bites you late.

### Dependencies & risks

- **`shared-go` changes are unavoidable and are delivered as separate, liftable tasks**
  (§10): the `spejder` consumer, the transfer command, the payment method, provenance, and
  the saga or `Settle` change. `spejder`, `orders` and `payment` are shared with
  **tilmelding**, which also creates members and orders, so each task states why tilmelding
  is unaffected. The sequencing risk is the real one: **this PRD cannot land hq-only**, and
  every hq task blocks on the dependency bump.
- **The seam is the contract, and it can drift silently.** hq depends on subject strings,
  JSON field names and exported signatures decided in another repo by an implementor who
  cannot see this PRD. A rename that compiles — a changed JSON tag, a subject with a
  different verb — produces a projection that consumes nothing and a UI that shows empty
  tables, with no build error anywhere. Pin the contract in the task specs and verify it at
  the bump (H0), not at Phase 4.
- **`patruljenumber` interaction.** Moving seats out can drop a team below
  `MinSeats = 3` paid seats, and the number is **not** revoked — there is no unassign
  path, and `isAssigned` makes assignment once-only (`saga.go:474-503`). Conversely a
  destination gaining seats is already numbered by construction (the picker only offers
  numbered teams), so no new number is triggered. Net effect: acceptance is sticky.
  Sticky is probably right — and the **start gate is the safety net** that makes it safe:
  a team that has been hollowed out keeps its number but cannot start, so a stale number
  never puts an ineligible team on the route. It should still be a **decision** rather
  than a side effect (§11), and it is one more reason the gate matters.
- **Order `Status` on the wire is binary**: `MarshalJSON`
  (`order/table.go:76-81`) renders `open → "OPEN"` and *everything else, including
  `cancelled` → `"PAID"`*. A cancelled transfer would display as *Betalt*. Any UI that
  needs to distinguish a transfer needs a field that survives serialisation.
- **Pre-existing bug in the neighbourhood**, found while researching and worth fixing
  before it is blamed on this feature: `patrulje/query.go:107-115` compares
  `PaidAmount` (**øre**, from `payment.amount`) against
  `TshirtCount*175 + MemberCount*250` (**kroner**), so the derived `signupStatus` reads
  `PAID` for almost any payment. Transfers change member and t-shirt counts and will
  make this wrong more visibly. Also, prices are hardcoded in **three** places —
  `TeamConfig` in four handlers with three different Min/Max sets, the product
  catalogue in øre, and that SQL.
- **`PatruljeView.vue` references `data.updatedAt`** on a member row, which is not on
  `data.Spejder` (`internal/data/member.go:16-38`) — always falsy today. Adjacent to the
  code being touched.
- **Dirty-state rule.** The expansion can hold an in-progress status correction, and a
  live payload can already replace `spejdere` underneath it (`.rules` → Live updates).
  The move dialog is modal so it is low-risk, but do not make the existing wrinkle worse.

## 9. Success Metrics

- **Zero delete-and-re-register reshuffles.** The workaround disappears because the
  supported path exists. Measured by absence of `spejder.*.deleted` followed by a
  re-signup with the same name pre-race.
- **Every transfer nets to zero.** A query summing transfer order totals per transfer
  returns 0 rows that do not. Any non-zero row is a bug, and this is the metric that
  catches it.
- **No permanently-open transfer orders.** Count of transfer orders with
  `status = 'open'` older than a minute is 0 — the direct test of obstacle 2 in §8.
- **No team is asked to pay twice.** Zero support contacts about a moved member's seat.
- Operator can complete a move in **under 30 seconds** without leaving the patrol page.

## 10. Rollout / Task Breakdown

Sequenced around a **cross-repo handoff**. The interesting constraint is not difficulty,
it is that the money and the roster both live in `shared-go` while the UI and the
endpoints live here — so the work splits into tasks that must be *executed in another
repo*, a dependency bump, and then hq work that cannot start until the bump lands.

Schema changes and saga changes are both in scope. What they are not is *ours to make
from this repo*.

### How shared-go tasks are handed off

Each shared-go task below is written to be **lifted wholesale into the `shared-go` repo**
and implemented there by an agent with no access to this PRD and no knowledge of hq. That
imposes a standard on them, and a task that fails it will come back wrong:

- **Self-contained.** State the problem, the current behaviour with file:line, and the
  desired behaviour. Never say "as described in PRD 012" — the implementor cannot read it.
  Quote the relevant code instead of pointing at it.
- **Justified in shared-go's own terms.** "hq needs it for a button" is not a reason to
  change a shared library. "A credit order can never reach a terminal state, so it stays
  mutable forever and a later edit silently rewrites a settled exchange" is.
- **Independently verifiable.** Acceptance criteria that hold without hq running — a unit
  test, or a stated invariant like *a transfer pair sums to zero*.
- **Explicit about the contract hq will depend on**: subject strings, JSON field names,
  exported signatures. This is the seam; if it drifts, the bump breaks hq silently rather
  than loudly.
- **Explicit about tilmelding.** `spejder`, `orders` and `payment` are shared with it.
  Any change that alters existing behaviour rather than adding to it needs a sentence on
  why tilmelding is unaffected — or a note that it is not.

Schema note for those tasks: `CREATE TABLE IF NOT EXISTS` **never alters an existing
table** (`.rules`), so a new column added only to a `table.sql` appears in stage/prod
(database cleared before deploy) and is silently missing in every existing dev database.
shared-go already has the idiom for this and it must be used rather than reinvented:
`cqrs.EnsureColumn(r, w, table, column, definition)` and `cqrs.EnsureIndex(...)`, called
from the package's `New` — see `shared-go/tables/order/table.go:138-153` for both, and
`tables/payment/table.go:145-161` for the older hand-rolled
`ALTER TABLE … ADD COLUMN IF NOT EXISTS` that predates them.

### Phases

**Phase 0 — decide.** Settle the two design questions in §11 (how a credit order closes,
and payment symmetry). They are inputs to the Phase 1 task specs, not code.

**Phase 1 — write the shared-go task specs and lift them.** Four tasks (S1–S4 below),
authored here, executed there. S1–S3 are independent of each other; S4 depends on S2 and
S3.

**Phase 2 — bump the dependency.** Update `github.com/nathejk/shared-go` in `go/go.mod`,
verify the read models still build and replay, and confirm the new columns exist in dev.
This is a real task with a real failure mode, not bookkeeping.

**Phase 3 — hq API.** The two endpoints, precondition checks, error mapping, OpenAPI
annotations, projection wiring for live signals.

**Phase 4 — SPA.** The button, the dialog with disabled-but-visible ineligible teams, the
"what moves" summary, and the Betalinger presentation of credits, transfers and
provenance.

### Tasks for `roadmap/tasks/open/`

Format per `roadmap/tasks/TASKS.md`. The `[shared-go]` prefix marks a task whose
Description and Acceptance Criteria are written for the other repo; it is tracked here
because this PRD depends on it, but it is not implemented in this working tree.

**Created on approval (2026-09-07):** S1 = **156**, S2 = **157**, S3 = **158**,
S4 = **159** — all four **done**, implemented in shared-go and documented there in
`docs/moving-a-paid-member.md`.

**hq tasks:** **160** (bump, done), **161** (both endpoints, done), **162** (SPA, done),
**163** (end-to-end verification, done), **166** (settlement-window defect, **open — blocks
`done`**), **164** (start gate, open), **165** (øre/kroner, open).

**What verification changed.** Two things only a real run could show, both now recorded
against §8:

- **Settlement is another service's job, and it can be late.** The saga reads *tilmelding's*
  order projection, so a replay-time race can leave a transfer order `open` past the retry
  budget — it logs `will settle on a later replay` and does, on the next boot. Self-healing,
  but it means "both orders terminal" is eventually-true, not immediately-true.
- **That window has a user-visible consequence** (task 166): while a charge order is
  unsettled the member's seat is invisible to `PaidLinesByMember`, so a second move transfers
  nothing and the dialog claims nobody paid for them. Milliseconds when tilmelding is
  healthy; unbounded when it is not — which is a dev machine's normal state.

**H3 turned out to be empty**, and that is worth recording rather than quietly dropping:
the transfer command only *publishes*, and `ordertable`, `paymenttable` and `spejdertable`
were already mounted and already inside the `projections` slice — so the live signals
(`order`, `payment`, `spejder`) flow with no wiring at all, and the SPA needed no new
`dependsOn` token. The entity-agnostic design in PRD 004 is what made a whole phase
unnecessary.

**Decisions taken during implementation, both by the shared-go side:**

- §11 Q1 (how a credit order closes) — **the saga was taught negative totals**, so one
  settlement path serves every order. A transfer whose lines are all zero-priced is the one
  exception and gets `order.paid` published directly.
- §11 Q2 (payment symmetry) — **symmetrical**: a negative payment covers the credit order,
  so `paidAmount == totalAmount` on both halves and each is auditable the same way.
- §11 Q3 (how a pair is identified) — **no new column**. Every id is a deterministic
  function of `(year, memberId, fromTeamId, toTeamId)`, so line ids carry the transfer
  (`transfer:{id}:{n}`), a repeat call is harmless, and a replay produces one transfer. The
  trade-off recorded there: the ids identify *a move between two teams*, not *an occasion*.

- [ ] **S1** `[shared-go]` Settle credit (negative-total) orders — the Pay saga bails on
      `TotalAmount <= 0` and `Settle` rejects a non-zero total, so such an order can never
      leave `open`. Includes the saga/`Settle` change chosen in §11 Q1.
- [ ] **S2** `[shared-go]` Introduce a typed payment method and an `internal-transfer`
      value — `method` is a free `VARCHAR` with one hardcoded literal and no type in
      `types/`.
- [ ] **S3** `[shared-go]` Payment provenance — reference the originating provider payment,
      root-preserving across repeated transfers, with an explicit "unknown" representation.
      Includes the schema decision (column vs `operations` JSON) and its migration.
- [ ] **S4** `[shared-go]` Transfer command — move a member's paid lines between two owners
      as a netting order pair, plus the `spejder.*.reassigned` event and making
      `spejder.teamId` mutable by it (and only by it). Depends on S2, S3.
- [ ] **H0** Bump `shared-go` in `go/go.mod`; verify build, replay and dev schema.
- [ ] **H1** `PUT /api/member/:memberId/reassign` — preconditions, error mapping, OpenAPI.
- [ ] **H2** `GET /api/member/:memberId/transfer-candidates` — eligibility + reason +
      transferable lines.
- [ ] **H3** Wire the reassignment projection into `projections`; verify live tokens.
- [ ] **H4** SPA — "Flyt til anden patrulje" button + dialog in `PatruljeView.vue`.
- [ ] **H5** SPA — render credits, transfer orders and seat provenance in Betalinger on
      both team pages.
- [ ] **H6** Verify replay safety end to end — a full JetStream replay must not
      double-credit a transfer.
- [ ] **X1** (separate, related) Enforce the 3-member minimum at the start gate —
      `startPatruljeHandler` has no count check today.
- [ ] **X2** (adjacent, optional) Fix the øre/kroner comparison in
      `patrulje/query.go:107-115`.

S1–S3 can be lifted in parallel. Nothing from H1 onwards can start before H0.

## 11. Open Questions

### Decisions already taken

- **hq may create the payments it initiates** (2026-09-07). Ownership is per payment, not
  per repo: tilmelding owns provider payments because it initiates them and takes the
  callback; a manual payment or an internal transfer has no provider, so hq initiating one
  owns it in exactly the same sense. `go/cmd/api/main.go:233` has been rephrased — the
  rule that stands is only that hq must not run a second copy of the Pay saga. This
  removes what was the blocking question and shrinks Phase 1 (see §8 obstacle 1).
- **The 3-member minimum is enforced at the start gate, not on the move** (2026-09-07). A
  team below three may not start; the remedy is to transfer the survivors into a team that
  has not started yet. Hence no lower-bound check on the origin, and hence the start gate
  is listed as a separate task in §10.

- **shared-go is changeable, including its schema and its sagas** (2026-09-07). Nothing
  here is constrained by "the library does it this way". The constraint is only that such
  work is specified as a **separate, self-contained task lifted to the `shared-go` repo**,
  implemented there, and consumed here via a dependency bump — which makes the task spec
  itself the deliverable, and the seam (subjects, JSON tags, signatures) the thing to get
  right. See §10.

**Design inputs to the Phase 1 task specs — answer before lifting S1 and S4:**

1. **How does the credit (negative-total) order close?** Generalise `Settle` from "free"
   to "fully covered", or teach the Pay saga about negative totals? The saga route keeps
   one settlement path for all orders; the `Settle` route keeps the saga strictly about
   money arriving from outside. Note the charge side needs neither change — tilmelding's
   existing saga will close it.
2. **Symmetrical negative payment, or a positive payment only on the charge side?**
   Symmetry makes the pair self-evidently auditable and keeps `paidAmount ==
   totalAmount` true on both orders; asymmetry keeps a never-negative `payment.amount`.
   This choice partly determines question 1.

**Needed before Phase 1 is designed:**

3. **How is a transfer pair linked and identified?** A convention on `lineId`, something
   in `order_line.attributes`, a `transferId` shared by both orders? This determines
   whether finance can pair them, and whether the UI can label them.
4. **How much provenance is enough?** The root provider payment reference is required
   (§6). Open: whether to also record each intermediate hop, whether to name the paying
   *team* as well as the payment, and what to store when the source payment cannot be
   identified at all — which is the common case, since most payments are reachable only
   through their owner (`go/cmd/api/order.go:52-57`: 151 of 189).
5. **Should the sending team's credit ever become a refund?** §4 says out of scope, so a
   credit sits as an over-paid order indefinitely. Is that acceptable to finance for a
   whole season, or does it need at least a report? The hollowed-out-team case makes this
   concrete: a team that transfers all its members away is left holding the full credit.
6. **Does a team that drops below `MinSeats = 3` paid seats keep its number?** §8 argues
   sticky acceptance is right, with the start gate as the safety net; confirm, because the
   alternative implies building an unassign path that does not exist and that PRD 003
   chose not to build.

**Lower stakes:**

7. **Where does the new event struct live** — `shared-go/messages` only, or duplicated
   into hq like `spejderstatus.TeamMoved` (`messages.go:172-180`)? The existing
   duplication looks accidental; this is a chance not to repeat it.
8. **Is `t-shirt follows the member` always right?** A t-shirt is bought by a team for a
   person; if the person leaves, presumably the shirt goes with them. Confirm with the
   people who hand out shirts — the order is authoritative for shipping
   (`order/table.go:24-42`), so getting this wrong misdirects a physical object.
9. **Should a reassignment be visible on the member's timeline** (`spejdernote`,
   `spejderstatuslog`)? Pre-race there is no status log row, so the move would leave no
   member-level trace beyond the two orders.
10. **Should this feature finally produce `MemberStatusSeated`?** It is the one place
    that knows a specific member's seat is paid for. §4 says no — but the constant exists
    for exactly this, and if not here, then arguably nowhere.
