package catalog

import (
	"fmt"
	"strings"

	"github.com/lib/pq"
	"github.com/mercedtime/api/db"
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

const blueprintQuery = `
	SELECT
		  subject,
		  course_num,
		  (array_agg(title))[1] AS title,
		  min(units) AS min_units,
		  max(units) AS max_units,
		  sum(c.enrolled) AS enrolled,
		  sum(c.capacity) AS capacity,
		  sum(c.enrolled)::float / sum(c.capacity)::float AS percent,
		  array_agg(c.crn) AS crns,
		  array_agg(c.id) AS ids,
		  count(*) AS count
	  FROM
		  course c
	 WHERE 0 = 0
		  %s
  GROUP BY
		  subject,
		  course_num
  ORDER BY
		  subject ASC,
		  course_num ASC
  LIMIT $%d OFFSET $%d`

// BlueprintParams is the set of
// parameters for the blueprint query.
type BlueprintParams struct {
	PageParams
	SemesterParams
	Units int `query:"units" form:"units"`
}

// GetBlueprints will
func GetBlueprints(params *BlueprintParams) ([]*CourseBlueprint, error) {
	var (
		where string
		err   error
		db    = db.Get()
		args  = make([]interface{}, 0, 2)
		resp  = make([]*CourseBlueprint, 0, 350)
	)
	if params.Subject != "" {
		where += "AND subject = $1 "
		args = append(args, strings.ToUpper(params.Subject))
	}
	if params.Term != "" {
		where += "AND term_id = $2 "
		args = append(args, GetTermID(params.Term))
	}
	if params.Year != 0 {
		where += "AND year = $3 "
		args = append(args, params.Year)
	}
	// TODO add other parameters to the query
	l := len(args) + 1
	err = db.Select(
		&resp, fmt.Sprintf(blueprintQuery, where, l, l+1),
		append(args, params.Limit, params.Offset)...,
	)
	if err != nil {
		return nil, err
	}
	return resp, err
}
