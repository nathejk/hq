-- Positions reported by the hej-app: the last known position of each person (PRD 011).
--
-- Two tables for one stream, split by access pattern rather than by entity. This one is read on
-- nearly every page in HQ — it is what puts a position glyph next to a person's name — while
-- `track_point` (point.sql) is read only when somebody asks to see a route. One table serving both
-- would mean either a per-person aggregate on every list render, or the whole history scanned to
-- answer "has this person ever reported?".
--
-- # Coordinates are DECIMAL, not VARCHAR
--
-- `scan` stores latitude/longitude as VARCHAR(99), which is a known wart: it cannot be ordered,
-- bounded or averaged without a cast. Tracks are queried by time and rendered as geometry, so they
-- are stored as numbers. DECIMAL(9,6)/(10,7) is ~11 cm of precision — far finer than any
-- consumer-grade GPS fix, and exact rather than approximate, so a point read back is the point
-- that was written.
--
-- # ts is milliseconds, and it is a key
--
-- The producer sends epoch milliseconds and treats `(personId, ts)` as a point's identity. It is
-- an integer for that reason: an RFC 3339 string can be re-serialised into a different-but-equal
-- form ("…16.954Z" vs "…16.954000Z") and a consumer would see two points where there is one.
--
-- BIGINT, not the INT seconds `scan.uts` uses. The two units meeting is a real trap when tracks
-- and scans share one time axis on a map, so the conversion happens once, at the read boundary
-- that joins them (task 149) — not here, and not per caller.
CREATE TABLE IF NOT EXISTS track_latest (
    -- Either a memberID (spejder, senior) or a crewmemberID (crew, and — sharing the same id
    -- space — gøgler/friend/bandit in `personnel`). Opaque and non-colliding, so this table needs
    -- no discriminator to be unambiguous and the presence endpoint needs no join: a people-list
    -- row already carries the id it should look up.
    personId VARCHAR(99) NOT NULL,

    -- The role the person held when their most recent batch was reported, as sent.
    --
    -- Stored, never derived. Roles change while this stream is kept indefinitely — a spejder
    -- becomes a bandit, crew get reclassified — so a join against today's directory would silently
    -- reinterpret last year's history.
    personType VARCHAR(99) NOT NULL DEFAULT "",

    year VARCHAR(99) NOT NULL DEFAULT "",

    latitude DECIMAL(9,6) NOT NULL DEFAULT 0,
    longitude DECIMAL(10,7) NOT NULL DEFAULT 0,

    -- Radius in metres the browser reported. Kept as sent, including poor fixes: the producer
    -- rejects >100 km as "not a fix" but deliberately passes multi-kilometre cell-tower fixes
    -- through, because a bad position is still the only evidence of where someone was. This column
    -- is what lets the UI mark such a point as low-confidence instead of hiding it.
    accuracy DECIMAL(10,2) NOT NULL DEFAULT 0,

    -- Epoch milliseconds of the most recent point, not of the message. Out-of-order delivery is
    -- normal (batches, retries, replay), so the writer only advances this row when the incoming
    -- point is newer — see consumer.go.
    ts BIGINT NOT NULL DEFAULT 0,

    updatedAt DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (personId),
    -- The presence read: one year's reporting people, which is fetched on nearly every page.
    KEY year_ts (year, ts)
);
