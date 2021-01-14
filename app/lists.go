package app

import (
	"database/sql"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/postgres" // need postgres dialect
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
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
			"year":    "year",
			"term":    "term_id",
			"subject": "subject",
		}
	)

	// using a view for this query
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	base := psql.Select("*").From("catalog").Where(
		"type IN ('LECT','SEM','STDO')")

	return func(c *gin.Context) {
		var (
			result = make(catalog.Catalog, 0, 250)
			p      = catalog.PageParams{}
			query  = base
		)
		if err := c.BindQuery(&p); err != nil {
			c.JSON(500, &Error{err.Error(), 500})
			return
		}
		for key, addon := range addons {
			if param, ok := c.Get(key); ok && param != nil {
				query = query.Where(sq.Eq{addon: param})
			}
		}

		order, ok := c.GetQuery("order")
		if ok {
			switch order {
			case "updated_at", "capacity", "enrolled", "remaining":
				query = query.OrderBy(order + " DESC")
			}
		}

		if p.Limit != nil {
			query = query.Limit(uint64(*p.Limit))
		}
		if p.Offset != nil {
			query = query.Offset(uint64(*p.Offset))
		}
		q, args, err := query.ToSql()
		if err != nil {
			senderr(c, err, 500)
			return
		}

		err = db.Select(&result, q, args...)
		if err != nil {
			senderr(c, err, 500)
			return
		}
		c.JSON(200, result)
	}
}

func (a *App) getCourseBluprints(c *gin.Context) {
	var (
		err    error
		params catalog.BlueprintParams
	)
	if err = c.BindQuery(&params); err != nil {
		senderr(c, err, 500)
		return
	}
	resp, err := catalog.GetBlueprints(&params)
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
			catalog.SemesterParams
			catalog.PageParams
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
	stmt = p.PageParams.AppendSelect(stmt)
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
