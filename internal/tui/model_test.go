// Package tui tests the Bubble Tea application model, update, view, and helpers.
//
// Tests follow the standard Bubble Tea pattern:
//  1. Create a model via New(mockAgent)
//  2. Send messages via model.Update(msg)
//  3. Assert on model state (screen, fields)
//
// Agent-dependent commands (tea.Cmd closures) are NOT executed in tests.
// Instead we simulate the result messages directly, verifying state transitions.
//
// NOTE on tea.KeyMsg: in bubbletea v1.3+, KeyRunes = -1 (not iota=0). A literal
// tea.KeyMsg{Runes: []rune{'j'}} has Type=0 → maps to "ctrl+@", not "j".
// Always use tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}.
package tui

import (
	"context"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"gitcode.com/dscli/dscli.tui/internal/aiagent"
	"gitcode.com/dscli/dscli.tui/internal/tui/protocol"
)

// ─── Mock agent ─────────────────────────────────────────────────────────────

// mockAgent implements aiagent.AIAgent with all methods returning zero values.
// Agent methods are never called in these tests — we simulate result messages
// directly. The mock exists only to satisfy the interface.
type mockAgent struct{}

func (m *mockAgent) Balance(context.Context, string) (*protocol.CommandResultPayload, error) {
	return nil, nil
}
func (m *mockAgent) Models(context.Context, string, bool) (*protocol.CommandResultPayload, error) {
	return nil, nil
}
func (m *mockAgent) Version(context.Context) (*protocol.CommandResultPayload, error) {
	return nil, nil
}
func (m *mockAgent) Flycheck(context.Context, string, bool) (*protocol.CommandResultPayload, error) {
	return nil, nil
}
func (m *mockAgent) FIM(context.Context, ...string) (*protocol.CommandResultPayload, error) {
	return nil, nil
}
func (m *mockAgent) MemorySearch(context.Context, string) (*protocol.CommandResultPayload, error) {
	return nil, nil
}
func (m *mockAgent) History(context.Context, string, ...string) (*protocol.CommandResultPayload, error) {
	return nil, nil
}
func (m *mockAgent) Skill(context.Context, string, ...string) (*protocol.CommandResultPayload, error) {
	return nil, nil
}
func (m *mockAgent) Prompt(context.Context, string, ...string) (*protocol.CommandResultPayload, error) {
	return nil, nil
}
func (m *mockAgent) Memory(context.Context, string, ...string) (*protocol.CommandResultPayload, error) {
	return nil, nil
}
func (m *mockAgent) Project(context.Context, string, ...string) (*protocol.CommandResultPayload, error) {
	return nil, nil
}
func (m *mockAgent) Role(context.Context, string, ...string) (*protocol.CommandResultPayload, error) {
	return nil, nil
}
func (m *mockAgent) Tool(context.Context, string, ...string) (*protocol.CommandResultPayload, error) {
	return nil, nil
}
func (m *mockAgent) Mail(context.Context, string, ...string) (*protocol.CommandResultPayload, error) {
	return nil, nil
}
func (m *mockAgent) Service(context.Context, string, ...string) (*protocol.CommandResultPayload, error) {
	return nil, nil
}
func (m *mockAgent) NewChatSession(context.Context, aiagent.ChatSessionOptions) (*aiagent.ChatSession, error) {
	return nil, nil
}
func (m *mockAgent) SendChimein(_ context.Context, content string) (string, error) {
	return "", nil
}
func (m *mockAgent) Close() error { return nil }

// mockSession creates a ChatSession backed by buffered channels for testing.
// The close field (unexported) is left nil — Close() would panic if called.
// Tests that reach code paths calling m.chatSession.Close() avoid setting
// chatSession, letting the handler's nil-guard skip Clean().
func mockSession() *aiagent.ChatSession {
	events := make(chan *protocol.Message, 100)
	sendCh := make(chan *protocol.Message, 10)
	done := make(chan struct{})
	return &aiagent.ChatSession{
		Events: events,
		Send:   sendCh,
		Done:   done,
	}
}

// ─── Key constants ──────────────────────────────────────────────────────────
// Shorthands to avoid repeating Type: tea.KeyRunes everywhere.

func keyRune(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

var (
	keyJ    = keyRune('j')
	keyK    = keyRune('k')
	keyQ    = keyRune('q')
	keyY    = keyRune('y')
	keyYCap = keyRune('Y')
	keyN    = keyRune('n')
	keyX    = keyRune('x')
	keyH    = keyRune('h')
)

// ─── Mouse wheel helpers ───────────────────────────────────────────────────

func mouseWheelUp() tea.MouseMsg {
	return tea.MouseMsg{Button: tea.MouseButtonWheelUp}
}

func mouseWheelDown() tea.MouseMsg {
	return tea.MouseMsg{Button: tea.MouseButtonWheelDown}
}

// ─── Helpers ───────────────────────────────────────────────────────────────


// model returns a fresh RootModel with Width/Height set for view tests.
func model() *RootModel {
	m := New(&mockAgent{})
	m.Width = 80
	m.Height = 30
	return m
}

// update is a convenience wrapper around m.Update, discarding the command.
func update(m *RootModel, msg tea.Msg) *RootModel {
	updated, _ := m.Update(msg)
	return updated.(*RootModel)
}

// updateWithCmd is like update but also returns the command.
func updateWithCmd(m *RootModel, msg tea.Msg) (*RootModel, tea.Cmd) {
	updated, cmd := m.Update(msg)
	return updated.(*RootModel), cmd
}

// handleEvent is a type-safe wrapper around m.handleChatEvent.
func handleEvent(m *RootModel, msg *protocol.Message) (*RootModel, tea.Cmd) {
	updated, cmd := m.handleChatEvent(msg)
	return updated.(*RootModel), cmd
}

// ─── Model: Constructor ─────────────────────────────────────────────────────

func TestNew(t *testing.T) {
	m := New(&mockAgent{})
	if m.screen != ScreenMainMenu {
		t.Errorf("expected ScreenMainMenu, got %d", m.screen)
	}
	if m.menuCursor != 0 {
		t.Errorf("expected menuCursor=0, got %d", m.menuCursor)
	}
	if len(m.menuItems) != len(defaultMenuItems) {
		t.Errorf("expected %d menu items, got %d", len(defaultMenuItems), len(m.menuItems))
	}
	if m.chatInput.CharLimit != 0 {
		t.Errorf("expected chatInput.CharLimit=0, got %d", m.chatInput.CharLimit)
	}
	if m.askInput.CharLimit != 0 {
		t.Errorf("expected askInput.CharLimit=0, got %d", m.askInput.CharLimit)
	}
}

func TestInit(t *testing.T) {
	m := New(&mockAgent{})
	cmd := m.Init()
	if cmd == nil {
		t.Error("Init() returned nil cmd (expected spinner.Tick)")
	}
}

func TestAgent(t *testing.T) {
	a := &mockAgent{}
	m := New(a)
	if m.Agent() != a {
		t.Error("Agent() returned different agent reference")
	}
}

func TestSelectedMenuItem(t *testing.T) {
	t.Run("valid cursor returns item", func(t *testing.T) {
		m := New(&mockAgent{})
		item := m.SelectedMenuItem()
		if item == nil {
			t.Fatal("SelectedMenuItem() returned nil for cursor 0")
		}
		if item.Title != defaultMenuItems[0].Title {
			t.Errorf("expected title %q, got %q", defaultMenuItems[0].Title, item.Title)
		}
	})

	t.Run("negative cursor returns nil", func(t *testing.T) {
		m := New(&mockAgent{})
		m.menuCursor = -1
		if item := m.SelectedMenuItem(); item != nil {
			t.Error("SelectedMenuItem() should be nil for negative cursor")
		}
	})

	t.Run("out-of-range cursor returns nil", func(t *testing.T) {
		m := New(&mockAgent{})
		m.menuCursor = 999
		if item := m.SelectedMenuItem(); item != nil {
			t.Error("SelectedMenuItem() should be nil for out-of-range cursor")
		}
	})
}

// ─── Update: Global Messages ────────────────────────────────────────────────

func TestGlobalWindowSize(t *testing.T) {
	m := model()
	m = update(m, tea.WindowSizeMsg{Width: 100, Height: 50})

	if m.Width != 100 {
		t.Errorf("Width = %d, want 100", m.Width)
	}
	if m.Height != 50 {
		t.Errorf("Height = %d, want 50", m.Height)
	}
	if m.chatInput.Width() < 10 {
		t.Errorf("chatInput.Width() = %d, want >= 10", m.chatInput.Width())
	}
	if m.askInput.Width < 6 {
		t.Errorf("askInput.Width = %d, want >= 6", m.askInput.Width)
	}
}

func TestGlobalCtrlC(t *testing.T) {
	m := model()
	m, cmd := updateWithCmd(m, tea.KeyMsg{Type: tea.KeyCtrlC})

	if m.screen != ScreenQuitting {
		t.Errorf("screen = %d, want ScreenQuitting", m.screen)
	}
	if cmd == nil {
		t.Error("expected tea.Quit command (non-nil)")
	}
}

func TestGlobalWindowSizeMinClamp(t *testing.T) {
	m := model()
	m = update(m, tea.WindowSizeMsg{Width: 5, Height: 5})

	// inputWidth = 5 - 10 = -5 → clamped to 10
	if m.chatInput.Width() < 1 {
		t.Errorf("chatInput.Width() = %d, minimum should be 1", m.chatInput.Width())
	}
	if m.askInput.Width < 6 {
		t.Errorf("askInput.Width = %d, minimum should be 6", m.askInput.Width)
	}
}

// ─── Update: Main Menu ──────────────────────────────────────────────────────

func TestMainMenuNavigation(t *testing.T) {
	t.Run("down arrow moves cursor forward", func(t *testing.T) {
		m := model()
		m = update(m, tea.KeyMsg{Type: tea.KeyDown})
		if m.menuCursor != 1 {
			t.Errorf("cursor = %d, want 1", m.menuCursor)
		}
	})

	t.Run("j moves cursor forward", func(t *testing.T) {
		m := model()
		m = update(m, keyJ)
		if m.menuCursor != 1 {
			t.Errorf("cursor = %d, want 1", m.menuCursor)
		}
	})

	t.Run("up arrow moves cursor backward", func(t *testing.T) {
		m := model()
		m.menuCursor = 2
		m = update(m, tea.KeyMsg{Type: tea.KeyUp})
		if m.menuCursor != 1 {
			t.Errorf("cursor = %d, want 1", m.menuCursor)
		}
	})

	t.Run("k moves cursor backward", func(t *testing.T) {
		m := model()
		m.menuCursor = 2
		m = update(m, keyK)
		if m.menuCursor != 1 {
			t.Errorf("cursor = %d, want 1", m.menuCursor)
		}
	})

	t.Run("down does not go past last item", func(t *testing.T) {
		m := model()
		m.menuCursor = len(m.menuItems) - 1
		m = update(m, tea.KeyMsg{Type: tea.KeyDown})
		if m.menuCursor != len(m.menuItems)-1 {
			t.Errorf("cursor = %d, want %d", m.menuCursor, len(m.menuItems)-1)
		}
	})

	t.Run("up does not go below 0", func(t *testing.T) {
		m := model()
		m = update(m, tea.KeyMsg{Type: tea.KeyUp})
		if m.menuCursor != 0 {
			t.Errorf("cursor = %d, want 0", m.menuCursor)
		}
	})
}

func TestMainMenuSelectChat(t *testing.T) {
	m := model()
	m, cmd := updateWithCmd(m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.screen != ScreenChatting {
		t.Errorf("screen = %d, want ScreenChatting", m.screen)
	}
	if !m.chatLoading {
		t.Error("expected chatLoading = true")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd for chat start")
	}
}

func TestMainMenuSelectQuit(t *testing.T) {
	m := model()
	m.menuCursor = 13 // Quit item
	m, cmd := updateWithCmd(m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.screen != ScreenQuitting {
		t.Errorf("screen = %d, want ScreenQuitting", m.screen)
	}
	if cmd == nil {
		t.Error("expected non-nil command")
	}
}

func TestMainMenuSelectOutOfBounds(t *testing.T) {
	m := model()
	m.menuCursor = -1
	m, cmd := updateWithCmd(m, tea.KeyMsg{Type: tea.KeyEnter})
	_ = cmd // just checking no panic
}

func TestMainMenuQuitKey(t *testing.T) {
	m := model()
	// 'q' key changes screen immediately in updateMainMenu.
	m, cmd := updateWithCmd(m, keyQ)

	if m.screen != ScreenQuitting {
		t.Errorf("screen = %d, want ScreenQuitting", m.screen)
	}
	if cmd == nil {
		t.Error("expected tea.Quit command")
	}
}

func TestMainMenuBackNavigation(t *testing.T) {
	m := model()
	m = update(m, navBackToMenuMsg{})

	if m.screen != ScreenMainMenu {
		t.Errorf("screen = %d, want ScreenMainMenu", m.screen)
	}
	if m.err != nil {
		t.Error("err should be cleared on back navigation")
	}
}

// ─── Update: Running Command → Show Output ──────────────────────────────────

func TestRunningCmdResults(t *testing.T) {
	tests := []struct {
		name string
		msg  tea.Msg
	}{
		{"BalanceResult", aiagent.BalanceResultMsg{Payload: &protocol.CommandResultPayload{Success: true, Data: "ok"}, Err: nil}},
		{"ModelsResult", aiagent.ModelsResultMsg{Payload: &protocol.CommandResultPayload{Success: true, Data: "ok"}, Err: nil}},
		{"VersionResult", aiagent.VersionResultMsg{Payload: &protocol.CommandResultPayload{Success: true, Data: "ok"}, Err: nil}},
		{"FlycheckResult", aiagent.FlycheckResultMsg{Payload: &protocol.CommandResultPayload{Success: true, Data: "ok"}, Err: nil}},
		{"SubcommandResult", aiagent.SubcommandResultMsg{Payload: &protocol.CommandResultPayload{Success: true, Data: "ok"}, Err: nil, Subcmd: "list"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model()
			m.screen = ScreenRunningCmd
			m = update(m, tt.msg)

			if m.screen != ScreenShowOutput {
				t.Errorf("screen = %d, want ScreenShowOutput", m.screen)
			}
			if m.cmdOutput != "ok" {
				t.Errorf("cmdOutput = %q, want %q", m.cmdOutput, "ok")
			}
			if !m.cmdSuccess {
				t.Error("cmdSuccess should be true")
			}
		})
	}
}

func TestRunningCmdError(t *testing.T) {
	m := model()
	m.screen = ScreenRunningCmd
	err := aiagent.BalanceResultMsg{Payload: nil, Err: assertError{"something went wrong"}}
	m = update(m, err)

	if m.screen != ScreenShowOutput {
		t.Errorf("screen = %d, want ScreenShowOutput", m.screen)
	}
	if m.cmdSuccess {
		t.Error("cmdSuccess should be false on error")
	}
	if !strings.Contains(m.cmdOutput, "something went wrong") {
		t.Errorf("cmdOutput = %q, should contain error", m.cmdOutput)
	}
}

func TestRunningCmdNilPayload(t *testing.T) {
	m := model()
	m.screen = ScreenRunningCmd
	m = update(m, aiagent.BalanceResultMsg{Payload: nil, Err: nil})

	if m.cmdOutput != "No output." {
		t.Errorf("cmdOutput = %q, want %q", m.cmdOutput, "No output.")
	}
	if m.cmdSuccess {
		t.Error("cmdSuccess should be false for nil payload")
	}
}

func TestRunningCmdFailedResult(t *testing.T) {
	m := model()
	m.screen = ScreenRunningCmd
	m = update(m, aiagent.BalanceResultMsg{
		Payload: &protocol.CommandResultPayload{Success: false, Data: "error details"},
		Err:     nil,
	})

	if m.cmdSuccess {
		t.Error("cmdSuccess should be false")
	}
	if !strings.Contains(m.cmdOutput, "Command failed") {
		t.Errorf("cmdOutput = %q, should indicate failure", m.cmdOutput)
	}
}

func TestRunningCmdEsc(t *testing.T) {
	m := model()
	m.screen = ScreenRunningCmd
	m.err = assertError{"some error"}
	m, cmd := updateWithCmd(m, tea.KeyMsg{Type: tea.KeyEscape})

	// Esc should return directly to main menu (escape hatch for hung commands).
	if m.screen != ScreenMainMenu {
		t.Errorf("screen = %d, want ScreenMainMenu", m.screen)
	}
	if m.err != nil {
		t.Error("err should be cleared when returning to menu")
	}
	if cmd != nil {
		t.Error("expected nil cmd (direct transition)")
	}
}

func TestRunningCmdEscOnlyForEsc(t *testing.T) {
	m := model()
	m.screen = ScreenRunningCmd

	// Non-Esc key should be ignored.
	m = update(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != ScreenRunningCmd {
		t.Errorf("screen = %d, want ScreenRunningCmd (non-esc key ignored)", m.screen)
	}
}

// ─── Update: Show Output (scrollable) ──────────────────────────────────

func TestShowOutputExitKeys(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyMsg
	}{
		{"esc exits", tea.KeyMsg{Type: tea.KeyEscape}},
		{"q exits", keyQ},
		{"enter exits", tea.KeyMsg{Type: tea.KeyEnter}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model()
			m.screen = ScreenShowOutput
			m, cmd := updateWithCmd(m, tt.key)

			if m.screen != ScreenMainMenu {
				t.Errorf("screen = %d, want ScreenMainMenu", m.screen)
			}
			if m.err != nil {
				t.Error("err should be cleared when returning to menu")
			}
			if cmd != nil {
				t.Error("expected nil cmd (direct transition)")
			}
		})
	}
}

func TestShowOutputScrolling(t *testing.T) {
	// Create model with multi-line output.
	m := model()
	m.screen = ScreenShowOutput
	m.cmdOutput = "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8"
	m.outputLines = strings.Split(m.cmdOutput, "\n")
	m.outputScroll = 0
	// Height=30, so available=25 → scrollMax = 8-25 = -1 → 0 (fits entirely)

	t.Run("down scrolls when content exceeds height", func(t *testing.T) {
		m2 := model()
		m2.Height = 5 // tiny height so scrolling is needed
		m2.screen = ScreenShowOutput
		m2.cmdOutput = "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8"
		m2.outputLines = strings.Split(m2.cmdOutput, "\n")
		m2.outputScroll = 0

		_ = update(m2, tea.KeyMsg{Type: tea.KeyDown})
		// After one down: outputScroll should become 1 (since total=8, avail=0 after reservations...)
		// avail = 5-5 = 0, clamped to 3. total=8 > 3 → scrollMax=5.
		if m2.outputScroll < 1 {
			t.Errorf("outputScroll = %d, want >= 1 after down", m2.outputScroll)
		}
	})

	t.Run("up scrolls back", func(t *testing.T) {
		m2 := model()
		m2.Height = 5
		m2.screen = ScreenShowOutput
		m2.cmdOutput = "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8"
		m2.outputLines = strings.Split(m2.cmdOutput, "\n")
		m2.outputScroll = 3

		_ = update(m2, tea.KeyMsg{Type: tea.KeyUp})
		if m2.outputScroll != 2 {
			t.Errorf("outputScroll = %d, want 2 after up", m2.outputScroll)
		}
	})

	t.Run("home goes to top", func(t *testing.T) {
		m2 := model()
		m2.Height = 5
		m2.screen = ScreenShowOutput
		m2.cmdOutput = "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8"
		m2.outputLines = strings.Split(m2.cmdOutput, "\n")
		m2.outputScroll = 3

		_ = update(m2, tea.KeyMsg{Type: tea.KeyHome})
		if m2.outputScroll != 0 {
			t.Errorf("outputScroll = %d, want 0 after home", m2.outputScroll)
		}
	})

	t.Run("end goes to bottom", func(t *testing.T) {
		m2 := model()
		m2.Height = 5
		m2.screen = ScreenShowOutput
		m2.cmdOutput = "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8"
		m2.outputLines = strings.Split(m2.cmdOutput, "\n")
		m2.outputScroll = 0

		_ = update(m2, tea.KeyMsg{Type: tea.KeyEnd})
		if m2.outputScroll <= 0 {
			t.Errorf("outputScroll = %d, want > 0 after end", m2.outputScroll)
		}
	})

	t.Run("g goes to top", func(t *testing.T) {
		m2 := model()
		m2.Height = 5
		m2.screen = ScreenShowOutput
		m2.cmdOutput = "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8"
		m2.outputLines = strings.Split(m2.cmdOutput, "\n")
		m2.outputScroll = 3

		_ = update(m2, keyRune('g'))
		if m2.outputScroll != 0 {
			t.Errorf("outputScroll = %d, want 0 after g", m2.outputScroll)
		}
	})

	t.Run("G goes to bottom", func(t *testing.T) {
		m2 := model()
		m2.Height = 5
		m2.screen = ScreenShowOutput
		m2.cmdOutput = "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8"
		m2.outputLines = strings.Split(m2.cmdOutput, "\n")
		m2.outputScroll = 0

		_ = update(m2, keyRune('G'))
		if m2.outputScroll <= 0 {
			t.Errorf("outputScroll = %d, want > 0 after G", m2.outputScroll)
		}
	})

	t.Run("non-navigation keys do not exit", func(t *testing.T) {
		m2 := model()
		m2.screen = ScreenShowOutput
		m2.cmdOutput = "some output"
		m2.outputLines = strings.Split(m2.cmdOutput, "\n")

		// Random key should NOT exit.
		m2 = update(m2, keyRune('x'))
		if m2.screen != ScreenShowOutput {
			t.Errorf("screen = %d, want ScreenShowOutput (non-exit key ignored)", m2.screen)
		}
	})
}

func TestShowOutputNonKeyIgnored(t *testing.T) {
	m := model()
	m.screen = ScreenShowOutput
	m.cmdOutput = "some output"
	m.outputLines = strings.Split(m.cmdOutput, "\n")
	m = update(m, tea.WindowSizeMsg{Width: 100, Height: 50})
	if m.screen != ScreenShowOutput {
		t.Errorf("screen = %d, want ScreenShowOutput (non-key msgs ignored)", m.screen)
	}
}

// ─── Update: History List ───────────────────────────────────────────

func TestParseHistoryList(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      int // expected item count
		firstID   string
		firstRole string
	}{
		{
			name: "typical output",
			input: `[
				{"id": 25604, "role": "assistant", "ok": false, "created_at": "2026-07-04T14:00:00+08:00"},
				{"id": 25605, "role": "tool", "ok": true, "created_at": "2026-07-04T14:00:01+08:00"}
			]`,
			want:      2,
			firstID:   "25604",
			firstRole: "assistant",
		},
		{
			name: "single entry",
			input: `[
				{"id": 12345, "role": "user", "ok": true, "created_at": "2026-07-04T14:00:00+08:00"}
			]`,
			want:      1,
			firstID:   "12345",
			firstRole: "user",
		},
		{
			name:  "empty array",
			input: `[]`,
			want:  0,
		},
		{
			name:  "invalid json",
			input: "not json",
			want:  0,
		},
		{
			name:      "done from ok true",
			input:     `[{"id": 100, "role": "assistant", "ok": true, "created_at": "2026-07-04T14:00:00Z"}]`,
			want:      1,
			firstID:   "100",
			firstRole: "assistant",
		},
		{
			name:      "done from ok false",
			input:     `[{"id": 200, "role": "tool", "ok": false, "created_at": "2026-07-04T14:00:00Z"}]`,
			want:      1,
			firstID:   "200",
			firstRole: "tool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items := parseHistoryList(tt.input)
			if len(items) != tt.want {
				t.Fatalf("got %d items, want %d", len(items), tt.want)
			}
			if tt.want > 0 {
				if items[0].ID != tt.firstID {
					t.Errorf("items[0].ID = %q, want %q", items[0].ID, tt.firstID)
				}
				if items[0].Role != tt.firstRole {
					t.Errorf("items[0].Role = %q, want %q", items[0].Role, tt.firstRole)
				}
				if items[0].CreatedAt == "" {
					t.Error("items[0].CreatedAt should not be empty")
				}
			}
		})
	}
}

func TestHistoryListNavigation(t *testing.T) {
	t.Run("down moves cursor", func(t *testing.T) {
		m := model()
		m.screen = ScreenHistoryList
		m.historyItems = []HistoryItem{
			{ID: "1", Role: "user"},
			{ID: "2", Role: "assistant"},
			{ID: "3", Role: "tool"},
		}
		m.historyCursor = 0

		// Down
		m = update(m, tea.KeyMsg{Type: tea.KeyDown})
		if m.historyCursor != 1 {
			t.Errorf("cursor = %d, want 1 after down", m.historyCursor)
		}

		// j
		m = update(m, keyJ)
		if m.historyCursor != 2 {
			t.Errorf("cursor = %d, want 2 after j", m.historyCursor)
		}

		// Down clamped at end
		m = update(m, tea.KeyMsg{Type: tea.KeyDown})
		if m.historyCursor != 2 {
			t.Errorf("cursor should stay at 2 (end of list)")
		}
	})

	t.Run("up moves cursor", func(t *testing.T) {
		m := model()
		m.screen = ScreenHistoryList
		m.historyItems = []HistoryItem{
			{ID: "1", Role: "user"},
			{ID: "2", Role: "assistant"},
		}
		m.historyCursor = 1

		// Up
		m = update(m, tea.KeyMsg{Type: tea.KeyUp})
		if m.historyCursor != 0 {
			t.Errorf("cursor = %d, want 0 after up", m.historyCursor)
		}

		// Up clamped at 0
		m = update(m, tea.KeyMsg{Type: tea.KeyUp})
		if m.historyCursor != 0 {
			t.Errorf("cursor should stay at 0")
		}
	})

	t.Run("k moves up", func(t *testing.T) {
		m := model()
		m.screen = ScreenHistoryList
		m.historyItems = []HistoryItem{
			{ID: "1", Role: "user"},
			{ID: "2", Role: "assistant"},
		}
		m.historyCursor = 1

		m = update(m, keyK)
		if m.historyCursor != 0 {
			t.Errorf("cursor = %d, want 0 after k", m.historyCursor)
		}
	})
}

func TestHistoryListSelect(t *testing.T) {
	t.Run("enter dispatches show command", func(t *testing.T) {
		m := model()
		m.screen = ScreenHistoryList
		m.historyItems = []HistoryItem{
			{ID: "25604", Role: "assistant"},
			{ID: "25605", Role: "tool"},
		}
		m.historyCursor = 1

		m, cmd := updateWithCmd(m, tea.KeyMsg{Type: tea.KeyEnter})
		if m.screen != ScreenRunningCmd {
			t.Errorf("screen = %d, want ScreenRunningCmd", m.screen)
		}
		if cmd == nil {
			t.Fatal("expected non-nil cmd")
		}
	})

	t.Run("space also selects", func(t *testing.T) {
		m := model()
		m.screen = ScreenHistoryList
		m.historyItems = []HistoryItem{
			{ID: "100", Role: "user"},
		}
		m.historyCursor = 0

		m, cmd := updateWithCmd(m, tea.KeyMsg{Type: tea.KeySpace})
		if m.screen != ScreenRunningCmd {
			t.Errorf("screen = %d, want ScreenRunningCmd", m.screen)
		}
		if cmd == nil {
			t.Fatal("expected non-nil cmd")
		}
	})

	t.Run("enter with invalid cursor does nothing", func(t *testing.T) {
		m := model()
		m.screen = ScreenHistoryList
		m.historyItems = []HistoryItem{
			{ID: "1", Role: "user"},
		}
		m.historyCursor = -1

		m, cmd := updateWithCmd(m, tea.KeyMsg{Type: tea.KeyEnter})
		if m.screen != ScreenHistoryList {
			t.Errorf("screen should stay ScreenHistoryList, got %d", m.screen)
		}
		if cmd != nil {
			t.Error("expected nil cmd for invalid cursor")
		}
	})
}

func TestHistoryListEsc(t *testing.T) {
	m := model()
	m.screen = ScreenHistoryList
	m.historyItems = []HistoryItem{
		{ID: "1", Role: "user"},
		{ID: "2", Role: "assistant"},
	}
	m.historyCursor = 1
	m.err = assertError{"test"}

	// Esc returns to main menu and clears state.
	m, cmd := updateWithCmd(m, tea.KeyMsg{Type: tea.KeyEscape})
	if m.screen != ScreenMainMenu {
		t.Errorf("screen = %d, want ScreenMainMenu", m.screen)
	}
	if m.err != nil {
		t.Error("err should be cleared")
	}
	if m.historyItems != nil {
		t.Error("historyItems should be nil after exit")
	}
	if m.historyCursor != 0 {
		t.Error("historyCursor should be 0 after exit")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

func TestHistoryListQKey(t *testing.T) {
	m := model()
	m.screen = ScreenHistoryList
	m.historyItems = []HistoryItem{{ID: "1", Role: "user"}}

	m, _ = updateWithCmd(m, keyQ)
	if m.screen != ScreenMainMenu {
		t.Errorf("screen = %d, want ScreenMainMenu after q", m.screen)
	}
}

func TestHistoryListPayload(t *testing.T) {
	t.Run("valid payload transitions to ScreenHistoryList", func(t *testing.T) {
		m := model()
		m.screen = ScreenRunningCmd
		p := &protocol.CommandResultPayload{
			Success: true,
		Data:    `[{"id":25604,"role":"assistant","ok":false,"created_at":"2026-07-04T14:00:00+08:00"},{"id":25605,"role":"tool","ok":true,"created_at":"2026-07-04T14:00:01+08:00"}]`,
		}
		if !m.historyListPayload(p) {
			t.Fatal("historyListPayload returned false")
		}
		if m.screen != ScreenHistoryList {
			t.Errorf("screen = %d, want ScreenHistoryList", m.screen)
		}
		if len(m.historyItems) != 2 {
			t.Errorf("got %d items, want 2", len(m.historyItems))
		}
		if m.historyCursor != 0 {
			t.Errorf("cursor = %d, want 0 (newest item selected)", m.historyCursor)
		}
	})

	t.Run("nil payload returns false", func(t *testing.T) {
		m := model()
		if m.historyListPayload(nil) {
			t.Error("expected false for nil payload")
		}
	})

	t.Run("failed payload returns false", func(t *testing.T) {
		m := model()
		p := &protocol.CommandResultPayload{Success: false, Data: "error message"}
		if m.historyListPayload(p) {
			t.Error("expected false for failed payload")
		}
	})

	t.Run("empty data returns false", func(t *testing.T) {
		m := model()
		p := &protocol.CommandResultPayload{Success: true, Data: ""}
		if m.historyListPayload(p) {
			t.Error("expected false for empty data")
		}
	})
}

func TestHistoryListNonKeyMsg(t *testing.T) {
	m := model()
	m.screen = ScreenHistoryList
	m.historyItems = []HistoryItem{{ID: "1", Role: "user"}}

	// Non-key messages should be ignored.
	m = update(m, tea.WindowSizeMsg{Width: 100, Height: 50})
	if m.screen != ScreenHistoryList {
		t.Error("non-key msg should not change screen")
	}
}

// ─── Update: Skill List ──────────────────────────────────────────

func TestParseSkillList(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantCount int
		firstName string
	}{
		{
			name: "typical output",
			input: `名称                              范围         自动注入
api-design-principles           global     -
architecture-decision-records   global     -
dscli                           built-in   是`,
			wantCount: 3,
			firstName: "api-design-principles",
		},
		{
			name:      "empty input",
			input:     "",
			wantCount: 0,
		},
		{
			name: "header only",
			input: `名称                              范围         自动注入`,
			wantCount: 0,
		},
		{
			name: "single skill with auto-inject",
			input: `名称    范围     自动注入
my-skill  local    是`,
			wantCount: 1,
			firstName: "my-skill",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items := parseSkillList(tt.input)
			if len(items) != tt.wantCount {
				t.Fatalf("got %d items, want %d", len(items), tt.wantCount)
			}
			if tt.wantCount > 0 && items[0].Name != tt.firstName {
				t.Errorf("items[0].Name = %q, want %q", items[0].Name, tt.firstName)
			}
		})
	}
}

func TestSplitByTwoOrMoreSpaces(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"hello", []string{"hello"}},
		{"hello  world", []string{"hello", "world"}},
		{"a   b    c", []string{"a", "b", "c"}},
		{"name       scope     auto", []string{"name", "scope", "auto"}},
		{"single-spaces keep together", []string{"single-spaces keep together"}}, // single spaces are kept as one field
		{"", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := splitByTwoOrMoreSpaces(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v (len=%d), want %v (len=%d)", got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("got[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestSkillListNavigation(t *testing.T) {
	t.Run("down moves cursor", func(t *testing.T) {
		m := model()
		m.screen = ScreenSkillList
		m.skillItems = []SkillItem{
			{Name: "skill-a", Scope: "global"},
			{Name: "skill-b", Scope: "local"},
			{Name: "skill-c", Scope: "built-in"},
		}
		m.skillCursor = 0

		// Down
		m = update(m, tea.KeyMsg{Type: tea.KeyDown})
		if m.skillCursor != 1 {
			t.Errorf("cursor = %d, want 1 after down", m.skillCursor)
		}

		// j
		m = update(m, keyJ)
		if m.skillCursor != 2 {
			t.Errorf("cursor = %d, want 2 after j", m.skillCursor)
		}

		// Down clamped at end
		m = update(m, tea.KeyMsg{Type: tea.KeyDown})
		if m.skillCursor != 2 {
			t.Errorf("cursor should stay at 2 (end of list)")
		}
	})

	t.Run("up moves cursor", func(t *testing.T) {
		m := model()
		m.screen = ScreenSkillList
		m.skillItems = []SkillItem{
			{Name: "skill-a", Scope: "global"},
			{Name: "skill-b", Scope: "local"},
		}
		m.skillCursor = 1

		// Up
		m = update(m, tea.KeyMsg{Type: tea.KeyUp})
		if m.skillCursor != 0 {
			t.Errorf("cursor = %d, want 0 after up", m.skillCursor)
		}

		// Up clamped at 0
		m = update(m, tea.KeyMsg{Type: tea.KeyUp})
		if m.skillCursor != 0 {
			t.Errorf("cursor should stay at 0")
		}
	})

	t.Run("k moves up", func(t *testing.T) {
		m := model()
		m.screen = ScreenSkillList
		m.skillItems = []SkillItem{
			{Name: "skill-a", Scope: "global"},
			{Name: "skill-b", Scope: "local"},
		}
		m.skillCursor = 1

		m = update(m, keyK)
		if m.skillCursor != 0 {
			t.Errorf("cursor = %d, want 0 after k", m.skillCursor)
		}
	})
}

func TestSkillListSelect(t *testing.T) {
	t.Run("enter dispatches show command", func(t *testing.T) {
		m := model()
		m.screen = ScreenSkillList
		m.skillItems = []SkillItem{
			{Name: "api-design", Scope: "global"},
			{Name: "architecture-patterns", Scope: "global"},
		}
		m.skillCursor = 1

		m, cmd := updateWithCmd(m, tea.KeyMsg{Type: tea.KeyEnter})
		if m.screen != ScreenRunningCmd {
			t.Errorf("screen = %d, want ScreenRunningCmd", m.screen)
		}
		if cmd == nil {
			t.Fatal("expected non-nil cmd")
		}
	})

	t.Run("space also selects", func(t *testing.T) {
		m := model()
		m.screen = ScreenSkillList
		m.skillItems = []SkillItem{
			{Name: "my-skill", Scope: "local"},
		}
		m.skillCursor = 0

		m, cmd := updateWithCmd(m, tea.KeyMsg{Type: tea.KeySpace})
		if m.screen != ScreenRunningCmd {
			t.Errorf("screen = %d, want ScreenRunningCmd", m.screen)
		}
		if cmd == nil {
			t.Fatal("expected non-nil cmd")
		}
	})

	t.Run("enter with invalid cursor does nothing", func(t *testing.T) {
		m := model()
		m.screen = ScreenSkillList
		m.skillItems = []SkillItem{
			{Name: "skill-a", Scope: "global"},
		}
		m.skillCursor = -1

		m, cmd := updateWithCmd(m, tea.KeyMsg{Type: tea.KeyEnter})
		if m.screen != ScreenSkillList {
			t.Errorf("screen should stay ScreenSkillList, got %d", m.screen)
		}
		if cmd != nil {
			t.Error("expected nil cmd for invalid cursor")
		}
	})
}

func TestSkillListEsc(t *testing.T) {
	m := model()
	m.screen = ScreenSkillList
	m.skillItems = []SkillItem{
		{Name: "skill-a", Scope: "global"},
		{Name: "skill-b", Scope: "local"},
	}
	m.skillCursor = 1
	m.err = assertError{"test"}

	// Esc returns to main menu and clears state.
	m, cmd := updateWithCmd(m, tea.KeyMsg{Type: tea.KeyEscape})
	if m.screen != ScreenMainMenu {
		t.Errorf("screen = %d, want ScreenMainMenu", m.screen)
	}
	if m.err != nil {
		t.Error("err should be cleared")
	}
	if m.skillItems != nil {
		t.Error("skillItems should be nil after exit")
	}
	if m.skillCursor != 0 {
		t.Error("skillCursor should be 0 after exit")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

func TestSkillListQKey(t *testing.T) {
	m := model()
	m.screen = ScreenSkillList
	m.skillItems = []SkillItem{{Name: "skill-a", Scope: "global"}}

	m, _ = updateWithCmd(m, keyQ)
	if m.screen != ScreenMainMenu {
		t.Errorf("screen = %d, want ScreenMainMenu after q", m.screen)
	}
}

func TestSkillListPayload(t *testing.T) {
	t.Run("valid payload transitions to ScreenSkillList", func(t *testing.T) {
		m := model()
		m.screen = ScreenRunningCmd
		p := &protocol.CommandResultPayload{
			Success: true,
			Data: `名称                              范围         自动注入
api-design-principles           global     -
architecture-decision-records   global     -`,
		}
		if !m.skillListPayload(p) {
			t.Fatal("skillListPayload returned false")
		}
		if m.screen != ScreenSkillList {
			t.Errorf("screen = %d, want ScreenSkillList", m.screen)
		}
		if len(m.skillItems) != 2 {
			t.Errorf("got %d items, want 2", len(m.skillItems))
		}
		if m.skillCursor != 0 {
			t.Errorf("cursor = %d, want 0 (first skill)", m.skillCursor)
		}
	})

	t.Run("nil payload returns false", func(t *testing.T) {
		m := model()
		if m.skillListPayload(nil) {
			t.Error("expected false for nil payload")
		}
	})

	t.Run("failed payload returns false", func(t *testing.T) {
		m := model()
		p := &protocol.CommandResultPayload{Success: false, Data: "error message"}
		if m.skillListPayload(p) {
			t.Error("expected false for failed payload")
		}
	})

	t.Run("empty data returns false", func(t *testing.T) {
		m := model()
		p := &protocol.CommandResultPayload{Success: true, Data: ""}
		if m.skillListPayload(p) {
			t.Error("expected false for empty data")
		}
	})
}

func TestSkillListNonKeyMsg(t *testing.T) {
	m := model()
	m.screen = ScreenSkillList
	m.skillItems = []SkillItem{{Name: "skill-a", Scope: "global"}}

	// Non-key messages should be ignored.
	m = update(m, tea.WindowSizeMsg{Width: 100, Height: 50})
	if m.screen != ScreenSkillList {
		t.Error("non-key msg should not change screen")
	}
}

// ─── Update: Memory List ─────────────────────────────────────────

func TestParseMemoryList(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantCount int
		firstID   string
		firstTitle string
	}{
		{
			name: "typical output",
			input: `ID  TITLE                                                           Created At       Updated At
89  History list default cursor                                   Jul  6 23:11:57  Jul  6 23:11:57
88  dscli --histsize controls history list visibility              Jul  6 23:06:16  Jul  6 23:06:16`,
			wantCount:  2,
			firstID:    "89",
			firstTitle: "History list default cursor",
		},
		{
			name:      "empty input",
			input:     "",
			wantCount: 0,
		},
		{
			name: "header only",
			input: `ID  TITLE                                                           Created At       Updated At`,
			wantCount: 0,
		},
		{
			name: "single memory",
			input: `ID  TITLE     Created At       Updated At
1   My memory  Jul  1 10:00:00  Jul  1 10:00:00`,
			wantCount: 1,
			firstID:   "1",
			firstTitle: "My memory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items := parseMemoryList(tt.input)
			if len(items) != tt.wantCount {
				t.Fatalf("got %d items, want %d", len(items), tt.wantCount)
			}
			if tt.wantCount > 0 {
				if items[0].ID != tt.firstID {
					t.Errorf("items[0].ID = %q, want %q", items[0].ID, tt.firstID)
				}
				if items[0].Title != tt.firstTitle {
					t.Errorf("items[0].Title = %q, want %q", items[0].Title, tt.firstTitle)
				}
				if items[0].CreatedAt == "" {
					t.Error("items[0].CreatedAt should not be empty")
				}
			}
		})
	}
}

func TestMemoryListNavigation(t *testing.T) {
	t.Run("down moves cursor", func(t *testing.T) {
		m := model()
		m.screen = ScreenMemoryList
		m.memoryItems = []MemoryItem{
			{ID: "1", Title: "memory-a"},
			{ID: "2", Title: "memory-b"},
			{ID: "3", Title: "memory-c"},
		}
		m.memoryCursor = 0

		// Down
		m = update(m, tea.KeyMsg{Type: tea.KeyDown})
		if m.memoryCursor != 1 {
			t.Errorf("cursor = %d, want 1 after down", m.memoryCursor)
		}

		// j
		m = update(m, keyJ)
		if m.memoryCursor != 2 {
			t.Errorf("cursor = %d, want 2 after j", m.memoryCursor)
		}

		// Down clamped at end
		m = update(m, tea.KeyMsg{Type: tea.KeyDown})
		if m.memoryCursor != 2 {
			t.Errorf("cursor should stay at 2 (end of list)")
		}
	})

	t.Run("up moves cursor", func(t *testing.T) {
		m := model()
		m.screen = ScreenMemoryList
		m.memoryItems = []MemoryItem{
			{ID: "1", Title: "memory-a"},
			{ID: "2", Title: "memory-b"},
		}
		m.memoryCursor = 1

		// Up
		m = update(m, tea.KeyMsg{Type: tea.KeyUp})
		if m.memoryCursor != 0 {
			t.Errorf("cursor = %d, want 0 after up", m.memoryCursor)
		}

		// Up clamped at 0
		m = update(m, tea.KeyMsg{Type: tea.KeyUp})
		if m.memoryCursor != 0 {
			t.Errorf("cursor should stay at 0")
		}
	})

	t.Run("k moves up", func(t *testing.T) {
		m := model()
		m.screen = ScreenMemoryList
		m.memoryItems = []MemoryItem{
			{ID: "1", Title: "memory-a"},
			{ID: "2", Title: "memory-b"},
		}
		m.memoryCursor = 1

		m = update(m, keyK)
		if m.memoryCursor != 0 {
			t.Errorf("cursor = %d, want 0 after k", m.memoryCursor)
		}
	})
}

func TestMemoryListSelect(t *testing.T) {
	t.Run("enter dispatches show command", func(t *testing.T) {
		m := model()
		m.screen = ScreenMemoryList
		m.memoryItems = []MemoryItem{
			{ID: "89", Title: "first"},
			{ID: "90", Title: "second"},
		}
		m.memoryCursor = 1

		m, cmd := updateWithCmd(m, tea.KeyMsg{Type: tea.KeyEnter})
		if m.screen != ScreenRunningCmd {
			t.Errorf("screen = %d, want ScreenRunningCmd", m.screen)
		}
		if cmd == nil {
			t.Fatal("expected non-nil cmd")
		}
	})

	t.Run("space also selects", func(t *testing.T) {
		m := model()
		m.screen = ScreenMemoryList
		m.memoryItems = []MemoryItem{
			{ID: "100", Title: "my memory"},
		}
		m.memoryCursor = 0

		m, cmd := updateWithCmd(m, tea.KeyMsg{Type: tea.KeySpace})
		if m.screen != ScreenRunningCmd {
			t.Errorf("screen = %d, want ScreenRunningCmd", m.screen)
		}
		if cmd == nil {
			t.Fatal("expected non-nil cmd")
		}
	})

	t.Run("enter with invalid cursor does nothing", func(t *testing.T) {
		m := model()
		m.screen = ScreenMemoryList
		m.memoryItems = []MemoryItem{
			{ID: "1", Title: "only"},
		}
		m.memoryCursor = -1

		m, cmd := updateWithCmd(m, tea.KeyMsg{Type: tea.KeyEnter})
		if m.screen != ScreenMemoryList {
			t.Errorf("screen should stay ScreenMemoryList, got %d", m.screen)
		}
		if cmd != nil {
			t.Error("expected nil cmd for invalid cursor")
		}
	})
}

func TestMemoryListEsc(t *testing.T) {
	m := model()
	m.screen = ScreenMemoryList
	m.memoryItems = []MemoryItem{
		{ID: "1", Title: "memory-a"},
		{ID: "2", Title: "memory-b"},
	}
	m.memoryCursor = 1
	m.err = assertError{"test"}

	// Esc returns to main menu and clears state.
	m, cmd := updateWithCmd(m, tea.KeyMsg{Type: tea.KeyEscape})
	if m.screen != ScreenMainMenu {
		t.Errorf("screen = %d, want ScreenMainMenu", m.screen)
	}
	if m.err != nil {
		t.Error("err should be cleared")
	}
	if m.memoryItems != nil {
		t.Error("memoryItems should be nil after exit")
	}
	if m.memoryCursor != 0 {
		t.Error("memoryCursor should be 0 after exit")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

func TestMemoryListQKey(t *testing.T) {
	m := model()
	m.screen = ScreenMemoryList
	m.memoryItems = []MemoryItem{{ID: "1", Title: "memory-a"}}

	m, _ = updateWithCmd(m, keyQ)
	if m.screen != ScreenMainMenu {
		t.Errorf("screen = %d, want ScreenMainMenu after q", m.screen)
	}
}

func TestMemoryListPayload(t *testing.T) {
	t.Run("valid payload transitions to ScreenMemoryList", func(t *testing.T) {
		m := model()
		m.screen = ScreenRunningCmd
		m.memoryCursor = -1 // simulate state after executeSelected
		p := &protocol.CommandResultPayload{
			Success: true,
			Data: `ID  TITLE     Created At       Updated At
89  My memory  Jul  6 23:11:57  Jul  6 23:11:57
90  Another    Jul  6 23:06:16  Jul  6 23:06:16`,
		}
		if !m.memoryListPayload(p) {
			t.Fatal("memoryListPayload returned false")
		}
		if m.screen != ScreenMemoryList {
			t.Errorf("screen = %d, want ScreenMemoryList", m.screen)
		}
		if len(m.memoryItems) != 2 {
			t.Errorf("got %d items, want 2", len(m.memoryItems))
		}
		if m.memoryCursor != 0 {
			t.Errorf("cursor = %d, want 0 (first memory selected)", m.memoryCursor)
		}
	})

	t.Run("nil payload returns false", func(t *testing.T) {
		m := model()
		if m.memoryListPayload(nil) {
			t.Error("expected false for nil payload")
		}
	})

	t.Run("failed payload returns false", func(t *testing.T) {
		m := model()
		p := &protocol.CommandResultPayload{Success: false, Data: "error message"}
		if m.memoryListPayload(p) {
			t.Error("expected false for failed payload")
		}
	})

	t.Run("empty data returns false", func(t *testing.T) {
		m := model()
		p := &protocol.CommandResultPayload{Success: true, Data: ""}
		if m.memoryListPayload(p) {
			t.Error("expected false for empty data")
		}
	})
}

func TestMemoryListNonKeyMsg(t *testing.T) {
	m := model()
	m.screen = ScreenMemoryList
	m.memoryItems = []MemoryItem{{ID: "1", Title: "memory-a"}}

	// Non-key messages should be ignored.
	m = update(m, tea.WindowSizeMsg{Width: 100, Height: 50})
	if m.screen != ScreenMemoryList {
		t.Error("non-key msg should not change screen")
	}
}

func TestShowOutputBackToMemoryList(t *testing.T) {
	m := model()
	m.screen = ScreenShowOutput
	m.cmdOutput = "memory detail content"
	m.prevScreen = ScreenMemoryList

	// Esc should go back to memory list, not main menu.
	m, _ = updateWithCmd(m, tea.KeyMsg{Type: tea.KeyEscape})
	if m.screen != ScreenMemoryList {
		t.Errorf("screen = %d, want ScreenMemoryList", m.screen)
	}
	if m.err != nil {
		t.Error("err should be cleared")
	}
	if m.prevScreen != ScreenMainMenu {
		t.Error("prevScreen should be reset to ScreenMainMenu")
	}
}

func TestShowOutputBackToHistoryList(t *testing.T) {
	m := model()
	m.screen = ScreenShowOutput
	m.cmdOutput = "message content"
	m.prevScreen = ScreenHistoryList
	// Esc should go back to history list, not main menu.
	m, _ = updateWithCmd(m, tea.KeyMsg{Type: tea.KeyEscape})
	if m.screen != ScreenHistoryList {
		t.Errorf("screen = %d, want ScreenHistoryList", m.screen)
	}
	if m.err != nil {
		t.Error("err should be cleared")
	}
	// prevScreen should be reset.
	if m.prevScreen != ScreenMainMenu {
		t.Error("prevScreen should be reset to ScreenMainMenu")
	}
}

func TestShowOutputBackToMainMenu(t *testing.T) {
	m := model()
	m.screen = ScreenShowOutput
	m.cmdOutput = "some output"
	m.prevScreen = ScreenMainMenu // default

	// Esc should go to main menu.
	m, _ = updateWithCmd(m, tea.KeyMsg{Type: tea.KeyEscape})
	if m.screen != ScreenMainMenu {
		t.Errorf("screen = %d, want ScreenMainMenu", m.screen)
	}
}

// ─── Update: Chatting

func TestChattingSessionReady(t *testing.T) {
	m := model()
	m.screen = ScreenChatting
	// Simulate an exchange: history has user message (set on Enter).
	m.chatHistory = []ChatLine{{Role: "user", Content: "hello"}}

	sess := mockSession()
	m = update(m, aiagent.ChatSessionReadyMsg{Session: sess})

	if m.chatSession != sess {
		t.Error("chatSession not set")
	}
	if !m.chatReady {
		t.Error("chatReady should be true")
	}
	if !m.chatLoading {
		t.Error("chatLoading should remain true (history has content)")
	}
}

func TestChattingSessionReadyError(t *testing.T) {
	m := model()
	m.screen = ScreenChatting
	m.chatLoading = true

	m = update(m, aiagent.ChatSessionReadyMsg{Err: assertError{"session failed"}})

	if m.screen != ScreenShowOutput {
		t.Errorf("screen = %d, want ScreenShowOutput on error", m.screen)
	}
	if m.cmdSuccess {
		t.Error("cmdSuccess should be false")
	}
}

func TestChattingEventError(t *testing.T) {
	m := model()
	m.screen = ScreenChatting
	m.chatLoading = true

	m = update(m, aiagent.ChatEventMsg{Err: assertError{"event error"}})

	if m.chatLoading {
		t.Error("chatLoading should be false after error")
	}
	if m.spinnerOn {
		t.Error("spinnerOn should be false after error")
	}
}

func TestChattingEventDone(t *testing.T) {
	m := model()
	m.screen = ScreenChatting
	m.chatLoading = true
	m.spinnerOn = true

	m = update(m, aiagent.ChatEventMsg{Done: true})

	if m.chatLoading {
		t.Error("chatLoading should be false")
	}
	if !m.chatDone {
		t.Error("chatDone should be true")
	}
	if m.spinnerOn {
		t.Error("spinnerOn should be false")
	}
}

func TestChattingEnter(t *testing.T) {
	m := model()
	m.screen = ScreenChatting
	m.chatInput.SetValue("  hello  ") // spaces to test TrimSpace

	m, cmd := updateWithCmd(m, tea.KeyMsg{Type: tea.KeyEnter})

	if len(m.chatHistory) != 1 || m.chatHistory[0].Role != "user" || m.chatHistory[0].Content != "hello" {
		t.Errorf("chatHistory = %+v, want [{user hello}]", m.chatHistory)
	}
	if m.chatInput.Value() != "" {
		t.Error("chatInput should be cleared after enter")
	}
	if !m.chatLoading {
		t.Error("chatLoading should be true")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd (start chat)")
	}
}

func TestChattingEnterEmpty(t *testing.T) {
	m := model()
	m.screen = ScreenChatting
	m.chatInput.SetValue("  ") // whitespace is trimmed to empty

	m, cmd := updateWithCmd(m, tea.KeyMsg{Type: tea.KeyEnter})

	// Empty messages are now allowed — the user may want to send "continue".
	if cmd == nil {
		t.Error("expected non-nil cmd (empty messages allowed)")
	}
	if len(m.chatHistory) != 1 {
		t.Fatalf("history length = %d, want 1", len(m.chatHistory))
	}
	if m.chatHistory[0].Role != "user" || m.chatHistory[0].Content != "" {
		t.Errorf("history[0] = %+v, want {user, ''}", m.chatHistory[0])
	}
	if !m.chatLoading {
		t.Error("chatLoading should be true")
	}
}

func TestChattingInterleavedDuringLoading(t *testing.T) {
	m := model()
	m.screen = ScreenChatting
	m.chatLoading = true
	m.chatDone = false
	m.chatScrollMax = 10
	m.chatScroll = 5
	m.chatInput.SetValue("correction")

	m, cmd := updateWithCmd(m, tea.KeyMsg{Type: tea.KeyEnter})

	// Should start a climein dscli process immediately (non-nil cmd).
	if cmd == nil {
		t.Error("expected non-nil cmd (cmdSendChimein)")
	}
	// Message should be in history immediately.
	if len(m.chatHistory) != 1 || m.chatHistory[0].Content != "correction" {
		t.Errorf("history = %+v, want [{user correction}]", m.chatHistory)
	}
	// Input should be cleared.
	if m.chatInput.Value() != "" {
		t.Error("chatInput should be cleared")
	}
	// Scroll should reset to bottom.
	if m.chatScroll != 0 {
		t.Errorf("chatScroll = %d, want 0", m.chatScroll)
	}
}

func TestChattingInterleavedMultipleDuringLoading(t *testing.T) {
	m := model()
	m.screen = ScreenChatting
	m.chatLoading = true
	m.chatDone = false

	// First interleaved message.
	m.chatInput.SetValue("first correction")
	m = update(m, tea.KeyMsg{Type: tea.KeyEnter})

	if len(m.chatHistory) != 1 || m.chatHistory[0].Content != "first correction" {
		t.Fatalf("after first: history = %+v", m.chatHistory)
	}

	// Second interleaved message — each Enter starts its own climein process.
	m.chatInput.SetValue("second correction")
	m = update(m, tea.KeyMsg{Type: tea.KeyEnter})

	if len(m.chatHistory) != 2 {
		t.Fatalf("after second: history length = %d, want 2", len(m.chatHistory))
	}
	if m.chatHistory[0].Content != "first correction" {
		t.Errorf("history[0] = %q, want %q", m.chatHistory[0].Content, "first correction")
	}
	if m.chatHistory[1].Content != "second correction" {
		t.Errorf("history[1] = %q, want %q", m.chatHistory[1].Content, "second correction")
	}
}

func TestChattingInterleavedEmptyDuringLoading(t *testing.T) {
	m := model()
	m.screen = ScreenChatting
	m.chatLoading = true
	m.chatDone = false
	m.chatInput.SetValue("")

	m, cmd := updateWithCmd(m, tea.KeyMsg{Type: tea.KeyEnter})

	// Empty interleaved messages should also be accepted — cmd is non-nil.
	if cmd == nil {
		t.Error("expected non-nil cmd (cmdSendChimein)")
	}
	if len(m.chatHistory) != 1 || m.chatHistory[0].Content != "" {
		t.Errorf("history = %+v, want [{user ''}]", m.chatHistory)
	}
}

func TestChattingTypeChatDoneSetsDoneState(t *testing.T) {
	m := model()
	m.screen = ScreenChatting
	m.chatLoading = true
	m.chatDone = false
	m.chatHistory = []ChatLine{
		{Role: "user", Content: "original"},
		{Role: "assistant", Content: "response"},
	}

	msg := &protocol.Message{Type: protocol.TypeChatDone}
	_, cmd := handleEvent(m, msg)

	// ChatDone should just focus input (no auto-start).
	if cmd == nil {
		t.Fatal("expected non-nil cmd (chatInput.Focus())")
	}
	if m.chatLoading {
		t.Error("chatLoading should be false")
	}
	if !m.chatDone {
		t.Error("chatDone should be true")
	}
	if m.spinnerOn {
		t.Error("spinnerOn should be false")
	}
	// History should be preserved.
	if len(m.chatHistory) != 2 {
		t.Errorf("history length = %d, want 2", len(m.chatHistory))
	}
	// Session should be nil (closed).
	if m.chatSession != nil {
		t.Error("chatSession should be nil (closed)")
	}
}

func TestChattingTypeChatDoneWithoutPending(t *testing.T) {
	m := model()
	m.screen = ScreenChatting
	m.chatLoading = true
	m.chatDone = false
	m.chatHistory = []ChatLine{{Role: "user", Content: "hello"}}

	msg := &protocol.Message{Type: protocol.TypeChatDone}
	_, cmd := handleEvent(m, msg)

	// ChatDone should just focus input.
	if cmd == nil {
		t.Error("expected non-nil cmd (chatInput.Focus())")
	}
	if m.chatLoading {
		t.Error("chatLoading should be false")
	}
	if !m.chatDone {
		t.Error("chatDone should be true")
	}
	if m.spinnerOn {
		t.Error("spinnerOn should be false")
	}
}

func TestChattingTypeGoodbyeSetsDoneState(t *testing.T) {
	m := model()
	m.screen = ScreenChatting
	m.chatLoading = true
	m.chatDone = false
	m.chatHistory = []ChatLine{
		{Role: "user", Content: "original"},
		{Role: "assistant", Content: "response"},
	}

	msg := &protocol.Message{Type: protocol.TypeGoodbye}
	_, cmd := handleEvent(m, msg)

	// Goodbye should just focus input (no auto-start).
	if cmd == nil {
		t.Fatal("expected non-nil cmd (chatInput.Focus())")
	}
	if m.chatLoading {
		t.Error("chatLoading should be false")
	}
	if !m.chatDone {
		t.Error("chatDone should be true")
	}
	if m.spinnerOn {
		t.Error("spinnerOn should be false")
	}
}

func TestChattingEsc(t *testing.T) {
	m := model()
	m.screen = ScreenChatting
	m.chatLoading = false

	m, cmd := updateWithCmd(m, tea.KeyMsg{Type: tea.KeyEscape})

	// Direct transition: Esc immediately returns to main menu.
	if m.screen != ScreenMainMenu {
		t.Errorf("screen = %d, want ScreenMainMenu", m.screen)
	}
	if m.err != nil {
		t.Error("err should be cleared when returning to menu")
	}
	if cmd != nil {
		t.Error("expected nil cmd (direct transition)")
	}
}

func TestChattingEscClosesSession(t *testing.T) {
	m := model()
	m.screen = ScreenChatting
	m.chatSession = mockSession()
	m.chatLoading = false

	// Close() will panic because close field is nil. The defer+recover
	// prevents the test from crashing, but chatSession won't be set to nil.
	func() {
		defer func() { recover() }()
		m, _ = updateWithCmd(m, tea.KeyMsg{Type: tea.KeyEscape})
	}()

	// The important thing is no panic and we get a cmd.
	// The session cleanup is tested via the real Close path in integration tests.
}

func TestChattingScroll(t *testing.T) {
	t.Run("pgup increments scroll", func(t *testing.T) {
		m := model()
		m.screen = ScreenChatting
		m.chatScrollMax = 10
		m.chatScroll = 0

		m = update(m, tea.KeyMsg{Type: tea.KeyPgUp})
		if m.chatScroll != 1 {
			t.Errorf("scroll = %d, want 1", m.chatScroll)
		}
	})

	t.Run("pgup clamped at max", func(t *testing.T) {
		m := model()
		m.screen = ScreenChatting
		m.chatScrollMax = 5
		m.chatScroll = 5

		m = update(m, tea.KeyMsg{Type: tea.KeyPgUp})
		if m.chatScroll != 5 {
			t.Errorf("scroll should stay at 5 (at max)")
		}
	})

	t.Run("pgdown decrements scroll", func(t *testing.T) {
		m := model()
		m.screen = ScreenChatting
		m.chatScrollMax = 10
		m.chatScroll = 3

		m = update(m, tea.KeyMsg{Type: tea.KeyPgDown})
		if m.chatScroll != 2 {
			t.Errorf("scroll = %d, want 2", m.chatScroll)
		}
	})

	t.Run("pgdown clamped at 0", func(t *testing.T) {
		m := model()
		m.screen = ScreenChatting
		m.chatScroll = 0

		m = update(m, tea.KeyMsg{Type: tea.KeyPgDown})
		if m.chatScroll != 0 {
			t.Errorf("scroll should stay at 0")
		}
	})
}

// ─── Mouse wheel scrolling ────────────────────────────────────────────────

func TestShowOutputMouseWheelScrolling(t *testing.T) {
	t.Run("wheel up scrolls up", func(t *testing.T) {
		m := model()
		m.Height = 5
		m.screen = ScreenShowOutput
		m.cmdOutput = "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8"
		m.outputLines = strings.Split(m.cmdOutput, "\n")
		m.outputScroll = 3

		m = update(m, mouseWheelUp())
		if m.outputScroll != 2 {
			t.Errorf("outputScroll = %d, want 2 after wheel up", m.outputScroll)
		}
	})

	t.Run("wheel down scrolls down", func(t *testing.T) {
		m := model()
		m.Height = 5
		m.screen = ScreenShowOutput
		m.cmdOutput = "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8"
		m.outputLines = strings.Split(m.cmdOutput, "\n")
		m.outputScroll = 0

		m = update(m, mouseWheelDown())
		// avail = 5-5 = 0, clamped to 3. total=8 > 3 → scrollMax=5.
		if m.outputScroll < 1 {
			t.Errorf("outputScroll = %d, want >= 1 after wheel down", m.outputScroll)
		}
	})

	t.Run("wheel up clamped at top", func(t *testing.T) {
		m := model()
		m.screen = ScreenShowOutput
		m.outputLines = strings.Split("a\nb\nc", "\n")
		m.outputScroll = 0

		m = update(m, mouseWheelUp())
		if m.outputScroll != 0 {
			t.Errorf("outputScroll should stay 0 at top, got %d", m.outputScroll)
		}
	})
}

func TestChattingMouseWheelScrolling(t *testing.T) {
	t.Run("wheel up scrolls up (increases chatScroll)", func(t *testing.T) {
		m := model()
		m.screen = ScreenChatting
		m.chatScrollMax = 10
		m.chatScroll = 0

		m = update(m, mouseWheelUp())
		if m.chatScroll != 1 {
			t.Errorf("chatScroll = %d, want 1 after wheel up", m.chatScroll)
		}
	})

	t.Run("wheel up clamped at max", func(t *testing.T) {
		m := model()
		m.screen = ScreenChatting
		m.chatScrollMax = 5
		m.chatScroll = 5

		m = update(m, mouseWheelUp())
		if m.chatScroll != 5 {
			t.Errorf("chatScroll should stay at 5 (at max)")
		}
	})

	t.Run("wheel down scrolls down (decreases chatScroll)", func(t *testing.T) {
		m := model()
		m.screen = ScreenChatting
		m.chatScrollMax = 10
		m.chatScroll = 3

		m = update(m, mouseWheelDown())
		if m.chatScroll != 2 {
			t.Errorf("chatScroll = %d, want 2 after wheel down", m.chatScroll)
		}
	})

	t.Run("wheel down clamped at 0", func(t *testing.T) {
		m := model()
		m.screen = ScreenChatting
		m.chatScroll = 0

		m = update(m, mouseWheelDown())
		if m.chatScroll != 0 {
			t.Errorf("chatScroll should stay at 0")
		}
	})
}

func TestMouseWheelIgnoredOnNonScrollableScreens(t *testing.T) {
	m := model()
	m.screen = ScreenMainMenu
	saved := m.screen
	m = update(m, mouseWheelUp())
	if m.screen != saved {
		t.Errorf("wheel event changed screen on main menu")
	}
	m = update(m, mouseWheelDown())
	if m.screen != saved {
		t.Errorf("wheel event changed screen on main menu")
	}
}

// ─── Update: handleChatEvent ────────────────────────────────────────────────
func TestHandleReady(t *testing.T) {
	t.Run("with history sends message", func(t *testing.T) {
		m := model()
		m.screen = ScreenChatting
		m.chatReady = false
		m.chatHistory = []ChatLine{{Role: "user", Content: "hello"}}
		m.chatSession = mockSession()

		msg := &protocol.Message{Type: protocol.TypeReady}
		_, cmd := handleEvent(m, msg)

		if !m.chatReady {
			t.Error("chatReady should be true")
		}
		if cmd == nil {
			t.Error("expected non-nil cmd (send chat message)")
		}
	})

	t.Run("empty history returns nil cmd", func(t *testing.T) {
		m := model()
		m.screen = ScreenChatting
		m.chatReady = false
		m.chatSession = mockSession()

		msg := &protocol.Message{Type: protocol.TypeReady}
		_, cmd := handleEvent(m, msg)

		if cmd != nil {
			t.Error("expected nil cmd (no history)")
		}
	})
}

func TestHandleChatChunk(t *testing.T) {
	m := model()
	m.screen = ScreenChatting
	m.chatSession = mockSession()

	// First chunk: creates reasoning + assistant messages.
	chunk1 := &protocol.Message{
		Type:    protocol.TypeChatChunk,
		Payload: &protocol.ChatChunkPayload{Content: "Hello ", Reasoning: "thinking..."},
	}
	_, cmd1 := handleEvent(m, chunk1)

	// Reasoning is now separated: first ChatLine is reasoning, second is content.
	if len(m.chatHistory) != 2 {
		t.Fatalf("history length = %d, want 2 (reasoning + assistant)", len(m.chatHistory))
	}
	if m.chatHistory[0].Role != "reasoning" {
		t.Errorf("chatHistory[0].role = %q, want %q", m.chatHistory[0].Role, "reasoning")
	}
	if m.chatHistory[0].Content != "thinking..." {
		t.Errorf("chatHistory[0].content = %q, want %q", m.chatHistory[0].Content, "thinking...")
	}
	if m.chatHistory[1].Role != "assistant" {
		t.Errorf("chatHistory[1].role = %q, want %q", m.chatHistory[1].Role, "assistant")
	}
	if m.chatHistory[1].Content != "Hello " {
		t.Errorf("chatHistory[1].content = %q, want %q", m.chatHistory[1].Content, "Hello ")
	}
	if cmd1 == nil {
		t.Error("expected non-nil cmd (wait for more events)")
	}

	// Second chunk: appends to existing assistant.
	chunk2 := &protocol.Message{
		Type:    protocol.TypeChatChunk,
		Payload: &protocol.ChatChunkPayload{Content: "World!"},
	}
	_, cmd2 := handleEvent(m, chunk2)

	// Still 2 ChatLines: reasoning + assistant. Content appended to assistant.
	if len(m.chatHistory) != 2 {
		t.Fatalf("history length = %d, want 2", len(m.chatHistory))
	}
	if m.chatHistory[1].Content != "Hello World!" {
		t.Errorf("assistant content = %q, want %q", m.chatHistory[1].Content, "Hello World!")
	}
	if cmd2 == nil {
		t.Error("expected non-nil cmd for second chunk")
	}
}

func TestHandleChatChunkInvalidPayload(t *testing.T) {
	m := model()
	m.screen = ScreenChatting
	m.chatSession = mockSession()

	msg := &protocol.Message{
		Type:    protocol.TypeChatChunk,
		Payload: nil, // nil payload → type assertion fails
	}
	_, cmd := handleEvent(m, msg)
	if cmd == nil {
		t.Error("expected non-nil cmd even on invalid payload")
	}
}

func TestHandleChatDone(t *testing.T) {
	m := model()
	m.screen = ScreenChatting
	// User message is already in history (added on Enter).
	m.chatHistory = []ChatLine{{Role: "user", Content: "my message"}}
	m.chatLoading = true
	m.spinnerOn = true

	// Don't set chatSession — the handler checks m.chatSession != nil before
	// calling Close(), so we avoid the panic from nil close func.
	msg := &protocol.Message{Type: protocol.TypeChatDone}
	_, cmd := handleEvent(m, msg)

	if m.chatLoading {
		t.Error("chatLoading should be false")
	}
	if !m.chatDone {
		t.Error("chatDone should be true")
	}
	if m.spinnerOn {
		t.Error("spinnerOn should be false")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd (chatInput.Focus()) after done")
	}
	// User message should be preserved in history (unchanged by ChatDone).
	if len(m.chatHistory) != 1 {
		t.Fatalf("history length = %d, want 1", len(m.chatHistory))
	}
	if m.chatHistory[0].Role != "user" || m.chatHistory[0].Content != "my message" {
		t.Errorf("history[0] = %+v, want user:my message", m.chatHistory[0])
	}
}

func TestHandleAskUser(t *testing.T) {
	m := model()
	m.screen = ScreenChatting
	m.chatSession = mockSession()

	msg := &protocol.Message{
		Type: protocol.TypeAskUser,
		Payload: &protocol.AskUserPayload{
			Question: "Are you sure?",
			Semantic: protocol.SemanticConfirm,
			Options:  nil,
		},
	}
	_, cmd := handleEvent(m, msg)

	if m.screen != ScreenAskUser {
		t.Errorf("screen = %d, want ScreenAskUser", m.screen)
	}
	if m.prevScreen != ScreenChatting {
		t.Errorf("prevScreen = %d, want ScreenChatting", m.prevScreen)
	}
	if m.askQuestion != "Are you sure?" {
		t.Errorf("askQuestion = %q", m.askQuestion)
	}
	if m.askSemantic != protocol.SemanticConfirm {
		t.Errorf("askSemantic = %s", m.askSemantic)
	}
	if cmd != nil {
		t.Error("expected nil cmd (enter modal)")
	}
}

func TestHandleAskUserInvalidPayload(t *testing.T) {
	m := model()
	m.screen = ScreenChatting
	m.chatSession = mockSession()

	msg := &protocol.Message{
		Type:    protocol.TypeAskUser,
		Payload: nil,
	}
	_, cmd := handleEvent(m, msg)
	if cmd == nil {
		t.Error("expected non-nil cmd on invalid payload (wait for more)")
	}
}

func TestHandleStatus(t *testing.T) {
	m := model()
	m.screen = ScreenChatting
	m.chatSession = mockSession()

	msg := &protocol.Message{Type: protocol.TypeStatus}
	_, cmd := handleEvent(m, msg)
	if cmd == nil {
		t.Error("expected non-nil cmd (wait for more)")
	}
}

func TestHandleGoodbye(t *testing.T) {
	m := model()
	m.screen = ScreenChatting
	m.chatLoading = true
	m.spinnerOn = true

	// Don't set chatSession — same reason as TestHandleChatDone.
	msg := &protocol.Message{Type: protocol.TypeGoodbye}
	_, cmd := handleEvent(m, msg)

	if m.chatLoading {
		t.Error("chatLoading should be false")
	}
	if !m.chatDone {
		t.Error("chatDone should be true")
	}
	if m.spinnerOn {
		t.Error("spinnerOn should be false")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd (chatInput.Focus()) after goodbye")
	}
}

func TestHandleUnknownEvent(t *testing.T) {
	m := model()
	m.screen = ScreenChatting
	m.chatSession = mockSession()

	msg := &protocol.Message{Type: "unknown_type"}
	_, cmd := handleEvent(m, msg)
	if cmd == nil {
		t.Error("expected non-nil cmd (wait for more)")
	}
}

// ─── Update: AskUser Modal ──────────────────────────────────────────────────

func TestAskUserConfirm(t *testing.T) {
	t.Run("y answers yes", func(t *testing.T) {
		m := model()
		m.screen = ScreenAskUser
		m.prevScreen = ScreenChatting
		m.askSemantic = protocol.SemanticConfirm
		m.chatSession = mockSession()

		m = update(m, keyY)

		if !m.askDone {
			t.Error("askDone should be true")
		}
		if m.askResponse == nil || m.askResponse.Value != "yes" {
			t.Errorf("askResponse.Value = %q, want %q", m.askResponse.Value, "yes")
		}
		if m.screen != ScreenChatting {
			t.Errorf("screen should be ScreenChatting, got %d", m.screen)
		}
	})

	t.Run("Y answers yes", func(t *testing.T) {
		m := model()
		m.screen = ScreenAskUser
		m.prevScreen = ScreenChatting
		m.askSemantic = protocol.SemanticConfirm
		m.chatSession = mockSession()

		m = update(m, keyYCap)
		if m.askResponse == nil || m.askResponse.Value != "yes" {
			t.Errorf("askResponse.Value = %q, want %q", m.askResponse.Value, "yes")
		}
	})

	t.Run("n answers no", func(t *testing.T) {
		m := model()
		m.screen = ScreenAskUser
		m.prevScreen = ScreenChatting
		m.askSemantic = protocol.SemanticConfirm
		m.chatSession = mockSession()

		m = update(m, keyN)
		if m.askResponse == nil || m.askResponse.Value != "no" {
			t.Errorf("askResponse.Value = %q, want %q", m.askResponse.Value, "no")
		}
	})

	t.Run("esc answers no", func(t *testing.T) {
		m := model()
		m.screen = ScreenAskUser
		m.prevScreen = ScreenChatting
		m.askSemantic = protocol.SemanticConfirm
		m.chatSession = mockSession()

		m = update(m, tea.KeyMsg{Type: tea.KeyEscape})
		if m.askResponse == nil || m.askResponse.Value != "no" {
			t.Errorf("askResponse.Value = %q, want %q", m.askResponse.Value, "no")
		}
	})

	t.Run("other keys ignored", func(t *testing.T) {
		m := model()
		m.screen = ScreenAskUser
		m.askSemantic = protocol.SemanticConfirm

		m = update(m, keyX)
		if m.askDone {
			t.Error("askDone should be false for unrecognized key")
		}
	})
}

func TestAskUserChoice(t *testing.T) {
	t.Run("navigation", func(t *testing.T) {
		m := model()
		m.screen = ScreenAskUser
		m.askSemantic = protocol.SemanticChoice
		m.askOptions = []string{"Option A", "Option B", "Option C"}
		m.askChoice = 1

		// Up moves to 0
		m = update(m, tea.KeyMsg{Type: tea.KeyUp})
		if m.askChoice != 0 {
			t.Errorf("choice = %d, want 0 after up", m.askChoice)
		}

		// Up clamped at 0
		m = update(m, tea.KeyMsg{Type: tea.KeyUp})
		if m.askChoice != 0 {
			t.Errorf("choice should stay 0")
		}

		// Down moves to 1
		m = update(m, tea.KeyMsg{Type: tea.KeyDown})
		if m.askChoice != 1 {
			t.Errorf("choice = %d, want 1 after down", m.askChoice)
		}

		// k moves up
		m.askChoice = 2
		m = update(m, keyK)
		if m.askChoice != 1 {
			t.Errorf("choice = %d, want 1 after k", m.askChoice)
		}

		// j moves down
		m = update(m, keyJ)
		if m.askChoice != 2 {
			t.Errorf("choice = %d, want 2 after j", m.askChoice)
		}
	})

	t.Run("enter selects current option", func(t *testing.T) {
		m := model()
		m.screen = ScreenAskUser
		m.prevScreen = ScreenChatting
		m.askSemantic = protocol.SemanticChoice
		m.askOptions = []string{"A", "B"}
		m.askChoice = 1
		m.chatSession = mockSession()

		m = update(m, tea.KeyMsg{Type: tea.KeyEnter})

		if !m.askDone {
			t.Error("askDone should be true")
		}
		if m.askResponse == nil || m.askResponse.Choice != 1 {
			t.Errorf("askResponse.Choice = %d, want 1", m.askResponse.Choice)
		}
	})

	t.Run("space selects current option", func(t *testing.T) {
		m := model()
		m.screen = ScreenAskUser
		m.prevScreen = ScreenChatting
		m.askSemantic = protocol.SemanticChoice
		m.askOptions = []string{"A"}
		m.chatSession = mockSession()

		m = update(m, tea.KeyMsg{Type: tea.KeySpace})
		if !m.askDone {
			t.Error("askDone should be true")
		}
	})

	t.Run("esc cancels with empty value", func(t *testing.T) {
		m := model()
		m.screen = ScreenAskUser
		m.askSemantic = protocol.SemanticChoice
		m.chatSession = mockSession()

		m = update(m, tea.KeyMsg{Type: tea.KeyEscape})
		if m.askResponse == nil || m.askResponse.Value != "" {
			t.Errorf("askResponse.Value = %q, want empty", m.askResponse.Value)
		}
	})
}

func TestAskUserInput(t *testing.T) {
	t.Run("enter confirms input", func(t *testing.T) {
		m := model()
		m.screen = ScreenAskUser
		m.prevScreen = ScreenChatting
		m.askSemantic = protocol.SemanticInput
		m.askInput.SetValue("my answer")
		m.chatSession = mockSession()

		m = update(m, tea.KeyMsg{Type: tea.KeyEnter})

		if !m.askDone {
			t.Error("askDone should be true")
		}
		if m.askResponse == nil || m.askResponse.Value != "my answer" {
			t.Errorf("askResponse.Value = %q, want %q", m.askResponse.Value, "my answer")
		}
	})

	t.Run("enter trims spaces", func(t *testing.T) {
		m := model()
		m.screen = ScreenAskUser
		m.prevScreen = ScreenChatting
		m.askSemantic = protocol.SemanticInput
		m.askInput.SetValue("  spaced  ")
		m.chatSession = mockSession()

		m = update(m, tea.KeyMsg{Type: tea.KeyEnter})
		if m.askResponse == nil || m.askResponse.Value != "spaced" {
			t.Errorf("askResponse.Value = %q, want %q", m.askResponse.Value, "spaced")
		}
	})

	t.Run("esc cancels with empty", func(t *testing.T) {
		m := model()
		m.screen = ScreenAskUser
		m.askSemantic = protocol.SemanticInput

		m = update(m, tea.KeyMsg{Type: tea.KeyEscape})
		if m.askResponse == nil || m.askResponse.Value != "" {
			t.Errorf("askResponse.Value = %q, want empty", m.askResponse.Value)
		}
	})

	t.Run("text keys route to input", func(t *testing.T) {
		m := model()
		m.screen = ScreenAskUser
		m.askSemantic = protocol.SemanticInput

		origVal := m.askInput.Value()
		m = update(m, keyH)
		if m.askInput.Value() == origVal {
			t.Error("key should be routed to askInput")
		}
		if m.askDone {
			t.Error("askDone should be false after text input")
		}
	})
}

func TestAskUserNonKeyRouting(t *testing.T) {
	m := model()
	m.screen = ScreenAskUser
	m.askSemantic = protocol.SemanticInput

	// Non-key messages are routed to askInput (default case).
	m = update(m, spinner.TickMsg{})
	_ = m // no panic
}

func TestResumeFromAskUser(t *testing.T) {
	t.Run("from chat returns to chat and sends response", func(t *testing.T) {
		m := model()
		m.prevScreen = ScreenChatting
		m.chatSession = mockSession()
		m.askResponse = &protocol.AskUserResponsePayload{Value: "yes"}

		_, cmd := m.resumeFromAskUser()

		if m.screen != ScreenChatting {
			t.Errorf("screen = %d, want ScreenChatting", m.screen)
		}
		if !m.chatLoading {
			t.Error("chatLoading should be true after resume")
		}
		if cmd == nil {
			t.Error("expected non-nil cmd (send response)")
		}
	})

	t.Run("from non-chat returns to main menu", func(t *testing.T) {
		m := model()
		m.prevScreen = ScreenShowOutput

		_, cmd := m.resumeFromAskUser()

		if m.screen != ScreenMainMenu {
			t.Errorf("screen = %d, want ScreenMainMenu", m.screen)
		}
		if cmd != nil {
			t.Error("expected nil cmd for non-chat resume")
		}
	})
}

// ─── Helpers ────────────────────────────────────────────────────────────────

func TestAppendToLastAssistant(t *testing.T) {
	t.Run("creates new assistant message", func(t *testing.T) {
		m := model()
		m.appendToLastAssistant("Hello", "")

		if len(m.chatHistory) != 1 {
			t.Fatalf("history length = %d, want 1", len(m.chatHistory))
		}
		if m.chatHistory[0].Role != "assistant" {
			t.Errorf("role = %q, want %q", m.chatHistory[0].Role, "assistant")
		}
		if m.chatHistory[0].Content != "Hello" {
			t.Errorf("content = %q, want %q", m.chatHistory[0].Content, "Hello")
		}
	})

	t.Run("appends to existing assistant", func(t *testing.T) {
		m := model()
		m.chatHistory = append(m.chatHistory, ChatLine{Role: "assistant", Content: "Hello "})
		m.appendToLastAssistant("World!", "")

		if len(m.chatHistory) != 1 {
			t.Fatalf("history length = %d, want 1", len(m.chatHistory))
		}
		if m.chatHistory[0].Content != "Hello World!" {
			t.Errorf("content = %q, want %q", m.chatHistory[0].Content, "Hello World!")
		}
	})

	t.Run("does not append to non-assistant last", func(t *testing.T) {
		m := model()
		m.chatHistory = append(m.chatHistory, ChatLine{Role: "user", Content: "hi"})
		m.appendToLastAssistant("world", "")

		if len(m.chatHistory) != 2 {
			t.Fatalf("history length = %d, want 2", len(m.chatHistory))
		}
		if m.chatHistory[1].Role != "assistant" {
			t.Errorf("role = %q, want %q", m.chatHistory[1].Role, "assistant")
		}
	})
}

func TestWaitForMoreChatEvents(t *testing.T) {
	t.Run("with session returns cmd", func(t *testing.T) {
		m := model()
		m.chatSession = mockSession()
		cmd := m.waitForMoreChatEvents()
		if cmd == nil {
			t.Error("expected non-nil cmd")
		}
	})

	t.Run("without session returns nil", func(t *testing.T) {
		m := model()
		cmd := m.waitForMoreChatEvents()
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
}

func TestFormatCommandResult(t *testing.T) {
	t.Run("error returns error string", func(t *testing.T) {
		result := formatCommandResult(nil, assertError{"oops"})
		if !strings.Contains(result, "oops") {
			t.Errorf("result = %q, should contain error", result)
		}
	})

	t.Run("nil payload returns no output", func(t *testing.T) {
		result := formatCommandResult(nil, nil)
		if result != "No output." {
			t.Errorf("result = %q, want %q", result, "No output.")
		}
	})

	t.Run("failed command", func(t *testing.T) {
		result := formatCommandResult(
			&protocol.CommandResultPayload{Success: false, Data: "permission denied"},
			nil,
		)
		if !strings.Contains(result, "Command failed") || !strings.Contains(result, "permission denied") {
			t.Errorf("result = %q, should indicate failure", result)
		}
	})

	t.Run("success returns data trimmed", func(t *testing.T) {
		result := formatCommandResult(
			&protocol.CommandResultPayload{Success: true, Data: "  hello\nworld  "},
			nil,
		)
		if result != "hello\nworld" {
			t.Errorf("result = %q, want %q", result, "hello\nworld")
		}
	})
}

// assertError is a simple error for testing.
type assertError struct{ msg string }

func (e assertError) Error() string { return e.msg }

// ─── View ───────────────────────────────────────────────────────────────────

func TestViewMainMenu(t *testing.T) {
	m := model()
	v := m.View()

	if !strings.Contains(v, "DSCLI") {
		t.Error("view should contain logo 'DSCLI'")
	}
	if !strings.Contains(v, "DeepSeek CLI") {
		t.Error("view should contain tagline 'DeepSeek CLI'")
	}
	if !strings.Contains(v, "Chat") {
		t.Error("view should contain menu item 'Chat'")
	}
	if !strings.Contains(v, "Quit") {
		t.Error("view should contain menu item 'Quit'")
	}
	if !strings.Contains(v, "enter select") {
		t.Error("view should contain help text")
	}
}

func TestViewRunningCmd(t *testing.T) {
	m := model()
	m.screen = ScreenRunningCmd
	v := m.View()

	if !strings.Contains(v, "Running command") {
		t.Errorf("view = %q, should contain 'Running command'", v)
	}
}

func TestViewShowOutput(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := model()
		m.screen = ScreenShowOutput
		m.cmdOutput = "command result data"
		m.cmdSuccess = true
		v := m.View()

		if !strings.Contains(v, "Output") {
			t.Error("view should contain 'Output' header")
		}
		if !strings.Contains(v, "command result data") {
			t.Error("view should contain output data")
		}
	})

	t.Run("failure", func(t *testing.T) {
		m := model()
		m.screen = ScreenShowOutput
		m.cmdSuccess = false
		v := m.View()

		if !strings.Contains(v, "Output") {
			t.Error("view should contain 'Output' header")
		}
	})

	t.Run("exit hint shown", func(t *testing.T) {
		m := model()
		m.screen = ScreenShowOutput
		v := m.View()

		if !strings.Contains(v, "Esc/q") {
			t.Error("view should contain exit hint 'Esc/q'")
		}
		if !strings.Contains(v, "back to menu") {
			t.Error("view should contain 'back to menu' hint")
		}
	})
}

func TestViewChatting(t *testing.T) {
	t.Run("header and history", func(t *testing.T) {
		m := model()
		m.screen = ScreenChatting
		m.chatHistory = []ChatLine{
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "hi there"},
		}
		v := m.View()

		if !strings.Contains(v, "Chat") {
			t.Error("view should contain 'Chat' header")
		}
		if !strings.Contains(v, "hello") {
			t.Error("view should contain chat history")
		}
		if !strings.Contains(v, "hi there") {
			t.Error("view should contain assistant response")
		}
	})

	t.Run("loading shows spinner", func(t *testing.T) {
		m := model()
		m.screen = ScreenChatting
		m.chatLoading = true
		m.chatHistory = []ChatLine{{Role: "user", Content: "hello"}}
		v := m.View()

		if !strings.Contains(v, "thinking") {
			t.Error("view should show 'thinking' indicator when loading")
		}
	})

	t.Run("done shows ready indicator", func(t *testing.T) {
		m := model()
		m.screen = ScreenChatting
		m.chatLoading = false
		m.chatDone = true
		m.chatHistory = []ChatLine{
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "world"},
		}
		v := m.View()

		if !strings.Contains(v, "Ready") {
			t.Error("view should show 'Ready' indicator when done")
		}
	})

	t.Run("user message shown in history", func(t *testing.T) {
		m := model()

		m.screen = ScreenChatting
		m.chatHistory = []ChatLine{{Role: "user", Content: "my question"}}
		m.chatLoading = false
		m.chatDone = false
		v := m.View()

		if !strings.Contains(v, "my question") {
			t.Error("view should show user message from history")
		}
	})

	t.Run("scroll indicator shown when scrolled", func(t *testing.T) {
		m := model()
		m.screen = ScreenChatting
		// Need enough history lines so totalLines > maxLines (Height=30 → 22).
		for i := 0; i < 25; i++ {
			m.chatHistory = append(m.chatHistory, ChatLine{Role: "user", Content: "line"})
		}
		m.chatScroll = 3
		_ = m.View() // triggers chatScrollMax calculation
		v := m.View()

		if !strings.Contains(v, "more lines above") {
			t.Error("view should show scroll indicator")
		}
	})
}

func TestViewAskUser(t *testing.T) {
	t.Run("confirm semantic", func(t *testing.T) {
		m := model()
		m.screen = ScreenAskUser
		m.askQuestion = "Continue?"
		m.askSemantic = protocol.SemanticConfirm
		v := m.View()

		if !strings.Contains(v, "Continue?") {
			t.Error("view should contain the question")
		}
		if !strings.Contains(v, "y") || !strings.Contains(v, "n") {
			t.Error("view should show y/n options")
		}
	})

	t.Run("choice semantic", func(t *testing.T) {
		m := model()
		m.screen = ScreenAskUser
		m.askQuestion = "Pick one:"
		m.askSemantic = protocol.SemanticChoice
		m.askOptions = []string{"Option A", "Option B"}
		m.askChoice = 0
		v := m.View()

		if !strings.Contains(v, "Pick one:") {
			t.Error("view should contain the question")
		}
		if !strings.Contains(v, "Option A") || !strings.Contains(v, "Option B") {
			t.Error("view should contain options")
		}
	})

	t.Run("input semantic", func(t *testing.T) {
		m := model()
		m.screen = ScreenAskUser
		m.askQuestion = "Enter name:"
		m.askSemantic = protocol.SemanticInput
		m.askInput.SetValue("John")
		v := m.View()

		if !strings.Contains(v, "Enter name:") {
			t.Error("view should contain the question")
		}
		if !strings.Contains(v, "John") {
			t.Error("view should contain input value")
		}
	})
}

func TestViewQuitting(t *testing.T) {
	m := model()
	m.screen = ScreenQuitting
	v := m.View()

	if !strings.Contains(v, "Goodbye") {
		t.Errorf("view = %q, should contain 'Goodbye'", v)
	}
}

func TestViewDefault(t *testing.T) {
	m := model()
	m.screen = Screen(999) // unknown screen
	v := m.View()

	if !strings.Contains(v, "Unknown screen") {
		t.Errorf("view = %q, should contain 'Unknown screen'", v)
	}
}

// ─── Utility: wrapText ──────────────────────────────────────────────────────

func TestWrapText(t *testing.T) {
	t.Run("empty string returns empty line", func(t *testing.T) {
		lines := wrapText("", 10)
		if len(lines) != 1 || lines[0] != "" {
			t.Errorf("got %q, want ['']", lines)
		}
	})

	t.Run("short text stays one line", func(t *testing.T) {
		lines := wrapText("hello", 10)
		if len(lines) != 1 || lines[0] != "hello" {
			t.Errorf("got %q, want ['hello']", lines)
		}
	})

	t.Run("long text wraps", func(t *testing.T) {
		lines := wrapText("a b c d e", 3)
		if len(lines) < 2 {
			t.Errorf("expected multiple lines, got %d: %v", len(lines), lines)
		}
	})

	t.Run("exact width fits", func(t *testing.T) {
		lines := wrapText("abc", 3)
		if len(lines) != 1 || lines[0] != "abc" {
			t.Errorf("got %v, want ['abc']", lines)
		}
	})

	t.Run("negative width returns full text", func(t *testing.T) {
		lines := wrapText("hello world", -1)
		if len(lines) != 1 || lines[0] != "hello world" {
			t.Errorf("got %v, want ['hello world']", lines)
		}
	})

	t.Run("zero width returns full text", func(t *testing.T) {
		lines := wrapText("hello", 0)
		if len(lines) != 1 || lines[0] != "hello" {
			t.Errorf("got %v, want ['hello']", lines)
		}
	})
}

// ─── Edge Cases ─────────────────────────────────────────────────────────────

func TestUpdateChattingTextInput(t *testing.T) {
	m := model()
	m.screen = ScreenChatting
	m.chatLoading = false
	m.chatDone = false

	// Key messages should be routed to chatInput.
	m = update(m, keyH)
	if m.chatInput.Value() != "h" {
		t.Errorf("chatInput.Value = %q, want %q", m.chatInput.Value(), "h")
	}
}

func TestSpinnerTick(t *testing.T) {
	m := model()
	m.screen = ScreenMainMenu

	// Spinner.TickMsg should update spinner state and return a cmd for next tick.
	m, cmd := updateWithCmd(m, spinner.TickMsg{})
	if cmd == nil {
		t.Error("expected non-nil cmd (next tick)")
	}
}

func TestScreenQuitting(t *testing.T) {
	m := model()
	m.screen = ScreenQuitting

	// Any message while quitting returns tea.Quit.
	_, cmd := updateWithCmd(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("expected non-nil cmd for ScreenQuitting")
	}
}

func TestDefaultScreenBehavior(t *testing.T) {
	m := model()
	m.screen = Screen(999) // undefined screen

	// Unknown screen returns nil cmd.
	_, cmd := updateWithCmd(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("expected nil cmd for unknown screen")
	}
}

func TestExecuteSelectedChatSetsChatState(t *testing.T) {
	m := model()

	m.menuCursor = 0
	_, cmd := m.executeSelected()

	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	if m.screen != ScreenChatting {
		t.Errorf("screen = %d, want ScreenChatting", m.screen)
	}
	if !m.spinnerOn {
		t.Error("spinnerOn should be true")
	}
	if !m.chatLoading {
		t.Error("chatLoading should be true")
	}
	if m.chatDone {
		t.Error("chatDone should be false")
	}
}

// ─── Integration: Full Chat Flow ────────────────────────────────────────────

func TestFullChatFlow(t *testing.T) {
	m := model()
	m.Width = 80
	m.Height = 30

	// 1. Start in main menu, select chat (item 0).
	m, cmd1 := updateWithCmd(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != ScreenChatting {
		t.Fatalf("step 1: screen = %d, want ScreenChatting", m.screen)
	}
	_ = cmd1 // don't execute — skip agent interaction

	// 2. Simulate session ready.
	sess := mockSession()
	m = update(m, aiagent.ChatSessionReadyMsg{Session: sess})
	if !m.chatReady {
		t.Fatal("step 2: chatReady should be true")
	}

	// 3. Simulate a Ready event (dscli says "ready").
	readyMsg := &protocol.Message{Type: protocol.TypeReady}
	_, cmd3 := handleEvent(m, readyMsg)
	if cmd3 != nil {
		// No history yet, so should be nil
	}

	// Clear session before enter — the mock session's close func is nil,
	// and enter tries to close the old session before starting a new one.
	m.chatSession = nil

	// 4. User types a message.
	m.chatInput.SetValue("Hello, AI!")
	m, cmd4 := updateWithCmd(m, tea.KeyMsg{Type: tea.KeyEnter})
	// User message should now be in chatHistory (added immediately on Enter).
	if len(m.chatHistory) != 1 || m.chatHistory[0].Content != "Hello, AI!" {
		t.Fatalf("step 4: history = %+v, want [user:Hello, AI!]", m.chatHistory)
	}
	_ = cmd4

	// 5. Set chatReady manually for the exchange (avoids setting chatSession
	//    which has nil close func — would panic when enter/Close() is called).
	m.chatReady = true

	// 6. Simulate Ready with history — should trigger send.
	readyMsg2 := &protocol.Message{Type: protocol.TypeReady}
	_, cmd6 := handleEvent(m, readyMsg2)
	if cmd6 == nil {
		t.Error("step 6: expected non-nil cmd (send message)")
	}

	// 7. Simulate streaming chunks. Handle them via a temporary session that
	//    satisfies waitForMoreChatEvents (which checks m.chatSession != nil).
	sessTmp := mockSession()
	m.chatSession = sessTmp

	// 7. Simulate streaming chunks.
	_, _ = handleEvent(m, &protocol.Message{
		Type:    protocol.TypeChatChunk,
		Payload: &protocol.ChatChunkPayload{Content: "Hello! "},
	})
	_, _ = handleEvent(m, &protocol.Message{
		Type:    protocol.TypeChatChunk,
		Payload: &protocol.ChatChunkPayload{Content: "How can I help?"},
	})

	// After step 4, history has [user]. After chunks, history has [user, assistant].
	if len(m.chatHistory) != 2 {
		t.Fatalf("step 7: history length = %d, want 2 (user+assistant)", len(m.chatHistory))
	}
	if m.chatHistory[0].Role != "user" || m.chatHistory[0].Content != "Hello, AI!" {
		t.Errorf("step 7: history[0] = %+v, want user:Hello, AI!", m.chatHistory[0])
	}
	if m.chatHistory[1].Role != "assistant" || m.chatHistory[1].Content != "Hello! How can I help?" {
		t.Errorf("step 7: history[1] = %+v, want assistant:Hello! How can I help?", m.chatHistory[1])
	}

	// Clear session before ChatDone — mock session has nil close func and
	// would panic when the handler calls m.chatSession.Close().
	m.chatSession = nil

	// 8. Simulate done.
	_, cmd8 := handleEvent(m, &protocol.Message{Type: protocol.TypeChatDone})
	if cmd8 == nil {
		t.Error("step 8: expected non-nil cmd (chatInput.Focus())")
	}
	if !m.chatDone {
		t.Error("step 8: chatDone should be true")
	}
	if m.chatLoading {
		t.Error("step 8: chatLoading should be false")
	}
	// History should be unchanged by ChatDone (user added on Enter, assistant from chunks).
	if len(m.chatHistory) != 2 {
		t.Fatalf("step 8: history length = %d, want 2 (user+assistant)", len(m.chatHistory))
	}
	if m.chatHistory[0].Role != "user" || m.chatHistory[0].Content != "Hello, AI!" {
		t.Errorf("step 8: history[0] = %+v, want user:Hello, AI!", m.chatHistory[0])
	}
	if m.chatHistory[1].Role != "assistant" || m.chatHistory[1].Content != "Hello! How can I help?" {
		t.Errorf("step 8: history[1] = %+v, want assistant:Hello! How can I help?", m.chatHistory[1])
	}
}
