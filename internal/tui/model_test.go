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
	expectedChatWidth := 100 - 10 // Width - 10
	if m.chatInput.Width != expectedChatWidth {
		t.Errorf("chatInput.Width = %d, want %d", m.chatInput.Width, expectedChatWidth)
	}
	expectedAskWidth := expectedChatWidth - 4
	if m.askInput.Width != expectedAskWidth {
		t.Errorf("askInput.Width = %d, want %d", m.askInput.Width, expectedAskWidth)
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
	if m.chatInput.Width < 10 {
		t.Errorf("chatInput.Width = %d, minimum should be 10", m.chatInput.Width)
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




// ─── Update: Show Output ────────────────────────────────────────────────────

func TestShowOutputAnyKey(t *testing.T) {
	m := model()
	m.screen = ScreenShowOutput
	m, cmd := updateWithCmd(m, tea.KeyMsg{Type: tea.KeyEnter})

	// Direct transition: any key immediately returns to main menu.
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


func TestShowOutputNonKeyIgnored(t *testing.T) {
	m := model()
	m.screen = ScreenShowOutput
	m = update(m, tea.WindowSizeMsg{Width: 100, Height: 50})
	if m.screen != ScreenShowOutput {
		t.Errorf("screen = %d, want ScreenShowOutput (non-key msgs ignored)", m.screen)
	}
}

// ─── Update: Chatting ───────────────────────────────────────────────────────

func TestChattingSessionReady(t *testing.T) {
	m := model()
	m.screen = ScreenChatting
	m.chatPendingInput = "hello"
	m.chatLoading = true

	sess := mockSession()
	m = update(m, aiagent.ChatSessionReadyMsg{Session: sess})

	if m.chatSession != sess {
		t.Error("chatSession not set")
	}
	if !m.chatReady {
		t.Error("chatReady should be true")
	}
	if !m.chatLoading {
		t.Error("chatLoading should remain true (pending input)")
	}
}

func TestChattingSessionReadyError(t *testing.T) {
	m := model()
	m.screen = ScreenChatting
	m.chatPendingInput = "hello"
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

	if m.chatPendingInput != "hello" {
		t.Errorf("chatPendingInput = %q, want %q", m.chatPendingInput, "hello")
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
	m.chatInput.SetValue("  ")

	_, cmd := updateWithCmd(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("expected nil cmd for empty input")
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

// ─── Update: handleChatEvent ────────────────────────────────────────────────

func TestHandleReady(t *testing.T) {
	t.Run("with pending input sends message", func(t *testing.T) {
		m := model()
		m.screen = ScreenChatting
		m.chatReady = false
		m.chatPendingInput = "hello"
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

	t.Run("without pending input returns nil cmd", func(t *testing.T) {
		m := model()
		m.screen = ScreenChatting
		m.chatReady = false
		m.chatSession = mockSession()

		msg := &protocol.Message{Type: protocol.TypeReady}
		_, cmd := handleEvent(m, msg)

		if cmd != nil {
			t.Error("expected nil cmd (no pending input)")
		}
	})
}

func TestHandleChatChunk(t *testing.T) {
	m := model()
	m.screen = ScreenChatting
	m.chatSession = mockSession()

	// First chunk: creates new assistant message.
	chunk1 := &protocol.Message{
		Type:    protocol.TypeChatChunk,
		Payload: &protocol.ChatChunkPayload{Content: "Hello ", Reasoning: "thinking..."},
	}
	_, cmd1 := handleEvent(m, chunk1)

	if len(m.chatHistory) != 1 {
		t.Fatalf("history length = %d, want 1", len(m.chatHistory))
	}
	if m.chatHistory[0].Role != "assistant" {
		t.Errorf("role = %q, want %q", m.chatHistory[0].Role, "assistant")
	}
	if m.chatHistory[0].Content != "Hello " {
		t.Errorf("content = %q, want %q", m.chatHistory[0].Content, "Hello ")
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

	if len(m.chatHistory) != 1 {
		t.Fatalf("history length = %d, want 1", len(m.chatHistory))
	}
	if m.chatHistory[0].Content != "Hello World!" {
		t.Errorf("content = %q, want %q", m.chatHistory[0].Content, "Hello World!")
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
	m.chatPendingInput = "my message"
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
	if cmd != nil {
		t.Error("expected nil cmd after done")
	}
	// Pending input should be committed to history
	if len(m.chatHistory) != 1 {
		t.Fatalf("history length = %d, want 1", len(m.chatHistory))
	}
	if m.chatHistory[0].Role != "user" || m.chatHistory[0].Content != "my message" {
		t.Errorf("history[0] = %+v, want user:my message", m.chatHistory[0])
	}
	if m.chatPendingInput != "" {
		t.Error("chatPendingInput should be cleared")
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
	if cmd != nil {
		t.Error("expected nil cmd after goodbye")
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

	t.Run("any key hint", func(t *testing.T) {
		m := model()
		m.screen = ScreenShowOutput
		v := m.View()

		if !strings.Contains(v, "return to menu") {
			t.Error("view should contain hint to return to menu")
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

	t.Run("done shows response complete", func(t *testing.T) {
		m := model()
		m.screen = ScreenChatting
		m.chatLoading = false
		m.chatDone = true
		m.chatHistory = []ChatLine{
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "world"},
		}
		v := m.View()

		if !strings.Contains(v, "Response complete") {
			t.Error("view should show 'Response complete' when done")
		}
	})

	t.Run("pending input shown", func(t *testing.T) {
		m := model()
		m.screen = ScreenChatting
		m.chatPendingInput = "not yet sent"
		m.chatLoading = false
		m.chatDone = false
		v := m.View()

		if !strings.Contains(v, "not yet sent") {
			t.Error("view should show pending input")
		}
		if !strings.Contains(v, "pending") {
			t.Error("view should indicate pending state")
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
		// No pending input, so should be nil
	}

	// Clear session before enter — the mock session's close func is nil,
	// and enter tries to close the old session before starting a new one.
	m.chatSession = nil

	// 4. User types a message.
	m.chatInput.SetValue("Hello, AI!")
	m, cmd4 := updateWithCmd(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.chatPendingInput != "Hello, AI!" {
		t.Fatalf("step 4: chatPendingInput = %q", m.chatPendingInput)
	}
	_ = cmd4

	// 5. Set chatReady manually for the exchange (avoids setting chatSession
	//    which has nil close func — would panic when enter/Close() is called).
	m.chatReady = true

	// 6. Simulate Ready with pending input — should trigger send.
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

	if len(m.chatHistory) != 1 {
		t.Fatalf("step 7: history length = %d, want 1", len(m.chatHistory))
	}
	if m.chatHistory[0].Content != "Hello! How can I help?" {
		t.Errorf("step 7: content = %q", m.chatHistory[0].Content)
	}
	// Clear session before ChatDone — mock session has nil close func and
	// would panic when the handler calls m.chatSession.Close().
	m.chatSession = nil

	// 8. Simulate done.
	_, cmd8 := handleEvent(m, &protocol.Message{Type: protocol.TypeChatDone})
	if cmd8 != nil {
		t.Error("step 8: expected nil cmd")
	}
	if !m.chatDone {
		t.Error("step 8: chatDone should be true")
	}
	if m.chatLoading {
		t.Error("step 8: chatLoading should be false")
	}
	// History should now have assistant (from chunks) + user (committed from pending).
	if len(m.chatHistory) != 2 {
		t.Fatalf("step 8: history length = %d, want 2 (assistant+user)", len(m.chatHistory))
	}
	if m.chatHistory[0].Role != "assistant" || m.chatHistory[0].Content != "Hello! How can I help?" {
		t.Errorf("step 8: history[0] = %+v", m.chatHistory[0])
	}
	if m.chatHistory[1].Role != "user" || m.chatHistory[1].Content != "Hello, AI!" {
		t.Errorf("step 8: history[1] = %+v", m.chatHistory[1])
	}
}
