-- Prose notes about a scout: what was agreed with a guardian, what was said on the phone, what
-- the next shift needs to know (PRD 008).
--
-- Its own table, in its own package, for two reasons that both matter:
--
--   1. Not on spejderstatus. That package owns status and team membership and is queued for
--      lifting to shared-go verbatim (task 083); a notes table in it makes the lift a rewrite.
--   2. Not on sos. Notes are not case-scoped — the shelter has no case, deliberately (PRD 007),
--      so a note that required one could not be written at all.
--
-- # Current text here, history in the stream
--
-- An edit UPDATEs the row. Every version stays in JetStream, which is the append-only record, so
-- showing an edit history later is a UI decision rather than a migration (PRD 008 §8). That is
-- simpler than sos_activity's refActivityId chaining and loses nothing that has been asked for.
--
-- Keyed by noteId rather than by (member, seq): a note is a thing that can be corrected, and the
-- correction has to find it by id. The member is indexed instead, which is how both reads work —
-- one scout's thread, and the per-scout counts behind the row badges.
CREATE TABLE IF NOT EXISTS spejdernote (
    noteId VARCHAR(99) NOT NULL,
    memberId VARCHAR(99) NOT NULL,
    year VARCHAR(99) NOT NULL,
    -- Capped at 2000 characters by the command, counted in runes. MySQL sizes VARCHAR in
    -- characters, so 2000 here matches the command's limit rather than being a byte count
    -- that would truncate Danish text early.
    note VARCHAR(2000) NOT NULL DEFAULT "",
    -- Empty until HQ has real login (PRD 001 §6). Written regardless, so notes start being
    -- attributed the day accounts exist with no change here — and an unsigned trail on race
    -- day is accepted (PRD 008 §5).
    actorUserId VARCHAR(99) NOT NULL DEFAULT "",
    createdAt DATETIME NOT NULL,
    -- Equal to createdAt until the note is corrected. The UI says "Rettet …" only when they
    -- differ, which is why this is not nullable: a note that has never been edited has a real
    -- updatedAt, it just happens to match.
    updatedAt DATETIME NOT NULL,
    PRIMARY KEY (noteId),
    -- Covers both reads: the thread (ordered by createdAt within a member) and the counts.
    KEY member_created (year, memberId, createdAt)
);
