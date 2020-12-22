package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/harrybrwn/edu/school/ucmerced/ucm"
	"github.com/mercedtime/api/catalog"
	"github.com/mercedtime/api/db/models"
)

var (
	// TimeFormat is a time format
	TimeFormat    = "15:04:05"
	mustAlsoRegex = regexp.MustCompile(`Must Also.*$`)
)

type genquery struct {
	Target string
	Tmp    string
	Vars   []string

	SetUpdated bool
	AutoUpdate int
}

var (
	updateTmpl = `UPDATE {{ .Target }}{{ $n := sub (len .Vars) 1 }}
SET {{ range $i, $v := .Vars }}
  {{ $v }} = new.{{ $v }}{{ if ne $i $n }},{{ end }}
{{- end }}
  {{- if gt .AutoUpdate 0 -}},auto_updated = {{ .AutoUpdate }}{{ end }}
  {{- if .SetUpdated }},updated_at = now(){{ end }}
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

func insertNew(target, tmp string, tx *sql.Tx) error {
	q := fmt.Sprintf(`INSERT INTO %[1]s
	SELECT * FROM %[2]s tmp WHERE NOT EXISTS (SELECT * FROM %[1]s c WHERE c.crn = tmp.crn)`, target, tmp)
	_, err := tx.Exec(q)
	return err
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

// TODO:
// - This has room for optimization.
//		* Temp tables droped after transaction (https://www.postgresql.org/docs/9.3/sql-createtable.html)
// 		* https://www.postgresql.org/docs/current/sql-selectinto.html
// 		* https://www.postgresql.org/docs/current/sql-createtableas.html
// - Also accept database and create transaction here (or do it from another function)
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
		})
	}
	q, _, err := goqu.Insert("enrollment").Rows(rows).ToSQL()
	if err != nil {
		return err
	}
	_, err = db.Exec(q)
	return err
}

func updateCourseTable(db *sql.DB, courses []*catalog.Entry) error {
	var (
		target   = "course"
		tmpTable = "_tmp_" + target
		rows     = make([]interface{}, 0, len(courses))
	)
	for _, c := range courses {
		if c.Description != "" {
			rows = append(rows, c)
		}
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
		} else {
			log.Println(err)
		}
	}()
	if err != nil {
		return err
	}
	if err = insertNew(target, tmpTable, tx); err != nil {
		return err
	}

	// auto_updated = 3 for enrollment count updates
	q, err := updatequery(genquery{
		Target:     target,
		Tmp:        tmpTable,
		SetUpdated: true,
		AutoUpdate: 3,
		Vars: []string{
			"capacity",
			"enrolled",
			"remaining"},
	})
	if err != nil {
		return err
	}
	if _, err = tx.Exec(q); err != nil {
		return err
	}

	// auto_updated = 2 for generate updates
	q, err = updatequery(genquery{
		Target:     target,
		Tmp:        tmpTable,
		SetUpdated: true,
		AutoUpdate: 2,
		Vars: []string{
			"subject",
			"course_num",
			"type",
			"units",
			"days",
			"title",
			"description",
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

func updateLectureTable(
	db *sql.DB,
	lectures []*models.Lecture,
) (err error) {
	var (
		target   = "lectures"
		tmpTable = "_tmp_" + target
		rows     = make([]interface{}, len(lectures))
	)
	for i, l := range lectures {
		m := map[string]interface{}{
			"crn":           l.CRN,
			"start_time":    l.StartTime.Format(TimeFormat),
			"end_time":      l.EndTime.Format(TimeFormat),
			"start_date":    l.StartDate.Format(models.DateFormat),
			"end_date":      l.EndDate.Format(models.DateFormat),
			"instructor_id": l.InstructorID,
			"auto_updated":  1,
		}
		rows[i] = m
	}
	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelDefault,
		ReadOnly:  false,
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
		} else {
			log.Println(err)
		}
	}()
	if err != nil {
		return err
	}
	// New lectures
	q := fmt.Sprintf(`
	INSERT INTO %[1]s
	SELECT * FROM %[2]s tmp
	WHERE NOT EXISTS (
	  SELECT * FROM %[1]s target
	  WHERE target.CRN = tmp.CRN
	)`, target, tmpTable)
	if _, err = tx.Exec(q); err != nil {
		return err
	}
	// Updated lectures

	q, err = updatequery(genquery{
		Target:     target,
		Tmp:        tmpTable,
		SetUpdated: true,
		AutoUpdate: 2,
		Vars: []string{
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

func updateLabsTable(
	db *sql.DB,
	labs []*models.LabDisc,
) (err error) {
	var (
		target   = "aux"
		tmpTable = "_tmp_" + target
		rows     = make([]interface{}, len(labs))
	)
	for i, l := range labs {
		rows[i] = map[string]interface{}{
			"crn":           l.CRN,
			"course_crn":    l.CourseCRN,
			"section":       l.Section,
			"start_time":    l.StartTime.Format(TimeFormat),
			"end_time":      l.EndTime.Format(TimeFormat),
			"building_room": l.Building,
			"instructor_id": l.InstructorID,
			"auto_updated":  1,
		}
	}

	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelDefault,
		ReadOnly:  false,
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
		} else {
			log.Println(err)
		}
	}()
	if err != nil {
		return err
	}
	if err = insertNew(target, tmpTable, tx); err != nil {
		return err
	}

	q, err := updatequery(genquery{
		Target:     target,
		Tmp:        tmpTable,
		SetUpdated: true,
		AutoUpdate: 2,
		Vars: []string{
			"section",
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
	var (
		target   = "instructor"
		tmpTable = "_tmp_" + target
		rows     = make([]interface{}, 0, len(instructors))
	)
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
	droptmp, err := createTmpTable(target, tx, tmpTable, rows)
	defer func() {
		e := droptmp()
		if e != nil && err == nil {
			err = e
		}
		if err == nil {
			err = tx.Commit()
		} else {
			log.Println(err)
		}
	}()
	if err != nil {
		return err
	}
	// new instructors
	q := fmt.Sprintf(`
	INSERT INTO %[1]s
	SELECT * FROM %[2]s tmp
	WHERE NOT EXISTS (
	  SELECT * FROM %[1]s target
	  WHERE target.id = tmp.id
	)`, target, tmpTable)
	if _, err = tx.Exec(q); err != nil {
		return err
	}
	return nil
}

func updateExamTable(db *sql.DB, exams []*models.Exam) error {
	var (
		target   = "exam"
		tmpTable = "_tmp_" + target
		rows     = make([]interface{}, len(exams))
	)
	for i, e := range exams {
		rows[i] = map[string]interface{}{
			"crn":        e.CRN,
			"date":       e.Date.Format(models.DateFormat),
			"start_time": e.StartTime.Format(TimeFormat),
			"end_time":   e.EndTime.Format(TimeFormat),
		}
	}

	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelDefault,
		ReadOnly:  false,
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
		} else {
			log.Println(err)
		}
	}()
	if err != nil {
		return err
	}
	if err = insertNew(target, tmpTable, tx); err != nil {
		return err
	}

	q, err := updatequery(genquery{
		Target:     target,
		Tmp:        tmpTable,
		SetUpdated: false,
		Vars:       []string{"date", "start_time", "end_time"},
	})
	if err != nil {
		return err
	}
	if _, err = tx.Exec(q); err != nil {
		return err
	}
	return err
}
