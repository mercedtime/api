package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"regexp"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/harrybrwn/edu/school/ucmerced/ucm"
	"github.com/mercedtime/api/db/models"
)

var (
	// TimeFormat is a time format
	TimeFormat    = "15:04:05"
	mustAlsoRegex = regexp.MustCompile(`Must Also.*$`)
)

type genquery struct {
	Target     string
	Tmp        string
	AutoUpdate int
	Vars       []string
}

var (
	updateTmpl = `UPDATE {{ .Target }}{{ $n := sub (len .Vars) 1 }}
SET {{ range $i, $v := .Vars }}
  {{ $v }} = new.{{ $v }}{{ if ne $i $n }},{{ end }}
{{- end }}
  {{- if gt .AutoUpdate 0 -}}, auto_updated = {{ .AutoUpdate }},
  updated_at = now(){{ end }}
FROM (
  SELECT * FROM {{ .Tmp }} tmp
  WHERE NOT EXISTS (
    SELECT * FROM {{ .Target }} target
    WHERE
      {{- range $i, $v := .Vars }}
      tmp.{{ $v }} = target.{{ . }}{{ if ne $i $n }} AND{{end}}
      {{- end }}
  )
) new
WHERE {{ .Target }}.crn = new.crn`
)

func updatequery(data genquery) (string, error) {
	var buf bytes.Buffer
	tmpl, err := template.New("sql-update-gen").Funcs(
		template.FuncMap{"sub": func(a, b int) int { return a - b }},
	).Parse(updateTmpl)
	if err != nil {
		return "", err
	}
	err = tmpl.Execute(&buf, data)
	return buf.String(), err
}

func cleanTitle(title string) string {
	title = mustAlsoRegex.ReplaceAllString(title, "")
	title = strings.Replace(title, "Class is fully online", ": Class is fully online", -1)
	title = strings.Replace(title, "*Lab and Discussion Section Numbers Do Not Have to Match*", "", -1)
	return title
}

type execable interface {
	Exec(string, ...interface{}) (sql.Result, error)
}

func createTmpTable(from string, tx execable, tmp string, rows []interface{}) (drop func() error, err error) {
	var q string
	drop = func() error {
		_, e := tx.Exec("DROP TABLE " + tmp)
		return e
	}
	_, err = tx.Exec("SELECT * INTO " + tmp + " FROM " + from + " LIMIT 0")
	if err != nil {
		return
	}
	if len(rows) == 0 {
		return drop, nil
	}
	q, _, err = goqu.Insert(tmp).Rows(rows).ToSQL()
	if err != nil {
		return
	}
	if _, err = tx.Exec(q); err != nil {
		return
	}
	return drop, nil
}

// RawCourse is a raw course row
type RawCourse struct {
	CRN         int       `db:"crn" goqu:"skipupdate"`
	Subject     string    `db:"subject"`
	CourseNum   int       `db:"course_num"`
	Title       string    `db:"title"`
	Units       int       `db:"units"`
	Type        string    `db:"type"`
	Days        string    `db:"days"`
	StartTime   time.Time `db:"start_time"`
	EndTime     time.Time `db:"end_time"`
	StartDate   time.Time `db:"start_date"`
	EndDate     time.Time `db:"end_date"`
	Instructor  string    `db:"instructor"`
	Description string    `db:"description"`
	Capacity    int       `db:"capacity"`
	Enrolled    int       `db:"enrolled"`
	Remaining   int       `db:"remaining"`
}

func insertRawRow(table string, sch ucm.Schedule) (string, error) {
	var (
		rows = make([]RawCourse, sch.Len())
	)
	for i, c := range sch.Ordered() {
		rows[i] = RawCourse{
			CRN:         c.CRN,
			Subject:     c.Subject,
			CourseNum:   c.CourseNumber(),
			Title:       cleanTitle(c.Title),
			Units:       c.Units,
			Type:        c.Activity,
			Days:        str(c.Days),
			StartTime:   c.Time.Start,
			EndTime:     c.Time.End,
			StartDate:   c.Date.Start,
			EndDate:     c.Date.End,
			Instructor:  c.Instructor,
			Description: "",
			Capacity:    c.Capacity,
			Enrolled:    c.Enrolled,
			Remaining:   c.SeatsOpen(),
		}
	}
	q, _, err := goqu.Insert(table).Rows(rows).ToSQL()
	return q, err
}

func recordHistoricalEnrollment(db *sql.DB, year, termcode int, crs []*ucm.Course) error {
	var rows = make([]interface{}, 0, len(crs))
	for _, c := range crs {
		rows = append(rows, map[string]interface{}{
			"crn":      c.CRN,
			"year":     year,
			"term":     termcode,
			"enrolled": c.Enrolled,
			"capacity": c.Capacity,
			// column "ts" defaults to now()
		})
	}
	q, _, err := goqu.Insert("enrollment").Rows(rows).ToSQL()
	if err != nil {
		return err
	}
	_, err = db.Exec(q)
	return err
}

func updateCourseTable(db *sql.DB, crs []*ucm.Course) error {
	var (
		target   = "course"
		tmpTable = "_tmp_course"
		rows     = make([]interface{}, 0, len(crs))
	)
	courselist, err := GetCourseTable(crs, 200)
	if err != nil {
		return err
	}
	for _, c := range courselist {
		rows = append(rows, c)
	}

	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelDefault, ReadOnly: false,
	})
	if err != nil {
		return err
	}

	droptmp, err := createTmpTable(target, tx, tmpTable, rows)
	defer func() {
		e := droptmp()
		if e != nil && err == nil {
			err = e
		}
		if err == nil {
			err = tx.Commit()
		}
	}()
	if err != nil {
		return err
	}

	// New values
	// auto_updated = 1 for new rows
	q := "INSERT INTO " + target + `
	SELECT * FROM ` + tmpTable + ` tmp
	WHERE NOT EXISTS (
	  SELECT * FROM ` + target + ` c
	  WHERE c.CRN = tmp.CRN
	)`
	if _, err = tx.Exec(q); err != nil {
		return err
	}

	// auto_updated = 3 for enrollment count updates
	q, err = updatequery(genquery{
		Target:     "course",
		Tmp:        tmpTable,
		AutoUpdate: 3,
		Vars:       []string{"capacity", "enrolled", "remaining"},
	})
	if err != nil {
		return err
	}
	if _, err = tx.Exec(q); err != nil {
		return err
	}

	// auto_updated = 2 for generate updates
	q, err = updatequery(genquery{
		Target:     "course",
		Tmp:        tmpTable,
		AutoUpdate: 2,
		Vars: []string{
			"subject",
			"course_num",
			"type",
			"title",
			"description"},
	})
	if err != nil {
		return err
	}
	if _, err = tx.Exec(q); err != nil {
		return err
	}
	return err
}

func updateLectureTable(
	db *sql.DB,
	crs []*ucm.Course,
	instructors map[string]*instructorMeta,
) (err error) {
	lectures, err := getLectures(crs, instructors)
	if err != nil {
		return err
	}
	var rows = make([]interface{}, len(lectures))
	for i, l := range lectures {
		m := map[string]interface{}{
			"crn":           l.CRN,
			"units":         l.Units,
			"days":          l.Days,
			"start_time":    l.StartTime.Format(TimeFormat),
			"end_time":      l.EndTime.Format(TimeFormat),
			"start_date":    l.StartDate.Format(models.DateFormat),
			"end_date":      l.EndDate.Format(models.DateFormat),
			"instructor_id": l.InstructorID,
			"auto_updated":  1,
		}
		// rows = append(rows, m)
		rows[i] = m
	}
	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelDefault,
		ReadOnly:  false,
	})
	if err != nil {
		return err
	}
	droptmp, err := createTmpTable("Lectures", tx, "_tmp_lectures", rows)
	defer func() {
		e := droptmp()
		if e != nil && err == nil {
			err = e
		}
		if err == nil {
			err = tx.Commit()
		}
	}()
	if err != nil {
		return err
	}
	// New lectures
	q := `
	INSERT INTO Lectures
	SELECT * FROM _tmp_lectures tmp
	WHERE NOT EXISTS (
	  SELECT * FROM Lectures l
	  WHERE l.CRN = tmp.CRN
	)`
	if _, err = tx.Exec(q); err != nil {
		return err
	}
	// Updated lectures

	q, err = updatequery(genquery{
		Target:     "lectures",
		Tmp:        "_tmp_lectures",
		AutoUpdate: 2,
		Vars: []string{
			"units",
			"days",
			"start_time",
			"end_time",
			"start_date",
			"end_date",
			"instructor_id",
		},
	})
	if err != nil {
		return err
	}
	if _, err = tx.Exec(q); err != nil {
		return err
	}
	return err
}

func updateLabsTable(db *sql.DB, sch ucm.Schedule, instructors map[string]*instructorMeta) (err error) {
	var rows = make([]interface{}, 0, len(sch))
	for _, c := range sch.Ordered() {
		if c.Activity == models.Lect {
			continue
		}
		var lectCRN int
		lect, err := getDiscussionLecture(c, sch)
		if err == nil {
			lectCRN = lect.CRN
		} else {
			lectCRN = 0
		}
		instructorID := 0
		instructor, ok := instructors[c.Instructor]
		if !ok {
			fmt.Println("Could not find instructor")
		} else {
			instructorID = instructor.id
		}
		m := map[string]interface{}{
			"crn":           c.CRN,
			"course_crn":    lectCRN,
			"section":       c.Section,
			"units":         c.Units,
			"days":          str(c.Days),
			"start_time":    c.Time.Start.Format(TimeFormat),
			"end_time":      c.Time.End.Format(TimeFormat),
			"building_room": c.BuildingRoom,
			"instructor_id": instructorID,
			"auto_updated":  1,
		}
		rows = append(rows, m)
	}
	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelDefault,
		ReadOnly:  false,
	})
	if err != nil {
		return err
	}
	droptmp, err := createTmpTable("aux", tx, "_tmp_labs", rows)
	defer func() {
		e := droptmp()
		if e != nil && err == nil {
			err = e
		}
		if err == nil {
			err = tx.Commit()
		}
	}()
	if err != nil {
		return err
	}

	q := `
	INSERT INTO aux
	SELECT * FROM _tmp_labs tmp
	WHERE NOT EXISTS (
	  SELECT * FROM aux l
	  WHERE l.CRN = tmp.CRN
	)`
	if _, err = tx.Exec(q); err != nil {
		return err
	}

	q, err = updatequery(genquery{
		Target:     "aux",
		Tmp:        "_tmp_labs",
		AutoUpdate: 2,
		Vars: []string{
			"section",
			"units",
			"days",
			"start_time",
			"end_time",
			"building_room",
			"instructor_id",
		},
	})
	if err != nil {
		return err
	}
	_, err = tx.Exec(q)
	if err != nil {
		return err
	}
	return nil
}

func updateInstructorsTable(db *sql.DB, instructors map[string]*instructorMeta) (err error) {
	var rows = make([]interface{}, 0, len(instructors))
	for _, inst := range instructors {
		in := models.Instructor{
			ID:   inst.id,
			Name: inst.name,
		}
		rows = append(rows, map[string]interface{}{
			"id":   in.ID,
			"name": in.Name,
		})
	}
	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelDefault,
		ReadOnly:  false,
	})
	if err != nil {
		return err
	}
	droptmp, err := createTmpTable("instructor", tx, "tmp_inst", rows)
	defer func() {
		e := droptmp()
		if e != nil && err == nil {
			err = e
		}
		if err == nil {
			err = tx.Commit()
		}
	}()
	if err != nil {
		return err
	}
	// new instructors
	q := `
	INSERT INTO instructor
	SELECT * FROM tmp_inst tmp
	WHERE NOT EXISTS (
	  SELECT * FROM instructor l WHERE l.id = tmp.id
	)`
	if _, err = tx.Exec(q); err != nil {
		return err
	}
	return nil
}

func updateExamTable(db *sql.DB, courses []*ucm.Course) error {
	var rows = make([]interface{}, 0, len(courses))
	for _, c := range courses {
		if c.Exam == nil {
			continue
		}
		e := &models.Exam{
			CRN:       c.CRN,
			Date:      c.Exam.Date,
			StartTime: c.Time.Start,
			EndTime:   c.Time.Start,
		}
		rows = append(rows, map[string]interface{}{
			"crn":        e.CRN,
			"date":       e.Date.Format(models.DateFormat),
			"start_time": e.StartTime.Format(TimeFormat),
			"end_time":   e.EndTime.Format(TimeFormat),
		})
	}
	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelDefault,
		ReadOnly:  false,
	})
	if err != nil {
		return err
	}
	droptmp, err := createTmpTable("exam", tx, "_tmp_exam", rows)
	defer func() {
		e := droptmp()
		if e != nil && err == nil {
			err = e
		}
		if err == nil {
			err = tx.Commit()
		}
	}()
	if err != nil {
		return err
	}
	q := `
	INSERT INTO exam
	SELECT * FROM _tmp_exam
	WHERE crn NOT IN (
	  SELECT crn FROM exam
	)`
	if _, err = tx.Exec(q); err != nil {
		return err
	}

	q, err = updatequery(genquery{
		Target: "exam",
		Tmp:    "_tmp_exam",
		Vars:   []string{"date", "start_time", "end_time"},
	})
	if err != nil {
		return err
	}
	if _, err = tx.Exec(q); err != nil {
		return err
	}
	return err
}
