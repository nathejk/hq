# 171 — decide whether a klan's track behaves like a patrol's

**Status:** open
**Priority:** low
**Created:** 2026-09-09
**Picked up by:**
**Started:**
**Completed:**

## Description

PRD 011 §11. A product decision, small to implement either way.

The spejder rule is specified and built: a patrulje's track is *every* member's track — current
and former, each clipped to the interval they belonged to the team — plus the team's QR scans
(task 149). A klan has seniors and also scans.

Open: should a klan aggregate its seniors the same way, or is one senior's own track enough? The
underlying difference is that a patrol moves as a unit by design, whereas a klan's seniors may
legitimately be in several places at once — which would make an aggregated klan track a picture
of nothing in particular.

## Acceptance Criteria

- [ ] Decision made with løbsledelse: aggregate a klan's seniors, or show individuals only
- [ ] Decision recorded in PRD 011 §11
- [ ] If aggregating: `/api/telemetry/klan/:teamId/track` mirrors the patrulje endpoint,
      reusing the membership-interval clipping rather than reimplementing it

## Progress Log

- 2026-09-09 — Created while closing PRD 011, so the question is not lost with the PRD.
