package tui

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/dscli/dscli.tui/internal/aiagent"
	"github.com/dscli/dscli.tui/internal/socket"
	"github.com/dscli/dscli.tui/internal/tui/protocol"
)

// SocketAskUserMsg is emitted by the socket bridge goroutine when a dscli
// ask_user request arrives via Unix socket.  The TUI enters ScreenAskUser,
// collects the user's answer, and writes it back via AskRequest.RespCh.
type SocketAskUserMsg struct {
	Request *socket.AskRequest
}

// ─── Update

// ─── Update ──────────────────────────────────────────────────────────

// Update implements tea.Model.Update.
func (m *RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle global messages first (window size, errors, quit).
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		// Resize chat/input widgets proportionally.
		inputWidth := msg.Width - 10
		if inputWidth < 10 {
			inputWidth = 10
		}
		m.chatInput.SetWidth(inputWidth)
		m.askInput.Width = inputWidth - 4
		return m, nil

	case tea.KeyMsg:
		// Global key: Ctrl+C always quits.
		if msg.String() == "ctrl+c" {
			m.screen = ScreenQuitting
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.MouseMsg:
		return m.handleMouseEvent(msg)

	case SocketAskUserMsg:
		// Route through Chat: append the question to chatHistory and set
		// askUserPending.  The user's next Enter in Chat will be routed
		// to askUserRespond instead of starting a new chat exchange.
		// This avoids the visual flashing caused by switching between
		// ScreenAskUser and ScreenChatting.
		m.chatHistory = append(m.chatHistory, ChatLine{
			Role:    "system",
			Content: "🤖 " + msg.Request.Question,
		})
		m.askUserPending = true
		respCh := msg.Request.RespCh
		prevWasChat := m.screen == ScreenChatting
		if !prevWasChat {
			m.prevScreen = m.screen
		}
		m.askUserRespond = func(answer string) tea.Cmd {
			respCh <- answer
			m.askUserPending = false
			m.askUserRespond = nil
			// Restore previous screen if we switched to Chat for this.
			if !prevWasChat {
				m.screen = m.prevScreen
				m.prevScreen = ScreenMainMenu
			}
			return nil
		}
		// Suppress chat loading state.
		m.chatLoading = false
		m.chatDone = false
		m.spinnerOn = false
		// Switch to Chat if not already there.
		if !prevWasChat {
			m.screen = ScreenChatting
			m.chatInput.Focus()
		}
		return m, nil
	}

	// Route by screen.
	switch m.screen {
	case ScreenMainMenu:
		return m.updateMainMenu(msg)
	case ScreenRunningCmd:
		return m.updateRunningCmd(msg)
	case ScreenShowOutput:
		return m.updateShowOutput(msg)
	case ScreenHistoryList:
		return m.updateHistoryList(msg)
	case ScreenChatting:
		return m.updateChatting(msg)
	case ScreenAskUser:
		return m.updateAskUser(msg)
	case ScreenSkillList:
		return m.updateSkillList(msg)
	case ScreenMemoryList:
		return m.updateMemoryList(msg)
	case ScreenToolList:
		return m.updateToolList(msg)
	case ScreenProjectList:
		return m.updateProjectList(msg)
	case ScreenQuitting:
		return m, tea.Quit

	default:
		return m, nil
	}
}

// handleMouseEvent processes mouse events, dispatching wheel events to the
func (m *RootModel) handleMouseEvent(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		switch m.screen {
		case ScreenShowOutput:
			// Recalculate scroll bounds (matching keyboard handler behavior).
			availableHeight := m.Height - 5
			if availableHeight < 3 {
				availableHeight = 3
			}
			totalLines := len(m.outputLines)
			if totalLines > availableHeight {
				m.outputScrollMax = totalLines - availableHeight
			} else {
				m.outputScrollMax = 0
			}
			if m.outputScroll > 0 {
				m.outputScroll--
			}
		case ScreenChatting:
			if m.chatScroll < m.chatScrollMax {
				m.chatScroll++
			}
		case ScreenHistoryList, ScreenSkillList, ScreenMemoryList, ScreenToolList, ScreenProjectList:
			// Wheel events are not applied to selectable lists (use keyboard).
		}

	case tea.MouseButtonWheelDown:
		switch m.screen {
		case ScreenShowOutput:
			availableHeight := m.Height - 5
			if availableHeight < 3 {
				availableHeight = 3
			}
			totalLines := len(m.outputLines)
			if totalLines > availableHeight {
				m.outputScrollMax = totalLines - availableHeight
			} else {
				m.outputScrollMax = 0
			}
			if m.outputScroll < m.outputScrollMax {
				m.outputScroll++
			}
		case ScreenChatting:
			if m.chatScroll > 0 {
				m.chatScroll--
			}
		case ScreenHistoryList, ScreenSkillList, ScreenMemoryList, ScreenToolList, ScreenProjectList:
			// Wheel events are not applied to selectable lists (use keyboard).
		}

	}
	return m, nil
}

// ─── Main Menu ───────────────────────────────────────────────────────

func (m *RootModel) updateMainMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.menuCursor > 0 {
				m.menuCursor--
			}
			return m, nil

		case "down", "j":
			if m.menuCursor < len(m.menuItems)-1 {
				m.menuCursor++
			}
			return m, nil

		case "enter", " ":
			return m.executeSelected()

		case "q":
			m.screen = ScreenQuitting
			return m, tea.Quit
		}
	}

	// Handle navigation back to menu (from other states).
	switch msg.(type) {
	case navBackToMenuMsg:
		m.screen = ScreenMainMenu
		m.err = nil
		m.historyItems = nil
		m.historyCursor = 0
		m.historyPage = 0
		m.skillItems = nil
		m.skillCursor = 0
		m.memoryItems = nil
		m.memoryCursor = 0
		m.memorySearchQuery = ""

		m.toolItems = nil
		m.toolCursor = 0
		m.toolPage = 0
		m.projectItems = nil
		m.projectCursor = 0
		m.projectRemovePendingID = ""
	}

	return m, nil
}

// executeSelected dispatches the appropriate Cmd for the selected menu item.
func (m *RootModel) executeSelected() (tea.Model, tea.Cmd) {
	idx := m.menuCursor
	if idx < 0 || idx >= len(m.menuItems) {
		return m, nil
	}

	switch idx {
	case 0: // Chat
		m.screen = ScreenChatting
		m.chatHistory = nil
		m.chatInput.SetValue("")
		m.chatInput.Focus()
		m.spinnerOn = true
		m.chatLoading = true
		m.chatReady = false
		m.chatDone = false
		m.chatScroll = 0
		m.chatScrollMax = 0
		return m, cmdStartChat(m.agent, nil)

	case 1: // Balance
		m.cmdTitle = "📊 Balance"
		m.screen = ScreenRunningCmd
		return m, cmdBalance(m.agent)

	case 2: // Models
		m.cmdTitle = "🤖 Models"
		m.screen = ScreenRunningCmd
		return m, cmdModels(m.agent)

	case 3: // Version
		m.cmdTitle = "ℹ️  Version"
		m.screen = ScreenRunningCmd
		return m, cmdVersion(m.agent)

	case 4: // Flycheck
		m.cmdTitle = "🔍 Flycheck"
		m.screen = ScreenRunningCmd
		return m, cmdFlycheck(m.agent)
	case 5: // History
		m.historyItems = nil
		m.historyCursor = -1
		m.historyPage = 0
		m.screen = ScreenRunningCmd
		return m, cmdSubcommand(m.agent, m.agent.History, "list", "history", "--json", "--histsize", "100")

	case 6: // Skill
		m.skillItems = nil
		m.skillCursor = -1
		m.screen = ScreenRunningCmd
		return m, cmdSubcommand(m.agent, m.agent.Skill, "list", "skill")

	case 7: // Memory
		m.memoryItems = nil
		m.memoryCursor = -1
		m.screen = ScreenRunningCmd
		return m, cmdSubcommand(m.agent, m.agent.Memory, "list", "memory")
	case 8: // Project
		m.projectItems = nil
		m.projectCursor = -1
		m.projectRemovePendingID = ""
		m.cmdTitle = "📁 Project"
		m.screen = ScreenRunningCmd
		return m, cmdSubcommand(m.agent, m.agent.Project, "list", "project")

	case 9: // Role
		m.cmdTitle = "👤 Role"
		m.screen = ScreenRunningCmd
		return m, cmdSubcommand(m.agent, m.agent.Role, "list", "role")

	case 10: // Tool
		m.toolItems = nil
		m.toolCursor = -1
		m.toolPage = 0
		m.cmdTitle = "🧰 Tool"
		m.screen = ScreenRunningCmd
		return m, cmdSubcommand(m.agent, m.agent.Tool, "list", "tool")

	case 11: // Mail
		m.cmdTitle = "✉️  Mail"
		m.screen = ScreenRunningCmd
		return m, cmdSubcommand(m.agent, m.agent.Mail, "list", "mail")

	case 12: // Service
		m.cmdTitle = "🔧 Service"
		m.screen = ScreenRunningCmd
		return m, cmdSubcommand(m.agent, m.agent.Service, "list", "service")

	case 13: // Quit
		m.screen = ScreenQuitting
		return m, tea.Quit

	default:
		return m, nil
	}
}

func (m *RootModel) updateRunningCmd(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok && msg.String() == "esc" {
		m.cmdTitle = ""
		m.screen = ScreenMainMenu
		m.err = nil
		return m, nil
	}

	switch msg := msg.(type) {
	case aiagent.BalanceResultMsg:
		m.showOutput(formatCommandResult(msg.Payload, msg.Err),
			msg.Err == nil && msg.Payload != nil && msg.Payload.Success)
		return m, nil

	case aiagent.ModelsResultMsg:
		m.showOutput(formatCommandResult(msg.Payload, msg.Err),
			msg.Err == nil && msg.Payload != nil && msg.Payload.Success)
		return m, nil

	case aiagent.VersionResultMsg:
		cmdOut := formatCommandResult(msg.Payload, msg.Err)
		success := msg.Err == nil && msg.Payload != nil && msg.Payload.Success
		if success {
			m.dscliVersion = strings.TrimSpace(msg.Payload.Data)
			// Keep only the first line (version summary).
			if idx := strings.Index(m.dscliVersion, "\n"); idx > 0 {
				m.dscliVersion = m.dscliVersion[:idx]
			}
		}
		m.showOutput(cmdOut, success)
		return m, nil

	case aiagent.FlycheckResultMsg:
		m.showOutput(formatCommandResult(msg.Payload, msg.Err),
			msg.Err == nil && msg.Payload != nil && msg.Payload.Success)
		return m, nil

	case aiagent.MemorySearchResultMsg:
		// Dedicated handler for memory search results.
		m.memorySearchPayload(msg.Payload)
		return m, nil

	case aiagent.SubcommandResultMsg:
		// Route list results to specific handlers based on group.
		if msg.Subcmd == "list" && msg.Group == "skill" && m.skillListPayload(msg.Payload) {
			// Parsed as skill items — transition to ScreenSkillList.
		} else if msg.Subcmd == "list" && msg.Group == "memory" && m.memoryListPayload(msg.Payload) {
			// Parsed as memory items — transition to ScreenMemoryList.
		} else if msg.Subcmd == "list" && msg.Group == "tool" && m.toolListPayload(msg.Payload) {
			// Parsed as tool items — transition to ScreenToolList.
		} else if msg.Subcmd == "list" && msg.Group == "project" && m.projectListPayload(msg.Payload) {
			// Parsed as project items — transition to ScreenProjectList.
		} else if msg.Subcmd == "list" && m.historyListPayload(msg.Payload) {
			// Parsed as history items — transition to ScreenHistoryList.
		} else if msg.Subcmd == "show" && msg.Group == "tool" {
			m.prevScreen = ScreenToolList
			m.showOutput(formatCommandResult(msg.Payload, msg.Err),
				msg.Err == nil && msg.Payload != nil && msg.Payload.Success)
		} else if msg.Subcmd == "show" && msg.Group == "skill" {
			m.prevScreen = ScreenSkillList
			m.showOutput(formatCommandResult(msg.Payload, msg.Err),
				msg.Err == nil && msg.Payload != nil && msg.Payload.Success)
		} else if msg.Subcmd == "show" && msg.Group == "memory" {
			m.prevScreen = ScreenMemoryList
			m.showOutput(formatCommandResult(msg.Payload, msg.Err),
				msg.Err == nil && msg.Payload != nil && msg.Payload.Success)
		} else if msg.Subcmd == "show" {
			m.prevScreen = ScreenHistoryList
			m.showOutput(formatCommandResult(msg.Payload, msg.Err),
				msg.Err == nil && msg.Payload != nil && msg.Payload.Success)
		} else if msg.Subcmd == "remove" && msg.Group == "project" {
			if msg.Err == nil && msg.Payload != nil && msg.Payload.Success {
				// Success: re-run list to refresh.
				m.projectItems = nil
				m.projectCursor = -1
				m.projectRemovePendingID = ""
				m.cmdTitle = "📁 Project"
				return m, cmdSubcommand(m.agent, m.agent.Project, "list", "project")
			}
			// Failure: show error, then back to project list on Esc.
			m.prevScreen = ScreenProjectList
			m.showOutput(formatCommandResult(msg.Payload, msg.Err),
				msg.Err == nil && msg.Payload != nil && msg.Payload.Success)
		} else {

			m.showOutput(formatCommandResult(msg.Payload, msg.Err),
				msg.Err == nil && msg.Payload != nil && msg.Payload.Success)
		}

		return m, nil
	// Catch-all: if we receive any other message while running, ignore.
	default:
		return m, nil
	}
}

func (m *RootModel) historyListPayload(p *protocol.CommandResultPayload) bool {
	if p == nil || !p.Success || p.Data == "" {
		return false
	}
	items := parseHistoryList(p.Data)
	if len(items) == 0 {
		return false
	}
	// Reverse items so newest appears first (dscli returns ascending order).
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
	m.historyItems = items
	m.historyCursor = 0 // select first (newest) item
	m.historyPage = 0   // start from first page
	m.screen = ScreenHistoryList
	return true
}

// histJSON mirrors the JSON structure from "dscli history list --json".
type histJSON struct {
	ID               int64  `json:"id"`
	Role             string `json:"role"`
	OK               bool   `json:"ok"`
	CreatedAt        string `json:"created_at"`
	Content          string `json:"content"`
	ReasoningContent string `json:"reasoning_content"`
}

// parseHistoryList parses "dscli history list --json" output into HistoryItem slice.
func parseHistoryList(data string) []HistoryItem {
	var entries []histJSON
	if err := json.Unmarshal([]byte(data), &entries); err != nil {
		return nil
	}
	items := make([]HistoryItem, 0, len(entries))
	for _, e := range entries {
		done := "false"
		if e.OK {
			done = "true"
		}
		items = append(items, HistoryItem{
			ID:               fmt.Sprint(e.ID),
			Role:             e.Role,
			Done:             done,
			CreatedAt:        e.CreatedAt,
			ReasoningContent: e.ReasoningContent,
			Content:          e.Content,
		})
	}
	return items
}

// showOutput transitions to ScreenShowOutput with pre-split lines for scrolling.
func (m *RootModel) showOutput(cmdOutput string, cmdSuccess bool) {
	m.cmdOutput = cmdOutput
	m.cmdSuccess = cmdSuccess
	m.outputLines = strings.Split(cmdOutput, "\n")
	m.outputScroll = 0
	m.outputScrollMax = 0
	m.screen = ScreenShowOutput
}

// ─── History List (selectable) ──────────────────────────────────

// updateHistoryList handles keyboard input on the history selection screen.
func (m *RootModel) updateHistoryList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Compute page size (fixed 20, capped by terminal height).
		pageSize := 20
		maxRows := m.Height - 7
		if maxRows < 3 {
			maxRows = 3
		}
		if maxRows < pageSize {
			pageSize = maxRows
		}
		totalPages := (len(m.historyItems) + pageSize - 1) / pageSize
		if totalPages < 1 {
			totalPages = 1
		}

		switch msg.String() {
		case "up", "k":
			if m.historyCursor > 0 {
				m.historyCursor--
				// If cursor moved before current page, flip to previous page.
				start := m.historyPage * pageSize
				if m.historyCursor < start && m.historyPage > 0 {
					m.historyPage--
					newStart := m.historyPage * pageSize
					m.historyCursor = newStart + pageSize - 1
					if m.historyCursor >= len(m.historyItems) {
						m.historyCursor = len(m.historyItems) - 1
					}
				}
			}
			return m, nil

		case "down", "j":
			if m.historyCursor < len(m.historyItems)-1 {
				m.historyCursor++
				// If cursor moved past current page, flip to next page.
				end := (m.historyPage + 1) * pageSize
				if m.historyCursor >= end {
					if m.historyPage < totalPages-1 {
						m.historyPage++
						m.historyCursor = m.historyPage * pageSize
					} else {
						m.historyCursor = end - 1 // restore
					}
				}
			}
			return m, nil

		case "pgup":
			if m.historyPage > 0 {
				m.historyPage--
				m.historyCursor = m.historyPage * pageSize
			}
			return m, nil

		case "pgdown":
			if m.historyPage < totalPages-1 {
				m.historyPage++
				m.historyCursor = m.historyPage * pageSize
			}
			return m, nil

		case "home", "g":
			m.historyPage = 0
			m.historyCursor = 0
			return m, nil

		case "end", "G":
			m.historyPage = totalPages - 1
			m.historyCursor = m.historyPage * pageSize
			return m, nil

		case "enter", " ":
			if m.historyCursor < 0 || m.historyCursor >= len(m.historyItems) {
				return m, nil
			}
			id := m.historyItems[m.historyCursor].ID
			m.screen = ScreenRunningCmd
			return m, cmdSubcommand(m.agent, m.agent.History, "show", "history", id)

		case "esc", "q":
			m.historyItems = nil
			m.historyCursor = 0
			m.historyPage = 0
			m.screen = ScreenMainMenu
			m.err = nil
			return m, nil
		}
	}
	return m, nil
}

// ─── Skill List (selectable) ────────────────────────────────────

// updateSkillList handles keyboard input on the skill selection screen.
func (m *RootModel) updateSkillList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.skillCursor > 0 {
				m.skillCursor--
			}
			return m, nil

		case "down", "j":
			if m.skillCursor < len(m.skillItems)-1 {
				m.skillCursor++
			}
			return m, nil

		case "enter", " ":
			if m.skillCursor < 0 || m.skillCursor >= len(m.skillItems) {
				return m, nil
			}
			name := m.skillItems[m.skillCursor].Name
			m.screen = ScreenRunningCmd
			return m, cmdSubcommand(m.agent, m.agent.Skill, "show", "skill", name)

		case "esc", "q":
			m.skillItems = nil
			m.skillCursor = 0
			m.screen = ScreenMainMenu
			m.err = nil
			return m, nil
		}
	}
	return m, nil
}

// skillListPayload tries to parse a CommandResultPayload as skill list output.
// Returns true and transitions to ScreenSkillList on success.
func (m *RootModel) skillListPayload(p *protocol.CommandResultPayload) bool {
	if p == nil || !p.Success || p.Data == "" {
		return false
	}
	items := parseSkillList(p.Data)
	if len(items) == 0 {
		return false
	}
	m.skillItems = items
	m.skillCursor = 0 // select first skill
	m.screen = ScreenSkillList
	return true
}

// parseSkillList parses the "dscli skill list" text table output into SkillItem slice.
// Expected format (Chinese headers):
//
//	名称                              范围         自动注入
//	api-design-principles           global     -
//	...
func parseSkillList(data string) []SkillItem {
	lines := strings.Split(data, "\n")
	var items []SkillItem
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Skip header line (contains Chinese characters).
		if strings.Contains(line, "名称") || strings.Contains(line, "范围") {
			continue
		}
		// Skip separator lines (e.g. "────").
		if strings.HasPrefix(line, "─") || strings.HasPrefix(line, "-") {
			continue
		}
		// Parse: name (multi-word)  scope  auto-inject
		// Columns are separated by 2+ spaces.
		fields := splitByTwoOrMoreSpaces(line)
		if len(fields) < 2 {
			continue
		}
		item := SkillItem{
			Name:  fields[0],
			Scope: fields[1],
		}
		if len(fields) >= 3 {
			item.AutoInject = fields[2]
		}
		items = append(items, item)
	}
	return items
}

// splitByTwoOrMoreSpaces splits a line by 2+ consecutive spaces.
func splitByTwoOrMoreSpaces(line string) []string {
	var fields []string
	var current strings.Builder
	spaceCount := 0
	for _, r := range line {
		if r == ' ' {
			spaceCount++
			if spaceCount >= 2 && current.Len() > 0 {
				fields = append(fields, strings.TrimSpace(current.String()))
				current.Reset()
				spaceCount = 0
			}
		} else {
			if spaceCount == 1 && current.Len() > 0 {
				// Single space within a field — keep it.
				current.WriteRune(' ')
			} else if spaceCount >= 2 && current.Len() > 0 {
				// Already handled above (split). Start collecting new field.
				current.Reset()
			}
			current.WriteRune(r)
			spaceCount = 0
		}
	}
	if current.Len() > 0 {
		fields = append(fields, strings.TrimSpace(current.String()))
	}
	return fields
}

// ─── Memory List (selectable) ───────────────────────────────────

// updateMemoryList handles keyboard input on the memory selection screen.
func (m *RootModel) updateMemoryList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.memoryCursor > 0 {
				m.memoryCursor--
			}
			return m, nil

		case "down", "j":
			if m.memoryCursor < len(m.memoryItems)-1 {
				m.memoryCursor++
			}
			return m, nil

		case "enter", " ":
			if m.memoryCursor < 0 || m.memoryCursor >= len(m.memoryItems) {
				return m, nil
			}
			id := m.memoryItems[m.memoryCursor].ID
			m.screen = ScreenRunningCmd
			return m, cmdSubcommand(m.agent, m.agent.Memory, "show", "memory", id)

		case "/", "s":
			// Enter search mode — AskUser modal for keyword input.
			m.memorySearchQuery = ""
			m.prevScreen = ScreenMemoryList
			m.screen = ScreenAskUser
			m.askSemantic = protocol.SemanticInput
			m.askQuestion = "Search memories: (enter keywords)"
			m.askInput.SetValue("")
			m.askOptions = nil
			m.askChoice = 0
			m.askDone = false
			m.askResponse = nil
			return m, nil

		case "esc", "q":
			m.memoryItems = nil
			m.memoryCursor = 0
			m.memorySearchQuery = ""
			m.screen = ScreenMainMenu
			m.err = nil
			return m, nil
		}
	}
	return m, nil
}

// ─── Tool List (selectable) ───────────────────────────────────

// updateToolList handles keyboard input on the tool selection screen.
func (m *RootModel) updateToolList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Pagination: fixed 10 per page, capped by terminal height.
		pageSize := 10
		maxRows := m.Height - 7
		if maxRows < 3 {
			maxRows = 3
		}
		if maxRows < pageSize {
			pageSize = maxRows
		}
		totalPages := (len(m.toolItems) + pageSize - 1) / pageSize
		if totalPages < 1 {
			totalPages = 1
		}

		switch msg.String() {
		case "up", "k":
			if m.toolCursor > 0 {
				m.toolCursor--
				start := m.toolPage * pageSize
				if m.toolCursor < start && m.toolPage > 0 {
					m.toolPage--
					newStart := m.toolPage * pageSize
					m.toolCursor = newStart + pageSize - 1
					if m.toolCursor >= len(m.toolItems) {
						m.toolCursor = len(m.toolItems) - 1
					}
				}
			}
			return m, nil

		case "down", "j":
			if m.toolCursor < len(m.toolItems)-1 {
				m.toolCursor++
				end := (m.toolPage + 1) * pageSize
				if m.toolCursor >= end {
					if m.toolPage < totalPages-1 {
						m.toolPage++
						m.toolCursor = m.toolPage * pageSize
					} else {
						m.toolCursor = end - 1 // restore
					}
				}
			}
			return m, nil

		case "pgup":
			if m.toolPage > 0 {
				m.toolPage--
				m.toolCursor = m.toolPage * pageSize
			}
			return m, nil

		case "pgdown":
			if m.toolPage < totalPages-1 {
				m.toolPage++
				m.toolCursor = m.toolPage * pageSize
			}
			return m, nil

		case "home", "g":
			m.toolPage = 0
			m.toolCursor = 0
			return m, nil

		case "end", "G":
			m.toolPage = totalPages - 1
			m.toolCursor = m.toolPage * pageSize
			return m, nil

		case "esc", "q":
			m.toolItems = nil
			m.toolCursor = 0
			m.toolPage = 0
			m.screen = ScreenMainMenu
			m.err = nil
			return m, nil
		}
	}
	return m, nil
}

// ─── Project List (selectable) ─────────────────────────────

// updateProjectList handles keyboard input on the project selection screen.
func (m *RootModel) updateProjectList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.projectCursor > 0 {
				m.projectCursor--
			}
			return m, nil

		case "down", "j":
			if m.projectCursor < len(m.projectItems)-1 {
				m.projectCursor++
			}
			return m, nil

		case "home", "g":
			m.projectCursor = 0
			return m, nil

		case "end", "G":
			m.projectCursor = len(m.projectItems) - 1
			return m, nil

		case "enter", " ":
			if m.projectCursor < 0 || m.projectCursor >= len(m.projectItems) {
				return m, nil
			}
			// Show project details as output.
			item := m.projectItems[m.projectCursor]
			detail := fmt.Sprintf("ID:        %s\nPath:      %s\nMaintainer: %s\nCreated:   %s",
				item.ID, item.Path, item.Maintainer, item.CreatedAt)
			if item.IsCurrent {
				detail += "\n(Current project)"
			}
			m.prevScreen = ScreenProjectList
			m.showOutput(detail, true)
			return m, nil

		case "d", "D":
			if m.projectCursor < 0 || m.projectCursor >= len(m.projectItems) {
				return m, nil
			}
			item := m.projectItems[m.projectCursor]
			m.projectRemovePendingID = item.ID
			m.prevScreen = ScreenProjectList
			m.screen = ScreenAskUser
			m.askSemantic = protocol.SemanticConfirm
			m.askQuestion = fmt.Sprintf("Are you sure you want to delete project %s (%s)?",
				item.ID, item.Path)
			m.askOptions = nil
			m.askInput.SetValue("")
			m.askChoice = 0
			m.askDone = false
			m.askResponse = nil
			return m, nil

		case "esc", "q":
			m.projectItems = nil
			m.projectCursor = 0
			m.projectRemovePendingID = ""
			m.screen = ScreenMainMenu
			m.err = nil
			return m, nil
		}
	}
	return m, nil
}

// projectListPayload tries to parse a CommandResultPayload as project list output.
// Returns true and transitions to ScreenProjectList on success.
func (m *RootModel) projectListPayload(p *protocol.CommandResultPayload) bool {
	if p == nil || !p.Success || p.Data == "" {
		return false
	}
	items := parseProjectList(p.Data)
	if len(items) == 0 {
		return false
	}
	m.projectItems = items
	m.projectCursor = 0 // select first project
	m.screen = ScreenProjectList
	return true
}

// parseProjectList parses the "dscli project list" text table output into ProjectItem slice.
// Expected format:
//
//	ID     Project                                        Maintainer             Created At
//	1      ~/go_project/agent                             黎曼(Riemann, 2)         2026-04-27 18:24:32
//	5      ~/go_project/textSearch                                               2026-05-09 16:37:19
//	16 →   ~/go_project/dscli.tui                         狄拉克(Dirac, 27)         2026-07-05 18:41:09
func parseProjectList(data string) []ProjectItem {
	lines := strings.Split(data, "\n")
	var items []ProjectItem
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Skip header line.
		if strings.Contains(line, "ID") && strings.Contains(line, "Project") {
			continue
		}
		// Skip separator lines.
		if strings.HasPrefix(line, "─") || strings.HasPrefix(line, "-") {
			continue
		}
		// Columns are separated by 2+ spaces.
		fields := splitByTwoOrMoreSpaces(line)
		if len(fields) < 3 {
			continue
		}
		// First field: ID (may have "→" suffix for current project).
		idField := fields[0]
		isCurrent := strings.Contains(idField, "→")
		id := strings.TrimRight(idField, "→ ")
		path := fields[1]
		// Last field is always the timestamp. The field(s) between path and
		// timestamp is the maintainer (may be empty, resulting in 3 fields).
		createdAt := fields[len(fields)-1]
		maintainer := ""
		if len(fields) >= 4 {
			maintainer = strings.Join(fields[2:len(fields)-1], " ")
		}
		items = append(items, ProjectItem{
			ID:         id,
			Path:       path,
			Maintainer: maintainer,
			CreatedAt:  createdAt,
			IsCurrent:  isCurrent,
		})
	}
	return items
}

// toolListPayload tries to parse a CommandResultPayload as tool list output.
// Returns true and transitions to ScreenToolList on success.
func (m *RootModel) toolListPayload(p *protocol.CommandResultPayload) bool {
	if p == nil || !p.Success || p.Data == "" {
		return false
	}
	items := parseToolList(p.Data)
	if len(items) == 0 {
		return false
	}
	m.toolItems = items
	m.toolCursor = 0 // select first tool
	m.toolPage = 0
	m.screen = ScreenToolList
	return true
}

// parseToolList parses the "dscli tool list" text table output into ToolItem slice.
// Expected format:
//
//	名称                               分类              描述
//	flycheck                         code_ops        Static analysis check
//	...
func parseToolList(data string) []ToolItem {
	lines := strings.Split(data, "\n")
	var items []ToolItem
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Skip header line (contains Chinese chars).
		if strings.Contains(line, "名称") {
			continue
		}
		// Skip separator lines.
		if strings.HasPrefix(line, "─") || strings.HasPrefix(line, "-") {
			continue
		}
		// Columns are separated by 2+ spaces.
		fields := splitByTwoOrMoreSpaces(line)
		if len(fields) < 2 {
			continue
		}
		item := ToolItem{
			Name:     fields[0],
			Category: fields[1],
		}
		if len(fields) >= 3 {
			item.Description = fields[2]
		}
		items = append(items, item)
	}
	return items
}

// memoryListPayload tries to parse a CommandResultPayload as memory list output.
// Returns true and transitions to ScreenMemoryList on success.
func (m *RootModel) memoryListPayload(p *protocol.CommandResultPayload) bool {
	if p == nil || !p.Success || p.Data == "" {
		return false
	}
	items := parseMemoryList(p.Data)
	if len(items) == 0 {
		return false
	}
	m.memorySearchQuery = "" // clear any search state
	m.memoryItems = items
	m.memoryCursor = 0 // select first memory item

	m.screen = ScreenMemoryList
	return true
}

// memorySearchPayload handles "dscli memory search" results.
// Unlike memoryListPayload, it always transitions to ScreenMemoryList
// even when results are empty (to show "no results" state).
func (m *RootModel) memorySearchPayload(p *protocol.CommandResultPayload) {
	m.memoryItems = nil
	m.memoryCursor = -1
	if p != nil && p.Success && p.Data != "" {
		items := parseMemorySearchResults(p.Data)
		m.memoryItems = items
		if len(items) > 0 {
			m.memoryCursor = 0
		}
	}
	m.screen = ScreenMemoryList
}

// memoryDatePattern matches date format "Mon DD HH:MM:SS" or "Mon  D HH:MM:SS".
var memoryDatePattern = regexp.MustCompile(`[A-Z][a-z]{2}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2}`)

// parseMemoryList parses the "dscli memory list" text table output into
//
//	89  History list default cursor on oldest instead of newest record  Jul  6 23:11:57  Jul  6 23:11:57
func parseMemoryList(data string) []MemoryItem {
	lines := strings.Split(data, "\n")
	var items []MemoryItem
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Skip header line.
		if strings.Contains(line, "ID") && strings.Contains(line, "TITLE") {
			continue
		}
		// Skip separator lines.
		if strings.HasPrefix(line, "─") || strings.HasPrefix(line, "-") {
			continue
		}
		// Find date columns by pattern (e.g. "Jul  6 23:11:57").
		dateIndexes := memoryDatePattern.FindAllStringIndex(line, -1)
		if len(dateIndexes) < 2 {
			continue
		}
		// Everything before the first date is ID + Title.
		before := strings.TrimSpace(line[:dateIndexes[0][0]])
		createdAt := strings.TrimSpace(line[dateIndexes[0][0]:dateIndexes[0][1]])
		updatedAt := strings.TrimSpace(line[dateIndexes[1][0]:dateIndexes[1][1]])

		// Split the "before" part into ID (first field) and Title (the rest).
		fields := strings.Fields(before)
		if len(fields) == 0 {
			continue
		}
		id := fields[0]
		title := strings.TrimSpace(before[len(id):])

		items = append(items, MemoryItem{
			ID:        id,
			Title:     title,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		})
	}
	return items
}

// memorySearchEntryPattern matches each entry header in "dscli memory search" output.
// Format: "[1] #80 [architecture] dscli.tui chat session: plain-text dscli chat fallback"
var memorySearchEntryPattern = regexp.MustCompile(`^\[\d+\]\s+#(\d+)\s+\[[^\]]*\]\s+(.*)$`)

// parseMemorySearchResults parses "dscli memory search" output into MemoryItem slice.
// The search format differs from the list table format:
//
//	🔍 找到 N 条记忆:
//
//	[1] #ID [Type] Title
//	    Description lines...
//	    ISO_DATE | 相关性: score
//
//	[2] #ID [Type] Title
//	    ...
func parseMemorySearchResults(data string) []MemoryItem {
	lines := strings.Split(data, "\n")
	var items []MemoryItem

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// Skip the header line (🔍).
		if strings.HasPrefix(trimmed, "🔍") {
			continue
		}
		// Match entry header: "[N] #ID [Type] Title"
		matches := memorySearchEntryPattern.FindStringSubmatch(trimmed)
		if matches == nil {
			continue
		}
		id := matches[1]
		title := strings.TrimSpace(matches[2])
		items = append(items, MemoryItem{
			ID:    id,
			Title: title,
		})
	}
	return items
}

// ─── Show Output (scrollable) ────────────────────────────────────
func (m *RootModel) updateShowOutput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Recalculate scroll max based on current terminal height.
		// Height reservations: header(2) + help lines(2) + status bar(1) = 5
		availableHeight := m.Height - 5
		if availableHeight < 3 {
			availableHeight = 3
		}
		totalLines := len(m.outputLines)
		if totalLines > availableHeight {
			m.outputScrollMax = totalLines - availableHeight
		} else {
			m.outputScrollMax = 0
		}

		switch msg.String() {
		case "up", "k":
			if m.outputScroll > 0 {
				m.outputScroll--
			}
			return m, nil
		case "down", "j":
			if m.outputScroll < m.outputScrollMax {
				m.outputScroll++
			}
			return m, nil
		case "pgup":
			pageSize := availableHeight
			m.outputScroll -= pageSize
			if m.outputScroll < 0 {
				m.outputScroll = 0
			}
			return m, nil
		case "pgdown":
			pageSize := availableHeight
			m.outputScroll += pageSize
			if m.outputScroll > m.outputScrollMax {
				m.outputScroll = m.outputScrollMax
			}
			return m, nil
		case "home", "g":
			m.outputScroll = 0
			return m, nil
		case "end", "G":
			m.outputScroll = m.outputScrollMax
			return m, nil
		case "esc", "q", "enter":
			// If prevScreen is set, go back there (e.g. history list).
			m.cmdTitle = ""
			if m.prevScreen != ScreenMainMenu {
				dest := m.prevScreen
				m.prevScreen = ScreenMainMenu
				m.screen = dest
				m.err = nil
				return m, nil
			}
			// Otherwise return to main menu.
			m.screen = ScreenMainMenu
			m.err = nil
			return m, nil
		}
		// All other keys ignored (no accidental exit).
	}
	return m, nil
}

// ─── Chatting ────────────────────────────────────────────────────────

func (m *RootModel) updateChatting(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	// ── Session ready ────────────────────────────────────────────
	case aiagent.ChatSessionReadyMsg:
		if msg.Err != nil {
			errMsg := fmt.Errorf("chat session error: %w", msg.Err)
			m.showOutput(errMsg.Error(), false)
			return m, nil
		}
		m.chatSession = msg.Session
		m.chatReady = true
		// Keep chatLoading: true if history has content (exchange in progress),
		// false for initial session creation (no pending message).
		m.chatLoading = len(m.chatHistory) > 0
		// Start reading events from the session.  The first event will be
		// TypeReady (emitted by the session immediately after creation),
		// which triggers handleChatEvent → cmdSendChatMessage → event loop.
		return m, cmdWaitChatEvent(msg.Session)

	// ── Chat event (chunk, ask_user, done) ───────────────────────
	case aiagent.ChatEventMsg:
		if msg.Err != nil {
			m.err = msg.Err
			m.chatLoading = false
			m.spinnerOn = false
			return m, nil
		}
		if msg.Done {
			return m.handleChatDone()
		}

		return m.handleChatEvent(msg.Message)

	// ── Chimein result (interleaved message sent during loading) ──
	case aiagent.ChimeinResultMsg:
		if msg.Err != nil {
			// dscli process exited abnormally — show error popup.
			errMsg := fmt.Sprintf("⚠️ 插话失败: %v\n%s", msg.Err, msg.Output)
			m.showOutput(errMsg, false)
			return m, nil
		}
		// Check if the process ran as climein (primary lock held).
		// Climein mode outputs a confirmation message and exits quickly.
		// Primary mode (edge case: old session finished) produces AI output.
		if strings.Contains(msg.Output, "已有主 chat 进程运行中") ||
			strings.Contains(msg.Output, "插话内容为空") {
			// Climein mode: the new dscli process wrote the message to the
			// chimeins table for the primary to pick up.  Continue waiting
			// for events from the old session.
			cmd := m.waitForMoreChatEvents()
			if cmd == nil {
				// The old session ended before the chimein was picked up.
				// Start a new session with the accumulated history
				// (which already contains the user's interleaved message).
				m.chatLoading = true
				m.spinnerOn = true
				m.chatDone = false
				return m, cmdStartChat(m.agent, m.chatHistory)
			}
			return m, cmd
		}
		// Primary mode (edge case): the old session released the lock.
		// The output is the AI response — add it to chat history.
		m.appendToLastAssistant(msg.Output, "")
		m.chatLoading = false
		m.chatDone = true
		m.spinnerOn = false
		focusCmd := m.chatInput.Focus()
		return m, focusCmd

	// ── Keyboard input ───────────────────────────────────────────
	case tea.KeyMsg:
		// During loading (AI responding), block Enter but allow
		// navigation and text input so the UI doesn't feel frozen.
		if m.chatLoading && !m.chatDone {
			switch msg.String() {
			case "esc":
				// Exit chat even during loading.
				if m.chatSession != nil {
					m.chatSession.Close() //nolint:errcheck
					m.chatSession = nil
				}
				m.screen = ScreenMainMenu
				m.err = nil
				return m, nil
			case "enter":
				// Interleaved chat (插入对话): user can type while AI is responding.
				// Start a new dscli chat process immediately — it enters climein mode
				// (writes to chimeins table) or becomes primary if the old session has
				// already released the lock.
				input := strings.TrimSpace(m.chatInput.Value())
				m.chatHistory = append(m.chatHistory, ChatLine{Role: "user", Content: input})
				m.chatInput.SetValue("")
				m.chatScroll = 0
				return m, cmdSendChimein(m.agent, input)
			case "pgup", "shift+up":
				if m.chatScroll < m.chatScrollMax {
					m.chatScroll++
				}
				return m, nil
			case "pgdown", "shift+down":
				if m.chatScroll > 0 {
					m.chatScroll--
				}
				return m, nil
			case "ctrl+up":
				if m.chatScroll < m.chatScrollMax {
					m.chatScroll++
				}
				return m, nil
			case "ctrl+down":
				if m.chatScroll > 0 {
					m.chatScroll--
				}
				return m, nil
			case "ctrl+s":
				// Stop the current AI response but stay in chat.
				if m.chatSession != nil {
					m.chatSession.Close() //nolint:errcheck
					m.chatSession = nil
				}
				m.chatLoading = false
				m.chatDone = true
				m.spinnerOn = false
				m.chatHistory = append(m.chatHistory, ChatLine{
					Role:    "system",
					Content: "🛑 Response stopped by user",
				})
				focusCmd := m.chatInput.Focus()
				return m, focusCmd
			default:
			}
		}

		// ── After-loading keyboard handling ──────────────────────────
		switch msg.String() {
		case "esc":
			// If a pending ask_user is active, cancel it by sending
			// an empty response.  The callback handles cleanup
			// (sending to RespCh or sending empty via session).
			if m.askUserPending {
				m.chatInput.SetValue("")
				// Don't add to history — user cancelled.
				cmd := m.askUserRespond("")
				m.askUserPending = false
				m.askUserRespond = nil
				// Close session if any.
				if m.chatSession != nil {
					m.chatSession.Close() //nolint:errcheck
					m.chatSession = nil
				}
				m.screen = ScreenMainMenu
				m.err = nil
				return m, cmd
			}
			// Close session and return to menu — direct transition.
			if m.chatSession != nil {
				m.chatSession.Close() //nolint:errcheck
				m.chatSession = nil
			}
			m.screen = ScreenMainMenu
			m.err = nil
			return m, nil
		case "enter":
			input := strings.TrimSpace(m.chatInput.Value())

			// ── Pending ask_user: route response ──────────────
			if m.askUserPending {
				m.chatInput.SetValue("")
				// Add user's answer to chat history.
				m.chatHistory = append(m.chatHistory, ChatLine{Role: "user", Content: input})
				// Deliver response via the callback.
				cmd := m.askUserRespond(input)
				m.askUserPending = false
				m.askUserRespond = nil
				// Scroll to show new message.
				m.chatScroll = 0
				return m, cmd
			}

			// Allow empty messages — user may want to send "continue" signal.
			// The interleaved chat case (during loading) is handled above.

			// Add user message to chat history immediately so it appears
			// in the view while the AI is responding.
			m.chatHistory = append(m.chatHistory, ChatLine{Role: "user", Content: input})
			m.chatInput.SetValue("")
			m.chatLoading = true
			m.spinnerOn = true
			m.chatDone = false
			m.chatScroll = 0

			// Close old session if still alive.
			if m.chatSession != nil {
				m.chatSession.Close() //nolint:errcheck
			}

			// Start a new session for this exchange.
			return m, cmdStartChat(m.agent, m.chatHistory)

		case "pgup", "shift+up":
			if m.chatScroll < m.chatScrollMax {
				m.chatScroll++
			}
			return m, nil

		case "pgdown", "shift+down":
			if m.chatScroll > 0 {
				m.chatScroll--
			}
			return m, nil

		case "ctrl+up":
			if m.chatScroll < m.chatScrollMax {
				m.chatScroll++
			}
			return m, nil

		case "ctrl+down":
			if m.chatScroll > 0 {
				m.chatScroll--
			}
			return m, nil
		}

		// Route remaining key messages to the textinput.
		var cmd tea.Cmd
		m.chatInput, cmd = m.chatInput.Update(msg)
		return m, cmd

	default:
		// Route non-key messages (e.g. paste) to textinput.
		var cmd tea.Cmd
		m.chatInput, cmd = m.chatInput.Update(msg)
		return m, cmd
	}
}

// handleChatEvent processes a single protocol message from dscli during chat.
func (m *RootModel) handleChatEvent(msg *protocol.Message) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case protocol.TypeReady:
		// dscli is ready. Send the chat request with the accumulated history
		// (the user message was already added on Enter).
		m.chatReady = true
		if len(m.chatHistory) == 0 {
			return m, nil // nothing to send (initial session creation)
		}
		return m, cmdSendChatMessage(m.chatSession, m.chatHistory)

	case protocol.TypeChatChunk:
		p, ok := msg.Payload.(*protocol.ChatChunkPayload)
		if !ok {
			return m, m.waitForMoreChatEvents()
		}
		// Accumulate the last assistant message in history.
		m.appendToLastAssistant(p.Content, p.Reasoning)
		return m, m.waitForMoreChatEvents()

	case protocol.TypeChatDone:
		return m.handleChatDone()

	case protocol.TypeAskUser:
		p, ok := msg.Payload.(*protocol.AskUserPayload)
		if !ok {
			return m, m.waitForMoreChatEvents()
		}
		// Build the question text, including options for SemanticChoice.
		questionText := "🤖 " + p.Question
		if p.Semantic == protocol.SemanticChoice && len(p.Options) > 0 {
			for i, opt := range p.Options {
				questionText += fmt.Sprintf("\n  %d. %s", i+1, opt)
			}
		}
		// Append to chat history instead of switching to ScreenAskUser.
		m.chatHistory = append(m.chatHistory, ChatLine{
			Role:    "system",
			Content: questionText,
		})
		m.askUserPending = true
		session := m.chatSession
		m.askUserRespond = func(input string) tea.Cmd {
			m.askUserPending = false
			m.askUserRespond = nil
			// Add user's answer to chat history.
			m.chatHistory = append(m.chatHistory, ChatLine{Role: "user", Content: input})
			// Build the response payload based on semantic type.
			resp := &protocol.AskUserResponsePayload{Value: input}
			if p.Semantic == protocol.SemanticChoice && len(p.Options) > 0 {
				// Try to match by option text or 1-based index.
				for i, opt := range p.Options {
					if input == opt || input == fmt.Sprintf("%d", i+1) {
						resp = &protocol.AskUserResponsePayload{Choice: i}
						break
					}
				}
			} else if p.Semantic == protocol.SemanticConfirm {
				if input == "y" || input == "Y" {
					resp = &protocol.AskUserResponsePayload{Value: "yes"}
				} else if input == "n" || input == "N" {
					resp = &protocol.AskUserResponsePayload{Value: "no"}
				}
			}
			return cmdSendAskUserResponse(session, resp)
		}
		// Suppress loading state (AI is waiting for user input).
		m.chatLoading = false
		m.chatDone = false
		m.spinnerOn = false
		// Stay in ScreenChatting — no screen switch.
		return m, nil

	case protocol.TypeStatus:
		// Spontaneous status update — ignore for now.
		return m, m.waitForMoreChatEvents()

	case protocol.TypeGoodbye:
		return m.handleChatDone()

	default:
		return m, m.waitForMoreChatEvents()
	}
}

// appendToLastAssistant appends content/reasoning to the most recent assistant
// or reasoning ChatLine in chat history, creating new ones as needed.
//
// When reasoning is non-empty, a separate "reasoning" ChatLine is created
// BEFORE the assistant ChatLine so the view renders it in the thinking style.
// This mirrors dscli's terminal output where 💭 (reasoning) precedes 🐋 (content).
func (m *RootModel) appendToLastAssistant(content, reasoning string) {
	if reasoning != "" {
		if len(m.chatHistory) == 0 || m.chatHistory[len(m.chatHistory)-1].Role != "reasoning" {
			m.chatHistory = append(m.chatHistory, ChatLine{Role: "reasoning", Content: ""})
		}
		last := &m.chatHistory[len(m.chatHistory)-1]
		last.Content += reasoning
	}
	if content != "" {
		// Ensure a trailing assistant ChatLine exists for content.
		if len(m.chatHistory) == 0 || m.chatHistory[len(m.chatHistory)-1].Role != "assistant" {
			m.chatHistory = append(m.chatHistory, ChatLine{Role: "assistant", Content: ""})
		}
		last := &m.chatHistory[len(m.chatHistory)-1]
		last.Content += content
	}
}

// handleChatDone is called when the current chat exchange is complete
// (via TypeChatDone, TypeGoodbye, or Events channel closed).
// It sets the done state and re-focuses the chat input for the next exchange.
// Interleaved messages (typed during loading) are handled immediately via
// cmdSendChimein — no pending-input queue.
func (m *RootModel) handleChatDone() (tea.Model, tea.Cmd) {
	m.chatLoading = false
	m.chatDone = true
	m.spinnerOn = false
	// The dscli process has already exited naturally (stdin was closed
	// and dscli finished processing). Mark session nil so the Enter
	// handler doesn't try to Close a dead process.
	m.chatSession = nil

	focusCmd := m.chatInput.Focus()
	return m, focusCmd
}

// waitForMoreChatEvents returns a Cmd that waits for the next chat event
// without sending any new request.
func (m *RootModel) waitForMoreChatEvents() tea.Cmd {
	if m.chatSession == nil {
		return nil
	}
	return cmdWaitChatEvent(m.chatSession)
}

// ─── AskUser Modal ───────────────────────────────────────────────────

func (m *RootModel) updateAskUser(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.askSemantic {
		case protocol.SemanticConfirm:
			return m.updateAskUserConfirm(msg)
		case protocol.SemanticChoice:
			return m.updateAskUserChoice(msg)
		case protocol.SemanticInput:
			return m.updateAskUserInput(msg)
		}
	default:
		// Route non-key messages (e.g. paste) to askInput.
		var cmd tea.Cmd
		m.askInput, cmd = m.askInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

// updateAskUserConfirm handles Y/N confirmation.
func (m *RootModel) updateAskUserConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.askDone = true
		m.askResponse = &protocol.AskUserResponsePayload{Value: "yes"}
		return m.resumeFromAskUser()

	case "n", "N", "esc":
		m.askDone = true
		m.askResponse = &protocol.AskUserResponsePayload{Value: "no"}
		return m.resumeFromAskUser()
	}
	return m, nil
}

// updateAskUserChoice handles selection from a list of options.
func (m *RootModel) updateAskUserChoice(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.askChoice > 0 {
			m.askChoice--
		}
		return m, nil

	case "down", "j":
		if m.askChoice < len(m.askOptions)-1 {
			m.askChoice++
		}
		return m, nil

	case "enter", " ":
		m.askDone = true
		m.askResponse = &protocol.AskUserResponsePayload{Choice: m.askChoice}
		return m.resumeFromAskUser()

	case "esc":
		m.askDone = true
		m.askResponse = &protocol.AskUserResponsePayload{Value: ""}
		return m.resumeFromAskUser()
	}
	return m, nil
}

// updateAskUserInput handles free-text input.
func (m *RootModel) updateAskUserInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		input := strings.TrimSpace(m.askInput.Value())
		m.askDone = true
		m.askResponse = &protocol.AskUserResponsePayload{Value: input}
		return m.resumeFromAskUser()

	case "esc":
		m.askDone = true
		m.askResponse = &protocol.AskUserResponsePayload{Value: ""}
		return m.resumeFromAskUser()

	default:
		// Route all other keys to the textinput model.
		var cmd tea.Cmd
		m.askInput, cmd = m.askInput.Update(msg)
		return m, cmd
	}
}

// resumeFromAskUser restores the state before the AskUser modal.
// Socket and Chat ask_user flows are now handled via the Chat
// integration (askUserPending + askUserRespond).  This function
// only handles:
//   - Project deletion confirmation flow
//   - Memory search flow
func (m *RootModel) resumeFromAskUser() (tea.Model, tea.Cmd) {
	prev := m.prevScreen
	askResponse := m.askResponse
	m.prevScreen = ScreenMainMenu

	// ── Project deletion flow ──────────────────────────────
	if prev == ScreenProjectList && askResponse != nil && askResponse.Value == "yes" && m.projectRemovePendingID != "" {
		id := m.projectRemovePendingID
		m.projectRemovePendingID = ""
		m.cmdTitle = "📁 Project"
		m.screen = ScreenRunningCmd
		return m, cmdSubcommand(m.agent, m.agent.Project, "remove", "project", id)
	}

	// ── Memory search flow ──────────────────────────────
	if prev == ScreenMemoryList && askResponse != nil && askResponse.Value != "" {
		m.memorySearchQuery = askResponse.Value
		m.memoryItems = nil
		m.memoryCursor = -1
		m.cmdTitle = "🔍 Search"
		m.screen = ScreenRunningCmd
		return m, cmdMemorySearch(m.agent, askResponse.Value)
	}

	// Fallback: return to prev screen or main menu.
	if prev == ScreenProjectList || prev == ScreenMemoryList {
		m.screen = prev
		return m, nil
	}
	m.screen = ScreenMainMenu
	return m, nil
}
