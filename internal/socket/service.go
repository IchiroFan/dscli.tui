// Package socket provides a Unix socket bridge between dscli's EDITOR-based
// ask_user mechanism and the TUI's interactive modal.
//
// Architecture:
//   - Service: listens on a Unix socket, accepts connections from the EDITOR
//     subprocess (dscli-tui client), and bridges requests to the TUI.
//   - Client: runs as the EDITOR subprocess, connects to the socket, sends
//     the question, and writes the user's response back to the temp file.
//
// Protocol (plain text, two lines over the socket):
//   Request:  <question>\n<file-path>\n
//   Response: <user-answer> (read until connection close)
package socket

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

// AskRequest represents a pending ask_user request forwarded from dscli.
// The TUI receives this via SocketAskUserMsg, displays the modal, and
// writes the user's answer to RespCh.
type AskRequest struct {
	Question string // the question dscli asked
	FilePath string // temp file path to write the answer back to
	RespCh   chan string // TUI writes the user's answer here
}

// Service manages a Unix socket that bridges dscli's ask_user to the TUI.
type Service struct {
	listener   net.Listener
	socketPath string
}

// New creates a new Service. Call Start to begin listening.
func New() *Service {
	return &Service{}
}

// Start creates the socket directory, removes any stale socket file, and
// begins listening on <projectRoot>/.dscli/dscli-tui.sock.
//
// It returns a channel of incoming AskRequest objects.  The caller must
// handle each request by reading from AskRequest.RespCh and forwarding it
// to the TUI update loop (via Bubble Tea's Program.Send).
//
// The accept loop runs in a background goroutine.  Stop() must be called
// to clean up.
func (s *Service) Start(projectRoot string) (<-chan *AskRequest, error) {
	socketDir := filepath.Join(projectRoot, ".dscli")
	if err := os.MkdirAll(socketDir, 0o755); err != nil {
		return nil, fmt.Errorf("create socket dir %s: %w", socketDir, err)
	}

	socketPath := filepath.Join(socketDir, "dscli-tui.sock")

	// Remove stale socket file from a previous (possibly crashed) run.
	os.Remove(socketPath)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", socketPath, err)
	}

	s.listener = listener
	s.socketPath = socketPath

	requests := make(chan *AskRequest)
	go s.acceptLoop(requests)
	return requests, nil
}

// Stop closes the listener and removes the socket file.
func (s *Service) Stop() error {
	if s.listener != nil {
		s.listener.Close()
	}
	if s.socketPath != "" {
		os.Remove(s.socketPath)
	}
	return nil
}

// SocketPath returns the Unix socket path, or "" if not started.
func (s *Service) SocketPath() string {
	return s.socketPath
}

// acceptLoop runs in a goroutine, accepting connections.
func (s *Service) acceptLoop(requests chan<- *AskRequest) {
	defer close(requests)

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			// Listener closed → Stop() was called.
			return
		}
		go s.handleConn(conn, requests)
	}
}

// handleConn processes one socket client connection.
//
//  1. Reads two lines: question + file path
//  2. Sends an AskRequest on the channel
//  3. Blocks waiting for the response on RespCh
//  4. Writes the response back to the client
//  5. Closes the connection
func (s *Service) handleConn(conn net.Conn, requests chan<- *AskRequest) {
	defer conn.Close()

	// Set a deadline so an unresponsive TUI doesn't hang the client forever.
	// 300s matches the ask_user timeout documented in the edge cases table.
	conn.SetDeadline(time.Now().Add(300 * time.Second))

	// Read two lines: question + file path.
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		return // EOF or error
	}
	question := scanner.Text()

	if !scanner.Scan() {
		return
	}
	filePath := scanner.Text()

	if err := scanner.Err(); err != nil {
		return
	}

	// Create a buffered channel so the TUI can write without blocking
	// (the TUI writes during its synchronous update loop).
	respCh := make(chan string, 1)

	requests <- &AskRequest{
		Question: question,
		FilePath: filePath,
		RespCh:   respCh,
	}

	// Block until the TUI sends the user's answer.
	response := <-respCh

	// Write the response back to the client.
	conn.Write([]byte(response))
}
