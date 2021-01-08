package catalog

import (
	"github.com/lib/pq"
)

// CourseBlueprint is an overview of all of the instances of one course.
// It tells what the course subject and number are and contains a list
// of IDs that point to spesific instances of the course.
type CourseBlueprint struct {
	Subject   string        `db:"subject" json:"subject"`
	CourseNum int           `db:"course_num" json:"course_num"`
	Title     string        `db:"title" json:"title"`
	MinUnits  int           `db:"min_units" json:"min_units"`
	MaxUnits  int           `db:"max_units" json:"max_units"`
	Enrolled  int           `db:"enrolled" json:"enrolled"`
	Capacity  int           `db:"capacity" json:"capacity"`
	Percent   float64       `db:"percent" json:"percent"`
	CRNs      pq.Int32Array `db:"crns" json:"crns"`
	IDs       pq.Int32Array `db:"ids" json:"ids"`
	Count     int           `db:"count" json:"count"`
}
