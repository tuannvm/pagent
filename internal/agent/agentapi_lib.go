package agent

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/coder/agentapi/lib/httpapi"
	"github.com/coder/agentapi/lib/logctx"
	"github.com/coder/agentapi/lib/msgfmt"
	"github.com/coder/agentapi/lib/termexec"
)

// LibClient provides direct library integration with agentapi
// instead of spawning the agentapi binary and communicating via HTTP
type LibClient struct {
	process *termexec.Process
	server  *httpapi.Server
	emitter *httpapi.EventEmitter
	port    int
	verbose bool
	logger  *slog.Logger
	ctx     context.Context
}

// LibClientConfig configures the library client
type LibClientConfig struct {
	Port          int
	Verbose       bool
	AgentCmd      string   // e.g., "claude"
	AgentArgs     []string // additional args for the agent
	TerminalWidth uint16
	TerminalHeight uint16
}

// NewLibClient creates a new agentapi library client
func NewLibClient(ctx context.Context, cfg LibClientConfig) (*LibClient, error) {
	if cfg.TerminalWidth == 0 {
		cfg.TerminalWidth = 180
	}
	if cfg.TerminalHeight == 0 {
		cfg.TerminalHeight = 50
	}
	if cfg.AgentCmd == "" {
		cfg.AgentCmd = "claude"
	}

	// Create logger - agentapi requires it in context
	var logger *slog.Logger
	if cfg.Verbose {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	} else {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	// Add logger to context - required by agentapi library
	ctx = logctx.WithLogger(ctx, logger)

	// Start the agent process directly using termexec
	process, err := termexec.StartProcess(ctx, termexec.StartProcessConfig{
		Program:        cfg.AgentCmd,
		Args:           cfg.AgentArgs,
		TerminalWidth:  cfg.TerminalWidth,
		TerminalHeight: cfg.TerminalHeight,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start agent process: %w", err)
	}

	// Create event emitter for status tracking
	emitter := httpapi.NewEventEmitter(100)

	// Create HTTP server using the library
	server, err := httpapi.NewServer(ctx, httpapi.ServerConfig{
		AgentType:      msgfmt.AgentTypeClaude,
		Process:        process,
		Port:           cfg.Port,
		AllowedHosts:   []string{"localhost", "127.0.0.1"},
		AllowedOrigins: []string{"http://localhost", "http://127.0.0.1"},
	})
	if err != nil {
		_ = process.Close(logger, 5*time.Second)
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	client := &LibClient{
		process: process,
		server:  server,
		emitter: emitter,
		port:    cfg.Port,
		verbose: cfg.Verbose,
		logger:  logger,
		ctx:     ctx,
	}

	// Start the snapshot loop - this is critical for status detection!
	// The snapshot loop monitors the terminal screen and updates the agent status
	server.StartSnapshotLoop(ctx)

	return client, nil
}

// Start begins serving the HTTP API (non-blocking)
func (c *LibClient) Start() error {
	// Start server in goroutine since Start() blocks
	errCh := make(chan error, 1)
	go func() {
		if err := c.server.Start(); err != nil {
			errCh <- err
		}
	}()

	// Give the server a moment to start and check for immediate errors
	select {
	case err := <-errCh:
		return fmt.Errorf("server failed to start: %w", err)
	case <-time.After(100 * time.Millisecond):
		// Server started successfully (no immediate error)
		return nil
	}
}

// Handler returns the HTTP handler for custom server setups
func (c *LibClient) Handler() http.Handler {
	return c.server.Handler()
}

// Port returns the port the server is listening on
func (c *LibClient) Port() int {
	return c.port
}

// SendMessage sends a message to the agent
func (c *LibClient) SendMessage(content string) error {
	// Write directly to the process
	_, err := c.process.Write([]byte(content + "\n"))
	return err
}

// ReadScreen returns the current terminal screen content
func (c *LibClient) ReadScreen() string {
	return c.process.ReadScreen()
}

// WaitForStable waits for the agent to reach stable state
func (c *LibClient) WaitForStable(timeout time.Duration) error {
	start := time.Now()
	pollInterval := 500 * time.Millisecond
	lastScreen := ""
	stableCount := 0
	requiredStableChecks := 3 // require 3 consecutive stable reads

	for {
		if time.Since(start) > timeout {
			return fmt.Errorf("timeout waiting for stable state")
		}

		screen := c.process.ReadScreen()

		// Check if screen has stabilized (no changes)
		if screen == lastScreen {
			stableCount++
			if stableCount >= requiredStableChecks {
				// Additional check: look for common "ready" indicators
				if c.isAgentReady(screen) {
					return nil
				}
			}
		} else {
			stableCount = 0
			lastScreen = screen
		}

		time.Sleep(pollInterval)
	}
}

// isAgentReady checks screen content for ready indicators
func (c *LibClient) isAgentReady(screen string) bool {
	// Claude Code typically shows a prompt indicator when ready
	readyIndicators := []string{
		">",           // Common prompt
		"claude>",     // Claude prompt
		"$",           // Shell prompt after completion
		"completed",   // Task completion message
	}

	lowerScreen := strings.ToLower(screen)
	for _, indicator := range readyIndicators {
		if strings.Contains(lowerScreen, indicator) {
			return true
		}
	}

	// If no activity for a while, consider it stable
	return true
}

// WaitForCompletion waits for the agent to finish processing a task
func (c *LibClient) WaitForCompletion(ctx context.Context, timeout time.Duration) error {
	start := time.Now()
	pollInterval := 1 * time.Second
	lastScreen := ""
	stableCount := 0
	wasRunning := false
	requiredStableChecks := 5 // require 5 seconds of no changes

	for {
		if timeout > 0 && time.Since(start) > timeout {
			return fmt.Errorf("timeout waiting for completion")
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		screen := c.process.ReadScreen()

		// Detect if agent is running (screen is changing)
		if screen != lastScreen {
			wasRunning = true
			stableCount = 0
			lastScreen = screen

			if c.verbose {
				fmt.Printf("[LIB] Screen updated (elapsed: %s)\n", time.Since(start).Round(time.Second))
			}
		} else {
			stableCount++

			// Agent is complete when it was running and now stable
			if wasRunning && stableCount >= requiredStableChecks {
				if c.verbose {
					fmt.Printf("[LIB] Agent completed (elapsed: %s)\n", time.Since(start).Round(time.Second))
				}
				return nil
			}
		}

		time.Sleep(pollInterval)
	}
}

// Close shuts down the agent and server
func (c *LibClient) Close(ctx context.Context) error {
	var errs []error

	if c.server != nil {
		if err := c.server.Stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("server stop: %w", err))
		}
	}

	if c.process != nil {
		if err := c.process.Close(c.logger, 10*time.Second); err != nil {
			errs = append(errs, fmt.Errorf("process close: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}
	return nil
}

// GetProcess returns the underlying termexec.Process for advanced usage
func (c *LibClient) GetProcess() *termexec.Process {
	return c.process
}
