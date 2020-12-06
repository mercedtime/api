package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	// database drivers

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"github.com/gin-gonic/gin"
	"github.com/harrybrwn/config"

	"github.com/mercedtime/api/app"
	"github.com/mercedtime/api/db/models"
)

func opendb(conf *app.Config) (*sql.DB, error) {
	db, err := sql.Open(config.GetString("database.driver"), conf.GetDSN())
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, err
}

var isSqlite = false

func main() {
	var conf = &app.Config{}
	config.SetFilename("api.yml")
	config.SetType("yml")
	config.AddPath(".")
	config.SetConfig(conf)
	if err := config.ReadConfigFile(); err != nil {
		log.Println("Warning:", err)
	}
	if err := conf.Init(); err != nil {
		log.Fatal(err)
	}

	if conf.Database.Driver == "sqlite3" {
		defaultLimit = -1
		models.TimeFormat = models.SQLiteTimeFormat
		isSqlite = true
	}

	db, err := opendb(conf)
	if err != nil {
		log.Fatal("Could not open db: ", err)
	}
	defer db.Close()

	gin.SetMode(config.GetString("mode"))
	r := gin.New()
	r.Use(gin.Recovery(), gin.LoggerWithConfig(gin.LoggerConfig{
		Formatter: func(f gin.LogFormatterParams) string {
			return fmt.Sprintf(
				"[%s] %6v %s%d%s %s %s\n",
				f.TimeStamp.Format(time.Stamp),
				f.Latency,
				statusColor(f.StatusCode), f.StatusCode, "\x1b[0m",
				f.Method,
				f.Path,
			)
		},
	}))

	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, map[string]interface{}{
			"error":  "no route for " + c.Request.URL.Path,
			"status": 404,
		})
	})

	listparams := func(c *gin.Context) {
		limit, offset := arrayOpts(c)
		c.Set("limit", limit)
		c.Set("offset", offset)
		subj, ok := c.GetQuery("subject")
		if ok {
			c.Set("subject", subj)
		}
		c.Next()
	}

	r.GET("/lectures", listparams, listLectures(db))
	r.GET("/exams", listparams, listExams(db))
	r.GET("/labs", listparams, listLabsDiscs(db))
	r.GET("/discussions", listparams, listLabsDiscs(db))
	r.GET("/instructors", listparams, listInstructors(db))

	lect := r.Group("/lecture", func(c *gin.Context) {
		crn, ok := c.Params.Get("crn")
		if !ok {
			c.JSON(400, map[string]string{"error": "no crn given", "status": "bad request"})
			return
		}
		c.Set("crn", crn)
		c.Next()
	})
	lect.GET("/:crn", lecture(db))
	lect.GET("/:crn/exam", exam(db))
	lect.GET("/:crn/labs", labsForLecture(db))
	lect.GET("/:crn/instructor", instructorFromLectureCRN(db))
	lect.GET("/:crn/enrollment", func(c *gin.Context) {
		c.Status(http.StatusNotImplemented)
	})

	r.GET("/instructor/:id", instructorFromID(db))
	r.GET("/instructor/:id/lectures")
	r.GET("/instructor/:id/labs")
	r.GET("/instructor/:id/discussions")

	r.Any("/test", func(c *gin.Context) {
		resp := map[string]interface{}{
			"time":    time.Now(),
			"testing": true,
			"mode":    conf.Mode,
			"db": map[string]interface{}{
				// "dsn":    conf.GetDSN(),
				"driver": conf.Database.Driver,
			},
		}
		b, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			c.Status(500)
			log.Println(err)
			return
		}
		c.Status(200)
		c.Writer.Write(b)
	})

	addr := app.Address()
	fmt.Printf("\n\nRunning on \x1b[32;4mhttp://%s\x1b[0m\n", addr)

	r.Run(addr)
}

var (
	// sqlite3:  -1
	// postgres: nil
	defaultLimit  interface{} = nil
	defaultOffset interface{} = 0 // always 0
)

func arrayOpts(c *gin.Context) (limit, offset interface{}) {
	limit = defaultLimit
	offset = defaultOffset
	if lim, ok := c.GetQuery("limit"); ok && lim != "" {
		limit = lim
	}
	if off, ok := c.GetQuery("offset"); ok && off != "" {
		offset = off
	}
	return limit, offset
}

func senderr(c *gin.Context, e error) {
	log.Println(e)
	c.JSON(http.StatusInternalServerError, map[string]string{"error": e.Error()})
}

func listquery(q string, db *sql.DB, c *gin.Context) (*sql.Rows, error) {
	limit, offset := arrayOpts(c)
	return db.Query(lecturesQuery, limit, offset)
}

var (
	lecturesQuery = `
		SELECT ` + strings.Join(models.GetNamedSchema("lectures", models.Lect{}), ",") + `
		FROM lectures
		LIMIT $1 OFFSET $2`
	lecturesBySubjectQuery = `
		SELECT l.` + strings.Join(models.GetSchema(models.Lect{}), ",l.") + `
		FROM lectures l, course c
		WHERE
			l.crn = c.crn AND
			c.subject = $1
		LIMIT $2 OFFSET $3`
	lectLabsQuery = `SELECT
		` + strings.Join(models.GetNamedSchema("lab", models.LabDisc{}), ",") + `
	FROM Labs_Discussions lab, lectures l
	WHERE
		l.crn = $1 AND
		lab.course_crn = l.crn`

	courseTAsQuery = `SELECT i.id,i.name
	FROM
		instructor i, lectures l, labs_discussions lab
	WHERE
		l.crn = $1 AND
		l.crn = lab.course_crn AND
		lab.instructor_id = i.id`

	labsQuery = `
		SELECT
			` + strings.Join(models.GetNamedSchema("labs_discussions", models.LabDisc{}), ",") + `
		FROM Labs_Discussions LIMIT $1 OFFSET $2`
	examsQuery = `
		SELECT
			crn,
			date,
			start_time,
			end_time
		FROM exam
		LIMIT $1 OFFSET $2`
)

func listLectures(db *sql.DB) func(*gin.Context) {
	var query = lecturesQuery
	return func(c *gin.Context) {
		var (
			rows     *sql.Rows
			err      error
			lectures = make([]interface{}, 0, 100)
		)
		limit, offset := arrayOpts(c)

		subject, ok := c.GetQuery("subject")
		if ok {
			query = lecturesBySubjectQuery
			rows, err = db.Query(query, strings.ToUpper(subject), limit, offset)
		} else {
			rows, err = db.Query(query, limit, offset)
		}
		if err != nil {
			senderr(c, err)
			return
		}
		for rows.Next() {
			l := models.Lect{}
			if err = l.Scan(rows); err != nil {
				senderr(c, err)
				return
			}
			lectures = append(lectures, &l)
		}
		if err = rows.Close(); err != nil {
			senderr(c, err)
			return
		}
		c.JSON(200, lectures)
	}
}

func labsForLecture(db *sql.DB) gin.HandlerFunc {
	scan := func(r *sql.Rows) (interface{}, error) {
		l := models.LabDisc{}
		return &l, l.Scan(r)
	}
	return getList(db, lectLabsQuery, scan, "crn")
}

func getList(db *sql.DB, query string, scan func(*sql.Rows) (interface{}, error), keys ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			list      = make([]interface{}, 0, 100)
			queryargs = make([]interface{}, 0, len(keys))
		)
		for _, k := range keys {
			arg, ok := c.Get(k)
			if ok {
				queryargs = append(queryargs, arg)
			}
		}
		rows, err := db.Query(query, queryargs...)
		if err != nil {
			senderr(c, err)
			return
		}
		for rows.Next() {
			l, err := scan(rows)
			if err != nil {
				senderr(c, err)
				return
			}
			list = append(list, l)
		}
		if err = rows.Close(); err != nil {
			senderr(c, err)
			return
		}
		c.JSON(200, list)
	}
}

func listLabsDiscs(db *sql.DB) gin.HandlerFunc {
	scan := func(r *sql.Rows) (interface{}, error) {
		l := models.LabDisc{}
		return &l, l.Scan(r)
	}
	return getList(db, labsQuery, scan, "limit", "offset")
}

func listExams(db *sql.DB) gin.HandlerFunc {
	scan := func(r *sql.Rows) (interface{}, error) {
		e := models.Exam{}
		return &e, e.Scan(r)
	}
	return getList(db, examsQuery, scan, "limit", "offset")
}

var (
	labQuery = `
		SELECT
			` + strings.Join(models.GetSchema(models.LabDisc{}), ",") + `
		FROM
			Labs_Discussion
		WHERE crn = $1`
	lectureQuery = `
		SELECT
			` + strings.Join(models.GetSchema(models.Lect{}), ",") + `
		FROM
			lectures
		WHERE crn = $1`
	examQuery = `
		SELECT
			crn,
			date,
			start_time,
			end_time
		FROM exam
		WHERE crn = $1`
	instructorQuery = `
		SELECT
			id,
			name
		FROM instructor
		WHERE id = $1`
)

func getFromCRN(db *sql.DB, query string, v interface{ Scan(models.Scanable) error }) gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			crn = c.GetString("crn")
			row = db.QueryRow(query, crn)
			err = v.Scan(row)
		)
		if err == sql.ErrNoRows {
			c.JSON(404, map[string]interface{}{
				"error":  "no results found for crn: " + crn,
				"status": 404,
			})
			return
		}
		if err != nil {
			senderr(c, err)
			return
		}
		c.JSON(200, v)
	}
}

func getLab(db *sql.DB) gin.HandlerFunc {
	var l models.LabDisc
	return getFromCRN(db, labQuery, &l)
}

func lecture(db *sql.DB) gin.HandlerFunc {
	var l models.Lect
	return getFromCRN(db, lectureQuery, &l)
}

func exam(db *sql.DB) gin.HandlerFunc {
	var e models.Exam
	return getFromCRN(db, examQuery, &e)
}

func instructor(db *sql.DB, id string) gin.HandlerFunc {
	var query = instructorQuery
	return func(c *gin.Context) {
		var inst models.Instructor
		row := db.QueryRow(query, id)
		if err := row.Scan(&inst.ID, &inst.Name); err != nil {
			senderr(c, err)
			return
		}
		c.JSON(200, &inst)
	}
}

func instructorFromID(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := c.Params.Get("id")
		if !ok {
			c.JSON(http.StatusBadRequest, map[string]string{"error": "no instructor id"})
			return
		}
		instructor(db, id)(c)
	}
}

func getLectureInstructors(db *sql.DB, crn string) ([]*models.Instructor, error) {
	var (
		insts = make([]*models.Instructor, 0)
		query = `
		SELECT id, name
		FROM lectures, instructor
		WHERE instructor_id = id AND crn = $1`
	)
	rows, err := db.Query(query, crn)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		inst := &models.Instructor{}
		err = rows.Scan(&inst.ID, &inst.Name)
		if err != nil {
			return insts, err
		}
		insts = append(insts, inst)
	}
	if err = rows.Close(); err != nil {
		return nil, err
	}
	return insts, nil
}

func instructorFromLectureCRN(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		insts, err := getLectureInstructors(db, c.GetString("crn"))
		if err != nil {
			senderr(c, err)
			return
		}
		c.JSON(200, insts)
	}
}

func listInstructors(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, map[string]string{"error": "not implemented"})
	}
}

func statusColor(status int) string {
	var id int
	if status == 0 {
		id = 0
	} else if status < 300 {
		id = 32
	} else if status < 400 {
		id = 34
	} else if status < 500 {
		id = 33
	} else if status < 600 {
		id = 31
	} else {
		status = 0
	}
	return fmt.Sprintf("\033[%d;1m", id)
}
