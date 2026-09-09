# 172 — decide whether klaner get a position layer too

**Status:** open
**Priority:** low
**Created:** 2026-09-09
**Picked up by:**
**Started:**
**Completed:**

## Description

PRD 013 §4 and §11. The position layer is patruljer only, deliberately: the operational question
during the race is about the racing field.

Klaner also scan and their seniors also report positions, so the endpoint would extend naturally
— `GET /api/telemetry/positions` folds two sources per team and knows nothing patrol-specific
beyond which teams it lists.

The question is whether the picture is helped or crowded. ~200 patruljer already group into dots
at the start and the finish; adding klaner puts a second population on the same screen answering
a different question. If it is wanted, it probably wants to be a **separate checkbox** rather
than more dots in the same layer, and a different colour — green currently means "a patrol was
here", and one glyph meaning two things is how a map stops being readable.

## Acceptance Criteria

- [ ] Decision made with løbsledelse: klan positions on `/kort`, or not
- [ ] If yes: own overlay, own colour, and PRD 013 §4's non-goal updated to say so
- [ ] Decision recorded in PRD 013 §11

## Progress Log

- 2026-09-09 — Created with PRD 013, from its own non-goal.
