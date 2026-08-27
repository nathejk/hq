# 120 — vehicle position seam (Phase 4, blocked)

**Status:** open
**Priority:** low
**Created:** 2026-08-27
**Picked up by:**
**Started:**
**Completed:**

## Description

PRD 009 §8 ("GPS, when it comes"). **Blocked on access to the tracker feed** — there is a
tracker on each car and no access to it. Nothing else in PRD 009 depends on this task.

The seam is a vehicle position, not a dispatch concern:
`NATHEJK.{year}.vehicle.{id}.position.reported` carrying lat/long/uts, projected onto the
existing `vehicle` read model as `lastLat` / `lastLng` / `lastSeenUts`.

When it lands: a unit's last position **and its age** on the capacity strip, and a distance term
in the leg allowance. No change to `dispatch` is needed — which is the point of keeping position
off the task.

Note before promising a map: **loks have no coordinates** (`lok` is `lokId, name, sortOrder,
userIds, teamIds`), so "how far to Lok 3" is unanswerable even with the feed. Checkpoints do
have `latitude` / `longitude`.

## Acceptance Criteria

- [ ] Tracker feed access confirmed (unblocks the rest)
- [ ] Position event consumed onto the `vehicle` read model
- [ ] Capacity strip shows last position and its age
- [ ] Distance term in the leg allowance, checkpoints only
- [ ] `dispatch` unchanged

## Progress Log

- 2026-08-27 — Task created from PRD 009 §10, deliberately left blocked.
