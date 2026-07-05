// Package tui implements the Bubble Tea application for dscli.tui.
//
// Architecture: the main model (RootModel) is a finite state machine.
// Each Screen maps to a distinct view and behavior.  Transitions are
// caused by user input (keyboard) or asynchronous agent results.
package tui

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"gitcode.com/dscli/dscli.tui/internal/aiagent"
	"gitcode.com/dscli/dscli.tui/internal/tui/protocol"
)

// ─── Screen ────────────────────────────────────────────────────────

// Screen represents the current application screen.
type Screen int

const (
	// ScreenMainMenu shows the command palette. This is the initial state.
	ScreenMainMenu Screen = iota
	// ScreenRunningCmd is a transient state: a non-interactive command has
	// been dispatched and we are waiting for the result.
	ScreenRunningCmd
	// ScreenShowOutput displays the result of a non-interactive command.
	ScreenShowOutput
	// ScreenChatting is the interactive chat view.
	ScreenChatting
	// ScreenAskUser is a modal overlay: dscli has asked a question and the
	// user must answer before the conversation can continue.
	ScreenAskUser
	// ScreenQuitting performs graceful shutdown.
	ScreenQuitting
)

// ─── MenuItem ────────────────────────────────────────────────────────

// MenuItem describes one entry in the main command palette.
type MenuItem struct {
	Title string // short name, shown in bold
	Desc  string // one-line description
}

// ─── ChatLine ────────────────────────────────────────────────────────

// ChatLine is a single message in the chat history, stored client-side.
type ChatLine struct {
	Role    string // "user" | "assistant" | "reasoning" | "system"
	Content string
}

// ─── RootModel ───────────────────────────────────────────────────────

// RootModel is the top-level Bubble Tea model.
//
// State machine overview:
//
//	ScreenMainMenu ←── ScreenShowOutput (any key → back to menu)
//	    │  │
//	    │  └──(chat selected)──→ ScreenChatting
//	    │                           │
//	    │                  (ask_user received)
//	    │                           │
//	    │                           └──→ ScreenAskUser
//	    │                                   │
//	    │                          (user responds)
//	    │                                   │
//	    │                           └──→ ScreenChatting (resume)
//	    │
//	    └──(command selected)──→ ScreenRunningCmd
//	                                │
//	                        (result received)
//	                                │
//	                        └──→ ScreenShowOutput
type RootModel struct {
	// Core
	screen Screen
	agent  aiagent.AIAgent

	// Terminal dimensions (updated by tea.WindowSizeMsg, exported for styles)
	Width  int
	Height int

	// Global error (cleared after display)
	err error

	// ── Main menu ─────────────────────────────────────────────────
	menuItems  []MenuItem
	menuCursor int // currently highlighted item index

	// ── Command output ────────────────────────────────────────────
	cmdOutput  string // rendered output text
	cmdSuccess bool   // whether the command succeeded

	// ── Chat ──────────────────────────────────────────────────────
	chatHistory      []ChatLine           // accumulated conversation
	chatInput        textinput.Model      // chat message input
	chatLoading      bool                 // true while waiting for AI response
	chatSession      *aiagent.ChatSession // active session (one per exchange)
	chatDone         bool                 // true when the current exchange is done
	chatPendingInput string               // user message waiting to be sent to dscli
	chatScroll       int                  // 0 = bottom, >0 = lines scrolled up
	chatScrollMax    int                  // max scroll offset from last render

	// ── Spinner (loading animation) ────────────────────────────────
	spinner   spinner.Model
	spinnerOn bool // true when spinner should be rendered

	// ── AskUser modal ─────────────────────────────────────────────
	prevScreen  Screen                         // screen to restore after answering
	askQuestion string
	askSemantic protocol.Semantic
	askOptions  []string
	askInput    textinput.Model                // for SemanticInput
	askChoice   int                            // for SemanticChoice (0 = first option)
	askDone     bool                           // true after user has answered
	askResponse *protocol.AskUserResponsePayload

	// ── Internal flags ────────────────────────────────────────────
	chatReady bool // true after first ready event in current exchange
}

// ─── Menu item definitions ───────────────────────────────────────────

var defaultMenuItems = []MenuItem{
	{Title: "💬 Chat", Desc: "Interactive chat with AI assistant"},
	{Title: "📊 Balance", Desc: "Check your dscli account balance"},
	{Title: "🤖 Models", Desc: "List available AI models"},
	{Title: "ℹ️  Version", Desc: "Show dscli version information"},
	{Title: "🔍 Flycheck", Desc: "Run static analysis on a file or project"},
	{Title: "📝 History", Desc: "Manage dscli conversation history"},
	{Title: "🛠  Skill", Desc: "Manage dscli skills"},
	{Title: "💾 Memory", Desc: "Manage dscli persistent memory"},
	{Title: "📁 Project", Desc: "Manage dscli projects"},
	{Title: "👤 Role", Desc: "Manage AI roles"},
	{Title: "🧰 Tool", Desc: "Manage dscli tools"},
	{Title: "✉️  Mail", Desc: "Send and receive AI mail"},
	{Title: "🔧 Service", Desc: "Manage dscli services"},
	{Title: "🚪 Quit", Desc: "Exit dscli.tui"},
}

// ─── Constructor ─────────────────────────────────────────────────────

// New creates a new RootModel with the given agent.
func New(agent aiagent.AIAgent) *RootModel {
	chatInput := textinput.New()
	chatInput.Placeholder = "Type your message..."
	chatInput.Focus()
	chatInput.CharLimit = 0 // no limit
	chatInput.Width = 40   // placeholder, resized on WindowSizeMsg

	askInput := textinput.New()
	askInput.Placeholder = "Type your answer..."
	askInput.Focus()
	askInput.CharLimit = 0
	askInput.Width = 40

	sp := spinner.New()
	sp.Style = SpinnerStyle

	return &RootModel{
		screen:     ScreenMainMenu,
		agent:      agent,
		menuItems:  defaultMenuItems,
		menuCursor: 0,
		chatInput:  chatInput,
		askInput:   askInput,
		spinner:    sp,
	}
}

// ─── Tea.Model interface ────────────────────────────────────────────

// Init implements tea.Model.Init.
func (m *RootModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// ─── Helpers ─────────────────────────────────────────────────────────

// Agent returns the AIAgent reference.
func (m *RootModel) Agent() aiagent.AIAgent { return m.agent }

// SelectedMenuItem returns the currently highlighted menu item.
func (m *RootModel) SelectedMenuItem() *MenuItem {
	if m.menuCursor < 0 || m.menuCursor >= len(m.menuItems) {
		return nil
	}
	return &m.menuItems[m.menuCursor]
}
