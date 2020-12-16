-- DROP TABLE
-- IF EXISTS
--     course,
--     exam,
--     instructor,
--     aux,
--     lectures,
--     prerequisites;

CREATE TABLE term (
    id INT,
    name VARCHAR(6)
);

CREATE TABLE subject (
    code    VARCHAR(4) UNIQUE NOT NULL,
    name    TEXT,
    year    INTEGER NOT NULL,
    term_id INT NOT NULL,
    PRIMARY KEY (code)
);

CREATE TABLE instructor (
    id   INTEGER UNIQUE NOT NULL,
    name VARCHAR(128),
    PRIMARY KEY(id)
);

-- TODO
-- * catalog: a full catalog hold common data
--      rename the Course struct to entry so its 'catalog.Entry'
-- * course:  the main, stand-alone course work
-- * subcourse: auxilary course work to go along with a course

CREATE TABLE course (
    id         SERIAL NOT NULL,
    crn        INTEGER UNIQUE NOT NULL, -- unique contraint may not hold in the future
    subject    VARCHAR(4),
    course_num INTEGER,  -- TODO change to 'num'
    type       VARCHAR(4),
    title      VARCHAR(1024),

    units       INTEGER,
    days        VARCHAR(65) DEFAULT '', -- max is the full name of all weekdays separated by ';'
    description TEXT,
    capacity    INTEGER,
    enrolled    INTEGER,
    remaining   INTEGER,

    updated_at   TIMESTAMP DEFAULT now(),
    auto_updated INTEGER DEFAULT 0,
    year         INT NOT NULL,
    term_id      INT NOT NULL,

    PRIMARY KEY (id, crn)
);

-- TODO: Add a course_id column that points to course(id).
--       Then create a trigger or rule on insert that matches
--       the lecture with a course and gets the course(id) value

CREATE TABLE lectures (
    crn           INTEGER NOT NULL,
    start_time    TIME,
    end_time      TIME,
    start_date    DATE,
    end_date      DATE,
    instructor_id INTEGER, -- move to catalog

    updated_at   TIMESTAMP DEFAULT now(),
    auto_updated INTEGER DEFAULT 0,

    FOREIGN KEY (instructor_id) REFERENCES instructor(id),
    PRIMARY KEY (crn),
    FOREIGN KEY (crn) REFERENCES course(crn)
);

-- CREATE RULE connect_lecture AS ON INSERT TO

-- Auxiliary course material
CREATE TABLE aux (
    crn           INTEGER NOT NULL,
    course_crn    INTEGER,
    section       VARCHAR(16),
    start_time    TIME,
    end_time      TIME,
    building_room TEXT,
    instructor_id INT, -- move to catalog

    updated_at TIMESTAMP DEFAULT now(),
    auto_updated INTEGER DEFAULT 0,

    FOREIGN KEY (instructor_id) REFERENCES instructor (id),
    PRIMARY KEY (crn),
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

CREATE TABLE enrollment (
    crn INTEGER NOT NULL, -- TODO change this to a course id when that is a thing
    year INT NOT NULL,
    term INT NOT NULL,
    ts TIMESTAMP DEFAULT now(),
    enrolled INT,
    capacity INT

    -- FOREIGN KEY (crn) REFERENCES course(crn)
);

CREATE TABLE users (
    id         SERIAL       NOT NULL,
    name       VARCHAR(255) UNIQUE NOT NULL,
    email      VARCHAR(128) UNIQUE NOT NULL,
    is_admin   BOOLEAN      DEFAULT 'f',
    created_at TIMESTAMP    DEFAULT now(),
    hash       VARCHAR(72)  UNIQUE NOT NULL, -- password hash
    PRIMARY KEY(id)
);

-- Triggers and Views

CREATE VIEW counts AS
  SELECT 'course'        AS name, COUNT(*) FROM course
   UNION
  SELECT 'lecture'       AS name, COUNT(*) FROM lectures
   UNION
  SELECT 'aux'           AS name, COUNT(*) FROM aux
   UNION
  SELECT 'instructor'    AS name, COUNT(*) FROM instructor
   UNION
  SELECT 'exam'          AS name, COUNT(*) FROM exam
   UNION
  SELECT 'enrollment'    AS name, COUNT(*) FROM enrollment
   UNION
  SELECT 'enroll_rows'   AS name, COUNT(DISTINCT ts) FROM enrollment -- number of enrollment updates
   UNION
  SELECT 'prerequisites' AS name, COUNT(*) FROM prerequisites;


CREATE VIEW auto_updated AS
SELECT
    c.crn,
    c.subject,
    c.course_num,
    c.type,

    c.auto_updated as course_updated,
    c.updated_at as course_updated_at,
    l.auto_updated as lecture_updated,
    l.updated_at as lecture_updated_at
FROM
    course c,
    lectures l
WHERE
    c.crn = l.crn AND
    (
        c.auto_updated != 0 OR
        l.auto_updated != 0
    );

CREATE VIEW course_small AS
  SELECT id,
         crn,
         subject,
         course_num,
         type,
         units,
         left(title, 30),
         enrolled,
         capacity,
         year,
         term_id
    FROM course;

-- course update times and count newest first
CREATE VIEW course_updates AS
      SELECT updated_at, count(*)
        FROM course
    GROUP BY updated_at
    ORDER BY updated_at DESC;

-- Enrollment data dumps by date newest first
CREATE VIEW enrollment_updates AS
     SELECT ts, count(*)
       FROM enrollment
   GROUP BY ts
   ORDER BY ts DESC;
