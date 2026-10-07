// Package utils contains small shared helpers.
package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

var timeLayouts = []string{
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
	"2006-01-02T15:04:05",
	"2006-01-02T15:04",
	time.RFC3339,
	"2006-01-02",
}

// ParseTime parses a user supplied timestamp. Layouts without a zone are read
// in loc. A bare date (YYYY-MM-DD) means 00:00:00, or 23:59:59 when endOfDay.
func ParseTime(s string, loc *time.Location, endOfDay bool) (time.Time, error) {
	s = strings.TrimSpace(s)
	for _, layout := range timeLayouts {
		t, err := time.ParseInLocation(layout, s, loc)
		if err != nil {
			continue
		}
		if layout == "2006-01-02" && endOfDay {
			t = time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, loc)
		}
		return t, nil
	}
	return time.Time{}, fmt.Errorf("invalid time %q (use YYYY-MM-DD HH:MM:SS, YYYY-MM-DD HH:MM, YYYY-MM-DD or RFC3339)", s)
}

// DayRange returns 00:00:00 and 23:59:59 of the day containing t, in t's zone.
func DayRange(t time.Time) (time.Time, time.Time) {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location()),
		time.Date(y, m, d, 23, 59, 59, 0, t.Location())
}

// WeekRange returns Monday 00:00:00 of t's week and 23:59:59 of t's day.
func WeekRange(t time.Time) (time.Time, time.Time) {
	start, end := DayRange(t)
	return start.AddDate(0, 0, -(int(t.Weekday())+6)%7), end
}

// MonthRange returns 00:00:00 on the 1st of t's month and 23:59:59 of t's day.
func MonthRange(t time.Time) (time.Time, time.Time) {
	start, end := DayRange(t)
	return start.AddDate(0, 0, 1-t.Day()), end
}

// Commas formats n with thousands separators: 1842 -> "1,842".
func Commas(n int) string {
	s := strconv.Itoa(n)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	if neg {
		s = "-" + s
	}
	return s
}

// Signed is like Commas but prefixes positive numbers with "+".
func Signed(n int) string {
	if n > 0 {
		return "+" + Commas(n)
	}
	return Commas(n)
}
