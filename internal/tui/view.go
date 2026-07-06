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

// ─── Status Bar ──────────────────────────────────────────────────────────────

// renderStatusBar returns the full-width status bar at the bottom of every screen.
// Format:  dscli v1.0 │ 📁 ~/proj │ 🤖 deepseek-chat  │ 💬 Chat
func (m *RootModel) renderStatusBar() string {
	// Version badge.
	version := strings.TrimSpace(m.dscliVersion)
	if version == "" {
		version = "dscli"
	}
	badge := StatusVersion.Render(" " + version + " ")

	// Project + model labels.
	projectLabel := StatusLabel.Render(" 📁 " + m.projectRoot + " ")
	modelLabel := StatusLabel.Render(" 🤖 " + m.modelName + " ")
	sep := StatusSep.Render("│")

	// Screen name badge (right side).
	var screenName string
	switch m.screen {
	case ScreenMainMenu:
		screenName = "📋 Menu"
	case ScreenChatting:
		screenName = "💬 Chat"
	case ScreenShowOutput:
		screenName = "📄 Output"
	case ScreenHistoryList:
		screenName = "📝 History"
	case ScreenAskUser:
		screenName = "❓ Ask"
	case ScreenRunningCmd:
		screenName = "⏳ Running"
	case ScreenQuitting:
		screenName = "👋 Quit"
	}
	screenBadge := StatusScreen.Render(" " + screenName + " ")

	// Assemble left section: badge │ project │ model
	left := badge + " " + sep + " " + projectLabel + " " + sep + " " + modelLabel
	right := screenBadge

	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	fillerW := m.Width - leftW - rightW
	if fillerW < 0 {
		fillerW = 0
	}
	filler := strings.Repeat(" ", fillerW)

	barText := left + filler + right
	return StatusBarBg.Width(m.Width).Render(barText)
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
	case ScreenHistoryList:
		return m.viewHistoryList()
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
		titleStyle := MenuItemStyle
		if i == m.menuCursor {
			titleStyle = MenuSelectedStyle
		}
		b.WriteString(titleStyle.Render(item.Title))
		b.WriteString("  ")
		b.WriteString(MenuDescStyle.Render("— " + item.Desc))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("↑↓ navigate • enter select • q quit • ctrl+c exit"))
	b.WriteString("\n")
	b.WriteString(m.renderStatusBar())
	return b.String()
}

// ─── Running Command ─────────────────────────────────────────────────

func (m *RootModel) viewRunningCmd() string {
	content := fmt.Sprintf("%s Running command...\n", m.spinner.View())
	return AppStyle.Width(m.Width).Render(content) + "\n" + m.renderStatusBar()
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

	// Calculate available height for output content.
	// Reservations: header(2) + help lines(2) + status bar(1) = 5
	availableHeight := m.Height - 5
	if availableHeight < 3 {
		availableHeight = 3
	}

	// Fallback: split cmdOutput if outputLines hasn't been initialised.
	outputLines := m.outputLines
	if outputLines == nil {
		if m.cmdOutput != "" {
			outputLines = strings.Split(m.cmdOutput, "\n")
		} else {
			outputLines = []string{}
		}
	}

	totalLines := len(outputLines)

	// Compute scroll max and clamp scroll (read-only, no side-effect on model).
	scrollMax := totalLines - availableHeight
	if scrollMax < 0 {
		scrollMax = 0
	}
	scroll := m.outputScroll
	if scroll > scrollMax {
		scroll = scrollMax
	}
	if scroll < 0 {
		scroll = 0
	}

	// ── Scroll indicator: top ──
	if scroll > 0 {
		b.WriteString(HelpStyle.Render(fmt.Sprintf("↑ %d more lines above  (g: top)", scroll)))
		b.WriteString("\n")
	}

	// ── Visible content ──
	end := scroll + availableHeight
	if end > totalLines {
		end = totalLines
	}
	for _, line := range outputLines[scroll:end] {
		b.WriteString(line)
		b.WriteString("\n")
	}

	// ── Scroll indicator: bottom ──
	remaining := totalLines - end
	if remaining > 0 {
		b.WriteString(HelpStyle.Render(fmt.Sprintf("↓ %d more lines below  (G: bottom)", remaining)))
		b.WriteString("\n")
	}

	// ── Help bar ──
	b.WriteString(HelpStyle.Render("↑↓ scroll · PgUp/PgDn page · g/G top/bottom · Esc/q back to menu"))
	b.WriteString("\n")

	return AppStyle.Width(m.Width).Render(b.String()) + "\n" + m.renderStatusBar()
}

// ─── History List ─────────────────────────────────────────────────

// viewHistoryList renders the selectable history message list.
func (m *RootModel) viewHistoryList() string {
	var b strings.Builder

	b.WriteString(HeaderStyle.Render("📝 History"))
	b.WriteString("\n")

	if len(m.historyItems) == 0 {
		b.WriteString(NoDataStyle.Render("No history messages found."))
		b.WriteString("\n\n")
		b.WriteString(HelpStyle.Render("Esc/q — back to menu"))
		b.WriteString("\n")
		return AppStyle.Width(m.Width).Render(b.String()) + "\n" + m.renderStatusBar()
	}

	// Calculate how many items fit.
	// Reservations: header(2) + help line(2) + status bar(1) = 5
	availableLines := m.Height - 5
	if availableLines < 3 {
		availableLines = 3
	}

	// Display range based on cursor position (keep cursor visible).
	cursor := m.historyCursor
	half := availableLines / 2
	start := cursor - half
	if start < 0 {
		start = 0
	}
	end := start + availableLines
	if end > len(m.historyItems) {
		end = len(m.historyItems)
		start = end - availableLines
		if start < 0 {
			start = 0
		}
	}

	// Scroll indicator: top
	if start > 0 {
		b.WriteString(HelpStyle.Render(fmt.Sprintf("↑ %d more items above", start)))
		b.WriteString("\n")
	}

	for i := start; i < end; i++ {
		item := m.historyItems[i]
		// Format: "ID  Role  [status]"
		roleIcon := ""
		switch item.Role {
		case "assistant":
			roleIcon = "🧠"
		case "user":
			roleIcon = "👤"
		case "tool":
			roleIcon = "🔧"
		default:
			roleIcon = "❓"
		}
		status := ""
		if item.Done == "false" {
			status = TimestampStyle.Render(" ⏳")
		}
		line := fmt.Sprintf("%s %s %s%s", item.ID, roleIcon, item.Role, status)

		if i == cursor {
			b.WriteString(ListSelectedStyle.Render("▸ " + line))
		} else {
			b.WriteString(ListItemStyle.Render("  " + line))
		}
		b.WriteString("\n")
	}

	// Scroll indicator: bottom
	remaining := len(m.historyItems) - end
	if remaining > 0 {
		b.WriteString(HelpStyle.Render(fmt.Sprintf("↓ %d more items below", remaining)))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("↑↓ navigate · Enter show · Esc/q back to menu"))
	b.WriteString("\n")

	return AppStyle.Width(m.Width).Render(b.String()) + "\n" + m.renderStatusBar()
}

// ─── Chatting

func (m *RootModel) viewChatting() string {
	var b strings.Builder

	// ── Header ──
	b.WriteString(HeaderStyle.Render("💬 Chat"))
	b.WriteString("\n")

	// ── Dimensions ──
	contentW := m.Width - 4 // AppStyle padding (2 left + 2 right)
	if contentW < 20 {
		contentW = 20
	}
	bubbleMaxW := contentW * BubbleMaxPercent / 100
	if bubbleMaxW < 20 {
		bubbleMaxW = 20
	}
	const borderPad = 4 // RoundedBorder (2) + Padding(0,1) (2) = 4
	contentAreaW := bubbleMaxW - borderPad
	if contentAreaW < 10 {
		contentAreaW = 10
	}
	wrapStyle := lipgloss.NewStyle().Width(contentAreaW)

	// ── History area ──
	// Calculate how many visual lines fit:
	//   header(1) + spacer(1) + history(N) + loading(1) + input(3) + footer(1) + status(1) = header+spacer+input+footer+status
	// Reserve: header(2) + input(3) + footer(1) + loading(1) + margin = ~8
	maxLines := m.Height - 8
	if maxLines < 5 {
		maxLines = 5
	}

	// Build rendered bubbles for each chat line.
	var renderedBubbles []string
	for _, line := range m.chatHistory {
		var rendered string
		switch line.Role {
		case "user":
			rendered = RenderBubble(UserBubbleBase, "👤 ", line.Content, wrapStyle, contentAreaW)
			// Right-align user bubbles: pad each line individually so that
			// wrapped multi-line bubbles stay aligned (not just the first line).
			lw := lipgloss.Width(rendered)
			if pad := contentW - lw; pad > 0 {
				lines := strings.Split(rendered, "\n")
				for i, l := range lines {
					lines[i] = strings.Repeat(" ", pad) + l
				}
				rendered = strings.Join(lines, "\n")
			}
		case "assistant":
			rendered = RenderBubble(AssistantBubbleBase, "🧠 ", line.Content, wrapStyle, contentAreaW)
		case "reasoning":
			rendered = RenderBubble(ThinkBubbleBase, "", line.Content, wrapStyle, contentAreaW)
		default:
			rendered = line.Content
		}
		renderedBubbles = append(renderedBubbles, rendered)
	}

	// Join all rendered bubbles into a single text block, then split into visual lines.


	var fullMsgText string
	for _, rb := range renderedBubbles {
		fullMsgText += rb + "\n"
	}
	allLines := strings.Split(strings.TrimSuffix(fullMsgText, "\n"), "\n")
	totalLines := len(allLines)

	// ── Scrolling ──
	if totalLines > maxLines {
		m.chatScrollMax = totalLines - maxLines
		if m.chatScroll > m.chatScrollMax {
			m.chatScroll = m.chatScrollMax
		}
		start := totalLines - maxLines - m.chatScroll
		if start < 0 {
			start = 0
		}
		end := start + maxLines
		if end > totalLines {
			end = totalLines
		}
		// Scroll indicator at top.
		if m.chatScroll > 0 {
			b.WriteString(HelpStyle.Render(fmt.Sprintf("↑ %d more lines above", m.chatScroll)))
			b.WriteString("\n")
		}

		for _, l := range allLines[start:end] {
			b.WriteString(l)
			b.WriteString("\n")
		}

		// Scroll indicator at bottom (when scrolled up from bottom).
		remaining := totalLines - end
		if remaining > 0 {
			b.WriteString(HelpStyle.Render(fmt.Sprintf("↓ %d more lines below", remaining)))
			b.WriteString("\n")
		}
	} else {
		m.chatScrollMax = 0
		m.chatScroll = 0
		for _, l := range allLines {
			b.WriteString(l)
			b.WriteString("\n")
		}
	}

	// ── Status / Spinner ──
	if m.chatLoading {
		b.WriteString("\n")
		b.WriteString(ChatLoadingStyle.Render(fmt.Sprintf("%s AI is thinking...", m.spinner.View())))
		b.WriteString("\n")
	} else if m.chatDone {
		b.WriteString("\n")
		b.WriteString(SpinnerDoneStyle.Render("✅ Response complete"))
		b.WriteString("\n")
	}

	// ── Input area (multi-line textarea with blue border) ──
	b.WriteString("\n")
	b.WriteString(m.chatInput.View())
	b.WriteString("\n")

	b.WriteString(HelpStyle.Render("Esc: menu • Enter: send • Ctrl+J: newline • PgUp/PgDn/Ctrl↑↓: scroll"))
	b.WriteString("\n")

	return AppStyle.Width(m.Width).Render(b.String()) + "\n" + m.renderStatusBar()
}

// ─── AskUser Modal ───────────────────────────────────────────────────

func (m *RootModel) viewAskUser() string {
	var content strings.Builder

	// ── Title ──
	title := "🤖 dscli asks:"
	content.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).Render(title))
	content.WriteString("\n\n")

	// ── Question (word-wrapped) ──
	boxInnerW := 44 // comfortable inner width for the modal
	if m.Width-8 < boxInnerW {
		boxInnerW = m.Width - 8
		if boxInnerW < 30 {
			boxInnerW = 30
		}
	}
	for _, line := range wrapText(m.askQuestion, boxInnerW-4) {
		content.WriteString("  ")
		content.WriteString(line)
		content.WriteString("\n")
	}
	content.WriteString("\n")

	// ── Separator ──
	content.WriteString("  ")
	content.WriteString(strings.Repeat("─", boxInnerW-4))
	content.WriteString("\n\n")

	// ── Input area ──
	switch m.askSemantic {
	case protocol.SemanticConfirm:
		content.WriteString("  ")
		content.WriteString(ChatRoleUserStyle.Render("[ y / n ]"))
		content.WriteString("  ")
		content.WriteString(HelpStyle.Render("(y = yes, n = no)"))
		content.WriteString("\n")

	case protocol.SemanticChoice:
		content.WriteString("  ")
		content.WriteString(HelpStyle.Render("Select an option:"))
		content.WriteString("\n")
		for i, opt := range m.askOptions {
			cursor := "  "
			if i == m.askChoice {
				cursor = MenuSelectedStyle.Render("▸ ")
			}
			content.WriteString(fmt.Sprintf("    %s%s\n", cursor, opt))
		}

	case protocol.SemanticInput:
		content.WriteString("  ")
		content.WriteString(m.askInput.View())
		content.WriteString("\n")
	}

	// ── Help line ──
	content.WriteString("\n")
	switch m.askSemantic {
	case protocol.SemanticConfirm:
		content.WriteString("  ")
		content.WriteString(HelpStyle.Render("Press y or n to answer"))
	case protocol.SemanticChoice:
		content.WriteString("  ")
		content.WriteString(HelpStyle.Render("↑↓ navigate · enter select"))
	case protocol.SemanticInput:
		content.WriteString("  ")
		content.WriteString(HelpStyle.Render("Type · enter confirm · esc cancel"))
	}
	content.WriteString("\n")

	// ── Wrap in a lipgloss rounded-border box ──
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(0, 1)

	result := boxStyle.Render(content.String()) + "\n"
	result += m.renderStatusBar()
	return result
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
