package app

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/mercedtime/api/db/models"
)

func listParamsMiddleware(c *gin.Context) {
	var (
		limit  interface{} = defaultLimit
		offset interface{} = defaultOffset
	)
	if lim, ok := c.GetQuery("limit"); ok && lim != "" {
		limit = lim
	}
	if off, ok := c.GetQuery("offset"); ok && off != "" {
		offset = off
	}
	c.Set("limit", limit)
	c.Set("offset", offset)
	c.Next()
}

// ListLectures returns a handlerfunc that lists lectures.
// Depends on "limit" and "offset" being set from middleware.
func ListLectures(db *sqlx.DB) func(*gin.Context) {
	var (
		lecturesQuery = `
		  SELECT ` + strings.Join(models.GetSchema(models.Lecture{}), ",") + `
		  FROM lectures
		  LIMIT $1 OFFSET $2`
		lecturesBySubjectQuery = `
		  SELECT ` + strings.Join(models.GetNamedSchema("l", models.Lecture{}), ",") + `
		  FROM lectures l, course c
		  WHERE
		  	l.crn = c.crn AND
		  	c.subject = $1
		  LIMIT $2 OFFSET $3`
		err      error
		lectures []models.Lecture
	)
	return func(c *gin.Context) {
		lectures = nil // deallocate from previous calls
		subject, ok := c.GetQuery("subject")
		if ok {
			err = db.Select(
				&lectures, lecturesBySubjectQuery,
				strings.ToUpper(subject),
				c.MustGet("limit"), c.MustGet("offset"),
			)
		} else {
			err = db.Select(&lectures, lecturesQuery, c.MustGet("limit"), c.MustGet("offset"))
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
