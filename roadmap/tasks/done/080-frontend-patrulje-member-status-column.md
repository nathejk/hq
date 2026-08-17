# 080 — Show the real member status on the patrol page

**Status:** done
**Priority:** low
**Created:** 2026-08-17
**Picked up by:** agent session
**Started:** 2026-08-17
**Completed:** 2026-08-17

## Description

From **PRD 006** §8 (risks).

`vue/src/views/PatruljeView.vue:114-118` has a Status column that renders a hardcoded tag:

```vue
<Column field="status" header="Status">
    <template #body="{data}">
        <Tag value="ikke startet" :severity="getSeverity(data.status)" />
    </template>
</Column>
```

Every member reads "ikke startet" regardless of their actual status, and `getSeverity` is
matching against values from a PrimeVue demo (`unqualified`, `qualified`, `negotiation`,
`renewal`) that this domain has never produced — so it always returns `undefined` too.

Once tasks 065 and 067 land, the endpoint serves real `types.MemberStatus` values. Render
them, with Danish labels and sensible severities.

## Notes

- This is why the status-value change in task 067 is safe on this screen: nothing was
  reading the value. It is also why the bug survived — the column looked populated.
- Labels come **from the backend** (PRD 006 §6), the same source the SOS card uses. Do not
  add a second hardcoded label map here; if `SosTeamCard` and this view disagree, one of them
  is wrong and nobody will notice.
- Suggested severities: `racing` success, `waiting` warn, `transit` / `sheltered` info,
  `finished` success, `reunited` / `released` secondary, empty/none secondary with a neutral
  label. Confirm the Danish wording with the product owner while implementing.
- `getSeverity` should be deleted or rewritten, not extended — it is dead demo code.
- Small and independent; safe to pick up any time after 067.

## Acceptance Criteria

- [x] Member status column shows the member's actual status
- [x] Danish labels sourced from the backend, not a local map
- [x] Severities render sensibly for every `MemberStatus`, including the empty one
- [x] Dead `getSeverity` demo code removed
- [x] `npm run build` and `vitest` clean; no new TypeScript errors

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006. Depends on tasks 065 and 067.
- 2026-08-17 — Picked up. `MemberStatuses()` added to `TeamConfig` (with `omitempty`: only
  the patrol page populates it, since klaner have no lifecycle here), and the hardcoded tag
  replaced.
- 2026-08-17 — Put the labels on `config` rather than loose in the envelope, because that is
  where this page already looks for the server's vocabulary — korps and t-shirt sizes arrive
  the same way. It also means task 084's correction row, on the same page, needs no second
  source.
- 2026-08-17 — Deleted `getSeverity`, which matched against `unqualified` / `qualified` /
  `negotiation` / `renewal` — values from a PrimeVue demo that this domain has never
  produced, so it always returned `undefined`. Two bugs stacked: a tag whose text was
  hardcoded and a severity that never resolved. Worth noting *why* it survived so long — the
  column looked populated, which is the most durable kind of wrong.
- 2026-08-17 — `'Ikke startet'` is the fallback for an absent status, which is the honest
  reading: the member is on the roster and the race has not claimed them yet. That string is
  what the column used to show for **everybody**.
- 2026-08-17 — ✅ Verified on the product owner's own 2026 test patrol, which happens to have
  mixed statuses: three members render **"Venter på at blive hentet"** and three **"I
  løbet"**. Before this change all six said "ikke startet". 8 status labels served in
  `config`.
- 2026-08-17 — ✅ All criteria met. Moving to done.
