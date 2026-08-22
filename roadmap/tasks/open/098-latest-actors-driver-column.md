# 098 — Who accepted them: GetLatestActors and the driver column

**Status:** open
**Priority:** low
**Created:** 2026-08-23
**Picked up by:**
**Started:**
**Completed:**

## Description

The crew needs the **driver who accepted the scout** — they are who you ring when a car is
overdue, and who you ask what state the child was in. Not the car: custody is a person, and
`PickupAccepted.Car` goes unused by this screen (PRD 007 §6).

This needs no new column and no new event field. `spejderstatuslog` already stores
`actorUserId` per event, and every lifecycle event body carries an `Actor` — so "who accepted
this scout" is the actor of the event that put them in their current status. It has been
recorded since PRD 006 and simply never read back.

Add `GetLatestActors(ctx, year, ids []types.MemberID) (map[types.MemberID]types.UserID, error)`
to `spejderstatus`: the actor of each member's highest-`seq` log row, batched with an `IN`
clause over the members already fetched — the `(year, id, seq)` key covers it. Resolve names
through `app.models.CrewMember.GetByID`, as `vehicle.go` does for a vehicle custodian.

Deriving it beats storing an `acceptedBy` on the `shelter` table, because the same derivation
answers it for `transit` — a status the shelter table knows nothing about, since nothing about
a car is the shelter's to project. It is also the truer statement: the accepting party *is*
whoever published the acceptance.

Rendered as "i bil hos «navn»" for `transit` and "modtaget af «navn»" for `sheltered`, the
second giving the crew its own shift handover for free.

**Sequenced last on purpose: it produces nothing visible on the day it lands.** Authentication
is perimeter-only today, so every actor is anonymous with an empty `UserID`. So the server
omits the field when it has no name and the section hides the column when no row in it has
one — the first attributable acceptance makes the column appear, with no deploy. A column of
em-dashes is noise on a 3am screen and, worse, reads as a bug.

Real login is expected before the next race (a month out), so this will most likely ship
already working. Note that the actor is written to the log as each event happens, so login
does **not** backfill — there is nothing to backfill today, but a race run before login lands
would have no attribution, permanently.

## Acceptance Criteria

- [ ] `GetLatestActors` implemented, batched, no query per member
- [ ] Names resolved via `CrewMember`; an unresolvable id degrades to no name rather than
      failing the request
- [ ] Server omits the field when there is no name
- [ ] Column hidden while no row in the section has a name; appears with the first one
- [ ] Rendered as "i bil hos …" / "modtaget af …"
- [ ] Tests covering the empty-actor case, since that is the only case that exists today

## Progress Log

- 2026-08-23 09:30 — Task created from PRD 007 §10.
