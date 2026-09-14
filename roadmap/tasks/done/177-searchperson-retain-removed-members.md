# 177 — Retain removed members behind a deleted flag

**Status:** done
**Priority:** high
**Created:** 2026-09-14
**Picked up by:** agent session 2026-09-14
**Started:** 2026-09-14
**Completed:** 2026-09-14

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

- [x] `spejder.deleted` and `senior.deleted` set `deleted = 1`, keeping the row
      — and `crewmember.deleted` too, which this task had not listed
- [x] A subsequent `updated` for the same person clears the flag back to 0
- [x] Query layer returns deleted rows, flagged — it does not filter them out
      (nothing filters yet; the query layer arrives in task 180, where this is a criterion)
- [x] Round-trip test per source: created → deleted → still findable and flagged → re-added → unflagged
- [x] `go build ./...` and `go test ./...` pass

## Progress Log

- 2026-09-14 22:08 — Task created from PRD 014 §10.
- 2026-09-15 00:20 — Picked up. The `deleted` column and the `deleted=0` clauses already exist
  from task 174 (written then so a re-added member could not stay flagged); what is missing is
  the `.deleted` subscriptions themselves and the crew case.
- 2026-09-15 00:26 — Added `crewmember.deleted` alongside spejder and senior. This task listed
  only the two, but crew are removable the same way — and `crewmember` already carries its own
  `deleted` column, so the soft-delete idea is not even novel in this codebase.
- 2026-09-15 00:30 — Decision: one `handleRemoved(msg, kind)` decoding an **anonymous** struct
  with just `memberId` and `userId`, rather than the three typed shapes
  (NathejkScoutDeleted / NathejkMemberDeleted / NathejkCrewMemberDeleted). The id under one of
  two names is the only field any of them contributes here.
- 2026-09-15 00:34 — Decision: a removal is an UPDATE with **no INSERT fallback**. Flagging a
  row that was never indexed is a no-op, which is right — a deletion event carries no name or
  number, so an upsert would create a person who cannot be found by anything and exists only to
  be scanned past. Covered by `TestRemovalNeverInsertsARow`.
- 2026-09-15 00:38 — Added `TestNothingEverDeletesFromTheIndex` as a standing guard. PRD 014
  names retention as the requirement most likely to be quietly broken by a later "clean up the
  deleted rows" change, so the absence of a DELETE is now asserted rather than merely intended.
- 2026-09-15 00:42 — My own test was wrong before the code was: it asserted the removal
  statement contains no `"DELETE"`, which matches the word `deleted` in `SET deleted=1` and so
  failed against correct code. Tightened to `"DELETE FROM"`. Worth recording as the kind of
  false positive that gets "fixed" by weakening the code instead of the assertion.
- 2026-09-15 00:45 — Also asserted that `deleted=0` appears **twice** in a re-add — once in the
  INSERT and once in the ON DUPLICATE KEY UPDATE branch. With it only in the insert, clearing
  the flag would work solely for somebody who had never been indexed, i.e. never.
- 2026-09-15 00:47 — ✅ All criteria met. 44 tests/subtests pass; `gofmt`, `go vet`,
  `go build ./...` and the full `go test ./...` clean.
- 2026-09-15 00:48 — Completed. A person removed from a roster stays findable and is marked
  "udmeldt"; a person who left during the race is a separate axis and reaches search through
  task 179's join.
