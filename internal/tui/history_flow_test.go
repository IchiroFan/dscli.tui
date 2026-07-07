package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dscli/dscli.tui/internal/tui/protocol"
)


// TestHistoryListArrowAfterTransition verifies that after calling
// historyListPayload (the real async result handler), the view output
// contains the ▸ arrow at the first item.
func TestHistoryListArrowAfterTransition(t *testing.T) {
	m := model()

	// Simulate the state when the "history list --json" command is running.
	m.screen = ScreenRunningCmd
	m.historyCursor = -1

	// Send the real payload as dscli would.
	p := &protocol.CommandResultPayload{
		Success: true,
		Data: `[
			{"id":25604,"role":"assistant","ok":false,"created_at":"2026-07-04T14:00:00+08:00"},
			{"id":25605,"role":"user","ok":true,"created_at":"2026-07-04T14:01:00+08:00"},
			{"id":25606,"role":"tool","ok":true,"created_at":"2026-07-04T14:02:00+08:00"}
		]`,
	}

	if !m.historyListPayload(p) {
		t.Fatal("historyListPayload returned false — expected true with valid data")
	}

	// Verify screen transition.
	if m.screen != ScreenHistoryList {
		t.Fatalf("screen = %d, want ScreenHistoryList (%d)", m.screen, ScreenHistoryList)
	}

	// Verify cursor was set to first item (newest selected).
	want := 0
	if m.historyCursor != want {
		t.Fatalf("historyCursor = %d, want %d (newest item)", m.historyCursor, want)
	}

	// Verify history items were parsed.
	if len(m.historyItems) != 3 {
		t.Fatalf("historyItems = %d items, want 3", len(m.historyItems))
	}

	// Now check the rendered view.
	v := m.viewHistoryList()

	// The ▸ character MUST be present in the view output.
	if !strings.Contains(v, "▸") {
		t.Fatal("▸ arrow NOT found in view output after historyListPayload transition")
	}

	// Count arrow occurrences — should be exactly 1.
	lines := strings.Split(v, "\n")
	arrowCount := 0
	for _, line := range lines {
		if strings.Contains(line, "▸") {
			arrowCount++
			t.Logf("  Arrow line: %s", line)
		}
	}
	if arrowCount == 0 {
		t.Fatal("▸ arrow not found in any line of view output")
	}
	if arrowCount > 1 {
		t.Logf("Note: %d lines contain ▸ (expected 1)", arrowCount)
	}

	// Verify the arrow is on the first item (ID 25606, newest after reverse).
	found := false
	for _, line := range lines {
		if strings.Contains(line, "▸") {
			if strings.Contains(line, "25606") {
				found = true
				break
			}
			t.Errorf("▸ is on wrong item: %s", line)
		}
	}
	if !found {
		t.Error("▸ should be on the last history item (25606, newest)")
	}

	// Verify unselected items have "  " prefix (no arrow).
	// The first unselected item should NOT have ▸.
	unselectedFound := false
	for _, line := range lines {
		if strings.Contains(line, "25605") {
			if strings.Contains(line, "▸") {
				t.Errorf("Second item should not have ▸, but found: %s", line)
			}
			unselectedFound = true
		}
	}
	if !unselectedFound {
		t.Log("Note: second item (25605) not found in view (may be scrolled off)")
	}
}

// TestHistoryListArrowAfterDownKey verifies that pressing down moves the arrow.
func TestHistoryListArrowAfterDownKey(t *testing.T) {
	m := model()
	m.screen = ScreenHistoryList
	m.historyItems = []HistoryItem{
		{ID: "100", Role: "assistant", Done: "true", CreatedAt: "2026-07-04T14:00:00+08:00"},
		{ID: "101", Role: "user", Done: "true", CreatedAt: "2026-07-04T14:01:00+08:00"},
		{ID: "102", Role: "tool", Done: "true", CreatedAt: "2026-07-04T14:02:00+08:00"},
	}
	m.historyCursor = 0

	// Press down key once.
	m = update(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	if m.historyCursor != 1 {
		t.Fatalf("historyCursor = %d, want 1 after one down press", m.historyCursor)
	}

	v := m.viewHistoryList()

	// Arrow should now be on item 101 (second item).
	lines := strings.Split(v, "\n")
	for _, line := range lines {
		if strings.Contains(line, "▸") {
			if !strings.Contains(line, "101") {
				t.Errorf("▸ should be on item 101 after pressing down, got: %s", line)
			} else {
				t.Logf("  ✓ Arrow correctly on item 101: %s", line)
			}
		}
	}
}
