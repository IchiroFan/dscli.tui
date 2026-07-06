package aiagent

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"gitcode.com/dscli/dscli.tui/internal/tui/protocol"
	"gitcode.com/dscli/dscli.tui/pkg/jsonline"
)

const chunkThreshold = 20 // emit chunk on \n or when this many bytes accumulate (smaller = smoother incremental display)

// ─── execAgent ──────────────────────────────────────────────

// execAgent implements AIAgent by executing dscli as a subprocess
// and communicating via the JSON-line protocol over stdio.
type execAgent struct {
	dscliPath string // resolved path to dscli binary
	mu        sync.Mutex
	closed    bool
}

// NewExecAgent creates an AIAgent that delegates to a dscli binary.
// The first call resolves the dscli path; subsequent calls reuse the result.
func NewExecAgent(dscliPath string) (AIAgent, error) {
	resolved, err := resolveDSCLIPath(dscliPath)
	if err != nil {
		return nil, err
	}
	return &execAgent{dscliPath: resolved}, nil
}

// ── Non-interactive commands (raw exec, no --json-line) ──

func (a *execAgent) Balance(ctx context.Context, format string) (*protocol.CommandResultPayload, error) {
	args := []string{"balance"}
	if format != "" {
		args = append(args, "--format", format)
	}
	return a.execDSRaw(ctx, args...)
}

func (a *execAgent) Models(ctx context.Context, format string, showPrice bool) (*protocol.CommandResultPayload, error) {
	args := []string{"models"}
	if format != "" {
		args = append(args, "--format", format)
	}
	if showPrice {
		args = append(args, "--price")
	}
	return a.execDSRaw(ctx, args...)
}

func (a *execAgent) Version(ctx context.Context) (*protocol.CommandResultPayload, error) {
	return a.execDSRaw(ctx, "version")
}

func (a *execAgent) Flycheck(ctx context.Context, path string, emacs bool) (*protocol.CommandResultPayload, error) {
	args := []string{"flycheck"}
	if emacs {
		args = append(args, "--emacs")
	}
	args = append(args, path)
	return a.execDSRaw(ctx, args...)
}

func (a *execAgent) FIM(ctx context.Context, args ...string) (*protocol.CommandResultPayload, error) {
	cmdArgs := append([]string{"fim"}, args...)
	return a.execDSRaw(ctx, cmdArgs...)
}

// ── Subcommand groups ───────────────────────────────────────

func (a *execAgent) History(ctx context.Context, subcmd string, args ...string) (*protocol.CommandResultPayload, error) {
	cmdArgs := append([]string{"history", subcmd}, args...)
	return a.execDSRaw(ctx, cmdArgs...)
}

func (a *execAgent) Skill(ctx context.Context, subcmd string, args ...string) (*protocol.CommandResultPayload, error) {
	cmdArgs := append([]string{"skill", subcmd}, args...)
	return a.execDSRaw(ctx, cmdArgs...)
}

func (a *execAgent) Prompt(ctx context.Context, subcmd string, args ...string) (*protocol.CommandResultPayload, error) {
	cmdArgs := append([]string{"prompt", subcmd}, args...)
	return a.execDSRaw(ctx, cmdArgs...)
}

func (a *execAgent) Memory(ctx context.Context, subcmd string, args ...string) (*protocol.CommandResultPayload, error) {
	cmdArgs := append([]string{"memory", subcmd}, args...)
	return a.execDSRaw(ctx, cmdArgs...)
}

func (a *execAgent) Project(ctx context.Context, subcmd string, args ...string) (*protocol.CommandResultPayload, error) {
	cmdArgs := append([]string{"project", subcmd}, args...)
	return a.execDSRaw(ctx, cmdArgs...)
}

func (a *execAgent) Role(ctx context.Context, subcmd string, args ...string) (*protocol.CommandResultPayload, error) {
	cmdArgs := append([]string{"role", subcmd}, args...)
	return a.execDSRaw(ctx, cmdArgs...)
}

func (a *execAgent) Tool(ctx context.Context, subcmd string, args ...string) (*protocol.CommandResultPayload, error) {
	cmdArgs := append([]string{"tool", subcmd}, args...)
	return a.execDSRaw(ctx, cmdArgs...)
}

func (a *execAgent) Mail(ctx context.Context, subcmd string, args ...string) (*protocol.CommandResultPayload, error) {
	cmdArgs := append([]string{"mail", subcmd}, args...)
	return a.execDSRaw(ctx, cmdArgs...)
}

func (a *execAgent) Service(ctx context.Context, subcmd string, args ...string) (*protocol.CommandResultPayload, error) {
	cmdArgs := append([]string{"service", subcmd}, args...)
	return a.execDSRaw(ctx, cmdArgs...)
}

// ── execDSRaw: raw exec (no --json-line) ────────────────────

// execDSRaw runs dscli with the given args directly (no --json-line mode),
// captures stdout+stderr, and wraps the result in a CommandResultPayload.
// Used by all non-interactive commands until dscli implements --json-line.
func (a *execAgent) execDSRaw(ctx context.Context, args ...string) (*protocol.CommandResultPayload, error) {
	cmd := exec.CommandContext(ctx, a.dscliPath, args...)
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))
	if err != nil {
		return &protocol.CommandResultPayload{
			Success: false,
			Data:    output,
		}, fmt.Errorf("dscli %v: %w\n%s", args, err, output)
	}
	return &protocol.CommandResultPayload{
		Success: true,
		Data:    output,
	}, nil
}

// ── execDS: JSON-line execution (reserved for Phase 4) ──────

// execDS runs dscli with the given args in JSON-line mode and returns
// the first CommandResultPayload received.  It consumes and discards
// any StatusPayload or Ready messages before the result.
//
// NOTE: Currently unused by non-interactive commands (they use execDSRaw).
// Kept for Phase 4 when dscli implements --json-line mode.
func (a *execAgent) execDS(ctx context.Context, args ...string) (*protocol.CommandResultPayload, error) {
	cmd := exec.CommandContext(ctx, a.dscliPath, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("execDS stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("execDS start: %w", err)
	}
	defer cmd.Wait() //nolint:errcheck

	dec := jsonline.NewDecoder(stdout)
	for dec.Decode() {
		msg := dec.Message()
		switch msg.Type {
		case protocol.TypeCmdResult:
			p, ok := msg.Payload.(*protocol.CommandResultPayload)
			if !ok {
				return nil, fmt.Errorf("unexpected payload type for command_result: %T", msg.Payload)
			}
			return p, nil

		case protocol.TypeError:
			if msg.Error != nil {
				return nil, msg.Error
			}
			return nil, fmt.Errorf("dscli returned error type without details")

		case protocol.TypeReady, protocol.TypeStatus, protocol.TypeGoodbye:
			// skip — status messages before the result

		default:
			return nil, fmt.Errorf("unexpected message from dscli: %s", msg.Type)
		}
	}

	if err := dec.Err(); err != nil {
		return nil, fmt.Errorf("execDS read: %w", err)
	}
	return nil, fmt.Errorf("dscli exited without returning a result")
}

// ── Chat ────────────────────────────────────────────────────

// NewChatSession starts a dscli chat subprocess and returns a ChatSession
// for one-shot exchange (one user message → one response).
//
// dscli runs without --stream (non-streaming API). It calls the DeepSeek
// API, receives the full response, then outputs the content to stdout via
// PrintContent.  The TUI reads stdout byte-by-byte and emits a ChatChunkPayload
// whenever the accumulated content reaches chunkThreshold bytes, or on EOF.
//
// Flow per exchange:
//  1. Emit TypeReady immediately (session starts ready).
//  2. Wait for ChatRequestPayload from TUI.
//  3. Write the user message to the process stdin and close stdin.
//  4. Read stdout byte-by-byte, emitting ChatChunkPayload per fixed-size chunk.
//  5. On EOF, emit ChatDonePayload.
//  6. Call cmd.Wait() to collect the process exit code.
//     If non-zero, emit an error chunk so the TUI can display it.
func (a *execAgent) NewChatSession(ctx context.Context, opts ChatSessionOptions) (*ChatSession, error) {
	args := []string{"chat"}

	ctx, cancel := context.WithCancel(ctx)

	cmd := exec.CommandContext(ctx, a.dscliPath, args...)
	if opts.ProjectDir != "" {
		cmd.Dir = opts.ProjectDir
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("chat stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("chat stdout pipe: %w", err)
	}
	// Capture stderr for error diagnostics (shown on terminal AND captured for TUI).
	var stderrBuf strings.Builder
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("start dscli chat: %w", err)
	}

	events := make(chan *protocol.Message, 3) // small buffer for backpressure — producer blocks when full, pacing events for the TUI
	sendCh := make(chan *protocol.Message, 10)
	done := make(chan struct{})
	// waitDone carries the result of cmd.Wait().  The goroutine calls
	// cmd.Wait() exactly once — Close() reads from this channel instead
	// of calling cmd.Wait() itself, which would race.
	waitDone := make(chan error, 1)

	// Single goroutine: handles the full exchange lifecycle.
	//
	// IMPORTANT: the goroutine calls cmd.Wait() exactly once after the
	// exchange completes.  Close() must NOT call cmd.Wait() — it cancels
	// the context and waits for waitDone instead.
	go func() {
		defer close(done)
		defer close(events)

		// 1. Emit TypeReady immediately — the session is ready to accept input.
		events <- &protocol.Message{Type: protocol.TypeReady}

		// 2. Wait for ChatRequestPayload or cancellation.
		select {
		case msg, ok := <-sendCh:
			if !ok {
				stdin.Close()
				waitDone <- cmd.Wait()
				return
			}
			p, ok := msg.Payload.(*protocol.ChatRequestPayload)
			if !ok || len(p.Messages) == 0 {
				stdin.Close()
				waitDone <- cmd.Wait()
				return
			}
			// Find the last user message.
			var userMsg string
			for i := len(p.Messages) - 1; i >= 0; i-- {
				if p.Messages[i].Role == "user" {
					userMsg = p.Messages[i].Content
					break
				}
			}
			if userMsg == "" {
				stdin.Close()
				waitDone <- cmd.Wait()
				return
			}

			// 3. Write user message to stdin, then close to signal EOF.
			//    dscli chat processes the input and exits when stdin closes.
			io.WriteString(stdin, userMsg+"\n")
			stdin.Close()

			// 4. Read stdout byte-by-byte, emitting a ChatChunkPayload whenever
			//    chunkThreshold bytes have accumulated, or on EOF.  dscli outputs
			//    the full response (non-streaming) via PrintContent, so data may
			//    arrive in bursts — the byte-by-byte loop handles that gracefully.
			var chunkBuf strings.Builder
			reader := bufio.NewReader(stdout)

			for {
				b, err := reader.ReadByte()
				if err != nil {
					// Emit any remaining content before EOF.
					if chunkBuf.Len() > 0 {
						events <- &protocol.Message{
							Type:    protocol.TypeChatChunk,
							Payload: &protocol.ChatChunkPayload{Content: chunkBuf.String()},
						}
					}
					break
				}
				chunkBuf.WriteByte(b)
				// Emit a chunk on each newline, or when the buffer reaches
				// the fixed threshold — ensures smooth incremental rendering
				// even for long lines without line breaks.
				if b == '\n' || chunkBuf.Len() >= chunkThreshold {
					events <- &protocol.Message{
						Type:    protocol.TypeChatChunk,
						Payload: &protocol.ChatChunkPayload{Content: chunkBuf.String()},
					}
					chunkBuf.Reset()
				}
			}

			// 5. Wait for the process to finish and collect its exit code.
			waitErr := cmd.Wait()
			waitDone <- waitErr

			// If the process exited abnormally, append an error chunk so the
			// TUI can display it.  Include captured stderr content for diagnostics.
			if waitErr != nil {
				errMsg := fmt.Sprintf("\n⚠️ dscli exited with error: %v", waitErr)
				if stderrContent := strings.TrimSpace(stderrBuf.String()); stderrContent != "" {
					errMsg += fmt.Sprintf("\n📋 stderr: %s", stderrContent)
				}
				errMsg += "\n"
				events <- &protocol.Message{
					Type: protocol.TypeChatChunk,
					Payload: &protocol.ChatChunkPayload{
						Content: errMsg,
					},
				}
			}

			// 6. Emit ChatDone signal.
			events <- &protocol.Message{Type: protocol.TypeChatDone}

		case <-ctx.Done():
			stdin.Close()
			waitDone <- cmd.Wait()
			return
		}
	}()

	return &ChatSession{
		Events: events,
		Send:   sendCh,
		Done:   done,
		close: func() error {
			cancel()
			// Wait for the goroutine to finish — it calls cmd.Wait() and
			// sends the result on waitDone.  Use a timeout to prevent
			// blocking the TUI update loop indefinitely.
			select {
			case err := <-waitDone:
				return err
			case <-time.After(10 * time.Second):
				return fmt.Errorf("timeout waiting for dscli process to exit")
			}
		},
	}, nil
}

// ── SendChimein ─────────────────────────────────────────────

// SendChimein runs dscli chat with the given content on stdin.
// If another dscli chat process holds the project lock, this instance
// enters climein mode (writes to chimeins table and exits quickly).
// Otherwise it becomes the primary process and produces a full AI response.
//
// Returns the combined stdout+stderr output and any process error.
func (a *execAgent) SendChimein(ctx context.Context, content string) (string, error) {
	cmd := exec.CommandContext(ctx, a.dscliPath, "chat")
	cmd.Stdin = strings.NewReader(content + "\n")
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))
	if err != nil {
		return output, fmt.Errorf("dscli chat (climein) failed: %w", err)
	}
	return output, nil
}

// ── Close ───────────────────────────────────────────────────

func (a *execAgent) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.closed = true
	return nil
}
