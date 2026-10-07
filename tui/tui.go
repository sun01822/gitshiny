// Package tui is the interactive Bubble Tea UI. It only talks to the
// statistics engine through the Options callbacks.
package tui

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sun01822/gitshiny/domain"
	"github.com/sun01822/gitshiny/utils"
)

// Options wires the UI to the rest of the application.
type Options struct {
	In         io.Reader
	Out        io.Writer
	Repository string
	Branch     string
	Author     string
	Now        func() time.Time
	Calculate  func(domain.Query) (domain.Stats, error)
	Render     func(io.Writer, domain.Stats) error
}

// Run starts the UI. Statistics still on screen at exit are printed with
// Render so they survive in the scrollback once the alt screen closes, and so
// does a one-line thanks unless the user bailed out with ctrl+c.
func Run(o Options) error {
	final, err := tea.NewProgram(newModel(o),
		tea.WithInput(o.In), tea.WithOutput(o.Out), tea.WithAltScreen()).Run()
	if err != nil {
		return err
	}
	m, ok := final.(model)
	if !ok {
		return nil
	}
	if m.screen == statsScreen || m.printStats {
		if err := o.Render(o.Out, m.stats); err != nil {
			return err
		}
	}
	if m.screen == farewellScreen {
		fmt.Fprintln(o.Out, accentText.Render("✦ "+thanks))
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
	farewellScreen
)

var periods = []string{"Today", "Yesterday", "This week", "This month", "This year", "Custom"}

const custom = 5 // index into periods

type (
	statsMsg    domain.Stats
	errMsg      struct{ err error }
	farewellMsg struct{}
)

type model struct {
	o       Options
	screen  screen
	cursor  int                // selected period
	inputs  [2]textinput.Model // custom start, end
	focus   int                // focused custom input
	all     bool               // every author instead of o.Author
	spin    spinner.Model
	frame   int          // loading or farewell animation frame
	q       domain.Query // query being collected
	err     error
	stats   domain.Stats
	updated time.Time
	w, h    int // terminal size, 0 until the first WindowSizeMsg

	printStats bool // quit from the stats screen: Run prints them
}

// minLoading keeps the loading screen up long enough to be seen: git usually
// answers in milliseconds, which would otherwise flash past unnoticed.
var minLoading = 700 * time.Millisecond

// The farewell plays farewellFrames frames, just under a second in total.
var farewellFrame = 50 * time.Millisecond

const (
	farewellFrames = 18
	thanks         = "Thanks for using GitShiny"
)

func newModel(o Options) model {
	dots := spinner.Spinner{Frames: spinner.MiniDot.Frames, FPS: time.Second / 20}
	m := model{o: o, spin: spinner.New(spinner.WithSpinner(dots), spinner.WithStyle(accentText))}
	day := o.Now().Format("2006-01-02")
	for i, placeholder := range []string{day + " 00:00", day + " 23:59"} {
		in := textinput.New()
		in.Prompt, in.Placeholder, in.CharLimit, in.Width = "", placeholder, 25, 28
		in.PlaceholderStyle = mutedText
		in.Cursor.Style = accentText
		m.inputs[i] = in
	}
	return m
}

func (m model) Init() tea.Cmd { return nil }

// query builds the query for the selected period. It reads the clock every
// time so refreshing "Today" stays correct across midnight.
func (m model) query() (domain.Query, error) {
	now := m.o.Now()
	q := domain.Query{Period: periods[m.cursor]}
	if !m.all {
		q.Authors = []string{m.o.Author}
	}
	switch m.cursor {
	case 0:
		q.Since, q.Until = utils.DayRange(now)
	case 1:
		q.Since, q.Until = utils.DayRange(now.AddDate(0, 0, -1))
	case 2:
		q.Since, q.Until = utils.WeekRange(now)
	case 3:
		q.Since, q.Until = utils.MonthRange(now)
	case 4:
		q.Since, q.Until = utils.YearRange(now)
	default:
		var err error
		if q.Since, err = utils.ParseTime(m.inputs[0].Value(), now.Location(), false); err != nil {
			return q, err
		}
		if q.Until, err = utils.ParseTime(m.inputs[1].Value(), now.Location(), true); err != nil {
			return q, err
		}
		if q.Since.After(q.Until) {
			return q, errors.New("start must not be after end")
		}
	}
	return q, nil
}

// author names whose commits are counted.
func (m model) author() string {
	if m.all {
		return "All authors"
	}
	return m.o.Author
}

// calculate switches to the loading screen and runs the engine off the UI loop.
func (m model) calculate() (tea.Model, tea.Cmd) {
	q, err := m.query()
	if err != nil {
		m.screen, m.err = customScreen, err
		return m, nil
	}
	m.screen, m.err, m.q, m.frame = loadingScreen, nil, q, 0
	calc := m.o.Calculate
	return m, tea.Batch(m.spin.Tick, func() tea.Msg {
		start := time.Now()
		st, err := calc(q)
		time.Sleep(minLoading - time.Since(start))
		if err != nil {
			return errMsg{err}
		}
		return statsMsg(st)
	})
}

// choose acts on the selected period: open the custom form or calculate.
func (m model) choose() (tea.Model, tea.Cmd) {
	if m.cursor == custom {
		m.screen, m.err = customScreen, nil
		return m.setFocus(0)
	}
	return m.calculate()
}

// farewell replaces an immediate quit with the short thank-you animation.
func (m model) farewell() (tea.Model, tea.Cmd) {
	m.printStats = m.screen == statsScreen
	m.screen, m.frame = farewellScreen, 0
	return m, farewellTick()
}

func farewellTick() tea.Cmd {
	return tea.Tick(farewellFrame, func(time.Time) tea.Msg { return farewellMsg{} })
}

func (m model) setFocus(i int) (tea.Model, tea.Cmd) {
	m.inputs[m.focus].Blur()
	m.focus = i
	return m, m.inputs[i].Focus()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
	case statsMsg:
		m.screen, m.stats, m.updated = statsScreen, domain.Stats(msg), m.o.Now()
	case errMsg:
		m.screen, m.err = errorScreen, msg.err
	case spinner.TickMsg:
		// Only animate while loading, so the tick chain stops afterwards.
		if m.screen == loadingScreen {
			m.frame++
			var cmd tea.Cmd
			m.spin, cmd = m.spin.Update(msg)
			return m, cmd
		}
	case farewellMsg:
		if m.screen != farewellScreen {
			return m, nil
		}
		if m.frame++; m.frame >= farewellFrames {
			return m, tea.Quit
		}
		return m, farewellTick()
	case tea.KeyMsg:
		return m.key(msg)
	default:
		if m.screen == customScreen { // cursor blink
			var cmd tea.Cmd
			m.inputs[m.focus], cmd = m.inputs[m.focus].Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m model) key(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	if k.Type == tea.KeyCtrlC {
		return m, tea.Quit
	}
	switch m.screen {
	case menuScreen:
		switch s := k.String(); s {
		case "q", "esc":
			return m.farewell()
		case "up", "k":
			m.cursor = (m.cursor + len(periods) - 1) % len(periods)
		case "down", "j":
			m.cursor = (m.cursor + 1) % len(periods)
		case "1", "2", "3", "4", "5", "6":
			m.cursor = int(s[0] - '1')
			return m.choose()
		case "a":
			m.all = !m.all
		case "enter":
			return m.choose()
		}
	case customScreen:
		switch k.Type {
		case tea.KeyEsc:
			m.screen, m.err = menuScreen, nil
			return m, nil
		case tea.KeyTab, tea.KeyShiftTab, tea.KeyUp, tea.KeyDown:
			return m.setFocus(1 - m.focus)
		case tea.KeyEnter:
			if m.focus == 0 {
				return m.setFocus(1)
			}
			return m.calculate()
		}
		// Everything else is text editing, including "q".
		var cmd tea.Cmd
		m.inputs[m.focus], cmd = m.inputs[m.focus].Update(k)
		m.err = nil
		return m, cmd
	case statsScreen, errorScreen:
		switch k.String() {
		case "q":
			return m.farewell()
		case "esc":
			m.screen, m.err = menuScreen, nil
		case "r":
			return m.calculate()
		case "a":
			m.all = !m.all
			return m.calculate()
		}
	case farewellScreen: // any key skips the animation
		return m, tea.Quit
	}
	return m, nil
}

const (
	layout   = "2006-01-02 15:04"
	inner    = 48 // content width inside the box
	barWidth = 47 // three cards plus two gaps
)

var (
	accent   = lipgloss.AdaptiveColor{Light: "#6C3BD9", Dark: "#A78BFA"}
	onAccent = lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#1A1033"}
	good     = lipgloss.AdaptiveColor{Light: "#0E8A5F", Dark: "#3DDC97"}
	bad      = lipgloss.AdaptiveColor{Light: "#D1345B", Dark: "#FF6B8B"}
	muted    = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#8B8FA3"}

	rounded    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	box        = rounded.BorderForeground(accent).Padding(1, 2).Width(inner + 4)
	errPanel   = rounded.BorderForeground(bad).Width(inner - 2)
	bold       = lipgloss.NewStyle().Bold(true)
	badge      = bold.Foreground(onAccent).Background(accent).Padding(0, 1)
	selected   = bold.Foreground(onAccent).Background(accent).Width(inner)
	label      = bold.Foreground(muted)
	accentText = bold.Foreground(accent)
	mutedText  = lipgloss.NewStyle().Foreground(muted)
	goodText   = lipgloss.NewStyle().Foreground(good)
	badText    = lipgloss.NewStyle().Foreground(bad)
	column     = lipgloss.NewStyle().Width(inner / 2)
)

func (m model) View() string {
	var body, keys string
	switch m.screen {
	case menuScreen:
		body, keys = m.menuView(), help("↑/↓", "move", "enter/1-6", "select", "a", "authors", "q", "quit")
	case customScreen:
		body, keys = m.customView(), help("tab", "switch", "enter", "next/calculate", "esc", "back")
	case loadingScreen:
		body, keys = m.loadingView(), help("ctrl+c", "quit")
	case statsScreen:
		body, keys = m.statsView(), help("r", "refresh", "a", "authors", "esc", "menu", "q", "quit")
	case errorScreen:
		body = errPanel.Render(badText.Bold(true).Render("✗ Something went wrong") + "\n" + m.err.Error())
		keys = help("r", "retry", "a", "authors", "esc", "menu", "q", "quit")
	case farewellScreen:
		body, keys = m.farewellView(), help("any key", "exit now")
	}
	out := box.Render(m.header() + "\n\n" + body + "\n\n" + keys)
	if m.w >= lipgloss.Width(out) && m.h >= lipgloss.Height(out) {
		return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, out)
	}
	return out + "\n"
}

// header is the title badge with "repository · branch" right-aligned.
func (m model) header() string {
	left := badge.Render("✦ GitShiny")
	where := m.o.Repository
	if m.o.Branch != "" {
		where = strings.TrimPrefix(where+" · "+m.o.Branch, " · ")
	}
	right := mutedText.MaxWidth(inner - lipgloss.Width(left) - 2).Render(where)
	gap := inner - lipgloss.Width(left) - lipgloss.Width(right)
	return left + strings.Repeat(" ", max(gap, 1)) + right
}

// help renders key/description pairs for the footer.
func help(pairs ...string) string {
	parts := make([]string, 0, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		parts = append(parts, accentText.Render(pairs[i])+" "+mutedText.Render(pairs[i+1]))
	}
	return strings.Join(parts, mutedText.Render(" · "))
}

func (m model) menuView() string {
	now := m.o.Now()
	week, _ := utils.WeekRange(now)
	hints := []string{now.Format("Mon, Jan 2"), now.AddDate(0, 0, -1).Format("Mon, Jan 2"),
		"since " + week.Format("Mon, Jan 2"), "since " + now.Format("Jan") + " 1", "since Jan 1", "pick any start and end"}
	var b strings.Builder
	b.WriteString(label.Render("TIME RANGE") + "\n")
	for i, p := range periods {
		if i == m.cursor {
			b.WriteString(selected.Render(fmt.Sprintf("▸ %d  %-12s %s", i+1, p, hints[i])) + "\n")
		} else {
			fmt.Fprintf(&b, "  %s  %-12s %s\n", mutedText.Render(fmt.Sprint(i+1)), p, mutedText.Render(hints[i]))
		}
	}
	b.WriteString("\n" + label.Render("AUTHOR") + "\n" + m.author())
	return b.String()
}

func (m model) customView() string {
	var b strings.Builder
	b.WriteString(label.Render("CUSTOM RANGE") + "\n")
	for i, name := range []string{"Start", "End"} {
		if i == m.focus {
			b.WriteString(accentText.Render(fmt.Sprintf("▸ %-6s", name)))
		} else {
			b.WriteString(mutedText.Render(fmt.Sprintf("  %-6s", name)))
		}
		b.WriteString(m.inputs[i].View() + "\n")
	}
	b.WriteString("\n" + mutedText.Render("YYYY-MM-DD, optionally with HH:MM or HH:MM:SS"))
	if m.err != nil {
		b.WriteString("\n\n" + badText.Render("✗ "+m.err.Error()))
	}
	return b.String()
}

// loadingView says what is being collected above a sliding bar. The bar is
// indeterminate: git log is a single call with no progress to report.
func (m model) loadingView() string {
	const block = 8
	what := m.q.Period + " · " + m.q.Since.Format("Mon, Jan 2") + " · " + m.author()
	if m.cursor == custom {
		what = m.q.Since.Format(layout) + " → " + m.q.Until.Format(layout)
	}
	pos := (m.frame*2+block)%(barWidth+block) - block
	from, to := max(pos, 0), min(pos+block, barWidth)
	bar := mutedText.Render(strings.Repeat("─", from)) + accentText.Render(strings.Repeat("━", to-from)) +
		mutedText.Render(strings.Repeat("─", barWidth-to))
	return m.spin.View() + " Collecting commits…\n" + mutedText.Render(what) + "\n\n" + bar
}

// farewellView types out the thanks between twinkling sparkles while a bar
// fills to show how long is left.
func (m model) farewellView() string {
	sparkles := []string{"✦", "✧", "·"}
	n := min(m.frame*3, len(thanks))
	line := accentText.Render(sparkles[m.frame/2%3]+"  "+thanks[:n]) + strings.Repeat(" ", len(thanks)-n) +
		accentText.Render("  "+sparkles[(m.frame/2+1)%3])
	wish := ""
	if n == len(thanks) {
		wish = mutedText.MaxWidth(inner).Render("Happy committing, " + m.o.Author)
	}
	fill := min((m.frame+1)*barWidth/farewellFrames, barWidth)
	centre := lipgloss.NewStyle().Width(inner).Align(lipgloss.Center)
	return centre.Render(line) + "\n" + centre.Render(wish) + "\n\n" +
		accentText.Render(strings.Repeat("━", fill)) + mutedText.Render(strings.Repeat("─", barWidth-fill))
}

func (m model) statsView() string {
	s := m.stats
	var b strings.Builder
	b.WriteString(bold.Render(s.Period) + mutedText.Render(" · ") + m.author() + "\n")
	b.WriteString(mutedText.Render(s.Since.Format(layout)+" → "+s.Until.Format(layout)) + "\n\n")
	if s.Commits == 0 {
		b.WriteString("No commits in this period.\n" + mutedText.Render("Press esc to pick another range.") + "\n")
	} else {
		net := lipgloss.TerminalColor(muted)
		if n := s.NetGrowth(); n > 0 {
			net = good
		} else if n < 0 {
			net = bad
		}
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top,
			card(utils.Signed(s.Added), "added", good), " ",
			card(utils.Signed(-s.Removed), "removed", bad), " ",
			card(utils.Signed(s.NetGrowth()), "net growth", net)) + "\n")
		if bar := ratioBar(s.Added, s.Removed); bar != "" {
			b.WriteString(bar + "\n")
		}
		b.WriteString("\n" + count("Commits", s.Commits) + count("Files changed", s.FilesChanged) + "\n")
	}
	b.WriteString("\n" + mutedText.Render("updated "+m.updated.Format("15:04:05")))
	return b.String()
}

// card is one bordered metric: a coloured value over a muted name.
func card(value, name string, c lipgloss.TerminalColor) string {
	return rounded.BorderForeground(c).Width(13).
		Render(bold.Foreground(c).Render(value) + "\n" + mutedText.Render(name))
}

func count(name string, n int) string {
	return column.Render(mutedText.Render(name) + "  " + bold.Render(utils.Commas(n)))
}

// ratioBar shows added against removed lines; any non-zero side gets a cell.
func ratioBar(added, removed int) string {
	total := added + removed
	if total == 0 {
		return ""
	}
	g := added * barWidth / total
	if added > 0 && g == 0 {
		g = 1
	}
	if removed > 0 && g == barWidth {
		g = barWidth - 1
	}
	return goodText.Render(strings.Repeat("█", g)) + badText.Render(strings.Repeat("▒", barWidth-g))
}
