package main

import (
	"flag"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/agnivade/levenshtein"
	"github.com/harrybrwn/config"
	"github.com/harrybrwn/edu/school/ucmerced/ucm"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/mercedtime/api/app"
	"github.com/mercedtime/api/catalog"
	"github.com/mercedtime/api/db/models"
	"github.com/pkg/errors"
)

var termcodeMap = map[string]int{
	"spring": 1,
	"summer": 2,
	"fall":   3,
}

type updateConfig struct {
	Database app.DatabaseConfig `config:"db" yaml:"db"`
	Year     int                `config:"year"`
	Term     string             `config:"term"`

	SkipCourses bool   `config:"skipcourses"`
	Logfile     string `config:"logfile" default:"mtupdate.log"`
}

func (conf *updateConfig) init() {
	flag.StringVar(&conf.Database.Password, "password", conf.Database.Password, "give postgres a password")
	flag.StringVar(&conf.Database.Host, "host", conf.Database.Host, "specify the database host")
	flag.IntVar(&conf.Database.Port, "port", conf.Database.Port, "specify the database port")
	flag.IntVar(&conf.Year, "year", conf.Year, "the year")
	flag.StringVar(&conf.Term, "term", conf.Term, "the term")
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	var (
		dbOpsOnly, csvOps            = false, false
		noEnrollment, enrollmentOnly = false, false

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

	if err := config.InitDefaults(); err != nil {
		log.Println("could not initialize config defaults")
	}
	config.ReadConfigFile() // ignore error if not there

	flag.BoolVar(&dbOpsOnly, "db", dbOpsOnly, "only perform database updates")
	flag.BoolVar(&csvOps, "csv", csvOps, "write the tables to csv files")
	flag.BoolVar(&enrollmentOnly, "enrollment-only", enrollmentOnly, "only updated the db with enrollmenet data")
	flag.BoolVar(&noEnrollment, "no-enrollment", noEnrollment, "do not update the enrollment table")
	conf.init()
	flag.Parse()

	if !dbOpsOnly && !csvOps && !enrollmentOnly {
		flag.Usage()
		println("\n")
		return errors.New("nothing to be done. use '-db' or '-csv' or '--enrollment-only'")
	}

	// ucm.SetHTTPClient(http.Client{Timeout: time.Second * 2})
	var out io.Writer = os.Stdout
	sch, err := ucm.NewSchedule(ucm.ScheduleConfig{
		Year:    conf.Year,
		Term:    conf.Term,
		Open:    false,
		Subject: "",
	})
	if err != nil {
		return err
	}

	if enrollmentOnly {
		db, err := sqlx.Connect("postgres", conf.Database.GetDSN())
		if err != nil {
			return err
		}
		defer db.Close()
		err = recordHistoricalEnrollment(
			db.DB, conf.Year, termcodeMap[conf.Term], sch.Ordered())
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%d courses updated\n", sch.Len())
		return nil
	}

	tab, err := PopulateTables(sch, &conf)
	if err != nil {
		return err
	}

	defer fmt.Println()
	if dbOpsOnly {
		db, err := sqlx.Connect("postgres", conf.Database.GetDSN())
		if err != nil {
			return err
		}
		defer closeDB(db)
		err = updates(os.Stdout, db, tab)
		if err != nil {
			return fmt.Errorf("DB Error: %w", err)
		}
		if !noEnrollment {
			err = recordHistoricalEnrollment(
				db.DB, conf.Year, termcodeMap[conf.Term],
				sch.Ordered(),
			)
			if err != nil {
				return err
			}
		}
		fmt.Print("db updates done ")
	}

	if csvOps {
		if err = writes(tab); err != nil {
			return fmt.Errorf("CSV Error: %w", err)
		}
		fmt.Print("csv files written ")
	}
	return nil
}

func closeDB(db *sqlx.DB) (err error) {
	_, err = db.Exec("REFRESH MATERIALIZED VIEW CONCURRENTLY catalog")
	if err != nil {
		log.Println("could not refresh materialized view:", err)
	}
	if e := db.Close(); e != nil && err == nil {
		err = e
	}
	return err
}

// Tables holds table data
type Tables struct {
	course     []*catalog.Entry
	lectures   []*models.Lecture
	aux        []*models.LabDisc
	exam       []*models.Exam
	instructor map[string]*models.Instructor
}

// PopulateTables will get table data
func PopulateTables(sch ucm.Schedule, conf *updateConfig) (*Tables, error) {
	var (
		courses = sch.Ordered()
		tab     = &Tables{
			instructor: make(map[string]*models.Instructor),
			exam:       make([]*models.Exam, 0, 128), // 128 is arbitrary
		}
		err error
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
		_, ok := tab.instructor[c.Instructor]
		if !ok {
			tab.instructor[c.Instructor], err = newInstructor(c.Instructor)
			if err != nil {
				return nil, err
			}
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
	return tab, tab.populateLabsLectures(courses, sch)
}

func updates(w io.Writer, db *sqlx.DB, tab *Tables) (err error) {
	t := time.Now()
	fmt.Fprintf(w, "[%s] ", t.Format(time.Stamp))
	t = time.Now()

	fmt.Fprintf(w, "%v ", time.Now().Sub(t))
	fmt.Fprintf(w, "instructor:")

	instructors := instructorMapToInterfaceSlice(tab.instructor)
	err = updateInstructorsTable("instructor", db.DB, instructors)
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

func writes(tab *Tables) error {
	for file, data := range map[string][]interface{}{
		"labs_disc.csv":  interfaceSlice(tab.aux),
		"lecture.csv":    interfaceSlice(tab.lectures),
		"exam.csv":       interfaceSlice(tab.exam),
		"course.csv":     interfaceSlice(tab.course),
		"instructor.csv": instructorMapToInterfaceSlice(tab.instructor),
	} {
		err := writeCSVFile(file, data)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *Tables) populateLabsLectures(courses []*ucm.Course, sch ucm.Schedule) error {
	var dup = make(map[int]struct{})
	for _, c := range courses {
		if c.Time.Start.Year() == 0 {
			c.Time.Start = c.Time.Start.AddDate(1, 0, 0)
		}
		if c.Time.End.Year() == 0 {
			c.Time.End = c.Time.End.AddDate(1, 0, 0)
		}
		instructorID := int64(0)
		instructor, ok := t.instructor[c.Instructor]
		if !ok {
			return errors.New("could not find an instructor")
		}
		instructorID = instructor.ID
		if _, ok := dup[c.CRN]; ok { // check for duplicate crn's
			return errors.New("table.populate: tried to put duplicate crn in db")
		}
		dup[c.CRN] = struct{}{}

		typ := ucm.CourseType(c.Activity)
		if typ == ucm.Lecture || typ == ucm.Seminar {
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

func newInstructor(name string) (*models.Instructor, error) {
	var (
		id   int64
		hash = fnv.New32a()
	)
	if name == "Staff" {
		id = 1
	} else {
		if _, err := hash.Write([]byte(name)); err != nil {
			return nil, err
		}
		id = int64(hash.Sum32())
	}
	return &models.Instructor{
		Name: name,
		ID:   id,
	}, nil
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
		mu     sync.Mutex
		wg     sync.WaitGroup
		errs   = make(chan error)
		ch     = make(chan *ucm.Course)
		result = make([]*catalog.Entry, 0, len(courses))
	)

	// The last worker will not finish until
	// the goroutine above finishes and will
	// not free the waitgroup so this function
	// will not return before all the courses
	// are processed.
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(worker int) {
			defer wg.Done()
			var innerRes []*catalog.Entry
			for c := range ch {
				info, err := c.Info()
				if err != nil {
					log.Println("could not get course description:", err, "skipping...")
					errs <- err
					continue
				}
				crs := catalog.Entry{
					CRN:         c.CRN,
					Subject:     c.Subject,
					CourseNum:   c.Number,
					Type:        c.Activity,
					Title:       cleanTitle(c.Title),
					Units:       c.Units,
					Days:        catalog.NewWeekdays(c.Days),
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
	go func() {
		// Convert the course list to
		// a channel in the background
		for _, c := range courses {
			ch <- c
		}
		close(ch)
		// Wait for all the workers to
		// finish before closing the
		// error channel.
		wg.Wait()
		close(errs)
	}()

	var err error
	for e := range errs {
		if e != nil && err == nil {
			err = e
		}
	}
	return result, err
}

type blueprint struct {
	Subject    string
	Num        int
	Title      string
	Enrollment int64
	Capacity   int64
	CRNs       []int
}

type blueprintKey struct {
	Subject string
	Num     int
}

func findBlueprints(courses []*ucm.Course) (map[blueprintKey][]*ucm.Course, error) {
	var (
		groups = make(map[blueprintKey][]*ucm.Course)
		key    blueprintKey
		list   []*ucm.Course
		ok     bool
	)
	for _, c := range courses {
		key = blueprintKey{Subject: strings.ToLower(c.Subject), Num: c.Number}
		list, ok = groups[key]
		if !ok {
			groups[key] = []*ucm.Course{c}
			continue
		}
		groups[key] = append(list, c)
	}

	blueprints := make([]blueprint, 0, len(groups))
	for _, list = range groups {
		titles := make(map[string]struct{}, 0)
		bp := blueprint{
			Subject: list[0].Subject,
			Num:     list[0].Number,
		}
		for _, c := range list {
			if c.Subject != bp.Subject || c.Number != bp.Num {
				return nil, errors.New("incorrect course blueprint classification")
			}
			titles[c.Title] = struct{}{}
			if c.Subject != bp.Subject {
				return nil, errors.New("course subject did not match")
			}
			if c.Number != bp.Num {
				return nil, errors.New("course number did not match")
			}
			bp.CRNs = append(bp.CRNs, c.CRN)
			bp.Capacity += int64(c.Capacity)
			bp.Enrollment += int64(c.Enrolled)
		}
		bp.findTitle(titles)
		blueprints = append(blueprints, bp)
	}
	return groups, nil
}

func (bp *blueprint) findTitle(titleSet map[string]struct{}) {
	if len(titleSet) >= 2 && len(titleSet) < 6 {
		titlesList := mapKeys(titleSet)
		dist := levenshtein.ComputeDistance(titlesList[0], titlesList[1])
		if dist >= 18 {
			bp.Title = titlesList[0] + ", " + titlesList[1]
		} else if dist < 18 {
			bp.Title = titlesList[0]
		}
	}
}
