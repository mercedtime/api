package models

import (
	"time"
)

//go:generate mkdir -p ../data
//go:generate go run ../../cmd/mtupdate -csv -out=../data

// Activity types for any given course
const (
	Lecture    = "LECT"
	Discussion = "DISC"
	Lab        = "LAB"
	Seminar    = "SEM"
	Studio     = "STDO"
	FieldWork  = "FLDW"

	// TODO find out what this is
	TheOtherWierdCourseType = "INI"
)

// Scanable is an sql.Row or sql.Rows
type Scanable interface {
	Scan(...interface{}) error
}

// Default date and time formats
var (
	SQLiteTimeFormat = "15:04:05"

	TimeFormat = "15:04:05"
	DateFormat = time.RFC3339
)

// Course is a course
type Course struct {
	CRN       int    `db:"crn" json:"crn"`
	Subject   string `db:"subject" json:"subject"`
	CourseNum int    `db:"course_num" json:"course_num"`
	Type      string `db:"type" json:"type"`
	Title     string `db:"title" json:"title"`
}

// Lect is a lecture
type Lect struct {
	CRN          int       `db:"crn" csv:"crn" json:"crn"`
	Units        int       `db:"units" csv:"units" json:"units"`
	Days         string    `db:"days" csv:"days" json:"days"`
	StartTime    time.Time `db:"start_time" csv:"start_time" json:"start_time"`
	EndTime      time.Time `db:"end_time" csv:"end_time" json:"end_time"`
	StartDate    time.Time `db:"start_date" csv:"start_date" json:"start_date"`
	EndDate      time.Time `db:"end_date" csv:"end_date" json:"end_date"`
	InstructorID int       `db:"instructor_id" csv:"instructor_id" json:"instructor_id"`
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

// LabDisc is a lab or a discussion
type LabDisc struct {
	CRN int `db:"crn" json:"crn"`
	// TODO change this to LectureCRN => lecture_crn
	CourseCRN    int       `db:"course_crn" json:"course_crn"`
	Section      string    `db:"section" json:"section"`
	Units        int       `db:"units" json:"units"`
	Days         string    `db:"days" json:"days"` // TODO move days to the course table
	StartTime    time.Time `db:"start_time" json:"start_time"`
	EndTime      time.Time `db:"end_time" json:"end_time"`
	Building     string    `db:"building_room" json:"building_room"`
	InstructorID int       `db:"instructor_id" json:"instructor_id"`
}

// Enrollment is the enrollment table
type Enrollment struct {
	CRN       int    `db:"crn" json:"crn" goqu:"skipupdate"`
	Desc      string `db:"description" json:"description"`
	Capacity  int    `db:"capacity" json:"capacity"`
	Enrolled  int    `db:"enrolled" json:"enrolled"`
	Remaining int    `db:"remaining" json:"remaining"`
}

// Scan helper
// SELECT crn,course_num,title,units,activity,days,start_time,end_time,start_date,end_date,instructor_id
func (l *Lect) Scan(sc Scanable) error {
	var (
		stime, etime string
		sdate, edate string
	)
	err := sc.Scan(
		&l.CRN,
		&l.Units,
		&l.Days,
		&stime,
		&etime,
		&sdate,
		&edate,
		&l.InstructorID,
	)
	if err != nil {
		return err
	}

	l.StartTime, err = time.Parse(TimeFormat, stime)
	if err != nil {
		return err
	}
	l.EndTime, err = time.Parse(TimeFormat, etime)
	if err != nil {
		return err
	}
	l.StartDate, err = time.Parse(DateFormat, sdate)
	if err != nil {
		return err
	}
	l.EndDate, err = time.Parse(DateFormat, edate)
	if err != nil {
		return err
	}
	return nil
}

// Scan helper function
func (e *Exam) Scan(sc Scanable) error {
	var (
		date, startTime, endTime string
	)
	err := sc.Scan(&e.CRN, &date, &startTime, &endTime)
	if err != nil {
		return err
	}
	e.Date, err = time.Parse(DateFormat, date)
	if err != nil {
		return err
	}
	e.StartTime, err = time.Parse(TimeFormat, startTime)
	if err != nil {
		return err
	}
	e.EndTime, err = time.Parse(TimeFormat, endTime)
	if err != nil {
		return err
	}
	return nil
}

// Scan helper
// SELECT crn,course_num,title,units,activity,days,start_time,end_time,start_date,end_date,instructor_id
func (l *LabDisc) Scan(sc Scanable) error {
	var stime, etime string
	err := sc.Scan(
		&l.CRN,
		&l.CourseCRN,
		&l.Section,
		&l.Units,
		&l.Days,
		&stime,
		&etime,
		&l.Building,
		&l.InstructorID,
	)
	if err != nil {
		return err
	}

	l.StartTime, err = time.Parse(TimeFormat, stime)
	if err != nil {
		return err
	}
	l.EndTime, err = time.Parse(TimeFormat, etime)
	if err != nil {
		return err
	}
	return nil
}
