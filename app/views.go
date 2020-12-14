package app

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/mercedtime/api/db/models"
)

// RegisterRoutes will setup all the app routes
func (a *App) RegisterRoutes(g *gin.RouterGroup) {
	g.POST("/user", a.PostUser)
	g.DELETE("/user/:id", a.deleteUser)
	g.GET("/instructor/:id", instructorFromID(a))
	g.GET("/instructor/:id/courses", instructorCourses(a.DB))

	a.listsGroup(g)
	a.lectureGroup(g)
}

func (a *App) listsGroup(g *gin.RouterGroup) {
	lists := g.Group("/", listParamsMiddleware)
	lists.GET("/lectures", ListLectures(a.DB))
	lists.GET("/exams", ListExams(a.DB))
	lists.GET("/labs", ListLabs(a.DB))
	lists.GET("/discussions", ListDiscussions(a.DB))
	lists.GET("/instructors", ListInstructors(a.DB))
}

// LectureGroup returns the router group for all the lecture routes.
func (a *App) lectureGroup(g *gin.RouterGroup) *gin.RouterGroup {
	lect := g.Group("/lecture", func(c *gin.Context) {
		crnStr, ok := c.Params.Get("crn")
		if !ok {
			c.JSON(400, ErrStatus(400, "no crn given"))
			return
		}
		crn, err := strconv.Atoi(crnStr)
		if err != nil {
			c.JSON(400, &Error{Msg: "crn given is not a number"})
			return
		}
		c.Set("crn", crn)
		c.Next()
	})
	lect.GET("/:crn", lecture(a.DB))
	lect.GET("/:crn/exam", exam(a.DB))
	lect.GET("/:crn/labs", labsForLecture(a.DB))
	lect.GET("/:crn/instructor", instructorFromLectureCRN(a.DB))
	lect.GET("/:crn/enrollment", lectureEnrollments(a))
	lect.DELETE("/:crn", func(c *gin.Context) {
		_, err := a.DB.Exec("DELETE FROM lectures WHERE crn = $1", c.MustGet("crn"))
		if err != nil {
			senderr(c, err, 500)
		}
	})
	return lect
}

var (
	// NoOp Defaults vary between databases
	// sqlite3:  -1
	// postgres: nil
	defaultLimit interface{} = nil

	defaultOffset interface{} = 0 // default to 0
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

func getFromCRN(db *sqlx.DB, query string, v interface{ Scan(models.Scanable) error }) gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			crn = c.GetInt("crn")
			row = db.QueryRow(query, crn)
			err = v.Scan(row)
		)
		if err == sql.ErrNoRows {
			c.JSON(404, &Error{
				Msg:    fmt.Sprintf("no results found for crn: %d", crn),
				Status: 404,
			})
			return
		}
		if err != nil {
			senderr(c, err, 500)
			return
		}
		c.JSON(200, v)
	}
}

func senderr(c *gin.Context, e error, status int) {
	c.JSON(
		status,
		&Error{
			Msg:    e.Error(),
			Status: status,
		},
	)
}
