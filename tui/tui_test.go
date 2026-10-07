package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sun01822/gitshiny/domain"
)

// harness drives the model without a terminal: each key goes through Update
// and the commands it returns are run and fed back.
type harness struct {
	m     tea.Model
	seen  []domain.Query
	stats domain.Stats // returned by Calculate, with the query's period and range
	fail  error
	quit  bool
}

func newHarness() *harness {
	minLoading, farewellFrame = 0, 0
	h := &harness{stats: domain.Stats{Repository: "repo", Branch: "main",
		Added: 1842, Removed: 391, Commits: 12, FilesChanged: 37}}
	h.m = newModel(Options{
		Repository: "repo", Branch: "main", Author: "Alice",
		Now: func() time.Time { return time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC) },
		Calculate: func(q domain.Query) (domain.Stats, error) {
			h.seen = append(h.seen, q)
			st := h.stats
			st.Authors, st.Period, st.Since, st.Until = q.Authors, q.Period, q.Since, q.Until
			return st, h.fail
		},
	})
	return h
}

// run executes cmd and feeds its messages back. Commands that sleep (spinner
// ticks, cursor blink) do not return in time and are dropped.
func (h *harness) run(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	ch := make(chan tea.Msg, 1)
	go func() { ch <- cmd() }()
	select {
	case msg := <-ch:
		switch msg := msg.(type) {
		case tea.BatchMsg:
			for _, c := range msg {
				h.run(c)
			}
		case tea.QuitMsg:
			h.quit = true
		default:
			var next tea.Cmd
			h.m, next = h.m.Update(msg)
			h.run(next)
		}
	case <-time.After(25 * time.Millisecond):
	}
}

var named = map[string]tea.KeyType{
	"enter": tea.KeyEnter, "esc": tea.KeyEsc, "up": tea.KeyUp, "down": tea.KeyDown,
	"tab": tea.KeyTab, "backspace": tea.KeyBackspace, "ctrl+c": tea.KeyCtrlC,
}

// press sends keys; anything that is not a named key is typed as text.
func (h *harness) press(keys ...string) *harness {
	for _, k := range keys {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		if t, ok := named[k]; ok {
			msg = tea.KeyMsg{Type: t}
		}
		var cmd tea.Cmd
		h.m, cmd = h.m.Update(msg)
		h.run(cmd)
	}
	return h
}

func TestPeriods(t *testing.T) {
	h := newHarness().press("enter")
	if len(h.seen) != 1 || h.seen[0].Period != "Today" || h.seen[0].Since.Day() != 2 {
		t.Fatalf("today = %+v", h.seen)
	}
	for _, want := range []string{"repo · main", "Alice", "Today", "2026-10-02 00:00 → 2026-10-02 23:59",
		"+1,842", "-391", "+1,451", "12", "37", "updated 12:00:00"} {
		if !strings.Contains(h.m.View(), want) {
			t.Errorf("stats view missing %q:\n%s", want, h.m.View())
		}
	}

	h = newHarness().press("down", "enter")
	if h.seen[0].Period != "Yesterday" || h.seen[0].Since.Day() != 1 {
		t.Errorf("yesterday = %+v", h.seen[0])
	}
	if h = newHarness().press("up", "up", "enter"); h.seen[0].Period != "This month" {
		t.Errorf("up should wrap around: %+v", h.seen[0])
	}
	if h = newHarness().press("2"); len(h.seen) != 1 || h.seen[0].Period != "Yesterday" {
		t.Errorf("shortcut 2 = %+v", h.seen)
	}

	h = newHarness().press("5", "2026-10-01 09:00:00", "enter", "2026-10-01 18:00:00", "enter")
	if len(h.seen) != 1 || h.seen[0].Period != "Custom" || h.seen[0].Since.Hour() != 9 || h.seen[0].Until.Hour() != 18 {
		t.Errorf("custom = %+v", h.seen)
	}
}

func TestWeekMonthAndAuthorToggle(t *testing.T) {
	// "now" is Friday 2026-10-02.
	if h := newHarness().press("3"); h.seen[0].Period != "This week" || h.seen[0].Since.Day() != 28 || h.seen[0].Until.Day() != 2 {
		t.Errorf("week = %+v", h.seen[0])
	}
	if h := newHarness().press("4"); h.seen[0].Period != "This month" || h.seen[0].Since.Day() != 1 {
		t.Errorf("month = %+v", h.seen[0])
	}

	h := newHarness().press("a")
	if !strings.Contains(h.m.View(), "All authors") || len(h.seen) != 0 {
		t.Fatalf("a on the menu should only switch the author:\n%s", h.m.View())
	}
	if h.press("enter"); len(h.seen[0].Authors) != 0 || !strings.Contains(h.m.View(), "Today · All authors") {
		t.Errorf("all authors: %+v\n%s", h.seen[0], h.m.View())
	}
	// On the stats screen it switches back and recalculates.
	if h.press("a"); len(h.seen) != 2 || len(h.seen[1].Authors) != 1 || !strings.Contains(h.m.View(), "Today · Alice") {
		t.Errorf("toggle back: %+v\n%s", h.seen, h.m.View())
	}
}

func TestCustomValidation(t *testing.T) {
	h := newHarness().press("j", "j", "j", "j", "enter", "2026-10-02", "tab", "2026-10-01", "enter")
	if len(h.seen) != 0 || !strings.Contains(h.m.View(), "start must not be after end") {
		t.Fatalf("reversed range: %v\n%s", h.seen, h.m.View())
	}
	// Fix the end date in place: 2026-10-01 -> 2026-10-02.
	h.press("backspace", "2")
	if strings.Contains(h.m.View(), "start must not be after end") {
		t.Error("error should clear once the field is edited")
	}
	if h.press("enter"); len(h.seen) != 1 || h.seen[0].Until.Day() != 2 || h.seen[0].Until.Hour() != 23 {
		t.Errorf("after fix = %+v", h.seen)
	}

	h = newHarness().press("5", "nope", "enter", "enter")
	if len(h.seen) != 0 || !strings.Contains(h.m.View(), "invalid time") {
		t.Errorf("bad timestamp: %v\n%s", h.seen, h.m.View())
	}
}

func TestErrorRetryAndNavigation(t *testing.T) {
	h := newHarness()
	h.fail = errors.New("boom")
	if h.press("enter"); !strings.Contains(h.m.View(), "boom") {
		t.Fatalf("error view:\n%s", h.m.View())
	}
	h.fail = nil
	if h.press("r"); len(h.seen) != 2 || !strings.Contains(h.m.View(), "1,842") {
		t.Errorf("retry: %v\n%s", h.seen, h.m.View())
	}
	if h.press("r"); len(h.seen) != 3 {
		t.Errorf("refresh did not recalculate: %v", h.seen)
	}
	if _, cmd := h.m.Update(spinner.TickMsg{}); cmd != nil {
		t.Error("spinner should stop ticking once loading is over")
	}
	if h.press("esc"); !strings.Contains(h.m.View(), "TIME RANGE") || h.quit {
		t.Errorf("esc should return to the menu:\n%s", h.m.View())
	}
}

func TestEmptyPeriod(t *testing.T) {
	h := newHarness()
	h.stats = domain.Stats{}
	if h.press("enter"); !strings.Contains(h.m.View(), "No commits in this period") {
		t.Errorf("empty view:\n%s", h.m.View())
	}
}

func TestLayout(t *testing.T) {
	h := newHarness()
	bare := h.m.View()
	if n := strings.Count(bare, "\n"); n > 20 {
		t.Errorf("unsized view should be the bare box, got %d lines", n)
	}
	h.m, _ = h.m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	if got := strings.Count(h.m.View(), "\n") + 1; got != 40 {
		t.Errorf("centred view height = %d, want 40", got)
	}
	h.m, _ = h.m.Update(tea.WindowSizeMsg{Width: 20, Height: 5})
	if h.m.View() != bare {
		t.Error("a terminal smaller than the box should get the bare box")
	}
}

func TestLoading(t *testing.T) {
	h := newHarness()
	h.m, _ = h.m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // command not run: stay on loading
	first := h.m.View()
	for _, want := range []string{"Collecting commits", "Today · Fri, Oct 2 · Alice", "━━━━"} {
		if !strings.Contains(first, want) {
			t.Errorf("loading view missing %q:\n%s", want, first)
		}
	}
	for i := 0; i < 5; i++ {
		h.m, _ = h.m.Update(h.m.(model).spin.Tick())
	}
	if h.m.View() == first {
		t.Error("loading bar should move between ticks")
	}

	// Fast results are held back so the loading screen is visible.
	minLoading = 60 * time.Millisecond
	defer func() { minLoading = 0 }()
	_, cmd := newModel(h.m.(model).o).calculate()
	start := time.Now()
	for _, c := range cmd().(tea.BatchMsg) {
		if _, ok := c().(statsMsg); ok && time.Since(start) < minLoading {
			t.Errorf("result after %v, want at least %v", time.Since(start), minLoading)
		}
	}
}

func TestRatioBar(t *testing.T) {
	if ratioBar(0, 0) != "" {
		t.Error("no changes should have no bar")
	}
	for _, c := range []struct{ added, removed, green int }{
		{10, 0, barWidth}, {0, 10, 0}, {1, 100000, 1}, {100000, 1, barWidth - 1},
	} {
		bar := ratioBar(c.added, c.removed)
		if g := strings.Count(bar, "█"); g != c.green || g+strings.Count(bar, "▒") != barWidth {
			t.Errorf("ratioBar(%d, %d) = %q", c.added, c.removed, bar)
		}
	}
}

func TestQuit(t *testing.T) {
	// q and esc play the farewell to its end, then quit.
	for _, k := range []string{"q", "esc"} {
		h := newHarness().press(k)
		if !h.quit || len(h.seen) != 0 || h.m.(model).printStats {
			t.Errorf("%s: quit=%v seen=%v", k, h.quit, h.seen)
		}
		for _, want := range []string{thanks, "Happy committing, Alice"} {
			if !strings.Contains(h.m.View(), want) {
				t.Errorf("%s: farewell missing %q:\n%s", k, want, h.m.View())
			}
		}
	}
	// ctrl+c skips the farewell.
	if h := newHarness().press("ctrl+c"); !h.quit || h.m.(model).screen != menuScreen {
		t.Errorf("ctrl+c: quit=%v screen=%v", h.quit, h.m.(model).screen)
	}
	// The farewell animates, and any key cuts it short.
	m, _ := newHarness().m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	first := m.View()
	if m.(model).screen != farewellScreen || strings.Contains(first, thanks) {
		t.Errorf("farewell should start with the thanks still untyped:\n%s", first)
	}
	if m, _ = m.Update(farewellMsg{}); m.View() == first {
		t.Error("farewell should change between frames")
	}
	if _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter}); cmd == nil {
		t.Error("a key during the farewell should quit")
	} else if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("a key during the farewell should quit")
	}
	// Quitting from the stats screen still leaves them for the scrollback.
	if h := newHarness().press("enter", "q"); !h.quit || !h.m.(model).printStats {
		t.Errorf("quit from stats: quit=%v printStats=%v", h.quit, h.m.(model).printStats)
	}
	// q inside a text field is text, not quit.
	if h := newHarness().press("5", "q"); h.quit || !strings.Contains(h.m.View(), "q") {
		t.Error("q in custom field should be typed, not quit")
	}
}
