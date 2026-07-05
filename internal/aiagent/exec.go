package aiagent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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

// ── Non-interactive commands ────────────────────────────────

func (a *execAgent) Balance(ctx context.Context, format string) (*protocol.CommandResultPayload, error) {
	args := []string{"--json-line", "balance"}
	if format != "" {
		args = append(args, "--format", format)
	}
	return a.execDS(ctx, args...)
}

func (a *execAgent) Models(ctx context.Context, format string, showPrice bool) (*protocol.CommandResultPayload, error) {
	args := []string{"--json-line", "models"}
	if format != "" {
		args = append(args, "--format", format)
	}
	if showPrice {
		args = append(args, "--price")
	}
	return a.execDS(ctx, args...)
}

func (a *execAgent) Version(ctx context.Context) (*protocol.CommandResultPayload, error) {
	return a.execDS(ctx, "--json-line", "version")
}

func (a *execAgent) Flycheck(ctx context.Context, path string, emacs bool) (*protocol.CommandResultPayload, error) {
	args := []string{"--json-line", "flycheck"}
	if emacs {
		args = append(args, "--emacs")
	}
	args = append(args, path)
	return a.execDS(ctx, args...)
}

func (a *execAgent) FIM(ctx context.Context, args ...string) (*protocol.CommandResultPayload, error) {
	cmdArgs := append([]string{"--json-line", "fim"}, args...)
	return a.execDS(ctx, cmdArgs...)
}

// ── Subcommand groups ───────────────────────────────────────

func (a *execAgent) History(ctx context.Context, subcmd string, args ...string) (*protocol.CommandResultPayload, error) {
	cmdArgs := append([]string{"--json-line", "history", subcmd}, args...)
	return a.execDS(ctx, cmdArgs...)
}

func (a *execAgent) Skill(ctx context.Context, subcmd string, args ...string) (*protocol.CommandResultPayload, error) {
	cmdArgs := append([]string{"--json-line", "skill", subcmd}, args...)
	return a.execDS(ctx, cmdArgs...)
}

func (a *execAgent) Prompt(ctx context.Context, subcmd string, args ...string) (*protocol.CommandResultPayload, error) {
	cmdArgs := append([]string{"--json-line", "prompt", subcmd}, args...)
	return a.execDS(ctx, cmdArgs...)
}

func (a *execAgent) Memory(ctx context.Context, subcmd string, args ...string) (*protocol.CommandResultPayload, error) {
	cmdArgs := append([]string{"--json-line", "memory", subcmd}, args...)
	return a.execDS(ctx, cmdArgs...)
}

func (a *execAgent) Project(ctx context.Context, subcmd string, args ...string) (*protocol.CommandResultPayload, error) {
	cmdArgs := append([]string{"--json-line", "project", subcmd}, args...)
	return a.execDS(ctx, cmdArgs...)
}

func (a *execAgent) Role(ctx context.Context, subcmd string, args ...string) (*protocol.CommandResultPayload, error) {
	cmdArgs := append([]string{"--json-line", "role", subcmd}, args...)
	return a.execDS(ctx, cmdArgs...)
}

func (a *execAgent) Tool(ctx context.Context, subcmd string, args ...string) (*protocol.CommandResultPayload, error) {
	cmdArgs := append([]string{"--json-line", "tool", subcmd}, args...)
	return a.execDS(ctx, cmdArgs...)
}

func (a *execAgent) Mail(ctx context.Context, subcmd string, args ...string) (*protocol.CommandResultPayload, error) {
	cmdArgs := append([]string{"--json-line", "mail", subcmd}, args...)
	return a.execDS(ctx, cmdArgs...)
}

func (a *execAgent) Service(ctx context.Context, subcmd string, args ...string) (*protocol.CommandResultPayload, error) {
	cmdArgs := append([]string{"--json-line", "service", subcmd}, args...)
	return a.execDS(ctx, cmdArgs...)
}

// ── execDS: generic non-interactive execution ───────────────

// execDS runs dscli with the given args in JSON-line mode and returns
// the first CommandResultPayload received.  It consumes and discards
// any StatusPayload or Ready messages before the result.
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

// NewChatSession starts a dscli chat subprocess in JSON-line mode and returns
// a ChatSession for bidirectional communication.
func (a *execAgent) NewChatSession(ctx context.Context, opts ChatSessionOptions) (*ChatSession, error) {
	args := []string{"--json-line", "chat"}

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
	// Forward stderr to os.Stderr for diagnostics (not part of JSON-line protocol).
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("start dscli chat: %w", err)
	}

	events := make(chan *protocol.Message, 100)
	sendCh := make(chan *protocol.Message, 10)
	done := make(chan struct{})

	// Read loop: stdout → events
	go func() {
		defer close(events)
		dec := jsonline.NewDecoder(stdout)
		for dec.Decode() {
			msg := dec.Message()
			events <- msg

			if msg.Type == protocol.TypeGoodbye {
				return
			}
		}
	}()

	// Write loop: sendCh → stdin
	go func() {
		enc := jsonline.NewEncoder(stdin)
		for msg := range sendCh {
			if err := enc.Encode(msg); err != nil {
				return
			}
		}
	}()

	// Cleanup goroutine: wait for process exit
	go func() {
		cmd.Wait()
		close(done)
		cancel()
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
