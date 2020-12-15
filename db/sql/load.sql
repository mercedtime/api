-- \COPY instructor FROM 'data/instructor.csv' DELIMITER ',' CSV HEADER;

\COPY instructor FROM 'data/instructor.csv' DELIMITER ',' CSV;

\COPY course (crn, subject, course_num, type, title, description, capacity, enrolled, remaining) FROM 'data/course.csv' DELIMITER ',' CSV;
\COPY lectures (crn,units,days,start_time,end_time,start_date,end_date,instructor_id) FROM 'data/lecture.csv'    DELIMITER ',' CSV;
\COPY aux (crn,course_crn,section,units,days,start_time,end_time,building_room,instructor_id) FROM 'data/labs_disc.csv'  DELIMITER ',' CSV;
\COPY exam FROM 'data/exam.csv'       DELIMITER ',' CSV;

UPDATE lectures
SET days = ''
WHERE days IS NULL;

UPDATE aux
SET days = ''
WHERE days IS NULL;
