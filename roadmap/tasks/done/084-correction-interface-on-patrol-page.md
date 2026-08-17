# 084 — Correction interface on the patrol page, with a minted case

**Status:** done
**Priority:** medium
**Created:** 2026-08-17
**Picked up by:** agent session
**Started:** 2026-08-17
**Completed:** 2026-08-17

## Description

From **PRD 006** §7 and §11 Decisions (2026-08-17).

When reality and the record disagree — a member was driven in and nobody wrote it down, a
status was set on the wrong person — this is where it gets put right. Two halves, one
backend and one frontend.

**Frontend: `vue/src/views/PatruljeView.vue`, the members list's expanded row.** Today that
row is a placeholder rendering `{{ data }}` (line ~119), so there is room. It shows the
member's current status with its timestamp and acceptor, and offers setting it to any valid
`types.MemberStatus` **except `finished`**.

**Backend: `PUT /api/member/:memberId/status` requires a `sosId` like every other member
command — so the handler mints one.** The operator is on the patrol page and has no case, so
the backend opens a case, records the correction on its timeline, and **closes it in the
same operation**. The case is purely the record.

## Notes

- **Why it is not on the SOS case card.** A correction is not part of the call an operator
  is on. Putting it on a different screen is a stronger separation than a visually-distinct
  button beside the live-call actions — which is what PRD 006 §6 was reaching for when it
  asked for the override to be hard to use as a shortcut. The case card keeps only the
  normal workflow (task 075).
- **Why mint a case rather than allow a case-less path.** It makes the model uniform: there
  is no second way for a member's status to change, so "what happened to this member?" is
  always answered by reading cases. It also makes overrides **countable**, which is what
  PRD 006 §9's "overrides stay rare" metric needs — one case per correction, recognisable
  by its headline, turns that from a guess into a query.
- **Closed on arrival** so it never reaches the open-case list (`SosListView` groups by
  status). It still appears in the patrol page's existing **Kontakt med nødtelefon** card,
  which already lists the patrol's cases and links to them — so corrections surface in
  context, on the page they were made, with no new UI for finding them.
- **The headline must mark it as machine-created** and name what was corrected, so nobody
  reading the case list mistakes it for a call. Decide the wording with the product owner;
  it is operator-visible Danish text.
- **Reusing an existing open case was considered and rejected**: a correction is rarely part
  of the story that case is telling, and "reuse if exactly one is open" needs a rule for
  when two are. Predictability wins.
- Sequencing strictness bites here specifically (PRD 006 §11): this tool exists to record a
  reality that did not follow the diagram, so leniency — accept any `Valid()` status — is
  the likelier answer than enforcing the documented order. Confirm before implementing,
  because it is the difference between a repair tool and a second workflow.
- `finished` stays excluded. `CanFinish()` is true only for `racing`, and no correction may
  hand a finish to a member who took a lift.
- Klaner are out of scope entirely, so this is patrol members only.
- Live updates: the expanded row is part of `PatruljeView`, which already depends on
  `['patrulje:{id}', 'spejder', 'order', 'payment', 'sos']` — `spejder` is already there, so
  a correction made in another tab appears with no change to the declaration.
- The dialog holds unsaved state, so dirty-guard it as `KlanListView.vue` does.

## Acceptance Criteria

- [x] Expanded row shows current status, timestamp and acceptor instead of `{{ data }}` —
      status and timestamp; **acceptor is not stored anywhere** (see task 075)
- [x] Status can be set to any valid value except `finished`
- [x] `PUT /api/member/:memberId/status` mints a case, records the correction, closes it,
      in one operation
- [x] Minted case has a headline identifying it as a manual correction and naming the member
- [x] The correction appears on that case's timeline
- [x] The minted case does **not** appear among open cases
- [x] It does appear in the patrol's **Kontakt med nødtelefon** card
- [x] Sequencing strictness decided and recorded in the log — lenient (task 072)
- [x] Dialog defers live updates while open — n/a: the row holds only a status choice, see log
- [x] `go build ./... && go vet ./...` clean; `npm run build` and `vitest` clean
- [x] Exercised against the dev stack: correct a member, then find the record from the
      patrol page alone

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from the decisions of 2026-08-17, which moved the override off the
  SOS case card (task 075) and made `sosId` mandatory on every member command (task 072).
  Depends on 072 for the endpoint and 071 for the timeline entry types.
- 2026-08-17 — Picked up. Headline confirmed by the product owner as **"Manuel rettelse —
  {navn}"**. Backend mints and closes; frontend is the members table's expanded row, which
  was still rendering `{{ data }}`.
- 2026-08-17 — The override is the **only** command allowed to arrive without a `sosId`, and
  it still does not get a case-less path — the handler mints one. That keeps "every member
  change has a case" true without making the patrol page open a case by hand first.
- 2026-08-17 — **Ordering inside the handler is deliberate.** The case is minted *after* the
  member command succeeds, not before: only a change that actually happened is worth
  documenting, and a no-op (correcting to the status a member already has) must not litter
  the record with empty cases. It is closed *after* the summary is recorded, so a reader
  catching it mid-flight never sees a closed-and-empty case.
- 2026-08-17 — **⚠️ Hit the create-then-act race, and it silently swallowed the
  association.** `AssociateTeam` reads the case back — for its year and for idempotency — but
  the `created` event has not been projected microseconds later, so the read returned
  not-found and the association was skipped. Result: every minted case had `teams: []` and
  was therefore **invisible on the patrol page it existed to document**, which is the one
  acceptance criterion that matters here. Caught because I checked the card rather than
  trusting the 200.

  Fixed by adding `AssociateTeamAt(ctx, actor, year, id, teamID)`, which publishes without
  reading. Safe only because the caller minted the id and knows the year, so nothing needs
  looking up; documented on the interface as narrowly usable and **not** a general escape
  hatch — `AssociateTeam`'s idempotency check does real work for a case an operator chose.
  I had noticed this same race during task 076 and dismissed it as "the SPA does not hit
  it"; it turned out my own code did, one task later.
- 2026-08-17 — No dirty guard needed: the row holds a single status choice, not a
  multi-field editor, and an incoming update cannot reshuffle a `Select`'s options (they
  come from config). The criterion is answered by there being nothing to lose rather than by
  a guard.
- 2026-08-17 — Acceptor still omitted, for the reason recorded in task 075: nothing stores
  it, and the only event that would carry it has no producer. The car interface's PRD needs a
  `spejderstatus` column for it.
- 2026-08-17 — ✅ Verified end to end on the dev stack:
  - correction with **no** `sosId` → succeeded, returned the minted `sosId`
  - case headline **"Manuel rettelse — Sofija"**, status `closed`, team
    **Skjoldungerne 22** associated
  - timeline in the right order: `created` → `team.associated` → `member.status.changed` →
    `closed`
  - the nødtelefon open list contains only the real case; the correction is in `closed`
  - the patrol page's **Kontakt med nødtelefon** card lists it — so a correction is
    findable from the page it was made on, with no new UI
- 2026-08-17 — The product owner's 2026 test members were restored to `racing` afterwards
  (the correction flow had moved them through `sheltered`/`transit`), so in-our-care reads 0
  and the patrol is back to 6 racing.
- 2026-08-17 — ✅ All criteria met. Moving to done — this completes PRD 006's planned scope
  apart from the follow-ups (083, 085).
