// Package repositories talks to Git by shelling out to the git binary.
package repositories

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/sun01822/gitshiny/domain"
)

var (
	// ErrNotRepo is returned when the directory is not inside a Git work tree.
	ErrNotRepo = errors.New("this directory is not a Git repository")
	// ErrNoGit is returned when the git binary is not installed.
	ErrNoGit = errors.New("git is not installed or not in PATH")
)

const (
	marker     = "@@gitshiny@@"
	timeFormat = "2006-01-02 15:04:05 -0700"
)

// Git reads data from a repository on disk.
type Git struct {
	Dir string // working directory; empty = current directory
}

func (g Git) run(args ...string) (string, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return "", ErrNoGit
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = g.Dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

// Root returns the top-level directory of the repository.
func (g Git) Root() (string, error) {
	out, err := g.run("rev-parse", "--show-toplevel")
	if err != nil {
		if errors.Is(err, ErrNoGit) {
			return "", err
		}
		return "", ErrNotRepo
	}
	return out, nil
}

// UserName returns `git config user.name` (empty if unset).
func (g Git) UserName() string {
	out, _ := g.run("config", "user.name")
	return out
}

// Branch returns the current branch name, or "HEAD" when detached.
func (g Git) Branch() string {
	out, err := g.run("symbolic-ref", "--short", "-q", "HEAD")
	if err != nil || out == "" {
		return "HEAD"
	}
	return out
}

// LogArgs builds the `git log` arguments for a query.
func LogArgs(q domain.Query) []string {
	args := []string{
		"log", "--no-merges", "--numstat", "--fixed-strings",
		"--pretty=format:" + marker + "%H%x09%an%x09%ae%x09%at",
	}
	for _, a := range q.Authors {
		args = append(args, "--author="+a)
	}
	if !q.Since.IsZero() {
		args = append(args, "--since="+q.Since.Format(timeFormat))
	}
	if !q.Until.IsZero() {
		args = append(args, "--until="+q.Until.Format(timeFormat))
	}
	switch {
	case q.AllBranches:
		args = append(args, "--all")
	case q.Branch != "":
		args = append(args, q.Branch)
	}
	args = append(args, "--")
	args = append(args, q.Paths...)
	for _, p := range q.Exclude {
		args = append(args, ":(exclude)"+p)
	}
	return args
}

// Log returns the commits matching q.
func (g Git) Log(q domain.Query) ([]domain.Commit, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return nil, ErrNoGit
	}
	cmd := exec.Command("git", LogArgs(q)...)
	cmd.Dir = g.Dir
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	commits, parseErr := ParseLog(stdout)
	_, _ = io.Copy(io.Discard, stdout) // never leave git blocked on a full pipe
	if err := cmd.Wait(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if strings.Contains(msg, "does not have any commits yet") {
			return nil, nil
		}
		return nil, fmt.Errorf("git log failed: %s", msg)
	}
	return commits, parseErr
}

// ParseLog parses the output of the log format produced by LogArgs.
func ParseLog(r io.Reader) ([]domain.Commit, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 8*1024*1024)

	var commits []domain.Commit
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, marker):
			parts := strings.SplitN(strings.TrimPrefix(line, marker), "\t", 4)
			if len(parts) != 4 {
				return commits, fmt.Errorf("malformed commit header: %q", line)
			}
			ts, err := strconv.ParseInt(parts[3], 10, 64)
			if err != nil {
				return commits, fmt.Errorf("bad commit timestamp %q: %w", parts[3], err)
			}
			commits = append(commits, domain.Commit{
				Hash: parts[0], Author: parts[1], Email: parts[2], Time: time.Unix(ts, 0),
			})
		case line == "" || len(commits) == 0:
			continue
		default:
			f := strings.SplitN(line, "\t", 3)
			if len(f) != 3 {
				continue
			}
			fc := domain.FileChange{Path: f[2]}
			if f[0] == "-" && f[1] == "-" {
				fc.Binary = true // binary files have no line counts
			} else {
				a, errA := strconv.Atoi(f[0])
				d, errD := strconv.Atoi(f[1])
				if errA != nil || errD != nil {
					continue
				}
				fc.Added, fc.Removed = a, d
			}
			c := &commits[len(commits)-1]
			c.Files = append(c.Files, fc)
		}
	}
	return commits, sc.Err()
}
