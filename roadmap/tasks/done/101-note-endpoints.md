# 101 — Note endpoints

**Status:** done
**Priority:** high
**Created:** 2026-08-23
**Picked up by:** agent session (Zed)
**Started:** 2026-08-23
**Completed:** 2026-08-23

## Description

Three endpoints over task 100's commands (PRD 008 §8), year-scoped via `X-YearSlug`:

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/member/:memberId/notes` | The thread, oldest first |
| POST | `/api/member/:memberId/notes` | Add a note |
| PATCH | `/api/member/:memberId/notes/:noteId` | Correct a note |

Registered on the member, beside the lifecycle routes. Deliberately **not** folded into
`GET /api/member/:memberId`: that endpoint feeds a modal opened one scout at a time, while the
thread is a separately cacheable, separately invalidated resource the SPA holds per member.

Watch httprouter's constraint — a static segment cannot sit where a sibling holds a wildcard.
`/api/member/:memberId/notes` is fine (the wildcard is the parent), but check the router still
builds; a conflict panics at boot, and `stream_test.go`'s single `app.routes()` call is what
catches it.

Refusals map to Danish messages phrased as what to do, following `shelterCommandError`.

**OpenAPI annotations on all three** — `@Summary`, `@Description`, `@Tags`, `@Produce`,
`@Success`, `@Failure`, `@Router` — per `cmd/api/order.go`.

## Acceptance Criteria

- [x] Three routes registered and handled; router boots
- [x] No `sosId` required
- [x] Empty/over-long/wrong-member refusals answered with Danish field messages, not raw domain
      errors
- [x] Unchanged edit answers 200
- [x] OpenAPI annotations complete on all three
- [x] Handler tests for the happy paths and each refusal

## Progress Log

- 2026-08-23 20:00 — Task created from PRD 008 §10.
- 2026-08-23 21:30 — `cmd/api/note.go` with the three handlers, OpenAPI annotations, and
  `noteCommandError` mapping refusals to Danish on the field they concern. Empty trails serve `[]`,
  not `null` — the lesson task 092 shipped to a browser before it was caught, so the test asserts
  the raw JSON rather than decoding (a decode turns both into a nil slice and passes either way).
- 2026-08-23 21:45 — ✅ All criteria met. 7 handler tests; `cmd/api` green; full Go suite green.
- 2026-08-23 22:00 — **Verified end to end against the running stack.** A note on a real sheltered
  scout:
  - written and returned its minted id; stored **trimmed** (the request had leading and trailing
    spaces);
  - `actorUserId` empty, as expected before login — the accepted unsigned trail;
  - an empty note refused with `{"note": "noten kan ikke være tom"}`;
  - corrected (06.00 → 05.30): text changed, **`createdAt` unmoved**, `updatedAt` advanced, so
    `Edited()` reads true;
  - the same correction resubmitted answered 200 and added no second version;
  - the same correction attempted **as a different member** refused with `{"noteId": "noten hører
    til en anden spejder"}` — the ownership hole closed.
- 2026-08-23 22:05 — **The replay hazard, proven rather than argued.** Restarted the API so the
  whole history re-ran through the projection. The corrected note came back still corrected
  (05.30), with its original `createdAt` and its correction's `updatedAt`, and there was **one**
  note, not two. That is the failure task 099's upsert was designed against, now demonstrated
  against the real event store rather than a fake writer.
- 2026-08-23 22:06 — Also confirmed by the restart: the router builds with
  `/api/member/:memberId/notes` and `/notes/:noteId` beside the existing member routes — an
  httprouter conflict panics at construction, so a clean boot is the proof.
- 2026-08-23 22:10 — Unrelated finding while checking the logs, recorded so it is not lost:
  `main.go:341` is `logger.PrintFatal(app.Serve(…), nil)`, and `Serve` returns **nil** on a
  graceful shutdown — so `PrintFatal` dereferences a nil error and every clean stop ends in a
  SIGSEGV panic. Pre-existing, unrelated to PRD 008, and fires on every hot-reload and every
  `docker compose restart` (187 of them in this session's log). Harmless to the running service,
  but it makes every ordinary shutdown look like a crash, which is exactly the noise that hides a
  real one. Not fixed here — it is in the composition root and touches shutdown behaviour, so it
  wants its own task and the owner's decision.
- 2026-08-23 22:11 — Moving to done.
