# PRDs

Product Requirements Documents: what we intend to build and why, and the decisions
taken along the way. Referenced from `.agents/rules/rules.md` and written by the
`prd` skill (`.agents/skills/prd/`), which holds the full authoring guidance.

## The folder is the status

```
roadmap/prd/
  draft/   ← being written; not yet agreed
  doing/   ← agreed and being implemented
  done/    ← shipped
```

Mirrors the task board's `open/doing/done`. Moving the file is how status changes,
and **the `Status` header must always agree with the folder** — it is one fact
recorded twice. A PRD in `draft/` whose tasks are all done is a bug in the board.

`doing/` may be absent: git does not track empty directories.

## Naming and references

`<NNN>-<slug>.md`, zero-padded. The number is permanent and never changes as the
file moves, so **refer to a PRD by number, not by path** — in task files, commit
messages and code comments. Paths go stale; "PRD 004" does not. Check the highest
number across all three folders before assigning a new one.

## Lifecycle

| Transition | Do |
|---|---|
| new | create in `draft/`; commit `prd(<n>): create — <title>` |
| `draft/` → `doing/` | set `Status: doing`, set `Approved`, create the tasks from the Rollout section in `roadmap/tasks/open/`; commit `prd(<n>): approve — …` |
| `doing/` → `done/` | when every derived task is in `roadmap/tasks/done/`: set `Status: done`, set `Shipped`; commit `prd(<n>): done — …` |
| scope changed materially | move back to `draft/`, clear `Approved`; commit `prd(<n>): reopen — …` |

Always bump `Last updated`. PRDs stay in `done/` — they are the record of intent.

Commit format: `prd(<number>): <action> — <short title>`, actions
`create` · `update` · `approve` · `done` · `reopen`.

## Closing a PRD honestly

Two habits worth keeping, both learned the hard way on PRD 004:

- **Tick the boxes as tasks land.** PRD 004 reached completion with every checkbox
  still unticked, so the document claimed nothing had been done while nearly all of
  it had shipped. The checkboxes are only useful if they track reality.
- **Never tick a box to close a PRD.** Anything unfinished at closure is either
  carried to a task (so it cannot be lost) or recorded in a closing section as
  deliberately out of scope, with the reason. PRD 004 §12 is the worked example: four
  items, one of which became task 041 because it was a real risk rather than a
  nice-to-have.

## Current state

| PRD | Status | Notes |
|---|---|---|
| 001 Nødtelefon / SOS case management | done | shipped 2026-08-11 (tasks 042–054, 056); live confirmed in two tabs; see its §12 |
| 002 Order-based payments | done | shipped 2026-07-31 (tasks 001–008) |
| 003 Patrulje number assignment | draft | not implemented — nothing publishes `numberassigned` |
| 004 Live updates for the SPA | done | shipped 2026-08-09 (tasks 023–040); see its §12 |
| 005 Boot gate, deployment & SPA reload | draft | skeleton; sections marked TBD |
| 006 Member lifecycle, team strength & discontinuation | draft | split from 001; four blocking open questions; 001 has shipped, so it is unblocked |
| 007 Hønsegården: the shelter interface | doing | approved 2026-08-23; tasks 086–098; the shelter half of the custody chain 006 deferred |
| 008 Member notes | doing | approved 2026-08-23; tasks 099–106; prose trail per scout, shared by the shelter and the nødtelefon |

Kept short on purpose, and easy to forget — the folders are authoritative if this
table and they disagree.
