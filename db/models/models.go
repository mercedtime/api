package models

import (
	"time"
)

//go:generate mkdir -p ../data
//go:generate go run ../../cmd/mtupdate -csv -out=../data/spring-2021

// Activity types for any given course
const (
	Lect       = "LECT"
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

// Date is an alias for time.Time
type Date time.Time

func (d *Date) String() string {
	return time.Time(*d).Format(time.RFC3339)
}

// Time is an alias for time.Time
type Time time.Time

func (t *Time) String() string {
	return time.Time(*t).Format("15:04:05")
}

// Default date and time formats
var (
	SQLiteTimeFormat = "15:04:05"

	TimeFormat = "15:04:05"
	// TimeFormat = time.RFC3339
	DateFormat = time.RFC3339
)

// Lecture is a lecture
type Lecture struct {
	CRN          int       `db:"crn" csv:"crn"`
	StartTime    time.Time `db:"start_time" csv:"start_time" json:"start_time"`
	EndTime      time.Time `db:"end_time" csv:"end_time" json:"end_time"`
	StartDate    time.Time `db:"start_date" csv:"start_date" json:"start_date"`
	EndDate      time.Time `db:"end_date" csv:"end_date" json:"end_date"`
	InstructorID int64     `db:"instructor_id" csv:"instructor_id" json:"instructor_id"`
	LastUpdated  time.Time `db:"updated_at" json:"updated_at" csv:"-"`
	AutoUpdated  int       `db:"auto_updated" json:"-" csv:"-"`
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
	ID   int64  `db:"id" json:"id"`
	Name string `db:"name" json:"name"`
}

// LabDisc is a lab or a discussion
//
// ONLY HERE FOR COMPATIBILITY
type LabDisc = SubCourse

// SubCourse is an auxillary course that is meant to be taken
// along side some other main course
type SubCourse struct {
	CRN int `db:"crn" json:"crn"`
	// TODO change this to LectureCRN => lecture_crn
	CourseCRN    int       `db:"course_crn" json:"course_crn"`
	Section      string    `db:"section" json:"section"`
	StartTime    time.Time `db:"start_time" json:"start_time,omitempty"`
	EndTime      time.Time `db:"end_time" json:"end_time,omitempty"`
	Building     string    `db:"building_room" json:"building_room"`
	InstructorID int64     `db:"instructor_id" json:"instructor_id"`

	UpdatedAt   time.Time `db:"updated_at" json:"updated_at" csv:"-"`
	AutoUpdated int       `db:"auto_updated" json:"-" csv:"-"`
}

// Scan helper
// SELECT crn,course_num,title,units,activity,days,start_time,end_time,start_date,end_date,instructor_id
func (l *Lecture) Scan(sc Scanable) error {
	var (
		stime, etime string
		sdate, edate string
	)
	err := sc.Scan(
		&l.CRN,
		&stime,
		&etime,
		&sdate,
		&edate,
		&l.InstructorID,
		&l.AutoUpdated,
	)
	if err != nil {
		return err
	}

	l.StartTime, err = time.Parse(DateFormat, stime)
	if err != nil {
		return err
	}
	l.EndTime, err = time.Parse(DateFormat, etime)
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
	e.StartTime, err = time.Parse(DateFormat, startTime)
	if err != nil {
		return err
	}
	e.EndTime, err = time.Parse(DateFormat, endTime)
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
		// &l.Units,
		// &l.Days,
		&stime,
		&etime,
		&l.Building,
		&l.InstructorID,
		&l.AutoUpdated,
	)
	if err != nil {
		return err
	}

	l.StartTime, err = time.Parse(DateFormat, stime)
	if err != nil {
		return err
	}
	l.EndTime, err = time.Parse(DateFormat, etime)
	if err != nil {
		return err
	}
	return nil
}
