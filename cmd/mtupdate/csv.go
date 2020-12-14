package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/harrybrwn/edu/school/ucmerced/ucm"
	"github.com/mercedtime/api/db/models"
	"github.com/pkg/errors"
)

var csvOutDir = "data"

func init() {
	flag.StringVar(&csvOutDir, "out", csvOutDir, "output directory for csv files")
}

func csvfile(name string) (*os.File, error) {
	return os.OpenFile(filepath.Join(csvOutDir, name), os.O_CREATE|os.O_WRONLY, 0644)
}

func courseTable(courses []*ucm.Course) error {
	f, err := csvfile("course.csv")
	if err != nil {
		return err
	}
	defer f.Close()
	var w = csv.NewWriter(f)
	table, err := GetCourseTable(courses, 200)
	if err != nil {
		return err
	}
	for _, c := range table {
		row, err := models.ToCSVRow(c)
		if err != nil {
			return err
		}
		if err = w.Write(row); err != nil {
			return errors.Wrap(err, "could not write to csv file")
		}
	}
	w.Flush()
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
			fmt.Println("Coudld not find instructor", c.Instructor)
		} else {
			instructorID = instructor.id
		}
		mtitle = max(mtitle, len(c.Title))
		// For type safety and so i get error messages
		// when the schema changes
		l := models.Lecture{
			CRN:          c.CRN,
			Units:        c.Units,
			Days:         str(c.Days),
			StartTime:    c.Time.Start,
			EndTime:      c.Time.End,
			StartDate:    c.Date.Start,
			EndDate:      c.Date.End,
			InstructorID: instructorID,
		}
		row, err := models.ToCSVRow(&l)
		if err != nil {
			log.Println("Could not create lecture row:", err)
			continue
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
		if c.Activity == models.Lect {
			continue
		}
		var lectCRN int
		lect, err := getDiscussionLecture(c, sch)
		if err == nil {
			lectCRN = lect.CRN
		} else {
			// TODO handle this case better its making
			// database managment harder
			lectCRN = 0
		}
		instructorID := 0
		instructor, ok := instructors[c.Instructor]
		if !ok {
			fmt.Println("Coudld not find instructor", c.Instructor)
		} else {
			instructorID = instructor.id
		}
		l := models.LabDisc{
			CRN:          c.CRN,
			CourseCRN:    lectCRN,
			Section:      c.Section,
			Units:        c.Units,
			Days:         str(c.Days),
			StartTime:    c.Time.Start,
			EndTime:      c.Time.End,
			Building:     c.BuildingRoom,
			InstructorID: instructorID,
		}
		row, err := models.ToCSVRow(l)
		if err != nil {
			return err
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
		e := models.Exam{
			CRN:       c.CRN,
			Date:      c.Exam.Date,
			StartTime: c.Time.Start,
			EndTime:   c.Time.End,
		}
		row, err := models.ToCSVRow(e)
		if err != nil {
			return err
		}
		if err = w.Write(row[:]); err != nil {
			return err
		}
	}
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
		if err = w.Write([]string{
			str(inst.id),
			inst.name,
			"0",
		}); err != nil {
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
			cleanTitle(c.Title),
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
