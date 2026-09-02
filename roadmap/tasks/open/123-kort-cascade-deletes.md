# 123 — Cascade checkpoint and checkgroup deletion into `kort.checkpointIds`

**Status:** open
**Priority:** medium
**Created:** 2026-09-03
**Picked up by:**
**Started:**
**Completed:**

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

- [ ] Deleting a checkpoint removes its id from every map in the year
- [ ] Deleting a checkgroup removes all of its checkpoints from every map
- [ ] Maps that did not reference the checkpoint are left untouched (no version churn)
- [ ] A test covers the JSON array manipulation
- [ ] `go build ./...` clean

## Progress Log

- 2026-09-03 — Task created from PRD 010 §10.
