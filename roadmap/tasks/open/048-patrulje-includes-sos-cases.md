# 048 — Patrol detail: include the patrol's SOS cases

**Status:** open
**Priority:** medium
**Created:** 2026-08-11
**Picked up by:**
**Started:**
**Completed:**

## Description

From **PRD 001** §6/§8. The patrol page shows the cases the patrol is involved in
("Kontakt med nødtelefon"). Delivered by **extending `GET /api/patrulje/:id`**, which
already assembles members, payments and orders (`go/cmd/api/patrulje.go:85-96`), rather
than by adding a by-team endpoint — one request, and the SPA only has to add `'sos'` to
that view's existing `dependsOn`.

Port legacy `data.SosModel.GetByTeam` (`_go/internal/data/sos.go:33`) as the query, over
the new `sos_team` + `sos` tables, excluding soft-deleted cases.

Depends on 043 and 046.

## Acceptance Criteria

- [ ] `GET /api/patrulje/:id` includes the patrol's cases (id, headline, createdAt,
      status), newest first
- [ ] Soft-deleted cases excluded
- [ ] A patrol with no cases returns an empty list, not null
- [ ] `go build ./... && go vet ./...` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-11 — Created from PRD 001 on approval.
