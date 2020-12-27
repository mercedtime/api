\COPY instructor FROM 'data/spring-2021/instructor.csv' DELIMITER ',' CSV;
\COPY exam FROM 'data/spring-2021/exam.csv' DELIMITER ',' CSV;
\COPY course (crn, subject, course_num, type, title, units, days, description, capacity, enrolled, remaining, year, term_id) FROM 'data/spring-2021/course.csv' DELIMITER ',' CSV;
\COPY lectures (crn,start_time,end_time,start_date,end_date,instructor_id) FROM 'data/spring-2021/lecture.csv' DELIMITER ',' CSV;
\COPY aux (crn,course_crn,section,start_time,end_time,building_room,instructor_id) FROM 'data/spring-2021/labs_disc.csv' DELIMITER ',' CSV;


-- Copy all the historical data.

-- Fall 2020

SELECT * INTO tmp FROM instructor LIMIT 0;
\COPY tmp FROM 'data/fall-2020/instructor.csv' DELIMITER ',' CSV;
INSERT INTO instructor SELECT DISTINCT id, name FROM tmp
      WHERE
        id NOT IN (SELECT id FROM instructor);
DELETE FROM tmp;
\COPY tmp FROM 'data/summer-2020/instructor.csv' DELIMITER ',' CSV;
INSERT INTO instructor SELECT DISTINCT id, name FROM tmp
      WHERE
        id NOT IN (SELECT id FROM instructor);
DROP TABLE tmp;

-- Fall 2020
\COPY exam FROM 'data/fall-2020/exam.csv' DELIMITER ',' CSV;
\COPY course (crn, subject, course_num, type, title, units, days, description, capacity, enrolled, remaining, year, term_id) FROM 'data/fall-2020/course.csv' DELIMITER ',' CSV;
\COPY lectures (crn,start_time,end_time,start_date,end_date,instructor_id) FROM 'data/fall-2020/lecture.csv' DELIMITER ',' CSV;
\COPY aux (crn,course_crn,section,start_time,end_time,building_room,instructor_id) FROM 'data/fall-2020/labs_disc.csv' DELIMITER ',' CSV;

-- Summer 2020
\COPY exam FROM 'data/summer-2020/exam.csv' DELIMITER ',' CSV;
\COPY course (crn, subject, course_num, type, title, units, days, description, capacity, enrolled, remaining, year, term_id) FROM 'data/summer-2020/course.csv' DELIMITER ',' CSV;
\COPY lectures (crn,start_time,end_time,start_date,end_date,instructor_id) FROM 'data/summer-2020/lecture.csv' DELIMITER ',' CSV;
\COPY aux (crn,course_crn,section,start_time,end_time,building_room,instructor_id) FROM 'data/summer-2020/labs_disc.csv' DELIMITER ',' CSV;
