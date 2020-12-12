package main

import (
	"fmt"
	"testing"

	"github.com/doug-martin/goqu/v9"
)

func Test(t *testing.T) {
	// conf := ucm.ScheduleConfig{Year: 2021, Term: "spring"}
	// sch, err := ucm.NewSchedule(conf)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// err = fullUpdate(nil, sch)
	// if err != nil {
	// 	t.Error(err)
	// }

	c := RawCourse{
		CRN:       1234,
		Subject:   "CSE",
		CourseNum: 160,
		Title:     "Computer Networks",
	}
	// q, args, err := goqu.Update("test").Set(
	// 	c,
	// ).From(
	// 	goqu.Select(&RawCourse{}).From("tmp").Where(),
	// ).ToSQL()
	// q, args, err := goqu.Select(c).ToSQL()
	q, args, err := goqu.Insert("test").Rows(c).ToSQL()
	if err != nil {
		t.Error(err)
	}
	fmt.Println(args)
	fmt.Println(q)
}

func TestReflection(t *testing.T) {
}

func TestToCsv(t *testing.T) {
	// conf := ucm.ScheduleConfig{Year: 2021, Term: "spring"}
	// sch, err := ucm.NewSchedule(conf)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// courses := sch.Ordered()
	// for _, c := range courses {
	// 	l := &models.Lect{
	// 		CRN:          c.CRN,
	// 		CourseNum:    c.CourseNumber(),
	// 		Title:        c.Title,
	// 		Units:        c.Units,
	// 		Activity:     c.Activity,
	// 		Days:         str(c.Days),
	// 		StartTime:    c.Time.Start,
	// 		EndTime:      c.Time.End,
	// 		StartDate:    c.Date.Start,
	// 		EndDate:      c.Date.End,
	// 		InstructorID: 0,
	// 	}
	// 	line, err := toCsvRow(l)
	// 	if err != nil {
	// 		t.Error(err)
	// 	}
	// 	fmt.Println(line)
	// 	if len(line) == 0 {
	// 		t.Error("line has no length")
	// 	}
	// }
}
