DROP TABLE
IF EXISTS
    course,
    enrollment,
    exam,
    instructor,
    aux,
    lectures,
    prerequisites;

CREATE TABLE instructor (
    id   INTEGER,
    name VARCHAR(64),

    auto_updated INT,
    PRIMARY KEY(id)
);

CREATE TABLE course (
    crn        INTEGER NOT NULL,
    subject    VARCHAR(4),
    course_num INTEGER,
    type       VARCHAR(4),
    title      VARCHAR(1024),

    auto_updated INTEGER DEFAULT 0,
    PRIMARY KEY(crn)
);

-- TODO merge this table with course
CREATE TABLE enrollment (
    crn         INTEGER NOT NULL,
    description TEXT,
    capacity    INTEGER,
    enrolled    INTEGER,
    remaining   INTEGER,

    auto_updated INTEGER DEFAULT 0,
    PRIMARY KEY (crn)
);


CREATE TABLE lectures (
    crn           INTEGER NOT NULL,
    units         INTEGER,
    days          TEXT DEFAULT '',
    start_time    TIME,
    end_time      TIME,
    start_date    DATE,
    end_date      DATE,
    instructor_id INTEGER,

    auto_updated INTEGER DEFAULT 0,

    PRIMARY KEY (crn),
    FOREIGN KEY (instructor_id) REFERENCES instructor(id),
    FOREIGN KEY (crn)           REFERENCES course(crn)
);

-- Auxiliary course material
--
-- Includes types
--   labs => LAB,
--   discussions => DISC,
--   seminars => SEM,
--   studio => STDO,
--   field work => FLDW,
--   study group => INI,
CREATE TABLE aux (
    crn           INTEGER NOT NULL,
    course_crn    INTEGER,
    section       VARCHAR(16),
    units         INTEGER,
    days          TEXT DEFAULT '',
    start_time    TIME,
    end_time      TIME,
    building_room TEXT,
    instructor_id INT,

    auto_updated INTEGER DEFAULT 0,

    PRIMARY KEY (crn),
    FOREIGN KEY (instructor_id) REFERENCES instructor (id),
    FOREIGN KEY (crn)           REFERENCES course(crn)
);

CREATE TABLE exam (
    crn        INTEGER NOT NULL,
    date       DATE,
    start_time TIME,
    end_time   TIME,

    PRIMARY KEY (crn)
);

CREATE TABLE prerequisites (
    course_crn INTEGER,
    prereq_crn INTEGER
);

CREATE VIEW counts AS
  SELECT
        'course' AS name,
        count(*)
    FROM course
   UNION
  SELECT
        'lecture' AS name,
        count(*)
    FROM lectures
   UNION
  SELECT
        'aux' AS name,
        COUNT(*)
    FROM aux
   UNION
  SELECT
        'enrollment' AS name,
        count(*)
    FROM enrollment
   UNION
  SELECT 'instructor' AS name, count(*)
    FROM instructor
   UNION
  SELECT 'exam' AS name, count(*) from exam
   UNION
  SELECT 'prerequisites' AS name, count(*)
    FROM prerequisites;

CREATE VIEW auto_updated AS
SELECT
    c.crn,
    c.subject,
    c.type,
    c.course_num,

    c.auto_updated as course_updated,
    l.auto_updated as lecture_updated,
    e.auto_updated as enrollment_updated
FROM
    course c,
    enrollment e,
    lectures l
WHERE
    c.crn = e.crn AND
    c.crn = l.crn AND
    (
        c.auto_updated != 0 OR
        l.auto_updated != 0 OR
        e.auto_updated != 0
    );
