# 146 — server-side track reduction

**Status:** open
**Priority:** medium
**Created:** 2026-09-03
**Picked up by:**
**Started:**
**Completed:**

## Description

PRD 011 §6, §8. Depends on task 145.

At ~30 s sampling the ceiling for a 30-hour race is ~3,600 points per person, ~21,600 for a
six-member patrol. Sparse recording will usually make it far less, but the endpoint cannot rely
on that. Raw points at the ceiling are megabytes of JSON and a janky Leaflet map — and more
detail than any screen can show, since a 30-hour route at display zoom cannot resolve
30-second steps.

So both track endpoints reduce: a `maxPoints` parameter, either nth-point or Douglas–Peucker
(prefer Douglas–Peucker — it keeps corners, which is what makes a route legible).

Two rules:

- **Reduction is applied within a segment, never across one.** Simplification must not be able
  to bridge a gap; that would undo task 145.
- **The response states the resolution it was reduced to**, so the UI can say so and offer full
  fidelity for a narrow time window.
- **Scans are never reduced** — they are few, exact, and the anchor points an operator reasons
  from (relevant to task 149).

Testable without a database, like task 145.

## Acceptance Criteria

- [ ] Reduction function over a segment's points, honouring `maxPoints`
- [ ] Never merges across segment boundaries
- [ ] Endpoints report the applied resolution / whether reduction occurred
- [ ] First and last point of a segment always retained
- [ ] Unit tests including "fewer points than the budget" (no-op) and a corner-preservation case
- [ ] `go test ./...` clean for the package

## Progress Log

- 2026-09-03 — Task created from PRD 011 §10.
