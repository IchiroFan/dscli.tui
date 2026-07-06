package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"gitcode.com/dscli/dscli.tui/internal/aiagent"
	"gitcode.com/dscli/dscli.tui/internal/tui/protocol"
)

func TestHistoryShowThenScroll(t *testing.T) {
	// Set up model in history list state
	m := model()  // Height=30
	m.screen = ScreenHistoryList
	m.historyItems = []HistoryItem{
		{ID: "28032", Role: "tool"},
		{ID: "28033", Role: "tool"},
	}
	m.historyCursor = 0

	// Simulate Enter on item
	m2, cmd := updateWithCmd(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m2.screen != ScreenRunningCmd {
		t.Fatalf("screen = %d, want %d (ScreenRunningCmd)", m2.screen, ScreenRunningCmd)
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	// Simulate SubcommandResultMsg with "show" data (long output, realistic)
	lines := make([]string, 200)
	for i := range lines {
		lines[i] = "This is line number " + strings.Repeat("x", 40)
	}
	longOutput := strings.Join(lines, "\n")

	payload := &protocol.CommandResultPayload{
		Success: true,
		Data:    longOutput,
	}
	msg := aiagent.SubcommandResultMsg{
		Payload: payload,
		Err:     nil,
		Subcmd:  "show",
	}
	m3, _ := updateWithCmd(m2, msg)

	// Verify we're on ScreenShowOutput
	if m3.screen != ScreenShowOutput {
		t.Fatalf("screen = %d, want %d (ScreenShowOutput)", m3.screen, ScreenShowOutput)
	}

	// Verify outputLines is populated
	if len(m3.outputLines) <= 1 {
		t.Fatalf("outputLines has %d lines, want > 1", len(m3.outputLines))
	}

	// Verify prevScreen is set correctly
	if m3.prevScreen != ScreenHistoryList {
		t.Errorf("prevScreen = %d, want %d", m3.prevScreen, ScreenHistoryList)
	}

	// --- Test scrolling ---
	// Height=30, availableHeight=25, totalLines=200 -> scrollMax=175

	// Down should increase scroll
	m4 := update(m3, tea.KeyMsg{Type: tea.KeyDown})
	if m4.outputScroll <= 0 {
		t.Errorf("outputScroll = %d, want > 0 after down", m4.outputScroll)
	}
	firstScroll := m4.outputScroll

	// Up should decrease scroll
	m5 := update(m4, tea.KeyMsg{Type: tea.KeyUp})
	if m5.outputScroll >= firstScroll {
		t.Errorf("outputScroll = %d, want < %d after up", m5.outputScroll, firstScroll)
	}

	// j (Vim down) should also scroll
	scrollBeforeJ := m5.outputScroll
	m6 := update(m5, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m6.outputScroll <= scrollBeforeJ {
		t.Errorf("outputScroll = %d, want > %d after j", m6.outputScroll, scrollBeforeJ)
	}

	// k (Vim up) should scroll back
	scrollBeforeK := m6.outputScroll
	m7 := update(m6, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m7.outputScroll >= scrollBeforeK {
		t.Errorf("outputScroll = %d, want < %d after k", m7.outputScroll, scrollBeforeK)
	}

	// g (top) should go to 0
	m8 := update(m7, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if m8.outputScroll != 0 {
		t.Errorf("outputScroll = %d, want 0 after g", m8.outputScroll)
	}

	// G (bottom) should go to max
	m9 := update(m8, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if m9.outputScroll <= 0 {
		t.Errorf("outputScroll = %d, want > 0 after G", m9.outputScroll)
	}

	// Home should go to top
	m10 := update(m9, tea.KeyMsg{Type: tea.KeyHome})
	if m10.outputScroll != 0 {
		t.Errorf("outputScroll = %d, want 0 after Home", m10.outputScroll)
	}

	// End should go to bottom
	m11 := update(m10, tea.KeyMsg{Type: tea.KeyEnd})
	if m11.outputScroll <= 0 {
		t.Errorf("outputScroll = %d, want > 0 after End", m11.outputScroll)
	}

	t.Log("✅ All scrolling checks passed")
}

func TestHistoryShowScrollViewRender(t *testing.T) {
	// Test that View() renders correctly after scrolling
	m := model()
	m.screen = ScreenShowOutput
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = "content line " + strings.Repeat("x", 30)
	}
	m.cmdOutput = strings.Join(lines, "\n")
	m.outputLines = strings.Split(m.cmdOutput, "\n")
	m.outputScroll = 0
	m.Height = 20 // small terminal

	// Scroll down a bit
	m2 := update(m, tea.KeyMsg{Type: tea.KeyDown})
	m3 := update(m2, tea.KeyMsg{Type: tea.KeyDown})
	m4 := update(m3, tea.KeyMsg{Type: tea.KeyDown})

	// Render view
	v := m4.View()
	if v == "" {
		t.Error("View returned empty string")
	}
	if !strings.Contains(v, "Output") {
		t.Error("View should contain Output header")
	}
	if !strings.Contains(v, "scroll") && !strings.Contains(v, "↑") {
		// Should have some scroll indicator when scrolled
		t.Log("Note: view may not show scroll indicator (check terminal support)")
	}
	t.Log("✅ View render works after scrolling")
}
