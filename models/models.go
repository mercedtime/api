package models

import "time"

// Lect is a lecture
type Lect struct {
	CRN          int       `db:"crn" csv:"crn" json:"crn"`
	CourseNum    int       `db:"course_num" csv:"course_num" json:"course_num"`
	Title        string    `db:"title" csv:"title" json:"title"`
	Units        int       `db:"units" csv:"units" json:"units"`
	Activity     string    `db:"activity" csv:"activity" json:"activity"`
	Days         string    `db:"days" csv:"days" json:"days"`
	StartTime    time.Time `db:"start_time" csv:"start_time" json:"start_time"`
	EndTime      time.Time `db:"end_time" csv:"end_time" json:"end_time"`
	StartDate    time.Time `db:"start_date" csv:"start_date" json:"start_date"`
	EndDate      time.Time `db:"end_date" csv:"end_date" json:"end_date"`
	InstructorID int       `db:"instructor_id" csv:"instructor_id" json:"instructor_id"`
}

type scanable interface {
	Scan(...interface{}) error
}

// LectColumns is all the columns in Lect
const LectColumns = `crn,course_num,title,units,activity,days,start_time,end_time,start_date,end_date,instructor_id`

// Scan helper
// SELECT crn,course_num,title,units,activity,days,start_time,end_time,start_date,end_date,instructor_id
func (l *Lect) Scan(sc scanable) error {
	return sc.Scan(
		&l.CRN,
		&l.CourseNum,
		&l.Title,
		&l.Units,
		&l.Activity,
		&l.Days,
		&l.StartTime,
		&l.EndTime,
		&l.StartDate,
		&l.EndDate,
		&l.InstructorID,
	)
}

// Exam is an exam
type Exam struct {
	CRN       int       `db:"crn" json:"crn"`
	Date      time.Time `db:"date" json:"date"`
	StartTime time.Time `db:"start_time" json:"start_time"`
	EndTime   time.Time `db:"end_time" json:"end_time"`
}

// Instructor is the instructor table
type Instructor struct {
	ID   int    `db:"id" json:"id"`
	Name string `db:"name" json:"name"`
}

// Course is a course
type Course struct {
	CRN       int    `db:"crn" json:"crn"`
	Subject   string `db:"subject" json:"subject"`
	CourseNum int    `db:"course_num" json:"course_num"`
	Type      string `db:"type" json:"type"`
}

// LabDisc is a lab or a discussion
type LabDisc struct {
	CRN          int       `db:"crn" json:"crn"`
	CourseCRN    int       `db:"course_crn" json:"course_crn"`
	CourseNum    int       `db:"course_num" json:"course_num"`
	Section      string    `db:"section" json:"section"`
	Title        string    `db:"title" json:"title"`
	Units        int       `db:"units" json:"units"`
	Activity     string    `db:"activity" json:"activity"`
	Days         string    `db:"days" json:"days"`
	StartTime    time.Time `db:"start_time" json:"start_time"`
	EndTime      time.Time `db:"end_time" json:"end_time"`
	Building     string    `db:"building_room" json:"building_room"`
	InstructorID int       `db:"instructor_id" json:"instructor_id"`
}

// Enrollment is the enrollment table
type Enrollment struct {
	CRN       int    `db:"crn" json:"crn"`
	Desc      string `db:"description" json:"description"`
	Capacity  int    `db:"capacity" json:"capacity"`
	Enrolled  int    `db:"enrolled" json:"enrolled"`
	Remaining int    `db:"remaining" json:"remaining"`
}
