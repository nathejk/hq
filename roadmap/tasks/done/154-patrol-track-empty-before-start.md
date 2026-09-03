# 154 — patrol track map is empty for a patrol that has not started

**Status:** done
**Priority:** high
**Created:** 2026-09-03
**Picked up by:** agent session
**Started:** 2026-09-03
**Completed:** 2026-09-03

## Description

Reported by knj against the shipped PRD 011 work: a spejder shows a position glyph, but opening
the patrol map says "Ingen positioner og ingen scanninger i det valgte tidsrum."

Two independent bugs, both introduced by me. Diagnosed against dev data for
`spejder 1` (`c9ce6de0-c12c-4ec8-9de2-22645fb6f898`, team `e4ccb553-…`, year 2026):

### 1. `TeamMemberships` misses current members entirely (task 148)

The scout has a `spejder` row, **no `spejderstatus` row and zero `spejderstatuslog` rows** —
because the patrol has not started ("Ikke startet" in the UI), so no lifecycle event has ever
been published for them.

`TeamMemberships` reads only the log and `spejderstatus`, so it returns **no members at all**,
and the map is empty however wide the window.

Task 148 reasoned correctly that `spejder` cannot be the *only* source, because
`NATHEJK.*.spejder.*.deleted` hard-deletes the row and a withdrawn scout would vanish. I
over-corrected and made it *no* source. The right model is a **union**: `spejder` is the current
roster, the log adds everyone who has since left. Before a race the log is empty and `spejder` is
all there is — which is the ordinary state for most of the season.

### 2. The default time window hides data that exists (task 150)

The only telemetry point in dev is from 2026-08-28 14:07; it is now 2026-09-03. The dialog
defaults to a window anchored on `Date.now()`, so the map is empty by construction.

The default is wrong in principle, not just for this fixture. "Recent" is the right frame *during*
a race and guarantees an empty map at every other time — including any later review of what
happened. Reduction already caps a response at 2,000 points, so asking for the whole track is
cheap; the presets should narrow from there rather than out.

## Acceptance Criteria

- [x] `TeamMemberships` returns current members from `spejder` even with no lifecycle events
- [x] Former members (log-only, `spejder` row deleted) still returned — task 148's case must not regress
- [x] A member appearing in both sources is returned once
- [x] The track dialog defaults to the whole track
- [x] Verified against the reported patrol in dev: the map shows the scout's track
- [x] `go test ./...` and the frontend suite pass

## Progress Log

- 2026-09-03 — Task created from knj's report, with both root causes diagnosed against dev data.
- 2026-09-03 — Fixed the membership query by making it a **union of two sources**, renaming `signedUpWithoutHistory` to `rosterWithoutHistory` to say what it now does. One SQL `UNION` (not `UNION ALL`, so a member in both is returned once) over `spejderstatus` and `spejder`, still skipping anyone the interval walk already covered — an extra open-ended interval for a member with history would put a moved-away scout back on this patrol's map for the whole race.
- 2026-09-03 — The reasoning is worth stating because it is the second time I have got this boundary wrong in opposite directions. `spejder` alone is wrong: withdrawn scouts are hard-deleted and would vanish, which is what task 148 correctly avoided. History alone is wrong too: before a patrol starts *nothing has happened* to its members, so there are no log rows and often no status row either. **Neither source is sufficient; the union is.** Both halves are now documented at the top of `TeamMemberships` and in `rosterWithoutHistory`.
- 2026-09-03 — Changed the dialog's default window from six hours to **the whole track**, and moved "Hele løbet" to the front of the presets. The old default was wrong in principle, not just for this fixture: anchored on `Date.now()`, it is right only *during* a race and guarantees an empty map at every other time — including any later review, which is one of the two reasons this feature exists. Asking for everything is cheap because reduction caps a response at 2,000 points whatever the span, so presets now narrow from the whole track rather than out from a slice, and narrowing is what buys detail.
- 2026-09-03 — Deleted a test I had just written (`TestRosterSkipsMembersWithHistory`) after noticing it only asserted things about its own fixture map and nothing about the code. A test that cannot fail when the code breaks is worse than no test, because it reads as coverage. The SQL-level behaviour is verified against the database instead.
- 2026-09-03 — ✅ **Verified against the exact patrol from the report** (`e4ccb553-…`): the endpoint now returns **3 members**, with `spejder 1` carrying 1,202 points across 2 segments — previously 0 members. `go test ./...`, `vite build` and the frontend suite (229 tests) all pass.
- 2026-09-03 — Completed. Worth noting what this says about the earlier verification: task 149 was checked against a *2025* patrol that had started, so it exercised the log path and never the pre-start path — which is the state most patrols are in for most of the season. A passing check on data that happens to avoid the broken branch is how this reached knj.
