// Package copilot provides utilities for programmatically running GitHub Copilot CLI.
//
// This package abstracts the details of launching and interacting with GitHub Copilot
// coding agent instances running in terminal emulators. It handles:
//
//   - CLI flag construction (--resume, --allow-all-tools)
//   - Session ID generation (UUID v4)
//   - Startup timing quirks
//   - Terminal integration via the [TerminalRunner] interface
//
// # Installation
//
//	go get github.com/dlorenc/multiclaude/pkg/copilot
//
// # Requirements
//
// This package requires the GitHub Copilot coding agent CLI to be installed.
// The binary is typically named "copilot" and should be available in PATH.
// Use [ResolveBinaryPath] to find it, and [Runner.IsBinaryAvailable] to verify
// it's installed before use.
//
// # Example Usage
//
//	package main
//
//	import (
//	    "log"
//	    "github.com/dlorenc/multiclaude/pkg/copilot"
//	    "github.com/dlorenc/multiclaude/pkg/tmux"
//	)
//
//	func main() {
//	    // Create terminal runner (tmux in this case)
//	    tmuxClient := tmux.NewClient()
//
//	    // Create Copilot runner with tmux as the terminal
//	    runner := copilot.NewRunner(
//	        copilot.WithTerminal(tmuxClient),
//	        copilot.WithBinaryPath(copilot.ResolveBinaryPath()),
//	    )
//
//	    // Verify Copilot CLI is available
//	    if !runner.IsBinaryAvailable() {
//	        log.Fatal("Copilot CLI is not installed")
//	    }
//
//	    // Prepare a session
//	    tmuxClient.CreateSession("demo", true)
//	    tmuxClient.CreateWindow("demo", "copilot")
//
//	    // Start Copilot
//	    result, err := runner.Start("demo", "copilot", copilot.Config{
//	        SystemPromptFile: "/path/to/prompt.md",
//	        OutputFile:       "/tmp/copilot-output.log",
//	    })
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//
//	    log.Printf("Copilot started with session ID: %s, PID: %d", result.SessionID, result.PID)
//
//	    // Send a message
//	    if err := runner.SendMessage("demo", "copilot", "Hello, Copilot!"); err != nil {
//	        log.Fatal(err)
//	    }
//	}
//
// # The TerminalRunner Interface
//
// The [TerminalRunner] interface abstracts terminal operations, allowing this package
// to work with any terminal emulator that implements it. The [pkg/tmux.Client] provides
// a ready-to-use implementation for tmux.
//
// # Session Management
//
// Each Copilot instance is identified by a session ID (UUID v4). This allows:
//
//   - Resuming sessions across process restarts
//   - Tracking multiple concurrent Copilot instances
//   - Correlating logs with specific sessions
//
// Use [GenerateSessionID] to create new session IDs, or provide your own via [Config.SessionID].
//
// # Timing Considerations
//
// Starting Copilot and sending messages requires careful timing:
//
//   - [Runner.StartupDelay] (default 500ms): Wait after launching before getting PID
//   - [Runner.MessageDelay] (default 1s): Wait before sending initial message
//
// These can be adjusted via [WithStartupDelay] and [WithMessageDelay] options.
package copilot
