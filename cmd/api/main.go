package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
	}

	db, err := opendb(conf)
	if err != nil {
		log.Fatal("Could not open db: ", err)
	}
	defer db.Close()

	gin.SetMode(config.GetString("mode"))
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/lectures", lectures(db))
	r.GET("/lecture/:crn", lecture(db))
	r.GET("/lecture/:crn/exam", exam(db))
	r.GET("/exams", exams(db))
	r.GET("/labs", labsdiscs(db))
	r.GET("/discussions", labsdiscs(db))
	r.GET("/instructor/:id", instructor(db))

	r.Any("/test", func(c *gin.Context) {
		resp := map[string]interface{}{
			"time":    time.Now(),
			"testing": true,
			"db":      conf.GetDSN(),
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
	defaultLimit interface{} = nil
	// always 0
	defaultOffset interface{} = 0
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

const (
	lecturesQuery = `
	SELECT ` +
		models.LectColumns + `
	FROM lectures
	LIMIT $1 OFFSET $2`
	labsQuery = `
	SELECT
		crn,
		course_num,
		section,
		title,
		units,
		activity,
		days,
		start_time,
		end_time,
		building_room,
		instructor_id
	FROM Labs_Discussion
	LIMIT $1 OFFSET $2`
	examsQuery = `
	SELECT
		crn,
		date,
		start_time,
		end_time
	FROM exam
	LIMIT $1 OFFSET $2`
)

func lectures(db *sql.DB) func(*gin.Context) {
	return func(c *gin.Context) {
		lectures := make([]interface{}, 0, 100)
		limit, offset := arrayOpts(c)
		rows, err := db.Query(lecturesQuery, limit, offset)
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

func labsdiscs(db *sql.DB) gin.HandlerFunc {
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

func exams(db *sql.DB) gin.HandlerFunc {
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

const (
	labQuery = `
	SELECT
		crn,
		course_num,
		section,
		title,
		units,
		activity,
		days,
		start_time,
		end_time,
		building_room,
		instructor_id
	FROM
		Labs_Discussion
	WHERE
		crn = $1`
	lectureQuery = `
	SELECT ` +
		models.LectColumns + `
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
		if err := v.Scan(row); err != nil {
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

func instructor(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := c.Params.Get("id")
		if !ok {
			c.JSON(http.StatusBadRequest, map[string]string{"error": "no instructor id"})
			return
		}
		var inst models.Instructor
		row := db.QueryRow(instructorQuery, id)
		if err := row.Scan(&inst.ID, &inst.Name); err != nil {
			senderr(c, err)
			return
		}
		c.JSON(200, &inst)
	}
}
