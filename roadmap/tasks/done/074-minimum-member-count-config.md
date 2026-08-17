# 074 — Move the 3-member minimum into configuration

**Status:** done
**Priority:** low
**Created:** 2026-08-17
**Picked up by:** agent session
**Started:** 2026-08-17
**Completed:** 2026-08-17

> **Closed without code changes.** The premise turned out to be wrong — see the log. Kept
> rather than deleted because the reasoning is worth finding again if somebody proposes
> year configuration for this a second time.

## Description

From **PRD 006** §6 and §11.

`MinMemberCount: 3` is a literal in a handler at `go/cmd/api/patrulje.go:99`. It is a
**rule of the event**, and the whole below-strength feature reads it, so it needs a home.

Note the same `TeamConfig` struct is built in four places, three of which pass `1`:

- `cmd/api/patrulje.go:99` — `MinMemberCount: 3`
- `cmd/api/klan.go:170` — `1`
- `cmd/api/badut.go:48` — `1`
- `cmd/api/mail.go:59` — `1`

So this is four call sites, not one, and the value differs by team type.

**Open question to settle first (PRD 006 §11):** where it lives. Year configuration seems
the natural home since it is a rule of the event — the `year` entity already carries the
edition's dates, cities and headline. Confirm, and confirm whether the minimum has ever
differed between years or between ligaer.

## Acceptance Criteria

- [x] Where the minimum lives confirmed with the product owner and recorded in the log
- [x] ~~Value read from configuration rather than a literal, at all four call sites~~ —
      not wanted, see log
- [x] Whether the rule applies to klaner answered and reflected
- [x] Below-strength logic reads the configured value — it reads `minMemberCount` from the
      API's `TeamConfig`, which is already per team type
- [x] `go build ./... && go vet ./...` clean (no changes made)

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006. Needs a product decision before implementation.
- 2026-08-17 — **Closed by product decision: the minimum is a property of the team type,
  and it is not enforced programmatically.** For patruljer — the only type this flow
  covers — it is 3; klaner and gøglere are 1.

  So the task's framing was wrong in two ways. It read the four differing literals as
  duplication to be consolidated, when they are in fact the per-team-type values correctly
  expressed. And it assumed the number needed a configuration home because something
  branched on it — nothing does: **no command rejects anything on account of it.** The
  number is displayed, an operator applies it or judges an exception warranted, and the
  tool records what happened either way.

  That makes year configuration ceremony around a per-type constant that nothing reads
  programmatically. Dropped. `cmd/api` already serves it to the SPA as `minMemberCount` in
  `TeamConfig`, which is all the below-strength warning needs, and the frontend tasks were
  already written to read it from there rather than hardcode 3.
- 2026-08-17 — Also closes the related open question "does the requirement apply to
  klaner?": the minimum is per type, it is informational, and klaner are not handled
  through the nødtelefon at all.
- 2026-08-17 — No code written. PRD 006 §6 and §11 amended to record the decision so the
  next reader does not re-open it. Moving to done as resolved.
