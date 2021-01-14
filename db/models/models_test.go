package models

import (
	"testing"
	"time"

	"github.com/mercedtime/api/catalog"
)

func TestCSVRow(t *testing.T) {
	now := time.Now()
	exam := &Exam{
		CRN:  123456,
		Date: now, StartTime: now, EndTime: now}
	row, err := ToCSVRow(exam)
	if err != nil {
		t.Fatal(err)
	}
	if row[0] != "123456" {
		t.Errorf("bad crn: %v", row[0])
	}

	l := catalog.Entry{
		CRN:       12345,
		Subject:   "CSE",
		CourseNum: 31,
		Type:      "LAB",
		Title:     "testing course",
	}
	row, err = ToCSVRow(l)
	if err != nil {
		t.Error(err)
	}
	if row[0] != "12345" {
		t.Error("bad csv row conversion")
	}
	if row[1] != "CSE" {
		t.Error("bad csv row conversion")
	}
	if row[2] != "31" {
		t.Error("bad csv row conversion")
	}
	if row[3] != "LAB" {
		t.Error("bad csv row conversion")
	}
	if row[4] != "testing course" {
		t.Error("bad csv row conversion")
	}
}

func TestGetSchema(t *testing.T) {
	type A struct {
		A string `db:"a"`
		B int    `db:"b"`
		C bool   `db:"-"`
	}
	sc := GetSchema(A{})
	if len(sc) != 2 {
		t.Fatal("bad length")
	}
	if sc[0] != "a" {
		t.Error("bad column name")
	}
	if sc[1] != "b" {
		t.Error("bad column name")
	}
}
