CREATE TABLE IF NOT EXISTS sos_assignable_section (
    year VARCHAR(99) NOT NULL,
    sectionSlug VARCHAR(99) NOT NULL,
    setAt DATETIME NOT NULL,
    PRIMARY KEY (year, sectionSlug)
);
