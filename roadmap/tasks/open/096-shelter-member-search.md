# 096 — Search for a scout in no section

**Status:** open
**Priority:** medium
**Created:** 2026-08-23
**Picked up by:**
**Started:**
**Completed:**

## Description

A car arrives with a scout nobody recorded as withdrawn: they are still `racing`, so they
appear in no section of the Hønsegården screen. This is the case that most needs to work at
3am, and the one a "only members already in our care" filter would block outright (PRD 007
§5).

A search field on the screen finds any scout of the year by name or patrol number, and offers
**Modtaget i Hønsegården** on the result behind the same confirm as accepting from *Afventer
afhentning* — it asserts an arrival the platform has no pickup for.

This is also the mitigation for the standing risk in PRD 007 §8: nothing in hq publishes
`transit` except the nødtelefon's manual override, so until the car interface exists the
arrivals queue can be empty while the crew is receiving people. Without this field the screen
would have nothing to offer them.

Check whether an existing endpoint can serve the lookup before adding one — the mail
recipients and patrol list endpoints already enumerate members. Prefer reuse; if a new
endpoint is needed it must carry OpenAPI annotations.

## Acceptance Criteria

- [ ] Search by scout name and by patrol number
- [ ] Results show status, so the operator sees what they are about to override
- [ ] **Modtaget** available from a result, behind a confirm
- [ ] Accepting a `racing` scout works end to end and lands them in *I Hønsegården*
- [ ] No new endpoint without OpenAPI annotations

## Progress Log

- 2026-08-23 09:30 — Task created from PRD 007 §10.
