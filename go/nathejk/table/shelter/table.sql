-- Where each sheltered scout physically is, inside Hønsegården.
--
-- Its own table rather than a column on spejderstatus, for three reasons that all point the
-- same way (PRD 007 §8):
--
--   1. spejderstatus owns status and team membership. Which tent somebody is asleep in is a
--      fact about the shelter, not about the lifecycle.
--   2. spejderstatus is queued for lifting to shared-go verbatim (task 083). An hq-specific
--      column in it turns that lift from a file move into a rewrite.
--   3. The vocabulary of placeringer is derived from this table (see DistinctPlacements),
--      which is what lets the zones come into existence at race start with nothing
--      configured.
--
-- A row exists only while the scout is in the shelter's care: handover.completed deletes it.
-- That is deliberate rather than a soft-delete flag. The question this table answers is
-- "where is this child right now", and a released child is not anywhere of ours — what
-- happened to them lives in spejderstatuslog, which is the append-only record and the right
-- place to ask. Keeping released rows here would also poison the placering suggestions with
-- tents nobody is in any more.
CREATE TABLE IF NOT EXISTS shelter (
    id VARCHAR(99) NOT NULL,
    year VARCHAR(99) NOT NULL,
    teamId VARCHAR(99) NOT NULL DEFAULT "",
    -- The placering as the crew typed it, or empty for a scout accepted before anybody had
    -- decided where to put them. Empty is a real state, not a missing value: it is an
    -- arrival recorded but not yet bedded down, and the screen shows it as something still
    -- to do.
    placement VARCHAR(64) NOT NULL DEFAULT "",
    acceptedAt DATETIME NOT NULL,
    placedAt DATETIME NULL,
    PRIMARY KEY (year, id),
    -- Covers the suggestion query, which groups by placering within a year.
    KEY year_placement (year, placement)
);
