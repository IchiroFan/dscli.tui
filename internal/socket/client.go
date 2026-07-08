package socket

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
)

// RunClient implements the EDITOR subprocess for ask_user.
//
// It is invoked by dscli as $EDITOR <tempfile> (i.e. "dscli-tui client <file>").
//
// Steps:
//  1. Reads the question from the temp file
//  2. Finds the Unix socket by walking up from cwd
//  3. Connects to the socket
//  4. Sends a two-line request: question + file path
//  5. Reads the user's answer (until connection close)
//  6. Appends the answer to the temp file
//  7. Exits (dscli reads the file as the ask_user result)
//
// Returns exit code 0 on success, 1 on error.
func RunClient(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: dscli-tui client <file>")
		return 1
	}
	filePath := args[0]

	// 1. Read the question from the temp file.
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot read %s: %v\n", filePath, err)
		return 1
	}

	// 2. Find the socket file by walking up from cwd.
	socketPath := findSocketPath()
	if socketPath == "" {
		fmt.Fprintln(os.Stderr, "error: dscli-tui service not running (socket not found)")
		return 1
	}

	// 3. Connect to the socket.
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot connect to dscli-tui: %v\n", err)
		return 1
	}
	defer conn.Close()

	// 4. Send the request: two lines (question + file path).
	question := strings.TrimSpace(string(content))
	fmt.Fprintf(conn, "%s\n%s\n", question, filePath)

	// 5. Read the response (read until connection close).
	respBytes, err := io.ReadAll(conn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: read response: %v\n", err)
		return 1
	}
	respContent := string(respBytes)

	// 6. Append the response to the temp file.
	//    dscli expects the answer in this file after the editor exits.
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: append to %s: %v\n", filePath, err)
		return 1
	}
	defer f.Close()
	if _, err := f.WriteString("\n" + respContent); err != nil {
		fmt.Fprintf(os.Stderr, "error: write to %s: %v\n", filePath, err)
		return 1
	}

	return 0
}

// findSocketPath walks up from the current working directory looking for
// .dscli/dscli-tui.sock.  Returns the full path or "" if not found.
func findSocketPath() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	dir := cwd
	for {
		socketPath := filepath.Join(dir, ".dscli", "dscli-tui.sock")
		if _, err := os.Stat(socketPath); err == nil {
			return socketPath
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached root
		}
		dir = parent
	}
	return ""
}
