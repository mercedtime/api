package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/harrybrwn/edu/school/ucmerced/ucm"
	_ "github.com/lib/pq"
	"github.com/mercedtime/api/db/models"
)

/*
TODO:
  - Add subject
  - unify the naming conventions for activity
*/

func main() {
	var (
		dbOpsOnly = false
		csvOps    = false
		password  string
		host      string = "localhost"
		port      string = "5432"
	)
	flag.StringVar(&password, "password", password, "give postgres a password")
	flag.StringVar(&host, "host", host, "specify the database host")
	flag.StringVar(&port, "port", port, "specify the database port")
	flag.BoolVar(&dbOpsOnly, "db", dbOpsOnly, "only perform database updates")
	flag.BoolVar(&csvOps, "csv", csvOps, "write the tables to csv files")
	flag.Parse()

	if !dbOpsOnly && !csvOps {
		flag.Usage()
		println("\n")
		log.Fatal("nothing to be done. use '-db-ops' or '-csv'")
	}

	conf := ucm.ScheduleConfig{Year: 2021, Term: "spring"}
	sch, err := ucm.NewSchedule(conf)
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	if dbOpsOnly {
		info := fmt.Sprintf("host=%s port=%s user=mt dbname=mercedtime sslmode=disable", host, port)
		if password != "" {
			info += " password=" + password
		}
		db, err := sql.Open("postgres", info)
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()
		if err = db.Ping(); err != nil {
			log.Fatal(err)
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			err := updates(os.Stdout, db, sch)
			if err != nil {
				log.Fatal("DB Error:", err)
			}
			fmt.Print("db updates done ")
		}()
	}
	if csvOps {
		schCP := make(ucm.Schedule)
		for k, v := range sch {
			cp := *v
			if v.Exam != nil {
				*cp.Exam = *v.Exam
			}
			schCP[k] = &cp
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err = writes(schCP); err != nil {
				log.Fatal("CSV Error:", err)
			}
			fmt.Print("csv files written ")
		}()
	}
	wg.Wait()
	fmt.Println()
}

func updates(w io.Writer, db *sql.DB, sch ucm.Schedule) (err error) {
	courses := sch.Ordered()
	inst := getInstructors(courses)
	fmt.Fprintf(w, "[%s] lectures:", time.Now().Format(time.Stamp))
	err = updateLectureTable(db, courses, inst)
	if err != nil {
		return err
	}
	fmt.Fprint(w, "ok|labs:")
	err = updateLabsTable(db, sch, inst)
	if err != nil {
		return err
	}
	fmt.Fprint(w, "ok|instructor:")
	err = updateInstructorsTable(db, inst)
	if err != nil {
		return err
	}
	fmt.Fprint(w, "ok|enrollments:")
	err = updateEnrollment(db, courses)
	if err != nil {
		return err
	}
	fmt.Fprint(w, "ok|")
	return nil
}

func writes(sch ucm.Schedule) error {
	courses := sch.Ordered()
	err := courseTable(courses)
	if err != nil {
		return err
	}
	inst, err := writeInstructorTable(sch.Ordered())
	if err != nil {
		log.Fatal(err)
	}
	_, err = lecturesTable(courses, inst)
	if err != nil {
		return err
	}
	if err = labsDiscTable(sch, inst); err != nil {
		return err
	}
	if err = examsTable(courses); err != nil {
		return err
	}
	if err = enrollmentTable(courses); err != nil {
		return err
	}
	return nil
}

const (
	dateformat = time.RFC3339
	timeformat = "15:04:05"
)

func toCsvRow(v interface{}) ([]string, error) {
	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Ptr:
		val = val.Elem()
	}
	var (
		row []string
		s   string
	)
	for i := 0; i < val.NumField(); i++ {
		f := val.Field(i)

	KindCheck:
		switch f.Kind() {
		case reflect.Ptr:
			f = f.Elem()
			goto KindCheck
		case reflect.String:
			s = f.String()
		case reflect.Int:
			s = strconv.FormatInt(f.Int(), 10)
		case reflect.Bool:
			s = strconv.FormatBool(f.Bool())
		case reflect.Float32:
			s = strconv.FormatFloat(f.Float(), 'f', -1, 32)
		case reflect.Float64:
			s = strconv.FormatFloat(f.Float(), 'f', -1, 64)
		case reflect.Struct:
			itf := f.Interface()
			switch itval := itf.(type) {
			case time.Time:
				s = itval.Format(dateformat)
			case ucm.Exam:
				s = fmt.Sprintf("Exam{%v}", itval.Day.String())
			case struct{ Start, End time.Time }:
				s = itval.Start.Format(dateformat)
			default:
				return nil, errors.New("cannot handle this struct")
			}
		case reflect.Slice:
			switch arr := f.Interface().(type) {
			case []byte:
				s = string(arr)
			case []time.Weekday:
				s = daysString(arr)
			default:
				return nil, errors.New("can't handle arrays")
			}
		case reflect.Invalid:
			s = "<nil>"
		default:
			fmt.Println("what type is this", f.Kind())
		}
		row = append(row, s)
	}
	return row, nil
}

// These are the activity types for any given course
const (
	Lecture    = "LECT"
	Discussion = "DISC"
	Lab        = "LAB"
	Seminar    = "SEM"
	// TODO find out what "STDO", "INI", "FLDW" are
)

type instructorMeta struct {
	name     string
	ncourses int
	id       int
	crns     []int
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func getInstructors(crs []*ucm.Course) map[string]*instructorMeta {
	var (
		i           = 1
		instructors = make(map[string]*instructorMeta)
	)
	for _, c := range crs {
		inst, ok := instructors[c.Instructor]
		if ok {
			inst.ncourses++
			inst.crns = append(inst.crns, c.CRN)
			continue
		}
		inst = &instructorMeta{
			name:     c.Instructor,
			ncourses: 1,
			id:       i,
		}
		inst.crns = append(inst.crns, c.CRN)
		instructors[c.Instructor] = inst
		i++
	}
	return instructors
}

func getDiscussionLecture(disc *ucm.Course, sch ucm.Schedule) (*ucm.Course, error) {
	var (
		ordered = sch.Ordered()
		end     = len(ordered)
		i       = 0
		c       *ucm.Course
	)
	for i < end {
		c = ordered[i]
		if c.Activity != Lecture {
			i++
			continue // these are the same
		}

		// if the current lecture has the same subject and course code
		// then we loop until we find another lecture and if we find
		// the discussion passed as an argument the we return the lecture
		if c.Number == disc.Number && c.Subject == disc.Subject {
			var (
				j         = i + 1
				subcourse = ordered[j]
			)
			for j < end &&
				subcourse.Number == disc.Number &&
				subcourse.Subject == disc.Subject &&
				subcourse.Activity != Lecture {
				if subcourse.CRN == disc.CRN {
					return c, nil
				}
				j++
				subcourse = ordered[j]
			}
			// did not find the discussion
			// update index and move on
			i = j
			continue // don't increment the index
		}
		i++
	}
	return nil, fmt.Errorf("could not find a lecture for \"%s %s\"", disc.Fullcode, disc.Title)
}
func generateLectureInsert(sch ucm.Schedule) string {
	insert := goqu.Insert("Lectures")
	rows := make([]*models.Lect, 0, len(sch))
	instructorID := 0
	for _, c := range sch.Ordered() {
		l := &models.Lect{
			CRN:          c.CRN,
			CourseNum:    c.Number,
			Title:        c.Title,
			Units:        c.Units,
			Activity:     c.Activity,
			Days:         str(c.Days),
			StartTime:    c.Time.Start,
			EndTime:      c.Time.End,
			StartDate:    c.Date.Start,
			EndDate:      c.Date.End,
			InstructorID: instructorID,
		}
		rows = append(rows, l)
	}
	s, _, err := insert.Rows(rows).ToSQL()
	if err != nil {
		panic(err)
	}
	return s
}

func daysString(days []time.Weekday) string {
	var s = make([]string, len(days))
	for i, d := range days {
		s[i] = d.String()
	}
	return strings.Join(s, ";")
}

func str(x interface{}) string {
	switch v := x.(type) {
	case int:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case []time.Weekday:
		return daysString(v)
	case time.Time:
		if v.Equal(time.Time{}) {
			return ""
		} else if v.Hour() == 0 && v.Minute() == 0 && v.Second() == 0 {
			return v.Format(dateformat)
		} else if v.Year() == 0 && v.Month() == time.January && v.Day() == 1 {
			return v.Format(timeformat)
		}
		return ""
	default:
		return ""
	}
}
