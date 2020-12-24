package main

import (
	"flag"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/harrybrwn/config"
	"github.com/harrybrwn/edu/school/ucmerced/ucm"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"github.com/mercedtime/api/app"
	"github.com/mercedtime/api/catalog"
	"github.com/mercedtime/api/db/models"
	"github.com/pkg/errors"
)

/*
TODO:
	- Add context to the mechanism that gets course descriptions
	  because it sometimes hanges and I have no idea why.
  	- unify the naming conventions for activity
*/

var termcodeMap = map[string]int{
	"spring": 1,
	"summer": 2,
	"fall":   3,
}

type updateConfig struct {
	Database app.DatabaseConfig `config:"database"`
	Year     int                `config:"year"`
	Term     string             `config:"term"`

	SkipCourses bool `config:"skipcourses"`

	Logfile string `config:"logfile" default:"mtupdate.log"`
}

func main() {
	var (
		dbOpsOnly      = false
		csvOps         = false
		noEnrollment   = false
		enrollmentOnly = false

		conf = updateConfig{
			Database: app.DatabaseConfig{
				Driver: "postgres",
				User:   "mt",
				Name:   "mercedtime",
			},
			Year:        2021,
			Term:        "spring",
			SkipCourses: false,
		}
	)

	config.SetFilename("mt.yml")
	config.SetType("yml")
	config.AddPath(".")
	config.SetConfig(&conf)
	config.ReadConfigFile() // ignore error if not there
	if err := config.InitDefaults(); err != nil {
		log.Println("could not initialize config defaults")
	}

	flag.StringVar(&conf.Database.Password, "password", conf.Database.Password, "give postgres a password")
	flag.StringVar(&conf.Database.Host, "host", conf.Database.Host, "specify the database host")
	flag.IntVar(&conf.Database.Port, "port", conf.Database.Port, "specify the database port")
	flag.BoolVar(&dbOpsOnly, "db", dbOpsOnly, "only perform database updates")
	flag.BoolVar(&csvOps, "csv", csvOps, "write the tables to csv files")
	flag.BoolVar(&enrollmentOnly, "enrollment-only", enrollmentOnly, "only updated the db with enrollmenet data")
	flag.BoolVar(&noEnrollment, "no-enrollment", noEnrollment, "do not update the enrollment table")

	flag.IntVar(&conf.Year, "year", conf.Year, "the year")
	flag.StringVar(&conf.Term, "term", conf.Term, "the term")
	// flag.BoolVar(&conf.SkipCourses, "skip-courses", conf.SkipCourses, "skip the courses update")
	flag.Parse()

	if !dbOpsOnly && !csvOps && !enrollmentOnly {
		flag.Usage()
		println("\n")
		log.Fatal("nothing to be done. use '-db' or '-csv' or '--enrollment-only'")
	}

	var out io.Writer = os.Stdout
	sch, err := ucm.NewSchedule(ucm.ScheduleConfig{
		Year:    conf.Year,
		Term:    conf.Term,
		Open:    false,
		Subject: "",
	})
	if err != nil {
		log.Fatal(err)
	}

	if enrollmentOnly {
		db, err := sqlx.Connect("postgres", conf.Database.GetDSN())
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()
		err = recordHistoricalEnrollment(
			db.DB, conf.Year, termcodeMap[conf.Term], sch.Ordered())
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(out, "%d courses updated\n", sch.Len())
		return
	}

	tab, err := getTablesData(sch, &conf)
	if err != nil {
		log.Fatal(err)
	}

	defer fmt.Println()
	if dbOpsOnly {
		db, err := sqlx.Connect("postgres", conf.Database.GetDSN())
		if err != nil {
			log.Fatal(err)
		}
		err = updates(os.Stdout, db, tab)
		if err != nil {
			log.Fatal("DB Error:", err)
		}
		if !noEnrollment {
			err = recordHistoricalEnrollment(
				db.DB, conf.Year, termcodeMap[conf.Term],
				sch.Ordered(),
			)
			if err != nil {
				log.Fatal(err)
			}
		}
		fmt.Print("db updates done ")
	}

	if csvOps {
		if err = writes(conf, tab); err != nil {
			log.Fatal("CSV Error:", err)
		}
		fmt.Print("csv files written ")
	}
}

type tables struct {
	course        []*catalog.Entry
	lectures      []*models.Lecture
	aux           []*models.LabDisc
	exam          []*models.Exam
	instructorMap map[string]*instructorMeta
}

// TODO try to squash some of these helper functions into one loop
func getTablesData(sch ucm.Schedule, conf *updateConfig) (*tables, error) {
	var (
		courses = sch.Ordered()
		tab     = &tables{
			instructorMap: make(map[string]*instructorMeta),
			exam:          make([]*models.Exam, 0, 128), // 128 is arbitrary
		}
		hash = fnv.New32a()
		err  error
	)

	tab.course, err = GetCourseTable(courses, 150)
	if err != nil {
		return nil, err
	}
	for i := range tab.course {
		tab.course[i].Year = conf.Year
		tab.course[i].TermID = termcodeMap[conf.Term]
	}

	for _, c := range courses {
		// Postgres does not allow a timestamp with a year of 0000
		if c.Time.Start.Year() == 0 {
			c.Time.Start = c.Time.Start.AddDate(1, 0, 0)
		}
		if c.Time.End.Year() == 0 {
			c.Time.End = c.Time.End.AddDate(1, 0, 0)
		}
		_, ok := tab.instructorMap[c.Instructor]
		if !ok {
			if _, err := hash.Write([]byte(c.Instructor)); err != nil {
				return nil, err
			}
			tab.instructorMap[c.Instructor] = &instructorMeta{
				name:     c.Instructor,
				ncourses: 1,
				id:       int64(hash.Sum32()),
				crns:     []int{c.CRN},
			}
			hash.Reset()
		}
		if c.Exam != nil {
			tab.exam = append(tab.exam, &models.Exam{
				CRN:       c.CRN,
				Date:      c.Exam.Date,
				StartTime: c.Time.Start,
				EndTime:   c.Time.End,
			})
		}
	}
	return tab, tab.populate(courses, sch)
}

func updates(w io.Writer, db *sqlx.DB, tab *tables) (err error) {
	t := time.Now()
	fmt.Fprintf(w, "[%s] ", t.Format(time.Stamp))
	t = time.Now()

	fmt.Fprintf(w, "%v ", time.Now().Sub(t))
	fmt.Fprintf(w, "instructor:")
	err = updateInstructorsTable(db.DB, tab.instructorMap)
	if err != nil {
		log.Println(err)
		return err
	}
	fmt.Fprintf(w, "%v ok|course:", time.Now().Sub(t))
	t = time.Now()

	// The course table must go first in case there are new
	// CRNs because other tables depend on this table
	// via foreign key constrains.

	if err = updateCourseTable(db, tab.course); err != nil {
		return errors.Wrap(err, "update course failed")
	}
	fmt.Fprintf(w, "%v ok|lectures:", time.Now().Sub(t))
	t = time.Now()

	err = updateLectureTable(db, tab.lectures)
	if err != nil {
		log.Println(err)
		return err
	}
	fmt.Fprintf(w, "%v ok|labs:", time.Now().Sub(t))
	t = time.Now()

	err = updateLabsTable(db.DB, tab.aux)
	if err != nil {
		log.Println(err)
		return err
	}
	fmt.Fprintf(w, "%v ok|exams:", time.Now().Sub(t))
	t = time.Now()

	err = updateExamTable(db.DB, tab.exam)
	if err != nil {
		log.Println("Error update exam table:", err)
		return err
	}
	fmt.Fprintf(w, "%v ok|", time.Now().Sub(t))
	t = time.Now()

	return nil
}

func writes(conf updateConfig, tab *tables) error {
	var (
		err         error
		i           = 0
		instructors = make([]interface{}, len(tab.instructorMap))
	)
	for _, in := range tab.instructorMap {
		instructors[i] = &models.Instructor{ID: in.id, Name: in.name}
		i++
	}
	for file, data := range map[string][]interface{}{
		"labs_disc.csv":  interfaceSlice(tab.aux),
		"lecture.csv":    interfaceSlice(tab.lectures),
		"exam.csv":       interfaceSlice(tab.exam),
		"course.csv":     interfaceSlice(tab.course),
		"instructor.csv": instructors,
	} {
		err = writeCSVFile(file, data)
		if err != nil {
			return err
		}
	}
	return nil
}

type instructorMeta struct {
	name     string
	ncourses int
	id       int64
	crns     []int
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// depricated for getTablesData
// TODO: remove this
func getInstructors(crs []*ucm.Course) (map[string]*instructorMeta, error) {
	var (
		instructors = make(map[string]*instructorMeta)
	)
	hash := fnv.New32a()
	for _, c := range crs {
		inst, ok := instructors[c.Instructor]
		if ok {
			inst.ncourses++
			inst.crns = append(inst.crns, c.CRN)
		} else {
			_, err := hash.Write([]byte(c.Instructor))
			if err != nil {
				return nil, err
			}
			inst = &instructorMeta{
				name:     c.Instructor,
				ncourses: 1,
				id:       int64(hash.Sum32()),
			}
			inst.crns = append(inst.crns, c.CRN)
			instructors[c.Instructor] = inst
			hash.Reset()
		}
	}
	return instructors, nil
}

func (t *tables) populate(courses []*ucm.Course, sch ucm.Schedule) error {
	var dup = make(map[int]struct{})
	for _, c := range courses {
		if c.Time.Start.Year() == 0 {
			c.Time.Start = c.Time.Start.AddDate(1, 0, 0)
		}
		if c.Time.End.Year() == 0 {
			c.Time.End = c.Time.End.AddDate(1, 0, 0)
		}
		instructorID := int64(0)
		instructor, ok := t.instructorMap[c.Instructor]
		if !ok {
			return errors.New("could not find an instructor")
		}
		instructorID = instructor.id

		if _, ok := dup[c.CRN]; ok {
			return errors.New("table.populate: tried to put duplicate crn in db")
		}
		dup[c.CRN] = struct{}{}

		if ucm.CourseType(c.Activity) == ucm.Lecture {
			t.lectures = append(t.lectures, &models.Lecture{
				CRN:          c.CRN,
				StartTime:    c.Time.Start,
				EndTime:      c.Time.End,
				StartDate:    c.Date.Start,
				EndDate:      c.Date.End,
				InstructorID: instructorID,
			})
		} else {
			var lectCRN int = 0
			lect, err := getDiscussionLecture(c, sch)
			if err == nil {
				lectCRN = lect.CRN
			}
			t.aux = append(t.aux, &models.LabDisc{
				CRN:          c.CRN,
				CourseCRN:    lectCRN,
				Section:      c.Section,
				StartTime:    c.Time.Start,
				EndTime:      c.Time.End,
				Building:     c.BuildingRoom,
				InstructorID: instructorID,
			})
		}
	}
	return nil
}

// GetDiscussionLecture will return the lecture for a given discussion.
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
func GetCourseTable(courses []*ucm.Course, workers int) ([]*catalog.Entry, error) {
	var (
		mu   sync.Mutex
		wg   sync.WaitGroup
		errs = make(chan error)
		ch   = make(chan *ucm.Course)
	)
	// These will be protected by the mutex
	var (
		result = make([]*catalog.Entry, 0, len(courses))
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
			var (
				innerRes []*catalog.Entry
			)
			for c := range ch {
				info, err := c.Info()
				if err != nil {
					log.Println("could not get course description:", err)
					log.Printf("setting description as \"%s\"\n", info)
					errs <- err
					continue
				}
				crs := catalog.Entry{
					CRN:       c.CRN,
					Subject:   c.Subject,
					CourseNum: c.Number,
					Type:      c.Activity,
					Title:     cleanTitle(c.Title),
					Units:     c.Units,
					// Days:      daysString(c.Days),
					Days: catalog.NewWeekdays(c.Days),

					Description: info,
					Capacity:    c.Capacity,
					Enrolled:    c.Enrolled,
					Remaining:   c.SeatsOpen(),
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

func daysString(days []time.Weekday) string {
	var s = make([]string, len(days))
	for i, d := range days {
		s[i] = strings.ToLower(d.String())
	}
	// return strings.Join(s, ";")

	arr := pq.Array(s)
	val, err := arr.Value()
	if err != nil {
		// return "{" + strings.Join(s, ",") + "}"
		return "{}"
	}
	return val.(string)
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
