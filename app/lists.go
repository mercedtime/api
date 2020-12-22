package app

import (
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/doug-martin/goqu/v9"
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
			c.JSON(400, &Error{
				Msg:    fmt.Sprintf("invalid %s", key),
				Status: 400,
			})
			c.Set(key, nil)
			return
		}
		c.Set(key, uint(u))
	}
	c.Next()
}

func (a *App) listCourses(c *gin.Context) {
	var (
		resp   = make([]catalog.Entry, 0, 500)
		params = goqu.Ex{}
	)
	if year, ok := c.Get("year"); ok {
		params["year"] = year
	}
	if term, ok := c.Get("term"); ok {
		params["term_id"] = term
	}
	if subj, ok := c.GetQuery("subject"); ok {
		params["subject"] = subj
	}

	stmt := goqu.From("course").Select("*").Where(params)
	if limit, ok := c.Get("limit"); ok && limit != nil {
		stmt = stmt.Limit(limit.(uint))
	}
	if offset, ok := c.Get("offset"); ok && offset != nil {
		stmt = stmt.Offset(offset.(uint))
	}
	query, _, err := stmt.ToSQL()
	if err != nil {
		c.JSON(500, Error{Msg: "internal error"})
		return
	}
	err = a.DB.Select(&resp, query)
	if err != nil {
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
		stmt     = goqu.From("lectures").Select(
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

		query, _, err := q.ToSQL()
		if err != nil {
			senderr(c, err, 500)
			return
		}
		err = db.Select(&lectures, query)
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
