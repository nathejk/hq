CREATE TABLE IF NOT EXISTS spejderstatus (
    id VARCHAR(99) NOT NULL,
    year VARCHAR(99) NOT NULL,
    initialTeamId VARCHAR(99) NOT NULL DEFAULT "",
    currentTeamId VARCHAR(99) NOT NULL DEFAULT "",
    status VARCHAR(99) NOT NULL,
    updatedAt DATETIME NOT NULL,
    PRIMARY KEY (year, id),
    KEY year_team_status (year, currentTeamId, status)
);
