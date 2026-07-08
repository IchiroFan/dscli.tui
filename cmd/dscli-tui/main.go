// Command dscli-tui is the terminal UI frontend for dscli.
//
// It provides an interactive menu for executing dscli commands and
// an immersive chat interface with AskUser modal support.
//
// Subcommands:
//
//	client <file> — Unix socket client for EDITOR-based ask_user bridge.
//	               Invoked by dscli as $EDITOR when ask_user triggers.
//	               Connects to the TUI's socket service and bridges the
//	               question to the interactive modal.
package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dscli/dscli.tui/internal/aiagent"
	"github.com/dscli/dscli.tui/internal/socket"
	"github.com/dscli/dscli.tui/internal/tui"
)

func main() {
	// ── Socket client mode (EDITOR subprocess) ──────────────────────────
	// When dscli invokes ask_user, it runs $EDITOR <tempfile>.
	// We register as "dscli-tui client <file>" so this branch handles it.
	if len(os.Args) >= 2 && os.Args[1] == "client" {
		os.Exit(socket.RunClient(os.Args[2:]))
	}

	// ── Normal TUI startup ──────────────────────────────────────────────

	// Resolve dscli executable.
	agent, err := aiagent.NewExecAgent("")
	if err != nil {
		log.Fatalf("dscli not found: %v\n\nMake sure dscli is installed and in your $PATH.", err)
	}
	defer agent.Close()

	// Start the Unix socket service so dscli's ask_user can reach the TUI.
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("getwd: %v", err)
	}
	sockService := socket.New()
	requests, err := sockService.Start(cwd)
	if err != nil {
		log.Fatalf("socket service: %v", err)
	}
	defer sockService.Stop()

	// Create the TUI model and start Bubble Tea.
	m := tui.New(agent)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())

	// Bridge goroutine: forwards socket AskRequest objects to the TUI
	// update loop as SocketAskUserMsg.  The update loop enters the AskUser
	// modal, and the user's answer flows back through AskRequest.RespCh
	// to the socket client, which writes it to the temp file for dscli.
	go func() {
		for req := range requests {
			p.Send(tui.SocketAskUserMsg{Request: req})
		}
	}()

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "dscli-tui: %v\n", err)
		os.Exit(1)
	}
}
