package catalog

import (
	"log"
	"strings"
	"time"

	"github.com/lib/pq"
)

// Entry is an entry in the catalog
type Entry struct {
	ID        int    `db:"id" json:"id" csv:"-"`
	CRN       int    `db:"crn" json:"crn"`
	Subject   string `db:"subject" json:"subject"`
	CourseNum int    `db:"course_num" json:"course_num"`
	Type      string `db:"type" json:"type"`
	Title     string `db:"title" json:"title"`

	Units int `db:"units" json:"units" csv:"units"`

	Days Weekdays `db:"days" json:"days" csv:"days" goqu:"skipinsert"`
	// Days string   `db:"days" json:"days" csv:"days"`

	Description string `db:"description" json:"description"`
	Capacity    int    `db:"capacity" json:"capacity"`
	Enrolled    int    `db:"enrolled" json:"enrolled"`
	Remaining   int    `db:"remaining" json:"remaining"`

	UpdatedAt time.Time `db:"updated_at" json:"updated_at" csv:"-" goqu:"skipupdate,skipinsert"`
	// AutoUpdated int       `db:"auto_updated" json:"-" csv:"-"`

	// InstructorID int    `db:"instructor_id" csv:"-"`
	// Instructor   string `db:"instructor" csv:"-"`

	Year   int `db:"year" json:"year" csv:"year"`
	TermID int `db:"term_id" json:"term_id" csv:"term_id"`
}

// Weekdays is a slice of weekdays
type Weekdays []Weekday

// Scan is used when querying the database for weekdays strings
func (wk *Weekdays) Scan(val interface{}) error {
	err := pq.Array(wk).Scan(val)
	if err != nil {
		log.Println("HERE", err)
	}
	return err
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
