package main

import (
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/mercedtime/api/db/models"
)

func daysString(days []time.Weekday) string {
	var s = make([]string, len(days))
	for i, d := range days {
		s[i] = strings.ToLower(d.String())
	}
	// return strings.Join(s, ";")

	arr := pq.Array(s)
	val, err := arr.Value()
	if err != nil {
		return "{}"
	}
	return val.(string)
}

func str(x interface{}) string {
	switch v := x.(type) {
	case int:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case []time.Weekday:
		return daysString(v)
	case time.Time:
		if v.Equal(time.Time{}) {
			return ""
		} else if v.Hour() == 0 && v.Minute() == 0 && v.Second() == 0 {
			return v.Format(models.DateFormat)
		} else if v.Year() == 0 && v.Month() == time.January && v.Day() == 1 {
			return v.Format(models.TimeFormat)
		}
		return ""
	default:
		return ""
	}
}

func instructorMapToInterfaceSlice(m map[string]*models.Instructor) []interface{} {
	var (
		arr = make([]interface{}, len(m))
		i   = 0
	)
	for _, inst := range m {
		arr[i] = inst
		i++
	}
	return arr
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minLen(list ...string) string {
	var ret = list[0]
	for _, s := range list {
		if len(s) < len(ret) {
			ret = s
		}
	}
	return ret
}

func mapKeys(m map[string]struct{}) []string {
	var (
		i   = 0
		arr = make([]string, len(m))
	)
	for k := range m {
		arr[i] = k
		i++
	}
	return arr
}
