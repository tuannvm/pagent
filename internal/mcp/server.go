package mcp

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	oauth "github.com/tuannvm/oauth-mcp-proxy"
	mcpoauth "github.com/tuannvm/oauth-mcp-proxy/mcp"
)

const (
	// ServerName is the MCP server name.
	ServerName = "pagent"
	// ServerVersion is the MCP server version.
	ServerVersion = "1.0.0"
)

// ServerInstructions provides usage guidance for LLMs.
const ServerInstructions = `Pagent is a PRD-to-code orchestration tool that transforms Product Requirement Documents into working code using specialized Claude Code agents.

Available tools:
- run_agent: Run a single agent (architect, qa, security, implementer, verifier)
- run_pipeline: Run the full agent pipeline with dependency resolution
- list_agents: List all available agents and their dependencies
- get_status: Check status of running agents
- send_message: Send guidance to a running agent
- stop_agents: Stop running agents

Typical workflow:
1. Use list_agents to understand available agents
2. Use run_pipeline with a PRD file to generate architecture, tests, security assessment, and code
3. Monitor progress with get_status
4. Send corrections with send_message if needed`

// ServerConfig holds configuration for creating an MCP server.
type ServerConfig struct {
	Name         string
	Version      string
	Instructions string
	Logger       *slog.Logger
	Handlers     *Handlers

	// Transport settings
	Port           int
	SessionTimeout time.Duration

	// OAuth settings (optional)
	OAuth *OAuthConfig
}

// OAuthConfig holds OAuth-specific configuration.
type OAuthConfig struct {
	Provider  string // okta, google, azure, hmac
	Issuer    string
	Audience  string
	ServerURL string // Base URL for OAuth callbacks (e.g., https://example.com:8080)
}

// DefaultServerConfig returns a ServerConfig with sensible defaults.
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Name:           ServerName,
		Version:        ServerVersion,
		Instructions:   ServerInstructions,
		Logger:         slog.Default(),
		Handlers:       NewHandlers(),
		Port:           8080,
		SessionTimeout: 30 * time.Minute,
	}
}

// Server represents the MCP server with all components.
type Server struct {
	mcpServer   *mcp.Server
	config      *ServerConfig
	oauthServer *oauth.Server
}

// NewServer creates a new MCP server instance with all components.
func NewServer(cfg *ServerConfig) *Server {
	if cfg == nil {
		cfg = DefaultServerConfig()
	}
	if cfg.Name == "" {
		cfg.Name = ServerName
	}
	if cfg.Version == "" {
		cfg.Version = ServerVersion
	}
	if cfg.Instructions == "" {
		cfg.Instructions = ServerInstructions
	}
	if cfg.Handlers == nil {
		cfg.Handlers = NewHandlers()
	}
	if cfg.Port == 0 {
		cfg.Port = 8080
	}
	if cfg.SessionTimeout == 0 {
		cfg.SessionTimeout = 30 * time.Minute
	}

	mcpServer := mcp.NewServer(
		&mcp.Implementation{
			Name:    cfg.Name,
			Version: cfg.Version,
		},
		&mcp.ServerOptions{
			Instructions: cfg.Instructions,
			Logger:       cfg.Logger,
		},
	)

	// Register all tools
	registerTools(mcpServer, cfg.Handlers)

	return &Server{
		mcpServer: mcpServer,
		config:    cfg,
	}
}

// ServeStdio starts the MCP server with STDIO transport.
func (s *Server) ServeStdio() error {
	log.Println("Starting pagent MCP server on stdio transport")
	return s.mcpServer.Run(context.Background(), &mcp.StdioTransport{})
}

// ServeHTTP starts the MCP server with streamable HTTP transport.
func (s *Server) ServeHTTP() error {
	mux := http.NewServeMux()

	// Create streamable HTTP handler (2025-11-25 spec)
	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return s.mcpServer
	}, &mcp.StreamableHTTPOptions{
		SessionTimeout: s.config.SessionTimeout,
		Logger:         s.config.Logger,
	})

	mux.Handle("/mcp", handler)
	s.addHealthCheck(mux)

	addr := fmt.Sprintf(":%d", s.config.Port)
	log.Printf("Starting pagent MCP server on http://localhost%s/mcp", addr)
	log.Printf("Health check: http://localhost%s/health", addr)

	return s.runHTTPServer(addr, mux)
}

// ServeHTTPWithOAuth starts the MCP server with OAuth 2.1 authentication.
func (s *Server) ServeHTTPWithOAuth() error {
	if s.config.OAuth == nil {
		return fmt.Errorf("OAuth configuration is required")
	}

	// Use configured ServerURL or fall back to localhost
	serverURL := s.config.OAuth.ServerURL
	if serverURL == "" {
		serverURL = fmt.Sprintf("http://localhost:%d", s.config.Port)
	}

	mux := http.NewServeMux()

	// Create OAuth-protected handler
	oauthServer, handler, err := mcpoauth.WithOAuth(mux, &oauth.Config{
		Provider:  s.config.OAuth.Provider,
		Issuer:    s.config.OAuth.Issuer,
		Audience:  s.config.OAuth.Audience,
		ServerURL: serverURL,
	}, s.mcpServer)
	if err != nil {
		return fmt.Errorf("failed to create OAuth server: %w", err)
	}
	s.oauthServer = oauthServer

	mux.Handle("/mcp", handler)
	s.addHealthCheck(mux)

	addr := fmt.Sprintf(":%d", s.config.Port)
	log.Printf("Starting pagent MCP server with OAuth on %s/mcp", serverURL)
	log.Printf("OAuth provider: %s", s.config.OAuth.Provider)
	log.Printf("OAuth issuer: %s", s.config.OAuth.Issuer)
	s.oauthServer.LogStartup(false)

	return s.runHTTPServer(addr, mux)
}

// addHealthCheck adds a health check endpoint to the mux.
func (s *Server) addHealthCheck(mux *http.ServeMux) {
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"status":"ok","version":"%s"}`, s.config.Version)
	})
}

// runHTTPServer runs an HTTP server with graceful shutdown.
func (s *Server) runHTTPServer(addr string, handler http.Handler) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second, // Allow time for long MCP operations
		IdleTimeout:       120 * time.Second,
	}

	// Graceful shutdown
	errCh := make(chan error, 1)
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		errCh <- srv.Shutdown(ctx)
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}

	return <-errCh
}

// boolPtr creates a pointer to a bool value.
func boolPtr(b bool) *bool {
	return &b
}

// registerTools registers all pagent tools with the MCP server.
func registerTools(server *mcp.Server, h *Handlers) {
	registerRunAgentTool(server, h)
	registerRunPipelineTool(server, h)
	registerListAgentsTool(server, h)
	registerGetStatusTool(server, h)
	registerSendMessageTool(server, h)
	registerStopAgentsTool(server, h)
}

func registerRunAgentTool(server *mcp.Server, h *Handlers) {
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "run_agent",
			Description: "Run a single pagent agent on a PRD file. Agents: architect (creates architecture), qa (test plan), security (threat model), implementer (code), verifier (validation).",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Run Agent",
				ReadOnlyHint:    false,
				DestructiveHint: boolPtr(false),
				IdempotentHint:  false,
				OpenWorldHint:   boolPtr(true),
			},
		},
		func(ctx context.Context, req *mcp.CallToolRequest, input RunAgentInput) (*mcp.CallToolResult, RunAgentOutput, error) {
			return nil, h.RunAgent(ctx, input), nil
		},
	)
}

func registerRunPipelineTool(server *mcp.Server, h *Handlers) {
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "run_pipeline",
			Description: "Run the full pagent pipeline on a PRD file. Executes agents in dependency order: architect -> qa/security (parallel) -> implementer -> verifier.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Run Pipeline",
				ReadOnlyHint:    false,
				DestructiveHint: boolPtr(false),
				IdempotentHint:  false,
				OpenWorldHint:   boolPtr(true),
			},
		},
		func(ctx context.Context, req *mcp.CallToolRequest, input RunPipelineInput) (*mcp.CallToolResult, RunPipelineOutput, error) {
			output, err := h.RunPipeline(ctx, input)
			return nil, output, err
		},
	)
}

func registerListAgentsTool(server *mcp.Server, h *Handlers) {
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "list_agents",
			Description: "List all available pagent agents with their outputs and dependencies.",
			Annotations: &mcp.ToolAnnotations{
				Title:          "List Agents",
				ReadOnlyHint:   true,
				IdempotentHint: true,
				OpenWorldHint:  boolPtr(false),
			},
		},
		func(ctx context.Context, req *mcp.CallToolRequest, input ListAgentsInput) (*mcp.CallToolResult, ListAgentsOutput, error) {
			return nil, h.ListAgents(ctx, input), nil
		},
	)
}

func registerGetStatusTool(server *mcp.Server, h *Handlers) {
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "get_status",
			Description: "Get the status of running pagent agents. Returns agent name, port, and current status (running/stable).",
			Annotations: &mcp.ToolAnnotations{
				Title:          "Get Status",
				ReadOnlyHint:   true,
				IdempotentHint: true,
				OpenWorldHint:  boolPtr(false),
			},
		},
		func(ctx context.Context, req *mcp.CallToolRequest, input GetStatusInput) (*mcp.CallToolResult, GetStatusOutput, error) {
			return nil, h.GetStatus(ctx, input), nil
		},
	)
}

func registerSendMessageTool(server *mcp.Server, h *Handlers) {
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "send_message",
			Description: "Send a message to a running pagent agent. Use this to provide guidance or corrections while an agent is processing.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Send Message",
				ReadOnlyHint:    false,
				DestructiveHint: boolPtr(false),
				IdempotentHint:  false,
				OpenWorldHint:   boolPtr(true),
			},
		},
		func(ctx context.Context, req *mcp.CallToolRequest, input SendMessageInput) (*mcp.CallToolResult, SendMessageOutput, error) {
			return nil, h.SendMessage(ctx, input), nil
		},
	)
}

func registerStopAgentsTool(server *mcp.Server, h *Handlers) {
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "stop_agents",
			Description: "Stop running pagent agents. Specify agent_name to stop a specific agent, or leave empty to stop all.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Stop Agents",
				ReadOnlyHint:    false,
				DestructiveHint: boolPtr(true),
				IdempotentHint:  true,
				OpenWorldHint:   boolPtr(true),
			},
		},
		func(ctx context.Context, req *mcp.CallToolRequest, input StopAgentsInput) (*mcp.CallToolResult, StopAgentsOutput, error) {
			return nil, h.StopAgents(ctx, input), nil
		},
	)
}
