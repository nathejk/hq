# 080 — Show the real member status on the patrol page

**Status:** open
**Priority:** low
**Created:** 2026-08-17
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] Member status column shows the member's actual status
- [ ] Danish labels sourced from the backend, not a local map
- [ ] Severities render sensibly for every `MemberStatus`, including the empty one
- [ ] Dead `getSeverity` demo code removed
- [ ] `npm run build` and `vitest` clean; no new TypeScript errors

## Progress Log

<!-- Append entries here — never edit or delete existing entries -->

- 2026-08-17 — Created from PRD 006. Depends on tasks 065 and 067.
