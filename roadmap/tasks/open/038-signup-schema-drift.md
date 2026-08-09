# 038 — `signup` projection is broken by schema drift

**Status:** open
**Priority:** high
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
- **Production is unverified.** If its table also predates the columns it has the same
  break. Check before assuming dev-only.

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

- [ ] The `signup` projection applies cleanly — no `Error 1054` in the API log
- [ ] The read model contains signups created after the columns were added
- [ ] Drift is repaired automatically on boot, not by a manual `ALTER`
- [ ] Production/stage checked for the same drift, and repaired if present
- [ ] A note on whether other entities are exposed to the same gap (only `order` and
      `product` currently call `EnsureColumn`)

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
