CREATE TABLE IF NOT EXISTS sos (
    id VARCHAR(99) NOT NULL,
    year VARCHAR(99) NOT NULL,
    headline VARCHAR(99) NOT NULL,
    description VARCHAR(9999) NOT NULL,

    createdAt VARCHAR(99) NOT NULL,
    createdBy VARCHAR(99) NOT NULL,

    status VARCHAR(99) NOT NULL,
    severity VARCHAR(99) NOT NULL DEFAULT "",

    PRIMARY KEY (id)
);
