# 018 — Resolve crew member names for order owners

**Status:** done
**Priority:** high
**Created:** 2026-07-31
**Picked up by:** agent (opus-4.8 session)
**Started:** 2026-07-31
**Completed:** 2026-07-31

## Description

Orders owned by a crew member show a raw UUID instead of a name in the "Ejer"
column of Ordrehistorik. `resolveOwnerName` in `go/cmd/api/order.go` only looks
up `patrulje`, `klan`, and `personnel` — but crew members are projected into
their own `crewmember` table (`go/nathejk/table/crewmember/`, `userId` is a
generated UUID), so the personnel lookup misses and the code falls back to the
raw `ownerId`.

Fix: resolve `types.TeamTypeCrew` owners via `app.models.CrewMember.GetByID`
(already wired into `data.Models`). Also cross-fall-back between the personnel
and crewmember projections so a person owner resolves regardless of which
projection they happen to live in.

## Acceptance Criteria

- [x] Crew-owned orders resolve via `CrewMember.GetByID` (name, not UUID).
- [x] Gøgler/other person owners still resolve via personnel.
- [x] Unknown ids still fall back to the raw ownerId (no crash).
- [x] `go build`/`go vet`/`staticcheck` green.

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-07-31 17:45 — Task created + picked up.
- 2026-07-31 17:50 — Added a `types.TeamTypeCrew` case to `resolveOwnerName` using `app.models.CrewMember.GetByID`, with a personnel fall-back; and made the `default` (gøgler/other) case fall back to crewmember too, so a person owner resolves from whichever projection holds them. Memoisation cache still keyed by ownerType+ownerId. build/vet/staticcheck clean. Not verified against live data here. Completed.
