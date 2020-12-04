package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/harrybrwn/edu/school/ucmerced/ucm"
	"github.com/mercedtime/api/db/models"
)

func TestReflection(t *testing.T) {
	now := time.Now()
	exam := &models.Exam{
		CRN:  123456,
		Date: now, StartTime: time.Now(), EndTime: time.Now()}
	row, err := toCsvRow(exam)
	if err != nil {
		t.Fatal(err)
	}
	if row[0] != "123456" {
		t.Errorf("bad crn: %v", row[0])
	}
	if row[1] != now.Format(dateformat) {
		t.Errorf("bad time string: got %v, want %v", row[1], now.String())
	}
}

func TestToCsv(t *testing.T) {
	conf := ucm.ScheduleConfig{Year: 2021, Term: "spring"}
	sch, err := ucm.NewSchedule(conf)
	if err != nil {
		t.Fatal(err)
	}

	courses := sch.Ordered()
	for _, c := range courses {
		l := &models.Lect{
			CRN:          c.CRN,
			CourseNum:    c.CourseNumber(),
			Title:        c.Title,
			Units:        c.Units,
			Activity:     c.Activity,
			Days:         str(c.Days),
			StartTime:    c.Time.Start,
			EndTime:      c.Time.End,
			StartDate:    c.Date.Start,
			EndDate:      c.Date.End,
			InstructorID: 0,
		}
		line, err := toCsvRow(l)
		if err != nil {
			t.Error(err)
		}
		fmt.Println(line)
		if len(line) == 0 {
			t.Error("line has no length")
		}
	}
}
