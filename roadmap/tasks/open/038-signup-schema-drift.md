# 038 — `signup` projection is broken by schema drift

**Status:** open
**Priority:** low
**Created:** 2026-08-09
**Picked up by:**
**Started:**
**Completed:**

## Description

The `signup` projection fails on **every** event, continuously, in dev today. Spotted
in the API logs while verifying task 033.

```
INSERT INTO signup SET teamId="40c0…", year="2026", teamType="patrulje", … 
  → Error 1054: Unknown column 'year' in 'field list'
UPDATE signup SET secret="1e4c…" WHERE teamId="40c0…"
  → Error 1054: Unknown column 'secret' in 'field list'
```

### Cause

The live table is missing two columns the entity writes:

| | columns |
|---|---|
| live `signup` table | `teamId teamType name emailPending email phonePending phone pincode createdAt` |
| `shared-go/tables/signup/table.sql` | the same **plus `year` and `secret`** |

The table was created by an older schema. `CREATE TABLE IF NOT EXISTS` is a no-op on
an existing table, so the two added columns were never applied, and nothing repairs
it: `signup`'s `New()` does not call `cqrs.EnsureColumn` — only `order` and `product`
do.

### Does hq even need this entity?

Asked, and worth the answer, because it changes the fix. Two halves, with opposite
verdicts:

**The read model is genuinely used by hq** — for team contact details:

- `export.go` (3 sites) — Excel exports of klaner, patruljer and personnel read
  `signup.EmailPending` / `signup.PhonePending`
- `mail.go:88` and `klan.go:226` — resolving recipients

So hq legitimately consumes signup events. Keep the projection.

**The write endpoints look like another service's job.** `POST /api/signup`,
`POST /api/signup/pincode` and `GET /api/signup/:id` implement the participant
registration flow — including sending the verification mail — and **nothing in the SPA
calls them**: `grep -rn signup vue/src` finds only `signupStatus` (a field on
patrulje/klan) and `signupStart` (year config). That flow belongs to tilmelding. Task
039 covers removing them; not this ticket.

That split also explains the drift: hq's table predates columns added for *tilmelding's*
needs. `secret` is a verification-flow concern hq never reads, and hq's only query is
`GetByID(teamId)` with no year filter — yet the shared entity writes both, so hq's
table must have them.

### Impact

> **Superseded 2026-08-09** — only dev reuses its database; other environments are
> cleared before deploy, so this is a dev-only defect. See the rescope in the log.

Measured rather than assumed, and less bad than first stated — but a trap:

- **No current team is missing.** `patrulje`/`klan` LEFT JOIN `signup` shows 0 orphans
  of 718 and 230. The 1345 existing rows predate the drift and cover everything.
- **But nothing lands any more.** The `INSERT … ON DUPLICATE KEY UPDATE` fails on the
  `year` column, so it is not only new signups that are lost — the `UPDATE` half never
  applies either. A team that corrects its email or phone after the drift began is
  invisible to hq's mail and exports.
- So the damage is *pending*, not historical: the next signup or contact change during
  an event silently does not reach hq. That is worse than a visible gap.
- It repeats on every boot, because projections replay the whole stream each time.
- ~~**Production is unverified.** If its table also predates the columns it has the same
  break. Check before assuming dev-only.~~ — resolved: prod/stage clear the database
  before deploy, so they always create this table from the current schema.

## Options

1. **`cqrs.EnsureColumn` in shared-go's `signup.New()`** — the mechanism the layout
   skill prescribes for exactly this, already used by `order` and `product`. Repairs
   every environment on next boot, and prevents the next occurrence. Needs a shared-go
   change plus a version bump here.
2. **Drop the table and let replay rebuild it.** Projections are derived data and
   replay from the beginning on every boot, so dropping `signup` makes the next start
   recreate it with the full schema and refill it. Fast, no code change — but manual,
   per environment, and undocumented unless recorded.
3. **Per-build databases (PRD 005 §8)** make this class of bug impossible, since every
   build creates its schema fresh. Right long-term answer; does not fix today.

Recommended: (1) as the fix, with (2) as the immediate unblock in dev. They compose.

## Acceptance Criteria

- [x] The `signup` projection applies cleanly — no `Error 1054` in the API log
- [x] The read model contains signups created after the columns were added
- [ ] Drift is repaired automatically on boot, not by a manual `ALTER` — **open, and
      now a dev-only question**; see the 2026-08-09 rescope below
- [x] ~~Production/stage checked for the same drift~~ — **void**: those environments
      clear the database before deploy, so drift is structurally impossible there
- [x] A note on whether other entities are exposed to the same gap

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-09 01:34 — Task created. Found while verifying 033; evidence gathered by
  comparing `SHOW COLUMNS FROM signup` against `shared-go/tables/signup/table.sql`.
- 2026-08-09 01:42 — Scoped after asking whether hq should own this entity at all.
  Answer: keep the projection (exports and mail read it), drop the write endpoints
  (nothing in the SPA calls them — task 039). Impact re-measured: no current team is
  missing a signup row, so this is a pending failure rather than historical data loss —
  the ON DUPLICATE KEY UPDATE half also never applies, so a contact-detail change after
  the drift is invisible to hq. Corrects the overstated "no signup has been projected"
  claim in the original description.
- 2026-08-09 — **Rescoped: high → low.** Only dev reuses its database; every other
  environment is cleared before deploy, so `CREATE TABLE` always runs against an empty
  schema there and this drift cannot occur. That voids the "production is unverified"
  risk that drove the original priority — the worst case was never a silent data gap
  during an event, only a broken dev environment.

  **But it also means dev is permanently exposed, by design.** PRD 005 §8 fixes this
  class for prod via per-build databases, and explicitly *excludes* dev: "Dev pins to a
  single fixed database name — `init-dev` restarts on every file change and must not
  create a database per restart." So after PRD 005 ships, dev will be the only place
  this bug can still happen, with nothing to repair it.
- 2026-08-09 — **Dev unblocked, and the diagnosis verified end to end** rather than
  assumed. Dropped the table and restarted the api so replay could rebuild it:
  - schema now has all 11 columns, `year` and `secret` included
  - **1345 rows and 399 with contact details — identical to before the drop**, which
    is the useful part: it proves replay is authoritative and that dropping a
    projection costs nothing but boot time
  - all 1345 rows now carry a `year` and 493 a `secret`; both writes previously
    failed outright, so this is direct proof the projection was dropping data
  - **0 `Unknown column` errors since restart**, against 17,336 in the preceding 14
    hours (bursts of 2167 per process start — one full replay each, no live events
    in between, confirming the pending-not-historical reading)
  - readers spot-checked: `excel/klan`, `excel/patrulje`, `mail/recipients` all 200
- 2026-08-09 — **Exposure survey (the last open AC).** The repair mechanism is the
  exception, not the rule:
  - shared-go: **2 of 10** entities call `EnsureColumn` (`order`, `product`)
  - hq-local: **1 of 17** `table.sql` files has any `EnsureColumn` caller
  So 8 shared entities and 16 local ones have no repair path. Any column added to any
  of them silently fails to appear in a dev database that already has the table.
- 2026-08-09 — **A second instance of the class already exists**, and it is a different
  shape worth distinguishing:
  - `signup` — `table.sql` *has* the columns, the live table does not. Caused by
    `CREATE TABLE IF NOT EXISTS` on a pre-existing table. Fixed by drop-and-replay or
    `EnsureColumn`.
  - `spejderstatus` — the *struct* (`spejderstatus.go:15-16`) declares `InitialTeamID`
    and `CurrentTeamID`, but `spejderstatus.sql` declares only
    `id, year, status, updatedAt`. Dropping the table would not help; the `.sql` itself
    is behind. **Zero impact today** — `Consumes()` returns an empty slice and
    `HandleMessage` is entirely commented out, so nothing writes those fields — but it
    is dormant scaffolding for exactly the member-reassignment work PRD 001 specifies,
    and PRD 001 will hit it the moment it wires the consumer up.

## Options, revised

The original (1) → (2) recommendation no longer holds, because (2) has now been done
and prod turned out never to have been at risk. What remains is purely about not
losing dev hours to the next column addition:

1. **`EnsureColumn` in `signup.New()`** — two lines plus a shared-go version bump.
   Self-healing, but only for the one entity, and 24 others would still be exposed.
   Fixes the instance, not the class.
2. **Do nothing; document the habit.** "If a projection logs `Unknown column`, drop the
   table and restart." Cheapest, but relies on someone reading 17k log lines to notice
   — the failure is invisible in the UI, which is what made this cost time in the
   first place.
3. **Dev drops its projection tables on boot.** The strongest option, and it is nearly
   free: dev *already* replays the entire stream on every start, as the 2167-event
   bursts show. The read model is already a build artefact in dev, so recreating the
   schema each boot adds a `DROP TABLE` per entity and no extra replay. That closes the
   whole class in the one environment PRD 005 leaves exposed, and makes dev behave like
   prod. Needs checking against boot time, and against `product.Seed()` (seeded rather
   than derived, but re-seeded on boot, so it should be fine).

Recommended: **(3)**, as a small addition to PRD 005 §8 rather than a shared-go change
— it is the only option that fixes the class rather than this one instance, and it
removes the need for `EnsureColumn` anywhere. Fall back to (1) if per-boot drops turn
out to cost real time.
