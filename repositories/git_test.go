package repositories

import (
	"strings"
	"testing"
	"time"

	"github.com/sun01822/gitshiny/domain"
)

const sample = `@@gitshiny@@aaa	Alice	alice@example.com	1790000000
10	2	main.go
3	0	README.md

@@gitshiny@@bbb	Bob	bob@example.com	1790000100
-	-	logo.png
5	5	main.go
`

func TestParseLog(t *testing.T) {
	commits, err := ParseLog(strings.NewReader(sample))
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 2 {
		t.Fatalf("got %d commits, want 2", len(commits))
	}
	a := commits[0]
	if a.Hash != "aaa" || a.Author != "Alice" || a.Email != "alice@example.com" || len(a.Files) != 2 {
		t.Errorf("commit 0 = %+v", a)
	}
	if a.Files[0].Added != 10 || a.Files[0].Removed != 2 || a.Files[0].Path != "main.go" {
		t.Errorf("file = %+v", a.Files[0])
	}
	if !commits[1].Files[0].Binary {
		t.Error("expected binary file to be flagged")
	}
}

func TestParseLogMalformedHeader(t *testing.T) {
	if _, err := ParseLog(strings.NewReader("@@gitshiny@@onlyhash\n")); err == nil {
		t.Error("expected error for malformed header")
	}
}

func TestLogArgs(t *testing.T) {
	loc := time.FixedZone("BDT", 6*3600)
	q := domain.Query{
		Authors: []string{"Alice (A)", "Bob"},
		Since:   time.Date(2026, 10, 1, 0, 0, 0, 0, loc),
		Until:   time.Date(2026, 10, 1, 23, 59, 59, 0, loc),
		Branch:  "dev",
		Paths:   []string{"cmd/"},
		Exclude: []string{"go.sum"},
	}
	args := strings.Join(LogArgs(q), "|")
	for _, want := range []string{
		"--fixed-strings", "--author=Alice (A)", "--author=Bob",
		"--since=2026-10-01 00:00:00 +0600", "--until=2026-10-01 23:59:59 +0600", "|dev|--|cmd/|:(exclude)go.sum",
	} {
		if !strings.Contains(args, want) {
			t.Errorf("args %q missing %q", args, want)
		}
	}
	if all := strings.Join(LogArgs(domain.Query{AllBranches: true}), "|"); !strings.Contains(all, "--all") {
		t.Error("expected --all")
	}
}
