-- \COPY instructor FROM 'data/instructor.csv' DELIMITER ',' CSV HEADER;

\COPY instructor FROM 'data/instructor.csv' DELIMITER ',' CSV;
\COPY course     FROM 'data/course.csv'     DELIMITER ',' CSV;
\COPY Lectures   FROM 'data/lecture.csv'    DELIMITER ',' CSV;
\COPY aux        FROM 'data/labs_disc.csv'  DELIMITER ',' CSV;
\COPY Exam       FROM 'data/exam.csv'       DELIMITER ',' CSV;
\COPY Enrollment FROM 'data/enrollment.csv' DELIMITER ',' CSV;

UPDATE lectures
SET days = ''
WHERE days IS NULL;

UPDATE aux
SET days = ''
WHERE days IS NULL;
