package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"path/filepath"

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
	courses []*ucm.Course,
	instructors map[string]*instructorMeta,
) error {
	f, err := csvfile("lecture.csv")
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	lectures, err := getLectures(courses, instructors)

	for _, l := range lectures {
		row, err := models.ToCSVRow(l)
		if err != nil {
			return err
		}
		if err = w.Write(row); err != nil {
			return err
		}
	}
	w.Flush()
	return nil
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
	)
	for _, inst := range instructors {
		if err = w.Write([]string{
			str(inst.id),
			inst.name,
		}); err != nil {
			panic(err)
		}
	}
	w.Flush()
	return instructors, nil
}
