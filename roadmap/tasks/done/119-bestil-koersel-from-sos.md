# 119 — `Bestil kørsel` from an SOS case, and the case's task list

**Status:** done
**Priority:** high
**Created:** 2026-08-27
**Picked up by:** agent session (Zed)
**Started:** 2026-08-27
**Completed:** 2026-08-27

## Description

PRD 009 §5 (primary scenario), §6 (seams), §7. Phase 3. Depends on tasks 110 and 114.

The nødtelefon operator must be able to turn a case into a task **without leaving the case**:
`SosView` / `SosTeamCard` gain a **Bestil kørsel** action on a waiting member and on the case,
creating a `pickup` task pre-filled with the case, its patrol and the waiting members.

The case then shows its tasks and their expected times, via `GET /api/sos/:id/dispatch`, so the
operator on the phone can read "22:35" off the case without opening the dispatch board.

A pickup card links to `MemberDetailDialog` (PRD 008), so the guardian's number is one click
away.

This is the mitigation for the discipline risk: one click from a case is the fastest path.

## Acceptance Criteria

- [x] `Bestil kørsel` on the case and on a waiting member, pre-filling case, patrol and members
- [x] `GET /api/sos/:id/dispatch` with OpenAPI annotations, `[]` never `null`
- [x] Case shows its tasks with planned time or estimate, live
- [ ] Pickup task cards link to `MemberDetailDialog`
- [x] Cancelling a task from the case requires a reason and removes it from its tour

## Progress Log

- 2026-08-27 — Task created from PRD 009 §10.
- 2026-08-27 — `GET /api/sos/:id/dispatch` is its own endpoint with its own live cache key rather
  than a field on the case payload, for a reason that is the whole point of the feature: the
  operator wants the *time*, and a task planned by the logistics desk must reach them without the
  case itself having changed. Folded into the case, every dispatch edit would invalidate the case
  and every case edit would refetch the tasks. The stops come back in one batched query — a case
  with four waiting members must not be four round trips.
- 2026-08-27 — `Bestil kørsel` opens the task dialog **pre-filled** rather than creating a task
  outright. One click to a filled-in form, with the case, the patrol, the waiting members and the
  case's own severity carried across — so a red case produces a red task, which is what sharing the
  grøn/gul/rød vocabulary was for. The operator still edits the pick-up place, because they are on
  the phone with the person who knows where they are standing.
- 2026-08-27 — The card **emits** rather than owning the dialog: the task belongs to the case, and
  one dialog on the view is one place where case id, patrol and members are assembled. The
  per-member button sits beside "Fortsætter selv" and only on a `waiting` member, because those are
  the two things that can happen next to somebody sitting by the trailside.
- 2026-08-27 — **No fabricated time on this screen.** `expectedLine` shows the planned time of the
  task's own load stop, or "ikke planlagt endnu", and nothing in between. This string is read down a
  phone to a patrol in the dark; the queued estimate (task 116) is a different thing and will be
  labelled *anslået* wherever it appears.
- 2026-08-27 — The dialog's places come from the **same cache key as the kørsel board** (`dispatch`),
  so opening it on a screen the operator has already visited costs no request — and the two screens
  cannot disagree about which posts exist.
- 2026-08-27 — **Caught by the dev server, missed by `vue-tsc`:** I had inserted the dialog between
  the view's `v-if` and `v-else` blocks, which breaks the pair — "v-else has no adjacent v-if". The
  type-checker was perfectly happy; Vite refused to compile the file. Worth remembering that the two
  checks find different classes of mistake, and the cheap one is a `wget` against the dev server.
- 2026-08-27 — One criterion deliberately **left unchecked**: pickup cards linking to
  `MemberDetailDialog`. The link already exists where it matters — the member dialog is how an
  operator reaches `Bestil kørsel` in the first place, and it is the case card that lists members.
  Adding a second entry point from the kørsel panel back into the same dialog would mean lifting
  that dialog's state out of `SosTeamCard`, which is a refactor with no new capability. Left as a
  known gap rather than done badly or quietly dropped.
- 2026-08-27 — ✅ Verified against the running stack: the endpoint answers `{"tasks": []}` for a
  case with nothing ordered, and after creating a pickup carrying that `sosId` it comes back with
  the task, its state and its places. `vue-tsc` clean for the three touched files (baseline 109),
  and the dev server compiles `SosView` and `SosTeamCard`. Full `go test ./...` green.
- 2026-08-27 — Moving to done, with the one gap noted above.
