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

	r.GET("/lectures", listLectures(db))
	r.GET("/lecture/:crn", lecture(db))
	r.GET("/lecture/:crn/exam", exam(db))
	r.GET("/lecture/:crn/instructor", instructorFromLectureCRN(db))
	r.GET("/lecture/:crn/enrollment", func(c *gin.Context) { c.Status(http.StatusNotImplemented) })
	r.GET("/exams", listExams(db))
	r.GET("/labs", listLabsDiscs(db))
	r.GET("/discussions", listLabsDiscs(db))
	r.GET("/instructor/:id", instructorFromID(db))
	r.GET("/instructors", listInstructors(db))

	r.Any("/test", func(c *gin.Context) {
		resp := map[string]interface{}{
			"time":    time.Now(),
			"testing": true,
			"mode":    conf.Mode,
			"db": map[string]interface{}{
				"dsn":    conf.GetDSN(),
				"driver": conf.Database.Driver,
			},
		}
		log.Println(c.Keys)
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
		SELECT ` + strings.Join(models.GetSchema(models.Lect{}), ",") + `
		FROM lectures
		LIMIT $1 OFFSET $2`
	lecturesBySubjectQuery = `
		SELECT l.` + strings.Join(models.GetSchema(models.Lect{}), ",l.") + `
		FROM lectures l, course c
		WHERE
			l.crn = c.crn AND
			c.subject = $1
		LIMIT $2 OFFSET $3`
	labsQuery = `
		SELECT
			` + strings.Join(models.GetSchema(models.LabDisc{}), ",") + `
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
			fmt.Println(query)
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

func listLabsDiscs(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		list := make([]interface{}, 0, 100)
		limit, offset := arrayOpts(c)
		rows, err := db.Query(labsQuery, limit, offset)
		if err != nil {
			senderr(c, err)
			return
		}
		for rows.Next() {
			l := models.LabDisc{}
			if err = l.Scan(rows); err != nil {
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

func listExams(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var examList = make([]interface{}, 0, 10)
		limit, offset := arrayOpts(c)
		rows, err := db.Query(examsQuery, limit, offset)
		if err != nil {
			senderr(c, err)
			return
		}
		for rows.Next() {
			var e models.Exam
			err = e.Scan(rows)
			if err != nil {
				senderr(c, err)
				return
			}
			examList = append(examList, e)
		}
		if err = rows.Close(); err != nil {
			senderr(c, err)
			return
		}
		c.JSON(200, examList)
	}
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
		crn, ok := c.Params.Get("crn")
		if !ok {
			c.JSON(http.StatusBadRequest, map[string]string{"error": "no crn"})
			return
		}
		row := db.QueryRow(query, crn)
		err := v.Scan(row)
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
		crn, ok := c.Params.Get("crn")
		if !ok {
			c.JSON(http.StatusBadRequest, map[string]string{"error": "no crn"})
			return
		}
		insts, err := getLectureInstructors(db, crn)
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
