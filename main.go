package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	// database drivers
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"github.com/gin-gonic/gin"
	"github.com/harrybrwn/config"

	"github.com/mercedtime/api/app"
	"github.com/mercedtime/api/models"
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
	r.GET("/instructor/:id", instructor(db))

	addr := app.Address()
	fmt.Printf("\n\nRunning on \x1b[32;4mhttp://%s\x1b[0m\n", addr)

	r.Run(addr)
}

func arrayOpts(c *gin.Context) (limit, offset interface{}) {
	limit = nil
	offset = 0
	if lim, ok := c.GetQuery("limit"); ok {
		limit = lim
	}
	if off, ok := c.GetQuery("offset"); ok {
		offset = off
	}
	return limit, offset
}

func senderr(c *gin.Context, e error) {
	c.JSON(http.StatusInternalServerError, map[string]string{"error": e.Error()})
}

func lectures(db *sql.DB) func(*gin.Context) {
	var lectures []models.Lect
	return func(c *gin.Context) {
		lectures = make([]models.Lect, 0, 100)

		limit, offset := arrayOpts(c)
		rows, err := db.Query(
			"SELECT "+models.LectColumns+" FROM lectures OFFSET $1 LIMIT $2", offset, limit)
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
			lectures = append(lectures, l)
		}
		if err = rows.Close(); err != nil {
			senderr(c, err)
			return
		}
		c.JSON(200, lectures)
	}
}

func lecture(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var l models.Lect
		crn, ok := c.Params.Get("crn")
		if !ok {
			c.JSON(http.StatusBadRequest, map[string]string{"error": "no crn"})
			return
		}

		row := db.QueryRow(
			"SELECT "+models.LectColumns+" FROM lectures WHERE crn = $1",
			crn,
		)
		if err := l.Scan(row); err != nil {
			senderr(c, err)
			return
		}
		c.JSON(200, &l)
	}
}

func exams(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var examList = make([]models.Exam, 0, 10)
		limit, offset := arrayOpts(c)
		rows, err := db.Query(
			`SELECT crn, date, start_time, end_time
			FROM exam LIMIT $1 OFFSET $2`, limit, offset)
		if err != nil {
			senderr(c, err)
			return
		}
		for rows.Next() {
			var e models.Exam
			if err := rows.Scan(&e.CRN, &e.Date, &e.StartTime, &e.EndTime); err != nil {
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

func exam(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var e models.Exam
		crn, ok := c.Params.Get("crn")
		if !ok {
			c.JSON(http.StatusBadRequest, map[string]string{"error": "no crn"})
			return
		}
		row := db.QueryRow("SELECT crn, date, start_time, end_time FROM exam WHERE crn = $1", crn)
		if err := row.Scan(&e.CRN, &e.Date, &e.StartTime, &e.EndTime); err != nil {
			senderr(c, err)
			return
		}
		c.JSON(200, &e)
	}
}

func instructor(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := c.Params.Get("id")
		if !ok {
			c.JSON(http.StatusBadRequest, map[string]string{"error": "no instructor id"})
			return
		}
		var inst models.Instructor
		row := db.QueryRow("SELECT id, name FROM instructor WHERE id = $1", id)
		if err := row.Scan(&inst.ID, &inst.Name); err != nil {
			senderr(c, err)
			return
		}
		c.JSON(200, &inst)
	}
}
