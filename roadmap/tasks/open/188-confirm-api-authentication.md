# 188 — Confirm who can reach /api, now that it returns minors' contact details

**Status:** open
**Priority:** high
**Created:** 2026-09-14
**Picked up by:**
**Started:**
**Completed:**

## Description

Raised while implementing task 181, whose acceptance criteria included "endpoint requires
authentication (asserted, not assumed)". That criterion **cannot be met inside hq**, and finding
out why is worth its own task.

`app.authenticate` in `go/cmd/api/routes.go` does not authenticate. It attaches an anonymous user
to the request context and calls the next handler:

```go
ctx := requestctx.WithUser(r.Context(), &requestctx.User{ID: types.UserID(""), Name: "anonymous"})
```

Its own comment says so: authentication lives in an external service (`AUTH_BASEURL`,
`lukmigind.nathejk.dk`), "which is why there is no token handling here: every request is
currently attributed to an anonymous user". So every `/api` route is reachable by anyone who can
reach the process, and whatever protection exists is in front of it — Traefik, the auth service,
or the network.

This is **pre-existing and not caused by PRD 014.** What PRD 014 changes is the stake. Before
`/api/search/person`, finding a participant's phone number required knowing which list they were
on and paging through it. Now one request with eight digits — or three letters — returns named
minors with contact numbers across the whole event, and a query of `Anders` returns fifty of
them. That is a materially different disclosure from the same data spread across a dozen
list endpoints, even though no new data was exposed.

## What this task is for

Establishing and writing down the truth, not necessarily changing code:

1. **Determine what actually guards `/api` in production.** Traefik middleware? The auth service
   as a reverse proxy? Nothing? The commented-out `app.cleo.ProxyHandler` block in `routes.go`
   suggests this has moved around before.
2. **Decide whether that is sufficient for this endpoint specifically**, and record the decision
   in PRD 014 §6 so the next person does not have to re-derive it.
3. If it is not sufficient, decide where the fix belongs — hq verifying a token, or the edge.
   Do not add a half-measure to hq that looks like protection without being it; that is worse
   than the current honest comment.

Note that `requestctx.User` being anonymous also means the audit question in PRD 014 §11
("should search be audited?") currently has no answer available: there is no identity to log.
The two questions are the same question.

## Acceptance Criteria

- [ ] What guards `/api` in production is established and written down
- [ ] A decision recorded in PRD 014 §6 as to whether that suffices for person search
- [ ] If a change is needed, it is either made or split into its own task with a clear scope
- [ ] The misleading name `authenticate` is either justified in a comment or renamed
- [ ] PRD 014 §11's auditing question is resolved or explicitly deferred with a reason

## Progress Log

- 2026-09-14 03:20 — Task created from a finding in task 181. Not a regression; a pre-existing
  condition whose stakes person search raises.
