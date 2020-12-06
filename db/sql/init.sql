DROP TABLE
IF EXISTS
    course,
    enrollment,
    exam,
    instructor,
    labs_discussions,
    lectures,
    prerequisites;


CREATE TABLE course (
    crn        INTEGER,
    subject    VARCHAR(4),
    course_num INTEGER,
    type       VARCHAR(4),

    auto_updated INTEGER DEFAULT 0,
    PRIMARY KEY(crn)
);

CREATE TABLE exam (
    crn        INTEGER,
    date       DATE,
    start_time TIME,
    end_time   TIME,
    PRIMARY KEY(crn)
);

CREATE TABLE enrollment (
    crn         INTEGER,
    description TEXT,
    capacity    INTEGER,
    enrolled    INTEGER,
    remaining   INTEGER,

    auto_updated INTEGER DEFAULT 0
);

CREATE TABLE instructor (
    id   INTEGER,
    name VARCHAR(64),

    PRIMARY KEY(id)
);

CREATE TABLE lectures (
    crn        INTEGER,
    course_num INTEGER,
    Title      VARCHAR(1024),
    units      INTEGER,
    activity   VARCHAR(4),
    days       TEXT DEFAULT '',
    start_time TIME,
    end_time   TIME,
    start_date DATE,
    end_date   DATE,
    instructor_id INTEGER,

    auto_updated INTEGER DEFAULT 0,
    FOREIGN KEY (instructor_id) REFERENCES instructor (id)
);

-- TODO this table name is bad
CREATE TABLE Labs_Discussions (
    crn        INTEGER,
    course_crn INTEGER,
    course_num  INT,
    section    VARCHAR(16),
    Title      VARCHAR(1024),
    units      INTEGER,
    activity   VARCHAR(4),
    days       TEXT DEFAULT '',
    start_time TIME,
    end_time   TIME,
    building_room TEXT,
    instructor_id INT,

    auto_updated INTEGER DEFAULT 0,
    FOREIGN KEY (instructor_id) REFERENCES instructor (id)
);

CREATE TABLE prerequisites (
    course_crn INTEGER,
    prereq_crn INTEGER
);
