package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"gitcode.com/dscli/dscli.tui/internal/tui/protocol"
)

// ─── Logo ────────────────────────────────────────────────────────────────────

// renderLogo returns the dscli ASCII art logo with gradient colors,
// inspired by dscli.gitcode's design pattern.
func renderLogo() string {
	// ASCII art: "DSCLI" in 5-row block letters (7-wide each, 39 chars total)
	logoLines := [5]string{
		"███████  ███████ ███████ ██    ██████",
		"██    ██ ██      ██      ██      ██   ",
		"██    ██ ███████ ██      ██      ██   ",
		"██    ██      ██ ██      ██      ██   ",
		"███████  ███████ ███████ █████ ██████",
	}

	// Gradient colors for the rows (purple → blue → cyan → teal → green)
	colors := []lipgloss.Color{
		colorMauve,   // Row 1 - purple
		colorPrimary, // Row 2 - blue
		colorBlue,    // Row 3 - cyan
		colorTeal,    // Row 4 - teal
		colorGreen,   // Row 5 - green
	}

	frameStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(colorOverlay).
		Padding(0, 1).
		MarginBottom(1)

	accentStyle := lipgloss.NewStyle().Foreground(colorMauve).Bold(true)
	taglineStyle := lipgloss.NewStyle().Foreground(colorSubtext).Italic(true)

	var b strings.Builder

	// Header line
	b.WriteString(accentStyle.Render(" 🐋 DSCLI TUI "))
	b.WriteString(strings.Repeat(" ", 16))
	b.WriteString(accentStyle.Render(" ONLINE "))
	b.WriteString("\n\n")

	// ASCII art with gradient
	for i, line := range logoLines {
		b.WriteString(" ")
		b.WriteString(lipgloss.NewStyle().Foreground(colors[i]).Bold(true).Render(line))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Tagline
	b.WriteString(taglineStyle.Render(" > dscli — DeepSeek CLI"))

	return frameStyle.Render(b.String()) + "\n"
}


// ─── View ────────────────────────────────────────────────────────────


// View implements tea.Model.View.
func (m *RootModel) View() string {
	switch m.screen {
	case ScreenMainMenu:
		return m.viewMainMenu()
	case ScreenRunningCmd:
		return m.viewRunningCmd()
	case ScreenShowOutput:
		return m.viewShowOutput()
	case ScreenChatting:
		return m.viewChatting()
	case ScreenAskUser:
		return m.viewAskUser()
	case ScreenQuitting:
		return "Goodbye.\n"
	default:
		return "Unknown screen.\n"
	}
}

// ─── Main Menu ───────────────────────────────────────────────────────

func (m *RootModel) viewMainMenu() string {
	var b strings.Builder

	b.WriteString(renderLogo())
	b.WriteString("\n")

	for i, item := range m.menuItems {
		if i == m.menuCursor {
			b.WriteString(MenuSelectedStyle.Render("▸ " + item.Title))
			b.WriteString("  ")
			b.WriteString(HelpStyle.Render("— " + item.Desc))
		} else {
			b.WriteString(MenuItemStyle.Render("  " + item.Title))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("↑↓ navigate • enter select • q quit • ctrl+c exit"))
	b.WriteString("\n")
	return b.String()
}

// ─── Running Command ─────────────────────────────────────────────────

func (m *RootModel) viewRunningCmd() string {
	return fmt.Sprintf("%s Running command...\n", m.spinner.View())
}

// ─── Show Output ─────────────────────────────────────────────────────

func (m *RootModel) viewShowOutput() string {
	var b strings.Builder

	icon := "📄"
	if !m.cmdSuccess {
		icon = "⚠️ "
	}

	b.WriteString(HeaderStyle.Render(fmt.Sprintf("%s Output", icon)))
	b.WriteString("\n")

	if m.cmdOutput != "" {
		b.WriteString(m.cmdOutput)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("Press any key to return to menu"))
	b.WriteString("\n")
	return AppStyle.Render(b.String())
}

// ─── Chatting ────────────────────────────────────────────────────────

func (m *RootModel) viewChatting() string {
	var b strings.Builder

	// ── Header ──
	b.WriteString(HeaderStyle.Render("💬 Chat"))
	b.WriteString("\n")

	// ── History area ──
	// Calculate how many lines fit: height minus header, footer, input, status.
	maxLines := m.Height - 8
	if maxLines < 5 {
		maxLines = 5
	}

	// Build message lines with proper styling.
	var msgLines []string
	for _, line := range m.chatHistory {
		var prefix string
		switch line.Role {
		case "user":
			prefix = ChatRoleUserStyle.Render("You:")
		case "assistant":
			prefix = ChatRoleAssistantStyle.Render("AI:")
		case "reasoning":
			prefix = ThinkLineStyle.Render("...")
		default:
			prefix = ""
		}
		msgLines = append(msgLines, prefix+" "+line.Content)
	}

	// Show pending user input (not yet committed to history).
	if m.chatPendingInput != "" && !m.chatLoading && !m.chatDone {
		msgLines = append(msgLines,
			fmt.Sprintf("%s %s %s",
				ChatRoleUserStyle.Render("You:"),
				m.chatPendingInput,
				HelpStyle.Render("(pending...)")))
	}

	// ── Scrolling ──
	totalLines := len(msgLines)
	if totalLines > maxLines {
		m.chatScrollMax = totalLines - maxLines
		// Clamp scroll.
		if m.chatScroll > m.chatScrollMax {
			m.chatScroll = m.chatScrollMax
		}
		// Show the window: from (totalLines - maxLines - scroll) to end.
		start := totalLines - maxLines - m.chatScroll
		if start < 0 {
			start = 0
		}
		end := start + maxLines
		if end > totalLines {
			end = totalLines
		}
		msgLines = msgLines[start:end]

		// Scroll indicator.
		if m.chatScroll > 0 {
			b.WriteString(HelpStyle.Render(fmt.Sprintf("↑ %d more lines above", m.chatScroll)))
			b.WriteString("\n")
		}
	} else {
		m.chatScrollMax = 0
		m.chatScroll = 0
	}

	for _, l := range msgLines {
		b.WriteString(l)
		b.WriteString("\n")
	}

	// ── Status / Spinner ──
	if m.chatLoading {
		b.WriteString("\n")
		b.WriteString(ChatLoadingStyle.Render(fmt.Sprintf("%s AI is thinking...", m.spinner.View())))
		b.WriteString("\n")
	} else if m.chatDone {
		b.WriteString("\n")
		b.WriteString(SpinnerDoneStyle.Render("✅ Response complete. Type another message or press Esc for menu."))
		b.WriteString("\n")
	}

	// ── Input line ──
	b.WriteString("\n")
	b.WriteString(m.chatInput.View())
	b.WriteString("\n")

	// ── Footer ──
	b.WriteString(HelpStyle.Render("Esc: menu • Enter: send • PgUp/PgDn: scroll"))
	b.WriteString("\n")

	return AppStyle.Render(b.String())
}

// ─── AskUser Modal ───────────────────────────────────────────────────

func (m *RootModel) viewAskUser() string {
	var b strings.Builder

	width := m.Width
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
		b.WriteString(fmt.Sprintf("│  %s  %-*s│\n",
			ChatRoleUserStyle.Render("[ y / n ]"),
			boxW-18, "(y = yes, n = no)"))

	case protocol.SemanticChoice:
		b.WriteString(fmt.Sprintf("│  %-*s│\n", boxW-4, "Select an option:"))
		for i, opt := range m.askOptions {
			cursor := "  "
			if i == m.askChoice {
				cursor = MenuSelectedStyle.Render("▸")
			}
			line := fmt.Sprintf("│    %s %s", cursor, opt)
			padding := boxW - 8 - len(cursor) - len(opt)
			if padding > 0 {
				line += strings.Repeat(" ", padding)
			}
			line += "│\n"
			b.WriteString(line)
		}

	case protocol.SemanticInput:
		b.WriteString(fmt.Sprintf("│  %s│\n", m.askInput.View()))
	}

	// ── Bottom border ──
	b.WriteString("└")
	b.WriteString(strings.Repeat("─", boxW-2))
	b.WriteString("┘\n")

	// ── Help line ──
	switch m.askSemantic {
	case protocol.SemanticConfirm:
		b.WriteString(HelpStyle.Render("Press y or n to answer"))
	case protocol.SemanticChoice:
		b.WriteString(HelpStyle.Render("↑↓ navigate • enter select"))
	case protocol.SemanticInput:
		b.WriteString(HelpStyle.Render("Type your answer • enter confirm • esc cancel"))
	}
	b.WriteString("\n")

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
