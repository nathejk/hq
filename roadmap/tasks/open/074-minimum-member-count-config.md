# 074 — Move the 3-member minimum into configuration

**Status:** open
**Priority:** low
**Created:** 2026-08-17
**Picked up by:**
**Started:**
**Completed:**

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

## Notes

- Also unresolved and worth asking at the same time: does the requirement apply to klaner
  at all? The rule as stated is patrol-specific and klaner presumably have their own or
  none — which may mean the klan/badut/mail `1` is not a minimum but a placeholder.
- Do **not** let this block the below-strength UI (task 077). A named constant in one place
  is enough to build against; this task replaces it with configuration.
- The number is served to the frontend in `TeamConfig`, so once configurable the SPA
  needs no change — it already reads `minMemberCount`.

## Acceptance Criteria

- [ ] Where the minimum lives confirmed with the product owner and recorded in the log
- [ ] Value read from configuration rather than a literal, at all four call sites
- [ ] Whether the rule applies to klaner answered and reflected
- [ ] Below-strength logic reads the configured value
- [ ] `go build ./... && go vet ./...` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006. Needs a product decision before implementation.
