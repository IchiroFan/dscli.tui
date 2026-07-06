// Package protocol defines the wire-format contract between dscli.tui and dscli.
//
// This package will eventually move to the dscli project — dscli is the
// protocol definer, dscli.tui is the follower. For now it lives here as
// the canonical implementation, co-located with its primary consumer.
//
// All communication uses JSON-line encoding: each message is a single JSON
// object terminated by '\n'.  Messages are unidirectional or request/response
// depending on MessageType.
package protocol

import (
	"encoding/json"
	"fmt"
)

// MessageType classifies a protocol message.
type MessageType string

const (
	// ── Request (TUI → dscli) ──────────────────────────────

	// TypeChatRequest starts or continues a chat interaction.
	// Payload: ChatRequestPayload
	TypeChatRequest MessageType = "chat_request"

	// TypeCommand executes a non-chat subcommand (models, balance, …).
	// Payload: CommandPayload
	TypeCommand MessageType = "command"

	// TypeAskUserResp carries the user's answer to an AskUser question.
	// Payload: AskUserResponsePayload
	TypeAskUserResp MessageType = "ask_user_response"

	// ── Response / Event (dscli → TUI) ─────────────────────

	// TypeChatChunk is a streaming content delta.
	// Payload: ChatChunkPayload
	TypeChatChunk MessageType = "chunk"

	// TypeChatDone signals the end of a chat round-trip.
	// Payload: ChatDonePayload
	TypeChatDone MessageType = "done"

	// TypeAskUser asks the TUI to present a question to the user.
	// The TUI enters a modal state — all other input is blocked.
	// Payload: AskUserPayload
	TypeAskUser MessageType = "ask_user"

	// TypeCmdResult carries the result of a non-chat command.
	// Payload: CommandResultPayload
	TypeCmdResult MessageType = "command_result"

	// TypeStatus carries a spontaneous status update (loading, connecting …).
	// Payload: StatusPayload
	TypeStatus MessageType = "status"

	// TypeReady announces that dscli has started and is ready for input.
	// Payload: ReadyPayload
	TypeReady MessageType = "ready"

	// TypeGoodbye signals normal shutdown.
	// No payload (or EmptyPayload).
	TypeGoodbye MessageType = "goodbye"

	// TypeError signals a fatal error.
	// The Error field on Message is set instead of Payload.
	TypeError MessageType = "error"
)

// Payload is a sealed interface — all concrete payload types are defined
// in this package and known at compile time.
type Payload interface {
	payloadMarker()
}

// ─── ErrorInfo ──────────────────────────────────────────────

// ErrorInfo carries structured error details.
type ErrorInfo struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *ErrorInfo) Error() string { return e.Message }

// ─── Message ────────────────────────────────────────────────

// Message is the top-level unit of the JSON-line protocol.
type Message struct {
	ID      string      `json:"id"`              // request ID, for correlation
	Type    MessageType `json:"type"`            // message classifier
	Payload Payload     `json:"payload"`         // type-asserted per MessageType
	Error   *ErrorInfo  `json:"error,omitempty"` // set for TypeError
}

// ─── Custom JSON for Message (dispatch Payload by Type) ────

// rawMessage mirrors Message for JSON round-tripping.
type rawMessage struct {
	ID      string           `json:"id"`
	Type    MessageType      `json:"type"`
	Payload *json.RawMessage `json:"payload,omitempty"`
	Error   *ErrorInfo       `json:"error,omitempty"`
}

// MarshalJSON implements json.Marshaler.
func (m *Message) MarshalJSON() ([]byte, error) {
	raw := rawMessage{
		ID:    m.ID,
		Type:  m.Type,
		Error: m.Error,
	}
	if m.Payload != nil {
		data, err := json.Marshal(m.Payload)
		if err != nil {
			return nil, fmt.Errorf("marshal payload for type %s: %w", m.Type, err)
		}
		rm := json.RawMessage(data)
		raw.Payload = &rm
	}
	return json.Marshal(raw)
}

// UnmarshalJSON implements json.Unmarshaler.
func (m *Message) UnmarshalJSON(data []byte) error {
	var raw rawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	m.ID = raw.ID
	m.Type = raw.Type
	m.Error = raw.Error

	if raw.Payload != nil && len(*raw.Payload) > 0 {
		p, err := decodePayload(m.Type, *raw.Payload)
		if err != nil {
			return fmt.Errorf("decode payload for type %s: %w", m.Type, err)
		}
		m.Payload = p
	}
	return nil
}

// decodePayload dispatches JSON decoding to the correct Payload type.
func decodePayload(mt MessageType, data json.RawMessage) (Payload, error) {
	switch mt {
	// requests
	case TypeChatRequest:
		return decodeOne[ChatRequestPayload](data)
	case TypeCommand:
		return decodeOne[CommandPayload](data)
	case TypeAskUserResp:
		return decodeOne[AskUserResponsePayload](data)

	// responses / events
	case TypeChatChunk:
		return decodeOne[ChatChunkPayload](data)
	case TypeChatDone:
		return decodeOne[ChatDonePayload](data)
	case TypeAskUser:
		return decodeOne[AskUserPayload](data)
	case TypeCmdResult:
		return decodeOne[CommandResultPayload](data)
	case TypeStatus:
		return decodeOne[StatusPayload](data)
	case TypeReady:
		return decodeOne[ReadyPayload](data)
	case TypeGoodbye:
		return &EmptyPayload{}, nil
	case TypeError:
		return nil, nil // Error is in Message.Error

	default:
		return nil, fmt.Errorf("unknown message type: %s", mt)
	}
}

// decodeOne is a small generic helper.
func decodeOne[T any](data json.RawMessage) (*T, error) {
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// ─── EmptyPayload ───────────────────────────────────────────

// EmptyPayload is used for messages that carry no data.
type EmptyPayload struct{}

func (EmptyPayload) payloadMarker() {}
