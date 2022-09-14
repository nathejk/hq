CREATE TABLE IF NOT EXISTS personnel (
    userId VARCHAR(99) NOT NULL,
    name VARCHAR(99) NOT NULL,
    email VARCHAR(99) NOT NULL,
    phone VARCHAR(99) NOT NULL,
    department VARCHAR(99) NOT NULL,
    hqAccess TINYINT NOT NULL,
    medlemNr VARCHAR(99) NOT NULL,
    corps VARCHAR(99) NOT NULL,
    createdAt VARCHAR(99) NOT NULL,
    updatedAt VARCHAR(99) NOT NULL,
    PRIMARY KEY (userId)
);
