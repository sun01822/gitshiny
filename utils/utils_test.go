package utils

import (
	"testing"
	"time"
)

func TestCommasAndSigned(t *testing.T) {
	cases := map[int]string{0: "0", 7: "7", 999: "999", 1842: "1,842", 1234567: "1,234,567", -1451: "-1,451"}
	for n, want := range cases {
		if got := Commas(n); got != want {
			t.Errorf("Commas(%d) = %q, want %q", n, got, want)
		}
	}
	if Signed(1451) != "+1,451" || Signed(-3) != "-3" || Signed(0) != "0" {
		t.Error("Signed formatting wrong")
	}
}

func TestParseTime(t *testing.T) {
	loc := time.UTC
	got, err := ParseTime("2026-10-01 09:30:15", loc, false)
	if err != nil || !got.Equal(time.Date(2026, 10, 1, 9, 30, 15, 0, loc)) {
		t.Fatalf("full timestamp: %v %v", got, err)
	}
	got, _ = ParseTime("2026-10-01", loc, true)
	if got.Hour() != 23 || got.Minute() != 59 || got.Second() != 59 {
		t.Errorf("end-of-day date = %v", got)
	}
	got, _ = ParseTime("2026-10-01", loc, false)
	if got.Hour() != 0 {
		t.Errorf("start-of-day date = %v", got)
	}
	if _, err := ParseTime("2026-10-01T09:00:00+06:00", loc, false); err != nil {
		t.Errorf("RFC3339: %v", err)
	}
	if _, err := ParseTime("yesterday-ish", loc, false); err == nil {
		t.Error("expected error for invalid input")
	}
}

func TestDayRange(t *testing.T) {
	s, e := DayRange(time.Date(2026, 10, 2, 15, 4, 5, 0, time.UTC))
	if s.Format("15:04:05") != "00:00:00" || e.Format("15:04:05") != "23:59:59" || s.Day() != 2 || e.Day() != 2 {
		t.Errorf("DayRange = %v .. %v", s, e)
	}
}

func TestWeekStart(t *testing.T) {
	for day, want := range map[int]string{
		2: "2026-09-28", // Friday, back across the month boundary
		4: "2026-09-28", // Sunday belongs to the week that began on Monday
		5: "2026-10-05", // Monday is its own start
	} {
		got := WeekStart(time.Date(2026, 10, day, 15, 4, 5, 0, time.UTC))
		if got.Format("2006-01-02 15:04:05") != want+" 00:00:00" {
			t.Errorf("WeekStart(Oct %d) = %v, want %s", day, got, want)
		}
	}
}
