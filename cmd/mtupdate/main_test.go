package main

import (
	"testing"
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

	// c := RawCourse{
	// 	CRN:       1234,
	// 	Subject:   "CSE",
	// 	CourseNum: 160,
	// 	Title:     "Computer Networks",
	// }

	// q, args, err := goqu.Update("test").Set(
	// 	c,
	// ).From(
	// 	goqu.Select(&RawCourse{}).From("tmp").Where(),
	// ).ToSQL()
	// q, args, err := goqu.Select(c).ToSQL()

	// q, args, err := goqu.Insert("test").Rows(c).ToSQL()
	// if err != nil {
	// 	t.Error(err)
	// }
	// fmt.Println(args)
	// fmt.Println(q)
}

func TestReflection(t *testing.T) {
}
