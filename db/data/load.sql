
\COPY instructor FROM 'data/spring-2021/instructor.csv' DELIMITER ',' CSV;
\COPY exam FROM 'data/spring-2021/exam.csv' DELIMITER ',' CSV;
\COPY course (crn, subject, course_num, type, title, units, days, description, capacity, enrolled, remaining, year, term_id) FROM 'data/spring-2021/course.csv' DELIMITER ',' CSV;
\COPY lectures (crn,start_time,end_time,start_date,end_date,instructor_id) FROM 'data/spring-2021/lecture.csv' DELIMITER ',' CSV;
\COPY aux (crn,course_crn,section,start_time,end_time,building_room,instructor_id) FROM 'data/spring-2021/labs_disc.csv' DELIMITER ',' CSV;

-- copy all the historical data

-- Fall 2020

-- SELECT * INTO tmp FROM instructor LIMIT 0;
-- \COPY instructor FROM 'data/fall-2020/instructor.csv' DELIMITER ',' CSV;
-- INSERT INTO instructor
-- SELECT * FROM tmp;
-- DROP TABLE tmp;

SELECT * INTO tmp FROM course LIMIT 0;
\COPY tmp (crn, subject, course_num, type, title, units, days, description, capacity, enrolled, remaining, year, term_id) FROM 'data/fall-2020/course.csv' DELIMITER ',' CSV;
INSERT INTO
course (crn, subject, course_num, type, title, units, days, description, capacity, enrolled, remaining, year, term_id)
SELECT  crn, subject, course_num, type, title, units, days, description, capacity, enrolled, remaining, year, term_id FROM tmp;
DROP TABLE tmp;

-- SELECT * INTO tmp FROM lectures LIMIT 0;
-- \COPY tmp (crn,units,days,start_time,end_time,start_date,end_date,instructor_id) FROM 'data/fall-2020/lecture.csv' DELIMITER ',' CSV;
-- INSERT INTO lectures(crn,units,days,start_time,end_time,start_date,end_date,instructor_id)
-- SELECT (crn,units,days,start_time,end_time,start_date,end_date,instructor_id)
-- FROM tmp;
-- DROP TABLE tmp;

-- SELECT * INTO tmp FROM aux LIMIT 0;
-- \COPY tmp (crn,course_crn,section,units,days,start_time,end_time,building_room,instructor_id) FROM 'data/fall-2020/labs_disc.csv' DELIMITER ',' CSV;
-- INSERT INTO aux (crn,course_crn,section,units,days,start_time,end_time,building_room,instructor_id)
-- SELECT (crn,course_crn,section,units,days,start_time,end_time,building_room,instructor_id)
-- FROM tmp;
-- DROP TABLE tmp;

-- SELECT * INTO tmp FROM exam LIMIT 0;
-- \COPY tmp FROM 'data/fall-2020/exam.csv' DELIMITER ',' CSV;
-- INSERT INTO exam
-- SELECT * FROM tmp;
-- DROP TABLE tmp;

-- Summer 2020

SELECT * INTO tmp FROM course LIMIT 0;
\COPY tmp (crn, subject, course_num, type, title, units, days, description, capacity, enrolled, remaining, year, term_id) FROM 'data/summer-2020/course.csv' DELIMITER ',' CSV;
INSERT INTO
course (crn, subject, course_num, type, title, units, days, description, capacity, enrolled, remaining, year, term_id)
SELECT  crn, subject, course_num, type, title, units, days, description, capacity, enrolled, remaining, year, term_id FROM tmp;
DROP TABLE tmp;
