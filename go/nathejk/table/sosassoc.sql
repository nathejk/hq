CREATE TABLE IF NOT EXISTS sos_team (
    sosId VARCHAR(99) NOT NULL,
    teamId VARCHAR(99) NOT NULL,

    PRIMARY KEY (sosId, teamId)
);
