// Package aiagent defines the AIAgent interface — the high-level abstraction
// that decouples the TUI from dscli implementation details.
//
// Design principle: all inputs and outputs use Go types (not raw strings).
// The wire format (JSON-line over stdio) is an internal detail of the
// implementations in this package and in pkg/jsonline.
package aiagent

import (
	"context"

	"github.com/dscli/dscli.tui/internal/tui/protocol"
)

// ─── AIAgent ────────────────────────────────────────────────

// AIAgent abstracts a dscli backend.
// The TUI interacts only through this interface — it never imports dscli packages.
type AIAgent interface {
	// ── Non-interactive commands ─────────────────────────

	// Balance returns account balance information.
	// Internally calls: dscli --json-line balance [--format json]
	Balance(ctx context.Context, format string) (*protocol.CommandResultPayload, error)

	// Models lists available models.
	// Internally calls: dscli --json-line models [--format json] [--price]
	Models(ctx context.Context, format string, showPrice bool) (*protocol.CommandResultPayload, error)

	// Version returns dscli version info.
	// Internally calls: dscli --json-line version
	Version(ctx context.Context) (*protocol.CommandResultPayload, error)

	// Flycheck runs static analysis.
	// Internally calls: dscli --json-line flycheck [--emacs] <path>
	Flycheck(ctx context.Context, path string, emacs bool) (*protocol.CommandResultPayload, error)

	// FIM performs fill-in-the-middle.
	// Internally calls: dscli --json-line fim [...args]
	FIM(ctx context.Context, args ...string) (*protocol.CommandResultPayload, error)

	// MemorySearch searches memories by keyword.
	// Internally calls: dscli memory search <query>
	MemorySearch(ctx context.Context, query string) (*protocol.CommandResultPayload, error)

	// ── Subcommand groups ────────────────────────────────

	// History delegates to dscli history <subcmd> [args...].
	History(ctx context.Context, subcmd string, args ...string) (*protocol.CommandResultPayload, error)

	// Skill delegates to dscli skill <subcmd> [args...].
	Skill(ctx context.Context, subcmd string, args ...string) (*protocol.CommandResultPayload, error)

	// Prompt delegates to dscli prompt <subcmd> [args...].
	Prompt(ctx context.Context, subcmd string, args ...string) (*protocol.CommandResultPayload, error)

	// Memory delegates to dscli memory <subcmd> [args...].
	Memory(ctx context.Context, subcmd string, args ...string) (*protocol.CommandResultPayload, error)

	// Project delegates to dscli project <subcmd> [args...].
	Project(ctx context.Context, subcmd string, args ...string) (*protocol.CommandResultPayload, error)

	// Role delegates to dscli role <subcmd> [args...].
	Role(ctx context.Context, subcmd string, args ...string) (*protocol.CommandResultPayload, error)

	// Tool delegates to dscli tool <subcmd> [args...].
	Tool(ctx context.Context, subcmd string, args ...string) (*protocol.CommandResultPayload, error)

	// Mail delegates to dscli mail <subcmd> [args...].
	Mail(ctx context.Context, subcmd string, args ...string) (*protocol.CommandResultPayload, error)

	// Service delegates to dscli service <subcmd> [args...].
	Service(ctx context.Context, subcmd string, args ...string) (*protocol.CommandResultPayload, error)

	// ── Interactive Chat ─────────────────────────────────

	// NewChatSession creates an interactive chat session.
	// The session communicates with dscli via JSON-line over stdio.
	// Returns a ChatSession that the TUI uses to send/receive messages.
	NewChatSession(ctx context.Context, opts ChatSessionOptions) (*ChatSession, error)

	// SendChimein runs dscli chat in climein mode: writes the content to the
	// chimeins table for the primary process to pick up. Returns the combined
	// stdout+stderr output and any process error.
	// When the primary process has already released its lock, the new process
	// becomes primary instead — the returned output contains the AI response.
	SendChimein(ctx context.Context, content string) (string, error)

	// Close releases any resources held by the agent.
	Close() error
}

// ─── ChatSessionOptions ─────────────────────────────────────

// ChatSessionOptions configures a chat session.
type ChatSessionOptions struct {
	Model      string // model name (default: deepseek-chat)
	Role       string // system role prompt (dev/expert/review/test)
	HistSize   int    // number of history messages to include context
	DscliPath  string // path to dscli executable (empty = use resolved path)
	ProjectDir string // working directory for the dscli process
}

// ─── ChatSession ────────────────────────────────────────────

// ChatSession represents an active chat interaction with dscli.
//
// Lifecycle:
//  1. TUI creates via AIAgent.NewChatSession
//  2. TUI sends the first user message via Send
//  3. TUI receives Events (chunks, ask_user, done, …)
//  4. For ask_user, TUI enters modal, then sends response via Send
//  5. On done, the session is complete (but dscli may still be running)
//  6. TUI calls Close() to clean up
type ChatSession struct {
	// Events carries messages from dscli → TUI.
	// The stream may contain: ChatChunkPayload, AskUserPayload,
	// ChatDonePayload, StatusPayload, ErrorInfo.
	Events <-chan *protocol.Message

	// Send carries messages from TUI → dscli.
	// Valid types: ChatRequestPayload, AskUserResponsePayload.
	Send chan<- *protocol.Message

	// Done is closed when the session ends (dscli process exits).
	Done <-chan struct{}

	close func() error
}

// Close terminates the session and kills the dscli process.
func (s *ChatSession) Close() error { return s.close() }

// ─── Agent Result Messages (for Bubble Tea) ─────────────────

// These types wrap agent results for use as tea.Msg in the TUI's Update loop.
// Each corresponds to an AIAgent method.

// BalanceResultMsg wraps the result of AIAgent.Balance.
type BalanceResultMsg struct {
	Payload *protocol.CommandResultPayload
	Err     error
}

// ModelsResultMsg wraps the result of AIAgent.Models.
type ModelsResultMsg struct {
	Payload *protocol.CommandResultPayload
	Err     error
}

// VersionResultMsg wraps the result of AIAgent.Version.
type VersionResultMsg struct {
	Payload *protocol.CommandResultPayload
	Err     error
}

// HistoryResultMsg wraps the result of AIAgent.History.
type HistoryResultMsg struct {
	Payload *protocol.CommandResultPayload
	Err     error
}

// SkillResultMsg wraps the result of AIAgent.Skill.
type SkillResultMsg struct {
	Payload *protocol.CommandResultPayload
	Err     error
}

// PromptResultMsg wraps the result of AIAgent.Prompt.
type PromptResultMsg struct {
	Payload *protocol.CommandResultPayload
	Err     error
}

// MemorySearchResultMsg wraps the result of AIAgent.MemorySearch.
type MemorySearchResultMsg struct {
	Payload *protocol.CommandResultPayload
	Err     error
}

// ChatEventMsg wraps a single *protocol.Message from a ChatSession.
type ChatEventMsg struct {
	Message *protocol.Message
	Done    bool // session ended
	Err     error
}

// FlycheckResultMsg wraps the result of AIAgent.Flycheck.
type FlycheckResultMsg struct {
	Payload *protocol.CommandResultPayload
	Err     error
}

// SubcommandResultMsg is a generic result wrapper for subcommand-group
// methods (History, Skill, Memory, Project, Role, Tool, Mail, Service).
// Group identifies which group (e.g. "history", "skill") — used by the TUI
// to route the result to the correct list parser.
type SubcommandResultMsg struct {
	Payload *protocol.CommandResultPayload
	Err     error
	Subcmd  string // which subcommand was invoked (e.g. "list", "show")
	Group   string // which subcommand group (e.g. "history", "skill")
}

// ChatSessionReadyMsg is emitted when a chat session has been created
// (or failed).  The TUI waits for this before entering StateChatting.
type ChatSessionReadyMsg struct {
	Session *ChatSession
	Err     error
}

// ChimeinResultMsg wraps the result of AIAgent.SendChimein.
// Output is the full stdout+stderr output of the climein dscli process.
type ChimeinResultMsg struct {
	Output string
	Err    error
}
