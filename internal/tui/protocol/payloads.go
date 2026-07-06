package protocol

// ─── Payload: ChatRequestPayload ────────────────────────────

// ChatRequestPayload is sent by the TUI to start / continue a chat.
type ChatRequestPayload struct {
	Model    string        `json:"model"`    // model name, e.g. "deepseek-chat"
	Messages []ChatMessage `json:"messages"` // full message history
	Stream   bool          `json:"stream"`   // enable streaming
}

// ChatMessage is a single entry in the message history (OpenAI-compatible).
type ChatMessage struct {
	Role    string `json:"role"` // "user" | "assistant" | "system"
	Content string `json:"content"`
}

func (ChatRequestPayload) payloadMarker() {}

// ─── Payload: ChatChunkPayload ──────────────────────────────

// ChatChunkPayload is a streaming content delta.
type ChatChunkPayload struct {
	Content   string `json:"content"`
	Reasoning string `json:"reasoning,omitempty"` // optional reasoning content
}

func (ChatChunkPayload) payloadMarker() {}

// ─── Payload: ChatDonePayload ───────────────────────────────

// ChatDonePayload signals the end of a chat round-trip.
type ChatDonePayload struct {
	Usage     *UsageInfo `json:"usage,omitempty"`
	TotalTime string     `json:"total_time,omitempty"`
	Cost      string     `json:"cost,omitempty"`
}

// UsageInfo carries token usage statistics.
type UsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func (ChatDonePayload) payloadMarker() {}

// ─── Payload: AskUserPayload ────────────────────────────────

// Semantic classifies the kind of answer expected from the user.
type Semantic string

const (
	SemanticConfirm Semantic = "confirm" // Y/N confirmation
	SemanticChoice  Semantic = "choice"  // pick one from Options
	SemanticInput   Semantic = "input"   // free-text input
)

// AskUserPayload is sent by dscli → TUI when the AI needs user input.
// The TUI MUST enter modal mode — all other input is blocked.
type AskUserPayload struct {
	Question string   `json:"question"`
	Semantic Semantic `json:"semantic"`
	Options  []string `json:"options,omitempty"`
	Timeout  int64    `json:"timeout,omitempty"` // seconds, 0 = no timeout
}

func (AskUserPayload) payloadMarker() {}

// ─── Payload: AskUserResponsePayload ────────────────────────

// AskUserResponsePayload carries the user's answer back to dscli.
// At least one of Value / Choice is set depending on Semantic:
//
//	confirm → Value is "yes" | "no"
//	choice  → Choice is the selected index
//	input   → Value is the free-text input
type AskUserResponsePayload struct {
	Value   string `json:"value,omitempty"`
	Choice  int    `json:"choice,omitempty"`
	Timeout bool   `json:"timeout,omitempty"`
}

func (AskUserResponsePayload) payloadMarker() {}

// ─── Payload: CommandPayload ────────────────────────────────

// CommandPayload executes a non-chat subcommand.
type CommandPayload struct {
	Name string   `json:"name"` // e.g. "models", "balance", "history"
	Args []string `json:"args,omitempty"`
}

func (CommandPayload) payloadMarker() {}

// ─── Payload: CommandResultPayload ──────────────────────────

// CommandResultPayload carries the result of a non-chat command.
type CommandResultPayload struct {
	Success bool   `json:"success"`
	Data    string `json:"data"` // text output (JSON or formatted)
}

func (CommandResultPayload) payloadMarker() {}

// ─── Payload: StatusPayload ─────────────────────────────────

// StatusPayload is a spontaneous status update from dscli.
type StatusPayload struct {
	Status  string `json:"status"` // "loading" | "connecting" | "processing"
	Message string `json:"message,omitempty"`
}

func (StatusPayload) payloadMarker() {}

// ─── Payload: ReadyPayload ──────────────────────────────────

// ReadyPayload is the first message dscli sends after startup.
type ReadyPayload struct {
	Version  string   `json:"version"`            // dscli version string
	Features []string `json:"features,omitempty"` // supported features
}

func (ReadyPayload) payloadMarker() {}
