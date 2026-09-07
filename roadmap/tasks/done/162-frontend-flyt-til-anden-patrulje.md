# 162 — Flyt til anden patrulje: button, dialog and transfer orders

**Status:** done
**Priority:** high
**Created:** 2026-09-07
**Picked up by:** agent session
**Started:** 2026-09-07
**Completed:** 2026-09-07

## Description

From **PRD 012** §10, phases H4 and H5. Depends on task 161.

Put the pre-race transfer in front of the operator, on the patrol page
(`/patrulje/:teamId`, `vue/src/views/PatruljeView.vue`):

- a **Flyt til anden patrulje** button in the member's expanded row, beside the existing
  "Ret status manuelt";
- a dialog that lists every accepted team — the ineligible ones **struck through with the
  reason**, not hidden — and states what will move before it is confirmed;
- transfer orders labelled in the Betalinger table, so a negative amount reads as money
  moving to another team rather than as a mistake.

## Notes

- The picker follows this repo's idiom (`SosTeamCard.vue`): an `InputText` filter over a
  short list with a **Vælg** button per row. Not `AutoComplete`, not `Select` — those are
  for small fixed option sets. Divergence from that prior art is deliberate and required:
  `SosTeamCard`'s `destinations()` filters ineligible teams *out*, and here they must be
  visible.
- **Eligibility is not recomputed in the browser.** The server annotates each candidate with
  `eligible` + `reason`, so the list and the refusal cannot word the same rule differently
  or drift apart. The dialog only renders it.
- The button is **hidden**, not disabled, once the patrol has started: at that point the
  action does not exist here at all — the move becomes the nødtelefon's case-based one — and
  a greyed-out button would invite the wrong question.
- "Nothing has been paid for this member" is stated in the dialog rather than shown as an
  empty box. It is an ordinary outcome, and an operator who moves somebody and sees no money
  move should be told why.
- Transfer orders are recognised by their line ids (`transfer:{transferId}:{n}`), which the
  shared-go command derives deterministically — so labelling them needed no new column and
  no new endpoint. Tagged *Intern overførsel* next to the amount.
- No new resource or `dependsOn` token: the reassignment publishes on `spejder` and the
  transfer on `order`/`payment`, all three already declared by this view. The `refresh()`
  after a successful move is belt-and-braces, matching `saveCorrection`.
- The candidate list is fetched when the dialog opens rather than held in a live resource.
  It is a short-lived modal, and a list revalidating underneath an operator mid-choice is
  the dirty-state problem the project rules warn about.

## Acceptance Criteria

- [x] Button in the expanded row, hidden when the patrol has started
- [x] Dialog lists accepted teams with member counts; ineligible ones struck through with
      their reason and no Vælg button
- [x] What moves (lines, sizes, total) shown before confirming
- [x] "Der er ikke betalt for denne deltager…" when there is nothing to transfer
- [x] Success toast naming the destination; list updates without a page reload
- [x] Transfer orders tagged in Betalinger; negative amounts render as credits
- [x] `vitest` green (243 tests), `vite build` clean

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-09-07 — Implemented in `PatruljeView.vue`. Ran the suite and a production build in
  the `hq-ui` image against the `hq_ui-node_modules` volume rather than the host's
  `node_modules`, which is a partial macOS install missing `vitest` — installing into the
  mounted directory would have replaced the host's binaries with Linux ones and broken
  local `npm run dev`.
- 2026-09-07 — Not covered: the dialog has no component test. The view is plain JS with no
  existing test, and the logic worth testing (candidate filtering) is a five-line computed
  over a server-authored list. Noted in task 163.
