CREATE TABLE IF NOT EXISTS lok (
    lokId VARCHAR(99),
    year VARCHAR(99),
    name VARCHAR(99) NOT NULL,
    sortOrder INT NOT NULL DEFAULT 0,
    userIds VARCHAR(999) NOT NULL,
    teamIds VARCHAR(999) NOT NULL,
    KEY year_sortOrder (year, sortOrder),
    PRIMARY KEY(lokId)
);
