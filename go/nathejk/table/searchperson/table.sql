CREATE TABLE IF NOT EXISTS search_person (
    -- What kind of person this is, as the *event subject's* entity token
    -- (spejder, senior, gøgler, friend, crew) or, for a contact person who is not a
    -- row anywhere else, patruljekontakt / klankontakt.
    --
    -- Part of the primary key because the id spaces do not merge: memberId for
    -- scouts and seniors, userId for personnel and crew, and teamId for a contact
    -- person, who has no id of their own. Without kind in the key, a team's contact
    -- person and a member could collide and silently overwrite each other.
    kind VARCHAR(31) NOT NULL,
    id VARCHAR(99) NOT NULL,
    year VARCHAR(99) NOT NULL,

    teamId VARCHAR(99) NOT NULL DEFAULT "",
    name VARCHAR(199) NOT NULL DEFAULT "",
    email VARCHAR(199) NOT NULL DEFAULT "",

    -- Every number is stored twice: as entered, and as digits only.
    --
    -- The stored form is free text — `+45 12 34 56 78`, `12 34 56 78` and `12345678`
    -- are all in the source tables — so matching has to be against a normalized
    -- column. The as-entered form is kept because it is what an operator reads back
    -- to a caller, and because it carries what the digits cannot:
    -- `mor 22 79 01 52 eller Far 22110715` tells the operator *which parent* the
    -- number they searched for belongs to.
    --
    -- The normalized form deliberately **collapses** such a field into one 16-digit
    -- string rather than trying to split it. Substring matching then finds either
    -- half (task 187), and nothing has to guess which label goes with which number —
    -- the entered text is returned and the operator reads it themselves.
    phone VARCHAR(99) NOT NULL DEFAULT "",
    phoneNormalized VARCHAR(31) NOT NULL DEFAULT "",

    -- A scout's guardian, and the reason searching by an unknown number works at
    -- all: it is very often the parent who rings. Kept in its own columns rather
    -- than as a second row so a hit can say *whose* number matched — an operator
    -- about to speak to someone must not be misled about who will answer.
    phoneParent VARCHAR(99) NOT NULL DEFAULT "",
    phoneParentNormalized VARCHAR(31) NOT NULL DEFAULT "",

    -- Removed from a roster, and deliberately still here.
    --
    -- A hard delete would make search answer "ingen match" for somebody the event
    -- has actually met — the guardian of a scout who left at 02:00 rings at 09:00.
    -- So this is a projected fact about the person, rendered as "udmeldt", not a
    -- tidiness flag. Note it is a different axis from a race withdrawal, which lives
    -- in spejderstatus: "udmeldt" (never started) is not "released" (went home
    -- during the night).
    deleted TINYINT(1) NOT NULL DEFAULT 0,

    updatedAt DATETIME NULL DEFAULT NULL,

    PRIMARY KEY (kind, id, year),

    -- The phone indexes.
    --
    -- **These no longer serve the phone lookup.** Task 187 made phone matching a
    -- substring (`LIKE '%digits%'`) so that a guardian field holding two numbers is
    -- findable by either half, and a leading wildcard across two columns cannot use
    -- an index — EXPLAIN reports `type: ALL`. Measured cost of that decision, on real
    -- data: 4.0ms p99 at 4.6k rows, 17.3ms p99 at 73k, against a 50ms target. So it
    -- grows linearly and stays inside budget for roughly thirty years of Nathejk, but
    -- the headroom is ~3x where an index seek gave ~140x.
    --
    -- Kept anyway, deliberately: the write cost is trivial at these volumes, and they
    -- are exactly what an exact-match fast path would need if the scan ever does get
    -- too slow (try the indexed equality first, fall back to the scan only when it
    -- finds nothing). Dropping them would mean a DROP INDEX migration against
    -- production for no measured gain. Do not read their presence as a claim that
    -- lookups are indexed — they are not.
    KEY idx_search_phone (year, phoneNormalized),
    KEY idx_search_phone_parent (year, phoneParentNormalized),

    -- Serves the year scope, the ordering, and — measured, not assumed — the name match as a
    -- **covering** index: MariaDB seeks on the year prefix and scans the name entries inside
    -- it (`type: ref`, `Using index`) rather than touching the table. So a mid-string name
    -- match is still a scan, just a cheap one over an index rather than over rows.
    --
    -- Measured at 73k rows: 4.7ms for a name scan. Now the same order of magnitude as the
    -- phone lookup above, which stopped being a seek in task 187.
    KEY idx_search_name (year, name)
);
