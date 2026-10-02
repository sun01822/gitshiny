// Package tui is the interactive terminal menu. It is dependency-free for now
// and can be swapped for a Bubble Tea UI later: it only talks to the
// statistics engine through the Options callbacks.
package tui

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/USERNAME/gitshiny/domain"
	"github.com/USERNAME/gitshiny/utils"
)

// Options wires the menu to the rest of the application.
type Options struct {
	In        io.Reader
	Out       io.Writer
	Author    string
	Now       func() time.Time
	Calculate func(domain.Query) (domain.Stats, error)
	Render    func(io.Writer, domain.Stats) error
}

const rule = "========================================"

// Run shows the menu once, calculates the chosen range and prints the result.
func Run(o Options) error {
	in := bufio.NewReader(o.In)
	fmt.Fprintf(o.Out, "\n%s\n              GitShiny\n%s\n\n", rule, rule)
	fmt.Fprintf(o.Out, "Author: %s\n\nSelect time range:\n\n  1) Today\n  2) Yesterday\n  3) Custom\n  q) Quit\n\n", o.Author)

	choice, err := prompt(in, o.Out, "Enter your choice [1-3]: ")
	if err != nil {
		return err
	}

	now := o.Now()
	q := domain.Query{Authors: []string{o.Author}}
	switch strings.ToLower(choice) {
	case "1":
		q.Since, q.Until = utils.DayRange(now)
		q.Period = "Today"
	case "2":
		q.Since, q.Until = utils.DayRange(now.AddDate(0, 0, -1))
		q.Period = "Yesterday"
	case "3":
		fmt.Fprintln(o.Out)
		s, err := prompt(in, o.Out, "Start [YYYY-MM-DD HH:MM:SS]: ")
		if err != nil {
			return err
		}
		e, err := prompt(in, o.Out, "End   [YYYY-MM-DD HH:MM:SS]: ")
		if err != nil {
			return err
		}
		if q.Since, err = utils.ParseTime(s, now.Location(), false); err != nil {
			return err
		}
		if q.Until, err = utils.ParseTime(e, now.Location(), true); err != nil {
			return err
		}
		if q.Since.After(q.Until) {
			return errors.New("start must not be after end")
		}
		q.Period = "Custom"
	case "q", "quit", "":
		return nil
	default:
		return errors.New("invalid option")
	}

	fmt.Fprintln(o.Out)
	st, err := o.Calculate(q)
	if err != nil {
		return err
	}
	return o.Render(o.Out, st)
}

func prompt(in *bufio.Reader, out io.Writer, label string) (string, error) {
	fmt.Fprint(out, label)
	line, err := in.ReadString('\n')
	if err != nil && !(errors.Is(err, io.EOF) && line != "") {
		if errors.Is(err, io.EOF) {
			return "", nil // closed input behaves like quit
		}
		return "", err
	}
	return strings.TrimSpace(line), nil
}
