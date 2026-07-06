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

	"gitcode.com/dscli/dscli.tui/internal/tui/protocol"
	"gitcode.com/dscli/dscli.tui/pkg/jsonline"
)

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
// Temporarily uses plain-text mode (no --json-line) since dscli doesn't
// yet support the JSON-line protocol.  When --json-line is implemented,
// this should switch to the JSON-line codec for streaming and AskUser support.
//
// Flow per exchange:
//  1. Emit TypeReady immediately (session starts ready).
//  2. Wait for ChatRequestPayload from TUI.
//  3. Write the user message to the process stdin and close stdin.
//  4. Read all stdout until process exits.
//  5. Emit ChatChunkPayload with the captured text, then ChatDonePayload.
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
	// Forward stderr to os.Stderr for diagnostics.
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("start dscli chat: %w", err)
	}

	events := make(chan *protocol.Message, 100)
	sendCh := make(chan *protocol.Message, 10)
	done := make(chan struct{})

	// Single goroutine: handles the full exchange lifecycle.
	go func() {
		defer close(events)
		defer close(done)

		// 1. Emit TypeReady immediately — the session is ready to accept input.
		events <- &protocol.Message{Type: protocol.TypeReady}

		// 2. Wait for ChatRequestPayload or cancellation.
		select {
		case msg, ok := <-sendCh:
			if !ok {
				stdin.Close()
				cmd.Wait()
				return
			}
			p, ok := msg.Payload.(*protocol.ChatRequestPayload)
			if !ok || len(p.Messages) == 0 {
				stdin.Close()
				cmd.Wait()
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
				cmd.Wait()
				return
			}

			// 3. Write user message to stdin, then close to signal EOF.
			//    dscli chat processes the input and exits when stdin closes.
			io.WriteString(stdin, userMsg+"\n")
			stdin.Close()

			// 4. Read all stdout output.
			var respBuilder strings.Builder
			reader := bufio.NewReader(stdout)
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					if line != "" {
						respBuilder.WriteString(line)
					}
					break
				}
				respBuilder.WriteString(line)
			}

			// 5. Emit ChatChunk with the full response text, then ChatDone.
			response := respBuilder.String()
			if response != "" {
				events <- &protocol.Message{
					Type:    protocol.TypeChatChunk,
					Payload: &protocol.ChatChunkPayload{Content: response},
				}
			}
			events <- &protocol.Message{Type: protocol.TypeChatDone}

		case <-ctx.Done():
			stdin.Close()
			cmd.Wait()
			return
		}

		// Wait for the process to fully exit before the goroutine ends.
		cmd.Wait()
	}()

	return &ChatSession{
		Events: events,
		Send:   sendCh,
		Done:   done,
		close: func() error {
			cancel()
			return cmd.Wait()
		},
	}, nil
}

// ── Close ───────────────────────────────────────────────────

func (a *execAgent) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.closed = true
	return nil
}
