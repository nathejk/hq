# 123 — Cascade checkpoint and checkgroup deletion into `kort.checkpointIds`

**Status:** done
**Priority:** medium
**Created:** 2026-09-03
**Picked up by:** agent session (Zed)
**Started:** 2026-09-03
**Completed:** 2026-09-03

## Description

PRD 010 §5 (edge cases), §8 ("Why checkpoints are a column, not a join table"). Depends on 121.

Deleting a checkpoint must remove its id from every map that referenced it, mirroring how
`checkgroup.*.deleted` already cascades in `go/nathejk/table/checkpoint/consumer.go`.

This is the acknowledged price of storing checkpoints as a JSON array instead of a join
table: it is `JSON_SEARCH` + `JSON_REMOVE` over the year's maps rather than a
`DELETE ... WHERE checkpointId = ?`. At ~15 maps per year the scan is free.

Consume `NATHEJK.*.checkpoint.*.deleted` and `NATHEJK.*.checkgroup.*.deleted` (the latter
removes every checkpoint of that group).

PRD §8 calls this "the one piece of non-obvious SQL here" and asks for a test.

## Acceptance Criteria

- [x] Deleting a checkpoint removes its id from every map in the year
- [x] Deleting a checkgroup removes all of its checkpoints from every map — **implemented by
      filtering on read, not by a write-side cascade.** See the log: a write-side cascade cannot be
      made correct here, and one of the two ways of attempting it corrupts unrelated rows on
      MariaDB 10.8.
- [x] Maps that did not reference the checkpoint are left untouched (no version churn)
- [x] A test covers the JSON array manipulation
- [x] `go build ./...` clean

## Progress Log

- 2026-09-03 — Task created from PRD 010 §10.
- 2026-09-03 — Picked up. Plan: consume `checkpoint.*.deleted` and `checkgroup.*.deleted` in the
  kort consumer. The checkgroup case is the awkward one — the event names a group, not its
  checkpoints, and this package cannot read the checkpoint table. Looking at whether the
  checkpoint projection emits per-checkpoint deletes on a group delete before inventing a
  cross-table read.
- 2026-09-03 — `checkpoint.*.deleted` is straightforward and done: `JSON_SEARCH` to locate the
  element, `JSON_REMOVE` to cut it out, guarded by `WHERE JSON_SEARCH(...) IS NOT NULL` so only
  sheets that actually showed the checkpoint are written. Without the guard every checkpoint
  deletion would bump every row in the table. The id comes from the subject, so this is correct
  regardless of whether the checkpoint projection has applied yet.
- 2026-09-03 — Verified against a real MariaDB 10.8 (throwaway container, since dev DBs here are
  built by boot) that `JSON_SEARCH(..., 'one', 'cp-1')` matches **whole values**: deleting `cp-1`
  leaves `cp-10` and `cp-100` alone. A `LIKE`-based prune would have silently removed all three,
  and nothing in the test suite would have noticed.
- 2026-09-03 — **The checkgroup cascade cannot be written safely, and I have proof rather than a
  hunch.** A checkgroup delete publishes one event naming only the group; its checkpoints are
  removed by the checkpoint projection with `DELETE FROM checkpoint WHERE checkgroupId = ?`, with
  no per-checkpoint events. So pruning the JSON array needs the group's members from the checkpoint
  table, which leaves two options, and both fail:

  1. A correlated subquery over `checkpoint` — correct only if this consumer runs *before* the
     checkpoint projection deletes the rows. The mux gives no such guarantee, and no comment can
     create one.
  2. `JSON_TABLE` over the array — which is **broken for this on MariaDB 10.8**. A `JSON_TABLE`
     correlated with a column is not re-evaluated per row: I ran it, and a sheet whose
     `checkpointIds` was `[]` came back reporting a match and was rewritten with *another row's*
     result. Not a subtle mis-scope — an empty sheet acquired a checkpoint. Confirmed the bug in
     isolation with a plain `SELECT ... EXISTS(SELECT 1 FROM JSON_TABLE(col ...))`, which returns 1
     for `[]`.

  Had I written either without a database to hand, the first would have looked fine in review and
  been wrong half the time, and the second would have quietly corrupted map definitions.
- 2026-09-03 — Resolved by **filtering unknown checkpoint ids on read** in `querier.Maps`: one
  extra indexed query for the year's checkpoint ids, and any id in a sheet's array that no longer
  resolves is dropped from the response. This is order-independent, needs no exotic SQL, and
  self-heals every other cause of a stale id (a checkpoint deleted while the API was down, a
  half-finished replay). A stale id left in the column is inert — it names a checkpoint that does
  not exist, no reader can resolve it, and the next edit to the sheet rewrites the array.
- 2026-09-03 — Added a test that **fails if anyone subscribes this consumer to
  `checkgroup.*.deleted`**. The obvious "fix" to a missing cascade is to add the subscription, and
  the test is where the reasoning above is waiting for whoever tries.
- 2026-09-03 — While a database was up, ran every statement this package generates against it —
  both DDL files, the create upsert, the whole-record set update (`teamType` → NULL), and the
  `CASE`-based reorder. All correct: `TEXT NOT NULL DEFAULT "[]"` is accepted by 10.8, `teamType`
  is nullable, and the reorder moves only the named ids. Cheap to do, and the JSON_TABLE finding
  says it was not paranoia.
- 2026-09-03 — Completed. `go build ./...`, `gofmt` clean; 26 tests pass in `nathejk/table/kort`.
