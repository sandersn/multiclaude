package copilot

import (
	"context"
	"crypto/rand"
	"fmt"
	"os/exec"
	"time"
)

// TerminalRunner abstracts terminal interaction for running Copilot.
// The tmux.Client implements this interface.
type TerminalRunner interface {
	// SendKeys sends text followed by Enter to submit.
	SendKeys(ctx context.Context, session, window, text string) error

	// SendKeysLiteral sends text without pressing Enter (supports multiline via paste-buffer).
	SendKeysLiteral(ctx context.Context, session, window, text string) error

	// SendEnter sends just the Enter key.
	SendEnter(ctx context.Context, session, window string) error

	// SendKeysLiteralWithEnter sends text + Enter atomically.
	// This prevents race conditions where Enter might be lost between separate calls.
	SendKeysLiteralWithEnter(ctx context.Context, session, window, text string) error

	// GetPanePID gets the process ID running in a pane.
	GetPanePID(ctx context.Context, session, window string) (int, error)

	// StartPipePane starts capturing pane output to a file.
	StartPipePane(ctx context.Context, session, window, outputFile string) error

	// StopPipePane stops capturing pane output.
	StopPipePane(ctx context.Context, session, window string) error
}

// Runner manages GitHub Copilot coding agent instances.
type Runner struct {
	// BinaryPath is the path to the copilot binary.
	// Defaults to "copilot" (relies on PATH).
	BinaryPath string

	// Terminal is the terminal runner for sending commands.
	Terminal TerminalRunner

	// StartupDelay is how long to wait after starting Copilot before
	// attempting to get the PID. Defaults to 500ms.
	StartupDelay time.Duration

	// MessageDelay is how long to wait after startup before sending
	// the first message. Defaults to 1s.
	MessageDelay time.Duration

	// AllowAllTools controls whether to pass --allow-all-tools.
	// This is required for non-interactive use. Defaults to true.
	AllowAllTools bool
}

// RunnerOption is a functional option for configuring a Runner.
type RunnerOption func(*Runner)

// WithBinaryPath sets a custom path to the copilot binary.
func WithBinaryPath(path string) RunnerOption {
	return func(r *Runner) {
		r.BinaryPath = path
	}
}

// WithTerminal sets the terminal runner.
func WithTerminal(t TerminalRunner) RunnerOption {
	return func(r *Runner) {
		r.Terminal = t
	}
}

// WithStartupDelay sets the startup delay.
func WithStartupDelay(d time.Duration) RunnerOption {
	return func(r *Runner) {
		r.StartupDelay = d
	}
}

// WithMessageDelay sets the message delay.
func WithMessageDelay(d time.Duration) RunnerOption {
	return func(r *Runner) {
		r.MessageDelay = d
	}
}

// WithAllowAllTools controls whether to pass --allow-all-tools.
// Set to false to require interactive tool approval prompts.
func WithAllowAllTools(allow bool) RunnerOption {
	return func(r *Runner) {
		r.AllowAllTools = allow
	}
}

// NewRunner creates a new Copilot runner with the given options.
func NewRunner(opts ...RunnerOption) *Runner {
	r := &Runner{
		BinaryPath:    "copilot",
		StartupDelay:  500 * time.Millisecond,
		MessageDelay:  1 * time.Second,
		AllowAllTools: true,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// ResolveBinaryPath attempts to find the copilot binary in PATH.
// Returns the full path if found, otherwise returns "copilot".
func ResolveBinaryPath() string {
	if path, err := exec.LookPath("copilot"); err == nil {
		return path
	}
	return "copilot"
}

// IsBinaryAvailable checks if the Copilot CLI is installed and available.
func (r *Runner) IsBinaryAvailable() bool {
	cmd := exec.Command(r.BinaryPath, "--version")
	return cmd.Run() == nil
}

// Config contains configuration for starting a Copilot instance.
type Config struct {
	// SessionID is the unique identifier for this Copilot session.
	// If empty, a new UUID will be generated.
	//
	// Copilot uses --resume <uuid> for both new and resumed sessions.
	// When starting fresh, a new UUID is passed to --resume to seed the session.
	// When resuming, the existing UUID is reused.
	SessionID string

	// Resume indicates this is resuming an existing session rather than starting fresh.
	// Both new and resumed sessions use --resume <uuid>.
	// This field exists to track intent — the command is the same either way.
	Resume bool

	// WorkDir is the working directory for Copilot.
	// If non-empty, the command will cd to this directory before launching Copilot.
	WorkDir string

	// SystemPromptFile is the path to a file containing the agent definition.
	// Copilot uses --agent <path> to load agent definitions.
	// The file must be located under ~/.copilot/agents/ for Copilot to find it.
	SystemPromptFile string

	// InitialMessage is an optional message to send to Copilot after startup.
	// If non-empty, sent after MessageDelay.
	InitialMessage string

	// OutputFile is the path to capture Copilot's output.
	// If non-empty, StartPipePane is called with this file.
	OutputFile string

	// MOTD is an optional message of the day to display before starting Copilot.
	// If empty, no MOTD is displayed.
	MOTD string
}

// StartResult contains information about a started Copilot instance.
type StartResult struct {
	// SessionID is the session ID used for this Copilot instance.
	SessionID string

	// PID is the process ID of the Copilot process.
	PID int

	// Command is the full command that was executed.
	Command string
}

// Start launches Copilot in the specified tmux session/window.
func (r *Runner) Start(ctx context.Context, session, window string, cfg Config) (*StartResult, error) {
	if r.Terminal == nil {
		return nil, fmt.Errorf("terminal runner not configured")
	}

	// Generate session ID if not provided
	sessionID := cfg.SessionID
	if sessionID == "" {
		var err error
		sessionID, err = GenerateSessionID()
		if err != nil {
			return nil, fmt.Errorf("failed to generate session ID: %w", err)
		}
	}

	// Build the command
	cmd := r.buildCommand(sessionID, cfg)

	// Start output capture if configured
	if cfg.OutputFile != "" {
		if err := r.Terminal.StartPipePane(ctx, session, window, cfg.OutputFile); err != nil {
			return nil, fmt.Errorf("failed to start output capture: %w", err)
		}
	}

	// Print MOTD before starting Copilot if configured
	if cfg.MOTD != "" {
		motd := fmt.Sprintf("echo %q", cfg.MOTD)
		if err := r.Terminal.SendKeys(ctx, session, window, motd); err != nil {
			// Non-fatal - just continue
		}
	}

	// Send the command to start Copilot
	if err := r.Terminal.SendKeys(ctx, session, window, cmd); err != nil {
		return nil, fmt.Errorf("failed to send copilot command: %w", err)
	}

	// Wait for Copilot to start (respecting context)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(r.StartupDelay):
	}

	// Get the PID
	pid, err := r.Terminal.GetPanePID(ctx, session, window)
	if err != nil {
		return nil, fmt.Errorf("failed to get Copilot PID: %w", err)
	}

	// Send initial message if configured
	if cfg.InitialMessage != "" {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(r.MessageDelay):
		}
		if err := r.Terminal.SendKeysLiteralWithEnter(ctx, session, window, cfg.InitialMessage); err != nil {
			return nil, fmt.Errorf("failed to send initial message: %w", err)
		}
	}

	return &StartResult{
		SessionID: sessionID,
		PID:       pid,
		Command:   cmd,
	}, nil
}

// buildCommand constructs the copilot CLI command string.
//
// Copilot uses --resume <uuid> for both new and existing sessions:
//   - New session: copilot --resume <new-uuid> --allow-all-tools
//   - Resume session: copilot --resume <existing-uuid> --allow-all-tools
func (r *Runner) buildCommand(sessionID string, cfg Config) string {
	var cmd string

	// If WorkDir is specified, cd to that directory first
	if cfg.WorkDir != "" {
		cmd = fmt.Sprintf("cd %q && ", cfg.WorkDir)
	}

	cmd += r.BinaryPath

	// Copilot uses --resume for both new and resumed sessions
	cmd += fmt.Sprintf(" --resume %s", sessionID)

	// Add allow-all-tools flag for non-interactive use
	if r.AllowAllTools {
		cmd += " --allow-all-tools"
	}

	// Add agent definition file
	if cfg.SystemPromptFile != "" {
		cmd += fmt.Sprintf(" --agent %s", cfg.SystemPromptFile)
	}

	return cmd
}

// SendMessage sends a message to a running Copilot instance.
// This properly handles multiline messages using paste-buffer and sends
// text + Enter atomically to prevent race conditions.
func (r *Runner) SendMessage(ctx context.Context, session, window, message string) error {
	if r.Terminal == nil {
		return fmt.Errorf("terminal runner not configured")
	}

	if err := r.Terminal.SendKeysLiteralWithEnter(ctx, session, window, message); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// GenerateSessionID generates a UUID v4 session ID.
func GenerateSessionID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate session ID: %w", err)
	}

	// Set version (4) and variant bits for UUID v4
	bytes[6] = (bytes[6] & 0x0f) | 0x40 // Version 4
	bytes[8] = (bytes[8] & 0x3f) | 0x80 // Variant 10

	return fmt.Sprintf("%x-%x-%x-%x-%x",
		bytes[0:4],
		bytes[4:6],
		bytes[6:8],
		bytes[8:10],
		bytes[10:16],
	), nil
}
