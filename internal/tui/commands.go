package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"gitcode.com/dscli/dscli.tui/internal/aiagent"
	"gitcode.com/dscli/dscli.tui/internal/tui/protocol"
)

// ─── Non-interactive command factories ──────────────────────────────
//
// Each returns a tea.Cmd that calls the corresponding AIAgent method and
// wraps the result into a typed message the update loop can handle.

func cmdBalance(agent aiagent.AIAgent) tea.Cmd {
	return func() tea.Msg {
		p, err := agent.Balance(context.Background(), "")
		return aiagent.BalanceResultMsg{Payload: p, Err: err}
	}
}

func cmdModels(agent aiagent.AIAgent) tea.Cmd {
	return func() tea.Msg {
		p, err := agent.Models(context.Background(), "", false)
		return aiagent.ModelsResultMsg{Payload: p, Err: err}
	}
}

func cmdVersion(agent aiagent.AIAgent) tea.Cmd {
	return func() tea.Msg {
		p, err := agent.Version(context.Background())
		return aiagent.VersionResultMsg{Payload: p, Err: err}
	}
}

func cmdFlycheck(agent aiagent.AIAgent) tea.Cmd {
	return func() tea.Msg {
		// TODO: let user specify path via AskUser or argument
		p, err := agent.Flycheck(context.Background(), ".", false)
		return aiagent.FlycheckResultMsg{Payload: p, Err: err}
	}
}

// cmdSubcommand is a generic factory for subcommand-group methods
// (history, skill, memory, project, role, tool, mail, service).
// args are additional arguments after the subcommand (e.g. "show", "12345").
func cmdSubcommand(agent aiagent.AIAgent, method func(context.Context, string, ...string) (*protocol.CommandResultPayload, error), subcmd string, args ...string) tea.Cmd {
	return func() tea.Msg {
		p, err := method(context.Background(), subcmd, args...)
		return aiagent.SubcommandResultMsg{
			Payload: p,
			Err:     err,
			Subcmd:  subcmd,
		}
	}
}

// ─── Chat command factories ─────────────────────────────────────────

// cmdStartChat spawns a new dscli chat process and returns a
// ChatSessionReadyMsg when the session is ready.
func cmdStartChat(agent aiagent.AIAgent, history []ChatLine) tea.Cmd {
	return func() tea.Msg {
		opts := aiagent.ChatSessionOptions{
			Model: "deepseek-chat",
		}
		session, err := agent.NewChatSession(context.Background(), opts)
		if err != nil {
			return aiagent.ChatSessionReadyMsg{Err: err}
		}
		return aiagent.ChatSessionReadyMsg{Session: session}
	}
}

// cmdSendChatMessage sends a chat request with the given history and waits
// for events. The history already contains the latest user message
// (added on Enter), so we use it as-is without adding a duplicate.
func cmdSendChatMessage(session *aiagent.ChatSession, history []ChatLine) tea.Cmd {
	// Build message list from history.
	messages := make([]protocol.ChatMessage, 0, len(history))
	for _, line := range history {
		messages = append(messages, protocol.ChatMessage{
			Role:    line.Role,
			Content: line.Content,
		})
	}
	req := &protocol.Message{
		Type: protocol.TypeChatRequest,
		Payload: &protocol.ChatRequestPayload{
			Messages: messages,
		},
	}
	// Send and then wait for events.
	return tea.Sequence(
		func() tea.Msg {
			session.Send <- req
			return nil
		},
		cmdWaitChatEvent(session),
	)
}

// cmdSendChimein runs dscli chat in climein mode with the given content.
// The new dscli process either:
//   - enters climein mode (lock held by primary) — writes to chimeins, exits quickly
//   - becomes primary (lock released) — produces full AI response
//   - exits abnormally — error is returned
//
// The result is delivered as aiagent.ChimeinResultMsg.
func cmdSendChimein(agent aiagent.AIAgent, content string) tea.Cmd {
	return func() tea.Msg {
		output, err := agent.SendChimein(context.Background(), content)
		return aiagent.ChimeinResultMsg{Output: output, Err: err}
	}
}

// cmdWaitChatEvent blocks until the next event arrives from the chat
// session's Events channel or the session is done.
func cmdWaitChatEvent(session *aiagent.ChatSession) tea.Cmd {
	return func() tea.Msg {
		select {
		case msg, ok := <-session.Events:
			if !ok {
				return aiagent.ChatEventMsg{Done: true}
			}
			return aiagent.ChatEventMsg{Message: msg}
		case <-session.Done:
			return aiagent.ChatEventMsg{Done: true}
		}
	}
}

// cmdSendAskUserResponse sends the user's answer back to dscli and
// resumes waiting for events.
func cmdSendAskUserResponse(session *aiagent.ChatSession, resp *protocol.AskUserResponsePayload) tea.Cmd {
	req := &protocol.Message{
		Type:    protocol.TypeAskUserResp,
		Payload: resp,
	}
	return tea.Sequence(
		func() tea.Msg {
			session.Send <- req
			return nil
		},
		cmdWaitChatEvent(session),
	)
}

// ─── Helpers ─────────────────────────────────────────────────────────

// cmdBackToMenu transitions to the main menu.
func cmdBackToMenu() tea.Cmd {
	return func() tea.Msg {
		return navBackToMenuMsg{}
	}
}

// cmdQuit requests graceful shutdown.
func cmdQuit() tea.Cmd {
	return tea.Quit
}

// ─── Custom message types (in addition to those in agent.go) ────────

// navBackToMenuMsg is an internal signal to return to the main menu.
type navBackToMenuMsg struct{}

// subcommandErrMsg wraps an error for subcommand execution.
type subcommandErrMsg struct{ Err error }

// ─── Additional tea.Msg types in aiagent that we use ────────────────
//
// aiagent.ChatSessionReadyMsg — emitted when a chat session is created
// aiagent.ChatEventMsg        — emitted when a chat event arrives
// aiagent.BalanceResultMsg    — result of Balance()
// aiagent.ModelsResultMsg     — result of Models()
// aiagent.VersionResultMsg    — result of Version()
// aiagent.FlycheckResultMsg   — result of Flycheck()
// aiagent.SubcommandResultMsg — generic subcommand result
//   (These are defined in internal/aiagent/agent.go)

// ─── Error formatting ───────────────────────────────────────────────

func formatCommandResult(p *protocol.CommandResultPayload, err error) string {
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	if p == nil {
		return "No output."
	}
	if !p.Success {
		return fmt.Sprintf("Command failed:\n%s", p.Data)
	}
	return strings.TrimSpace(p.Data)
}
