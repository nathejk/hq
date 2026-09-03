# Erasing one person's telemetry

**For:** whoever handles a deletion request (GDPR art. 17, or a parent asking)
**Owner:** hq (`go/nathejk/table/track`)
**Introduced by:** PRD 011, task 151
**Last updated:** 2026-09-03

Position history is personal data about named people, many of them minors. This is how
one person's is removed.

---

## Why this needs a procedure at all

The `TELEMETRY` stream is **retained indefinitely**, and `hej-api` publishes on a
per-person subject precisely so that one individual can be purged from it:

```
TELEMETRY.{year}.track.{personId}.reported
```

That design gives a clean erasure story for the stream. It does **not**, on its own,
give one for hq — and that is the trap this document exists for.

hq projects the stream into MySQL (`track_latest`, `track_point`). Purging the stream
removes the source, but hq's rows persist afterwards: a projection is not notified of a
purge, and there is no event saying "forget this". **Without the second half of this
procedure, hq quietly becomes the place erased location data survives.**

---

## The order matters

> **Purge the stream first. Delete hq's rows second.**

Do it the other way round and the deletion accomplishes nothing: hq replays every
projection from the stream on **every api restart**, so the next deploy — or a crash, or
a hot reload in dev — puts every deleted point straight back.

Doing it in the right order is what makes the erasure durable: after the purge there is
nothing left to replay, so the rows stay gone through every future restart.

---

## 1. Purge the person's subject from the stream

Needs access to the NATS deployment.

```sh
nats stream purge TELEMETRY --subject 'TELEMETRY.*.track.<personId>.reported'
```

Note the `*` for the year: a person who took part in more than one event has a subject
per year, and all of them should go.

Verify nothing remains:

```sh
nats stream view TELEMETRY --subject 'TELEMETRY.*.track.<personId>.reported'
```

## 2. Delete hq's rows

Both tables. `track_latest` is what the position glyph reads; `track_point` is the
history behind the map.

```sql
DELETE FROM track_point  WHERE personId = '<personId>';
DELETE FROM track_latest WHERE personId = '<personId>';
```

In dev:

```sh
docker compose exec mysql mysql -uroot -pib hq -e "
  DELETE FROM track_point  WHERE personId = '<personId>';
  DELETE FROM track_latest WHERE personId = '<personId>';"
```

In stage and production, run the same two statements against that environment's
database. Nothing else in hq holds position data — no other table, no cache, no export —
so these two statements plus the purge are complete.

## 3. Confirm

```sql
SELECT COUNT(*) FROM track_point  WHERE personId = '<personId>';  -- expect 0
SELECT COUNT(*) FROM track_latest WHERE personId = '<personId>';  -- expect 0
```

And through the API, which should report the person as never having reported:

```sh
curl -s .../api/telemetry/presence | grep '<personId>'   # expect no match
```

The glyph then disappears from every list in HQ, because absence from that response is
exactly what "has never reported" means.

---

## What is *not* erased, and why

- **QR scans.** `scan` rows are the record of a team touching a post — an operational fact
  about the event, witnessed by crew, not a location trail derived from someone's phone.
  They are out of scope here. If a request covers them too, that is a separate decision
  with a different justification.
- **Membership and status history.** `spejderstatus` / `spejderstatuslog` record that a
  person took part and what happened to them, which is what the event is administered
  from. Also out of scope.
- **Other people's tracks.** The per-person subject is what makes step 1 surgical. Never
  purge `TELEMETRY.>`.

---

## Why there is no button for this

There is deliberately no endpoint and no admin UI.

hq does not authenticate anyone: `app.authenticate` attributes every request to an
anonymous user, and identity lives in an external service (`AUTH_BASEURL`). A destructive,
irreversible, unauthenticated endpoint that deletes personal data on request would be a
worse risk than the one this document mitigates.

If erasure should become self-service, it needs a verified actor first — which is a real
piece of work and not part of PRD 011. Until then a human with database access runs two
statements, which is slower and safe.
