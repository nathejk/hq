CREATE TABLE IF NOT EXISTS sos (
    id VARCHAR(99) NOT NULL,
    year VARCHAR(99) NOT NULL,
    headline VARCHAR(199) NOT NULL DEFAULT "",
    description TEXT,
    status VARCHAR(9) NOT NULL DEFAULT "open",
    severity VARCHAR(9) NOT NULL DEFAULT "",
    assigneeSectionSlug VARCHAR(99) NOT NULL DEFAULT "",
    createdAt DATETIME NOT NULL,
    createdBy VARCHAR(99) NOT NULL DEFAULT "",
    lastActivityAt DATETIME NOT NULL,
    deletedAt DATETIME NULL DEFAULT NULL,
    PRIMARY KEY (id),
    KEY year_status_activity (year, status, lastActivityAt)
);

CREATE TABLE IF NOT EXISTS sos_team (
    sosId VARCHAR(99) NOT NULL,
    teamId VARCHAR(99) NOT NULL,
    year VARCHAR(99) NOT NULL,
    createdAt DATETIME NOT NULL,
    PRIMARY KEY (sosId, teamId),
    KEY team (year, teamId)
);

CREATE TABLE IF NOT EXISTS sos_activity (
    seq BIGINT UNSIGNED NOT NULL,
    sosId VARCHAR(99) NOT NULL,
    year VARCHAR(99) NOT NULL,
    type VARCHAR(49) NOT NULL,
    actorUserId VARCHAR(99) NOT NULL DEFAULT "",
    activityId VARCHAR(99) NOT NULL DEFAULT "",
    refActivityId VARCHAR(99) NOT NULL DEFAULT "",
    value TEXT,
    createdAt DATETIME NOT NULL,
    PRIMARY KEY (seq),
    KEY sos_created (sosId, seq)
);
