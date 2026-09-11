-- Every map sheet ever handed to a team, kept as history rather than a current binding.
--
-- # Why hq keeps this at all
--
-- The physical map ↔ QR code ↔ team binding is made in the skan app, whose `qr`
-- projection stores only the *current* holder (one row per year+code, overwritten on
-- re-bind). That is right for skan — it hands the next sheet to whoever holds the code
-- now — but it cannot answer the question a patrol's own page asks: which maps has *this*
-- team ever been given? When a team is discontinued its remaining scouts, and their
-- sheets, are reassigned to another team, so a sheet legitimately changes hands and the
-- skan row moves with it. To list a team's history we must remember each binding, not
-- just the last.
--
-- So this is a copy of skan's projection widened to history: keyed by (year, code, team)
-- rather than (year, code), one row per team a code was ever bound to. The current holder
-- is derived on read as the binding with the newest lastUts for a code.
--
-- QR codes and teams are identified as the qr.registered event carries them: the code as
-- a per-year integer id (stickers restart at 1 each event and may be reused, so the id is
-- only unique within a year — hence year in the key), the team by its id.
CREATE TABLE IF NOT EXISTS maphandout (
    year VARCHAR(99) NOT NULL,
    qrId VARCHAR(99) NOT NULL,
    teamId VARCHAR(99) NOT NULL,

    -- The team number recorded at registration. Kept for display when the team is not a
    -- patrulje (and therefore not in the patrulje table to look a current number up from).
    teamNumber VARCHAR(99) NOT NULL DEFAULT "",

    -- The kort sheet handed over. "" for a code registered before the sheet was recorded,
    -- which means "unknown sheet" rather than "no sheet" — and must never be erased by a
    -- later empty value, the same rule skan's qr projection follows.
    mapId VARCHAR(99) NOT NULL DEFAULT "",

    -- Who scanned the code to bind it, and when this team first and last held it.
    registeredBy VARCHAR(99) NOT NULL DEFAULT "",
    firstUts INT NOT NULL DEFAULT 0,
    lastUts INT NOT NULL DEFAULT 0,

    PRIMARY KEY (year, qrId, teamId),
    -- The two reads: a team's whole history, and every binding of one code (to find its
    -- current holder).
    KEY year_team (year, teamId),
    KEY year_qr (year, qrId, lastUts)
);
