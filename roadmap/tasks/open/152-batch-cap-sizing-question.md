# 152 — raise the batch-cap sizing with hej-app

**Status:** open
**Priority:** low
**Created:** 2026-09-03
**Picked up by:**
**Started:**
**Completed:**

## Description

PRD 011 §4a. **Not a hq code change** — a question for the hej-app repo. Low likelihood,
unbounded consequence, cheap to check.

`track.MaxPointsPerBatch = 2000` is justified in its own comment against a **12-hour** race:
"at 30 s sampling a full 12-hour race is ~1,440 points, so a participant who was offline for
the entire event still ships their backlog in one request." Nathejk runs closer to **30 hours**,
whose ceiling is ~3,600 points — over the cap.

In practice a phone recording unbroken for 30 hours is unlikely, so the cap will rarely bind.
But if an unchunked backlog ever does exceed it, the server returns `ErrBatchTooLarge`, the
client retries the same oversized batch forever, and every later point queues behind it — which
is exactly the poison pill that `Clean`'s drop-don't-reject design exists to prevent.

hq can neither fix nor directly detect this. The symptom on this side is a person whose track
never arrives after a long offline stretch, which is indistinguishable from a phone that was
simply off.

Ask: does the client chunk a backlog larger than `MaxPointsPerBatch`? If not, either it should,
or the cap should be raised to cover a 30-hour race with headroom.

## Acceptance Criteria

- [ ] Question raised with whoever owns hej-app
- [ ] Answer recorded here: does the client chunk oversized backlogs?
- [ ] If it does not, a task exists in the hej-app repo (reference it here)

## Progress Log

- 2026-09-03 — Task created from PRD 011 §10.
