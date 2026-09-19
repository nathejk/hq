CREATE TABLE IF NOT EXISTS checkpersonnel (
    id VARCHAR(99) NOT NULL,
    year VARCHAR(99) NOT NULL,
    checkpointId VARCHAR(99) NOT NULL,
    userId VARCHAR(99) NOT NULL,
    startUts INT NOT NULL DEFAULT 0,
    endUts INT NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    -- Attribution reads this by scanner and moment: a scan counts for a post if the
    -- scanner was on a registered shift there at that instant. Without this index the
    -- join rescans every shift for every scan, which was the single largest cost of
    -- both the post list and the bingo table — 1.4s against 0.2s at 100k scans.
    KEY idx_checkpersonnel_user (userId, startUts, endUts)
);
