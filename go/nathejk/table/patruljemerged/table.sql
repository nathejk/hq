CREATE TABLE IF NOT EXISTS patruljemerged (
    teamId VARCHAR(99) NOT NULL,
    parentTeamId VARCHAR(99) NOT NULL,
    PRIMARY KEY (teamId, parentTeamId)
);
