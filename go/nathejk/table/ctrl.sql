CREATE TABLE IF NOT EXISTS controlpoint (
    controlGroupId VARCHAR(99) NOT NULL,
    controlGroupName VARCHAR(99) NOT NULL DEFAULT "",
    controlIndex TINYINT NOT NULL,
    controlName VARCHAR(99) NOT NULL DEFAULT "",
    openFromUts INT NOT NULL DEFAULT 0,
    openUntilUts INT NOT NULL DEFAULT 0,
    plusMinutes INT NOT NULL DEFAULT 0,
    minusMinutes INT NOT NULL DEFAULT 0,
    PRIMARY KEY (controlGroupId, controlIndex)
);
