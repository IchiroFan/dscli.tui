package tui

import (
	"fmt"
	"strings"

	"gitcode.com/dscli/dscli.tui/internal/tui/protocol"
)

// ─── View ────────────────────────────────────────────────────────────

// View implements tea.Model.View.
func (m *RootModel) View() string {
	switch m.state {
	case StateMainMenu:
		return m.viewMainMenu()
	case StateRunningCmd:
		return m.viewRunningCmd()
	case StateShowOutput:
		return m.viewShowOutput()
	case StateChatting:
		return m.viewChatting()
	case StateAskUser:
		return m.viewAskUser()
	case StateQuitting:
		return "Goodbye.\n"
	default:
		return "Unknown state.\n"
	}
}

// ─── Main Menu ───────────────────────────────────────────────────────

func (m *RootModel) viewMainMenu() string {
	var b strings.Builder

	b.WriteString("╔══════════════════════════════════════════╗\n")
	b.WriteString("║           dscli.tui — Main Menu          ║\n")
	b.WriteString("╚══════════════════════════════════════════╝\n\n")

	for i, item := range m.menuItems {
		cursor := "  "
		style := ""
		reset := ""

		if i == m.menuCursor {
			cursor = "▸ "
			style = "\033[1;36m" // bold cyan
			reset = "\033[0m"
		}

		line := fmt.Sprintf("%s%s%-2s %s\033[0m", style, cursor, item.Title, reset)
		b.WriteString(line)
		// Description on same line, dimmed.
		if i == m.menuCursor {
			b.WriteString(fmt.Sprintf(" \033[2m— %s\033[0m", item.Desc))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n\033[2m↑↓ navigate • enter select • q quit • ctrl+c exit\033[0m\n")
	return b.String()
}

// ─── Running Command ─────────────────────────────────────────────────

func (m *RootModel) viewRunningCmd() string {
	return "⏳ Running command...\n"
}

// ─── Show Output ─────────────────────────────────────────────────────

func (m *RootModel) viewShowOutput() string {
	var b strings.Builder

	title := "Output"
	icon := "📄"
	if !m.cmdSuccess {
		icon = "⚠️ "
	}

	b.WriteString(fmt.Sprintf("╔══════════════════════════════════════════╗\n"))
	b.WriteString(fmt.Sprintf("║  %s %s", icon, title))
	// Right-align closing border
	b.WriteString(strings.Repeat(" ", 36-len(title)))
	b.WriteString("║\n")
	b.WriteString(fmt.Sprintf("╚══════════════════════════════════════════╝\n\n"))

	if m.cmdOutput != "" {
		b.WriteString(m.cmdOutput)
		b.WriteString("\n")
	}

	b.WriteString("\n\033[2mPress any key to return to menu\033[0m\n")
	return b.String()
}

// ─── Chatting ────────────────────────────────────────────────────────

func (m *RootModel) viewChatting() string {
	var b strings.Builder

	b.WriteString("╔══════════════════════════════════════════╗\n")
	b.WriteString("║  💬 Chat")
	b.WriteString(strings.Repeat(" ", 32))
	b.WriteString("║\n")
	b.WriteString("╚══════════════════════════════════════════╝\n\n")

	// Render chat history (scrollable region).
	// We show as many lines as fit in the available height, minus 4 for
	// borders, input line, and status.
	maxLines := m.height - 6
	if maxLines < 5 {
		maxLines = 5
	}

	// Build message lines.
	var msgLines []string
	for _, line := range m.chatHistory {
		prefix := "  "
		switch line.Role {
		case "user":
			prefix = "\033[1;33mYou:\033[0m "
		case "assistant":
			prefix = "\033[1;32mAI:\033[0m  "
		case "reasoning":
			prefix = "\033[2;35m...\033[0m "
		}
		msgLines = append(msgLines, prefix+line.Content)
	}

	// Show pending user input (not yet committed to history).
	if m.chatPendingInput != "" && !m.chatLoading && !m.chatDone {
		msgLines = append(msgLines,
			fmt.Sprintf("\033[1;33mYou:\033[0m %s \033[2m(pending...)\033[0m", m.chatPendingInput))
	}

	// Show last N lines.
	if len(msgLines) > maxLines {
		msgLines = msgLines[len(msgLines)-maxLines:]
	}
	for _, l := range msgLines {
		b.WriteString(l)
		b.WriteString("\n")
	}

	// Status line.
	if m.chatLoading {
		b.WriteString("\n\033[2m⏳ AI is thinking...\033[0m\n")
	} else if m.chatDone {
		b.WriteString("\n\033[2m✅ Response complete. Type another message or press Esc for menu.\033[0m\n")
	}

	// Input line.
	inputStr := string(m.chatInput)
	b.WriteString(fmt.Sprintf("\n\033[1;36m>\033[0m %s", inputStr))
	// Cursor (blinking bar simulated with block).
	if len(inputStr) == m.chatCursor {
		b.WriteString("\033[5m█\033[0m")
	} else {
		b.WriteString("\033[0m") // reset
	}

	b.WriteString(fmt.Sprintf("  \033[2m(%d/%d)\033[0m", m.chatCursor, len(inputStr)))
	b.WriteString("\n")

	b.WriteString("\033[2mEsc: menu • Enter: send\033[0m\n")
	return b.String()
}

// ─── AskUser Modal ───────────────────────────────────────────────────

func (m *RootModel) viewAskUser() string {
	var b strings.Builder

	// Dimmed background (overlay effect) — just draw a modal box.
	width := m.width
	if width < 50 {
		width = 50
	}
	boxW := width - 4
	if boxW < 40 {
		boxW = 40
	}

	// ── Top border ──
	b.WriteString("┌")
	b.WriteString(strings.Repeat("─", boxW-2))
	b.WriteString("┐\n")

	// ── Title ──
	title := "🤖 dscli asks:"
	b.WriteString(fmt.Sprintf("│ %-*s│\n", boxW-4, title))

	// ── Separator ──
	b.WriteString("│")
	b.WriteString(strings.Repeat("─", boxW-4))
	b.WriteString("│\n")

	// ── Question ──
	lines := wrapText(m.askQuestion, boxW-6)
	for _, line := range lines {
		b.WriteString(fmt.Sprintf("│  %-*s│\n", boxW-6, line))
	}

	// ── Input area ──
	b.WriteString("│")
	b.WriteString(strings.Repeat("─", boxW-4))
	b.WriteString("│\n")

	switch m.askSemantic {
	case protocol.SemanticConfirm:
		b.WriteString(fmt.Sprintf("│  \033[1;33m[ y / n ]\033[0m  %-*s│\n",
			boxW-18, "(y = yes, n = no)"))

	case protocol.SemanticChoice:
		b.WriteString(fmt.Sprintf("│  %-*s│\n", boxW-4, "Select an option:"))
		for i, opt := range m.askOptions {
			cursor := "  "
			if i == m.askChoice {
				cursor = "▸ "
			}
			line := fmt.Sprintf("│    %s%s", cursor, opt)
			padding := boxW - 6 - len(cursor) - len(opt)
			if padding > 0 {
				line += strings.Repeat(" ", padding)
			}
			line += "│\n"
			b.WriteString(line)
		}

	case protocol.SemanticInput:
		inputStr := string(m.askInput)
		b.WriteString(fmt.Sprintf("│  \033[1;36m>\033[0m %s", inputStr))
		if len(inputStr) == m.askCursor {
			b.WriteString("\033[5m█\033[0m")
		}
		// padding to box width
		padding := boxW - 8 - len(inputStr)
		if padding > 0 {
			b.WriteString(strings.Repeat(" ", padding))
		}
		b.WriteString("│\n")
	}

	// ── Bottom border ──
	b.WriteString("└")
	b.WriteString(strings.Repeat("─", boxW-2))
	b.WriteString("┘\n")

	// ── Help line ──
	switch m.askSemantic {
	case protocol.SemanticConfirm:
		b.WriteString("\033[2mPress y or n to answer\033[0m\n")
	case protocol.SemanticChoice:
		b.WriteString("\033[2m↑↓ navigate • enter select\033[0m\n")
	case protocol.SemanticInput:
		b.WriteString("\033[2mType your answer • enter confirm • esc cancel\033[0m\n")
	}

	return b.String()
}

// ─── Utility ─────────────────────────────────────────────────────────

// wrapText wraps text to a given width, splitting on word boundaries.
func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	var lines []string
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	current := words[0]
	for _, word := range words[1:] {
		if len(current)+1+len(word) <= width {
			current += " " + word
		} else {
			lines = append(lines, current)
			current = word
		}
	}
	lines = append(lines, current)
	return lines
}
