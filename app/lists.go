package app

import (
	"database/sql"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/postgres" // need postgres dialect
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/mercedtime/api/catalog"
	"github.com/mercedtime/api/db/models"
)

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

func getCatalog(db *sqlx.DB) gin.HandlerFunc {
	var (
		addons = map[string]string{
			"year":    " AND year = $%d",
			"term":    " AND term_id = $%d",
			"subject": " AND subject = $%d",
		}
	)

	// using a view for this query
	catalogQuery := `select * from catalog
					 where type in ('LECT','SEM','STDO')`

	return func(c *gin.Context) {
		var (
			q      = catalogQuery
			args   = make([]interface{}, 0, 5)
			argc   = 1
			result = make(catalog.Catalog, 0, 32)
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

		order, ok := c.GetQuery("order")
		if ok {
			switch order {
			case "updated_at":
				q += " ORDER BY updated_at DESC"
			case "capacity":
				q += " ORDER BY capacity ASC"
			case "enrolled":
				q += " ORDER BY enrolled ASC"
			}
		}

		if params.Limit != nil {
			q += fmt.Sprintf(" LIMIT $%d", argc)
			args = append(args, params.Limit)
			argc++
		}
		if params.Offset != nil {
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
	Percent   float32       `db:"percent" json:"percent"`
	CRNs      pq.Int32Array `db:"crns" json:"crns"`
	IDs       pq.Int32Array `db:"ids" json:"ids"`
	Count     int           `db:"count" json:"count"`
}

func (a *App) getCourseBluprints(c *gin.Context) {
	var (
		query = `
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
		err    error
		where  string
		args   = make([]interface{}, 0, 2)
		resp   = make([]CourseBlueprint, 0, 350)
		params struct {
			PageParams
			SemesterParams
			Units int `query:"units" form:"units"`
		}
	)

	if params.Subject != "" {
		where += "AND subject = $1 "
		args = append(args, strings.ToUpper(params.Subject))
	}
	if params.Term != "" {
		where += "AND term_id = $2 "
		args = append(args, getTermID(params.Term))
	}
	if params.Year != 0 {
		where += "AND year = $3 "
		args = append(args, params.Year)
	}
	if err = c.BindQuery(&params); err != nil {
		senderr(c, err, 500)
		return
	}
	l := len(args) + 1
	err = a.DB.Select(
		&resp, fmt.Sprintf(query, where, l, l+1),
		append(args, params.Limit, params.Offset)...,
	)
	if err != nil {
		senderr(c, err, 500)
		return
	}
	c.JSON(200, resp)
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
