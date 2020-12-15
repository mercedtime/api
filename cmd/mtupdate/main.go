package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/harrybrwn/edu/school/ucmerced/ucm"
	_ "github.com/lib/pq"
	"github.com/mercedtime/api/db/models"
	"github.com/pkg/errors"
)

/*
TODO:
  - Add subject
  - unify the naming conventions for activity
*/

func main() {
	var (
		dbOpsOnly    = false
		csvOps       = false
		password     string
		host         string = "localhost"
		port         string = "5432"
		user, dbname string = "mt", "mercedtime"
		conf                = ucm.ScheduleConfig{Year: 2021, Term: "spring"}
	)
	flag.StringVar(&password, "password", password, "give postgres a password")
	flag.StringVar(&host, "host", host, "specify the database host")
	flag.StringVar(&port, "port", port, "specify the database port")
	flag.BoolVar(&dbOpsOnly, "db", dbOpsOnly, "only perform database updates")
	flag.BoolVar(&csvOps, "csv", csvOps, "write the tables to csv files")

	flag.IntVar(&conf.Year, "year", conf.Year, "the year")
	flag.StringVar(&conf.Term, "term", conf.Term, "the term")
	flag.Parse()

	if !dbOpsOnly && !csvOps {
		flag.Usage()
		println("\n")
		log.Fatal("nothing to be done. use '-db-ops' or '-csv'")
	}

	sch, err := ucm.NewSchedule(conf)
	if err != nil {
		log.Fatal(err)
	}

	defer fmt.Println()
	var wg sync.WaitGroup
	if dbOpsOnly {
		info := fmt.Sprintf(
			"host=%s port=%s user=%s dbname=%s sslmode=disable",
			host, port, user, dbname)
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
			err := updates(os.Stdout, db, sch, &wg)
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
		defer fmt.Print("csv files written ")
		go func() {
			defer wg.Done()
			if err = writes(schCP, &wg); err != nil {
				log.Fatal("CSV Error:", err)
			}
		}()
	}
	wg.Wait()
}

func updates(w io.Writer, db *sql.DB, sch ucm.Schedule, wg *sync.WaitGroup) (err error) {
	courses := sch.Ordered()
	inst := getInstructors(courses)

	t := time.Now()
	fmt.Fprintf(w, "[%s] ", t.Format(time.Stamp))

	fmt.Fprintf(w, "instructor:")
	err = updateInstructorsTable(db, inst)
	if err != nil {
		log.Println(err)
		return err
	}
	fmt.Fprintf(w, "%v ok|course:", time.Now().Sub(t))
	// The course table must go first in case there are new
	// CRNs because other tables depend on this table
	// via foreign key constrains.

	// wg.Add(1)
	// go func() {
	// defer wg.Done()
	if err = updateCourseTable(db, courses); err != nil {
		// log.Println(err)
		return err
	}
	// }()
	fmt.Fprintf(w, "%v ok|lectures:", time.Now().Sub(t))
	err = updateLectureTable(db, courses, inst)
	if err != nil {
		log.Println(err)
		return err
	}
	fmt.Fprintf(w, "%v ok|labs:", time.Now().Sub(t))
	err = updateLabsTable(db, sch, inst)
	if err != nil {
		log.Println(err)
		return err
	}
	fmt.Fprintf(w, "%v ok|exams:", time.Now().Sub(t))
	err = updateExamTable(db, courses)
	if err != nil {
		log.Println("Error update exam table:", err)
		return err
	}
	fmt.Fprintf(w, "%v ok|", time.Now().Sub(t))
	return nil
}

func writes(sch ucm.Schedule, wg *sync.WaitGroup) error {
	courses := sch.Ordered()
	var (
		err error
	)
	wg.Add(4)
	go func() {
		defer wg.Done()
		if err := courseTable(courses); err != nil {
			log.Println(err)
		}
	}()
	go func() {
		defer wg.Done()
		if err = examsTable(courses); err != nil {
			log.Println(err)
		}
	}()
	inst, err := writeInstructorTable(sch.Ordered())
	if err != nil {
		log.Println(err)
	}
	go func() {
		defer wg.Done()
		if err = lecturesTable(courses, inst); err != nil {
			log.Println(err)
		}
	}()
	go func() {
		defer wg.Done()
		if err = labsDiscTable(sch, inst); err != nil {
			log.Println(err)
		}
	}()
	return nil
}

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
		if c.Activity != models.Lect {
			i++
			continue // these are the same
		}

		// if the current lecture has the same subject
		// and course code then we loop until we find
		// another lecture and if we find the discussion
		// passed as an argument the we return the lecture
		if c.Number == disc.Number && c.Subject == disc.Subject {
			var (
				j         = i + 1
				subcourse = ordered[j]
			)
			for j < end &&
				subcourse.Number == disc.Number &&
				subcourse.Subject == disc.Subject &&
				subcourse.Activity != models.Lect {
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
	return nil, fmt.Errorf(
		"could not find a lecture for \"%s %s\"",
		disc.Fullcode, disc.Title,
	)
}

// GetCourseTable will get all the updated info needed by course table.
// Parameter courses is just a full list of raw courses and workers is the
// number of goroutines spawned that will be making requests to get the
// course description. The number of workers should be fairly high to get
// the most performance and should be limited to the number of connections
// that your computer can have open at one time.
//
// Side note: performance drops if the number of workers is too high
func GetCourseTable(courses []*ucm.Course, workers int) ([]*models.Course, error) {
	var (
		result = make([]*models.Course, 0, len(courses))
		mu     sync.Mutex
		wg     sync.WaitGroup
		errs   = make(chan error)
		ch     = make(chan *ucm.Course)
	)
	go func() {
		// Convert the course list to
		// a channel in the background
		for _, c := range courses {
			ch <- c
		}
		close(ch)
	}()

	// The last worker will not finish until
	// the goroutine above finishes and will
	// not free the waitgroup so this function
	// will not return before all the courses
	// are processed.
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(worker int) {
			defer wg.Done()
			var innerRes []*models.Course
			for c := range ch {
				info, err := c.Info()
				if err != nil {
					log.Println("could not get course description:", err)
					log.Printf("setting description as \"%s\"\n", info)
					errs <- err
				}
				crs := models.Course{
					CRN:         c.CRN,
					Subject:     c.Subject,
					CourseNum:   c.Number,
					Type:        c.Activity,
					Title:       cleanTitle(c.Title),
					Description: info,
					Capacity:    c.Capacity,
					Enrolled:    c.Enrolled,
					Remaining:   c.SeatsOpen(),
					AutoUpdated: 0,
				}
				innerRes = append(innerRes, &crs)
			}
			mu.Lock()
			result = append(result, innerRes...)
			mu.Unlock()
		}(i)
	}
	wg.Wait()

	// Close the error channel and drain it
	close(errs)
	var err error
	for e := range errs {
		if err == nil && e != nil {
			err = e
		}
	}
	return result, err
}

func getLectures(courses []*ucm.Course, instructors map[string]*instructorMeta) ([]*models.Lecture, error) {
	var (
		list     = make([]*models.Lecture, 0, len(courses))
		lectures = make(map[int]*ucm.Course, len(courses))
	)

	for _, c := range courses {
		if c.Activity != models.Lect {
			continue
		}
		if _, ok := lectures[c.CRN]; ok {
			return nil, errors.New("lectures: tried to put a duplicate crn in lectures table")
		}
		lectures[c.CRN] = c
		instructorID := 0
		instructor, ok := instructors[c.Instructor]
		if !ok {
			return nil, errors.New("could not find an instructor")
		}
		instructorID = instructor.id
		// For type safety and so i get error messages
		// when the schema changes
		list = append(list, &models.Lecture{
			CRN:          c.CRN,
			Units:        c.Units,
			Days:         str(c.Days),
			StartTime:    c.Time.Start,
			EndTime:      c.Time.End,
			StartDate:    c.Date.Start,
			EndDate:      c.Date.End,
			InstructorID: instructorID,
		})
	}
	return list, nil
}

func generateLectureInsert(sch ucm.Schedule) string {
	insert := goqu.Insert("lectures")
	rows := make([]*models.Lecture, 0, len(sch))
	instructorID := 0
	for _, c := range sch.Ordered() {
		l := &models.Lecture{
			CRN:          c.CRN,
			Units:        c.Units,
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
			return v.Format(models.DateFormat)
		} else if v.Year() == 0 && v.Month() == time.January && v.Day() == 1 {
			return v.Format(models.TimeFormat)
		}
		return ""
	default:
		return ""
	}
}
