CREATE TABLE IF NOT EXISTS patrulje (
    teamId VARCHAR(99) NOT NULL,
    year VARCHAR(99) NOT NULL DEFAULT "",
    teamNumber VARCHAR(99) NOT NULL DEFAULT "",
    name VARCHAR(999) NOT NULL DEFAULT "",
    groupName VARCHAR(999) NOT NULL DEFAULT "",
    korps VARCHAR(9) NOT NULL DEFAULT "",
    liga VARCHAR(99) NOT NULL DEFAULT "",
    memberCount INT NOT NULL DEFAULT 0,
    activeMemberCount INT NOT NULL DEFAULT 0,
    startedUts INT NOT NULL DEFAULT 0,
    contactName VARCHAR(99) NOT NULL DEFAULT "",
    contactPhone VARCHAR(99) NOT NULL DEFAULT "",
    contactEmail VARCHAR(99) NOT NULL DEFAULT "",
    contactRole VARCHAR(99) NOT NULL DEFAULT "",
    signupStatus VARCHAR(9) NOT NULL DEFAULT "",

    -- Operational note shown to banditter and postmandskab ("Info til banditter og
    -- postmandskab"), and how loudly to show it. Authored by HQ, not by the team.
    --
    -- remarkSeverity doubles as the on/off switch: `inactive` means the note is filed but
    -- not in force, which is why it is one field rather than a note plus a boolean that
    -- could disagree with it. "" is an unused note, the state almost every patrol is in.
    remark TEXT NOT NULL DEFAULT "",
    remarkSeverity VARCHAR(20) NOT NULL DEFAULT "",

    PRIMARY KEY (teamId)
);
