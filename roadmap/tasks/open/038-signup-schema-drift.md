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

### Impact

Worse than noisy logs:

- **No signup has been projected since the columns were added.** The INSERT fails
  outright, so the row never appears. The table holds 1345 rows from the last time the
  projection worked, newest `createdAt` 2026-07-30 — it looks populated, which is why
  this went unnoticed.
- Anything reading the signup read model is therefore working from stale data —
  `GET /api/signup/:id`, pincode verification, and the signup flow generally.
- It repeats on every boot, because projections replay the whole stream each time.
- **Production is unverified.** If its table also predates the columns, it has the same
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
