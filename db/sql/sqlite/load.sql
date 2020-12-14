.mode "csv"
.separator ","
.headers off

.import "data/course.csv" "course"
.import "data/lecture.csv" "Lectures"
.import "data/labs_disc.csv" "Labs_Discussions"
.import "data/exam.csv" "Exam"
.import "data/instructor.csv" "instructor"