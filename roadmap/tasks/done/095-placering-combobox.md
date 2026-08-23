# 095 — Placering combobox with self-defining suggestions

**Status:** done
**Priority:** medium
**Created:** 2026-08-23
**Picked up by:** agent session (Zed)
**Started:** 2026-08-23
**Completed:** 2026-08-23

## Description

The zones scouts are kept in **are not known until race start**, so they are neither
configured nor hardcoded (PRD 007 §6). The placering field is an editable combobox whose
suggestions are the distinct placeringer already recorded this year — `placements` in the
`/api/shelter` envelope (task 087's `DistinctPlacements`), most-used first — with free text
still accepted.

The first scout into a tent is typed; every one after that is a pick. That is what stops
"Telt 4", "telt4" and "t4" becoming three places, with no zone entity, no admin screen and
no setup step on the night.

**Unsaved state must never be overwritten.** This is an editor, so while it is dirty,
incoming live payloads are deferred and applied when the edit ends, and the screen says
updates are paused — exactly as `KlanListView.vue` and `KortView.vue` do. A crew member
typing a tent number at 3am must not have the field yanked out from under them because
somebody else's scout changed status.

Max 64 characters, trimmed, matching the server's validation.

A typo becomes a suggestion — accepted: ordering by use count keeps the real zone at the top
and the mistake at the bottom, and correcting the affected scout's placering removes it. No
rename tool in v1.

## Acceptance Criteria

- [x] Editable combobox; suggestions from the payload, most-used first
- [x] Free text accepted and submitted unchanged (bar trimming)
- [x] Dirty state defers incoming payloads and applies them when the edit ends
- [x] The screen states that updates are paused while editing
- [x] 64-character cap, trimmed
- [x] Manually verified: typing a new zone makes it a suggestion for the next scout

## Progress Log

- 2026-08-23 09:30 — Task created from PRD 007 §10.
- 2026-08-23 18:20 — Picked up. Read `KlanListView`'s dirty-defer for the house pattern: a
  `dirty` flag, a watch that drops incoming payloads while set, and an apply on save.
- 2026-08-23 18:30 — Payload now mirrored into `applied` and everything renders from there, so
  incoming revalidations can be held back. **Why this page needs it at all, given it is almost
  read-only:** if a payload landed while somebody was typing a tent name, PrimeVue would
  re-render the table under them — the row can move between sections, the input loses focus
  mid-word, and the half-typed value is gone. That is how an operator stops trusting a screen at
  3am.
- 2026-08-23 18:35 — One deliberate difference from `KlanListView`: **the pause lasts only while
  a field is actually open**, not until a save button is pressed. Nothing else here is editable,
  so there is nothing else to protect and no reason to hold the whole night's updates. The
  banner says so, and says when something is waiting.
- 2026-08-23 18:40 — Trap, caught by `vue-tsc`: `editing` has to be declared *above* the
  deferral watch. The watch is `immediate: true`, so it runs synchronously during setup, and a
  later `const` would still be in its temporal dead zone — the whole view would fail to mount.
  Left a comment saying so, because the natural place to put it is with the rest of the editor.
- 2026-08-23 18:45 — `AutoComplete` with `dropdown`, suggestions unfiltered on an empty query and
  ordered by use: the tent four scouts are already in is the likeliest answer for the fifth, so
  it should be the first thing offered. Enter saves, Escape cancels, and explicit check/cancel
  buttons — no save-on-blur, which would fire while reaching for cancel.
- 2026-08-23 18:50 — An unchanged value closes the editor without a request. The server would
  answer 200 anyway (the command dirty-checks), but a request that cannot change anything is not
  worth making. A blank value is refused client-side too, phrased as the choice it is: clearing a
  placering is not offered, because "nowhere" is not a fact about a child in our care.
- 2026-08-23 18:55 — Verified against the running stack: `"  Sovesalen  "` was stored trimmed;
  the same value sent twice produced **one** `shelter.placed` event, not two; a blank was refused
  with "placering skal angives"; `placedAt` was stamped; and "Sovesalen" joined the suggestion
  list. Confirmed `AutoComplete` resolves through the auto-import plugin by fetching the compiled
  SFC from the dev server.
- 2026-08-23 19:00 — ✅ All criteria met. `vue-tsc` clean; 106 vitest passing; Go suite green.
  **Honest gap:** the deferral logic itself has no automated test, because this repo has no
  component-test infrastructure — no `@vue/test-utils`, no jsdom — and the logic lives inside the
  SFC. Adding that infrastructure is a larger decision than this task. A cheaper follow-up would
  be extracting the defer-while-editing into a composable and unit-testing that; it would also
  give `KlanListView` and `KortView` one implementation instead of three. Not done here because a
  composable with one caller is speculative, and refactoring those two views is out of scope.
- 2026-08-23 19:01 — Moving to done.
- 2026-08-23 19:30 — Appended after completion: column tweaks from the crew, applied across the
  three sections (see also PRD 007 §7).
  1. "startede i X" moved from the Navn column to Patrulje — it is a statement about which
     patrol, not about who.
  2. Status column dropped in *I Hønsegården*: every row there is `sheltered`, so it was one
     repeated word.
  3. Telefon column dropped everywhere. The nødtelefon does the ringing; the fields are still in
     the payload and typed, because task 096's search will want to match on them.
  4. Sag column now only in *På vej*. A scout who is here, or handed on, is no longer what the
     case is about.
  5. "Siden" → "Ankommet" in *I Hønsegården*, and the cell drops the "siden" prefix with it —
     "Ankommet: siden 21:40" reads as a mistake. New `formatAt()` beside `formatSince()`.
  Also fixed while in there: the search box never filtered anything. PrimeVue needs
  `globalFilterFields` when columns use custom body templates, and it was missing since 092 — so
  the field looked functional and did nothing. Now matches name, patrol, number and placering.
- 2026-08-23 19:45 — Appended after completion: timestamp cells now name the weekday —
  "siden fre 21.40 (2t 14m)" and "fre 21.40 (2t 14m)". The race runs through a night into the
  next day, so a bare time hid whether "21.40" was four hours ago or yesterday evening; the
  elapsed span answered that, but the clock time is what gets read out over the radio and written
  on paper, so it has to stand on its own. Changed in `formatClock`, so both `formatSince` and
  `formatAt` inherit it.
  Short weekday ("fre"), not "fredag": three characters scan faster in a narrow column and there
  are only two or three days in play. The abbreviation period is normalised away because some ICU
  versions add it and some do not — this Node gives "fre", a browser may give "fre." — and a
  format that differs between dev and production is worse than either.
  One test of mine was wrong and the code was right: I asserted the whole string contained no
  period, but da-DK writes the *time* with a dot (21.40). The assertion is now on the weekday
  alone, with a comment saying why it cannot be on the whole string.
- 2026-08-23 19:55 — Appended after completion: the "siden" prefix is gone from timestamp cells.
  It repeated in every cell of every row what the column header already said, and cost width in a
  table read at arm's length. Cells now read "fre 21.40 (2t 14m)" under both "Siden" and
  "Ankommet".
  Dropping it left `formatSince` and `formatAt` identical, so there is now one `formatTimestamp`
  rather than two functions and a ternary in the template choosing between them. The view is
  simpler than before the "Ankommet" rename introduced the split.
