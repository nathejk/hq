# 152 — raise the batch-cap sizing with hej-app

**Status:** done
**Priority:** low
**Created:** 2026-09-03
**Picked up by:** agent session (with knj)
**Started:** 2026-09-09
**Completed:** 2026-09-09

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

- [x] Question raised with whoever owns hej-app
- [x] Answer recorded here: does the client chunk oversized backlogs?
- [x] If it does not, a task exists in the hej-app repo (reference it here) — **N/A, it does**

## Answer: yes, the client chunks. The cap cannot bind.

Read from the hej-app source rather than asked, which is a better answer than a
conversation would have produced — it is the code that will run on the night.

`hej/vue/src/config/track.ts`:

- `TRACK_UPLOAD_CHUNK = 500` — points per request, and its comment names the server's
  2,000-point limit as the reason it is "well under" it. So the client already knows about
  the cap and deliberately stays clear of it.
- `TRACK_UPLOAD_MAX_CHUNKS_PER_RUN = 4` — at most 2,000 points per upload run, bounded to
  stay inside the endpoint's 20-requests-per-minute per-user limit.

`hej/vue/src/stores/track.store.ts` implements it as a loop over chunks:
`pendingPoints(userId, TRACK_UPLOAD_CHUNK)` per iteration, breaking early on a short chunk.
So a backlog is *never* sent as one oversized request — the request size is bounded by
construction, independently of how long the phone was offline.

### The 30-hour concern is therefore not a problem

The worry in this task was a ~3,600-point backlog against a 2,000-point cap. With chunking
that backlog is 8 requests of 500, shipped 4 per run at a 120 s interval — so it clears in
two runs, about four minutes, with no request anywhere near the cap.

The poison pill this task feared (client retries an oversized batch for ever, later points
queue behind it) cannot arise from the hej-app. `ErrBatchTooLarge` remains a correct guard
against a *non-conforming* client, which is what a server-side cap is for.

### What this does not cover

Only the web client was read. A future native client, or a manual/import path, would be
subject to the cap and would need the same chunking — the cap is a contract, and this task
verified one implementation of it, not all future ones.

No hq code change, and none wanted: raising `MaxPointsPerBatch` "for headroom" would loosen a
bound that is doing its job.

## Progress Log

- 2026-09-03 — Task created from PRD 011 §10.
- 2026-09-09 15:57 — Picked up (previous owner gone). Plan: rather than raise the question with
  a person, answer it from the hej-app source, which is available locally at
  `~/Development/nathejk/hej`.
- 2026-09-09 16:05 — Answered from `config/track.ts` and `stores/track.store.ts`: the client
  chunks at 500 points and caps a run at 4 chunks, so an oversized batch is impossible by
  construction. Recorded above with the reasoning and the residual risk (non-web clients).
  Completed — no code change in either repo, and no hej-app task needed.
