# PRD 004 — Live updates for the SPA

**Status:** draft
**Author:** agent session
**Created:** 2026-08-07
**Last updated:** 2026-08-09
**Target users:** organizer (every operator using the HQ admin panel)

---

## 1. Summary

Give the HQ SPA a single, reusable way to stay current without the operator
refreshing, and to render instantly when navigating between pages. The server
publishes small `entity.changed` signals over one SSE endpoint; the frontend has
one cache primitive that pages compose. Both halves are entity-agnostic, so
adopting them on a new page is a few lines rather than a new pipeline.

## 2. Problem & Motivation

- **What problem does this solve?** The SPA is REST-only: every page fetches on
  mount and then goes stale until the operator reloads. Two operators looking at
  the same screen see different things, and navigating away and back costs a
  round trip — so the app feels slow exactly when it is used hardest, during the
  event.
- **Why now?** PRD 001 (Nødtelefon / SOS) needs it first and most: a dispatch
  desk showing stale cases, a stale "in our care" count or a stale patrol
  strength is not merely inconvenient, it is misleading. Rather than building a
  bespoke channel for that one screen, build the capability once and let SOS be
  its first adopter.
- **Evidence.**
  - The legacy platform had this and lost it: `_go/cmd/api/dims.go`,
    `_go/pkg/sockethub`, `_vue/src/store/dims.js`. It worked, but was difficult
    to work with and around — §2.1 records exactly why, because those are the
    traps to avoid.
  - The current SPA has no push transport at all: no `EventSource`, no
    websocket, no `http.Flusher` anywhere in `go/` or `vue/src`.
  - Pages that would benefit immediately: betalinger, patruljer, klaner, poster,
    ordrer, kort — all of which show data that other operators and other
    services change while you are looking at it.

### 2.1 Why the legacy `dims` channel was painful

Kept deliberately, because most of these were architectural rather than
transport problems and are easy to reintroduce:

1. **A second, parallel read model.** `dims.go` ran an in-memory aggregate stack
   (`member.NewMemberModel`, `sos.NewSosModel`, …) *alongside* the SQL tables and
   pushed snapshots from it via `websync.NewWebsyncModel(map[aggregate]view)`.
   REST served MySQL; the socket served aggregates. Two sources of truth, and the
   SOS activity timeline existed *only* in the socket one.
2. **All-or-nothing subscription.** The client sent `{View:'personnel'}`,
   `{View:'klan'}`, `{View:'patrulje'}`, `{View:'sos'}` … on connect, so every
   screen received every other screen's traffic.
3. **A hand-rolled connect race.** The subscribe block was duplicated verbatim —
   once in the `open` listener, once behind `if (ws.readyState == 1)`.
4. **Hand-rolled reconnect** with manual backoff re-entering
   `commit('dims/initialize')`, and no recovery of messages missed while down.
5. **Auth and transport failures entangled** — the socket's `error` handler called
   `fetchUser()`, which redirected to the login host as a side effect.
6. **No honest loading state.** `lastModify[view]` timestamps doubled as "have we
   loaded?", and every getter repeated `if (!state.lastModify['sos']) return {}`,
   so loading, empty and missing were indistinguishable.
7. **Arrays used as maps, recomputed per message:** `state.list[view] = []` keyed
   by id, then `state.ids[view] = Object.keys(...)` on every message.
8. **A scattered write path:** a dozen near-identical actions each hand-writing
   `axios.post(..., { withCredentials: true })`, some HTTP issued from Vuex
   *mutations*.

## 3. Goals

- One **transport**: a single SSE endpoint carrying entity-agnostic
  `entity.changed` signals for the whole SPA.
- One **client primitive**: a cache with explicit loading semantics that any page
  composes, so live-updating a new page needs no new plumbing.
- **Zero per-entity backend code.** Making an entity live is a wiring change at
  the mux, not a new handler, projection or event type.
- **Instant navigation.** Returning to a page renders from cache with no request
  and no spinner.
- **Honest state.** The UI can always tell the operator whether it is live,
  reconnecting, or disconnected. It never has to reason about *stale* data: the
  API does not serve until it is fully caught up (§8 "Boot and readiness").
- Adoptable incrementally: pages migrate one at a time, and a page that has not
  migrated keeps working exactly as it does now.

## 4. Non-Goals

- Push**ing data**. The stream carries change notifications only; data is always
  fetched over REST. See §8 for why.
- Offline support or a persistent client-side database.
- Collaborative editing, presence, cursors, or locking.
- Replacing REST. Every read stays available as a plain request; the stream only
  tells clients *when* to re-read.
- Migrating every page. This PRD delivers the capability plus SOS as first
  adopter (PRD 001); other pages are follow-up tasks.
- Participant-facing or public real-time features. HQ operators only.

## 5. User Stories & Scenarios

- As an **operator**, I want the screen I am looking at to reflect what my
  colleagues just did, without pressing reload.
- As an **operator**, I want moving between pages to feel instant, so I can check
  something and come back mid-phone-call without losing my place.
- As an **operator**, I want to know when the app is *not* live, so I do not trust
  a frozen screen.
- As a **developer**, I want to make a new page live by composing one primitive,
  without touching the stream, the API, or any projection.

### Primary happy path

1. Operator opens the SPA. It connects to `/api/stream` once, for the whole app.
2. They open Betalinger. The page fetches its list and caches it.
3. Another operator marks a payment received. The `payment` projection applies
   it; a `entity.changed` signal for `payment/123` follows.
4. The open page revalidates just that payment and re-renders. No reload, no
   full-list refetch.
5. The operator switches to Patruljer, then back to Betalinger — both render
   instantly from cache, already current because the stream kept them so.

### Edge cases & error scenarios

- **Connection drops.** `EventSource` reconnects on its own; on reconnect the
  client revalidates what it has cached, because signals may have been missed.
- **Server restarts mid-session.** The API refuses traffic until it has caught
  up, so the client sees *unavailable* rather than *stale* — a much easier failure
  to handle honestly. It reconnects until the API is ready, then revalidates.
- **Burst of changes** (mass import, whole-team collection): signals are coalesced
  per `(entity, id)` so the client is not flooded. Note **replay is no longer a burst
  source**: the boot gate (PRD 005) means no client is connected while a build
  replays, so the entire history passes the notifier with nobody listening.
  Coalescing therefore exists for *live* bursts only, which lowers the stakes on
  tuning its window.
- **A signal for an entity id the client has never seen** must still reach *lists and
  aggregates* that depend on that entity type — otherwise a newly created case never
  appears. Only resources that depend on neither the id nor the type ignore it.
  See §8 "Dependencies: ids are not enough".
- **The operator's JWT expires** while the stream is open: the reconnect fails
  with 401 and the app prompts re-login rather than retrying invisibly.
- **A page not yet migrated** receives no signals and behaves exactly as today.

## 6. Requirements

### Functional

- [x] `GET /api/stream` — one SSE endpoint per client, authenticated by the
      existing JWT cookie, emitting `entity.changed` events.
- [x] Signal payload is entity-agnostic and data-free:
      `{"entity":"payment","id":"123","year":"2026","event":"received"}`.
- [x] Optional filter: `?entities=sos,patrulje` so a client receives only what it
      renders.
- [x] **Year is specified when subscribing**, as a query parameter rather than a
      header — `EventSource` cannot set headers, so the `X-YearSlug` mechanism the
      rest of the SPA uses (`vue/src/plugins/axios`, read by `YearSlug()` at
      `go/cmd/api/routes.go:102`) is unavailable here. A missing year defaults to the
      current calendar year, matching existing server behaviour; the SPA always sends
      it explicitly regardless.
- [x] **Switching year flushes everything.** Changing the selected year tears down the
      stream, **clears the entire cache**, reconnects with the new year and refetches.
      No entry may survive a year change — stale cross-year data is worse than a
      brief spinner.
- [x] **One source of truth for the year.** The stream's query parameter and the REST
      `X-YearSlug` header both derive from `globalstate.yearSlug`. If they can
      disagree, a client will receive signals for one year while fetching another and
      appear frozen.
- [x] Heartbeat comment every ~20s to survive Traefik's idle timeout.
- [x] Signals are derived generically from the message subject
      (`NATHEJK.{year}.{entity}.{id}.{event}`) — no per-entity code.
- [x] Signals are emitted **only after** the projection has applied the event.
- [x] Signals are coalesced per `(entity, id)` over a short window (~50–100ms).
- [x] Subject shapes that carry no id degrade to a coarser signal rather than
      mis-parsing: year-level `NATHEJK.{year}.created|updated`
      (`table/year/consumer.go:21`) and collection-level
      `NATHEJK.{year}.checkgroups.sorted` (`table/checkgroup/consumer.go:25`).
      Event names may contain dots (`status.changed`, `armNumber.assigned`), so
      the event is everything after the id.
- [x] No staleness or replay UI is required, because the API only ever serves a
      caught-up read model (§8 "Boot and readiness"). The client distinguishes
      *connected* from *unavailable*, not *fresh* from *stale*.
- [x] Client primitive — `useLiveResource(key, fetcher)` or equivalent — owning a
      cache entry, subscribing to signals for its key, and exposing explicit
      `data` / `pending` / `error` state. No page implements its own cache.
- [x] A resource declares **what it depends on — entity types and/or specific
      instances** (`dependsOn: ['sos']`, `dependsOn: ['sos:123', 'sos_activity']`), not
      just its own key, so lists and derived aggregates revalidate when *any* member of
      a type changes (§8 "Dependencies: ids are not enough"). Without this, new rows
      never appear and computed figures never move.
- [x] `dependsOn` is a **mandatory** argument, even when empty. A missing dependency
      fails silently — a figure that never updates, with no error anywhere — so the
      decision must be visible at every call site and in review.
- [x] Cache survives navigation (module-level state) and renders immediately on
      return, revalidating in the background (stale-while-revalidate).
- [x] Spinner shown only when there is no cached value at all.
- [x] Connection state (`live` / `reconnecting` / `polling` / `offline`) is
      exposed to the UI and displayed somewhere persistent.
- [x] Optimistic write helper so a page can apply the operator's own change
      immediately and reconcile when the signal arrives.
- [x] Transport sits behind a small interface with a polling implementation, so
      SSE can be swapped in or out without touching a page. The interface must
      express both semantics: SSE reports **what** changed, polling can only say
      **something might have** — so the polling implementation revalidates every
      mounted resource on an interval, and the primitive must behave correctly under
      both.
- [x] A `deleted` signal, or a revalidation that 404s, **evicts** the cache entry
      rather than surfacing an error.
- [x] Per-client buffering is bounded, and on overflow the backlog collapses into a
      single `resync` signal rather than dropping invalidations. A client must never
      end up silently stale.

### Non-Functional

- **Latency:** a change is visible to other operators in ≲1s under normal
  conditions.
- **Navigation:** returning to a cached page issues **zero** blocking requests and
  renders in the same frame.
- **Cold load:** first paint of a page's data in well under a second, with no
  request waterfall.
- **Cost:** signals are small and carry no entity data; a client subscribing to a
  filtered set must not receive unrelated high-frequency traffic (e.g. scans).
- **Security:** signals leak no entity data, so learning that `payment/123`
  changed reveals nothing; the refetch goes through the normal authorised
  endpoint. No new authorisation model.
- **Resilience:** no message loss may leave the UI permanently stale —
  reconnect-and-revalidate must converge without operator action.
- **No regression:** pages that have not adopted the primitive keep working.

## 7. UX / UI Notes

- **Connection indicator.** Small, persistent, unobtrusive when healthy; clearly
  degraded when not. A dispatch desk that looks live but is frozen is worse than
  one that admits it. States: live, reconnecting, polling fallback, offline.
- **No "data may be incomplete" banner.** There is no such state to show: the API
  serves only live data. If it is still catching up it is not answering, so the
  honest message is "kan ikke nå serveren" — not a half-trustworthy screen. This is
  a meaningful simplification: aggregate numbers such as PRD 001's "in our care"
  count are either correct or absent, never plausible-but-wrong.
- **No spinners on navigation.** Cached pages render immediately; background
  revalidation is silent. Reserve spinners for genuinely empty caches.
- **Preserve table state** across navigation — scroll, sort, filters — via
  `<KeepAlive>` on list views. Losing an operator's place mid-call is a real cost.
- **Preload the most-used route chunks** after boot, so the first navigation does
  not pay a lazy-chunk download.
- **Do not persist entity data to `localStorage`.** Instant paint from disk is
  tempting, but stale operational data is dangerous on a dispatch desk.
  Structural data (e.g. organisation sections) is fine. If revisited, stale
  content must be visibly marked.

## 8. Technical Considerations

### Transport choice

| Option | Latency | Cost in this stack | Verdict |
|---|---|---|---|
| Short-interval polling | up to the interval | Trivial; no new infrastructure | **Fallback / day-one implementation** |
| **SSE** | ~immediate | Low: plain HTTP, `http.Flusher`, `EventSource` with built-in reconnect | **Chosen** |
| WebSocket | ~immediate | Upgrade handling, proxy config, heartbeats, a library — and bidirectionality we do not need, since writes go over REST | Rejected |
| Long-polling | ~immediate | Awkward with Go's handler model | Rejected |
| NATS-over-WebSocket to the browser | ~immediate | Exposes the broker, needs subject-level authorisation, leaks the event model into the SPA | Rejected |

Why SSE fits this codebase specifically:

- **Auth already works.** `EventSource` cannot set an `Authorization` header but
  does send cookies same-origin — and hq authenticates with a JWT cookie
  (`JWT_COOKIE_NAME: jwttoken`). A header-token scheme would have forced a
  query-string token or a websocket.
- **The API is already a stream consumer.** `xstream.NewMux(js)` +
  `AddConsumer(...)` (`go/cmd/api/main.go:177-178`) means the hub is another
  consumer; no polling MySQL for changes.
- **Same origin in every environment, for free.** In production the Go binary serves
  the built SPA *and* the API from one process on one port, so the stream is
  same-origin by construction — no CORS, and credentials (basic auth now, JWT cookie
  later) are sent automatically. In dev, Vite proxies `/api` under the same host, so it
  holds there too.
- **No shared bus or sticky sessions are needed.** Each API process consumes the
  whole stream, so it can serve its own SSE clients unaided. Note PRD 005 assumes
  **one instance per service per server**, so this is not about horizontal scaling —
  it matters during a blue/green switch, when two *different builds* run side by side
  and each serves the clients routed to it.

### Signals, not data

The stream carries `{entity, id, year, event}` and nothing more. One
serialization path (REST), no risk of pushed and fetched shapes drifting, tiny
payloads, and no per-entity authorisation model. The cost is a round trip per
changed item, which is negligible at this data size. If one list later proves hot
enough to matter, promote *that* payload to a snapshot — do not design for it up
front.

This is the direct fix for legacy trap §2.1.1: there is no second read model to
disagree with the first.

**Reserve the SSE `event:` name as an extension point.** `entity.changed` should not
be the only thing this stream can ever carry — PRD 005 proposes a `version.changed`
signal so the SPA learns a new build was deployed, and this stream is its natural
carrier. Dispatch on the event name in the client from day one, so a second type is
additive rather than a format change. Do not invent further types speculatively.

### Deriving signals generically

`subject.StringSubject` already gives what the parser needs:

- `Parts()` splits the subject into tokens (`subject/subject.go:48`).
- `FromStr` **normalises the first `:` into `.`** (`subject/subject.go:22`), so the
  mixed `NATHEJK:` (45 occurrences) and `NATHEJK.` (83) forms in hq need no
  special handling and no cleanup first.

So one decorator over any `stream.Consumer` yields signals for every entity:

```go
mux.AddConsumer(
    notify(hub, patruljetable),
    notify(hub, paymenttable),
    notify(hub, ordertable),
    // …
)
```

`stream.Consumer` is just `Consumes() []Subject` + `HandleMessage(Message) error`
(`cqrs.go:90-93`), so the decorator needs no upstream cooperation.

### Dependencies: ids are not enough

The most consequential gap found while reviewing this design. A signal names one
entity instance (`payment/123`), but much of what the SPA renders is **not** an
entity instance:

- **Collections.** A newly created SOS case produces a signal for an id the client
  has never seen. If the cache only reacts to ids it already holds, the case never
  appears in the list — the single most visible thing this feature is supposed to do.
- **Derived aggregates.** PRD 001's in-our-care count is a `COUNT(*)` over
  `spejderstatus`; team strength is a count per team. Neither has an id. They change
  when *some member* changes, and no `entity:id` key expresses that.
- **Cross-entity derivations.** A team's paid status derives from `payment` and
  `order` rows; a case's timeline derives from member events for teams it is
  associated with. The entity that changed is not the entity being displayed.

So the cache primitive needs **dependency declaration**, not just a key:

```ts
useLiveResource('sos:list',   fetchList,  { dependsOn: ['sos'] })
useLiveResource('care:count', fetchCount, { dependsOn: ['spejderstatus'] })
useLiveResource(`sos:${id}`,  fetchOne,   { dependsOn: [`sos:${id}`, 'sos_activity'] })
```

A signal invalidates every resource declaring a dependency on that entity **type**
or that exact **instance**. Consequences worth accepting deliberately:

**Decided:** dependencies may be entity **types** or specific **instances**, and
`dependsOn` is **mandatory** at every call site even when empty — a missing
declaration fails silently, so it must be visible in review. Rejected: having the
server compute view-level signals; it would couple the hub to every aggregate and
reintroduce the per-entity backend code this design exists to avoid (and it is how
the legacy `dims` channel acquired its second read model).

- Type-level dependencies are coarse: any scan invalidates anything depending on
  `scan`. That is fine for cheap queries and wrong for expensive ones. **Deferred:**
  no per-resource debouncing until something demonstrably hurts — central coalescing
  plus the boot gate (replay reaches no client) removes the obvious burst sources, so
  adding a second timer now would be speculative.
- The entity names in declarations are the **subject tokens**, not UI names: the
  betalinger page depends on `payment` and `order`; `gøgler` contains a non-ASCII
  character; `checkgroups` (plural) appears as a collection-level subject. Cache keys
  and the `?entities=` filter must handle those exact strings.

**Client-side fan-out: reuse the existing bus.** `vue/src/plugins/bus` already exports
a `mitt` instance (the axios plugin imports it), so one `EventSource` owned by a
plugin publishes signals onto the bus and `useLiveResource` subscribes — no second
emitter, and pages never touch the stream directly.

### Ordering: signal after the write

If the hub subscribes independently, it can signal before the projection commits;
the client refetches, reads the old row, and shows stale data no later event
corrects. So the notifier **wraps** the consumer and emits only after
`HandleMessage` returns `nil`.

This is safe because the write is synchronous: `sqlpersister.Writer.Consume`
executes `db.Exec` inline (`sqlpersister/writer.go:70-77`), and MySQL autocommits.

**One caveat, precisely.** `cqrs/deadletter` is a `Writer` decorator that diverts
failing statements to a table instead of failing the projection loop. If a
notified consumer is wired through it, `HandleMessage` can return `nil` while the
read model was *not* updated — producing a signal for a change that is not
visible. Either do not compose `deadletter` beneath `notify`, or treat a diverted
statement as a failure for notification purposes.

### Hub shape: backpressure is the one that bites silently

A slow or stalled client must not block the notifier, and this is where push systems
usually acquire their worst bug. Both obvious options are wrong:

- **Unbounded per-client buffer** — a laptop that sleeps with a tab open becomes a
  memory leak on the API.
- **Bounded buffer that drops signals** — the client silently misses an invalidation
  and shows stale data indefinitely, with no error anywhere. Strictly worse than
  having no live updates, because the UI *looks* live.

The rule: **bounded buffer per client, and on overflow do not drop — collapse.**
Replace the backlog with a single `resync` signal meaning "you have missed
something, revalidate everything you hold". The client already has that code path —
it runs on every reconnect (§5) — so overflow degrades into well-tested behaviour
rather than a new one. A slow client gets coarser updates; it never gets wrong ones.

Two related shape decisions:

- **Coalesce once, centrally, then fan out.** Per-connection coalescing multiplies
  buffers and timers by the number of operators for no benefit — every client wants
  the same `(entity, id)` collapse.
- **Bound writes in time.** A write to a dead connection must not pin a goroutine;
  use the request context plus a write deadline so stale connections are reaped.

Mundane but easy to omit: set `Content-Type: text/event-stream` and
`Cache-Control: no-cache`, and disable intermediary buffering
(`X-Accel-Buffering: no` is harmless and helps if a buffering proxy ever appears in
front).

### Serving SSE through this API — decided

**Mount `/api/stream` outside the `Metrics` middleware.** Everything under `/api/`
currently runs as `app.Metrics(app.authenticate(router))` (`go/cmd/api/routes.go:95`),
and `metricsResponseWriter` implements `Header`, `WriteHeader`, `Write` and `Unwrap`
but **not `Flush`** (`app/middleware.go:10-38`) — so the idiomatic
`w.(http.Flusher)` assertion fails and nothing is ever flushed to the browser.

Skipping the middleware fixes this directly: `authenticate` does not wrap the
writer (it only augments the request context, `routes.go:184-187`), so with `Metrics`
out of the path the handler receives the raw `http.ResponseWriter` and flushing
works. `http.ServeMux` prefers the longer pattern, so registering `/api/stream`
alongside `/api/` is enough:

```go
mux.Handle("/api/stream", app.authenticate(streamHandler)) // no Metrics
mux.Handle("/api/", app.Metrics(app.authenticate(router)))
```

It also removes a second problem: a stream open for hours would otherwise be
recorded as one multi-hour request, skewing `total_processing_time_μs`.

Note the same mux also serves the SPA in production, with a fallback filesystem that
returns `index.html` for unknown paths (`SpaFileSystem`, `routes.go:110-123`). An
explicit `/api/stream` pattern is matched before that fallback, so this is safe — but
it is worth knowing that a typo in the route would not 404, it would quietly return
`index.html` and the client would fail parsing an HTML "event stream".

**Use `http.NewResponseController(w)` anyway.** One line, works whether or not the
writer is wrapped, and immune to someone adding middleware later — which is exactly
how this trap would return. Belt and braces, not redundancy.

(The metrics middleware is barely used in practice; whether to keep it at all is a
separate cleanup, not this PRD's business.)

### Authentication — decided

**Ship at parity with the rest of the API, and change nothing when auth changes.**
The current posture, confirmed:

- **Dev:** no authentication. `app.authenticate` has its body commented out and
  injects an anonymous user (`routes.go:125-191`).
- **Prod / stage:** HTTP **basic auth** with a single shared username/password,
  enforced outside this repo (infra-level, not in `docker-compose.yml`).
- **Planned:** a proper auth tool issuing a **JWT**.

The stream works under all three without special handling, which is the useful part:

- **Basic auth is compatible with `EventSource`.** It cannot *set* headers, but it
  does not need to — once the browser holds credentials for the origin it replays
  them automatically on subsequent requests, including the stream. A 401 will
  trigger the browser's own credential prompt, which is ugly but functional.
- **Cookie-borne JWT is compatible too**, same-origin, for the same reason cookies
  work today — so the migration needs no change here. A token in an `Authorization`
  header would have been the one scheme that broke this, and that is precisely what
  the plan avoids.
- The **exposure is small either way**: signals carry ids, never entity data, so the
  worst an unauthenticated client learns is that `payment/123` changed — not what
  changed. It does leak activity timing and id existence, which is acceptable for an
  internal tool but worth stating rather than glossing.

Two consequences to carry:

- The 401-on-reconnect path (§5) **cannot be exercised in dev** — there is no auth
  there. Test it against stage.
- Under JWT the client should surface a re-login prompt rather than let the browser
  prompt; that behaviour belongs with whoever implements the auth migration, and this
  PRD only needs the stream to fail cleanly on 401.

### Boot and readiness (out of scope, but load-bearing)

**The API does not serve requests until its projections are fully caught up.** It
serves live data or nothing — never the read model as it looked at some arbitrary
point in history. That gating is specified in **PRD 005 — Boot gate, deployment &
SPA reload** (`roadmap/prd/005-boot-gate-deployment-spa-reload.md`), not here. This
PRD depends on it and is materially simpler because of it:

- No staleness indicator, no "replaying" banner, no partial-data caveats anywhere
  in the UI. A connected client is talking to a caught-up API by construction.
- The only failure the client must model is **unavailable**, not **wrong**. During
  a restart, requests and the stream fail; the client reconnects and revalidates
  when the API returns. Distinguishing "cannot reach the server" from "the server
  is lying to me" is the whole benefit.
- Aggregate figures — PRD 001's "in our care" count is the sharp example — are
  either correct or absent. There is no window in which they look plausible and are
  not.

Two findings from this PRD's investigation, recorded in PRD 005 §8 and repeated
here because they shaped the design above:

- The mechanism already exists: `stream.CatchupListener` is an optional interface
  on the handler (`stream.go:126-134`); the JetStream transport type-asserts it,
  treats an empty backlog as caught up immediately, and otherwise reports when
  `NumPending == 0` (`jetstream/stream.go:149-155`, `205-220`). Readiness is the
  conjunction of `CaughtUp()` across every registered consumer.
- **`xstream.MuxBlockUntilLive()` looks like exactly the right tool and is not.**
  It is declared and sets `opts.blockUntilLive` (`xstream/options.go:11-13`), but
  the branch that would honour it is commented out (`xstream/mux.go:37-39`), so
  calling it silently does nothing. Either implement it upstream or gate on
  `CaughtUp()` directly — but do not assume the option works.

Readiness also has to be visible to the proxy so Traefik does not route to an
instance that is still replaying — and PRD 005 covers the harder consequence: a
blue/green instance replaying into the **shared** read-model database would rewrite
rows the live instance is serving. That is a deployment problem, not a live-updates
problem, but it is the reason the gate alone is not sufficient.

### Upstream (`jrgensen/stream`, `jrgensen/cqrs`)

**No upstream change is required.** Verified against `stream v0.1.2` and
`cqrs v0.1.0`:

| Need | Status |
|---|---|
| Wrap a consumer to observe handled messages | `stream.Consumer` / `cqrs.Consumer` is a two-method interface — decorate freely |
| Read entity/id/event from a message | `Message.Subject()` + `subject.Parts()`; `:`/`.` already normalised |
| Know when replay has finished (for the boot gate) | `stream.CatchupListener` + JetStream tracker — exists, though the gate itself is out of scope |
| Resume token for SSE `Last-Event-ID` | `Message.Sequence()` exists on the interface |
| Guarantee the projection wrote before signalling | `sqlpersister` executes synchronously; `HandleMessage() == nil` suffices |

One **defect worth reporting upstream**, which this PRD does not depend on but the
boot gate probably will: `xstream.MuxBlockUntilLive()` is declared and sets
`opts.blockUntilLive` (`xstream/options.go:11-13`), but the branch that would
honour it is commented out (`xstream/mux.go:37-39`), so calling it silently does
nothing. Given that the API must not serve until caught up, this is the one place
an upstream change is plausibly wanted — either implement the option or remove it
so nobody relies on it. Precise ask: honour `opts.blockUntilLive` in `mux.Run` by
waiting for `CaughtUp()` from every registered consumer before returning, or delete
the option and document `CatchupListener` as the supported route.

If anything *does* turn out to need upstream work, the rule is to state the exact
symbol, file and desired behaviour — as above — rather than requesting a general
capability.

### Frontend

- `vue/src/plugins/axios`'s `http` stays the only HTTP path (baseURL `/api/`,
  `X-YearSlug` interceptor). The stream is a sibling plugin, not a replacement.
- One `EventSource` per app, owned by a plugin/composable; pages never open their
  own.
- Cache keyed `entity:id` plus named collections, with a derived list — not arrays
  used as maps (legacy trap §2.1.7).
- Explicit `pending` / `error` / `data`, once, in the primitive (trap §2.1.6).
- `KeepAlive` on list views; route-chunk preload after boot.
- Pages opt in by replacing their `onMounted(fetch)` with the primitive; no page is
  forced to migrate.

### Data / storage

None. No new tables, no schema changes, no persistence. The hub is in-memory per
API instance and holds only connected clients and a short coalescing buffer.

### Dependencies & risks

- **Sequencing tension with PRD 005.** This PRD is the foundation to build first,
  but its "no staleness UI" simplification only holds once the API refuses to serve
  a partially-replayed read model. The gate itself is a *small* slice of PRD 005 —
  gate HTTP serving on `CaughtUp()` across consumers — quite separable from
  per-build databases and blue/green. Recommendation: pull just that slice forward
  alongside this work, and treat the rest of PRD 005 as genuinely later. Otherwise
  this PRD ships with an unstated assumption that is briefly false on every restart.
- **Depends on PRD 005 (boot gate).** The "no staleness UI" simplification above is
  only sound if the API never serves a partially-replayed read model. If PRD 005
  does not land, this PRD and PRD 001 both need staleness indicators added back —
  so it is a design dependency, not just a sequencing one.
- **Proxy behaviour is the one real unknown — and the two environments differ.**
  **Spike this first**; it is the only finding that could force polling or a websocket
  instead.

  - **Production / stage:** `browser → Traefik → Go`. One hop. The Go binary serves
    both `/api/…` and the built SPA from the same process and port, so the stream is
    same-origin by construction. What to verify: Traefik's idle timeout (heartbeats
    address it), that no `buffering` middleware is attached to the router (it would
    break streaming), and that the basic-auth middleware only inspects the request —
    it should be transparent to a streamed response.
  - **Development only:** `browser → Traefik → Vite → Go`. Vite proxies `/api` to the
    `api` container, adding a hop that production does not have. Vite's proxy streams
    by default and needs no special configuration for SSE (unlike websockets, which
    need `ws: true`), but it is unverified here.

  The distinction matters for triage: **a failure in the dev path is a developer
  annoyance, not a production risk** — yet it will present as "the feature is broken",
  which is exactly how a sound design gets abandoned. Test both, and label which is
  which. It is also why the surface shrank twice: the `jrgensen/gateway` container is
  gone in favour of per-service Traefik labels, and production never had the Vite hop
  at all.
- **Signal volume.** A global stream carries everything, including checkpoint
  scans at peak. Coalescing plus the `?entities=` filter should be enough;
  measure against real scan volume before relying on it.
- **HTTP/1.1 connection limit** (6 per host) — largely resolved: the `ui` service now
  serves over HTTPS via Traefik (redirect + `desec`), so HTTP/2 multiplexing applies
  and a long-lived stream no longer competes with normal requests for a connection
  slot. Worth confirming during the spike rather than assuming.
- **Reconnect storms** if the API restarts with many clients connected: jitter the
  reconnect and keep the revalidation cheap.
- **Scope creep into a data channel.** The moment a payload is added "just for this
  list", the legacy dual-model problem returns. Any such change should be a
  deliberate, documented exception.

## 9. Success Metrics

- Navigating back to a visited page issues no blocking request and shows no
  spinner — verifiable in the network panel.
- Median signal-to-render latency ≲1s during the event.
- SOS (PRD 001) ships on this capability with no SOS-specific transport code.
- At least two further pages (betalinger, patruljer) migrated afterwards with no
  changes to the transport or the primitive — the real test of whether it is
  extensible.
- No operator-reported "the screen was stale/frozen and I did not know".
- Override/refresh behaviour: operators stop pressing browser reload, which is the
  crude signal that the tool is trusted to be current.

## 10. Rollout / Task Breakdown

- **Phase 1 — Client foundation.** Transport interface, `useLiveResource`, cache
  semantics, polling implementation. Ships perceived speed immediately and
  unblocks PRD 001 without any backend work.
- **Phase 2 — Backend push.** `notify` decorator with subject parsing and
  coalescing, `GET /api/stream`, heartbeats. Swap the transport; no page changes.
- **Phase 3 — Adoption.** SOS first (PRD 001), then betalinger, patruljer,
  klaner, poster.

Proposed tasks for `roadmap/tasks/open/`:

- [ ] Task: Spike — verify SSE on the **production path** (Traefik → Go: idle timeout, no buffering middleware, basic auth transparent, heartbeat interval)
- [x] Task: Spike — verify SSE on the **dev path** (Traefik → Vite → Go); a dev-only stall is an annoyance, not a design failure
- [x] Task: Frontend — transport interface + polling implementation
- [x] Task: Frontend — `useLiveResource` primitive: keyed cache, mandatory `dependsOn` (types + instances), stale-while-revalidate, explicit loading states
- [x] Task: Frontend — year switch: tear down stream, clear cache, reconnect, refetch (single source of truth: `globalstate.yearSlug`)
- [x] Task: Frontend — optimistic write helper + reconciliation on signal
- [x] Task: Frontend — connection-state indicator (live / reconnecting / polling / unavailable)
- [ ] Task: Frontend — `KeepAlive` on list views + route-chunk preload
- [x] Task: BFF — generic subject → signal parser (incl. id-less and dotted-event shapes)
- [x] Task: BFF — `notify(hub, consumer)` decorator emitting after successful `HandleMessage`, with per-`(entity,id)` coalescing
- [x] Task: BFF — SSE hub + `GET /api/stream`: mounted **outside** the `Metrics` middleware, `http.NewResponseController` for flushing, `?entities=` + `?year=` params, heartbeats
- [x] Task: BFF — hub internals: central coalescing, bounded per-client buffers, overflow → `resync`, write deadlines
- [x] Task: Frontend — handle `resync` (revalidate everything held) and dispatch on the SSE `event:` name so new signal types are additive
- [x] Task: BFF — wire existing consumers through `notify` (patrulje, klan, payment, order, lok, section, scan, …)
- [ ] Task: Upstream — report `xstream.MuxBlockUntilLive()` as a no-op (implement or remove); flag it to whoever builds the boot gate
- [x] Task: Frontend — the API advertises the entity tokens it can emit (`entities` frame) and the SPA warns in dev about a `dependsOn` nothing can satisfy (task 040)
- [ ] Task: Adopt on SOS (PRD 001), then ~~betalinger, patruljer, klaner, poster~~ — **all but SOS done** (tasks 036, 037: patruljer, betalinger, poster, badutter, klaner, kort, forsiden, patrulje detail, active scan trail). SOS waits on PRD 001 being built.

## 11. Open Questions

- **~~Proxy behaviour~~ — answered (task 035), and it was not the proxies.** SSE survives
  the dev path (Traefik → Vite → Go) with no buffering: resync at 0.05s, heartbeats at
  20.1s / 40.1s / 60.1s. HTTPS serves over HTTP/2, so the HTTP/1.1 connection-limit
  concern is moot. The spike did find a blocker, in our own code: the server's
  `WriteTimeout` (30s, `app/server.go:22`) is a deadline on the *whole* response, so the
  stream delivered its first events and then died silently mid-flight — which reads
  exactly like proxy buffering. Fixed by clearing the write/read deadlines per response
  via `http.ResponseController`, leaving the global timeout protecting every other
  endpoint. Regression test included, verified to fail without the fix.
  Still outstanding: the production-shaped path (Traefik → Go), one hop fewer than the
  verified dev path.
- **Two Traefik faults found while verifying, both worth knowing:** naming a router's
  `.service` that nothing defines silently **disables** the router (Traefik reports it
  only via its API), and the shared `redirect-to-https` middleware the org rules
  describe does not exist in the running Traefik — hq defines a repo-scoped
  `hq-redirect-to-https` instead. Decide whether to add the shared one to the infra repo
  and switch back.
- **Confirmed working end to end (tasks 036 and 037).** The patrulje list composes
  `useLiveResource`; an edit in one browser tab appears in another without navigation,
  and returning to the page costs no request. Task 037 rolled this out to eight more
  views — betalinger, poster, badutter, klaner, kort, forsiden, the patrulje detail and
  the active-patrulje scan trail — all confirmed in two tabs by the user.

  Two of those needed a guard the read-only pages did not: **kort** and **klaner** hold
  unsaved operator state (dirty marker positions; an unsaved LOK arrangement), so a
  background revalidation would have destroyed work in progress. Both defer incoming
  payloads while dirty and apply them when the edit ends, and klaner says so on screen.
  That is the pattern for any future editor — the mechanical three-line adoption only
  suits pages that own no unsaved state.
- **~~Filtering granularity~~ — answered by measurement (task 037), and the simple
  version wins.** The fear was that depending on scans would revalidate continuously
  during a checkpoint rush. Measured: the busiest minute in the existing scan data is
  **17 scans** (2025-09-20 13:45), and the endpoints involved answer in **~3.5ms**
  (`/checkgroups`, 10KB) and **~3.4ms** (`/patrulje/{id}/scans`, 5.5KB). That is about a
  third of a request per second per open page, so `?entities=` is more than enough and
  per-id subscriptions are not needed. Poster and the active-patrulje trail both depend
  on scans directly. Revisit if either number moves by an order of magnitude.
- **A token that cannot fire is now caught (task 040), which was the real risk in
  `dependsOn` — not granularity.** Two of six tokens in task 037's plan were wrong
  (`scan` for `qr`, `personnel` for `gøgler`/`friend`/`bandit`), and a wrong token fails
  silently: the page looks live and never updates. The API now advertises the tokens
  derivable from its wired consumers, as an `entities` frame on connect, and the SPA
  warns in dev. The set reports whether it is **exhaustive**, because five consumers
  subscribe with a wildcard entity — so the warning is advisory rather than a gate.
  Narrowing those subscriptions would make it definitive, if it ever produces a false
  positive.
- **Coalescing window.** 50–100ms is a guess. Long enough to collapse bursts,
  short enough to feel instant — worth tuning against a real event replay.
- **`Last-Event-ID`: recommend deciding *against* it now.** `Message.Sequence()` makes
  a resume token possible, but signals are *derived and coalesced*, not stream
  messages, so resuming would require the server to buffer emitted signals per
  connection — real state, real memory, real bugs. Blanket revalidation on reconnect
  is cheap at this data size and needs none of it. Worth settling before the endpoint
  contract is written, since adding an `id:` field later is easy but promising resume
  semantics is not.
- **Does the `?entities=` filter change while the app runs?** An `EventSource` URL is
  fixed at construction, so a filter derived from currently-mounted pages would force
  a reconnect on every navigation — defeating the point. Options: subscribe to
  everything and filter client-side (simplest, more traffic); compute a stable
  superset once at boot; or accept reconnects. This needs answering before the
  endpoint contract, because it decides whether the filter is per-connection at all.
- **Is `event` in the payload useful at all?** Coalescing collapses several events for
  one `(entity, id)` into one signal, so the surviving `event` name is arbitrary. It
  should be treated as advisory — never used to decide *what* to refetch. If nothing
  legitimately consumes it, consider dropping it to avoid inviting misuse.
- **Cross-tab sharing.** One `EventSource` per tab is simplest. A `SharedWorker`
  or `BroadcastChannel` would reduce connections for operators with several tabs
  open — worth it only if that turns out to be common.
- **Do non-projection changes need signals?** Some commands publish events that no
  projection consumes; those produce no signal under this design. Confirm nothing
  a page displays depends on such an event.
- **Testing approach.** How do we test the notifier's ordering guarantee?
  `cqrs/cqrstest` and `stream/streamtest` provide in-memory fakes — worth
  confirming they can assert "signal emitted after write".
