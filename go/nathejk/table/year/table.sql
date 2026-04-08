CREATE TABLE IF NOT EXISTS years (
    slug VARCHAR(99) NOT NULL,
    headline VARCHAR(99) NOT NULL DEFAULT "",
    description TEXT NOT NULL DEFAULT "",
    cityDeparture VARCHAR(99) NOT NULL DEFAULT "",
    cityDestination VARCHAR(99) NOT NULL DEFAULT "",
    dateStart VARCHAR(10) NOT NULL DEFAULT "",
    dateEnd VARCHAR(10) NOT NULL DEFAULT "",
    PRIMARY KEY (slug)
);
