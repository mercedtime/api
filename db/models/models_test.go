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
	if row[1] != now.Format(DateFormat) {
		t.Errorf("bad time string: got %v, want %v", row[1], now.String())
	}
	if row[2] != now.Format(TimeFormat) {
		t.Errorf("bad time string: got %v, want %v", row[1], now.Format(TimeFormat))
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
