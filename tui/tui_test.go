package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sun01822/gitshiny/domain"
)

// harness drives the model without a terminal: each key goes through Update
// and any returned command is run synchronously and fed back.
type harness struct {
	m    tea.Model
	seen []domain.Query
	fail error
	quit bool
}

func newHarness() *harness {
	h := &harness{}
	h.m = model{o: Options{
		Author: "Alice",
		Now:    func() time.Time { return time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC) },
		Calculate: func(q domain.Query) (domain.Stats, error) {
			h.seen = append(h.seen, q)
			return domain.Stats{Repository: "repo", Branch: "main", Authors: q.Authors,
				Period: q.Period, Added: 1842, Removed: 391, Commits: 12, FilesChanged: 37}, h.fail
		},
	}}
	return h
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
		if h.m, cmd = h.m.Update(msg); cmd != nil {
			out := cmd()
			if _, ok := out.(tea.QuitMsg); ok {
				h.quit = true
			}
			h.m, _ = h.m.Update(out)
		}
	}
	return h
}

func TestPeriods(t *testing.T) {
	h := newHarness().press("enter")
	if len(h.seen) != 1 || h.seen[0].Period != "Today" || h.seen[0].Since.Day() != 2 {
		t.Fatalf("today = %+v", h.seen)
	}
	for _, want := range []string{"repo", "main", "Alice", "Today", "1,842", "391", "+1,451", "12", "37"} {
		if !strings.Contains(h.m.View(), want) {
			t.Errorf("stats view missing %q:\n%s", want, h.m.View())
		}
	}

	h = newHarness().press("down", "enter")
	if h.seen[0].Period != "Yesterday" || h.seen[0].Since.Day() != 1 {
		t.Errorf("yesterday = %+v", h.seen[0])
	}

	h = newHarness().press("j", "j", "enter", "2026-10-01 09:00:00", "enter", "2026-10-01 18:00:00", "enter")
	if len(h.seen) != 1 || h.seen[0].Period != "Custom" || h.seen[0].Since.Hour() != 9 || h.seen[0].Until.Hour() != 18 {
		t.Errorf("custom = %+v", h.seen)
	}
}

func TestCustomValidation(t *testing.T) {
	h := newHarness().press("down", "down", "enter", "2026-10-02", "tab", "2026-10-01", "enter")
	if len(h.seen) != 0 || !strings.Contains(h.m.View(), "start must not be after end") {
		t.Fatalf("reversed range: %v\n%s", h.seen, h.m.View())
	}
	// Fix the end date in place: 2026-10-01 -> 2026-10-02.
	h.press("backspace", "2", "enter")
	if len(h.seen) != 1 || h.seen[0].Until.Day() != 2 || h.seen[0].Until.Hour() != 23 {
		t.Errorf("after fix = %+v", h.seen)
	}

	h = newHarness().press("down", "down", "enter", "nope", "enter", "enter")
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
	if h.press("esc"); !strings.Contains(h.m.View(), "Time Range") || h.quit {
		t.Errorf("esc should return to the menu:\n%s", h.m.View())
	}
}

func TestQuit(t *testing.T) {
	for _, k := range []string{"q", "esc", "ctrl+c"} {
		if h := newHarness().press(k); !h.quit || len(h.seen) != 0 {
			t.Errorf("%s: quit=%v seen=%v", k, h.quit, h.seen)
		}
	}
	// q inside a text field is text, not quit.
	if h := newHarness().press("down", "down", "enter", "q"); h.quit {
		t.Error("q in custom field quit the program")
	}
}
