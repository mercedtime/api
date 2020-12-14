package app

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/mercedtime/api/db/models"
	"github.com/mercedtime/api/users"
)

// RegisterRoutes will setup all the app routes
func (a *App) RegisterRoutes(g *gin.RouterGroup) {
	g.POST("/user", a.PostUser)
	g.GET("/instructor/:id", instructorFromID(a.DB))
	g.GET("/instructor/:id/courses", instructorCourses(a.DB))

	lists := g.Group("/", listParamsMiddleware)
	lists.GET("/lectures", ListLectures(a.DB))
	lists.GET("/exams", ListExams(a.DB))
	lists.GET("/labs", ListLabs(a.DB))
	lists.GET("/discussions", ListDiscussions(a.DB))
	lists.GET("/instructors", ListInstructors(a.DB))
}

// PostUser handles user creation
func (a *App) PostUser(c *gin.Context) {
	type user struct {
		users.User
		Password string
	}
	u := user{}
	err := c.BindJSON(&u)
	if err != nil {
		c.JSON(500, NewErr("could not read body"))
		return
	}

	// TODO check auth for permissions to set is_admin
	u.IsAdmin = false

	if u.Password == "" {
		c.JSON(400, ErrStatus(400, "no password for new user"))
		return
	}
	if _, err = a.CreateUser(&u.User, u.Password); err != nil {
		senderr(c, err, 500)
		return
	}
	c.JSON(200, u.User)
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
	log.Println(e)
	c.JSON(
		status,
		&Error{
			Msg:    e.Error(),
			Status: status,
		},
	)
}
