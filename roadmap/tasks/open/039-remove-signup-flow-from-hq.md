# 039 — Remove the participant signup flow from hq

**Status:** open
**Priority:** medium
**Created:** 2026-08-09
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] Confirmed tilmelding owns participant signup, and hq's endpoints are redundant
- [ ] The three `/api/signup*` routes and their handlers removed
- [ ] Code that becomes dead is removed with them; shared code left alone
- [ ] The `signup` projection and its readers (exports, mail) still work
- [ ] `go test ./...`, `go vet ./...`, `go tool staticcheck ./...` pass
- [ ] Excel export and mail recipients spot-checked after the change

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-09 01:46 — Task created while scoping 038, after asking whether the signup
  entity belongs in this service. The projection does; the participant write flow does
  not.
