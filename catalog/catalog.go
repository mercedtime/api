package catalog

import "time"

// Entry is an entry in the catalog
type Entry struct {
	ID        int    `db:"id" json:"id" csv:"-"`
	CRN       int    `db:"crn" json:"crn"`
	Subject   string `db:"subject" json:"subject"`
	CourseNum int    `db:"course_num" json:"course_num"`
	Type      string `db:"type" json:"type"`
	Title     string `db:"title" json:"title"`

	Units       int    `db:"units" json:"units" csv:"units"`
	Days        string `db:"days" json:"days" csv:"days"`
	Description string `db:"description" json:"description"`
	Capacity    int    `db:"capacity" json:"capacity"`
	Enrolled    int    `db:"enrolled" json:"enrolled"`
	Remaining   int    `db:"remaining" json:"remaining"`

	UpdatedAt   time.Time `db:"updated_at" json:"updated_at" csv:"-"`
	AutoUpdated int       `db:"auto_updated" json:"-" csv:"-"`

	Year   int `db:"year" json:"year" csv:"year"`
	TermID int `db:"term_id" json:"term_id" csv:"term_id"`
}
