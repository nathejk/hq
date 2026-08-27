-- Duty windows: when a dispatch unit is available, agreed in advance with the logistics crew.
--
-- Per **unit**, not per person (PRD 009 §6): the unit is what is available or asleep, and a
-- window recorded per driver would have to be intersected with the co-driver's to answer the
-- only question the board asks — "who is on now, and who comes on next".
--
-- Its own table rather than `checkpersonnel`, whose shape it copies. Deliberate: a shift on a
-- post and a shift behind a wheel are different facts, and one table would make "which units are
-- driving now" a query with a checkpoint join in it. The cost is a near-duplicate schema; the
-- benefit is that neither entity's changes can break the other's screen.
CREATE TABLE IF NOT EXISTS dispatch_duty (
    id VARCHAR(99) NOT NULL,
    year VARCHAR(99) NOT NULL,
    sectionSlug VARCHAR(99) NOT NULL,
    startUts BIGINT NOT NULL DEFAULT 0,
    endUts BIGINT NOT NULL DEFAULT 0,
    setBy VARCHAR(99) NOT NULL DEFAULT "",
    setUts BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    -- Covers both questions the board asks: this unit's roster, and who is on at an instant.
    KEY year_section_start (year, sectionSlug, startUts),
    KEY year_start (year, startUts)
);
