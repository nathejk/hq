CREATE TABLE IF NOT EXISTS scan (
    qrId VARCHAR(99) NOT NULL,
    teamId VARCHAR(99) NOT NULL,
    teamNumber VARCHAR(99) NOT NULL,
    scannerId VARCHAR(99) NOT NULL,
    scannerPhone VARCHAR(99) NOT NULL,
    uts INT NOT NULL DEFAULT 0,
    latitude VARCHAR(99) NOT NULL,
    longitude VARCHAR(99) NOT NULL,
    PRIMARY KEY(qrId, uts)
);
