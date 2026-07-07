// Command dscli-tui is the terminal UI frontend for dscli.
//
// It provides an interactive menu for executing dscli commands and
// an immersive chat interface with AskUser modal support.
package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dscli/dscli.tui/internal/aiagent"
	"github.com/dscli/dscli.tui/internal/tui"
)

func main() {
	// Resolve dscli executable.
	agent, err := aiagent.NewExecAgent("")
	if err != nil {
		log.Fatalf("dscli not found: %v\n\nMake sure dscli is installed and in your $PATH.", err)
	}
	defer agent.Close()

	// Create the TUI model and start Bubble Tea.
	m := tui.New(agent)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "dscli-tui: %v\n", err)
		os.Exit(1)
	}
}
