# `GET /api/kort` — the printed map sheets

**For:** the hej-app team (separate repo)
**Owner:** hq (`go/nathejk/table/kort`, `go/cmd/api/kort.go`)
**Introduced by:** PRD 010
**Last updated:** 2026-09-03

HQ now records the sheets we print and hand out during the event: what each is
called, which set it belongs to, what ground it shows, and — the part you want —
**which checkpoints are drawn on it**.

This document covers the things you cannot infer from the JSON and would
otherwise get wrong. The endpoint's own OpenAPI annotations describe the shape;
this describes the meaning.

---

## The request

```
GET /api/kort
X-YearSlug: 2026
```

Year-scoped by the header, like every other HQ endpoint; omit it to get the
current year. One request returns every set and every sheet for the year — on the
order of fifteen rows — so there is no pagination, no filter, and deliberately no
per-sheet read. Poll this and work from the result.

## The response

Captured from the running dev API, not hand-written:

```json
{
  "kortsaet": [
    {
      "id": "kortsaet-8dca3371-7120-4d58-a28e-1ded516eaaf6",
      "year": "2026",
      "version": 0,
      "name": "Crew",
      "sortOrder": 0,
      "teamType": null,
      "kort": []
    },
    {
      "id": "kortsaet-13609b94-a594-41bd-8fc7-6a6de6fdb69f",
      "year": "2026",
      "version": 0,
      "name": "Patruljer",
      "sortOrder": 1,
      "teamType": "patrulje",
      "kort": [
        {
          "id": "kort-c556ad94-adb1-490c-9e3c-2bb623dd06f7",
          "kortsaetId": "kortsaet-13609b94-a594-41bd-8fc7-6a6de6fdb69f",
          "year": "2026",
          "version": 0,
          "name": "Kort 1",
          "format": "a4",
          "note": "",
          "sortOrder": 1,
          "checkpointIds": [
            "97476862-4930-4ea8-8789-d8dda1ce8159",
            "fdaf3511-5e78-4b8f-80c6-efd96653da1a"
          ],
          "extents": [
            {
              "northWest": { "latitude": 56, "longitude": 9 },
              "southEast": { "latitude": 55, "longitude": 9.4 }
            }
          ]
        }
      ]
    }
  ],
  "orphanKort": []
}
```

---

## 1. There are two reveal units, not one

This is the most important thing on the page, and the thing most likely to be
implemented as one rule instead of two.

- **A sheet's QR is linked or scanned** → that sheet's `checkpointIds` become
  visible.
- **A checkpoint is scanned** → its whole **checkgroup** becomes visible.

They do not nest. A skitse shows a subset of one checkgroup; a double-sided A3
spans two. So neither rule can be derived from the other, and HQ deliberately
implements neither — it records the facts and leaves the policy to you.

**A skitse has no QR code at all.** It is a hand-drawn slip handed over at a post,
and it is never scanned, so its checkpoints are revealed off the *previous*
checkpoint's scan. Its `checkpointIds` list is the only trace of it in the system,
which is why sheets with no extent and no QR still matter. `format` is `"skitse"`.

## 2. Find the patrol sheets by `teamType`, never by name

Set names are Danish free text an organizer may rename mid-season — "Patruljer",
"Patruljekort", "Patruljer nord". Matching on them will break.

Match on the set's `teamType` instead: `"patrulje"`, `"klan"`, `"crew"`,
`"gøgler"`, or `null`.

## 3. `teamType` is a filter, not a key — and three consequences

Read it as *"this set is specifically for this team type"*, **not** *"only this
team type uses it"*.

1. **`null` is the ordinary case, not missing data.** The crew set covers gøglere,
   banditter and crew, who are not one team type. It is nullable and always
   present in the JSON, never omitted.

2. **It is not unique.** More than one set may carry the same `teamType` — a year
   that splits its patrol maps into two sets is legitimate. So collect *all*
   matching sets; do not look for "the" patrol set.

3. **Filtering by `"klan"` will usually return nothing, and that is not an
   error.** Klaner normally draw from the unmarked crew set. A caller that
   concludes "this klan has no maps" is wrong — **fall back to the unmarked
   set(s)**.

## 4. `checkpointIds` is always live

Ids that no longer resolve are filtered out server-side before you see them, so
the list never contains a deleted checkpoint. That is also how deleting a whole
*checkgroup* reaches the maps — see `pruneCheckpoint` in
`go/nathejk/table/kort/consumer.go` if you want the reason it works that way.

Order carries no meaning: a sheet's checkpoints are a set. It is stable between
saves, but do not read anything into it.

A checkpoint may appear on **any number of sheets**, including several in one set:
adjacent sheets overlap by design, and every patrol sheet's ground is also on the
crew map. Seeing an id twice is normal.

## 5. `extents` is a list of zero, one or two rectangles

- **Zero** — normal. A skitse has no area worth recording, and a sheet may be
  described before anyone draws it.
- **One** — an ordinary sheet.
- **Two** — a double-sided sheet, which is **one** `kort` with two areas. One
  sheet, one QR, one handover, one reveal.

The two are simply two areas. Nothing says which is the front, and the
checkpoints are **not** split per side, because both sides are handed over at
once. Do not try to infer sides.

Corners are always a true north-west/south-east pair — normalised on save — so no
min/max dance is needed before handing them to a map library.

## 6. Empty means `[]`, never `null`

`kortsaet`, `kort`, `checkpointIds`, `extents` and `orphanKort` are always arrays.
A set with no sheets sends `"kort": []`; it does not omit the key.

`orphanKort` holds sheets whose set no longer exists. Normally empty; it exists so
a mis-assigned sheet cannot become invisible in HQ. **Ignore it** unless you are
building an admin view — a sheet in it has no set and therefore no `teamType`, so
it cannot be attributed to anybody.

## 7. Maps are per year and never carried forward

The event is in a different area each year, so last year's sheets and coordinates
have no meaning this year. There is no cloning, and a new year starts empty.

---

## What is *not* here yet

- **Which QR code sits on which sheet.** Every printed sheet carries a unique QR,
  and scanning an unknown one prompts HQ to link it to a patrulje. Linking it to a
  *sheet* is the natural next step and is not built. Until it is, this endpoint
  gives you the **candidate** sheets for a team type; matching a specific scan to a
  specific sheet has to come from elsewhere.
- **Where a sheet is handed out.** Not recorded, and not planned: it varies, it is
  not reliably known, and the sheet a team holds is established by the QR link
  rather than by inferring it from where they had been.

## Questions

Raise them against PRD 010 (`roadmap/prd/doing/010-kort-definitions.md`), which
carries the reasoning behind all of the above. If something here needs to change,
it needs to change there first — this endpoint is a contract with your repo, and
its shape should not move without warning.
