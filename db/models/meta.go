package models

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/harrybrwn/edu/school/ucmerced/ucm"
	"github.com/lib/pq"
	"github.com/mercedtime/api/catalog"
)

// GetSchema will get the database schema from a struct
func GetSchema(v interface{}) []string {
	var schema []string
	ty := reflect.TypeOf(v)
	if ty.Kind() == reflect.Ptr {
		ty = ty.Elem()
	}
Outer:
	for i := 0; i < ty.NumField(); i++ {
		fld := ty.Field(i)
		var tag string
		for _, t := range []string{"db"} {
			tag = fld.Tag.Get(t)
			if tag == "-" {
				continue Outer
			}
			if tag != "" {
				break
			}
		}
		if tag != "" {
			schema = append(schema, tag)
		} else {
			schema = append(schema, fld.Name)
		}
	}
	return schema
}

// GetNamedSchema will return a table schema with the named columns
func GetNamedSchema(tableName string, v interface{}) []string {
	schema := GetSchema(v)
	for i := range schema {
		schema[i] = tableName + "." + schema[i]
	}
	return schema
}

// ToCSVRow converts a flat struct to a slice of strings on order
func ToCSVRow(v interface{}) ([]string, error) {
	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Ptr:
		val = val.Elem()
	}
	typ := val.Type()
	var (
		row []string
		s   string
	)
	for i := 0; i < val.NumField(); i++ {
		f := val.Field(i)
		feildType := typ.Field(i)
		if feildType.Tag.Get("db") == "-" || feildType.Tag.Get("csv") == "-" {
			continue
		}

	KindCheck:
		switch f.Kind() {
		case reflect.Ptr:
			f = f.Elem()
			goto KindCheck

		case reflect.String:
			s = f.String()
		case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64:
			s = strconv.FormatInt(f.Int(), 10)
		case reflect.Bool:
			s = strconv.FormatBool(f.Bool())
		case reflect.Float32:
			s = strconv.FormatFloat(f.Float(), 'f', -1, 32)
		case reflect.Float64:
			s = strconv.FormatFloat(f.Float(), 'f', -1, 64)
		case reflect.Struct:
			itf := f.Interface()
			switch itval := itf.(type) {
			case time.Time:
				if strings.Contains(feildType.Name, "Time") {
					// s = itval.Format(TimeFormat)
					if itval.Year() == 0 {
						s = itval.AddDate(1, 0, 0).Format(time.RFC1123Z)
					} else {
						s = itval.Format(time.RFC1123Z)
					}
				} else {
					// s = itval.Format(DateFormat)
					s = itval.Format(time.RFC1123Z)
				}
			case ucm.Exam:
				s = fmt.Sprintf("Exam{%v}", itval.Day.String())
			case struct{ Start, End time.Time }:
				s = itval.Start.Format(DateFormat)
			default:
				return nil, errors.New("cannot handle this struct")
			}
		case reflect.Slice:
			switch arr := f.Interface().(type) {
			case []byte:
				s = string(arr)
			case []time.Weekday:
				s = daysString(arr)
			case []catalog.Weekday:
				s = weekdayStr(arr)
			case catalog.Weekdays:
				s = weekdayStr(arr)
			default:
				return nil, errors.New("can't handle arrays")
			}
		case reflect.Invalid:
			s = "<nil>"
		default:
			panic(fmt.Sprintf("csv generator doesn't support this type \"%s\"", f.Kind()))
		}
		row = append(row, s)
	}
	return row, nil
}

func daysString(days []time.Weekday) string {
	var s = make([]string, len(days))
	for i, d := range days {
		s[i] = d.String()
	}
	return strings.Join(s, ";")
}

func weekdayStr(days []catalog.Weekday) string {
	arr := pq.Array(days)
	val, err := arr.Value()
	if err != nil {
		return "{}"
	}
	if val == nil {
		return "{}"
	}
	return val.(string)
}
