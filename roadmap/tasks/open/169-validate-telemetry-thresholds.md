# 169 — check the gap and staleness thresholds against real race data

**Status:** open
**Priority:** medium
**Created:** 2026-09-09
**Picked up by:**
**Started:**
**Completed:**

## Description

PRD 011 §11 and PRD 013 §11. Three numbers were chosen by reasoning rather than evidence,
each in one place, each labelled in code as a default and not a decision. After the first
real race there will be a distribution to check them against.

- `GapThresholdMs` = **5 minutes** (`nathejk/table/track`, task 145) — where a track splits
  into a new segment. PRD 011 calls this "the one number the whole track rendering hangs on":
  too small and a normal track shatters into confetti, too large and we bridge a gap we
  should have shown.
- `STALE_AFTER_MS` = **30 minutes** (`composables/usePositionPresence.ts`, task 143) — when
  the presence glyph mutes. Hour-long gaps are routine, so a tight threshold would leave most
  glyphs muted most of the time and the state would stop carrying information.
- `LIVE_WITHIN_MS` = **5 minutes** (`composables/patruljePositions.ts`, PRD 013) — when a
  position dot stops being drawn filled. A different question from the glyph: not "does this
  phone report?" but "can I act on this dot right now?".

The measurement is the same for all three: the distribution of deltas between consecutive
points, per person, across a real event.

## Acceptance Criteria

- [ ] Delta distribution extracted from `track_point` after a real race (median, p90, p99)
- [ ] Each of the three thresholds either confirmed against it or changed
- [ ] Numbers recorded in PRD 011 §11 (and PRD 013 §11 for the dot)
- [ ] If any threshold changes, the comment explaining it changes with it — these constants
      each carry the reasoning that justified the old value

## Progress Log

- 2026-09-09 — Created while closing PRD 011 and writing PRD 013, so that closing them does
  not lose the three guessed numbers. Blocked on a real event, not on work.
