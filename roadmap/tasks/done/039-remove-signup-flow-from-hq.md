# 039 — Remove the participant signup flow from hq

**Status:** done
**Priority:** medium
**Created:** 2026-08-09
**Picked up by:** agent session (zed)
**Started:** 2026-08-09
**Completed:** 2026-08-09

## Description

hq carries the participant registration flow, and nothing in hq uses it. It belongs to
tilmelding.

Evidence:

- `grep -rn signup vue/src` finds **no call to any `/signup` endpoint**. The only hits
  are `signupStatus` (a field on patrulje/klan) and `signupStart` (year config).
- The endpoints exist and are wired: `POST /api/signup`, `POST /api/signup/pincode`,
  `GET /api/signup/:id` (`routes.go:44-46`, `cmd/api/signup.go`).
- `cmd/api/signup.go` does rather more than expose a read: it calls
  `app.commands.Team.Signup(...)`, then publishes a `mail.…sent` event and sends the
  verification mail — with the year **hardcoded as `"2024"`** in the subject
  (`signup.go:140`), which is itself a sign this code is not being exercised.

This is a write-side flow for participants living inside the organisers' admin panel.

### Keep the read model

The `signup` **projection stays**: `export.go` (3 sites) reads `EmailPending` /
`PhonePending` for the Excel exports, and `mail.go` / `klan.go` use it to resolve
recipients. Only the participant-facing write path and its unused read endpoint are in
scope here.

### Scope

- Remove `POST /api/signup`, `POST /api/signup/pincode`, `GET /api/signup/:id` and the
  handlers behind them.
- Remove whatever becomes dead as a result — likely `commands.Team.Signup` if hq is its
  only caller, and the verification-mail template if it is used nowhere else. Check
  before deleting; do not remove a template another flow shares.
- Leave the projection, `data.Models.Signup` and every reader of it untouched.

### Care

- **Confirm tilmelding actually owns this flow** before deleting, rather than inferring
  it from hq's SPA not calling it. If neither service owns it, this is a feature being
  removed, not dead code.
- The hardcoded `"2024"` suggests the path has been broken for a while, which supports
  the reading — but is not proof anyone stopped using it.
- Anything reaching these endpoints from outside the SPA (a legacy client, a bookmark,
  another service) would break. Worth a look at access logs if any exist.

## Acceptance Criteria

- [x] Confirmed tilmelding owns participant signup, and hq's endpoints are redundant
      — the code says so itself; see the log
- [x] The three `/api/signup*` routes and their handlers removed
- [x] Code that becomes dead is removed with them; shared code left alone
- [x] The `signup` projection and its readers (exports, mail) still work
- [x] `go test ./...`, `go vet ./...`, `go tool staticcheck ./...` pass
- [x] Excel export and mail recipients spot-checked after the change

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-09 01:46 — Task created while scoping 038, after asking whether the signup
  entity belongs in this service. The projection does; the participant write flow does
  not.
- 2026-08-09 01:52 — Picked up. Plan: map every reference first (handlers, routes,
  what `commands.Team.Signup` and the mail template are used by elsewhere) before
  removing anything, so the read-side projection and any shared code are left intact.
- 2026-08-09 02:04 — The code confirms its own provenance, which settles the ownership
  question better than the SPA's silence did: both the command
  (`commands/team.go:48`) and the mail event (`signup.go:143`) set
  `Producer: "tilmelding-api"`, and both hardcode the year as `"2024"`. This is
  tilmelding's registration flow copy-pasted into the organisers' admin panel. The mail
  event also smuggled the verification secret through `Metadata.Phase`.
- 2026-08-09 02:06 — Removed: the three routes; `cmd/api/signup.go`; `Signup` from the
  `commands.Team` interface and its implementation; `verify_email.tmpl`; and two imports
  that became unused (`messages` in commands.go, `math/rand/v2` in team.go).
  **`cmd/api/signup.go` went entirely**, which deserves flagging: besides the three
  handlers, it held ~160 lines of *commented-out* product CRUD handlers (lines 159–322)
  plus a commented `homeHandler`. Nothing there was live, none of it was routed, and the
  product/order work it belonged to has since shipped through shared-go
  (`tables/product`, `tables/order`; PRD 002, tasks 001–007), so it read as superseded
  leftovers. It is in git history if any of it is still wanted.
- 2026-08-09 02:10 — Deliberately **not** touched: the `signup` projection
  (`main.go:167,212`) and every reader of it. `export.go` (3 sites) needs
  `EmailPending`/`PhonePending` for the Excel exports, and `mail.go`/`klan.go` use it to
  resolve recipients. Removing the write flow does not remove hq's legitimate interest
  in the read model.
- 2026-08-09 02:12 — ✅ Verified live rather than by compilation alone: `excel/klan` 200,
  `mail/recipients` 200, and `GET /api/signup/abc` now 404. Gates green — build,
  `go test ./...`, `vet`, `staticcheck`, gofmt.
- 2026-08-09 02:14 — Completed. Note this does **not** resolve 038: the projection hq
  keeps is still missing the `year` and `secret` columns, so contact-detail updates
  still fail to land. `secret` is now doubly redundant here — hq never reads it and no
  longer writes the flow that produces it — but the shared entity still writes it, so
  the column is still required.
