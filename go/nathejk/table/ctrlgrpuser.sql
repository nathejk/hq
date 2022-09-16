CREATE TABLE IF NOT EXISTS controlgroup_user (
    controlGroupId VARCHAR(99) NOT NULL,
    controlIndex TINYINT NOT NULL,
    userId VARCHAR(99) NOT NULL,
    startUts INT NOT NULL DEFAULT 0,
    endUts INT NOT NULL DEFAULT 0
);
