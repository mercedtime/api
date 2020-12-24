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

	"github.com/doug-martin/goqu/v9"
	"github.com/harrybrwn/edu/school/ucmerced/ucm"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/mercedtime/api/catalog"
	"github.com/mercedtime/api/db/models"
	"github.com/pkg/errors"
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
}

var (
	updateTmpl = `UPDATE "{{ .Target }}"{{ $n := sub (len .Vars) 1 }}
SET {{ range $i, $v := .Vars }}
  "{{ $v }}" = "new"."{{ $v }}"{{ if ne $i $n }},{{ end }}
  {{- end }}
  {{- if .SetUpdated }},updated_at = now(){{ end }}
FROM (
  SELECT * FROM "{{ .Tmp }}" tmp
  WHERE NOT EXISTS (
    SELECT * FROM "{{ .Target }}" target
	WHERE
	  tmp.crn = target.crn AND
      {{- range $i, $v := .Vars }}
      "tmp"."{{ $v }}" = "target"."{{ . }}"{{ if ne $i $n }} AND{{end}}
      {{- end }}
  )
) new
WHERE "{{ .Target }}"."crn" = "new"."crn"`
	tableDiffTempl = `
	SELECT * FROM {{ .Tmp }} tmp {{ $n := sub (len .Vars) 1 }}
	WHERE NOT EXISTS (
	  SELECT * FROM {{ .Target }} target
	  WHERE
	    tmp.crn = target.crn AND
		{{- range $i, $v := .Vars }}
		tmp.{{ $v }} = target.{{ . }}{{if ne $i $n }} AND{{end}}
		{{- end }}
	)`
	tmplFuncs = template.FuncMap{
		"sub": func(a, b int) int { return a - b },
	}
)

func printTableDiff(tx *sql.Tx, data genquery) error {
	tmpl, err := template.New("debug").Funcs(tmplFuncs).Parse(tableDiffTempl)
	if err != nil {
		log.Fatal(err)
	}
	b := bytes.Buffer{}
	if err = tmpl.Execute(&b, data); err != nil {
		return err
	}
	rows, err := tx.Query(b.String())
	if err != nil {
		return err
	}
	if err = printQueryRows(rows); err != nil {
		return err
	}
	return rows.Close()
}

func printQueryRows(rows *sql.Rows) error {
	cols, err := rows.Columns()
	if err != nil {
		return err
	}
	fmt.Println(cols)
	i := 0
	for rows.Next() {
		columns := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := 0; i < len(cols); i++ {
			ptrs[i] = &columns[i]
		}
		if err = rows.Scan(ptrs...); err != nil {
			return err
		}
		for i, col := range cols {
			fmt.Printf("%v: %v, ", col, columns[i])
		}
		fmt.Println()
		i++
	}
	fmt.Println(i, "updated rows")
	return nil
}

func updatequery(data genquery) (string, error) {
	var buf bytes.Buffer
	tmpl, err := template.New(
		"sql-update-gen",
	).Funcs(tmplFuncs).Parse(updateTmpl)
	if err != nil {
		return "", err
	}
	err = tmpl.Execute(&buf, data)
	return buf.String(), err
}

func insertNew(target, tmp string, tx *sql.Tx, cols ...string) error {
	var q string
	if len(cols) > 0 {
		tmpl := `
	  INSERT INTO %[1]s (
		%[3]s
	  )
	  SELECT %[3]s FROM %[2]s tmp
	  WHERE NOT EXISTS (
	    SELECT * FROM %[1]s c
	    WHERE c.crn = tmp.crn
	  )`
		q = fmt.Sprintf(
			tmpl, target,
			tmp, strings.Join(cols, ","),
		)
	} else {
		tmpl := `
	  INSERT INTO %[1]s
	  SELECT * FROM %[2]s tmp
	  WHERE NOT EXISTS (
	    SELECT * FROM %[1]s c
	    WHERE c.crn = tmp.crn
	  )`
		q = fmt.Sprintf(tmpl, target, tmp)
	}
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

type tableUpdate struct {
	target, tmp string
	tx          *sqlx.Tx
	err         error
}

func newUpdate(target string, tx *sqlx.Tx) (*tableUpdate, error) {
	u := &tableUpdate{
		target: target,
		tmp:    "_tmp_" + target,
		tx:     tx,
		err:    nil,
	}
	_, u.err = tx.Exec(fmt.Sprintf("SELECT * INTO %s FROM %s LIMIT 0", u.tmp, u.target))
	if u.err != nil {
		return nil, u.err
	}
	return u, nil
}

func (tu *tableUpdate) start(columns ...string) error {
	// stmt, err := tu.tx.Prepare(pq.CopyIn(tu.tmp, columns...))
	// if err != nil {
	// 	return err
	// }
	return nil
}

func (tu *tableUpdate) close() (err error) {
	// drop the table no matter what
	_, err = tu.tx.Exec("DROP TABLE " + tu.tmp)
	if err != nil {
		tu.tx.Rollback()
		return err
	}
	if tu.err != nil {
		tu.tx.Rollback()
		return tu.err
	}
	// commit if everything went ok
	return tu.tx.Commit()
}

type tmptable struct {
	target, tmp string

	tx *sql.Tx
}

func newtmptable(target string, tx *sql.Tx) (*tmptable, error) {
	t := &tmptable{
		target: target,
		tmp:    "_tmp_" + target,
		tx:     tx,
	}
	_, err := tx.Exec(
		fmt.Sprintf("SELECT * INTO %s FROM %s LIMIT 0", t.tmp, t.target))
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (t *tmptable) String() string {
	return t.tmp
}

func (t *tmptable) close() error {
	_, err := t.tx.Exec(fmt.Sprintf("DROP TABLE %s", t.tmp))
	return err
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
		// 	return drop, nil
		panic("ok so this case where there are no rows actually does happen, come fix this updates.go")
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

func updateCourseTable(db *sqlx.DB, courses []*catalog.Entry) (err error) {
	var (
		target   = "course"
		tmpTable = "_tmp_" + target
	)
	tx, err := db.BeginTxx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelDefault, ReadOnly: false,
	})
	if err != nil {
		return err
	}
	tmp, err := newtmptable(target, tx.Tx)
	if err != nil {
		return err
	}
	defer func() {
		tmp.close()
		if err == nil {
			err = tx.Commit()
		}
	}()

	cols := []string{
		"crn",
		"subject",
		"course_num",
		"type",
		"title",
		"units",
		"days",
		"description",
		"capacity",
		"enrolled",
		"remaining",
		"year",
		"term_id",
	}
	stmt, err := tx.Prepare(pq.CopyIn(tmpTable, cols...))
	if err != nil {
		return errors.Wrap(err, "could not create prepared statment")
	}
	for _, c := range courses {
		if c.Description == "" {
			continue
		}
		_, err = stmt.Exec(
			c.CRN, c.Subject, c.CourseNum, c.Type, c.Title, c.Units, pq.Array(c.Days),
			c.Description, c.Capacity, c.Enrolled, c.Remaining, c.Year, c.TermID,
		)
		if err != nil {
			stmt.Close()
			return errors.Wrap(err, "could not insert into temp course table")
		}
	}
	if err = stmt.Close(); err != nil {
		return err
	}
	if err = insertNew(target, tmpTable, tx.Tx, cols...); err != nil {
		return errors.Wrap(err, "could not insert new values from tmp course table")
	}

	var q string
	q, err = updatequery(genquery{
		Target:     target,
		Tmp:        tmpTable,
		SetUpdated: true,
		Vars: []string{
			"subject",
			"course_num",
			"type",
			"units",
			"days",
			"title",
			"description",
			"capacity",
			"enrolled",
			"remaining",
			"year",
			"term_id",
		},
	})
	if err != nil {
		return errors.Wrap(err, "could not generate update query")
	}
	if _, err = tx.Exec(q); err != nil {
		return errors.Wrap(err, "could not perform updates from temp course table")
	}
	return nil
}

func updateLectureTable(
	db *sqlx.DB,
	lectures []*models.Lecture,
) (err error) {
	var (
		target   = "lectures"
		tmpTable = "_tmp_" + target
		rows     = interfaceSlice(lectures)
	)
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
	q, err := updatequery(genquery{
		Target:     target,
		Tmp:        tmpTable,
		SetUpdated: true,
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
		rows     = interfaceSlice(labs)
	)
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
		Vars: []string{
			"section",
			"start_time",
			"end_time",
			"building_room",
			"instructor_id"},
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
		rows = append(rows, &models.Instructor{
			ID:   inst.id,
			Name: inst.name,
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
		rows     = interfaceSlice(exams)
	)
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
