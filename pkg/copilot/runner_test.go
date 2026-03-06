package copilot

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

type mockTerminal struct {
	sendKeysCalls                 []sendKeysCall
	sendKeysLiteralCalls          []sendKeysCall
	sendKeysLiteralWithEnterCalls []sendKeysCall
	sendEnterCalls                []targetCall
	getPanePIDCalls               []targetCall
	startPipePaneCalls            []pipePaneCall
	stopPipePaneCalls             []targetCall

	getPanePIDReturn int
	getPanePIDError  error
	sendKeysError    error
}

type sendKeysCall struct {
	session string
	window  string
	text    string
}

type targetCall struct {
	session string
	window  string
}

type pipePaneCall struct {
	session    string
	window     string
	outputFile string
}

func (m *mockTerminal) SendKeys(ctx context.Context, session, window, text string) error {
	m.sendKeysCalls = append(m.sendKeysCalls, sendKeysCall{session, window, text})
	return m.sendKeysError
}

func (m *mockTerminal) SendKeysLiteral(ctx context.Context, session, window, text string) error {
	m.sendKeysLiteralCalls = append(m.sendKeysLiteralCalls, sendKeysCall{session, window, text})
	return m.sendKeysError
}

func (m *mockTerminal) SendKeysLiteralWithEnter(ctx context.Context, session, window, text string) error {
	m.sendKeysLiteralWithEnterCalls = append(m.sendKeysLiteralWithEnterCalls, sendKeysCall{session, window, text})
	return m.sendKeysError
}

func (m *mockTerminal) SendEnter(ctx context.Context, session, window string) error {
	m.sendEnterCalls = append(m.sendEnterCalls, targetCall{session, window})
	return nil
}

func (m *mockTerminal) GetPanePID(ctx context.Context, session, window string) (int, error) {
	m.getPanePIDCalls = append(m.getPanePIDCalls, targetCall{session, window})
	return m.getPanePIDReturn, m.getPanePIDError
}

func (m *mockTerminal) StartPipePane(ctx context.Context, session, window, outputFile string) error {
	m.startPipePaneCalls = append(m.startPipePaneCalls, pipePaneCall{session, window, outputFile})
	return nil
}

func (m *mockTerminal) StopPipePane(ctx context.Context, session, window string) error {
	m.stopPipePaneCalls = append(m.stopPipePaneCalls, targetCall{session, window})
	return nil
}

func TestNewRunner(t *testing.T) {
	runner := NewRunner()
	if runner == nil {
		t.Fatal("NewRunner() returned nil")
	}
	if runner.BinaryPath != "copilot" {
		t.Errorf("expected default BinaryPath to be 'copilot', got %q", runner.BinaryPath)
	}
	if runner.StartupDelay != 500*time.Millisecond {
		t.Errorf("expected default StartupDelay to be 500ms, got %v", runner.StartupDelay)
	}
	if runner.MessageDelay != 1*time.Second {
		t.Errorf("expected default MessageDelay to be 1s, got %v", runner.MessageDelay)
	}
	if !runner.AllowAllTools {
		t.Error("expected default AllowAllTools to be true")
	}
}

func TestNewRunnerWithOptions(t *testing.T) {
	terminal := &mockTerminal{}
	runner := NewRunner(
		WithBinaryPath("/custom/copilot"),
		WithTerminal(terminal),
		WithStartupDelay(1*time.Second),
		WithMessageDelay(2*time.Second),
		WithAllowAllTools(false),
	)

	if runner.BinaryPath != "/custom/copilot" {
		t.Errorf("expected BinaryPath to be '/custom/copilot', got %q", runner.BinaryPath)
	}
	if runner.Terminal != terminal {
		t.Error("expected Terminal to be set")
	}
	if runner.StartupDelay != 1*time.Second {
		t.Errorf("expected StartupDelay to be 1s, got %v", runner.StartupDelay)
	}
	if runner.MessageDelay != 2*time.Second {
		t.Errorf("expected MessageDelay to be 2s, got %v", runner.MessageDelay)
	}
	if runner.AllowAllTools {
		t.Error("expected AllowAllTools to be false")
	}
}

func TestStart(t *testing.T) {
	ctx := context.Background()
	terminal := &mockTerminal{getPanePIDReturn: 12345}

	// Create a real temp prompt file since installAgentFile reads it
	tmpFile, err := os.CreateTemp("", "test-prompt-*.md")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.WriteString("test prompt content")
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	runner := NewRunner(
		WithTerminal(terminal),
		WithBinaryPath("/path/to/copilot"),
		WithStartupDelay(0),
	)

	result, err := runner.Start(ctx, "my-session", "my-window", Config{
		SystemPromptFile: tmpFile.Name(),
	})
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	if result.SessionID == "" {
		t.Error("expected SessionID to be generated")
	}
	if result.PID != 12345 {
		t.Errorf("expected PID to be 12345, got %d", result.PID)
	}
	if len(terminal.sendKeysCalls) != 1 {
		t.Fatalf("expected 1 SendKeys call, got %d", len(terminal.sendKeysCalls))
	}

	call := terminal.sendKeysCalls[0]
	if call.session != "my-session" {
		t.Errorf("expected session 'my-session', got %q", call.session)
	}
	if !strings.Contains(call.text, "/path/to/copilot") {
		t.Errorf("expected command to contain binary path, got %q", call.text)
	}
	if !strings.Contains(call.text, "--resume") {
		t.Errorf("expected command to contain --resume, got %q", call.text)
	}
	if !strings.Contains(call.text, "--allow-all-tools") {
		t.Errorf("expected command to contain --allow-all-tools, got %q", call.text)
	}
	if !strings.Contains(call.text, "--agent ") {
		t.Errorf("expected command to contain --agent flag, got %q", call.text)
	}
}

func TestStartWithMOTD(t *testing.T) {
	ctx := context.Background()
	terminal := &mockTerminal{getPanePIDReturn: 12345}

	runner := NewRunner(
		WithTerminal(terminal),
		WithBinaryPath("/path/to/copilot"),
		WithStartupDelay(0),
	)

	result, err := runner.Start(ctx, "my-session", "my-window", Config{
		MOTD: "Welcome to the agent session!",
	})
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	if result.SessionID == "" {
		t.Error("expected SessionID to be generated")
	}
	if len(terminal.sendKeysCalls) != 2 {
		t.Fatalf("expected 2 SendKeys calls (MOTD + command), got %d", len(terminal.sendKeysCalls))
	}
	if !strings.Contains(terminal.sendKeysCalls[0].text, "Welcome to the agent session!") {
		t.Errorf("expected MOTD, got %q", terminal.sendKeysCalls[0].text)
	}
	if !strings.Contains(terminal.sendKeysCalls[1].text, "/path/to/copilot") {
		t.Errorf("expected command, got %q", terminal.sendKeysCalls[1].text)
	}
}

func TestStartWithCustomSessionID(t *testing.T) {
	ctx := context.Background()
	terminal := &mockTerminal{getPanePIDReturn: 12345}

	runner := NewRunner(WithTerminal(terminal), WithStartupDelay(0))

	result, err := runner.Start(ctx, "session", "window", Config{
		SessionID: "my-custom-session-id",
	})
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	if result.SessionID != "my-custom-session-id" {
		t.Errorf("expected SessionID 'my-custom-session-id', got %q", result.SessionID)
	}
	if !strings.Contains(terminal.sendKeysCalls[0].text, "--resume my-custom-session-id") {
		t.Errorf("expected --resume with session ID, got %q", terminal.sendKeysCalls[0].text)
	}
}

func TestStartWithOutputCapture(t *testing.T) {
	ctx := context.Background()
	terminal := &mockTerminal{getPanePIDReturn: 12345}

	runner := NewRunner(WithTerminal(terminal), WithStartupDelay(0))

	_, err := runner.Start(ctx, "session", "window", Config{OutputFile: "/tmp/output.log"})
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	if len(terminal.startPipePaneCalls) != 1 {
		t.Fatalf("expected 1 StartPipePane call, got %d", len(terminal.startPipePaneCalls))
	}
	if terminal.startPipePaneCalls[0].outputFile != "/tmp/output.log" {
		t.Errorf("expected outputFile '/tmp/output.log', got %q", terminal.startPipePaneCalls[0].outputFile)
	}
}

func TestStartWithInitialMessage(t *testing.T) {
	ctx := context.Background()
	terminal := &mockTerminal{getPanePIDReturn: 12345}

	runner := NewRunner(WithTerminal(terminal), WithStartupDelay(0), WithMessageDelay(0))

	_, err := runner.Start(ctx, "session", "window", Config{InitialMessage: "Hello, Copilot!"})
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	if len(terminal.sendKeysLiteralWithEnterCalls) != 1 {
		t.Fatalf("expected 1 SendKeysLiteralWithEnter call, got %d", len(terminal.sendKeysLiteralWithEnterCalls))
	}
	if terminal.sendKeysLiteralWithEnterCalls[0].text != "Hello, Copilot!" {
		t.Errorf("expected 'Hello, Copilot!', got %q", terminal.sendKeysLiteralWithEnterCalls[0].text)
	}
}

func TestStartNoTerminal(t *testing.T) {
	ctx := context.Background()
	_, err := NewRunner().Start(ctx, "session", "window", Config{})
	if err == nil {
		t.Error("expected error when terminal not configured")
	}
	if !strings.Contains(err.Error(), "terminal runner not configured") {
		t.Errorf("expected 'terminal runner not configured', got %q", err.Error())
	}
}

func TestStartSendKeysError(t *testing.T) {
	ctx := context.Background()
	terminal := &mockTerminal{sendKeysError: errors.New("send keys failed")}
	runner := NewRunner(WithTerminal(terminal), WithStartupDelay(0))

	_, err := runner.Start(ctx, "session", "window", Config{})
	if err == nil {
		t.Error("expected error when SendKeys fails")
	}
	if !strings.Contains(err.Error(), "send keys failed") {
		t.Errorf("expected 'send keys failed', got %q", err.Error())
	}
}

func TestStartGetPIDError(t *testing.T) {
	ctx := context.Background()
	terminal := &mockTerminal{getPanePIDError: errors.New("get PID failed")}
	runner := NewRunner(WithTerminal(terminal), WithStartupDelay(0))

	_, err := runner.Start(ctx, "session", "window", Config{})
	if err == nil {
		t.Error("expected error when GetPanePID fails")
	}
	if !strings.Contains(err.Error(), "get PID failed") {
		t.Errorf("expected 'get PID failed', got %q", err.Error())
	}
}

func TestStartContextCancellation(t *testing.T) {
	terminal := &mockTerminal{getPanePIDReturn: 12345}
	runner := NewRunner(WithTerminal(terminal), WithStartupDelay(100*time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := runner.Start(ctx, "session", "window", Config{})
	if err == nil {
		t.Error("expected error when context is cancelled")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestSendMessage(t *testing.T) {
	ctx := context.Background()
	terminal := &mockTerminal{}
	runner := NewRunner(WithTerminal(terminal))

	err := runner.SendMessage(ctx, "session", "window", "Hello, Copilot!")
	if err != nil {
		t.Fatalf("SendMessage() failed: %v", err)
	}
	if len(terminal.sendKeysLiteralWithEnterCalls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(terminal.sendKeysLiteralWithEnterCalls))
	}
	if terminal.sendKeysLiteralWithEnterCalls[0].text != "Hello, Copilot!" {
		t.Errorf("expected 'Hello, Copilot!', got %q", terminal.sendKeysLiteralWithEnterCalls[0].text)
	}
}

func TestSendMessageMultiline(t *testing.T) {
	ctx := context.Background()
	terminal := &mockTerminal{}
	runner := NewRunner(WithTerminal(terminal))

	msg := "Line 1\nLine 2\nLine 3"
	if err := runner.SendMessage(ctx, "session", "window", msg); err != nil {
		t.Fatalf("SendMessage() failed: %v", err)
	}
	if terminal.sendKeysLiteralWithEnterCalls[0].text != msg {
		t.Errorf("expected multiline message preserved, got %q", terminal.sendKeysLiteralWithEnterCalls[0].text)
	}
}

func TestSendMessageNoTerminal(t *testing.T) {
	ctx := context.Background()
	err := NewRunner().SendMessage(ctx, "session", "window", "Hello")
	if err == nil {
		t.Error("expected error when terminal not configured")
	}
}

func TestGenerateSessionID(t *testing.T) {
	id1, err := GenerateSessionID()
	if err != nil {
		t.Fatalf("GenerateSessionID() failed: %v", err)
	}
	parts := strings.Split(id1, "-")
	if len(parts) != 5 {
		t.Errorf("expected 5 parts in UUID, got %d", len(parts))
	}

	id2, _ := GenerateSessionID()
	if id1 == id2 {
		t.Error("expected different session IDs")
	}
}

func TestBuildCommand(t *testing.T) {
	runner := NewRunner(WithBinaryPath("/path/to/copilot"), WithAllowAllTools(true))

	tests := []struct {
		name     string
		config   Config
		contains []string
	}{
		{
			name:   "basic",
			config: Config{SessionID: "test-session"},
			contains: []string{
				"/path/to/copilot",
				"--resume test-session",
				"--allow-all-tools",
			},
		},
		{
			name:   "with prompt file",
			config: Config{SessionID: "test-session", SystemPromptFile: "/path/to/prompt.md"},
			contains: []string{
				"--agent /path/to/prompt.md",
			},
		},
		{
			name:   "with workdir",
			config: Config{SessionID: "test-session", WorkDir: "/path/to/workdir"},
			contains: []string{
				`cd "/path/to/workdir" &&`,
				"/path/to/copilot",
				"--add-dir /path/to/workdir",
			},
		},
		{
			name:   "resume uses same --resume flag",
			config: Config{SessionID: "existing-session", Resume: true},
			contains: []string{
				"--resume existing-session",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := runner.buildCommand(tc.config.SessionID, tc.config)
			for _, s := range tc.contains {
				if !strings.Contains(cmd, s) {
					t.Errorf("expected command to contain %q, got %q", s, cmd)
				}
			}
		})
	}
}

func TestBuildCommandWithoutAllowAllTools(t *testing.T) {
	runner := NewRunner(WithBinaryPath("copilot"), WithAllowAllTools(false))
	cmd := runner.buildCommand("session-id", Config{})
	if strings.Contains(cmd, "--allow-all-tools") {
		t.Error("expected command not to contain --allow-all-tools when disabled")
	}
}

func TestResolveBinaryPath(t *testing.T) {
	path := ResolveBinaryPath()
	if path == "" {
		t.Error("ResolveBinaryPath() returned empty string")
	}
}

func TestIsBinaryAvailable(t *testing.T) {
	runner := NewRunner(WithBinaryPath("echo"))
	if !runner.IsBinaryAvailable() {
		t.Error("IsBinaryAvailable() should return true for 'echo'")
	}

	runner = NewRunner(WithBinaryPath("/nonexistent/binary/path"))
	if runner.IsBinaryAvailable() {
		t.Error("IsBinaryAvailable() should return false for nonexistent binary")
	}
}
