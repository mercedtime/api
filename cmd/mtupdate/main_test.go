package main

import (
	"sync"
	"testing"

	"github.com/harrybrwn/edu/school/ucmerced/ucm"
)

var (
	testingSchedule ucm.Schedule
	scheduleOnce    sync.Once
	scheduleMu      sync.Mutex
)

func testSchedule(t *testing.T) ucm.Schedule {
	t.Helper()
	var (
		err  error
		sch  ucm.Schedule = make(ucm.Schedule)
		conf              = ucm.ScheduleConfig{Year: 2021, Term: "spring"}
	)
	scheduleOnce.Do(func() {
		testingSchedule, err = ucm.NewSchedule(conf)
		if err != nil {
			t.Fatal(err)
		}
	})
	scheduleMu.Lock()
	for key, val := range testingSchedule {
		var c = new(ucm.Course) // make copies bc they are pointers
		*c = *val
		sch[key] = c
	}
	scheduleMu.Unlock()
	return testingSchedule
}

func TestGetDiscussionLecture(t *testing.T) {
	sch := testSchedule(t)
	var c *ucm.Course
	for _, c = range sch.Ordered() {
		if c.Activity == "DISC" {
			break
		}
	}
	lect, err := getDiscussionLecture(c, sch)
	if err != nil {
		t.Fatal(err)
	}
	if lect.Number != c.Number {
		t.Error("lecture for a discussion should have the same course number as the disc")
	}
}

func TestGetCourseTable(t *testing.T) {
	sch := testSchedule(t)
	list := sch.Ordered()
	courses, err := GetCourseTable(list, 200)
	if err != nil {
		t.Error(err)
	}
	if len(courses) == 0 {
		t.Error("dummy check failed, did not return any courses")
	}
	if len(courses) != len(list) {
		t.Error("GetCourseTable: goroutines finished too early, expected longer result")
	}
}
