package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/doug-martin/goqu/v9"
	"github.com/harrybrwn/edu/school/ucmerced/ucm"
	"github.com/mercedtime/api/db/models"
)

var mustAlsoRegex = regexp.MustCompile(`Must Also.*$`)

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
	q, _, err = goqu.Insert(tmp).Rows(rows).ToSQL()
	if err != nil {
		return
	}
	if _, err = tx.Exec(q); err != nil {
		return
	}
	return drop, nil
}

func updateEnrollmentCounts(db *sql.DB, crs []*ucm.Course) (err error) {
	var (
		tmpTable = "_tmp_enrollments"
		rows     = make([]interface{}, 0)
	)
	for _, c := range crs {
		rows = append(rows, &models.Enrollment{
			CRN:       c.CRN,
			Capacity:  c.Capacity,
			Enrolled:  c.Enrolled,
			Remaining: c.SeatsOpen(),
		})
	}
	if err != nil {
		return err
	}
	droptmp, err := createTmpTable("enrollment", db, tmpTable, rows)
	defer func() {
		e := droptmp()
		if e != nil && err == nil {
			err = e
		}
	}()
	if err != nil {
		return err
	}
	q := `
UPDATE Enrollment
SET
	capacity    = tmp.capacity,
	enrolled    = tmp.enrolled,
	remaining   = tmp.remaining,
	auto_updated = 1
FROM _tmp_enrollments tmp
WHERE Enrollment.CRN = tmp.CRN`
	_, err = db.Exec(q)
	if err != nil {
		return err
	}
	return err
}

func updateEnrollment(db *sql.DB, crs []*ucm.Course, outerWg *sync.WaitGroup) (err error) {
	var (
		workers     = 300
		wg          sync.WaitGroup
		rows        = make([]interface{}, 0)
		courses     = make(chan *ucm.Course)
		enrollments = make(chan *models.Enrollment)
		tmpTable    = "_tmp_enrollments"
		insert      = goqu.Insert(tmpTable)
	)
	defer outerWg.Done()
	// Create the temp table
	go func() {
		for _, c := range crs {
			courses <- c
		}
		close(courses)
	}()

	wg.Add(workers)
	go func() {
		wg.Wait()
		close(enrollments)
	}()
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for c := range courses {
				desc, err := c.Info()
				if err != nil {
					fmt.Println("Error:", err)
					return
				}
				enrollments <- &models.Enrollment{
					CRN: c.CRN, Desc: desc,
					Capacity:  c.Capacity,
					Enrolled:  c.Enrolled,
					Remaining: c.SeatsOpen(),
				}
			}
		}()
	}
	for e := range enrollments {
		rows = append(rows, e)
	}
	q, _, err := insert.Rows(rows...).ToSQL()
	if err != nil {
		return err
	}
	_, err = db.Exec("SELECT * INTO _tmp_enrollments FROM Enrollment LIMIT 0")
	if err != nil {
		return err
	}
	defer func() {
		_, e := db.Exec("DROP TABLE _tmp_enrollments")
		if e != nil && err == nil {
			err = e
		}
	}()
	if _, err = db.Exec(q); err != nil {
		return err
	}
	q = `
UPDATE Enrollment
SET
	description = tmp.description,
	capacity    = tmp.capacity,
	enrolled    = tmp.enrolled,
	remaining   = tmp.remaining,
	auto_updated = 1
FROM _tmp_enrollments tmp
WHERE Enrollment.CRN = tmp.CRN`
	_, err = db.Exec(q)
	if err != nil {
		return err
	}
	return err
}

func updateLectureTable(
	db *sql.DB,
	crs []*ucm.Course,
	instructors map[string]*instructorMeta,
) (err error) {
	var (
		rows         = make([]interface{}, 0, len(crs))
		instructorID = 0
	)

	for _, c := range crs {
		if c.Activity != Lecture {
			continue
		}
		instructorID = 0
		instructor, ok := instructors[c.Instructor]
		if !ok {
			fmt.Println("Coudld not find instructor", c.Instructor)
		} else {
			for _, crn := range instructor.crns {
				if crn == c.CRN {
					goto FoundCRN
				}
			}
			return errors.New("bad instructor")
		FoundCRN:
			instructorID = instructor.id
		}
		m := map[string]interface{}{
			"crn":           c.CRN,
			"course_num":    c.Number,
			"title":         cleanTitle(c.Title),
			"units":         c.Units,
			"activity":      c.Activity,
			"days":          str(c.Days),
			"start_time":    c.Time.Start.Format(timeformat),
			"end_time":      c.Time.End.Format(timeformat),
			"start_date":    c.Date.Start.Format(dateformat),
			"end_date":      c.Date.End.Format(dateformat),
			"instructor_id": instructorID,
			"auto_updated":  0,
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
	q = `
UPDATE Lectures
SET
  title = new.title,
  units = new.units,
  days  = new.days,
  start_time    = new.start_time,
  end_time      = new.end_time,
  start_date    = new.start_date,
  end_date      = new.end_date,
  instructor_id = new.instructor_id,
  auto_updated = 1
FROM (
  SELECT * FROM _tmp_lectures tmp
  WHERE NOT EXISTS (
    SELECT * FROM Lectures l
    WHERE
      tmp.title = l.title AND
      tmp.units = l.units AND
      tmp.days  = l.days AND
      tmp.start_time = l.start_time AND
      tmp.end_time   = l.end_time AND
      tmp.start_date = l.start_date AND
      tmp.end_date   = l.end_date AND
      tmp.instructor_id = l.instructor_id
  )
) new WHERE Lectures.CRN = new.CRN`
	if _, err = tx.Exec(q); err != nil {
		return err
	}
	return err
}

func updateCourseTable(db *sql.DB, crs []*ucm.Course) error {
	var (
		target   = "course"
		tmpTable = "_tmp_course"
		rows     = make([]interface{}, 0, len(crs))
	)
	for _, c := range crs {
		m := map[string]interface{}{
			"crn":        c.CRN,
			"subject":    c.Subject,
			"course_num": c.CourseNumber(),
			"type":       c.Activity,
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
	q := "INSERT INTO " + target + `
	SELECT * FROM ` + tmpTable + ` tmp
	WHERE NOT EXISTS (
	  SELECT * FROM ` + target + ` c
	  WHERE c.CRN = tmp.CRN
	)`
	if _, err = tx.Exec(q); err != nil {
		return err
	}
	q = `
UPDATE ` + target + `
SET
  subject = new.subject,
  course_num = new.course_num,
  type = new.type,
  auto_updated = 1
FROM (
  SELECT * FROM ` + tmpTable + ` tmp
  WHERE NOT EXISTS (
    SELECT * FROM ` + target + ` l
	WHERE
	  tmp.subject = l.subject AND
	  tmp.course_num = l.course_num AND
	  tmp.type = l.type
  )
) new WHERE ` + target + `.crn = new.crn`
	if _, err = tx.Exec(q); err != nil {
		return err
	}
	return err
}

func updateLabsTable(db *sql.DB, sch ucm.Schedule, instructors map[string]*instructorMeta) (err error) {
	var rows = make([]interface{}, 0, len(sch))
	for _, c := range sch.Ordered() {
		if c.Activity == Lecture {
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
			"course_num":    c.CourseNumber(),
			"section":       c.Section,
			"title":         c.Title,
			"units":         c.Units,
			"activity":      c.Activity,
			"days":          str(c.Days),
			"start_time":    c.Time.Start.Format(timeformat),
			"end_time":      c.Time.End.Format(timeformat),
			"building_room": c.BuildingRoom,
			"instructor_id": instructorID,
			"auto_updated":  "0",
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
	droptmp, err := createTmpTable("Labs_Discussions", tx, "_tmp_labs", rows)
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
	INSERT INTO Labs_Discussions
	SELECT * FROM _tmp_labs tmp
	WHERE NOT EXISTS (
	  SELECT * FROM Labs_Discussions l
	  WHERE l.CRN = tmp.CRN
	)`
	if _, err = tx.Exec(q); err != nil {
		return err
	}
	q = `
UPDATE Labs_Discussions
SET
  title = new.title,
  units = new.units,
  days  = new.days,
  start_time = new.start_time,
  end_time   = new.end_time,
  building_room = new.building_room,
  instructor_id = new.instructor_id,
  auto_updated = 1
FROM (
  SELECT * FROM _tmp_labs tmp
  WHERE NOT EXISTS (
    SELECT * FROM Labs_Discussions l
    WHERE
      tmp.title = l.title AND
	  tmp.units = l.units AND
      tmp.days  = l.days AND
      tmp.start_time = l.start_time AND
      tmp.end_time   = l.end_time AND
	  tmp.building_room = l.building_room AND
	  tmp.instructor_id = l.instructor_id
  )
) new WHERE new.CRN = Labs_Discussions.CRN`
	_, err = tx.Exec(q)
	if err != nil {
		return err
	}
	return nil
}

func updateInstructorsTable(db *sql.DB, instructors map[string]*instructorMeta) (err error) {
	var (
		rows = make([]interface{}, 0, len(instructors))
		// up   = goqu.Insert("tmp_inst")
	)
	for _, inst := range instructors {
		rows = append(rows, models.Instructor{ID: inst.id, Name: inst.name})
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
