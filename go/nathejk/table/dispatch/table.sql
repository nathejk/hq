-- Kørsel: the tasks the cars are asked to do, and the tours they are put into (PRD 009).
--
-- Tasks and tours are one aggregate in one package because a stop is meaningless without
-- its tour and a task's state is driven by its stops. They are five tables because they are
-- five different things, and because the questions the desk asks at 3am — "what is not
-- planned", "what is this car doing", "when will somebody reach Post 2B" — are each one
-- indexed query here.
--
-- Times are unix seconds (`…Uts`) rather than DATETIME, unlike sos and shelter. Deliberate:
-- every number on this screen is arithmetic — waited-for, time-until-deadline, departure
-- plus a per-leg allowance — and doing that arithmetic in seconds avoids a timezone
-- question at every step. `checkpersonnel` already stores shifts this way.

-- One thing that needs moving.
CREATE TABLE IF NOT EXISTS dispatch_task (
    id VARCHAR(99) NOT NULL,
    year VARCHAR(99) NOT NULL,

    -- pickup | transport | collection | delivery. Four kinds because they read
    -- differently on a board and default their places differently — not because their
    -- lifecycles differ.
    kind VARCHAR(19) NOT NULL DEFAULT "",

    -- green | yellow | red — the SOS vocabulary, so a pickup created from a red case can
    -- arrive red (PRD 009 §8). Empty is allowed: most tasks are ordinary.
    priority VARCHAR(9) NOT NULL DEFAULT "",

    description TEXT,

    -- What space it needs, in words ("needs most of the boot"). Not an inventory: PRD 009
    -- §4 refuses to track how many maps exist.
    spaceNeeds VARCHAR(199) NOT NULL DEFAULT "",

    -- Places are type + reference + label rather than a foreign key, because a place may be
    -- free text ("på Slangerupvej ved skovbrynet") and because a checkpoint's name should be
    -- preserved on the task even if the checkpoint is later renamed.
    pickupKind VARCHAR(19) NOT NULL DEFAULT "",
    pickupRefId VARCHAR(99) NOT NULL DEFAULT "",
    pickupLabel VARCHAR(199) NOT NULL DEFAULT "",
    dropoffKind VARCHAR(19) NOT NULL DEFAULT "",
    dropoffRefId VARCHAR(99) NOT NULL DEFAULT "",
    dropoffLabel VARCHAR(199) NOT NULL DEFAULT "",

    -- queued | planned | underway | done | cancelled.
    state VARCHAR(19) NOT NULL DEFAULT "queued",

    -- The waiting clock. Never reset: a task dropped from a tour returns to the queue
    -- having waited since the call, not since the re-plan (PRD 009 §5).
    createdUts BIGINT NOT NULL DEFAULT 0,
    -- Not before / hard deadline. NULL is "no constraint", which is the common case and is
    -- why these are nullable rather than zero-defaulted: 0 would be 1970 and would render as
    -- an overdue deadline on every task without one.
    notBeforeUts BIGINT NULL DEFAULT NULL,
    deadlineUts BIGINT NULL DEFAULT NULL,

    -- People aboard. Recorded separately from `done` because custody changes here, and that
    -- is not when the task finishes (PRD 009 §6).
    pickedUpUts BIGINT NULL DEFAULT NULL,
    doneUts BIGINT NULL DEFAULT NULL,
    cancelledUts BIGINT NULL DEFAULT NULL,
    cancelReason VARCHAR(199) NOT NULL DEFAULT "",

    -- Where a pickup came from, and who is being collected. memberIds is a JSON array: it is
    -- read as a unit ("this stop collects these two"), never joined on, and a side table for
    -- a list of at most a handful of ids that only ever moves together would be a join in
    -- every query for no query it enables.
    sosId VARCHAR(99) NOT NULL DEFAULT "",
    teamId VARCHAR(99) NOT NULL DEFAULT "",
    memberIds TEXT,

    createdBy VARCHAR(99) NOT NULL DEFAULT "",
    lastActivityUts BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    -- The queue: unplanned tasks for a year, oldest first.
    KEY year_state_created (year, state, createdUts),
    -- Deadlines at risk, and the case's own task list.
    KEY year_deadline (year, deadlineUts),
    KEY year_sos (year, sosId)
);

-- One car's run: from A to B with as many stops as it takes.
CREATE TABLE IF NOT EXISTS dispatch_tour (
    id VARCHAR(99) NOT NULL,
    year VARCHAR(99) NOT NULL,

    -- The dispatchable subsection running it — not a vehicle id and not a person. The unit
    -- is who took the job, and it survives a car being swapped mid-night (PRD 009 §8).
    -- Recorded at planning time, so a car later moved to another subsection on the
    -- Organisation page changes who owns future tours, not this one.
    sectionSlug VARCHAR(99) NOT NULL DEFAULT "",

    departureUts BIGINT NULL DEFAULT NULL,
    notes TEXT,

    -- planned | underway | completed | cancelled.
    state VARCHAR(19) NOT NULL DEFAULT "planned",
    createdUts BIGINT NOT NULL DEFAULT 0,
    underwayUts BIGINT NULL DEFAULT NULL,
    completedUts BIGINT NULL DEFAULT NULL,
    cancelledUts BIGINT NULL DEFAULT NULL,
    cancelReason VARCHAR(199) NOT NULL DEFAULT "",

    createdBy VARCHAR(99) NOT NULL DEFAULT "",
    lastActivityUts BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    KEY year_state (year, state, departureUts),
    KEY year_section (year, sectionSlug)
);

-- A place on a tour, in order.
CREATE TABLE IF NOT EXISTS dispatch_stop (
    id VARCHAR(99) NOT NULL,
    tourId VARCHAR(99) NOT NULL,
    year VARCHAR(99) NOT NULL,
    sortOrder INT NOT NULL DEFAULT 0,

    placeKind VARCHAR(19) NOT NULL DEFAULT "",
    placeRefId VARCHAR(99) NOT NULL DEFAULT "",
    placeLabel VARCHAR(199) NOT NULL DEFAULT "",

    -- The planned time, derived from the tour's departure plus a per-leg allowance unless
    -- somebody overrode it. The override flag is stored rather than inferred, because the
    -- screen marks an overridden stop and the stops after it re-derive from it — and because
    -- a derived time that happens to equal the derived time is not the same fact as a
    -- dispatcher saying "no, 22:35".
    plannedUts BIGINT NULL DEFAULT NULL,
    plannedOverride TINYINT(1) NOT NULL DEFAULT 0,

    -- Visited stops are fixed: they cannot be reordered or removed (PRD 009 §6).
    visitedUts BIGINT NULL DEFAULT NULL,
    PRIMARY KEY (id),
    KEY tour_order (tourId, sortOrder)
);

-- What is done at a stop.
--
-- Its own table rather than a JSON column on the stop, which is a deliberate departure from
-- PRD 009 §8's list of tables. A task may occupy *two* stops — where it is loaded and where
-- it is unloaded — so this is a many-to-many, and the task's state is derived by asking
-- "which stops does this task sit on, and have they been visited". With the pairs inside a
-- JSON column that question is an unindexed JSON_CONTAINS scan of every stop of every tour;
-- with a row per pair it is a primary-key lookup. The PRD's own reasoning ("a tourId column
-- on the task would be a lie half the time") is what forces this table to exist.
CREATE TABLE IF NOT EXISTS dispatch_stop_task (
    stopId VARCHAR(99) NOT NULL,
    taskId VARCHAR(99) NOT NULL,
    tourId VARCHAR(99) NOT NULL,
    year VARCHAR(99) NOT NULL,
    -- load | unload | action. A task that moves something has a load and an unload; a pickup
    -- of people has the same two; a task that is simply done somewhere has one `action`.
    role VARCHAR(9) NOT NULL DEFAULT "action",
    PRIMARY KEY (stopId, taskId),
    KEY task (taskId),
    KEY tour (tourId)
);

-- The timeline. A dispatch desk is a log first (PRD 009 §6).
--
-- Keyed by the stream sequence of the event that produced it, exactly as sos_activity is:
-- that gives the log a total order agreeing with the event stream, and makes replay
-- idempotent for free rather than by de-duplication.
CREATE TABLE IF NOT EXISTS dispatch_activity (
    seq BIGINT UNSIGNED NOT NULL,
    year VARCHAR(99) NOT NULL,
    -- Exactly one of these is set. Two columns rather than one polymorphic id, so "this
    -- task's history" and "this tour's history" are both indexed lookups and neither needs
    -- to know the other exists.
    taskId VARCHAR(99) NOT NULL DEFAULT "",
    tourId VARCHAR(99) NOT NULL DEFAULT "",
    type VARCHAR(49) NOT NULL,
    actorUserId VARCHAR(99) NOT NULL DEFAULT "",
    -- The note an operator would read: a cancellation reason, the unit a tour was assigned
    -- to, the members taken aboard. One loose column rather than one per type, because the
    -- set of entry types grows and must not need a schema change to do it.
    value TEXT,
    createdUts BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (seq),
    KEY task_seq (taskId, seq),
    KEY tour_seq (tourId, seq)
);
