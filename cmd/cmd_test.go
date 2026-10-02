package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// newRepo builds a repo: Alice adds 3 lines (+3), then +2/-1 on 2026-10-01;
// Bob adds 10 lines on 2026-09-30.
func newRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	dir := t.TempDir()
	git := func(date string, args ...string) {
		t.Helper()
		c := exec.Command("git", args...)
		c.Dir = dir
		c.Env = append(os.Environ(), "GIT_AUTHOR_DATE="+date, "GIT_COMMITTER_DATE="+date)
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	write := func(name, body string) {
		if err := os.WriteFile(dir+"/"+name, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git("2026-09-30T08:00:00Z", "init", "-q", "-b", "main")
	git("2026-09-30T08:00:00Z", "config", "user.name", "Alice")
	git("2026-09-30T08:00:00Z", "config", "user.email", "alice@example.com")

	write("b.txt", strings.Repeat("x\n", 10))
	git("2026-09-30T10:00:00Z", "add", ".")
	git("2026-09-30T10:00:00Z", "-c", "user.name=Bob", "-c", "user.email=bob@example.com", "commit", "-q", "-m", "bob")

	write("a.txt", "1\n2\n3\n")
	git("2026-10-01T09:00:00Z", "add", ".")
	git("2026-10-01T09:00:00Z", "commit", "-q", "-m", "a1")

	write("a.txt", "1\n2\nX\n4\n")
	git("2026-10-01T10:00:00Z", "add", ".")
	git("2026-10-01T10:00:00Z", "commit", "-q", "-m", "a2")
	return dir
}

func run(t *testing.T, dir string, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := Execute(App{
		In: strings.NewReader(""), Out: &out, Err: &errb, Version: "test", Dir: dir,
		Now: func() time.Time { return time.Date(2026, 10, 1, 12, 30, 0, 0, time.UTC) },
	}, args)
	return code, out.String(), errb.String()
}

type result struct {
	Author       string `json:"author"`
	Period       string `json:"period"`
	Branch       string `json:"branch"`
	Added        int    `json:"added"`
	Removed      int    `json:"removed"`
	NetGrowth    int    `json:"net_growth"`
	Commits      int    `json:"commits"`
	FilesChanged int    `json:"files_changed"`
}

func stats(t *testing.T, dir string, args ...string) result {
	t.Helper()
	code, out, errs := run(t, dir, append([]string{"stats", "--format", "json"}, args...)...)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errs)
	}
	var r result
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatalf("bad json %q: %v", out, err)
	}
	return r
}

func TestStatsTodayDefaultsToGitUser(t *testing.T) {
	r := stats(t, newRepo(t), "--today")
	want := result{Author: "Alice", Period: "Today", Branch: "main", Added: 5, Removed: 1, NetGrowth: 4, Commits: 2, FilesChanged: 1}
	if r != want {
		t.Errorf("got %+v, want %+v", r, want)
	}
}

func TestStatsYesterdayOtherAuthor(t *testing.T) {
	r := stats(t, newRepo(t), "--yesterday", "--author", "Bob")
	if r.Added != 10 || r.Removed != 0 || r.Commits != 1 || r.Period != "Yesterday" {
		t.Errorf("got %+v", r)
	}
}

func TestStatsCustomRangeAllAuthors(t *testing.T) {
	r := stats(t, newRepo(t), "--all-authors", "--since", "2026-09-30", "--until", "2026-10-01")
	if r.Added != 15 || r.Removed != 1 || r.Commits != 3 || r.FilesChanged != 2 || r.Author != "All authors" {
		t.Errorf("got %+v", r)
	}
}

func TestStatsMultipleAuthors(t *testing.T) {
	r := stats(t, newRepo(t), "--author", "Alice,Bob", "--since", "2026-09-30", "--until", "2026-10-01")
	if r.Commits != 3 {
		t.Errorf("got %+v", r)
	}
}

func TestStatsCSVAndText(t *testing.T) {
	dir := newRepo(t)
	code, out, _ := run(t, dir, "stats", "--today", "--format", "csv")
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if code != 0 || len(lines) != 2 || !strings.HasPrefix(lines[0], "repository,author") {
		t.Fatalf("csv = %q", out)
	}
	code, out, _ = run(t, dir, "stats", "--today")
	if code != 0 || !strings.Contains(out, "Added lines    : 5") || !strings.Contains(out, "Net growth     : +4") {
		t.Errorf("text = %q", out)
	}
}

func TestErrors(t *testing.T) {
	dir := newRepo(t)
	if code, _, e := run(t, t.TempDir(), "stats"); code != 1 || !strings.Contains(e, "not a Git repository") {
		t.Errorf("non-repo: code %d, %q", code, e)
	}
	if code, _, _ := run(t, dir, "stats", "--today", "--yesterday"); code != 2 {
		t.Errorf("conflicting flags code = %d", code)
	}
	if code, _, _ := run(t, dir, "stats", "--format", "xml"); code != 2 {
		t.Errorf("bad format code = %d", code)
	}
	if code, _, _ := run(t, dir, "stats", "--since", "2026-10-02", "--until", "2026-10-01"); code != 2 {
		t.Errorf("reversed range code = %d", code)
	}
	if code, _, _ := run(t, dir, "nope"); code != 2 {
		t.Errorf("unknown command code = %d", code)
	}
}

func TestEmptyRepoAndVersion(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	dir := t.TempDir()
	for _, a := range [][]string{{"init", "-q"}, {"config", "user.name", "Z"}} {
		c := exec.Command("git", a...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
	}
	if r := stats(t, dir); r.Commits != 0 || r.Added != 0 {
		t.Errorf("empty repo = %+v", r)
	}
	if _, out, _ := run(t, dir, "version"); strings.TrimSpace(out) != "gitshiny test" {
		t.Errorf("version = %q", out)
	}
}
