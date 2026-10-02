package services

import (
	"testing"

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
