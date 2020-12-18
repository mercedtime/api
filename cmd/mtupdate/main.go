package main

import (
	"database/sql"
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

	"github.com/doug-martin/goqu/v9"
	"github.com/harrybrwn/config"
	"github.com/harrybrwn/edu/school/ucmerced/ucm"
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

	Logfile string `config:"logfile" default:"mtupdate.log"`
}

func main() {
	var (
		dbOpsOnly = false
		csvOps    = false

		noEnrollment   = false
		enrollmentOnly = false

		conf = updateConfig{
			Database: app.DatabaseConfig{
				Driver: "postgres",
				User:   "mt",
				Name:   "mercedtime",
			},
			Year: 2021,
			Term: "spring",
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
	flag.Parse()

	if !dbOpsOnly && !csvOps && !enrollmentOnly {
		flag.Usage()
		println("\n")
		log.Fatal("nothing to be done. use '-db' or '-csv' or '--enrollment-only'")
	}

	var (
		out io.Writer = os.Stdout
	)

	sch, err := ucm.NewSchedule(ucm.ScheduleConfig{Year: conf.Year, Term: conf.Term})
	if err != nil {
		log.Fatal(err)
	}

	if enrollmentOnly {
		db, err := opendb(conf)
		defer db.Close()
		err = recordHistoricalEnrollment(
			db, conf.Year, termcodeMap[conf.Term], sch.Ordered())
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
		db, err := opendb(conf)
		if err != nil {
			log.Fatal(err)
		}
		err = updates(os.Stdout, db, tab)
		if err != nil {
			log.Fatal("DB Error:", err)
		}
		if !noEnrollment {
			err = recordHistoricalEnrollment(
				db, conf.Year, termcodeMap[conf.Term],
				sch.Ordered(),
			)
			if err != nil {
				log.Fatal(err)
			}
		}
		fmt.Print("db updates done ")
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
		defer fmt.Print("csv files written ")
		if err = writes(conf, tab, schCP); err != nil {
			log.Fatal("CSV Error:", err)
		}
	}
}

func opendb(conf updateConfig) (*sql.DB, error) {
	db, err := sql.Open("postgres", conf.Database.GetDSN())
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
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
	// t1, t2 := time.Now(), time.Now()
	report := func(name ...string) {
		// t2 = time.Now()
		// fmt.Println(name, t2.Sub(t1))
		// t1 = t2
	}
	var (
		courses = sch.Ordered()
		err     error
		tab     = &tables{
			instructorMap: getInstructors(courses),
		}
	)
	report("instructors")
	tab.course, err = GetCourseTable(courses, 150)
	if err != nil {
		return nil, err
	}
	for i := range tab.course {
		tab.course[i].Year = conf.Year
		tab.course[i].TermID = termcodeMap[conf.Term]
	}
	report("courses")
	tab.lectures, err = getLectures(courses, tab.instructorMap)
	if err != nil {
		return nil, err
	}
	report("lectures")
	tab.aux = getLabsTable(courses, sch, tab.instructorMap)
	report("labs")
	tab.exam = make([]*models.Exam, 0, len(tab.lectures))
	for _, c := range courses {
		if c.Exam == nil {
			continue
		}
		tab.exam = append(tab.exam, &models.Exam{
			CRN:       c.CRN,
			Date:      c.Exam.Date,
			StartTime: c.Time.Start,
			EndTime:   c.Time.End,
		})
	}
	report("exams")
	return tab, nil
}

func updates(w io.Writer, db *sql.DB, tab *tables) (err error) {
	t := time.Now()
	fmt.Fprintf(w, "[%s] ", t.Format(time.Stamp))
	t = time.Now()

	fmt.Fprintf(w, "%v ", time.Now().Sub(t))
	fmt.Fprintf(w, "instructor:")
	err = updateInstructorsTable(db, tab.instructorMap)
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
		log.Println(err)
		return err
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

	err = updateLabsTable(db, tab.aux)
	if err != nil {
		log.Println(err)
		return err
	}
	fmt.Fprintf(w, "%v ok|exams:", time.Now().Sub(t))
	t = time.Now()

	err = updateExamTable(db, tab.exam)
	if err != nil {
		log.Println("Error update exam table:", err)
		return err
	}
	fmt.Fprintf(w, "%v ok|", time.Now().Sub(t))
	t = time.Now()

	return nil
}

func writes(conf updateConfig, tab *tables, sch ucm.Schedule) error {
	var (
		all = make([]interface{}, 0, len(tab.course))
		err error
	)
	for _, a := range tab.aux {
		all = append(all, a)
	}
	if err = writeCSVFile("labs_disc.csv", all); err != nil {
		return err
	}
	all = all[:0]

	for _, a := range tab.lectures {
		all = append(all, a)
	}
	if err = writeCSVFile("lecture.csv", all); err != nil {
		return err
	}
	all = all[:0]

	for _, a := range tab.exam {
		all = append(all, a)
	}
	if err = writeCSVFile("exam.csv", all); err != nil {
		return err
	}
	all = all[:0]

	for _, a := range tab.instructorMap {
		all = append(all, &models.Instructor{
			ID:   a.id,
			Name: a.name,
		})
	}
	if err = writeCSVFile("instructor.csv", all); err != nil {
		return err
	}
	all = all[:0]

	for _, a := range tab.course {
		all = append(all, a)
	}
	if err = writeCSVFile("course.csv", all); err != nil {
		return err
	}
	all = all[:0]
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

func getInstructors(crs []*ucm.Course) map[string]*instructorMeta {
	var (
		i           = 1
		instructors = make(map[string]*instructorMeta)
	)
	hash := fnv.New32a()
	for _, c := range crs {
		inst, ok := instructors[c.Instructor]
		if ok {
			inst.ncourses++
			inst.crns = append(inst.crns, c.CRN)
			continue
		}
		_, err := hash.Write([]byte(c.Instructor))
		if err != nil {
			panic(err) // TODO FIX THIS ASAP
		}
		inst = &instructorMeta{
			name:     c.Instructor,
			ncourses: 1,
			id:       int64(hash.Sum32()),
		}
		inst.crns = append(inst.crns, c.CRN)
		instructors[c.Instructor] = inst
		i++
		hash.Reset()
	}
	return instructors
}

func getLabsTable(
	courses []*ucm.Course,
	sch ucm.Schedule,
	instructors map[string]*instructorMeta,
) []*models.LabDisc {
	var labs = make([]*models.LabDisc, 0, len(courses))
	for _, c := range sch.Ordered() {
		if ucm.CourseType(c.Activity) == ucm.Lecture {
			continue
		}
		var lectCRN int
		lect, err := getDiscussionLecture(c, sch)
		if err == nil {
			lectCRN = lect.CRN
		} else {
			lectCRN = 0
		}
		instructorID := int64(0)
		instructor, ok := instructors[c.Instructor]
		if !ok {
			fmt.Println("Could not find instructor")
		} else {
			instructorID = instructor.id
		}
		labs = append(labs, &models.LabDisc{
			CRN:          c.CRN,
			CourseCRN:    lectCRN,
			Section:      c.Section,
			StartTime:    c.Time.Start,
			EndTime:      c.Time.End,
			Building:     c.BuildingRoom,
			InstructorID: instructorID,
		})
	}
	return labs
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
				}
				crs := catalog.Entry{
					CRN:         c.CRN,
					Subject:     c.Subject,
					CourseNum:   c.Number,
					Type:        c.Activity,
					Title:       cleanTitle(c.Title),
					Units:       c.Units,
					Days:        str(c.Days),
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
		instructorID := int64(0)
		instructor, ok := instructors[c.Instructor]
		if !ok {
			return nil, errors.New("could not find an instructor")
		}
		instructorID = instructor.id
		// For type safety and so i get error messages
		// when the schema changes
		list = append(list, &models.Lecture{
			CRN:          c.CRN,
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
	instructorID := int64(0)
	for _, c := range sch.Ordered() {
		l := &models.Lecture{
			CRN:          c.CRN,
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
