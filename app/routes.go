package app

import (
	"database/sql"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// RegisterRoutes will setup all the app routes
func (a *App) RegisterRoutes(g *gin.RouterGroup) {
	a.lectureGroup(g)

	// utility endpoints
	g.GET("/subjects", a.subjects)
	g.GET("/terms", a.availbleTerms)

	lists := g.Group("/", listParamsMiddleware)
	lists.GET("/lectures", ListLectures(a.DB))
	lists.GET("/exams", ListExams(a.DB))
	lists.GET("/labs", ListLabs(a.DB))
	lists.GET("/discussions", ListDiscussions(a.DB))
	lists.GET("/instructors", ListInstructors(a.DB))
	g.GET("/courses", termyearMiddle, a.listCourses)
	lists.GET("/catalog/:year/:term", termyearMiddle, getCatalog(a.DB))

	ugroup := g.Group("/user")
	ugroup.POST("/", a.PostUser)
	ugroup.GET("/:id", idParamMiddleware, a.getUser)
	ugroup.DELETE("/:id", idParamMiddleware, a.deleteUser)

	inst := g.Group("/instructor")
	inst.GET("/:id", instructorFromID(a))
	inst.GET("/:id/courses", instructorCourses(a.DB))
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

func (a *App) subjects(c *gin.Context) {
	type response struct {
		Code string `json:"code" db:"code"`
		Name string `json:"name" db:"name"`
	}
	resp := make([]response, 0, 10)
	err := a.DB.Select(&resp, "select code,name from subject")
	if err != nil {
		c.JSON(500, Error{Msg: "could not get subjects"})
		return
	}
	c.JSON(200, resp)
}

func (a *App) availbleTerms(c *gin.Context) {
	type response struct {
		Year     int    `db:"year" json:"year"`
		Term     int    `db:"term_id" json:"id"`
		TermName string `db:"name" json:"name"`
	}
	resp := make([]response, 0, 5)
	err := a.DB.Select(
		&resp,
		`SELECT year, term_id, term.name
		   FROM course
		   JOIN term ON term.id = term_id
	   GROUP BY year, term_id, term.name`,
	)
	if err != nil {
		c.JSON(500, Error{Msg: "could not get availible terms"})
		return
	}
	c.JSON(200, resp)
}

func getTermID(term string) int {
	switch term {
	case "spring":
		return 1
	case "summer":
		return 2
	case "fall":
		return 3
	default:
		return 0
	}
}

func setTerm(c *gin.Context, term string) {
	id := getTermID(term)
	if id == 0 {
		c.AbortWithStatusJSON(400, &Error{"invalid term", 400})
		return
	}
	c.Set("term", id)
}

func setYear(c *gin.Context, year string) {
	yeari, err := strconv.ParseInt(year, 10, 32)
	if err != nil {
		c.AbortWithStatusJSON(400, &Error{"not a valid year", 400})
		return
	}
	c.Set("year", yeari)
	return
}

func termyearMiddle(c *gin.Context) {
	var (
		err error
		p   = SemesterParams{}
	)
	if err = p.bind(c); err != nil {
		c.AbortWithStatusJSON(400, &Error{
			Msg:    "bad parameters: " + err.Error(),
			Status: 400,
		})
		return
	}

	if p.Subject != "" {
		c.Set("subject", strings.ToUpper(p.Subject))
	}
	if p.Year != 0 {
		c.Set("year", p.Year)
	}
	if p.Term != "" {
		setTerm(c, p.Term)
	}
	c.Next()
}

func crnParamMiddleware(c *gin.Context) {
	crnStr, ok := c.Params.Get("crn")
	if !ok {
		c.AbortWithStatusJSON(400, ErrStatus(400, "no crn given"))
		return
	}
	crn, err := strconv.Atoi(crnStr)
	if err != nil {
		c.AbortWithStatusJSON(400, &Error{Msg: "crn given is not a number"})
		return
	}
	c.Set("crn", crn)
	c.Next()
}

func idParamMiddleware(c *gin.Context) {
	idStr, ok := c.Params.Get("id")
	if !ok {
		c.AbortWithStatusJSON(400, &Error{
			Msg:    "no id given",
			Status: 400,
		})
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.AbortWithStatusJSON(400, &Error{Msg: "id is not a number"})
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

func getFromCRN(db *sqlx.DB, query string, v interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := db.Get(v, query, c.GetInt("crn"))
		if err == sql.ErrNoRows {
			c.JSON(404, &Error{"could not find exam", 404})
			return
		}
		if err != nil {
			c.JSON(500, Error{err.Error(), 500})
			return
		}
		c.JSON(200, v)
	}
}

func senderr(c *gin.Context, e error, status int) {
	c.AbortWithStatusJSON(
		status,
		&Error{
			Msg:    strings.Replace(e.Error(), "\"", "'", -1),
			Status: status,
		},
	)
}
