\COPY instructor FROM 'data/instructor.csv' DELIMITER ',' CSV HEADER;
\COPY course FROM     'data/course.csv' DELIMITER ',' CSV HEADER;
\COPY Lectures FROM   'data/lecture.csv' DELIMITER ',' CSV HEADER;
\COPY Labs_Discussions FROM 'data/labs_disc.csv' DELIMITER ',' CSV HEADER;
\COPY Exam FROM       'data/exam.csv' DELIMITER ',' CSV HEADER;
\COPY Enrollment FROM 'data/enrollment.csv' DELIMITER ',' CSV HEADER;
