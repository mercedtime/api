package app

import (
	"database/sql"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/mercedtime/api/db/models"
)

// RegisterRoutes will setup all the app routes
func (a *App) RegisterRoutes(g *gin.RouterGroup) {
	a.instructorGroup(g)
	a.listsGroup(g)
	a.lectureGroup(g)
	a.userGroup(g)
}

func (a *App) userGroup(g *gin.RouterGroup) {
	g.POST("/user", a.PostUser)
	g.GET("/user/:id", idParamMiddleware, a.getUser)
	g.DELETE("/user/:id", idParamMiddleware, a.deleteUser)
}

func (a *App) instructorGroup(g *gin.RouterGroup) {
	g.GET("/instructor/:id", instructorFromID(a))
	g.GET("/instructor/:id/courses", instructorCourses(a.DB))
}

func (a *App) listsGroup(g *gin.RouterGroup) *gin.RouterGroup {
	lists := g.Group("/", listParamsMiddleware)
	lists.GET("/lectures", ListLectures(a.DB))
	lists.GET("/exams", ListExams(a.DB))
	lists.GET("/labs", ListLabs(a.DB))
	lists.GET("/discussions", ListDiscussions(a.DB))
	lists.GET("/instructors", ListInstructors(a.DB))
	return lists
}

// LectureGroup returns the router group for all the lecture routes.
func (a *App) lectureGroup(g *gin.RouterGroup) *gin.RouterGroup {
	lect := g.Group("/lecture", crnParamMiddleware)
	lect.GET("/:crn", lecture(a.DB))
	lect.GET("/:crn/exam", exam(a.DB))
	lect.GET("/:crn/labs", labsForLecture(a.DB))
	lect.GET("/:crn/instructor", instructorFromLectureCRN(a.DB))
	lect.DELETE("/:crn", func(c *gin.Context) {
		_, err := a.DB.Exec("DELETE FROM lectures WHERE crn = $1", c.MustGet("crn"))
		if err != nil {
			senderr(c, err, 500)
		}
	})
	return lect
}

func crnParamMiddleware(c *gin.Context) {
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
}

func idParamMiddleware(c *gin.Context) {
	idStr, ok := c.Params.Get("id")
	if !ok {
		c.JSON(400, &Error{
			Msg:    "no id given",
			Status: 400,
		})
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(400, &Error{Msg: "id is not a number"})
		return
	}
	c.Set("id", id)
	c.Next()
}

var (
	// NoOp Defaults vary between databases
	// sqlite3:  -1
	// postgres: nil
	defaultLimit interface{} = nil

	defaultOffset interface{} = 0 // default to 0
)

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
