package main

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/harrybrwn/edu/school/ucmerced/ucm"
)

var (
	testingSchedule ucm.Schedule
	scheduleOnce    sync.Once
	scheduleMu      sync.Mutex

	subjects = []string{"CSE", "BIO", "CHEM", "MATH", "PHYS", "ENGR", "ECON", "GASP", "ME"}
)

func Test(t *testing.T) {
	s := testSchedule(t)
	bp, err := findBlueprints(s.Ordered())
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(len(s), len(bp))
}

func testSchedule(t *testing.T) ucm.Schedule {
	t.Helper()
	var (
		err error
		sch ucm.Schedule = make(ucm.Schedule)
	)
	scheduleOnce.Do(func() {
		rand.Seed(time.Now().Unix())
		testingSchedule, err = ucm.NewSchedule(ucm.ScheduleConfig{
			Year: 2021,
			Term: "spring",
			// Subject: subjects[rand.Intn(len(subjects))],
		})
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
		// Get the first discussion
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

func TestPopulateTables(t *testing.T) {
	sch := testSchedule(t)
	tab, err := PopulateTables(sch, &updateConfig{})
	if err != nil {
		t.Error(err)
	}
	if len(tab.course) != len(tab.aux)+len(tab.lectures) {
		t.Error("should have same number of courses and subcourses as the catalog")
	}
}

func TestDetectSemester(t *testing.T) {
	// var tm time.Time
	// // tm = time.Date(2020, time.January, 4, 1, 1, 1, 1, time.FixedZone("America/Los_Angeles", 0))
	// tm = time.Date(2020, time.January, 4, 1, 1, 1, 1, time.UTC)
	// fmt.Println(tm)
}
