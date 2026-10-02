package tui

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/sun01822/gitshiny/domain"
)

func runMenu(t *testing.T, input string) (string, []domain.Query, error) {
	t.Helper()
	var out bytes.Buffer
	var seen []domain.Query
	err := Run(Options{
		In: strings.NewReader(input), Out: &out, Author: "Alice",
		Now: func() time.Time { return time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC) },
		Calculate: func(q domain.Query) (domain.Stats, error) {
			seen = append(seen, q)
			return domain.Stats{Period: q.Period, Added: 1}, nil
		},
		Render: func(w io.Writer, s domain.Stats) error {
			_, err := io.WriteString(w, "RESULT "+s.Period+"\n")
			return err
		},
	})
	return out.String(), seen, err
}

func TestMenuChoices(t *testing.T) {
	out, seen, err := runMenu(t, "1\n")
	if err != nil || len(seen) != 1 || seen[0].Period != "Today" || !strings.Contains(out, "RESULT Today") {
		t.Fatalf("today: %v %v %q", err, seen, out)
	}
	if seen[0].Since.Day() != 2 {
		t.Errorf("today since = %v", seen[0].Since)
	}
	_, seen, _ = runMenu(t, "2\n")
	if seen[0].Period != "Yesterday" || seen[0].Since.Day() != 1 {
		t.Errorf("yesterday = %+v", seen[0])
	}
	_, seen, err = runMenu(t, "3\n2026-10-01 09:00:00\n2026-10-01 18:00:00\n")
	if err != nil || seen[0].Period != "Custom" || seen[0].Until.Hour() != 18 {
		t.Errorf("custom = %+v %v", seen, err)
	}
}

func TestMenuQuitAndInvalid(t *testing.T) {
	if _, seen, err := runMenu(t, "q\n"); err != nil || len(seen) != 0 {
		t.Errorf("quit: %v %v", seen, err)
	}
	if _, _, err := runMenu(t, ""); err != nil {
		t.Errorf("EOF should quit quietly: %v", err)
	}
	if _, _, err := runMenu(t, "9\n"); err == nil {
		t.Error("expected error for invalid option")
	}
	if _, _, err := runMenu(t, "3\n2026-10-02\n2026-10-01\n"); err == nil {
		t.Error("expected error for reversed range")
	}
}
