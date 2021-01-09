CREATE TYPE weekday AS ENUM(
	'sunday',
	'monday',
	'tuesday',
	'wednesday',
	'thursday',
	'friday',
	'saturday'
);

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
    id   BIGINT UNIQUE NOT NULL,
    name VARCHAR(128),
    PRIMARY KEY(id)
);

-- CREATE RULE bump_instructor_id AS ON INSERT
--     TO instructor
--     WHERE NEW.id IN OLD.id

-- TODO
-- * catalog: a full catalog hold common data
--      rename the Course struct to entry so its 'catalog.Entry'
-- * course:  the main, stand-alone course work
-- * subcourse: auxilary course work to go along with a course

CREATE TABLE course (
    id          SERIAL NOT NULL,
    crn         INTEGER UNIQUE NOT NULL, -- unique contraint may not hold in the future
    subject     VARCHAR(4),
    course_num  INTEGER,  -- TODO change to 'num'
    type        VARCHAR(4),
    title       VARCHAR(1024),
    units       INTEGER,
    days        text[],
    description TEXT,
    capacity    INTEGER,
    enrolled    INTEGER,
    remaining   INTEGER,

    updated_at   TIMESTAMPTZ DEFAULT now(),
    year         INT NOT NULL,
    term_id      INT NOT NULL,

    PRIMARY KEY (id, crn)
);

-- TODO: Add a course_id column that points to course(id).
--       Then create a trigger or rule on insert that matches
--       the lecture with a course and gets the course(id) value

CREATE TABLE lectures (
    crn           INTEGER NOT NULL,
    start_time    TIMESTAMPTZ,
    end_time      TIMESTAMPTZ,
    start_date    DATE,
    end_date      DATE,
    instructor_id BIGINT, -- move to catalog
    updated_at   TIMESTAMPTZ DEFAULT now(),

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
    start_time    TIMESTAMPTZ,
    end_time      TIMESTAMPTZ,
    building_room TEXT,
    instructor_id BIGINT, -- move to catalog
    updated_at   TIMESTAMPTZ DEFAULT now() NOT NULL,

    FOREIGN KEY (instructor_id) REFERENCES instructor (id),
    PRIMARY KEY (crn),
    FOREIGN KEY (crn)           REFERENCES course(crn)
);

-- CREATE INDEX aux_course_crn_idx ON aux (course_crn);
-- CREATE INDEX non_zero_aux_course_crn_idx
--           ON aux (course_crn) WHERE course_crn != 0;

CREATE TABLE exam (
    crn        INTEGER NOT NULL,
    date       TIMESTAMPTZ,
    start_time TIMESTAMPTZ,
    end_time   TIMESTAMPTZ,
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
    ts TIMESTAMPTZ DEFAULT now(),
    enrolled INT,
    capacity INT
);

CREATE TABLE users (
    id         SERIAL       NOT NULL,
    name       VARCHAR(255) NOT NULL,
    email      VARCHAR(128) NOT NULL,
    is_admin   BOOLEAN      DEFAULT 'f',
    created_at TIMESTAMP    DEFAULT now(),
    hash       VARCHAR(72)  UNIQUE NOT NULL, -- password hash

    UNIQUE(name, email),
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


     CREATE VIEW schedule_page AS
          SELECT
                c.crn,
                c.subject,
                c.course_num,
                left(c.title, 25) as title,
                c.units,
                c.type,
                c.days,
                l.start_time::time,
                l.end_time::time,
                left(i.name, 40) as instructor,
                c.capacity,
                c.enrolled,
                c.remaining
           FROM course c
LEFT OUTER JOIN (
         SELECT crn, start_time, end_time, instructor_id
           FROM aux
          UNION
         SELECT crn, start_time, end_time, instructor_id
           FROM lectures
       ) l   ON c.crn = l.crn
LEFT OUTER JOIN instructor i ON i.id = l.instructor_id
          WHERE c.year = 2021
       ORDER BY c.subject ASC,
                c.course_num ASC,
                c.type DESC;
