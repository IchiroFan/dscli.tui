// Package tui implements the Bubble Tea application for dscli.tui.
//
// Styles follow the Tokyo Night color palette, inspired by
// the dscli.gitcode project's design system.
package tui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ─── Colors (Tokyo Night palette) ───────────────────────────────────────────

var (
	colorBase    = lipgloss.Color("#1a1b26") // Dark background
	colorSurface = lipgloss.Color("#24253e") // Panel background
	colorOverlay = lipgloss.Color("#565f89") // Muted borders
	colorText    = lipgloss.Color("#c0caf5") // Light text
	colorSubtext = lipgloss.Color("#9aa5ce") // Dim text
	colorPrimary = lipgloss.Color("#7aa2f7") // Primary blue
	colorGreen   = lipgloss.Color("#9ece6a") // Success
	colorPeach   = lipgloss.Color("#ff9e64") // Warm accent
	colorRed     = lipgloss.Color("#f7768e") // Soft red
	colorBlue    = lipgloss.Color("#2ac3de") // Cyan
	colorMauve   = lipgloss.Color("#bb9af7") // Mauve
	colorYellow  = lipgloss.Color("#e0af68") // Gold
	colorTeal    = lipgloss.Color("#1abc9c") // Teal
)

// ─── Layout Styles ──────────────────────────────────────────────────────────

var (
	// AppStyle is the outer container for all non-chat screens.
	AppStyle = lipgloss.NewStyle().
		Foreground(colorText).
		Padding(1, 2)

	// HeaderStyle for screen titles.
	HeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorPrimary).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(colorOverlay).
		PaddingBottom(1).
		MarginBottom(1)

	// HelpStyle for keyboard hint lines.
	HelpStyle = lipgloss.NewStyle().
		Foreground(colorSubtext).
		MarginTop(1)

	// ErrorStyle for error messages.
	ErrorStyle = lipgloss.NewStyle().
		Foreground(colorRed).
		Bold(true).
		Padding(0, 1)
)

// ─── Dashboard Styles ───────────────────────────────────────────────────────

var (
	// MenuItemStyle for unselected menu items.
	MenuItemStyle = lipgloss.NewStyle().
		Foreground(colorText).
		PaddingLeft(2)

	// MenuSelectedStyle for the currently highlighted menu item.
	MenuSelectedStyle = lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		PaddingLeft(1)

	// TitleStyle for section titles within a screen.
	TitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorMauve).
		MarginBottom(1)

	// LogoStyle for the dscli ASCII art frame.
	LogoStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorPrimary).
		Border(lipgloss.DoubleBorder()).
		BorderForeground(colorOverlay).
		Padding(0, 2).
		MarginBottom(1)
)

// ─── List Styles ────────────────────────────────────────────────────────────

var (
	// ListItemStyle for unselected list items.
	ListItemStyle = lipgloss.NewStyle().
		Foreground(colorText).
		PaddingLeft(2)

	// ListSelectedStyle for the currently selected list item.
	ListSelectedStyle = lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		PaddingLeft(1)

	// TimestampStyle for metadata (dates, IDs, counts).
	TimestampStyle = lipgloss.NewStyle().
		Foreground(colorSubtext).
		Italic(true)

	// ContentPreviewStyle for truncated content previews.
	ContentPreviewStyle = lipgloss.NewStyle().
		Foreground(colorSubtext).
		PaddingLeft(4)

	// NoDataStyle for empty-state messages.
	NoDataStyle = lipgloss.NewStyle().
		Foreground(colorSubtext).
		Italic(true).
		PaddingLeft(2).
		MarginTop(1)
)

// ─── Detail Styles ──────────────────────────────────────────────────────────

var (
	// SectionHeadingStyle for detail section headers.
	SectionHeadingStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorMauve).
		MarginTop(1).
		MarginBottom(1)

	// DetailLabelStyle for field labels (right-aligned, fixed width).
	DetailLabelStyle = lipgloss.NewStyle().
		Foreground(colorSubtext).
		Width(14).
		Align(lipgloss.Right).
		PaddingRight(1)

	// DetailValueStyle for field values.
	DetailValueStyle = lipgloss.NewStyle().
		Foreground(colorText)

	// DetailContentStyle for multi-line content sections.
	DetailContentStyle = lipgloss.NewStyle().
		Foreground(colorText).
		PaddingLeft(2)
)

// ─── Chat Styles ────────────────────────────────────────────────────────────

var (
	// ChatInputStyle for the chat input field border.
	ChatInputStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(colorPrimary).
		Foreground(colorText).
		Padding(0, 1).
		MarginTop(1)

	// ChatLoadingStyle for the "AI is thinking..." indicator.
	ChatLoadingStyle = lipgloss.NewStyle().
		Foreground(colorSubtext).
		Italic(true).
		PaddingLeft(2)

	// ChatRoleUserStyle for "You:" role labels.
	ChatRoleUserStyle = lipgloss.NewStyle().
		Foreground(colorGreen).
		Bold(true)

	// ChatRoleAssistantStyle for "AI:" role labels.
	ChatRoleAssistantStyle = lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true)

	SpinnerStyle = lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true)

	SpinnerDoneStyle = lipgloss.NewStyle().
		Foreground(colorGreen).
		Bold(true)
)

// ─── Chat Bubble Styles ─────────────────────────────────────────────────────

var (
	// UserBubbleBase for user message bubbles. Call .MaxWidth(w) at render time.
	UserBubbleBase = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorGreen).
		Padding(0, 1)

	// AssistantBubbleBase for assistant message bubbles.
	AssistantBubbleBase = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(0, 1)

	// ThinkBubbleBase for reasoning/thinking bubbles (mauve border).
	ThinkBubbleBase = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorMauve).
		Padding(0, 1)

	// ThinkLineStyle for reasoning/thinking content (italic, dim).
	ThinkLineStyle = lipgloss.NewStyle().
		Foreground(colorSubtext).
		Italic(true)

	// ToolLineStyle for tool-call result lines (yellow, italic).
	ToolLineStyle = lipgloss.NewStyle().
		Foreground(colorYellow).
		Italic(true)

	// TruncationWarnBubble for truncation warnings (red, bold, centered).
	TruncationWarnBubble = lipgloss.NewStyle().
		Foreground(colorRed).
		Bold(true)

	// ── Unified bubble internal styles ─────────────────────────────────

	// AssistantBodyStyle: white/bold for assistant's final answer.
	AssistantBodyStyle = lipgloss.NewStyle().
		Foreground(colorText).
		Bold(true)

	// ThinkBodyStyle: italic dim for thinking content inside unified bubble.
	ThinkBodyStyle = lipgloss.NewStyle().
		Foreground(colorSubtext).
		Italic(true)

	// ToolBodyStyle: yellow/italic for tool results inside unified bubble.
	ToolBodyStyle = lipgloss.NewStyle().
		Foreground(colorYellow).
		Italic(true)

	// StateBodyStyle: subtle for session-state lines.
	StateBodyStyle = lipgloss.NewStyle().
		Foreground(colorSubtext).
		Italic(true)

	// TruncationBodyStyle: red/bold for truncation warning inside bubble.
	TruncationBodyStyle = lipgloss.NewStyle().
		Foreground(colorRed).
		Bold(true)
)

// BubbleMaxPercent is the maximum bubble width as a percentage of available
// content width (excludes borders and padding).
const BubbleMaxPercent = 72

// ─── Status Badge Styles ────────────────────────────────────────────────────

var (
	// BadgeSuccessStyle for success badges.
	BadgeSuccessStyle = lipgloss.NewStyle().
		Foreground(colorGreen).
		Bold(true)

	// BadgeWarnStyle for warning badges.
	BadgeWarnStyle = lipgloss.NewStyle().
		Foreground(colorYellow).
		Bold(true)
)

// ─── Status Bar Styles ──────────────────────────────────────────────────────

var (
	// StatusBarBg is the full-width bar background.
	StatusBarBg = lipgloss.NewStyle().
		Background(colorSurface)

	// StatusVersion is the version badge (mauve bg, dark text).
	StatusVersion = lipgloss.NewStyle().
		Background(colorMauve).
		Foreground(colorBase).
		Bold(true).
		Padding(0, 1)

	// StatusLabel for labels like 📁 project / 🤖 model.
	StatusLabel = lipgloss.NewStyle().
		Foreground(colorSubtext)

	// StatusSep is the separator between sections.
	StatusSep = lipgloss.NewStyle().
		Foreground(colorOverlay)

	// StatusScreen is the current screen name (primary accent).
	StatusScreen = lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		Padding(0, 1)
)

// ─── Bubble Rendering ───────────────────────────────────────────────────────

// RenderBubble renders a chat bubble with proper word-wrapping.
//
// In lipgloss v1.1.0, MaxWidth incorrectly truncates content instead of
// wrapping it. We work around this by using Width() (which wraps correctly)
// but only when the content exceeds contentAreaW. Short content skips Width
// so the bubble shrinks to fit naturally.
//
// Parameters:
//   - base:     bubble border style (e.g. UserBubbleBase)
//   - prefix:   leading text like "  👤 " (empty for thinking bubbles)
//   - content:  the message text (may contain embedded newlines)
//   - wrapStyle: pre-built Width(contentAreaW) style for wrapping
//   - contentAreaW: max text area width (bubbleMaxW - border - padding)
func RenderBubble(base lipgloss.Style, prefix, content string, wrapStyle lipgloss.Style, contentAreaW int) string {
	fullText := prefix + content

	// Fast path: content fits without wrapping → render as-is (bubble shrinks).
	needsWrap := false
	for _, line := range strings.Split(fullText, "\n") {
		if lipgloss.Width(line) > contentAreaW {
			needsWrap = true
			break
		}
	}
	if !needsWrap {
		return base.Render(fullText)
	}

	// Slow path: content needs wrapping.
	wrappedContent := wrapStyle.Render(content)
	wrappedLines := strings.Split(wrappedContent, "\n")

	// Attach the prefix to the first wrapped line if it fits.
	if prefix != "" && len(wrappedLines) > 0 {
		firstLine := wrappedLines[0]
		if lipgloss.Width(prefix+firstLine) <= contentAreaW {
			wrappedLines[0] = prefix + firstLine
		} else {
			// Prefix doesn't fit → give it its own line.
			wrappedLines = append([]string{prefix}, wrappedLines...)
		}
	}

	return base.Render(strings.Join(wrappedLines, "\n"))
}

// ─── Padding Helpers ────────────────────────────────────────────────────────
// These use plain spaces instead of lipgloss alignment to avoid
// ANSI-on-ANSI rendering corruption that caused top-border clipping.

// PadRight returns s with each line right-aligned within width w,
// using plain spaces for 2-char left/right margins.
func PadRight(s string, w int) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lw := lipgloss.Width(line)
		left := w - 4 - lw // 2 margin + text + 2 margin = w
		if left < 0 {
			left = 0
		}
		lines[i] = strings.Repeat(" ", 2+left) + line + "  "
	}
	return strings.Join(lines, "\n")
}

// PadLeft returns s with each line left-aligned within width w,
// using plain spaces for 2-char left/right margins.
func PadLeft(s string, w int) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lw := lipgloss.Width(line)
		right := w - 4 - lw
		if right < 0 {
			right = 0
		}
		lines[i] = "  " + line + strings.Repeat(" ", 2+right)
	}
	return strings.Join(lines, "\n")
}

// PadCenter returns s with each line center-aligned within width w,
// using plain spaces for 2-char left/right margins.
func PadCenter(s string, w int) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lw := lipgloss.Width(line)
		totalPad := w - 4 - lw
		if totalPad < 0 {
			totalPad = 0
		}
		left := totalPad / 2
		right := totalPad - left
		lines[i] = strings.Repeat(" ", 2+left) + line + strings.Repeat(" ", 2+right)
	}
	return strings.Join(lines, "\n")
}

// ─── Utility Helpers ────────────────────────────────────────────────────────

// TruncateStr truncates a string to max runes, replacing newlines with spaces.
func TruncateStr(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}

// ShortenPath replaces the home directory prefix with ~ for compact display.
func ShortenPath(p string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	if strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	// Fallback: show only the last two components (parent/base).
	base := filepath.Base(p)
	parent := filepath.Base(filepath.Dir(p))
	if parent != "." && parent != "/" && parent != "" {
		return parent + "/" + base
	}
	return base
}

// ModelDisplayName returns the currently configured chat model name.
func ModelDisplayName() string {
	return "deepseek-chat"
}
