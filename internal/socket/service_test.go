package socket

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

// TestServiceStartStop verifies that the service starts, creates the socket
// file, and Stop removes it.
func TestServiceStartStop(t *testing.T) {
	tmpDir := t.TempDir()

	svc := New()
	requests, err := svc.Start(tmpDir)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Socket file should exist.
	socketPath := filepath.Join(tmpDir, ".dscli", "dscli-tui.sock")
	if _, err := os.Stat(socketPath); err != nil {
		t.Fatalf("socket file %s not created: %v", socketPath, err)
	}

	// Stop should remove the socket file.
	if err := svc.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Fatalf("socket file %s should be removed after Stop", socketPath)
	}

	// Requests channel should be closed after Stop.
	_, ok := <-requests
	if ok {
		t.Fatal("requests channel should be closed after Stop")
	}
}

// TestServiceRequestResponse verifies a full request/response cycle:
//  1. Start the service
//  2. Connect a client
//  3. Send a two-line request
//  4. Read the AskRequest from the channel
//  5. Write a response
//  6. Verify the client receives the response
func TestServiceRequestResponse(t *testing.T) {
	tmpDir := t.TempDir()

	svc := New()
	requests, err := svc.Start(tmpDir)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer svc.Stop()

	socketPath := filepath.Join(tmpDir, ".dscli", "dscli-tui.sock")

	// Connect a client in a goroutine.
	clientErr := make(chan error, 1)
	clientResp := make(chan string, 1)
	go func() {
		conn, err := net.Dial("unix", socketPath)
		if err != nil {
			clientErr <- err
			return
		}
		defer conn.Close()

		// Send two-line request.
		if _, err := conn.Write([]byte("What is your favorite color?\n/tmp/test-file.md\n")); err != nil {
			clientErr <- err
			return
		}

		// Read response.
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			clientErr <- err
			return
		}
		clientResp <- string(buf[:n])
	}()

	// Read the AskRequest from the service channel.
	req := <-requests
	if req == nil {
		t.Fatal("expected non-nil AskRequest")
	}

	if req.Question != "What is your favorite color?" {
		t.Fatalf("expected question 'What is your favorite color?', got %q", req.Question)
	}
	if req.FilePath != "/tmp/test-file.md" {
		t.Fatalf("expected file path '/tmp/test-file.md', got %q", req.FilePath)
	}

	// Send response via the channel.
	req.RespCh <- "blue"

	// Verify the client received it.
	select {
	case resp := <-clientResp:
		if resp != "blue" {
			t.Fatalf("expected response 'blue', got %q", resp)
		}
	case err := <-clientErr:
		t.Fatalf("client error: %v", err)
	}
}

// TestStaleSocketCleanup verifies that Start removes a stale socket file
// from a previous run before listening.
func TestStaleSocketCleanup(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a stale socket file.
	socketDir := filepath.Join(tmpDir, ".dscli")
	if err := os.MkdirAll(socketDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	stalePath := filepath.Join(socketDir, "dscli-tui.sock")
	stale, err := net.Listen("unix", stalePath)
	if err != nil {
		t.Fatalf("create stale socket: %v", err)
	}
	stale.Close()

	// Start service — should remove the stale socket and bind successfully.
	svc := New()
	requests, err := svc.Start(tmpDir)
	if err != nil {
		t.Fatalf("Start with stale socket: %v", err)
	}
	defer svc.Stop()

	// Verify the socket file is the new one (service is listening).
	conn, err := net.Dial("unix", stalePath)
	if err != nil {
		t.Fatalf("connect after stale cleanup: %v", err)
	}
	conn.Close()

	// Drain channel to avoid goroutine leak.
	go func() {
		for range requests {
		}
	}()
}
