-- The point history: every position ever reported, per person (PRD 011).
--
-- Append-only in practice — nothing ever updates a row here. Read only when somebody asks to see
-- a route, which is why it is separate from `track_latest` (table.sql).
--
-- # This is the first table in HQ that grows with wall time
--
-- The ceiling is ~30 s sampling over a ~30-hour race, so ~3,600 points per person per event, and
-- the stream is retained indefinitely. Real coverage will be a fraction of that — nobody records
-- unbroken for 30 hours — but the schema is sized against the ceiling rather than the expectation,
-- because the one participant who did keep their phone awake must not be the one who breaks it.
-- Bounding what HQ actually keeps is a separate decision (task 153).
CREATE TABLE IF NOT EXISTS track_point (
    personId VARCHAR(99) NOT NULL,
    ts BIGINT NOT NULL,

    -- Denormalised from the message rather than joined from track_latest: the role held when *this*
    -- batch was reported, which is not necessarily the role held now.
    personType VARCHAR(99) NOT NULL DEFAULT "",
    year VARCHAR(99) NOT NULL DEFAULT "",

    latitude DECIMAL(9,6) NOT NULL DEFAULT 0,
    longitude DECIMAL(10,7) NOT NULL DEFAULT 0,
    accuracy DECIMAL(10,2) NOT NULL DEFAULT 0,

    -- (personId, ts) is the producer's own definition of a point's identity, so it is the key here
    -- too — deliberately *without* year, which would let one point exist twice under two year
    -- values and defeat the dedupe this key exists to provide.
    --
    -- With INSERT IGNORE this single index does three jobs: it collapses a batch the client
    -- retried, it makes replay-on-boot idempotent for free, and it is the exact index the track
    -- query needs (one person, ordered by time, optionally bounded). No second index is justified
    -- until a query exists that this one cannot serve.
    PRIMARY KEY (personId, ts)
);
