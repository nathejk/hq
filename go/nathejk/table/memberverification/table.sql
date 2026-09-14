-- Which phone numbers a member has verified themselves, in the hej app.
--
-- One row per member per year, carrying the numbers the member proved rather than the
-- numbers the register holds. The register's own values stay where they are (spejder);
-- this table answers a different question: "has the member already confirmed this
-- number, so the counter need not ask at check-in?" (hej, PRD 015).
--
-- # Why two columns, and why either may stay empty
--
-- The two numbers are verified by two different mechanisms at two different moments, and
-- the event carries whichever one just happened (see messages.NathejkMemberVerified):
--
--   phone        — the member's own number, proven by the SMS PIN they typed to log in.
--   phoneContact — the emergency contact number the member confirmed or supplied.
--
-- A login event therefore carries `phone` alone. Writing "" over an already-verified
-- contact number would erase a fact the member established, so an absent value never
-- overwrites a present one — each column is only ever written when the event names it.
-- Empty means "not verified yet", which is the answer the counter acts on.
--
-- The timestamps come from the event (verifiedAt), not from delivery, so they survive a
-- replay unchanged. They are kept because "verified before arriving" is a question worth
-- being able to ask, and GREATEST on write makes the pick order-independent: a replay
-- that delivers an older verification after a newer one must not move the number back.
CREATE TABLE IF NOT EXISTS memberverification (
    year VARCHAR(99) NOT NULL,
    memberId VARCHAR(99) NOT NULL,

    -- The member's own verified number, normalized as the event carried it.
    phone VARCHAR(99) NOT NULL DEFAULT "",
    phoneUts INT NOT NULL DEFAULT 0,

    -- The verified emergency contact number (a parent or guardian, for a spejder).
    phoneContact VARCHAR(99) NOT NULL DEFAULT "",
    phoneContactUts INT NOT NULL DEFAULT 0,

    PRIMARY KEY (year, memberId)
);
