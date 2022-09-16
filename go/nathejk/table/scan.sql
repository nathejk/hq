CREATE TABLE IF NOT EXISTS scan (
    qrId VARCHAR(99) NOT NULL,
    teamId VARCHAR(99) NOT NULL,
    teamNumber VARCHAR(99) NOT NULL,
    scannerId VARCHAR(99) NOT NULL,
    scannerPhone VARCHAR(99) NOT NULL,
    uts INT NOT NULL DEFAULT 0,
    PRIMARY KEY(qrId, uts)
);
