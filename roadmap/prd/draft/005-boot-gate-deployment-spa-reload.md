# PRD 005 — Boot gate, deployment & SPA reload

**Status:** draft
**Author:** agent session (captured from the PRD 001/004 design discussion)
**Created:** 2026-08-07
**Last updated:** 2026-08-09
**Approved:**
**Shipped:**
**Status note:** skeleton — sections marked TBD are deliberately unfinished
**Target users:** organizer (indirectly — every operator using the HQ panel), plus
whoever operates deployments

---

> **State of this document.** This is a placeholder capturing everything already
> established or discovered while writing PRD 001 (Nødtelefon / SOS) and PRD 004
> (Live updates for the SPA), so none of it is lost. Sections marked **TBD** are
> deliberately unfinished. Where something was verified in code, the file and line
> are cited so it does not have to be re-derived.

## 1. Summary

Three coupled concerns that together decide whether a deploy or a restart is
invisible to operators or disruptive:

1. **Boot gate** — the API must not serve any request until its projections are
   fully caught up. It serves live data or nothing, never the read model as it
   looked at some arbitrary point in history.
2. **Deployment** — because a caught-up boot takes time, a new version must be
   built up *before* it takes traffic. Blue/green (or equivalent) is the intended
   shape.
3. **SPA reload** — when a new version is deployed, browsers running the old SPA
   must be told to reload, promptly and safely.

## 2. Problem & Motivation

- **What problem does this solve?**
  - Serving a partially-replayed read model means answering with numbers that look
    plausible and are wrong. PRD 006 has the sharpest example: a "how many people
    are in our care right now" count that is quietly incomplete is worse than no
    count at all.
  - A caught-up boot is not instant, so naive restarts mean downtime proportional
    to stream length — during an event, at the worst possible moment.
  - Operators keep the panel open for hours. Without a reload mechanism they run
    old frontend code against a new API indefinitely, and hit broken lazy chunks
    (§8) with no idea why.
- **Why now?** PRD 004 depends on the boot gate for a real safety property: with
  it, live data is guaranteed and the client only has to model *unavailable*, never
  *stale*. That removes an entire category of staleness UI from every page. The gate
  is the enabling assumption, so it needs to exist.
- **Evidence / current state.**
  - `mux.Run` does **not** block until live (`go/cmd/api/main.go:181`), so a
    freshly started API serves reads immediately, whatever state the projections
    are in.
  - `cqrs` documents the cost explicitly: "Projections are rebuilt by replaying the
    stream from the beginning on every boot, so every event will be handled again"
    (`cqrs.go:87-89`). Boot time therefore grows with total history, not with
    downtime.
  - `/api/v1/healthcheck` (`routes.go:94`) returns a hardcoded
    `"status": "available"` (`app/healthcheck.go:7-17`) — it is a liveness probe
    that cannot express readiness.
  - The legacy platform did gate: `dims.Subscribe()` blocked on a `live` channel
    before returning — slow boot, correct reads. The intent had already half-rotted
    (`WaitLive()` was left a no-op stub), which is a caution about relying on an
    unenforced convention.

## 3. Goals

- The API answers no request — REST or stream — until every projection has caught
  up.
- Readiness is observable to the proxy, so traffic is never routed to an instance
  that is still replaying.
- A deploy causes no operator-visible downtime: the new version is caught up
  before it takes traffic.
- A browser running a superseded SPA build learns about it quickly and reloads
  without losing the operator's work.
- Restart cost is understood and bounded — we know roughly how long a caught-up
  boot takes and it does not grow unmanageably through an event.
- Each build owns its own read model, so a starting build can never disturb the
  serving one.
- A deploy is repeatable by someone who did not design it, and the gap between
  "documented" and "automated" is written down rather than tribal.
- A new version can be verified against the outgoing one before it serves anything,
  mechanically rather than by judgement.

## 4. Non-Goals

- Zero-downtime *database* migrations. Out of scope here.
- Rewriting the CI/CD pipeline. The existing GitHub Actions → DockerHub
  (`nathejk/hq`) flow stays; this concerns what happens at rollout.
- Live updates themselves — PRD 004 owns the stream and the client cache.
- Snapshotting or checkpointing projections as a way to skip replay. Worth
  considering (§11) but not assumed.
- Per-page behaviour on reload. Individual features may want to preserve drafts;
  this PRD defines the mechanism, not each screen's use of it.

## 5. User Stories & Scenarios

- As an **operator**, I never see a screen that shows confidently wrong numbers
  because the server was still starting up.
- As an **operator**, a deploy during the event does not interrupt me; at most I am
  asked to reload at a moment that suits me.
- As an **operator** on the phone, a reload prompt does not discard what I was
  typing.
- As a **deployer**, I can roll out a new version during the event and know it is
  serving only once it is fully caught up.
- As a **deployer**, I can roll back quickly if the new version is bad.

### Primary happy path (deployment)

1. New image is published.
2. Green starts alongside blue, connects to JetStream and replays into **its own**
   database. This is **completely invisible**: green takes no traffic, affects no
   interface, and touches no data blue is serving. However long it takes and
   whatever it does, no operator can tell.
3. Green reports caught up. Now **both versions are running with independent read
   models built from the same stream** — so they can be compared against each
   other (§8 "Pre-switch verification").
4. The comparison passes. Traffic switches from blue to green.
5. Blue drains — including long-lived SSE connections (PRD 004) — then stops.
6. Browsers on the old build notice the version change and reload, ideally without
   the operator noticing.

The shape of this matters: **only step 4 is user-visible, and it is brief.** All the
risk and all the time sit in steps 2–3, where nothing is at stake. That is what makes
deploying during an event thinkable at all — though still only when genuinely
needed.

### Edge cases & error scenarios

- Green never becomes ready (bad build, broken projection): blue keeps serving,
  rollout aborts.
- Green becomes ready but is faulty in a way readiness does not capture: rollback
  path needed. **TBD.**
- An operator ignores the reload prompt indefinitely and their SPA breaks against
  the new API (§8 lazy chunks). Needs a forced-reload escalation.
- An operator is mid-form when the prompt appears — must not lose input.
- Deploy happens while an operator has several tabs open.
- Replay takes longer than the deployment system's readiness timeout.

## 6. Requirements

### Functional — boot gate

- [ ] The API serves **no** HTTP traffic until all registered consumers report
      caught up.
- [ ] A **readiness** signal distinct from liveness, suitable for the proxy and for
      deployment tooling to poll.
- [ ] `/api/v1/healthcheck`'s hardcoded `"status": "available"` must stop being a
      lie — either it reflects real readiness or a separate readiness endpoint is
      added and the health one is documented as liveness only.
- [ ] Startup logging that makes replay progress visible, so a slow boot is
      diagnosable rather than mysterious.

### Functional — deployment

- [ ] Each build uses its **own database**, named deterministically from the build
      (`hq_<build>`), created on boot with `CREATE DATABASE IF NOT EXISTS`.
- [ ] The database name is a pure function of the build, so a restarting container
      reattaches to its own read model instead of creating a new one.
- [ ] Dev pins to a single fixed database name — `init-dev` restarts on every file
      change and must not create a database per restart.
- [ ] **Dev recreates its schema on boot** (drops its projection tables or its
      database), so it does not become the one environment where projection schema
      drift survives. Nearly free: dev already replays the whole stream on every
      start. Must be explicitly opted into, never inferred from a missing
      `BUILD_NUMBER`, so it cannot fire outside dev.
- [ ] Old build databases are cleaned up, retaining at least the current and the
      previous build.
- [ ] A new version is fully caught up before it receives traffic.
- [ ] Building up a new version is **invisible**: it takes no traffic, affects no
      interface, and writes only to its own database. Boot duration is therefore not
      an operational risk, only a wait.
- [ ] Each instance exposes a diagnostic report of `{lastAppliedSequence, counts…}`
      so two versions can be compared at the **same stream sequence**.
- [ ] Before switching, blue and green are compared automatically; undeclared
      differences block the switch.
- [ ] A deploy can **declare expected differences** (per table/aggregate), so a
      deliberate projection change does not fail its own gate.
- [ ] The previous version keeps serving until the new one is ready.
- [ ] Connection draining on switchover, allowing for long-lived SSE connections
      (PRD 004). Clients reconnect automatically, so draining can be brisk but
      must not be abrupt mid-response.
- [ ] A documented rollback procedure. **TBD.**
- [ ] A **deploy checklist**: every step a deployer must perform by hand, including
      the things that happen *around* the deploy rather than in it. It has two jobs:
      make a deploy repeatable by someone who did not design it, and serve as the
      explicit backlog of what must be automated before automatic deployment is
      credible. Every entry is annotated with whether it is manual today and what
      automating it would require.

### Functional — SPA reload

- [ ] The SPA can detect that a newer version is deployed.
- [ ] Reload is **silent where it can be**: restore the current route, scroll
      position and table state, so an operator who was not typing notices nothing.
- [ ] It falls back to a prompt — never a surprise reload — when there is unsaved
      input or an in-flight request.
- [ ] Escalation to a forced reload when continuing is genuinely unsafe (e.g. a
      failed lazy chunk import) — **policy TBD**.
- [ ] Unsaved input survives, or the prompt is deferred while input is pending.
      **TBD which.**

### Non-Functional

- **Boot time:** measured and bounded. **Target TBD** — needs a real measurement of
  full-history replay first.
- **Deploy frequency:** must be safe to deploy *during* an event, not only between
  them.
- **Honesty over availability:** given the choice, refusing to answer beats
  answering with a partial read model. This is the ordering principle behind the
  whole document.

## 7. UX / UI Notes

- **Reload prompt:** unobtrusive, dismissible, persistent once shown ("Ny version
  tilgængelig — genindlæs"). Not a modal that steals focus mid-call.
- **Unavailable state:** during switchover the SPA may briefly fail to reach the
  API. PRD 004's connection indicator already covers this; the point is that the
  operator sees "cannot reach the server", never wrong data.
- **No "data may be incomplete" state exists.** The gate makes it unrepresentable,
  which is why PRD 004 has no staleness UI at all.
- Wording, placement, escalation behaviour: **TBD.**

## 8. Technical Considerations

### The boot gate — mechanism already exists upstream

Verified against `stream v0.1.2` / `cqrs v0.1.0`:

- `stream.CatchupListener` is an **optional interface on the handler**: implement
  `CaughtUp()` and the transport calls it once the subscribed backlog is drained
  (`stream.go:126-134`).
- The JetStream transport implements the tracking: it type-asserts the handler,
  treats an empty backlog as caught up immediately, and otherwise reports when
  `NumPending == 0` (`jetstream/stream.go:149-155`, `205-220`).
- The mux passes each consumer as the handler (`xstream/mux.go:49`), so any
  consumer — or a decorator around it, such as PRD 004's `notify` — can implement
  `CaughtUp()`.
- Readiness is therefore the **conjunction of `CaughtUp()` across every registered
  consumer**.

**One upstream defect, precisely.** `xstream.MuxBlockUntilLive()` looks like exactly
the right tool and is not: it is declared and sets `opts.blockUntilLive`
(`xstream/options.go:11-13`), but the branch that would honour it is commented out
(`xstream/mux.go:37-39`), so calling it silently does nothing.

Precise ask, if we want it upstream: *honour `opts.blockUntilLive` in `mux.Run` by
waiting for `CaughtUp()` from every registered consumer before returning* — or
delete the option and document `CatchupListener` as the supported route. Either is
fine; the current state, where calling it appears correct and does nothing, is not.

### Read model per build — the decided approach

**Each build gets its own database on the shared database server, named after the
build.** This removes the hazard that would otherwise sink blue/green: two
instances sharing one read model.

The hazard, for the record: every instance replays *from the beginning* on boot and
upserts as it goes, so a new instance warming up against the shared `hq` database
would rewrite the very rows the live instance is serving, transiently setting them
back to older values. The live instance would serve wrong data *because* a new one
was starting — precisely what the boot gate exists to prevent. Separate databases
make the overlap harmless: the outgoing build keeps serving its own read model while
the incoming build builds its own.

**Operating assumption:** never more than one container instance of a given service
per server, so a database is owned by exactly one process at a time. Blue/green
means two *different builds* coexisting briefly, each with its own database — not
two instances of the same build.

**Why today's practice cannot simply continue.** Stage and prod currently *clear the
read-model database before deploy*, which is what makes projection schema changes free
there today. That works only because a deploy is stop-then-start: nothing is serving
while the schema is recreated. Blue/green removes that assumption — clearing a shared
database would destroy the read model the **outgoing** build is still serving traffic
from. So per-build databases are not an alternative to clearing; they are what lets the
clearing habit survive the move to zero-downtime deploys, by giving each build its own
empty database instead of emptying a shared one.

#### Naming

Derive from the build, e.g. `hq_<build>`. Details to settle:

- **Source of the identifier.** The prod image already carries `BUILD_NUMBER`,
  `GIT_COMMIT`, `GIT_BRANCH` and `BUILD_VERSION` as environment variables
  (`docker/Dockerfile:94-101`), so the process can derive its own database name with
  no deployment-side templating. `BUILD_NUMBER` is the natural choice: CI tags
  images `{branch}.{run_number}`, so the run number is short and monotonic.
- **Sanitisation is required.** MySQL identifiers are limited to 64 characters, and
  branch names contain `/`, `.` and `-`, none of which belong in a database name
  unquoted. Lowercase, replace anything outside `[a-z0-9_]` with `_`, truncate, and
  — if the branch is part of the name — append a short hash so two truncated names
  cannot collide.
- **Determinism matters more than uniqueness.** The name must be a pure function of
  the build, so a container that restarts (crash, OOM, manual bounce) reattaches to
  *its own* database rather than creating another one.
- **Dev must pin to a fixed name.** `init-dev` rebuilds and restarts the API on
  every file change, so a name that varied per restart would litter the server with
  databases within minutes. With the name derived from `BUILD_NUMBER` (absent or `0`
  in dev) this falls out naturally — but it needs to be deliberate, not accidental.

#### Creation

Cheap, because the schema already bootstraps itself: every projection issues
`CREATE TABLE IF NOT EXISTS` on construction. So boot needs only
`CREATE DATABASE IF NOT EXISTS <name>` before the consumers are wired, and the rest
follows.

- `DB_DSN` (`main.go:103`) currently carries the database name inline
  (`root:ib@tcp(mysql:3306)/hq?…`). Either the DSN becomes server-level and the
  process appends the derived name, or the deployment substitutes it. The former
  keeps the logic in one place and works in dev unchanged.
- **Privileges.** The connecting user needs `CREATE DATABASE`, and `DROP DATABASE`
  for cleanup. Dev connects as `root`, so this is invisible there; production
  should use a dedicated account with exactly those grants rather than inheriting
  root by default.

#### An unexpected benefit: projection schema changes become free

Today `CREATE TABLE IF NOT EXISTS` does nothing to an existing table, so **adding a
column to a `table.sql` has no effect on a database that already has that table** —
silently. The projection then writes to a column that does not exist, or reads a
column that is never populated.

This is not a theoretical risk; it has already cost time, in two distinct shapes worth
telling apart:

- **The live table is behind its `table.sql`.** `shared-go/tables/signup/table.sql`
  declares `year` and `secret`; the dev table, created by an older schema, had
  neither. Every replayed signup event failed — **17,336 `Error 1054` in a single
  14-hour container lifetime**, in bursts of 2167 per process start. Nothing surfaced
  it: no UI error, no failed request, just log lines. Because the failing statement was
  an `INSERT … ON DUPLICATE KEY UPDATE`, the *update* half never applied either, so a
  team correcting its email or phone would have been invisible to hq's exports and
  mail. Dropping the table and letting replay rebuild it restored exactly the same 1345
  rows, now with `year` populated on all of them. Task 038.
- **The `table.sql` is behind its own struct.** `spejderstatus.go:15-16` declares
  `InitialTeamID` and `CurrentTeamID`; `spejderstatus.sql` declares only
  `id, year, status, updatedAt`. A fresh database does *not* fix this one — the schema
  itself is incomplete. It is inert today (`Consumes()` returns an empty slice and
  `HandleMessage` is entirely commented out) so nothing writes those fields, but it is
  dormant scaffolding for the member-lifecycle work PRD 006 specifies, and PRD 006
  will hit it the moment it wires the consumer up.

With a fresh database per build, every deploy creates its schema from scratch, so the
**first** shape disappears: projection schema evolution needs no migrations at all.
That is a substantial simplification and arguably reason enough on its own — read
models are derived data, and treating them as a build artefact rather than as durable
state is the honest model. The second shape is not a migration problem and stays a
review problem.

#### …except in dev, which this design deliberately excludes

The per-build database rule above pins **dev to a single fixed name**, because
`init-dev` restarts on every file change and must not litter the server. That is the
right call, but it has a consequence worth stating plainly rather than discovering
later:

> Once per-build databases ship, **dev becomes the only environment where projection
> schema drift can still happen** — and the only one with nothing to repair it.

Every other environment clears its database before deploy, so `CREATE TABLE` always
runs against an empty schema there. Dev reuses `hq` indefinitely. And the repair
mechanism is the exception rather than the rule: `cqrs.EnsureColumn` is called by
**2 of 10** shared-go entities (`order`, `product`) and **1 of 17** hq-local
`table.sql` files, leaving 24 entities with no repair path at all.

So the environment where a problem is hardest to notice — nobody reads dev logs — is
the one left exposed. That is backwards.

**Proposal: dev drops its projection tables on boot.** This is nearly free, because
dev *already* rebuilds from scratch on every start: the JetStream consumer is an
`OrderedConsumer` with no deliver policy, so it replays the whole stream regardless of
what the database contains (see *What this does not buy*, above). The read model is
already a build artefact in dev — it is simply not treated as one. Dropping the tables
adds a `DROP TABLE` per entity and **no extra replay**, and makes dev behave like every
other environment.

Points to settle:

- **Guard it hard.** This must be impossible to trigger anywhere but dev — keyed off an
  explicit opt-in (an env var set only by `docker-compose.yml`), never inferred from the
  *absence* of `BUILD_NUMBER`, so a misconfigured prod container cannot silently wipe
  its read model on boot. Failing closed matters more than convenience here.
- **Drop tables, or drop the database?** Dropping the database is simpler and matches
  prod's "cleared before deploy" exactly; dropping tables leaves the database and its
  grants alone. Either is fine; the database is probably cleaner.
- **`product` is seeded, not projected.** `producttable.Seed(product.Seeds2026())` runs
  at boot (`main.go:183`), so it should re-seed and be unaffected — but it is the one
  table that is not purely derived, so verify rather than assume.
- **Measure the boot cost first.** The claim "no extra replay" is sound in principle;
  confirm it against a real dev restart before adopting, since `init-dev` runs on every
  file save and developers feel every second of it.
- **Alternative if this proves too slow:** add `EnsureColumn` to the entities that need
  it. That fixes instances rather than the class, and 24 entities would still be
  exposed, so it is the fallback rather than the plan.

#### Retention and cleanup

Databases accumulate, so cleanup is part of the design, not an afterthought:

- Keep the **current** build (serving) and at least the **previous** one (rollback
  target). Drop older ones.
- **Who drops them, and when?** A boot step after the new build reports ready is the
  simplest place — by then the outgoing build is identifiable and the incoming one
  is proven. A separate maintenance job is the alternative. **TBD.**
- Never drop a database belonging to a running container. With one instance per
  service per server this is checkable, but the rule should be explicit.
- **Disk cost is unmeasured** — one full read model per retained build. Needs a
  number before choosing how many to keep.

#### What this does *not* buy

Worth being clear, so nobody expects it: **existing rows do not make boot faster.**
The JetStream consumer is created with `OrderedConsumer` and no explicit deliver
policy, so it replays the whole stream every time regardless of what the database
already contains. A per-build database therefore costs a full build-up per deploy —
the same cost as today — and rollback to a previous build also replays from scratch
rather than resuming. If boot time turns out to be the problem, the fix is
checkpointing or durable consumers (§11), which is orthogonal to this decision.

### Pre-switch verification: compare blue against green

Because both versions are running with independently built read models derived from
the same stream, the new build can be **checked against the old one before it takes
any traffic**. This turns "does this number look right?" from human judgement into a
mechanical comparison — the most valuable property the per-build database buys us,
beyond merely avoiding corruption.

What makes it sound: both instances folded the *same events* through *possibly
different code*. Any difference is therefore either an intended consequence of the
new build, or a bug in it. There is no third explanation.

#### The timing problem, and how to make the comparison exact

Blue and green are not caught up to the same point. Blue has been consuming
continuously; green reached its catch-up point later, and events keep arriving
throughout. Comparing raw counts would produce differences that mean nothing.

`stream.Message` exposes `Sequence()`, so the fix is to make every comparison
**sequence-qualified**: each instance reports its counts together with the sequence
of the last event it applied, and a comparison is only valid between two reports at
the *same* sequence. Practically:

- Each instance exposes a diagnostic endpoint returning
  `{lastAppliedSequence, counts…}`.
- The comparator polls both and compares only reports whose sequences match. In a
  quiet moment they converge immediately; under load, retry until they do.
- A difference that reconciles as sequences advance is drift, not a fault. A
  difference that persists at equal sequence is real.

Without this, the check is noisy and will be ignored within two deploys — which is
worse than not having it.

#### What to compare

Cheap, high-signal aggregates rather than full table dumps:

- Row counts per projection table.
- Domain aggregates that would embarrass us if wrong: open SOS cases (PRD 001),
  PRD 006's in-our-care count, started patrols, discontinued teams, teams below the
  3-member requirement.
- **Money** — payment and order totals. The best canary available: a difference here
  is never acceptable drift.
- Counts grouped by status (`signupStatus`, `MemberStatus`), which catch a
  mis-mapped enum — exactly the class of bug PRD 006's legacy status-value
  normalisation could introduce.

#### Expected differences must be declarable

The obvious trap: a deploy whose *purpose* is to change a projection will fail its
own comparison. PRD 006 is precisely such a deploy — reviving `spejderstatus` makes
counts appear that were previously zero, and normalising legacy status values changes
how members are grouped.

So the gate cannot be "all counts identical". It must be:

- differences are **reported**, and
- a deploy may **declare** which are expected (which table, which aggregate, ideally
  the direction), and
- anything undeclared fails the gate.

That turns the report into a genuinely useful artefact: a deploy that changes
projections has to say so up front, and gets checked against what it claimed.

#### Consequence for the checklist

This is what makes the previously-unautomatable sanity check automatable, and it
removes most of the argument against deploying during an event: boot is invisible,
verification is mechanical, and only the switch is exposed.

### Replay cost

- Replay is from the beginning of the stream on every boot (`cqrs.go:87-89`), so
  boot time grows with cumulative history — and history grows fastest during an
  event, exactly when restarts are riskiest.
- Projections are idempotent by contract, so replay is *safe*; the issue is purely
  time.
- **Unmeasured.** First task: how long does a full caught-up boot take today, and
  what will it be by the end of an event? Everything else (readiness timeouts,
  deploy strategy, whether checkpointing is needed) depends on that number.

### Version identity and SPA reload

**Decision: drop `vcs.Version()` and take the version from an injected build-arg
env var.** Everything needed is already plumbed; the current mechanism is the one
piece that cannot work.

Why `vcs.Version()` has to go — it reads `runtime/debug` build settings
(`vcs.revision`, `vcs.time`, `vcs.modified`; see `go/internal/vcs/vcs.go`), but the
builder stage does `COPY go/ .` with **no `.git` in the build context**, so the Go
toolchain has no VCS information to stamp. It therefore reports nothing useful in
every image we actually ship. It is not merely inelegant, it is inert.

What is already injected and verified end to end:

| Build arg | CI value (`.github/workflows/build-and-publish.yml:69-73`) | Available as |
|---|---|---|
| `GIT_COMMIT` | `github.sha` | prod `ENV` + SPA meta tag |
| `GIT_BRANCH` | `github.ref_name` | prod `ENV` + SPA meta tag |
| `BUILD_NUMBER` | `github.run_number` | prod `ENV` + SPA meta tag |
| `BUILD_VERSION` | `{ref_name}.{run_number}` — **identical to the published image tag** | prod `ENV` only |

The prod stage promotes all four to `ENV` (`docker/Dockerfile:94-101`); the SPA gets
three of them substituted into `index.html` at build time
(`docker/Dockerfile:84-86`) and reads them via `useBuildInfo()`
(`vue/src/composables/useBuildInfo.ts`).

Instruction for implementation:

1. **Use `BUILD_VERSION` as the API's version string**, defaulting to `dev` when
   unset (dev containers do not set it). It equals the image tag, so "which version
   is running" and "which artefact was deployed" become the same string — which is
   exactly what the deploy checklist and rollback need.
2. **Replace both definitions.** There are two independent `version` vars, each
   calling `vcs.Version()`:
   - `go/cmd/api/main.go:55` — feeds `expvar` (`main.go:194`)
   - `go/cmd/api/app/jsonapi.go:14` — feeds `/api/v1/healthcheck`
     (`app/healthcheck.go`)
   Both must read the same env var so they cannot disagree.
3. **Delete `go/internal/vcs`** (33 lines) once both call sites are gone — nothing
   else imports it.
4. **Fix the hardcoded publisher version at the same time.** `metatagger.New(js,
   map[string]any{"producer": "hq-api", "version": "1234"})` (`main.go:133`) stamps
   every published event with the literal `"1234"`. Use the real version: this is the
   field that answers "which build produced this event?" when diagnosing a bad
   deploy, and it is currently a lie in the permanent event log.
5. **Decide what the SPA compares against** (still open). `BUILD_VERSION` is not
   currently exposed to the SPA, so either:
   - add `__BUILD_BUILD_VERSION__` to `index.html` and substitute it alongside the
     other three — then both sides share the same identifier; **recommended**; or
   - have the API also report `GIT_COMMIT`, which the SPA already carries, and
     compare on that.
   Either way the API should expose the whole set (`version`, `gitCommit`,
   `gitBranch`, `buildNumber`) on the healthcheck: it costs four fields and makes
   the deployed artefact self-describing.
6. **Detection mechanism** remains open: (a) a response header on every request,
   (b) polling the healthcheck, or (c) a `version.changed` signal over PRD 004's
   stream — the natural carrier, free once that exists, with the post-switchover
   reconnect as the obvious moment to compare.

### The lazy-chunk trap

The SPA lazy-loads all routes except Home (per `.rules`, `vue/src/router`). Asset
filenames are content-hashed, so after a deploy an old tab navigating to a
not-yet-loaded route requests a chunk that no longer exists → the dynamic import
rejects and the navigation fails, with no obvious cause to the operator.

This is the concrete reason the reload feature is not cosmetic. Mitigations:

- Keep the previous build's assets served through the overlap window (blue/green
  helps, if both slots serve their own assets).
- Catch dynamic-import failures in the router and escalate to a forced reload —
  this is the one case where reloading without asking is the kinder behaviour.

### Draining SSE on switchover

PRD 004's stream is long-lived by design, so every client holds an open connection
to the outgoing instance. `EventSource` reconnects automatically, so switchover is
graceful provided connections are closed cleanly rather than severed mid-event.
Reconnect storms are the thing to watch: many clients reconnecting at once, all
revalidating. Jittered reconnect is specified in PRD 004.

### Data / storage

One database per build on the shared server (see above). Consequences:

- Read models become **build artefacts rather than durable state** — rebuildable by
  definition, which is what makes this safe.
- No projection migrations, ever: each build creates its own schema.
- Disk grows with retained builds; retention policy and a measured per-build size
  are needed. **TBD.**
- The connecting user needs `CREATE DATABASE` / `DROP DATABASE` grants.
- Anything that is *not* a projection must not live in these databases. Worth an
  audit: if any table holds state that is not derived from the event stream, it
  cannot be discarded with the build and needs a different home. **TBD.**

### Dependencies & risks

- **PRD 004** depends on this gate for its "no staleness UI" simplification. If the
  gate does not happen, PRD 004 and PRD 006 both need staleness indicators added
  back — PRD 006 especially, since its counters are the numbers that must be either
  right or visibly unavailable.
- **Replay time is unknown** and is the input to every other decision here.
- **Per-build databases shift the risk rather than removing it:** the failure mode
  is now disk growth and cleanup correctness (dropping a database still in use)
  rather than cross-instance corruption. Both are easier to reason about.
- **Traefik configuration** for readiness-gated routing and draining is unverified.
  Each service now carries its own Traefik labels (the `jrgensen/gateway` container
  has been removed), so the blue/green switch is a Traefik-level concern: how two
  builds' labels coexist, and how the router is moved from one to the other.
- Deploying during an event is inherently risky; the gate reduces the "wrong data"
  risk but raises the "slow rollout" one.

## 9. Success Metrics

- No request is ever served from a partially-replayed read model (structurally
  guaranteed, not merely observed).
- Deploys during an event cause no operator-visible interruption beyond an optional
  reload prompt.
- Measured caught-up boot time stays within whatever bound we set once measured.
- No operator ever encounters a broken lazy chunk without an automatic recovery.
- Rollback demonstrated at least once outside an event.

## 10. Rollout / Task Breakdown

Ordering matters: measure first, because the numbers decide the strategy.

- [ ] Task: **Measure** full caught-up boot time today, and project it to end-of-event history size
- [ ] Task: Per-build database — derive the name from `BUILD_NUMBER` (sanitised, deterministic, dev pinned) and `CREATE DATABASE IF NOT EXISTS` on boot
- [ ] Task: Move the database name out of `DB_DSN` (server-level DSN + derived name), or template it at deploy time
- [ ] Task: Dedicated DB account with `CREATE DATABASE` / `DROP DATABASE` grants instead of root
- [ ] Task: Cleanup of superseded build databases, retaining current + previous, never dropping one in use
- [ ] Task: Measure per-build read-model size to choose a retention count
- [ ] Task: Audit for any non-projection state living in the read-model database (it cannot be discarded with the build)
- [ ] Task: Dev recreates its schema on boot (drop projection tables or database), behind an explicit dev-only opt-in — closes the drift class in the one environment per-build databases leave exposed. Measure the added boot cost first; expected ≈ 0 since dev already replays in full
- [ ] Task: Implement the boot gate — gate HTTP serving on `CaughtUp()` across all consumers
- [ ] Task: Add a readiness signal distinct from liveness; stop `/api/v1/healthcheck` reporting a hardcoded `"available"`
- [ ] Task: Replay-progress logging on startup

Note: a container-level readiness hook was already anticipated —
`docker/Dockerfile:117` carries a commented-out
`HEALTHCHECK … --start-period=900s … CMD test -f /tmp/healthy`. The 15-minute start
period suggests a long caught-up boot was expected even then, and the touch-a-file
pattern is a reasonable readiness signal for Docker/Traefik to consume.
- [ ] Task: Upstream — implement or remove `xstream.MuxBlockUntilLive()` (currently a silent no-op)
- [ ] Task: Blue/green rollout mechanics in Traefik (readiness-gated routing, draining) — **TBD scope**
- [ ] Task: Diagnostic endpoint per instance: `{lastAppliedSequence, counts…}` for projection tables and key domain aggregates
- [ ] Task: Comparator that polls blue and green, matches on stream sequence, and produces a difference report
- [ ] Task: Declared-expected-differences mechanism, with the switch gated on undeclared ones
- [ ] Task: Decide the aggregate set to compare (money totals and status groupings first)
- [ ] Task: Drop `vcs.Version()` — take `version` from the injected `BUILD_VERSION` env (default `dev`) in both `cmd/api/main.go:55` and `cmd/api/app/jsonapi.go:14`; delete `go/internal/vcs`
- [ ] Task: Expose `version` + `gitCommit` + `gitBranch` + `buildNumber` on `/api/v1/healthcheck` so the artefact is self-describing
- [ ] Task: Expose `BUILD_VERSION` to the SPA too (substitute `__BUILD_BUILD_VERSION__` in `index.html`) so both sides compare the same identifier
- [ ] Task: SPA version-change detection (mechanism TBD; likely a PRD 004 signal)
- [ ] Task: SPA reload prompt + unsaved-input safety
- [ ] Task: Router-level dynamic-import failure recovery → forced reload
- [ ] Task: Rollback procedure, documented and rehearsed
- [ ] Task: Write the deploy checklist (seed below), annotate every step with manual/automatable, and keep it current as steps get automated
- [ ] Task: Fix hardcoded `"version": "1234"` in the event publisher (`main.go:133`) — stamp the real version, since this lands in the permanent event log
- [ ] Task: Consider projection checkpointing/durable consumers if measured replay time demands it — note per-build databases do **not** reduce boot time
- [ ] Task: Revisit the commented-out `HEALTHCHECK` in `docker/Dockerfile:117` (`test -f /tmp/healthy`, `--start-period=900s`) as the container-level readiness signal

### Deploy checklist (seed — to be completed and kept current)

Lives with the operational docs rather than in this PRD once written; seeded here
from what this design work already established. The **Auto?** column is the point:
it is the automation backlog, and automatic deployment is only credible once the
rows that matter are ✅.

| # | Step | When | Manual today | Auto? |
|---|---|---|---|---|
| 1 | Confirm the intended image tag/build number is the one published | before | yes | easy — read from CI |
| 2 | Confirm `shared-go` pin is the intended revision (see PRD 001 and PRD 006 — events depend on it) | before | yes | easy — CI assertion |
| 3 | Check JetStream is healthy and reachable; a boot that cannot replay will never become ready | before | yes | easy — pre-flight probe |
| 4 | Check database server disk headroom for one more build's read model | before | yes | easy once per-build size is known |
| 5 | Confirm the DB account has `CREATE DATABASE` (first deploy after the change) | before | yes | one-off |
| 5b | **Clear the read-model database** — today's practice in stage/prod, and load-bearing: it is the only reason projection schema changes need no migrations there. **Per-build databases make this step obsolete** rather than automated, since a new build's database is empty by construction; until then it must not be skipped | before | yes | ✅ superseded by per-build databases (§8) |
| 6 | Announce to nødtelefon operators if deploying during an event | before | yes | partly — could post automatically, but the judgement is human |
| 7 | Deploying in peak hours: only if genuinely needed. Note boot is invisible — only the switch is exposed, so the window of concern is seconds, not the whole rollout | before/switch | yes | partly — can warn/require confirmation; the judgement stays human |
| 8 | Start the new build; watch replay progress | during | yes | yes, once readiness is exposed |
| 9 | Wait for readiness before switching traffic — never switch on "container started" | during | yes | **must** be automated; it is the whole point of the gate |
| 10 | Switch traffic; verify the new build is serving | during | yes | yes |
| 11 | Drain the old container, allowing for open SSE connections (PRD 004) | during | yes | yes |
| 8b | With green caught up and blue still serving, run the **differential comparison** at equal stream sequence (row counts, domain aggregates, money totals, status groupings) | before switch | yes | ✅ fully automatable — this is the point of two read models |
| 8c | Review the difference report; confirm every difference was declared by this deploy | before switch | yes | ✅ gate on undeclared differences; declaration is a human input |
| 12 | Verify a real page loads and a case opens (smoke check) | after | yes | partly — synthetic check |
| 14 | Confirm operators' browsers prompted a reload (once that exists) | after | yes | yes |
| 15 | Record which build is live, and which is the rollback target | after | yes | easy |
| 16 | Drop superseded build databases, keeping current + previous | after | yes | yes, with an in-use guard |
| 17 | Keep the previous image/build available for rollback | after | yes | easy |

The row that used to resist automation — "do these numbers look right?" — is now
**#8b/#8c**, and it is mechanical: two independent read models built from the same
stream can be compared directly, so the only human input is *declaring intended
changes* rather than *eyeballing plausibility*.

What genuinely stays human is **#7**: whether to deploy at all right now. Boot being
invisible shrinks that decision to the switch itself, but a pipeline should still
require explicit confirmation to switch during an event rather than deciding for
itself.

## 11. Open Questions

- **How long does a caught-up boot actually take** — now, and with a full event's
  history? Everything else depends on this.
- **Which identifier names the database** — `BUILD_NUMBER` alone (short, monotonic,
  but not unique across branches) or a branch-qualified name (needs sanitising and a
  collision-safe hash within 64 characters)?
- **Who performs cleanup**, and when — a boot step in the incoming build once it is
  ready, or a separate maintenance job? And how does it verify a database is not in
  use before dropping it?
- **How many builds to retain?** Depends on the measured per-build size and on how
  far back a rollback might realistically need to go.
- **Is blue/green the right shape**, or is a readiness-gated rolling restart
  enough given a single API instance today?
- **What is the readiness timeout**, and what happens when replay legitimately
  exceeds it?
- **How does the SPA learn about a new version** — response header, healthcheck
  poll, or a PRD 004 stream signal? (The identifier itself is settled: the injected
  `BUILD_VERSION`, per §8.)
- **How aggressive should reload be?** Prompt-only risks operators running stale
  code for hours; forced reload risks interrupting a call. Probably: prompt
  normally, force on a broken chunk import.
- **What counts as unsaved work** worth deferring a reload for — a half-typed SOS
  comment certainly; anything else?
- **Rollback:** switch traffic back to the old slot (fast, but its read model may
  now be behind), or redeploy the old image (slow)?
- **Where does the comparator run** — in CI, as a step in the deploy script, or as a
  small tool an operator invokes? It needs network access to both instances while
  only one is routed.
- **How are expected differences declared** — a file in the repo alongside the
  change, a deploy-time flag, or an annotation on the PR? The repo option keeps it
  reviewable with the code that causes the difference.
- **How long do we wait for sequences to converge** before giving up and reporting
  "could not compare"? Under sustained load equal sequences may be rare.
- **Can green's diagnostic report be trusted if green is faulty?** The comparison
  assumes both instances report honestly; a build broken enough to miscount may also
  misreport. Row counts straight from SQL are harder to get wrong than derived
  aggregates.
- **Should dev drop its schema on boot** (§8), and tables or the whole database? Nearly
  free in principle since dev already replays in full — but `init-dev` runs on every
  file save, so the measured cost decides it. If it is not free, the fallback is
  `EnsureColumn` per entity, which fixes instances rather than the class.
- **What is the smallest checklist that is still honest?** A long checklist nobody
  follows is worse than a short one that is always done. Which steps are genuinely
  required every time, and which only after specific kinds of change?
- **Does the gate apply to non-API processes** — are there other consumers or
  workers with the same constraint?
- **Multiple replicas:** is running more than one API instance an actual goal? It
  changes the read-model question substantially.
