// Package tui implements the Bubble Tea application for dscli.tui.
//
// Architecture: the main model (RootModel) is a finite state machine.
// Each AppState maps to a distinct view and behavior.  Transitions are
// caused by user input (keyboard) or asynchronous agent results.
package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"gitcode.com/dscli/dscli.tui/internal/aiagent"
	"gitcode.com/dscli/dscli.tui/internal/tui/protocol"
)

// ─── AppState ────────────────────────────────────────────────────────

// AppState represents the current application state.
type AppState int

const (
	// StateMainMenu shows the command palette. This is the initial state.
	StateMainMenu AppState = iota
	// StateRunningCmd is a transient state: a non-interactive command has
	// been dispatched and we are waiting for the result.
	StateRunningCmd
	// StateShowOutput displays the result of a non-interactive command.
	StateShowOutput
	// StateChatting is the interactive chat view.
	StateChatting
	// StateAskUser is a modal overlay: dscli has asked a question and the
	// user must answer before the conversation can continue.
	StateAskUser
	// StateQuitting performs graceful shutdown.
	StateQuitting
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
//	StateMainMenu ←── StateShowOutput (any key → back to menu)
//	    │  │
//	    │  └──(chat selected)──→ StateChatting
//	    │                           │
//	    │                  (ask_user received)
//	    │                           │
//	    │                           └──→ StateAskUser
//	    │                                   │
//	    │                          (user responds)
//	    │                                   │
//	    │                           └──→ StateChatting (resume)
//	    │
//	    └──(command selected)──→ StateRunningCmd
//	                                │
//	                        (result received)
//	                                │
//	                        └──→ StateShowOutput
type RootModel struct {
	// Core
	state AppState
	agent aiagent.AIAgent

	// Terminal dimensions (updated by tea.WindowSizeMsg)
	width  int
	height int

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
	chatInput        []rune               // current input buffer
	chatCursor       int                  // cursor position inside input buffer
	chatLoading      bool                 // true while waiting for AI response
	chatSession      *aiagent.ChatSession // active session (one per exchange)
	chatDone         bool                 // true when the current exchange is done
	chatPendingInput string               // user message waiting to be sent to dscli

	// ── AskUser modal ─────────────────────────────────────────────
	askPrevState  AppState        // state to restore after answering
	askQuestion   string
	askSemantic   protocol.Semantic
	askOptions    []string
	askInput      []rune   // for SemanticInput
	askCursor     int      // cursor inside askInput
	askChoice     int      // for SemanticChoice (-1 = unselected)
	askDone       bool     // true after user has answered
	askResponse   *protocol.AskUserResponsePayload

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
	return &RootModel{
		state:      StateMainMenu,
		agent:      agent,
		menuItems:  defaultMenuItems,
		menuCursor: 0,
	}
}

// ─── Tea.Model interface ────────────────────────────────────────────

// Init implements tea.Model.Init.
func (m *RootModel) Init() tea.Cmd { return nil }

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
