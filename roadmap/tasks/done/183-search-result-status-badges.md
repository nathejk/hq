# 183 — Status badges on search result rows

**Status:** done
**Priority:** medium
**Created:** 2026-09-14
**Picked up by:** agent session 2026-09-14
**Started:** 2026-09-14
**Completed:** 2026-09-14

## Description

Depends on task 182. Renders the status that task 179 joined in.

**Status is shown on every row, not only on the unusual ones.** A badge that appears only
when something is wrong makes its absence carry meaning the operator has to know to read;
showing "racing" as plainly as "udmeldt" means a row can always be taken at face value.

Use the existing severity vocabulary in `vue/src/composables/severity.ts` so a departed person
does not look like a current one. The distinction that matters operationally is not
current-vs-departed but *in our care right now* (`sheltered`, `transit`, `waiting`) versus
*somebody else's charge* (`released`, `reunited`) versus *never started* (`deleted` →
"udmeldt").

`deleted` and lifecycle status are separate fields and must stay distinguishable: "udmeldt"
(never started) is not "released" (went home during the night). A scout can only be one of
them, but the UI must not render them identically.

An unknown status — a spejder with no `spejderstatus` row, ordinary for early signups and old
years — renders as unknown. Do not dress it up as `registered`.

Danish wording throughout. Note the open question in PRD 014 §11: a klan member's
`signupStatus` of `PAY` reads badly and is a fact about the *team*, not the person. Pick
something defensible and log the choice rather than blocking on it.

## Acceptance Criteria

- [x] Every result row shows a status, including ordinary ones
- [ ] ~~Severity styling from `composables/severity.ts`; departed ≠ current visually~~ — **half
      wrong; see log 2026-09-14 23:30.** `severity.ts` is the SOS/kørsel *priority* vocabulary
      (Grøn/Gul/Rød) and has no member statuses in it. Member statuses already have a badge
      vocabulary in HQ, so search reuses that instead; `severity.ts` is used for the team-status
      axis, which had none. **Departed ≠ current visually is done and tested.**
- [x] "udmeldt" distinguishable from "released"
- [x] Unknown status rendered as unknown
- [x] Danish labels; wording choice for team-level statuses logged
- [x] Component test asserting a departed row is styled differently from a racing one
- [x] `npm run build` and unit tests pass

## Progress Log

- 2026-09-14 22:08 — Task created from PRD 014 §10.
- 2026-09-14 23:22 — Picked up. Plan: a `statusBadge(result)` in `composables/personSearch.ts`
  returning label + PrimeVue severity + icon, plus a separate always-independent "Udmeldt" badge for
  `removed`, and a `Tag` per axis in `SearchView.vue`. Two vocabularies to reconcile: member
  lifecycle statuses already have badges in HQ (`memberStatusBadge` in `composables/sos.ts`, PRD
  006) and team signup statuses have none. Reading `composables/severity.ts` first — it may not be
  the vocabulary this criterion assumes.
- 2026-09-14 23:30 — **Criterion struck.** "Severity styling from `composables/severity.ts`" cannot
  be followed as written. `severity.ts` holds exactly one vocabulary — Grøn/Gul/Rød, the SOS and
  kørsel *priority* scale — and contains no member or team statuses at all. Applying it to a scout's
  status would mean minting a second set of words for states HQ already badges: `memberStatusBadge`
  in `composables/sos.ts` renders `waiting` as "Udgår"/warn and `sheltered` as "Udgået"/danger on the
  patrol and nødtelefon screens. A search row saying something else about the same scout is the
  drift that `sos.ts`'s own comment warns about, and worse here than elsewhere, because the operator
  is comparing the two on a phone call. So: member statuses reuse the existing vocabulary verbatim,
  and `severity.ts` is used for the *team* axis, which genuinely had no vocabulary. Reported to the
  requester.
- 2026-09-14 23:34 — To reuse it without the wrong dependency, extracted the member lifecycle
  vocabulary from `sos.ts` into a neutral `composables/memberStatus.ts`, with `sos.ts` re-exporting
  it. This is not a new pattern: it is exactly what task 112 did with `severity.ts`, for the stated
  reason that "a delivery of dinner has no business depending on the emergency-phone module". A
  results page importing the nødtelefon module to learn what `sheltered` means would be the same
  mistake. Pure move plus re-export; the whole suite (306) still passes untouched.
- 2026-09-14 23:40 — Team-status wording decided, answering PRD 014 §11's open question. The
  problem with `PAY` is not that it is English but that it is a fact about the *team* rendered on a
  *person's* row, so it reads as a debt of theirs. Every team status is therefore prefixed with its
  subject: "Holdet: mangler betaling", "Holdet: betalt", "Holdet: startet", "Holdet: afventer",
  "Holdet: tilmeldt", "Holdet: delvist betalt", "Holdet: ude". Cheaper than a second column and
  impossible to misread. `OUT` is "ude" and deliberately **not** "udgået", because `sheltered`
  already owns "Udgået" in the member vocabulary — two states sharing a word is precisely what these
  badges exist to prevent. Colours via `severityTagSeverity`: paid/started green, the four in-progress
  states yellow, OUT red.
- 2026-09-14 23:44 — Added a `title` (long form, shown on hover) to every badge, which is not
  decoration. The inherited member vocabulary spreads *in our care* across two colours
  (`waiting`/`transit` warn, `sheltered` danger) and *somebody else's charge* across two more
  (`released` secondary, `reunited` info), because it was designed for race-night screens rather than
  for a phone call — so the three-way distinction this task names is not readable from colour alone.
  Forking the vocabulary would have fixed the colours and broken the consistency; carrying the
  unambiguous sentence with the badge fixes it without forking. Flagged as PRD-worthy.
- 2026-09-14 23:48 — ✅ `removed` is rendered as its own `Tag` beside the status, never instead of
  it: "Udmeldt" (own word, own glyph `pi-user-minus`, danger) against `released`'s "Afhentet"
  (secondary, `pi-sign-out`). Both badges appear when both are true, which is the case the PRD says
  must not be collapsed. Tested on the mapping and in the DOM.
- 2026-09-14 23:50 — ✅ Unknown renders "Ukendt status" with a `contrast` tag and the hover text
  "Ingen status registreret — helt normalt før løbet", so it reads as ordinary without asserting a
  status the read model does not hold. Explicitly not `registered`. Covered for `statusKind: ''`,
  for `member` with an empty status, and for both fields absent (which is what the API actually
  sends — both are `omitempty`, verified against a live response).
- 2026-09-14 23:52 — An unrecognised team slug keeps its own name ("Holdet: WHATEVER") rather than
  becoming "ukendt": that case is a deploy skew, and hiding it would look like data loss.
- 2026-09-14 23:56 — ✅ Component test added: two rows, one `racing` and one `released`, asserting
  the rendered tags differ in class/`data-p` signature *and* in wording. On markup rather than on the
  mapping deliberately — the failure that matters is a `severity` that never reaches the Tag, which
  renders every row identically while every unit test still passes.
- 2026-09-14 23:58 — ✅ `npm run test:unit` 318 passed (21 files), `npm run build` clean, `vue-tsc`
  clean on all four touched files.
- 2026-09-15 00:00 — Completed. Every row badged on two independent axes, sharing HQ's existing
  member vocabulary, with one criterion struck rather than fudged.
