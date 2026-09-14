# PRD 014 — Person search by phone number and name

**Status:** doing
**Author:** agent session (2026-09-14)
**Created:** 2026-09-14
**Last updated:** 2026-09-15
**Approved:** 2026-09-14
**Shipped:**
**Target users:** organizer (HQ operators, nødtelefon crew)

---

## 1. Summary

One search box in HQ that takes a phone number or a name and returns every person
it matches — spejder, senior, gøgler, crew member, or a team's contact person —
each with enough context to identify them and a link to the page that holds them.
Backed by a dedicated `search_person` projection so a lookup is an indexed
equality match rather than a scan across six tables.

## 2. Problem & Motivation

**What problem does this solve?** An unknown number rings the nødtelefon and the
operator has no way to find out whose it is. HQ has no search of any kind — a grep
for `search` across `go/` returns nothing — so today the only route to a person is
to already know which list they are on: patrulje, klan, badut, hønsegård,
organisation. When the operator does not know that (and on an inbound call they
never do), the answer is reachable only by opening lists and reading.

The people who could be called are spread over six sources, and each one is
reached by a different page:

| Source | Name | Phone | Who |
|---|---|---|---|
| `spejder` (shared-go) | `name` | `phone`, `phoneParent` | patrol members |
| `senior` (shared-go) | `name` | `phone` | klan members |
| `personnel` | `name` | `phone` | gøglere / badutter / friends |
| `crewmember` (shared-go) | `name` | `phone` | crew |
| `patrulje` | `contactName` | `contactPhone` | patrol contact person |
| `signup` (shared-go) | `name` | `phone`, `phonePending` | klan/crew contact person |

**Why now?** PRD 001 (nødtelefon/SOS) and PRD 006 (member lifecycle) gave HQ a
place to record what happens to a person during the race, and PRD 011 gave it
their position. All of it is keyed on already having found the person. Search is
the missing entry point to work that is otherwise built.

**Evidence.** Two structural facts, rather than tickets:

- A **contact person is not a row anywhere.** For patruljer they are three columns
  on the team; for klaner and crew they exist only on `signup`, because `klan` has
  no contact columns at all. So the most likely caller during signup season —
  the adult who submitted the team — is the hardest person in the system to look
  up, and no existing page lists them.
- A **phone number is free text.** `types.PhoneNumber.Normalize()` strips
  non-digits at read time, but the stored value is whatever was typed: `+45 12 34
  56 78`, `12345678`, `12 34 56 78`. Any lookup that compares stored values
  directly will miss.

## 3. Goals

- An operator with only a phone number can name the person it belongs to, and say
  which team or section they are on, within one interaction.
- An operator with only a (possibly partial, possibly misspelled-by-ear) name can
  find the matching people across every population HQ knows about.
- A number that belongs to a parent rather than a participant is resolved to the
  participant, and says that it was the parent's number that matched.
- Contact persons are as findable as participants.
- Somebody who has left — withdrawn during the race, or removed from a roster
  before it — is still findable, and the result says plainly where they now stand.
  "Ingen match" must never be the answer for a person the event has met.
- Every result is one click from the page where the operator can act.

## 4. Non-Goals

- **Not** a search over non-people entities: poster, kort, loks, dispatch tasks,
  orders. The same projection pattern would extend to them, and deliberately is
  not extended here.
- **Not** fuzzy or phonetic matching (Levenshtein, Soundex, Danish
  name equivalences). Substring matching only; see Open Questions.
- **Not** relevance ranking beyond a fixed, explainable ordering.
- **Not** a write surface. No editing from search results; results link out.
- **Not** cross-year, other than as an explicit act by the operator. The default
  and the whole primary function is *this year*; searching previous years is a
  second, deliberate step, never something the system slides into on its own. See
  Requirements.
- **Not** a lifecycle write surface. Search states a person's current status; it
  does not let the operator change it. That is PRD 006's surface, one click away.
- **Not** a replacement for the existing per-population lists and their filters.

**Related open task.** Task 096 ("Search for a scout in no section") wants a
name/patrol-number lookup on the Hønsegården screen and says to check for a
reusable endpoint before adding one. `/api/search/person` is that endpoint. This
PRD deliberately does not absorb 096: the write action there (**Modtaget i
Hønsegården**) is PRD 007's, and search staying read-only is what keeps it usable
from any screen. 096 should consume this endpoint rather than grow its own.

## 5. User Stories & Scenarios

- As an **HQ operator**, I want to type a phone number that just rang and see who
  it is, so that I can answer the call already knowing who I am talking to.
- As an **HQ operator**, I want to type a half-heard name, so that I can find a
  participant a caller is asking about.
- As an **HQ operator**, I want to reach a contact person, so that I can call the
  adult responsible for a team.

**Happy path.** The operator presses the search shortcut anywhere in HQ, types
`12345678`, and gets one hit: *Anders Andersen · spejder · Patrulje 42 "Ørnene" ·
Hvidovre Gruppe*. Clicking it opens `PatruljeView` for team 42. Total interaction:
one shortcut, eight digits, one click.

**Edge cases and error scenarios.**

- **The number is a parent's.** `12345678` matches `spejder.phoneParent`. The hit
  is the *scout*, labelled "forælders nummer" — because the operator wants the
  scout, but must know they are speaking to the parent.
- **One number, several people.** A parent with two children in the event, or a
  leader who is both a klan member and the patrol's contact. All matches are
  returned; none is guessed at or hidden.
- **The number is formatted differently than stored.** `+45 12 34 56 78` typed,
  `12345678` stored (or the reverse). Both normalize to the same digits and match.
- **A name matches many people.** "Anders" matches thirty. Results are capped and
  the cap is stated on screen, so the operator narrows rather than scrolls.
- **A partial number.** Six of eight digits, read back badly. Falls back to a
  prefix match; see Requirements.
- **Nothing matches.** An explicit "ingen match" that distinguishes *searched and
  found nothing* from *not searched yet* — an empty box and an empty result must
  not look alike.
- **A person who has left stays findable, with their status stated.** This is the
  case search exists for as much as any: the guardian of a scout who went home at
  02:00 rings at 09:00, and the operator must be able to say so. Two different
  departures, and the result must not blur them:
  - **Left during the race.** They are still a `spejder` row; their
    `spejderstatus` is `waiting`, `transit`, `sheltered`, `reunited` or
    `released`. The hit shows that status, so the operator can distinguish *in our
    care right now* from *handed to a guardian hours ago*.
  - **Removed from a roster.** A `spejder.deleted` / `senior.deleted` event takes
    them off the team entirely — a signup-time removal, not a race withdrawal. The
    search row is kept and flagged, and the hit reads "udmeldt", naming the team
    they were on.

  A result whose status is anything but ordinary is marked as such rather than
  looking like every other row, because a stale hit presented as current is worse
  than no hit: the operator would call a scout who went home last night.

## 6. Requirements

### Functional

- [ ] A `search_person` read projection holds one row per findable person, keyed
      by `(kind, id, year)`, covering all six sources in §2.
- [ ] The projection stores each phone number twice: as entered, and normalized to
      digits only. A scout's own number and their parent's are stored separately,
      so a match can say which one hit.
- [ ] A query is classified as a phone query when its digits, after
      normalization, are at least 4 and it contains no letters; otherwise it is a
      name query. A query may be run as both when ambiguous.
- [ ] A phone query of exactly 8 digits is an equality match on the normalized
      column. Fewer digits is a prefix match. This distinction exists so the
      common case is index-only.
- [ ] A name query is a case-insensitive substring match against the person's
      name, and against a contact person's name.
- [ ] Every result carries: `kind` (spejder, senior, gøgler, friend, crew,
      patruljekontakt, klankontakt), display name, the phone that matched and which
      role it played, the team or section, and a route the SPA can link to.
- [ ] Every result states the person's current standing: the member lifecycle
      status for a spejder, the team's signup status otherwise, and whether they
      have been removed from their roster.
- [ ] A person removed from a roster is retained in the projection behind a
      `deleted` flag, not deleted from it, and is returned by search marked as
      such. Retention is what makes them findable; the flag is what stops them
      being mistaken for current.
- [ ] Search runs against the active year (the SPA's `X-YearSlug`). This is the
      primary and default behaviour.
- [ ] Previous years are searched **only** when the operator asks, through a
      control that is visibly off until they do. The server must not widen the year
      scope on its own — not on an empty result, not on an exact phone match, not
      ever. An operator who has not asked for history must be able to trust that
      what they are reading is this year.
- [ ] A cross-year result states which year it belongs to, on every row, and is
      visually distinct from a current-year result.
- [ ] Results are capped (default 50) and the response states whether the cap was
      hit.
- [ ] A `GET /api/search/person` endpoint serves the above, with OpenAPI
      annotations.
- [ ] A search box reachable from anywhere in HQ, plus a results view.
- [ ] Results are live: a person edited, added or removed while results are on
      screen is reflected without a manual refresh.
- [ ] The projection is added to the `projections` slice in `cmd/api/main.go`, so
      it emits live signals through `live.NotifyAll`.

### Non-Functional

- **Performance.** A phone lookup must be an index seek, not a scan — that is the
  whole reason this PRD builds a projection rather than a `UNION` over the six
  tables. Target: server time under 50 ms at p99 for a phone query.
- **Privacy.** This endpoint returns contact details for minors, aggregated across
  the whole event, which no existing endpoint does. It must sit behind the same
  authentication as every other `/api` route — **and task 181 found that this is a
  weaker guarantee than it sounds.** `app.authenticate` does not authenticate: it
  attaches an anonymous user and calls the next handler, because authentication lives
  in an external service. So "the same as every other route" is true and may not be
  enough. Pre-existing, not caused by this PRD, but this PRD raises the stake — one
  request now returns fifty named minors with phone numbers. **Task 188 owns
  establishing what actually guards `/api` and recording the decision here.**
  Results deliberately do **not** include address, birthday or notes: those stay on
  the detail pages the results link to, so search is a way to *find* a person, not a
  way to bulk-export the population.
- **No new PII at rest.** The projection stores name, phone and team — all already
  stored in the six source tables. It duplicates; it does not widen.
- **i18n.** UI text in Danish, per convention. Matching must handle æ/ø/å and be
  case-insensitive for them.
- **Accessibility.** The search box is reachable and operable by keyboard alone,
  results are a navigable list, and the result count is announced.

## 7. UX / UI Notes

- **Entry point.** A search field in `vue/src/components/Navigation.vue`, present
  on every page, with a keyboard shortcut. It is in the chrome and not on a page
  of its own because the operator's hands are on the phone, not on the navigation.
- **Results.** A new `vue/src/views/SearchView.vue` at route `/search?q=…`. The
  query lives in the URL so a search is linkable and survives reload.
- **Result rows** are grouped by `kind` with Danish headings (Spejdere, Seniorer,
  Gøglere, Crew, Kontaktpersoner). Each row shows name, the matched phone, the
  team/section and the person's current status, and the whole row is the link.
- **Status is shown on every row, not only on the unusual ones.** A badge that
  appears only when something is wrong makes its absence carry meaning the
  operator has to know to read; showing "racing" as plainly as "udmeldt" means the
  row can always be read at face value. ~~Use the existing severity vocabulary in
  `composables/severity.ts`~~ — **corrected in task 183:** `severity.ts` is only the
  Grøn/Gul/Rød *priority* scale and holds no person statuses, so following it would
  have minted a second set of Danish words for states HQ already badges elsewhere.
  The member-status vocabulary is inherited from PRD 006 (extracted into
  `composables/memberStatus.ts`); `severity.ts` is used only for the *team* status
  axis, which genuinely had no vocabulary.
- **Colour alone does not carry the distinction search needs.** The inherited member
  badge colours were designed for race-night screens, so *in our care*
  (`waiting`/`transit`/`sheltered`) spans warn+danger and *somebody else's charge*
  (`released`/`reunited`) spans secondary+info — the three-way grouping an operator
  actually reasons about is not expressible in them. Task 183 deliberately did not
  fork the palette (drift between two copies is worse) and instead puts the
  unambiguous long form on hover. If that proves insufficient in use, the fix is to
  change PRD 006's vocabulary in one place, not to add a second one here.
- **Team statuses are prefixed "Holdet:"** — e.g. "Holdet: mangler betaling" for
  `PAY`. This resolves §11's wording question, and identifies the real problem with
  it: not that `PAY` is English, but that a fact about the *team* sits on a person's
  row and needs saying so. `OUT` renders "ude", deliberately not "udgået", which
  `sheltered` already owns.
- **Six headings, not five:** Spejdere, Seniorer, Gøglere, **Friends**, Crew,
  Kontaktpersoner. `friend` is its own entity token and its own population.
- **The year control is off by default and states what it will do** — a checkbox
  reading **"Søg også i andre år"**, not "tidligere år" and not a year dropdown
  pre-filled with the current year. "Andre" rather than "tidligere" because the data
  is not only historical: the dev stream carries fixtures under year 9999, and
  nothing stops a future edition existing before the current one closes. The
  dropdown was rejected because it invites changing the year without noticing, and
  every list in HQ is already year-scoped by the global year selector. Cross-year
  rows carry their year and are grouped after the current year's, never interleaved.
  The API parameter is `includeOtherYears` for the same reason.
- **The matched phone is shown, and annotated** when it is not the person's own:
  "forælders nummer" for `phoneParent`, "kontaktperson" for a team contact. An
  operator who is about to speak to someone must not be misled about who will
  answer.
- **Loading.** `pending` from `useLiveResource` wired to the list's loading state,
  and no separate spinner — a repeated query renders from cache and must not
  flash.

  Implementation note from task 182, because it fails *silently*: a dynamic-key
  resource must be held in a **`shallowRef`**, not a `ref`. A deep ref wraps the
  resource in a reactive proxy which unwraps the refs inside it, so `pending.value`
  and `data.value` read as `undefined` — a loading state that never appears and data
  that never arrives, with no error anywhere. Worth knowing for any other view that
  re-keys a resource.
- **Empty states are three, not two:** nothing typed yet, too short to search
  (under the minimum), and searched-with-no-match.
- The view is read-only, so it needs none of the dirty-state deferral that
  `KlanListView` and `KortView` carry.

## 8. Technical Considerations

### Data / storage

New package `go/nathejk/table/searchperson/`, following the established shape
(`table.go`, `consumer.go`, `query.go`, `table.sql`, `filter.go`), with one table:

```
search_person
  kind                   VARCHAR   -- spejder|senior|gøgler|friend|crew|patruljekontakt|klankontakt
  id                     VARCHAR   -- memberId, userId, or teamId for a contact person
  year                   VARCHAR
  teamId                 VARCHAR
  name                   VARCHAR
  phone                  VARCHAR   -- as entered
  phoneNormalized        VARCHAR   -- digits only
  phoneParent            VARCHAR
  phoneParentNormalized  VARCHAR
  email                  VARCHAR
  deleted                TINYINT(1) NOT NULL DEFAULT 0  -- removed from roster; retained and findable
  PRIMARY KEY (kind, id, year)
  KEY idx_phone        (year, phoneNormalized)
  KEY idx_phone_parent (year, phoneParentNormalized)
  KEY idx_name         (year, name)
```

`kind` is part of the primary key because the id spaces are not one space:
`memberId`, `userId` and — for contact persons, who have no id of their own —
`teamId`. Without `kind` in the key, a team's contact person and a member could
collide.

The `name` index serves ordering and the year scope, but **not** a leading-wildcard
`LIKE`, which cannot use an index at all. That is accepted: name search scans, and
at Nathejk's row counts (low thousands) that is fine. The index exists for phone,
which is the case this PRD is actually for.

`CREATE TABLE IF NOT EXISTS` never alters an existing table, so a column added to
`table.sql` later will silently not appear in dev. Follow the `ensureColumn`
pattern already in `patrulje/table.go` from the start rather than after being bitten.

### BFF (Go)

**The consumer derives every field from the event payload and reads no other
projection's table.** This matters: a projection that read `spejder` to build its
row would become dependent on the order in which two consumers happen to see the
same event, which is not guaranteed. Everything needed is in the events the
existing six tables already consume:

| Subjects | Produces |
|---|---|
| `NATHEJK.*.spejder.*.{updated,deleted,reassigned}` | spejder rows |
| `NATHEJK.*.senior.*.{updated,deleted}` | senior rows |
| `NATHEJK.*.{gøgler,friend}.*.{signedup,updated}` | personnel rows |
| `NATHEJK.*.crewmember.*.{registered,updated}`, `NATHEJK.*.crew.*.signedup` | crew rows |
| `NATHEJK:*.patrulje.*.{signedup,updated}` | patruljekontakt rows |
| `NATHEJK:*.*.*.signedup` | klankontakt / crew contact rows |

**There is no `bandit` kind.** Task 175 established this: `bandit.*.armNumber.assigned` is
the only bandit-entity event, it carries neither a name nor a phone number, and both the
senior and personnel projections consume it merely to stamp an arm number onto a row that
already exists. The people themselves arrive as seniors. An earlier draft of this PRD listed
`bandit` as a kind, which would have produced a table of empty rows and advertised a live
dependency that never fires.

**Crew and personnel rows carry no `teamId`**, because their events carry no team: a crew
member's section arrives on `crewmember.*.section.assigned`, which is identity-free and so
not subscribed to here. Their context therefore comes from the read-time join in task 179
(to `personnel` and `crewmember`), exactly as team names do — not from a column here.

Two traps in that list, both already visible in the existing consumers:

- **The subject separator is normalized in subscriptions but not in `Match`.**
  `subject.FromStr` replaces the *first* `:` with `.`, so subscribing with
  `NATHEJK:*.patrulje.*.updated` and `NATHEJK.*.patrulje.*.updated` is the same
  subscription — the inconsistency across the existing consumers is cosmetic.
  `Subject.Match`, however, takes a raw pattern and escapes `.` without touching
  `:`, so a **`Match("NATHEJK:...")` can never match anything**: by then the
  subject holds a dot. Every handler in the codebase spells its `Match` patterns
  with dots for this reason. `Match` is also case-insensitive, which is why the
  existing handlers get away with lowercase `nathejk.*`.
- **`signup` subscribes with a wildcard in the entity position**
  (`NATHEJK:*.*.*.signedup`), which is what makes `live.EntitySet.Exhaustive`
  false. `searchperson` deliberately does **not** copy that (task 176): it names
  `patrulje` and `klan` explicitly, because we already know which team types have a
  contact person, and an entity wildcard would cost the SPA its ability to warn
  about a dependency nothing can satisfy.
- **`signup.phone` and `signup.phonePending` are one number in two states**, not two
  numbers — verification runs `SET phone = phonePending`. An earlier draft of this
  PRD said to index both; one indexed pair finds an unverified contact and a
  verified one alike.
- **`klan.updated` carries a contact person that no existing table keeps.** klan has
  no contact columns, and signup retains only what the *signup* event carried, so a
  corrected klan contact exists nowhere in the read model today. `search_person` is
  the first place it lands — worth knowing when reading the joins below.
- **The contact fields on `NathejkTeamUpdated` are prefixed** (`ContactName`, not
  `Name`). The unprefixed ones describe the team, so reading the wrong pair files
  the patrol's name as a human being.

**`spejder.deleted` and `senior.deleted` set `deleted = 1`; they do not delete the
row.** A hard delete would make search answer "ingen match" for somebody the event
has actually met, which §5 rules out. The flag is not a soft-delete for tidiness —
it is the field the UI renders as "udmeldt", so it must be projected as a fact
about the person rather than treated as absence. A re-added member sets it back to
0, so the upsert must write the column explicitly rather than leaving it alone.

Team and section *names* are **not** denormalized into the row; the query joins
`patrulje`/`klan`/`section` for display. Joining at read time is safe (the tables
are consistent by then) where denormalizing at write time would not be.

Join teams on `teamId` alone, not on `teamId` *and* `year`: patrulje's own projection
writes its `year` from `msg.Time().Year()` while `search_person` takes the subject's
year token. The two agree in practice — the publisher derives both from the same
clock — but `teamId` is unique, so there is no reason to depend on that.

**Current status is likewise a read-time join, not a projected column** — to
`spejderstatus` for a spejder's lifecycle status, and to
`patrulje`/`klan`/`personnel.signupStatus` otherwise. Same reasoning as the
team-name join, with a second payoff: the search consumer needs none of the status
subjects, so it stays subscribed to the six identity-bearing event families and its
replay and signal surface stay narrow. Projecting status would have meant
re-implementing PRD 006's state machine in a second place, where it could disagree
with the first.

A spejder with no `spejderstatus` row is an ordinary case, not an error: statuses
begin at signup-time `registered`/`seated`, and older years predate the table
entirely. Render an absent status as unknown rather than inventing one — a LEFT
JOIN, and no `COALESCE` to a status that would be a lie.

Normalization is **two** steps, and the second is not optional.
`types.PhoneNumber.Normalize()` from shared-go — the same function the SMS gateway uses —
reduces a number to its digits, but it keeps *every* digit, so `+45 12 34 56 78` becomes
`4512345678` while `12 34 56 78` becomes `12345678`. On its own it therefore does **not**
unify the formats actually present in the source tables, which is the entire problem this
projection exists to solve. (shared-go offers no help: its `IsValid()` calls an 8-digit
number valid and so calls the `+45` form invalid, and `InternationalNumber()` concatenates
an empty country code onto it.)

So search stores a **national form**: digits, minus a leading `00`, minus a leading `45`
*only when the result is 10 digits long*. `45` is a real Danish prefix — `45123456` is
somebody's actual number — so stripping it from an 8-digit value would silently make every
45-prefixed subscriber in the event unfindable. Non-Danish numbers are left as dialled
rather than mangled.

The query side must apply the identical function to the operator's input (task 180), or the
two halves will disagree and the disagreement will look like "this person is not in the
system".

### Frontend (Vue 3 / TS)

- `vue/src/views/SearchView.vue`, lazy-loaded route `/search` in `router/`.
- Search field in `components/Navigation.vue`.
- Data via `useLiveResource('search:' + query, …, { dependsOn: [...] })`.
- **Debounce before the key changes, not after.** Each distinct key is a cache
  entry that lives at module level, so feeding raw keystrokes into the key would
  leave an entry per prefix of everything ever typed. Debounce, then set the key.
- **Minimum query length** (3 characters / 4 digits) enforced client-side, so the
  first keystroke does not ask the server for every person in the event.
- `dependsOn` names entity **types**, since a new match is a row whose id was
  never seen. The tokens are the *event subject's* entity, not the projection's
  name: `spejder`, `senior`, `patrulje`, `klan`, `gøgler`, `friend`,
  `crewmember`, `crew`. There is no `personnel` token — that exact mistake is
  documented in `go/internal/live/entities.go`. Verify against the advertised set
  and the SPA's dev-console warning rather than against this list.

### API endpoints

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/search/person?q=&includePreviousYears=` | Person search |

`includePreviousYears` (default false) rather than `year=` or `allYears=`: the
parameter names the operator's *decision*, and its default is the primary
behaviour, so the year scope cannot widen through an omitted or malformed
parameter. A caller that forgets it gets this year.

`/api/search/person` rather than `/api/search`, so a later search over poster or
kort is a sibling and not a breaking change to this one's response shape.

**OpenAPI annotations are required** and are accounted for: the handler carries
the `// @Summary` / `// @Description` / `// @Param` / `// @Success` block in the
style of the existing handlers (see `cmd/api/reassign.go`). The description must
state the cap, the year scoping and the phone-vs-name classification, because all
three change what an empty result means — and must document that results include
departed and removed people, since a caller that assumed otherwise would present
them as current.

### Dependencies & risks

- **No new external dependencies.** No new events, no new commands, no write path,
  no third-party service.
- **Replay cost.** One more consumer replaying the same event families on every
  API restart. Marginal, but not free.
- **Duplication.** The row is a copy, so a bug in this consumer shows as search
  disagreeing with the page it links to. Mitigated by the consumer being
  append-mostly and by tests asserting a round trip per source.
- **Signal noise during the race** is now largely a feature rather than a risk. A
  results view depending on `spejder` is invalidated by *any* projection handling a
  `spejder` subject, including `spejderstatus`' race-time churn — and since status
  is joined at read time and shown on every row, that churn is exactly what keeps a
  displayed status true. Revalidating on a status change is the point. It is still
  chattier than the strict minimum; acceptable at this scale, and the fix if it
  ever bites is to stop revalidating a query the operator has moved on from, not to
  weaken the dependency.
- **Retention makes the population monotonic.** Rows are now only ever added, so
  `search_person` grows across years where the source tables shrink. Harmless at
  this scale, and worth stating: nothing prunes it, by design.
- **Option A was considered and rejected for now:** a query-time `UNION ALL` over
  the six tables with normalization in SQL. Less machinery and no duplication, but
  no index can serve `REPLACE(REPLACE(phone,' ',''),'+45','')`, so every lookup
  scans six tables. At today's volumes it would work; it is rejected because the
  projection is also what makes ranking and prefix matching possible later, and
  because search is on the critical path of an inbound emergency call, where a
  predictable index seek is worth the duplication. Keep the querier behind an
  interface so the two remain interchangeable.

  **Measured afterwards (task 186), and the latency argument above did not hold up.**
  The index does work — `index_merge` across both phone columns, 5 rows examined out
  of 4,602, p99 5.2ms over HTTP against 50ms target, and flat at 0.36ms p99 when the
  table is grown 16x to 73k rows. But a name *scan* at that same 73k rows costs only
  4.7ms, so a scanning implementation would have met the target comfortably too.
  Latency alone did not justify this projection. What justifies it is the rest:
  contact persons becoming findable at all, normalisation done once at write time
  rather than in an unindexable expression per query, one place to query instead of
  six, and the klan contact corrections that turned out to be kept nowhere else.
  Scope the next search feature on those reasons rather than on speed.

## 9. Success Metrics

- An unknown inbound number is resolved to a person in **one** interaction, where
  today it is not resolvable at all.
- p99 server time for an 8-digit phone query under 50 ms, measured on the
  production row count.
- Coverage: for every one of the six sources, a known person in it is findable by
  name and by each of their stored numbers. Asserted by tests, not by sampling.
- A person who withdrew or was removed is still found, and their row states their
  standing. Also asserted by tests — this is the requirement most likely to be
  quietly broken by a later "clean up the deleted rows" change.
- No search reaches beyond the active year without `includePreviousYears`.
  Asserted by a test, because it is a privacy-adjacent default that would be easy
  to erode.
- Zero live-dependency warnings in the dev console for the search view.
- Qualitative: the nødtelefon crew stops opening lists to find people.

## 10. Rollout / Task Breakdown

Sequenced so the projection is proven before anything depends on it. No feature
flag: the endpoint is additive and the UI entry point is the last step, so an
incomplete feature is invisible rather than broken.

1. **Projection first, one source at a time.** `spejder` alone, with tests,
   establishes the table, the normalization and the deletion handling. The
   remaining five are then mechanical.
2. **Endpoint** against the finished projection.
3. **UI** last.

Proposed tasks for `roadmap/tasks/open/` (created 2026-09-14 on approval):

- [x] 174 — `search_person` table and projection skeleton, spejder source only
- [x] 175 — Add senior, personnel and crewmember sources to `search_person`
- [x] 176 — Add contact-person sources (patrulje columns, signup) to `search_person`
- [x] 177 — Retain removed members behind a `deleted` flag, with round-trip tests
- [x] 178 — Wire `searchperson` into the `projections` slice and confirm live tokens
- [x] 179 — Join current status (spejderstatus, signupStatus) into search results
- [x] 180 — Phone/name query classification and matching, with table-driven tests
- [x] 181 — `GET /api/search/person` handler with OpenAPI annotations
- [x] 182 — `SearchView.vue` and `/search` route, live via `useLiveResource`
- [x] 183 — Status badges on result rows
- [x] 184 — Opt-in other-years search, off by default, with year-labelled rows
- [x] 185 — Search box and keyboard shortcut in `Navigation.vue`
- [x] 186 — Verify p99 phone-query latency on production-sized data
- [ ] 187 — Find people whose number shares a field with another number (from task 180)
- [ ] 188 — Confirm who can reach `/api`, now that it returns minors' contact details (from task 181)

Note the sequencing differs slightly from the numbering: 178 (wiring) lands before
179 (status join), because the join is easier to verify against a projection that
is already live.

**This PRD stays in `doing/` until 187 and 188 close.** The feature works and is
serving — 4,602 people indexed, an 8-digit lookup at 5.2ms p99 — but 187 is a
correctness gap in the headline use case (a guardian field holding two numbers makes
neither parent findable, 33 of 1459 rows) and 188 is an open question about who can
reach an endpoint that returns minors' contact details. Neither is a reason to
unship; both are reasons not to call it done.

## 11. Open Questions

- **One row per person, or a `search_person_phone` pair table?** The proposed
  schema hard-codes two phone roles in column names, which is informative but does
  not extend to a third number. A `(phoneNormalized, kind, id, role)` table would
  make any number findable through one index. ~~Collapse later if a third number
  appears.~~ **A third number has appeared** — task 180's verification found 33 of
  1459 guardian fields holding two numbers as free text
  (`mor 22 79 01 52 eller Far 22110715`), which normalize to 16 digits and make
  *neither* parent findable. Task 187 owns resolving this, and the pair table is the
  likely answer.
- **A data-quality report for the organisers.** The same measurement found numbers
  that are simply damaged: `2128151q`, `5O401639` (letter O for zero), 7-digit
  numbers, `00000000`, `112`, and one value that looks like a number pasted twice.
  Search deliberately does **not** guess at these — in an emergency a confident
  wrong match is worse than no match — so they stay unfindable by number. Somebody
  should still be told they are wrong in the register, and search is now the thing
  that can find them.
- **Duplicate registrations are common enough to shape the UI.** Eight distinct
  members of one 2026 patrol are all registered as "Rakel A. Koch" — the contact
  person's own name, filled in for every seat. A name search returns eight rows that
  differ only by id. Nothing is wrong with the query; it means result rows need
  whatever distinguishing context exists, and it is a further argument for the cap.
- **Is the 4-digit minimum for a phone prefix search right?** Too low and it
  returns noise; too high and a badly-heard number is unsearchable.
- **Should name search cover team and group names too?** An operator given
  "Ørnene fra Hvidovre" has a name that is not a person's. Probably a separate
  team search rather than muddying this one's result shape.
- **Should a zero-result current-year search *hint* that previous years exist?**
  Widening automatically is settled (§6: it must not). But a count — "ingen match i
  2026 · 2 match i tidligere år" — would let the operator make the deliberate
  choice knowing there is something to find. It costs a second query on the empty
  path only, and it is the one thing the resolved decision leaves on the table.
- **How should a status be worded for a non-spejder?** ~~The lifecycle statuses read
  naturally in Danish; a klan member's `signupStatus` of `PAY` does not…~~
  **Resolved in task 183** with a "Holdet:" prefix — the problem was not the English
  but that a team fact sits on a person's row. See §7.
- **Should the nav search box be hidden on `/search`?** Task 185 kept it, mirroring
  `?q=`, so a shared link does not show an empty box above full results — at the cost
  of two search inputs on that one page. Hiding it would break "present on every
  page" and leave the keyboard shortcut with nothing to focus. Needs an operator's
  opinion rather than an engineer's.
- **A cross-year result links into a year-scoped detail page** that may not hold the
  record. Task 182 left this: the link is correct, the destination may shrug. Either
  the detail pages need to accept a year, or cross-year rows should not link.
- **Fuzzy matching.** Explicitly out of scope, and the case for it is real: names
  are taken down by ear over a phone. Revisit once we can see what operators
  actually type and fail to find.
- **Should search be audited?** It returns minors' contact details across the
  whole event; a log of who searched for what may be warranted. Note this is
  currently **unanswerable**: `requestctx.User` is always anonymous (see task 188),
  so there is no identity to log. The two questions are one question.
