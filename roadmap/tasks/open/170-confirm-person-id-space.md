# 170 — confirm the personnel / crewmember id space with the producer

**Status:** open
**Priority:** medium
**Created:** 2026-09-09
**Picked up by:**
**Started:**
**Completed:**

## Description

PRD 011 §11, last residual of the "person identity" question. **Not a code change here** — a
question for the hej-app repo, like task 152 was.

HQ's presence indicator and its track endpoints treat `personId` as a single opaque space
holding both a `memberID` (spejder, senior) and a `crewmemberID`. That is what lets
`track_latest` need no discriminator and lets any people-list row look itself up with the id it
already carries.

The residual: a gøgler, friend or bandit lives in HQ's `personnel` table, **not** in
`crewmember`, though both are keyed by `userId`. The indicator relies on those ids being one
space and non-colliding. If the producer ever mints a `personnel` id that collides with a
`crewmember` id, one person's positions would appear against another's name — a silent and
fairly serious mis-attribution.

Answer the same way task 152 was answered: read the hej-app, which is checked out alongside
this repo, rather than only asking. `~/Development/nathejk/hej`.

## Acceptance Criteria

- [ ] Established from the hej-app source (and confirmed with its owner) whether `personnel`
      and `crewmember` ids share one non-colliding space
- [ ] Answer recorded here and in PRD 011 §11
- [ ] If they can collide, a task exists for adding a discriminator to `track_latest` and to
      the presence payload

## Progress Log

- 2026-09-09 — Created while closing PRD 011, so the residual is not lost with the PRD.
