// Package tui is the interactive Bubble Tea UI. It only talks to the
// statistics engine through the Options callbacks.
package tui

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sun01822/gitshiny/domain"
	"github.com/sun01822/gitshiny/utils"
)

// Options wires the UI to the rest of the application.
type Options struct {
	In        io.Reader
	Out       io.Writer
	Author    string
	Now       func() time.Time
	Calculate func(domain.Query) (domain.Stats, error)
	Render    func(io.Writer, domain.Stats) error
}

// Run starts the UI. Statistics still on screen at exit are printed with
// Render so they survive in the scrollback once the alt screen closes.
func Run(o Options) error {
	final, err := tea.NewProgram(model{o: o},
		tea.WithInput(o.In), tea.WithOutput(o.Out), tea.WithAltScreen()).Run()
	if err != nil {
		return err
	}
	if m, ok := final.(model); ok && m.screen == statsScreen {
		return o.Render(o.Out, m.stats)
	}
	return nil
}

type screen int

const (
	menuScreen screen = iota
	customScreen
	loadingScreen
	statsScreen
	errorScreen
)

var periods = []string{"Today", "Yesterday", "Custom"}

const custom = 2 // index into periods

type (
	statsMsg domain.Stats
	errMsg   struct{ err error }
)

type model struct {
	o      Options
	screen screen
	cursor int       // selected period
	fields [2]string // custom start, end
	focus  int       // focused custom field
	err    error
	stats  domain.Stats
}

func (m model) Init() tea.Cmd { return nil }

// query builds the query for the selected period. It reads the clock every
// time so refreshing "Today" stays correct across midnight.
func (m model) query() (domain.Query, error) {
	now := m.o.Now()
	q := domain.Query{Authors: []string{m.o.Author}, Period: periods[m.cursor]}
	switch m.cursor {
	case 0:
		q.Since, q.Until = utils.DayRange(now)
	case 1:
		q.Since, q.Until = utils.DayRange(now.AddDate(0, 0, -1))
	default:
		var err error
		if q.Since, err = utils.ParseTime(m.fields[0], now.Location(), false); err != nil {
			return q, err
		}
		if q.Until, err = utils.ParseTime(m.fields[1], now.Location(), true); err != nil {
			return q, err
		}
		if q.Since.After(q.Until) {
			return q, errors.New("start must not be after end")
		}
	}
	return q, nil
}

// calculate switches to the loading screen and runs the engine off the UI loop.
func (m model) calculate() (tea.Model, tea.Cmd) {
	q, err := m.query()
	if err != nil {
		m.screen, m.err = customScreen, err
		return m, nil
	}
	m.screen, m.err = loadingScreen, nil
	calc := m.o.Calculate
	return m, func() tea.Msg {
		st, err := calc(q)
		if err != nil {
			return errMsg{err}
		}
		return statsMsg(st)
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case statsMsg:
		m.screen, m.stats = statsScreen, domain.Stats(msg)
	case errMsg:
		m.screen, m.err = errorScreen, msg.err
	case tea.KeyMsg:
		return m.key(msg)
	}
	return m, nil
}

func (m model) key(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	if k.Type == tea.KeyCtrlC {
		return m, tea.Quit
	}
	switch m.screen {
	case menuScreen:
		switch k.String() {
		case "q", "esc":
			return m, tea.Quit
		case "up", "k":
			m.cursor = max(m.cursor-1, 0)
		case "down", "j":
			m.cursor = min(m.cursor+1, len(periods)-1)
		case "enter":
			if m.cursor == custom {
				m.screen, m.focus, m.err = customScreen, 0, nil
				return m, nil
			}
			return m.calculate()
		}
	case customScreen:
		switch k.Type {
		case tea.KeyEsc:
			m.screen, m.err = menuScreen, nil
		case tea.KeyTab, tea.KeyShiftTab, tea.KeyUp, tea.KeyDown:
			m.focus = 1 - m.focus
		case tea.KeyEnter:
			if m.focus == 0 {
				m.focus = 1
				return m, nil
			}
			return m.calculate()
		case tea.KeyBackspace:
			if r := []rune(m.fields[m.focus]); len(r) > 0 {
				m.fields[m.focus] = string(r[:len(r)-1])
			}
		case tea.KeyRunes, tea.KeySpace:
			m.fields[m.focus] += string(k.Runes)
		}
	case statsScreen, errorScreen:
		switch k.String() {
		case "q":
			return m, tea.Quit
		case "esc":
			m.screen, m.err = menuScreen, nil
		case "r":
			return m.calculate()
		}
	}
	return m, nil
}

const width = 36 // inner width of the box

var (
	box   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).Width(width + 2)
	title = lipgloss.NewStyle().Bold(true).Width(width).Align(lipgloss.Center)
	bold  = lipgloss.NewStyle().Bold(true)
	dim   = lipgloss.NewStyle().Faint(true)
	green = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	red   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
)

func (m model) View() string {
	var b strings.Builder
	b.WriteString(title.Render("GitShiny") + "\n" + dim.Render(strings.Repeat("─", width)) + "\n\n")
	help := ""
	switch m.screen {
	case menuScreen:
		b.WriteString(bold.Render("Time Range") + "\n")
		for i, p := range periods {
			if i == m.cursor {
				b.WriteString(bold.Render("> "+p) + "\n")
			} else {
				b.WriteString("  " + p + "\n")
			}
		}
		b.WriteString("\n" + bold.Render("Author") + "\n" + m.o.Author + "\n")
		help = "↑/↓ move  enter select  q quit"
	case customScreen:
		b.WriteString(bold.Render("Custom Range") + "\n" + dim.Render("YYYY-MM-DD [HH:MM[:SS]]") + "\n\n")
		for i, label := range []string{"Start", "End  "} {
			cur := "  "
			if i == m.focus {
				cur = "> "
				label += " " + m.fields[i] + "█"
			} else {
				label += " " + m.fields[i]
			}
			b.WriteString(cur + label + "\n")
		}
		if m.err != nil {
			b.WriteString("\n" + red.Render(m.err.Error()) + "\n")
		}
		help = "tab next  enter calculate  esc back"
	case loadingScreen:
		b.WriteString("Calculating…\n")
		help = "ctrl+c quit"
	case statsScreen:
		s := m.stats
		net := green
		if s.NetGrowth() < 0 {
			net = red
		}
		fmt.Fprintf(&b, "%-12s %s\n%-12s %s\n%-12s %s\n%-12s %s\n\n",
			"Repository", s.Repository, "Branch", s.Branch,
			"Author", strings.Join(s.Authors, ", "), "Period", s.Period)
		fmt.Fprintf(&b, "%-14s %12s\n%-14s %12s\n%-14s %s\n%-14s %12s\n%-14s %12s\n",
			"Added", utils.Commas(s.Added), "Removed", utils.Commas(s.Removed),
			"Net Growth", net.Render(fmt.Sprintf("%12s", utils.Signed(s.NetGrowth()))),
			"Commits", utils.Commas(s.Commits), "Files Changed", utils.Commas(s.FilesChanged))
		help = "r refresh  esc menu  q quit"
	case errorScreen:
		b.WriteString(bold.Render("Error") + "\n" + red.Render(m.err.Error()) + "\n")
		help = "r retry  esc menu  q quit"
	}
	b.WriteString("\n" + dim.Render(help))
	return box.Render(b.String()) + "\n"
}
