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
    -- are all in the source tables — so an equality match has to be against a
    -- normalized column. The as-entered form is kept because it is what an operator
    -- reads back to a caller.
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

    -- The point of this whole projection: an 8-digit lookup must be an index seek.
    -- Year leads because every search is year-scoped, and searching previous years
    -- is a deliberate act rather than the default.
    KEY idx_search_phone (year, phoneNormalized),
    KEY idx_search_phone_parent (year, phoneParentNormalized),

    -- Serves the year scope, the ordering, and — measured, not assumed — the name match as a
    -- **covering** index: MariaDB seeks on the year prefix and scans the name entries inside
    -- it (`type: ref`, `Using index`) rather than touching the table. So a mid-string name
    -- match is still a scan, just a cheap one over an index rather than over rows.
    --
    -- Measured at 73k rows: 4.7ms for a name scan against 0.27ms for an indexed phone lookup.
    KEY idx_search_name (year, name)
);
