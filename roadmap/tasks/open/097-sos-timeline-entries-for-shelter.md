# 097 — Sos timeline entries for shelter operations

**Status:** open
**Priority:** low
**Created:** 2026-08-23
**Picked up by:**
**Started:**
**Completed:**

## Description

A nødtelefon operator who sent a scout in a car wants to see that they arrived, without being
told (PRD 007 §5, last user story).

Where an **open** case is associated with the scout's patrol, a shelter operation appends a
summarising activity entry to it, reusing PRD 006's `RecordMemberStatusChanged` —
`ActivityMemberStatusChanged` already exists and the case card already renders it.

Where there is no case, **nothing is created**. This is the important half: the shelter's
acceptances are legitimate acts, not corrections, so they must not mint cases the way
`mintCorrectionCase` does for a manual status override. A case per arrival would bury the
real calls.

One entry per operation, not per member — a multi-select acceptance of three scouts off one
car reads as one line, same as team collection.

Failure to record the entry must not fail the operation: custody of a child is the fact that
matters, the timeline entry is a courtesy. Log and carry on, as `memberStatusOperation`
already does for its summary.

## Acceptance Criteria

- [ ] Shelter acceptance and handover append an activity entry to an associated open case
- [ ] No case is created when none exists
- [ ] One entry per operation, not per member
- [ ] A failure to append is logged and does not fail the status change
- [ ] Verified on the case card: the entry renders with the existing component

## Progress Log

- 2026-08-23 09:30 — Task created from PRD 007 §10.
