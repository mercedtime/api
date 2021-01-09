package catalog

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
)

// Catalog is the course catalog
type Catalog []*Course

// Course is a course
type Course struct {
	Entry
	Exam       Exam          `db:"exam" json:"exam"`
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

// Exam is an exam
type Exam struct {
	Date      time.Time `db:"date" json:"date"`
	StartTime time.Time `db:"start_time" json:"start_time"`
	EndTime   time.Time `db:"end_time" json:"end_time"`
}

// Scan implements the database/sql scan interface
func (e *Exam) Scan(v interface{}) error {
	if v == nil {
		return nil
	}
	if b, ok := v.([]byte); ok {
		return json.Unmarshal(b, e)
	}
	return errors.New("could not scan object")
}

// TODO start using this for both the REST api and graphql api
func genCatalogQuery(page PageParams, order string, sem SemesterParams) (string, []interface{}) {
	var (
		base = `SELECT * FROM catalog
				where type in ('LECT','SEM','STDO')`
		c    = 1
		args = make([]interface{}, 0, 2)
	)
	if sem.Subject != "" {
		base += fmt.Sprintf(" AND subject = $%d", c)
		c++
		args = append(args, strings.ToUpper(sem.Subject))
	}
	if sem.Term != "" {
		base += fmt.Sprintf(" AND term_id = $%d", c)
		c++
		args = append(args, GetTermID(sem.Term))
	}
	if sem.Year != 0 {
		base += fmt.Sprintf(" AND year = $%d", sem.Year)
		c++
		args = append(args, sem.Year)
	}
	if order != "" {
		switch order {
		case "updated_at":
			base += " ORDER BY updated_at DESC"
		case "capacity":
			base += " ORDER BY capacity ASC"
		case "enrolled":
			base += " ORDER BY enrolled ASC"
		}
	}
	if page.Limit != nil {
		base += fmt.Sprintf(" LIMIT $%d", c)
		c++
		args = append(args, *page.Limit)
	}
	if page.Offset != nil {
		base += fmt.Sprintf(" OFFSET $%d", c)
		c++
		args = append(args, *page.Offset)
	}
	return base, args
}

// GetTermID will return the term
// id given the term name
func GetTermID(term string) int {
	switch term {
	case "spring":
		return 1
	case "summer":
		return 2
	case "fall":
		return 3
	default:
		return 0
	}
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
	Capacity     int       `db:"capacity" json:"capacity"`
	Remaining    int       `db:"remaining" json:"remaining"`
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
