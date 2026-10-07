package services

import (
	"reflect"
	"testing"
	"time"

	"github.com/sun01822/gitshiny/domain"
)

type fakeSource []domain.Commit

func (f fakeSource) Log(domain.Query) ([]domain.Commit, error) { return f, nil }

func TestCalculate(t *testing.T) {
	src := fakeSource{
		{Files: []domain.FileChange{{Path: "a.go", Added: 10, Removed: 2}, {Path: "b.go", Added: 1}}},
		{Files: []domain.FileChange{{Path: "a.go", Added: 5, Removed: 5}, {Path: "logo.png", Binary: true}}},
	}
	s, err := Calculate(src, domain.Query{Period: "Today"})
	if err != nil {
		t.Fatal(err)
	}
	if s.Added != 16 || s.Removed != 7 || s.NetGrowth() != 9 || s.Commits != 2 || s.FilesChanged != 3 {
		t.Errorf("stats = %+v", s)
	}
}

func TestCalculateEmpty(t *testing.T) {
	s, err := Calculate(fakeSource{}, domain.Query{})
	if err != nil || s.Commits != 0 || s.Added != 0 || s.FilesChanged != 0 {
		t.Errorf("stats = %+v, err = %v", s, err)
	}
}

func TestCalculateGroups(t *testing.T) {
	day := func(d int) time.Time { return time.Date(2026, 10, d, 12, 0, 0, 0, time.Local) }
	src := fakeSource{
		{Author: "Bob", Time: day(2), Files: []domain.FileChange{{Path: "a.go", Added: 1}}},
		{Author: "Alice", Time: day(1), Files: []domain.FileChange{{Path: "a.go", Added: 10, Removed: 2}, {Path: "b.go", Added: 30}}},
		{Author: "Alice", Time: day(2), Files: []domain.FileChange{{Path: "a.go", Added: 5, Removed: 5}}},
	}
	for by, want := range map[string][]domain.Group{
		"author": {{Key: "Alice", Added: 45, Removed: 7, Commits: 2, Files: 2}, {Key: "Bob", Added: 1, Commits: 1, Files: 1}},
		"day":    {{Key: "2026-10-01", Added: 40, Removed: 2, Commits: 1, Files: 2}, {Key: "2026-10-02", Added: 6, Removed: 5, Commits: 2, Files: 1}},
		"file":   {{Key: "b.go", Added: 30, Commits: 1, Files: 1}, {Key: "a.go", Added: 16, Removed: 7, Commits: 3, Files: 1}},
	} {
		s, err := Calculate(src, domain.Query{GroupBy: by})
		if err != nil || !reflect.DeepEqual(s.Groups, want) {
			t.Errorf("by %s = %+v, err = %v", by, s.Groups, err)
		}
	}
	if s, _ := Calculate(src, domain.Query{}); s.Groups != nil {
		t.Errorf("no GroupBy should give no groups: %+v", s.Groups)
	}
}
