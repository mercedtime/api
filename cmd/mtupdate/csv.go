package main

import (
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/harrybrwn/edu/school/ucmerced/ucm"
)

var csvOutDir = "data"

func init() {
	flag.StringVar(&csvOutDir, "out", csvOutDir, "output directory for csv files")
}

func csvfile(name string) (*os.File, error) {
	return os.OpenFile(filepath.Join(csvOutDir, name), os.O_CREATE|os.O_WRONLY, 0644)
}

func courseTable(crs []*ucm.Course) error {
	f, err := csvfile("course.csv")
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	var (
		mact    = 0
		maxSubj = 0
	)
	for _, c := range crs {
		mact = max(len(c.Activity), mact)
		maxSubj = max(len(c.Subject), maxSubj)
		err = w.Write([]string{
			strconv.FormatInt(int64(c.CRN), 10),
			c.Subject,
			str(c.Number),
			c.Activity,
			"0",
		})
		if err != nil {
			return err
		}
		w.Flush()
	}
	fmt.Printf(`Course Table:
	 max activity: %d
	 max subject:  %d`+"\n", mact, maxSubj)
	return nil
}

func lecturesTable(
	crs []*ucm.Course,
	instructors map[string]*instructorMeta,
) (map[int]*ucm.Course, error) {
	f, err := csvfile("lecture.csv")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	lectures := make(map[int]*ucm.Course)
	w := csv.NewWriter(f)

	var mtitle = 0
	for _, c := range crs {
		if c.Activity != Lecture {
			continue
		}
		if _, ok := lectures[c.CRN]; ok {
			return nil, errors.New("lectures: tried to put a duplicate crn in lectures table")
		}
		lectures[c.CRN] = c
		instructorID := 0
		instructor, ok := instructors[c.Instructor]
		if !ok {
			fmt.Println("Coudld not find instructor", c.Instructor)
		} else {
			instructorID = instructor.id
		}
		mtitle = max(mtitle, len(c.Title))
		row := [...]string{
			str(c.CRN),
			str(c.Number),
			cleanTitle(c.Title),
			str(c.Units),
			c.Activity,
			str(c.Days),
			c.Time.Start.Format(timeformat),
			c.Time.End.Format(timeformat),
			c.Date.Start.Format(dateformat),
			c.Date.End.Format(dateformat),
			str(instructorID),
			"0",
		}
		if err = w.Write(row[:]); err != nil {
			return nil, err
		}
	}
	w.Flush()
	fmt.Println("Lecture Table:")
	fmt.Println("	max title:", mtitle)
	return lectures, nil
}

func labsDiscTable(sch ucm.Schedule, instructors map[string]*instructorMeta) error {
	f, err := csvfile("labs_disc.csv")
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	for _, c := range sch.Ordered() {
		if c.Activity == Lecture {
			continue
		}
		var lectCRN string
		lect, err := getDiscussionLecture(c, sch)
		if err == nil {
			lectCRN = str(lect.CRN)
		} else {
			lectCRN = "0"
		}
		instructorID := 0
		instructor, ok := instructors[c.Instructor]
		if !ok {
			fmt.Println("Coudld not find instructor", c.Instructor)
		} else {
			instructorID = instructor.id
		}
		row := [...]string{
			str(c.CRN),
			lectCRN,
			str(c.Number),
			c.Section,
			c.Title,
			str(c.Units),
			c.Activity,
			str(c.Days),
			c.Time.Start.Format(timeformat),
			c.Time.End.Format(timeformat),
			c.BuildingRoom,
			str(instructorID),
			"0",
		}
		if err = w.Write(row[:]); err != nil {
			return err
		}
	}
	w.Flush()
	return nil
}

func examsTable(crs []*ucm.Course) error {
	f, err := csvfile("exam.csv")
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	for _, c := range crs {
		if c.Exam == nil {
			continue
		}
		row := [...]string{
			str(c.CRN),
			c.Exam.Date.Format(dateformat),
			c.Time.Start.Format(timeformat),
			c.Time.End.Format(timeformat),
		}
		if err = w.Write(row[:]); err != nil {
			return err
		}
	}
	w.Flush()
	return nil
}

func enrollmentTable(crs []*ucm.Course) error {
	f, err := csvfile("enrollment.csv")
	if err != nil {
		return err
	}
	defer f.Close()
	var (
		w       = csv.NewWriter(f)
		mu      sync.Mutex
		wg      sync.WaitGroup
		workers = 200
		courses = make(chan *ucm.Course)
	)

	go func() {
		for _, c := range crs {
			courses <- c
		}
		close(courses)
	}()
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for c := range courses {
				desc, err := c.Info()
				if err != nil {
					log.Println("Error:", err)
					return
				}
				row := [...]string{
					str(c.CRN),
					desc,
					str(c.Capacity),
					str(c.Enrolled),
					str(c.SeatsOpen()),
					"0",
				}
				mu.Lock()
				err = w.Write(row[:])
				mu.Unlock()
				if err != nil {
					log.Println("Error:", err)
				}
			}
		}()
	}
	wg.Wait()
	w.Flush()
	return nil
}

func writeInstructorTable(crs []*ucm.Course) (map[string]*instructorMeta, error) {
	f, err := csvfile("instructor.csv")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var (
		w           = csv.NewWriter(f)
		instructors = getInstructors(crs)
		maxname     = 0
	)
	for _, inst := range instructors {
		maxname = max(maxname, len(inst.name))
		if err = w.Write([]string{str(inst.id), inst.name}); err != nil {
			panic(err)
		}
	}
	w.Flush()
	fmt.Printf("Instructor Table:\n\tmax name len: %d\n", maxname)
	return instructors, nil
}

func writeAllData(crs []*ucm.Course) error {
	f, err := csvfile("raw_page.csv")
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	row := [...]string{
		"crn",
		"course_number",
		"subject",
		"title",
		"units",
		"activity",
		"days",
		"building",

		"start_time",
		"end_time",
		"start_date",
		"end_date",

		"instructor",
		"max_enrolled",
		"active_enrolled",
		"seats_availible",
	}

	// if err = w.Write(row[:]); err != nil {
	// 	return err
	// }
	for _, c := range crs {
		row = [...]string{
			strconv.FormatInt(int64(c.CRN), 10),
			c.Fullcode,
			c.Subject,
			c.Title,
			strconv.FormatInt(int64(c.Units), 10),
			c.Activity,
			str(c.Days),
			c.BuildingRoom,

			c.Time.Start.String(),
			c.Time.End.String(),
			c.Date.Start.String(),
			c.Date.End.String(),

			c.Instructor,
			str(c.Capacity),
			str(c.Enrolled),
			str(c.SeatsOpen()),
		}
		if err = w.Write(row[:]); err != nil {
			return err
		}
	}
	w.Flush()
	return nil
}
