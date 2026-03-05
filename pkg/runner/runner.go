// Package runner defines the common interface for AI agent runners.
//
// Both pkg/claude and pkg/copilot implement the Runner interface, allowing
// the rest of the codebase to work with either provider generically.
package runner

import (
	"context"
	"fmt"

	"github.com/dlorenc/multiclaude/pkg/claude"
	"github.com/dlorenc/multiclaude/pkg/copilot"
)

// Provider identifies which AI agent backend to use.
type Provider string

const (
	ProviderClaude  Provider = "claude"
	ProviderCopilot Provider = "copilot"
)

// ValidProviders lists all supported provider values.
var ValidProviders = []Provider{ProviderClaude, ProviderCopilot}

// IsValid returns true if the provider is a recognized value.
func (p Provider) IsValid() bool {
	for _, v := range ValidProviders {
		if p == v {
			return true
		}
	}
	return false
}

// String returns the provider name.
func (p Provider) String() string {
	return string(p)
}

// DefaultProvider is the provider used when none is specified.
const DefaultProvider = ProviderClaude

// TerminalRunner abstracts terminal interaction for running agents.
// The tmux.Client implements this interface.
type TerminalRunner interface {
	SendKeys(ctx context.Context, session, window, text string) error
	SendKeysLiteral(ctx context.Context, session, window, text string) error
	SendEnter(ctx context.Context, session, window string) error
	SendKeysLiteralWithEnter(ctx context.Context, session, window, text string) error
	GetPanePID(ctx context.Context, session, window string) (int, error)
	StartPipePane(ctx context.Context, session, window, outputFile string) error
	StopPipePane(ctx context.Context, session, window string) error
}

// Config contains provider-agnostic configuration for starting an agent.
type Config struct {
	SessionID        string
	Resume           bool
	WorkDir          string
	SystemPromptFile string
	InitialMessage   string
	OutputFile       string
	MOTD             string
}

// StartResult contains information about a started agent instance.
type StartResult struct {
	SessionID string
	PID       int
	Command   string
}

// Runner is the common interface for AI agent runners.
type Runner interface {
	Start(ctx context.Context, session, window string, cfg Config) (*StartResult, error)
	SendMessage(ctx context.Context, session, window, message string) error
	IsBinaryAvailable() bool
}

// claudeAdapter wraps claude.Runner to implement Runner.
type claudeAdapter struct {
	r *claude.Runner
}

func (a *claudeAdapter) Start(ctx context.Context, session, window string, cfg Config) (*StartResult, error) {
	result, err := a.r.Start(ctx, session, window, claude.Config{
		SessionID:        cfg.SessionID,
		Resume:           cfg.Resume,
		WorkDir:          cfg.WorkDir,
		SystemPromptFile: cfg.SystemPromptFile,
		InitialMessage:   cfg.InitialMessage,
		OutputFile:       cfg.OutputFile,
		MOTD:             cfg.MOTD,
	})
	if err != nil {
		return nil, err
	}
	return &StartResult{
		SessionID: result.SessionID,
		PID:       result.PID,
		Command:   result.Command,
	}, nil
}

func (a *claudeAdapter) SendMessage(ctx context.Context, session, window, message string) error {
	return a.r.SendMessage(ctx, session, window, message)
}

func (a *claudeAdapter) IsBinaryAvailable() bool {
	return a.r.IsBinaryAvailable()
}

// copilotAdapter wraps copilot.Runner to implement Runner.
type copilotAdapter struct {
	r *copilot.Runner
}

func (a *copilotAdapter) Start(ctx context.Context, session, window string, cfg Config) (*StartResult, error) {
	result, err := a.r.Start(ctx, session, window, copilot.Config{
		SessionID:        cfg.SessionID,
		Resume:           cfg.Resume,
		WorkDir:          cfg.WorkDir,
		SystemPromptFile: cfg.SystemPromptFile,
		InitialMessage:   cfg.InitialMessage,
		OutputFile:       cfg.OutputFile,
		MOTD:             cfg.MOTD,
	})
	if err != nil {
		return nil, err
	}
	return &StartResult{
		SessionID: result.SessionID,
		PID:       result.PID,
		Command:   result.Command,
	}, nil
}

func (a *copilotAdapter) SendMessage(ctx context.Context, session, window, message string) error {
	return a.r.SendMessage(ctx, session, window, message)
}

func (a *copilotAdapter) IsBinaryAvailable() bool {
	return a.r.IsBinaryAvailable()
}

// NewRunner creates a Runner for the given provider using the specified terminal.
func NewRunner(provider Provider, terminal TerminalRunner) (Runner, error) {
	switch provider {
	case ProviderClaude:
		r := claude.NewRunner(claude.WithTerminal(terminal))
		return &claudeAdapter{r: r}, nil
	case ProviderCopilot:
		r := copilot.NewRunner(copilot.WithTerminal(terminal))
		return &copilotAdapter{r: r}, nil
	default:
		return nil, fmt.Errorf("unknown provider: %q", provider)
	}
}

// GenerateSessionID generates a new UUID v4 session ID.
// Delegates to the claude package (both use the same UUID format).
func GenerateSessionID() (string, error) {
	return claude.GenerateSessionID()
}

// ResolveBinaryPath finds the binary for the given provider in PATH.
func ResolveBinaryPath(provider Provider) string {
	switch provider {
	case ProviderCopilot:
		return copilot.ResolveBinaryPath()
	default:
		return claude.ResolveBinaryPath()
	}
}

// BinaryName returns the expected binary name for the provider.
func BinaryName(provider Provider) string {
	switch provider {
	case ProviderCopilot:
		return "copilot"
	default:
		return "claude"
	}
}
