# 048 — Patrol detail: include the patrol's SOS cases

**Status:** done
**Priority:** medium
**Created:** 2026-08-11
**Picked up by:** agent session
**Started:** 2026-08-11
**Completed:** 2026-08-11

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

- [x] `GET /api/patrulje/:id` includes the patrol's cases (id, headline, createdAt,
      status), newest first
- [x] Soft-deleted cases excluded
- [x] A patrol with no cases returns an empty list, not null
- [x] `go build ./... && go vet ./...` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-11 — Created from PRD 001 on approval.
- 2026-08-11 — Picked up. The query (`GetByTeam`) already existed from task 043, so this
  was only the handler.
- 2026-08-11 — A failed case lookup is logged and the page still renders. The case list is
  *context* on somebody else's page; losing it should not take down a patrol's own screen,
  which is what a `ServerErrorResponse` here would do.
- 2026-08-11 — ✅ Verified against the dev stack: a patrol with no cases returns
  `"sosCases": []` (empty list, not null — the SPA can iterate it without a guard), and
  after creating a case and associating the patrol it appears in the payload. The
  soft-deleted case from task 047's testing is correctly absent.
- 2026-08-11 — All criteria met. The card that renders this is task 052. Moving to done.
