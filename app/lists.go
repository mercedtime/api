package app

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/postgres" // need postgres dialect
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/mercedtime/api/catalog"
	"github.com/mercedtime/api/db/models"
)

// PageParams are url params for api pagination
type PageParams struct {
	Limit  uint `form:"limit"  query:"limit"  db:"limit"`
	Offset uint `form:"offset" query:"offset" db:"offset"`
}

func (pp *PageParams) toExpr() goqu.Ex {
	ex := goqu.Ex{}
	if pp.Limit != 0 {
		ex["limit"] = pp.Limit
	}
	if pp.Offset != 0 {
		ex["offset"] = pp.Offset
	}
	return ex
}

func (pp *PageParams) appendSelect(stmt *goqu.SelectDataset) *goqu.SelectDataset {
	if pp.Limit != 0 {
		stmt = stmt.Limit(pp.Limit)
	}
	if pp.Offset != 0 {
		stmt = stmt.Offset(pp.Offset)
	}
	return stmt
}

func (pp *PageParams) asSQL(stmtIndex int) (string, []interface{}, int) {
	var (
		q    string
		args = make([]interface{}, 0, 2)
	)
	if pp.Limit != 0 {
		args = append(args, pp.Limit)
		q += fmt.Sprintf(" LIMIT $%d", stmtIndex)
		stmtIndex++
	}
	if pp.Offset != 0 {
		args = append(args, pp.Offset)
		q += fmt.Sprintf(" OFFSET $%d", stmtIndex)
		stmtIndex++
	}
	return q, args, stmtIndex
}

// Expression implements the goqu.Expression interface
func (pp *PageParams) Expression() goqu.Expression { return pp.toExpr() }

// Clone implements the goqu.Expression interface
func (pp *PageParams) Clone() goqu.Expression { return pp.toExpr() }

// SemesterParams is a structure that defines
// parameters that control which courses are returned from a query
type SemesterParams struct {
	Year    int    `form:"year" uri:"year" query:"year" db:"year"`
	Term    string `form:"term" uri:"term" query:"term" db:"term_id"`
	Subject string `form:"subject" query:"subject" db:"subject"`
}

func (sp *SemesterParams) toExpr() goqu.Ex {
	ex := goqu.Ex{}
	if sp.Year != 0 {
		ex["year"] = sp.Year
	}
	if sp.Term != "" {
		if id := getTermID(sp.Term); id != 0 {
			ex["term_id"] = id
		}
	}
	if sp.Subject != "" {
		ex["subject"] = sp.Subject
	}
	return ex
}

// Expression implements the goqu.Expression interface
func (sp *SemesterParams) Expression() goqu.Expression { return sp.toExpr() }

// Clone implements the goqu.Expression interface
func (sp *SemesterParams) Clone() goqu.Expression { return sp.toExpr() }

func listParamsMiddleware(c *gin.Context) {
	for _, key := range []string{"limit", "offset"} {
		query, ok := c.GetQuery(key)
		if !ok || query == "" {
			c.Set(key, nil)
			continue
		}
		u, err := strconv.ParseUint(query, 10, 32)
		if err != nil {
			c.Set(key, nil)
			c.AbortWithStatusJSON(400, &Error{fmt.Sprintf("invalid %s", key), 400})
			return
		}
		c.Set(key, uint(u))
	}
	c.Next()
}

// SubCourseList is a list of SubCourses that maintains
// interoperability with postgresql json blobs.
type subCourseList []struct {
	models.SubCourse

	Enrolled        int              `db:"enrolled" json:"enrolled"`
	Days            catalog.Weekdays `db:"days" json:"days"`
	CourseUpdatedAt time.Time        `db:"course_updated_at" json:"course_updated_at"`
}

// Scan will convert the list of subcourses from json
// to a serialized struct slice.
func (sc *subCourseList) Scan(val interface{}) error {
	b, ok := val.([]byte)
	if ok {
		return json.Unmarshal(b, sc)
	}
	return nil
}

func getCatalog(db *sqlx.DB) gin.HandlerFunc {
	var (
		// TODO fix time parsing so I can un-comment out the following SQL
		catalogQuery = `
SELECT c.*, array_to_json(sub) AS subcourses
FROM course c
LEFT OUTER JOIN (
	SELECT
		course_crn,
		array_agg(json_build_object(
			'crn', aux.crn,
			'course_crn', aux.course_crn,
			'section', aux.section,
			'days', course.days,
			'enrolled', course.enrolled,
			'start_time', aux.start_time,
			'end_time', aux.end_time,
			'building_room', aux.building_room,
			'instructor_id', aux.instructor_id,
			'updated_at', aux.updated_at,
			'course_updated_at', course.updated_at
		)) AS sub
		 FROM aux
		 JOIN course ON aux.crn = course.crn
	    WHERE aux.course_crn != 0
	 GROUP BY aux.course_crn
  ) a
           ON c.crn = a.course_crn
		WHERE c.crn IN (SELECT crn FROM exam)`
		addons = map[string]string{
			"year": " AND c.year = $%d",
			"term": " AND c.term_id = $%d",
		}
	)
	type response struct {
		catalog.Entry
		Subcourses subCourseList `db:"subcourses" json:"subcourses"`
	}

	return func(c *gin.Context) {
		var (
			q      = catalogQuery
			args   = make([]interface{}, 0, 5)
			argc   = 1
			result = make([]response, 0, 32)
			params = PageParams{}
		)
		if err := c.BindQuery(&params); err != nil {
			c.JSON(500, &Error{err.Error(), 500})
			return
		}

		for key, addon := range addons {
			if param, ok := c.Get(key); ok && param != nil {
				q += fmt.Sprintf(addon, argc)
				args = append(args, param)
				argc++
			}
		}

		if params.Limit != 0 {
			q += fmt.Sprintf(" LIMIT $%d", argc)
			args = append(args, params.Limit)
			argc++
		}
		if params.Offset != 0 {
			q += fmt.Sprintf(" OFFSET $%d", argc)
			args = append(args, params.Offset)
			argc++
		}

		err := db.Select(&result, q, args...)
		if err != nil {
			senderr(c, err, 500)
			return
		}
		c.JSON(200, result)
	}
}

var listCoursesQuery = `select * from course `

func (a *App) listCourses(c *gin.Context) {
	var (
		resp = make([]catalog.Entry, 0, 500)
		p    struct {
			SemesterParams
			PageParams
		}
	)

	err := c.Bind(&p)
	if err != nil {
		senderr(c, err, 400)
		return
	}
	stmt := goqu.From("course").SetDialect(
		goqu.GetDialect("postgres"),
	).Prepared(true).Select(
		"*",
	).Where(
		p.SemesterParams.Expression(),
	)
	stmt = p.PageParams.appendSelect(stmt)
	query, args, err := stmt.ToSQL()
	if err != nil {
		c.JSON(500, Error{Msg: "internal error"})
		return
	}
	err = a.DB.Select(&resp, query, args...)
	if err != nil {
		log.Println(err)
		c.JSON(500, Error{Msg: "could not query database"})
		return
	}
	c.JSON(200, resp)
}

func interfaceSlice(slice interface{}) []interface{} {
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		panic("InterfaceSlice() given a non-slice type")
	}
	ret := make([]interface{}, s.Len())
	for i := 0; i < s.Len(); i++ {
		ret[i] = s.Index(i).Interface()
	}
	return ret
}

// ListLectures returns a handlerfunc that lists lectures.
// Depends on "limit" and "offset" being set from middleware.
func ListLectures(db *sqlx.DB) func(*gin.Context) {
	var (
		lectures []models.Lecture
		// TODO tell goqu to generate a prepared statment to
		// prevent sql injection
		stmt = goqu.From("lectures").SetDialect(goqu.GetDialect("postgres")).Prepared(true).Select(
			interfaceSlice(models.GetNamedSchema("lectures", models.Lecture{}))...)
	)
	return func(c *gin.Context) {
		lectures = nil // deallocate from previous calls
		subject, ok := c.GetQuery("subject")
		var q = stmt
		if ok {
			q = q.Join(
				goqu.I("course"),
				goqu.On(goqu.I("course.crn").Eq(goqu.I("lectures.crn"))),
			).Where(
				goqu.Ex{"course.subject": strings.ToUpper(subject)})
		}

		if limit, ok := c.Get("limit"); ok && limit != nil {
			q = q.Limit(limit.(uint))
		}
		if offset, ok := c.Get("offset"); ok && offset != nil {
			q = q.Offset(offset.(uint))
		}
		query, args, err := q.ToSQL()
		if err != nil {
			senderr(c, err, 500)
			return
		}

		err = db.Select(&lectures, query, args...)
		if err == sql.ErrNoRows {
			c.JSON(404, Error{"no lectures found", 404})
			return
		}
		if err != nil {
			senderr(c, err, 500)
			return
		}
		c.JSON(200, lectures)
	}
}

// ListLabs returns a handlerfunc that lists labs.
// Depends on "limit" and "offset" being set from middleware.
func ListLabs(db *sqlx.DB) gin.HandlerFunc {
	var (
		err   error
		query = `
	  SELECT
	  	` + strings.Join(models.GetNamedSchema("aux", models.LabDisc{}), ",") + `
	  FROM aux,course
	  WHERE
	  	aux.crn = course.crn AND
	  	course.type = 'LAB'
	  LIMIT $1 OFFSET $2`
	)
	var list []models.LabDisc
	return func(c *gin.Context) {
		list = nil
		if err = db.Select(
			&list, query,
			c.MustGet("limit"),
			c.MustGet("offset"),
		); err != nil {
			senderr(c, err, 500)
			return
		}
		c.JSON(200, list)
	}
}

// ListDiscussions returns a handlerfunc that lists discussions.
// Depends on "limit" and "offset" being set from middleware.
func ListDiscussions(db *sqlx.DB) gin.HandlerFunc {
	var err error
	query := `
	  SELECT
	  	` + strings.Join(models.GetNamedSchema("aux", models.LabDisc{}), ",") + `
	  FROM aux,course
	  WHERE
	  	aux.crn = course.crn AND
	  	course.type = 'DISC'
	  LIMIT $1 OFFSET $2`
	var (
		list []models.LabDisc
	)
	return func(c *gin.Context) {
		list = nil
		if err = db.Select(
			&list, query,
			c.MustGet("limit"),
			c.MustGet("offset"),
		); err != nil {
			senderr(c, err, 500)
			return
		}
		c.JSON(200, list)
	}
}

// ListExams returns a handlerfunc that lists exams.
// Depends on "limit" and "offset" being set from middleware.
func ListExams(db *sqlx.DB) gin.HandlerFunc {
	var err error
	query := `
	  SELECT
	  	crn, date, start_time, end_time
	  FROM exam
	  LIMIT $1 OFFSET $2`
	var list []models.Exam
	return func(c *gin.Context) {
		list = nil
		if err = db.Select(
			&list, query,
			c.MustGet("limit"),
			c.MustGet("offset"),
		); err != nil {
			senderr(c, err, 500)
			return
		}
		c.JSON(200, list)
	}
}
