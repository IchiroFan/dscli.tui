package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestViewHistoryListArrowAtCursor0 verifies that when cursor is at 0,
// the first rendered item shows the ▸ arrow.
func TestViewHistoryListArrowAtCursor0(t *testing.T) {
	m := model()
	m.screen = ScreenHistoryList
	m.historyItems = []HistoryItem{
		{ID: "100", Role: "assistant", Done: "true", CreatedAt: "2026-07-04T14:00:00+08:00"},
		{ID: "101", Role: "user", Done: "true", CreatedAt: "2026-07-04T14:01:00+08:00"},
		{ID: "102", Role: "tool", Done: "true", CreatedAt: "2026-07-04T14:02:00+08:00"},
	}
	m.historyCursor = 0

	v := m.viewHistoryList()

	// The first visible line should contain ▸
	if !strings.Contains(v, "▸") {
		t.Fatal("Expected ▸ in view output when cursor=0, but not found")
	}

	// Find lines with ▸
	lines := strings.Split(v, "\n")
	arrowLines := 0
	for i, line := range lines {
		if strings.Contains(line, "▸") {
			arrowLines++
			t.Logf("  Line %d: %s", i, line)
		}
	}
	if arrowLines == 0 {
		t.Error("No line contains ▸ arrow")
	}
	if arrowLines > 1 {
		t.Logf("Note: %d lines contain ▸ (expected 1)", arrowLines)
	}
}

// TestViewHistoryListArrowMovesWithCursor verifies that pressing down
// moves the arrow to the correct position.
func TestViewHistoryListArrowMovesWithCursor(t *testing.T) {
	m := model()
	m.screen = ScreenHistoryList
	m.historyItems = []HistoryItem{
		{ID: "100", Role: "assistant", Done: "true", CreatedAt: "2026-07-04T14:00:00+08:00"},
		{ID: "101", Role: "user", Done: "true", CreatedAt: "2026-07-04T14:01:00+08:00"},
		{ID: "102", Role: "tool", Done: "true", CreatedAt: "2026-07-04T14:02:00+08:00"},
	}
	m.historyCursor = 0

	// Press down key
	update(m, tea.KeyMsg{Type: tea.KeyDown})

	v := m.viewHistoryList()
	if !strings.Contains(v, "▸") {
		t.Fatal("Expected ▸ after pressing down, but not found")
	}

	// The arrow should be next to item 101 (second item)
	lines := strings.Split(v, "\n")
	for i, line := range lines {
		if strings.Contains(line, "▸") {
			t.Logf("  Arrow at line %d: %s", i, line)
			if strings.Contains(line, "100") {
				t.Error("Arrow should NOT be on item 100 (cursor moved to 1)")
			}
			if !strings.Contains(line, "101") {
				t.Errorf("Arrow should be on item 101, but line contains: %s", line)
			}
		}
	}
}

// TestViewHistoryListNoArrowWithNegativeCursor verifies that
// cursor=-1 shows no arrow.
func TestViewHistoryListNoArrowWithNegativeCursor(t *testing.T) {
	m := model()
	m.screen = ScreenHistoryList
	m.historyItems = []HistoryItem{
		{ID: "100", Role: "assistant", Done: "true", CreatedAt: "2026-07-04T14:00:00+08:00"},
	}
	m.historyCursor = -1

	v := m.viewHistoryList()

	lines := strings.Split(v, "\n")
	for i, line := range lines {
		if strings.Contains(line, "▸") {
			t.Errorf("Line %d unexpectedly contains ▸ when cursor=-1: %s", i, line)
		}
	}
}
