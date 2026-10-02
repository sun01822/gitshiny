// Package cmd implements the gitshiny command line.
package cmd

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/x/term"

	"github.com/sun01822/gitshiny/config"
	"github.com/sun01822/gitshiny/domain"
	"github.com/sun01822/gitshiny/repositories"
	"github.com/sun01822/gitshiny/services"
	"github.com/sun01822/gitshiny/tui"
	"github.com/sun01822/gitshiny/utils"
)

// App carries the process environment so the CLI is easy to test.
type App struct {
	In      io.Reader
	Out     io.Writer
	Err     io.Writer
	Version string
	Dir     string           // starting directory; empty = current
	Now     func() time.Time // defaults to time.Now
}

const usage = `GitShiny - Git contribution statistics

Usage:
  gitshiny                       interactive TUI
  gitshiny stats [flags]         print statistics (automation friendly)
  gitshiny version
  gitshiny help

Flags for "stats":
  --today               today (default when no range is given)
  --yesterday           yesterday
  --since  "TIME"       start of a custom range
  --until  "TIME"       end of a custom range
  --author NAME         author to include (repeatable or comma separated;
                        default: git config user.name)
  --all-authors         include every author
  --branch NAME         branch to analyse (default: current branch)
  --all-branches        analyse all branches
  --format text|json|csv   output format (default: text, env GITSHINY_FORMAT)
  -C DIR                run as if started in DIR

TIME is "YYYY-MM-DD HH:MM:SS", "YYYY-MM-DD HH:MM", "YYYY-MM-DD" or RFC3339.

Examples:
  gitshiny stats --today --format json
  gitshiny stats --since "2026-10-01 09:00:00" --until "2026-10-01 18:00:00"
  gitshiny stats --yesterday --author Alice --author Bob --format csv
`

// Execute runs the CLI and returns the process exit code.
func Execute(app App, args []string) int {
	if app.Now == nil {
		app.Now = time.Now
	}
	if len(args) == 0 {
		if isTerminal(app.In) && isTerminal(app.Out) {
			return runInteractive(app)
		}
		fmt.Fprint(app.Out, usage)
		return 0
	}
	switch args[0] {
	case "stats":
		return runStats(app, args[1:])
	case "version", "--version", "-v":
		fmt.Fprintf(app.Out, "gitshiny %s\n", app.Version)
		return 0
	case "help", "--help", "-h":
		fmt.Fprint(app.Out, usage)
		return 0
	}
	fmt.Fprintf(app.Err, "Error: unknown command %q\n\n%s", args[0], usage)
	return 2
}

type listFlag []string

func (l *listFlag) String() string { return strings.Join(*l, ",") }
func (l *listFlag) Set(v string) error {
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			*l = append(*l, p)
		}
	}
	return nil
}

func runStats(app App, args []string) int {
	cfg := config.Load()
	fs := flag.NewFlagSet("stats", flag.ContinueOnError)
	fs.SetOutput(app.Err)
	today := fs.Bool("today", false, "")
	yesterday := fs.Bool("yesterday", false, "")
	since := fs.String("since", "", "")
	until := fs.String("until", "", "")
	var authors listFlag
	fs.Var(&authors, "author", "")
	allAuthors := fs.Bool("all-authors", false, "")
	branch := fs.String("branch", "", "")
	allBranches := fs.Bool("all-branches", false, "")
	format := fs.String("format", cfg.Format, "")
	dir := fs.String("C", app.Dir, "")
	fs.Usage = func() { fmt.Fprint(app.Err, usage) }
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() > 0 {
		return fail(app, 2, fmt.Errorf("unexpected argument %q", fs.Arg(0)))
	}

	*format = strings.ToLower(*format)
	if *format != "text" && *format != "json" && *format != "csv" {
		return fail(app, 2, fmt.Errorf("unknown format %q (use text, json or csv)", *format))
	}
	custom := *since != "" || *until != ""
	if n := b2i(*today) + b2i(*yesterday) + b2i(custom); n > 1 {
		return fail(app, 2, errors.New("choose one of --today, --yesterday or --since/--until"))
	}
	if *allBranches && *branch != "" {
		return fail(app, 2, errors.New("--branch and --all-branches cannot be combined"))
	}
	if *allAuthors && len(authors) > 0 {
		return fail(app, 2, errors.New("--author and --all-authors cannot be combined"))
	}

	now := app.Now()
	q := domain.Query{Branch: *branch, AllBranches: *allBranches}
	switch {
	case *yesterday:
		q.Since, q.Until = utils.DayRange(now.AddDate(0, 0, -1))
		q.Period = "Yesterday"
	case custom:
		var err error
		if *since != "" {
			if q.Since, err = utils.ParseTime(*since, now.Location(), false); err != nil {
				return fail(app, 2, err)
			}
		}
		if *until != "" {
			if q.Until, err = utils.ParseTime(*until, now.Location(), true); err != nil {
				return fail(app, 2, err)
			}
		}
		if !q.Since.IsZero() && !q.Until.IsZero() && q.Since.After(q.Until) {
			return fail(app, 2, errors.New("--since must not be after --until"))
		}
		q.Period = "Custom"
	default:
		q.Since, q.Until = utils.DayRange(now)
		q.Period = "Today"
	}

	git := repositories.Git{Dir: *dir}
	root, err := git.Root()
	if err != nil {
		return fail(app, 1, err)
	}
	switch {
	case *allAuthors:
	case len(authors) > 0:
		q.Authors = authors
	default:
		name := git.UserName()
		if name == "" {
			return fail(app, 1, errors.New("git config user.name is not configured\n\nRun:\n  git config --global user.name \"Your Name\"\n\nor pass --author NAME / --all-authors"))
		}
		q.Authors = []string{name}
	}

	st, err := compute(git, root, q)
	if err != nil {
		return fail(app, 1, err)
	}
	if err := Render(app.Out, st, *format); err != nil {
		return fail(app, 1, err)
	}
	return 0
}

func runInteractive(app App) int {
	git := repositories.Git{Dir: app.Dir}
	root, err := git.Root()
	if err != nil {
		return fail(app, 1, err)
	}
	name := git.UserName()
	if name == "" {
		return fail(app, 1, errors.New("git config user.name is not configured\n\nRun:\n  git config --global user.name \"Your Name\""))
	}
	err = tui.Run(tui.Options{
		In: app.In, Out: app.Out, Now: app.Now,
		Repository: filepath.Base(root), Branch: git.Branch(), Author: name,
		Calculate: func(q domain.Query) (domain.Stats, error) { return compute(git, root, q) },
		Render:    func(w io.Writer, s domain.Stats) error { return Render(w, s, "text") },
	})
	if err != nil {
		return fail(app, 1, err)
	}
	return 0
}

func compute(git repositories.Git, root string, q domain.Query) (domain.Stats, error) {
	st, err := services.Calculate(git, q)
	if err != nil {
		return st, err
	}
	st.Repository = filepath.Base(root)
	switch {
	case q.AllBranches:
		st.Branch = ""
	case q.Branch == "":
		st.Branch = git.Branch()
	}
	return st, nil
}

func fail(app App, code int, err error) int {
	fmt.Fprintf(app.Err, "Error: %v\n", err)
	return code
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

func isTerminal(v any) bool {
	f, ok := v.(*os.File)
	return ok && term.IsTerminal(f.Fd())
}
