package catalog

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/lib/pq"
)

type Catalog []*Course

type Course struct {
	Entry
	Subcourses SubCourseList `db:"subcourses" json:"subcourses"`
}

// Entry is an entry in the catalog
type Entry struct {
	ID          int       `db:"id" json:"id" csv:"-"`
	CRN         int       `db:"crn" json:"crn"`
	Subject     string    `db:"subject" json:"subject"`
	CourseNum   int       `db:"course_num" json:"course_num"`
	Type        string    `db:"type" json:"type"`
	Title       string    `db:"title" json:"title"`
	Units       int       `db:"units" json:"units" csv:"units"`
	Days        Weekdays  `db:"days" json:"days" csv:"days" goqu:"skipinsert"`
	Description string    `db:"description" json:"description"`
	Capacity    int       `db:"capacity" json:"capacity"`
	Enrolled    int       `db:"enrolled" json:"enrolled"`
	Remaining   int       `db:"remaining" json:"remaining"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at" csv:"-" goqu:"skipupdate,skipinsert"`
	Year        int       `db:"year" json:"year" csv:"year"`
	TermID      int       `db:"term_id" json:"term_id" csv:"term_id"`
}

// SubCourseList is a list of SubCourses that maintains
// interoperability with postgresql json blobs.
type SubCourseList []SubCourse

// Scan will convert the list of subcourses from json
// to a serialized struct slice.
func (sc *SubCourseList) Scan(val interface{}) error {
	b, ok := val.([]byte)
	if ok {
		return json.Unmarshal(b, sc)
	}
	return nil
}

// SubCourse is an auxillary course that is meant to be taken
// along side some other main course
type SubCourse struct {
	CRN          int       `db:"crn" json:"crn"`
	CourseCRN    int       `db:"course_crn" json:"course_crn"`
	Section      string    `db:"section" json:"section"`
	StartTime    time.Time `db:"start_time" json:"start_time,omitempty"`
	EndTime      time.Time `db:"end_time" json:"end_time,omitempty"`
	Building     string    `db:"building_room" json:"building_room"`
	InstructorID int64     `db:"instructor_id" json:"instructor_id"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at" csv:"-" goqu:"skipupdate,skipinsert"`
	Enrolled     int       `db:"enrolled" json:"enrolled"`
	Days         Weekdays  `db:"days" json:"days"`
}

// Weekdays is a slice of weekdays
type Weekdays []Weekday

// Scan is used when querying the database for weekdays strings
func (wk *Weekdays) Scan(val interface{}) error {
	return pq.Array(wk).Scan(val)
}

// Weekday is a weekday
type Weekday string

// These are all weekday values
const (
	Sunday    Weekday = "sunday"
	Monday    Weekday = "monday"
	Tuesday   Weekday = "tuesday"
	Wednesday Weekday = "wednesday"
	Thursday  Weekday = "thursday"
	Friday    Weekday = "friday"
	Saturday  Weekday = "saturday"
)

// Scan is populate the weekday's value from
// the results of a database query
func (w *Weekday) Scan(val interface{}) error {
	if b, ok := val.([]byte); ok {
		*w = Weekday(string(b))
	}
	return nil
}

// WeekdayFromTimePkg will create a new Weekday from
// a time.Weekdays
func WeekdayFromTimePkg(w time.Weekday) Weekday {
	return Weekday(strings.ToLower(w.String()))
}

// NewWeekdays will convert a slice of time.Weekday
// to []catalog.Weekday
func NewWeekdays(days []time.Weekday) []Weekday {
	wk := make([]Weekday, len(days))
	for i, d := range days {
		wk[i] = Weekday(WeekdayFromTimePkg(d))
	}
	return wk
}
