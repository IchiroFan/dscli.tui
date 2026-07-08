// Package tui implements the Bubble Tea application for dscli.tui.
//
// Styles are managed through a theme system.  At startup, initStyles is called
// (once during package init with Tokyo Night, and again by New() if the user's
// config specifies a different theme).  All style variables are recomputed from
// the active Colors palette.
package tui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ─── Theme Colors ─────────────────────────────────────────────────────────

// Colors holds the complete color palette for a theme.
type Colors struct {
	Base    lipgloss.Color // Dark background
	Surface lipgloss.Color // Panel background
	Overlay lipgloss.Color // Muted borders
	Text    lipgloss.Color // Light text
	Subtext lipgloss.Color // Dim text
	Primary lipgloss.Color // Primary accent
	Green   lipgloss.Color // Success
	Peach   lipgloss.Color // Warm accent
	Red     lipgloss.Color // Soft red
	Blue    lipgloss.Color // Cyan
	Mauve   lipgloss.Color // Mauve
	Yellow  lipgloss.Color // Gold
	Teal    lipgloss.Color // Teal
}

// Theme definitions (5 built-in color schemes).
var (
	// ThemeTokyoNight is the default dark theme with deep blue-purple tones.
	ThemeTokyoNight = Colors{
		Base:    "#1a1b26",
		Surface: "#24253e",
		Overlay: "#565f89",
		Text:    "#c0caf5",
		Subtext: "#9aa5ce",
		Primary: "#7aa2f7",
		Green:   "#9ece6a",
		Peach:   "#ff9e64",
		Red:     "#f7768e",
		Blue:    "#2ac3de",
		Mauve:   "#bb9af7",
		Yellow:  "#e0af68",
		Teal:    "#1abc9c",
	}

	// ThemeDracula is a dark purple-themed palette inspired by the Dracula scheme.
	ThemeDracula = Colors{
		Base:    "#282a36",
		Surface: "#44475a",
		Overlay: "#6272a4",
		Text:    "#f8f8f2",
		Subtext: "#b0b0c0",
		Primary: "#bd93f9",
		Green:   "#50fa7b",
		Peach:   "#ffb86c",
		Red:     "#ff5555",
		Blue:    "#8be9fd",
		Mauve:   "#ff79c6",
		Yellow:  "#f1fa8c",
		Teal:    "#66d9ef",
	}

	// ThemeMonokai is a high-contrast dark palette inspired by Monokai.
	ThemeMonokai = Colors{
		Base:    "#272822",
		Surface: "#3e3d32",
		Overlay: "#75715e",
		Text:    "#f8f8f2",
		Subtext: "#a7a58a",
		Primary: "#66d9ef",
		Green:   "#a6e22e",
		Peach:   "#fd971f",
		Red:     "#f92672",
		Blue:    "#89d7f8",
		Mauve:   "#ae81ff",
		Yellow:  "#e6db74",
		Teal:    "#a1efe4",
	}

	// ThemeNord is an arctic blue-themed palette inspired by Nord.
	ThemeNord = Colors{
		Base:    "#2e3440",
		Surface: "#3b4252",
		Overlay: "#4c566a",
		Text:    "#d8dee9",
		Subtext: "#7f8c9d",
		Primary: "#88c0d0",
		Green:   "#a3be8c",
		Peach:   "#d08770",
		Red:     "#bf616a",
		Blue:    "#8fbcbb",
		Mauve:   "#b48ead",
		Yellow:  "#ebcb8b",
		Teal:    "#81a1c1",
	}

	// ThemeSolarizedLight is a light theme with warm, low-contrast tones.
	ThemeSolarizedLight = Colors{
		Base:    "#fdf6e3",
		Surface: "#eee8d5",
		Overlay: "#93a1a1",
		Text:    "#657b83",
		Subtext: "#839496",
		Primary: "#268bd2",
		Green:   "#859900",
		Peach:   "#cb4b16",
		Red:     "#dc322f",
		Blue:    "#2aa198",
		Mauve:   "#6c71c4",
		Yellow:  "#b58900",
		Teal:    "#00a0a0",
	}
)

// themeByName maps user-facing names to Colors.
var themeByName = map[string]Colors{
	"tokyo-night":     ThemeTokyoNight,
	"dracula":         ThemeDracula,
	"monokai":         ThemeMonokai,
	"nord":            ThemeNord,
	"solarized-light": ThemeSolarizedLight,
}

// ─── Color variables (declared, set by initStyles) ────────────────────────

var (
	colorBase, colorSurface, colorOverlay lipgloss.Color
	colorText, colorSubtext               lipgloss.Color
	colorPrimary, colorGreen, colorPeach  lipgloss.Color
	colorRed, colorBlue, colorMauve       lipgloss.Color
	colorYellow, colorTeal                lipgloss.Color
)

// ─── Style variables (declared, set by initStyles) ────────────────────────

var (
	// ── Layout ──────────────────────────────────────────────────────────
	AppStyle     lipgloss.Style
	HeaderStyle  lipgloss.Style
	HelpStyle    lipgloss.Style
	ErrorStyle   lipgloss.Style

	// ── Dashboard / Menu ────────────────────────────────────────────────
	MenuItemStyle     lipgloss.Style
	MenuSelectedStyle lipgloss.Style
	MenuDescStyle     lipgloss.Style
	TitleStyle        lipgloss.Style
	LogoStyle         lipgloss.Style

	// ── List ───────────────────────────────────────────────────────────
	ListItemStyle      lipgloss.Style
	ListSelectedStyle  lipgloss.Style
	TimestampStyle     lipgloss.Style
	ContentPreviewStyle lipgloss.Style
	NoDataStyle        lipgloss.Style
	PageInfoStyle      lipgloss.Style

	// ── Detail ──────────────────────────────────────────────────────────
	SectionHeadingStyle lipgloss.Style
	DetailLabelStyle    lipgloss.Style
	DetailValueStyle    lipgloss.Style
	DetailContentStyle  lipgloss.Style

	// ── Chat (non-bubble) ─────────────────────────────────────────────
	ChatLoadingStyle       lipgloss.Style
	ChatRoleUserStyle      lipgloss.Style
	ChatRoleAssistantStyle lipgloss.Style
	SpinnerStyle           lipgloss.Style
	SpinnerDoneStyle       lipgloss.Style

	// ── Chat Bubbles ──────────────────────────────────────────────────
	UserBubbleBase      lipgloss.Style
	AssistantBubbleBase lipgloss.Style
	ThinkBubbleBase     lipgloss.Style
	ThinkLineStyle      lipgloss.Style
	ToolLineStyle       lipgloss.Style
	TruncationWarnBubble lipgloss.Style

	// ── Unified bubble inner styles ───────────────────────────────────
	AssistantBodyStyle  lipgloss.Style
	ThinkBodyStyle      lipgloss.Style
	ToolBodyStyle       lipgloss.Style
	StateBodyStyle      lipgloss.Style
	TruncationBodyStyle lipgloss.Style

	// ── Badges ──────────────────────────────────────────────────────────
	BadgeSuccessStyle lipgloss.Style
	BadgeWarnStyle    lipgloss.Style

	// ── Status Bar ─────────────────────────────────────────────────────
	StatusBarBg   lipgloss.Style
	StatusVersion lipgloss.Style
	StatusLabel   lipgloss.Style
	StatusSep     lipgloss.Style
	StatusScreen  lipgloss.Style
)

// initStyles sets all color and style variables from the given Colors palette.
// Called once during package init (with Tokyo Night) and again by New() if
// the user's config specifies a different theme.
func initStyles(c Colors) {
	// ── Set color vars ───────────────────────────────────────────────
	colorBase = c.Base
	colorSurface = c.Surface
	colorOverlay = c.Overlay
	colorText = c.Text
	colorSubtext = c.Subtext
	colorPrimary = c.Primary
	colorGreen = c.Green
	colorPeach = c.Peach
	colorRed = c.Red
	colorBlue = c.Blue
	colorMauve = c.Mauve
	colorYellow = c.Yellow
	colorTeal = c.Teal

	// ── Layout Styles ────────────────────────────────────────────────

	AppStyle = lipgloss.NewStyle().
		Background(colorBase).
		Foreground(colorText).
		Padding(1, 2)

	HeaderStyle = lipgloss.NewStyle().
		Background(colorBase).
		Bold(true).
		Foreground(colorPrimary).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(colorOverlay).
		PaddingBottom(1).
		MarginBottom(1)

	HelpStyle = lipgloss.NewStyle().
		Background(colorBase).
		Foreground(colorSubtext).
		MarginTop(1)

	ErrorStyle = lipgloss.NewStyle().
		Background(colorBase).
		Foreground(colorRed).
		Bold(true).
		Padding(0, 1)
	// ── Dashboard / Menu Styles ──────────────────────────────────────

	MenuItemStyle = lipgloss.NewStyle().
		Background(colorBase).
		Foreground(colorText).
		PaddingLeft(2)

	MenuSelectedStyle = lipgloss.NewStyle().
		Background(colorBase).
		Foreground(colorPrimary).
		Bold(true).
		PaddingLeft(1)

	MenuDescStyle = lipgloss.NewStyle().
		Background(colorBase).
		Foreground(colorSubtext)

	TitleStyle = lipgloss.NewStyle().
		Background(colorBase).
		Bold(true).
		Foreground(colorMauve).
		MarginBottom(1)

	LogoStyle = lipgloss.NewStyle().
		Background(colorBase).
		Bold(true).
		Foreground(colorPrimary).
		Border(lipgloss.DoubleBorder()).
		BorderForeground(colorOverlay).
		Padding(0, 2).
		MarginBottom(1)

	// ── List Styles ──────────────────────────────────────────────────

	ListItemStyle = lipgloss.NewStyle().
		Background(colorBase).
		Foreground(colorText).
		PaddingLeft(2)

	ListSelectedStyle = lipgloss.NewStyle().
		Background(colorBase).
		Foreground(colorPrimary).
		Bold(true).
		PaddingLeft(1)

	TimestampStyle = lipgloss.NewStyle().
		Background(colorBase).
		Foreground(colorSubtext).
		Italic(true)

	ContentPreviewStyle = lipgloss.NewStyle().
		Background(colorBase).
		Foreground(colorSubtext).
		PaddingLeft(4)

	NoDataStyle = lipgloss.NewStyle().
		Background(colorBase).
		Foreground(colorSubtext).
		Italic(true).
		PaddingLeft(2).
		MarginTop(1)

	PageInfoStyle = lipgloss.NewStyle().
		Background(colorBase).
		Foreground(colorMauve).
		Bold(true).
		PaddingLeft(2)

	// ── Detail Styles ────────────────────────────────────────────────

	SectionHeadingStyle = lipgloss.NewStyle().
		Background(colorBase).
		Bold(true).
		Foreground(colorMauve).
		MarginTop(1).
		MarginBottom(1)

	DetailLabelStyle = lipgloss.NewStyle().
		Background(colorBase).
		Foreground(colorSubtext).
		Width(14).
		Align(lipgloss.Right).
		PaddingRight(1)

	DetailValueStyle = lipgloss.NewStyle().
		Background(colorBase).
		Foreground(colorText)

	DetailContentStyle = lipgloss.NewStyle().
		Background(colorBase).
		Foreground(colorText).
		PaddingLeft(2)

	// ── Chat (non-bubble) Styles ─────────────────────────────────────

	ChatLoadingStyle = lipgloss.NewStyle().
		Background(colorBase).
		Foreground(colorSubtext).
		Italic(true).
		PaddingLeft(2)

	ChatRoleUserStyle = lipgloss.NewStyle().
		Background(colorBase).
		Foreground(colorGreen).
		Bold(true)

	ChatRoleAssistantStyle = lipgloss.NewStyle().
		Background(colorBase).
		Foreground(colorPrimary).
		Bold(true)

	SpinnerStyle = lipgloss.NewStyle().
		Background(colorBase).
		Foreground(colorPrimary).
		Bold(true)

	SpinnerDoneStyle = lipgloss.NewStyle().
		Background(colorBase).
		Foreground(colorGreen).
		Bold(true)

	// ── Chat Bubble Styles (Surface background for visual distinction) ───

	UserBubbleBase = lipgloss.NewStyle().
		Background(colorSurface).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorGreen).
		Padding(0, 1)

	AssistantBubbleBase = lipgloss.NewStyle().
		Background(colorSurface).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(0, 1)

	ThinkBubbleBase = lipgloss.NewStyle().
		Background(colorSurface).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorMauve).
		Padding(0, 1)

	ThinkLineStyle = lipgloss.NewStyle().
		Background(colorBase).
		Foreground(colorSubtext).
		Italic(true)

	ToolLineStyle = lipgloss.NewStyle().
		Background(colorBase).
		Foreground(colorYellow).
		Italic(true)

	TruncationWarnBubble = lipgloss.NewStyle().
		Background(colorBase).
		Foreground(colorRed).
		Bold(true)

	// ── Unified bubble inner styles ──────────────────────────────────

	AssistantBodyStyle = lipgloss.NewStyle().
		Background(colorBase).
		Foreground(colorText).
		Bold(true)

	ThinkBodyStyle = lipgloss.NewStyle().
		Background(colorBase).
		Foreground(colorSubtext).
		Italic(true)

	ToolBodyStyle = lipgloss.NewStyle().
		Background(colorBase).
		Foreground(colorYellow).
		Italic(true)

	StateBodyStyle = lipgloss.NewStyle().
		Background(colorBase).
		Foreground(colorSubtext).
		Italic(true)

	TruncationBodyStyle = lipgloss.NewStyle().
		Background(colorBase).
		Foreground(colorRed).
		Bold(true)

	// ── Status Badge Styles ──────────────────────────────────────────

	BadgeSuccessStyle = lipgloss.NewStyle().
		Background(colorBase).
		Foreground(colorGreen).
		Bold(true)

	BadgeWarnStyle = lipgloss.NewStyle().
		Background(colorBase).
		Foreground(colorYellow).
		Bold(true)

	// ── Status Bar Styles ────────────────────────────────────────────

	StatusBarBg = lipgloss.NewStyle().
		Background(colorSurface)

	StatusVersion = lipgloss.NewStyle().
		Background(colorMauve).
		Foreground(colorBase).
		Bold(true).
		Padding(0, 1)

	StatusLabel = lipgloss.NewStyle().
		Background(colorSurface).
		Foreground(colorSubtext)

	StatusSep = lipgloss.NewStyle().
		Background(colorSurface).
		Foreground(colorOverlay)

	StatusScreen = lipgloss.NewStyle().
		Background(colorSurface).
		Foreground(colorPrimary).
		Bold(true).
		Padding(0, 1)
}

// init applies the default Tokyo Night theme so all package-level style
// variables are properly initialized before any caller uses them.
func init() {
	initStyles(ThemeTokyoNight)
}

// BubbleMaxPercent is the maximum bubble width as a percentage of available
// content width (excludes borders and padding).
const BubbleMaxPercent = 72

// ─── Bubble Rendering ─────────────────────────────────────────────────────

// PadBubbleToWidth right-pads each line of a rendered bubble string with spaces
// so the total visual width equals targetW (excluding ANSI sequences).
// This ensures all bubbles on the same side have a stable right border position.
func PadBubbleToWidth(s string, targetW int) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lw := lipgloss.Width(line)
		if pad := targetW - lw; pad > 0 {
			lines[i] = line + strings.Repeat(" ", pad)
		}
	}
	return strings.Join(lines, "\n")
}

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

// ─── Padding Helpers ──────────────────────────────────────────────────────
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

// ─── Utility Helpers ──────────────────────────────────────────────────────

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

// ChatInputBaseStyle returns the base style for the chat textarea border.
// focused=true uses the primary blue border; focused=false uses a muted overlay.
func ChatInputBaseStyle(focused bool) lipgloss.Style {
	borderColor := colorPrimary
	if !focused {
		borderColor = colorOverlay
	}
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(borderColor).
		Foreground(colorText).
		Background(colorBase).
		Padding(0, 1)

}
