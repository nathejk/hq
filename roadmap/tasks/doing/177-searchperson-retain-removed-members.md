# 177 — Retain removed members behind a deleted flag

**Status:** doing
**Priority:** high
**Created:** 2026-09-14
**Picked up by:** agent session 2026-09-14
**Started:** 2026-09-14
**Completed:**

## Description

Depends on task 176. Implements the retention decision in PRD 014 §5/§6: **a person removed
from a roster stays findable.**

`spejder.deleted` and `senior.deleted` set `deleted = 1`. They must not delete the row. A
hard delete would make search answer "ingen match" for somebody the event has actually met,
which is the outcome the whole PRD exists to prevent — the guardian of a scout who left at
02:00 rings at 09:00.

The flag is not a soft-delete for tidiness. It is the field the UI renders as **"udmeldt"**,
so it is a projected fact about the person, not an absence.

Two details that will bite:

- **A re-added member must clear the flag.** The upsert has to write `deleted` explicitly
  rather than leaving the column alone, or a member removed and re-added stays marked
  udmeldt forever.
- **This is not the same thing as withdrawing during the race.** A scout who left the route
  is still on the roster; their departure lives in `spejderstatus` and is task 179's job.
  Do not conflate the two — the UI must be able to distinguish "udmeldt" (never started)
  from "released" (went home during the night).

Retention makes the table monotonic: rows are only ever added, so `search_person` grows
across years where the source tables shrink. That is by design; nothing prunes it.

## Acceptance Criteria

- [ ] `spejder.deleted` and `senior.deleted` set `deleted = 1`, keeping the row
- [ ] A subsequent `updated` for the same person clears the flag back to 0
- [ ] Query layer returns deleted rows, flagged — it does not filter them out
- [ ] Round-trip test per source: created → deleted → still findable and flagged → re-added → unflagged
- [ ] `go build ./...` and `go test ./...` pass

## Progress Log

- 2026-09-14 22:08 — Task created from PRD 014 §10.
- 2026-09-15 00:20 — Picked up. The `deleted` column and the `deleted=0` clauses already exist
  from task 174 (written then so a re-added member could not stay flagged); what is missing is
  the `.deleted` subscriptions themselves and the crew case.
