package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"gitcode.com/dscli/dscli.tui/internal/aiagent"
	"gitcode.com/dscli/dscli.tui/internal/tui/protocol"
)

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
	case ScreenQuitting:
		return m, tea.Quit
	default:
		return m, nil
	}
}

// handleMouseEvent processes mouse events, dispatching wheel events to the
// active screen's scroll logic. Non-wheel mouse events (clicks) are ignored.
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
		return m, nil
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
		m.screen = ScreenRunningCmd
		return m, cmdBalance(m.agent)

	case 2: // Models
		m.screen = ScreenRunningCmd
		return m, cmdModels(m.agent)

	case 3: // Version
		m.screen = ScreenRunningCmd
		return m, cmdVersion(m.agent)

	case 4: // Flycheck
		m.screen = ScreenRunningCmd
		return m, cmdFlycheck(m.agent)

	case 5: // History
		m.historyItems = nil
		m.historyCursor = 0
		m.screen = ScreenRunningCmd
		return m, cmdSubcommand(m.agent, m.agent.History, "list")

	case 6: // Skill
		m.screen = ScreenRunningCmd
		return m, cmdSubcommand(m.agent, m.agent.Skill, "list")

	case 7: // Memory
		m.screen = ScreenRunningCmd
		return m, cmdSubcommand(m.agent, m.agent.Memory, "search")

	case 8: // Project
		m.screen = ScreenRunningCmd
		return m, cmdSubcommand(m.agent, m.agent.Project, "list")

	case 9: // Role
		m.screen = ScreenRunningCmd
		return m, cmdSubcommand(m.agent, m.agent.Role, "list")

	case 10: // Tool
		m.screen = ScreenRunningCmd
		return m, cmdSubcommand(m.agent, m.agent.Tool, "list")

	case 11: // Mail
		m.screen = ScreenRunningCmd
		return m, cmdSubcommand(m.agent, m.agent.Mail, "list")

	case 12: // Service
		m.screen = ScreenRunningCmd
		return m, cmdSubcommand(m.agent, m.agent.Service, "list")

	case 13: // Quit
		m.screen = ScreenQuitting
		return m, tea.Quit

	default:
		return m, nil
	}
}

func (m *RootModel) updateRunningCmd(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Allow Esc to cancel and return to menu (e.g. if command hangs).
	if msg, ok := msg.(tea.KeyMsg); ok && msg.String() == "esc" {
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

	case aiagent.SubcommandResultMsg:
		// History "list" goes to selection screen, all others go to output.
		if msg.Subcmd == "list" && m.historyListPayload(msg.Payload) {
			// Parsed as history items — transition to ScreenHistoryList.
		} else if msg.Subcmd == "show" {
			// Coming from history list selection — return there on exit.
			m.prevScreen = ScreenHistoryList
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

// historyListPayload tries to parse a CommandResultPayload as history list output.
// Returns true and transitions to ScreenHistoryList on success.
func (m *RootModel) historyListPayload(p *protocol.CommandResultPayload) bool {
	if p == nil || !p.Success || p.Data == "" {
		return false
	}
	items := parseHistoryList(p.Data)
	if len(items) == 0 {
		return false
	}
	m.historyItems = items
	m.historyCursor = 0
	m.screen = ScreenHistoryList
	return true
}

// parseHistoryList parses "dscli history list" output into HistoryItem slice.
// Format per line: <ID>  <Role>  <ToolCallID>  <Done>
// Fields are space-separated; 4th field is always last.
func parseHistoryList(data string) []HistoryItem {
	lines := strings.Split(strings.TrimSpace(data), "\n")
	items := make([]HistoryItem, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Split by whitespace — expect at least 2 fields (ID, Role).
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		id := fields[0]
		role := fields[1]
		done := "false"
		if len(fields) >= 4 {
			done = fields[len(fields)-1] // last field is always Done
		}
		items = append(items, HistoryItem{ID: id, Role: role, Done: done})
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
		switch msg.String() {
		case "up", "k":
			if m.historyCursor > 0 {
				m.historyCursor--
			}
			return m, nil

		case "down", "j":
			if m.historyCursor < len(m.historyItems)-1 {
				m.historyCursor++
			}
			return m, nil

		case "enter", " ":
			if m.historyCursor < 0 || m.historyCursor >= len(m.historyItems) {
				return m, nil
			}
			id := m.historyItems[m.historyCursor].ID
			m.screen = ScreenRunningCmd
			return m, cmdSubcommand(m.agent, m.agent.History, "show", id)

		case "esc", "q":
			m.historyItems = nil
			m.historyCursor = 0
			m.screen = ScreenMainMenu
			m.err = nil
			return m, nil
		}
	}
	return m, nil
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
			default:
				// Fall through to route text keys to chat input.
			}
		}

		// ── After-loading keyboard handling ──────────────────────────
		switch msg.String() {
		case "esc":
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
		// Enter modal state.
		m.prevScreen = ScreenChatting
		m.screen = ScreenAskUser
		m.askQuestion = p.Question
		m.askSemantic = p.Semantic
		m.askOptions = p.Options
		m.askInput.SetValue("")
		m.askInput.Focus()
		m.askChoice = 0
		m.askDone = false
		m.askResponse = nil
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

// appendToLastAssistant appends content to the most recent assistant message
// in the chat history, or creates a new one.
func (m *RootModel) appendToLastAssistant(content, reasoning string) {
	if len(m.chatHistory) == 0 || m.chatHistory[len(m.chatHistory)-1].Role != "assistant" {
		m.chatHistory = append(m.chatHistory, ChatLine{Role: "assistant", Content: ""})
	}
	last := &m.chatHistory[len(m.chatHistory)-1]
	last.Content += content
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

// resumeFromAskUser restores the state before the AskUser modal and sends
// the user's response back to dscli (if coming from chat).
func (m *RootModel) resumeFromAskUser() (tea.Model, tea.Cmd) {
	prev := m.prevScreen
	m.prevScreen = ScreenMainMenu

	if prev == ScreenChatting && m.chatSession != nil && m.askResponse != nil {
		m.screen = ScreenChatting
		m.chatLoading = true // waiting for more events
		return m, cmdSendAskUserResponse(m.chatSession, m.askResponse)
	}

	// Fallback: return to main menu.
	m.screen = ScreenMainMenu
	return m, nil
}
