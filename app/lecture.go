package app

import (
	"database/sql"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/mercedtime/api/db/models"
)

// ListInstructors returns a handler func that
// lists the isntructors in the database. Requires that
// limit and offset have been set in middleware before this
// is called.
func ListInstructors(db *sqlx.DB) gin.HandlerFunc {
	var list []models.Instructor
	return func(c *gin.Context) {
		list = nil
		err := db.Select(
			&list,
			"SELECT * FROM instructor LIMIT $1 OFFSET $2",
			c.MustGet("limit"), c.MustGet("offset"),
		)
		if err != nil {
			c.JSON(500, map[string]interface{}{"error": err})
			return
		}
		c.JSON(200, list)
	}
}

func newlect(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var l models.Lecture
		if err := c.BindJSON(&l); err != nil {
			log.Println(err)
		}
		_, err := db.Exec(`
		INSERT INTO lectures (crn,units,activity, days)
		VALUES ($1,$2,$3,$4,$5,$6)`, l.CRN, l.Units, "LECT", l.Days)
		if err != nil {
			senderr(c, err, 500)
			return
		}
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
			` + strings.Join(models.GetSchema(models.Lecture{}), ",") + `
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
)

func labsForLecture(db *sqlx.DB) gin.HandlerFunc {
	var (
		err   error
		crn   interface{}
		query = `
		  SELECT
		  ` + strings.Join(models.GetNamedSchema("a", models.LabDisc{}), ",") + `
		  FROM aux a, lectures l
		  WHERE
	  	    a.course_crn = l.crn AND
	  	    l.crn = $1`
	)
	return func(c *gin.Context) {
		var list []models.LabDisc
		crn = c.MustGet("crn")
		err = db.Select(&list, query, crn)
		if err != nil {
			senderr(c, err, 500)
		}
		c.JSON(200, list)
	}
}

func getLab(db *sqlx.DB) gin.HandlerFunc {
	var l models.LabDisc
	return getFromCRN(db, labQuery, &l)
}

func lecture(db *sqlx.DB) gin.HandlerFunc {
	var l models.Lecture
	return getFromCRN(db, lectureQuery, &l)
}

func exam(db *sqlx.DB) gin.HandlerFunc {
	var e models.Exam
	return getFromCRN(db, examQuery, &e)
}

func getLectureInstructors(db *sqlx.DB, crn int) ([]*models.Instructor, error) {
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

func instructorFromLectureCRN(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		insts, err := getLectureInstructors(db, c.GetInt("crn"))
		if err == sql.ErrNoRows {
			c.JSON(404, &Error{
				Msg: "could not find that instructor",
			})
			return
		}
		if err != nil {
			senderr(c, err, 500)
			return
		}
		c.JSON(200, insts)
	}
}
