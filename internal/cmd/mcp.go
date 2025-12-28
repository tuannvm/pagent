package cmd

import (
	"flag"
	"fmt"
	"log"
	"time"

	pagentmcp "github.com/tuannvm/pagent/internal/mcp"
)

func mcpMain(args []string) error {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)

	var (
		transport      string
		port           int
		enableOAuth    bool
		oauthProvider  string
		oauthIssuer    string
		oauthAudience  string
		sessionTimeout time.Duration
		configPath     string
		mcpVerbose     bool
	)

	fs.StringVar(&transport, "transport", "stdio", "transport mode: stdio, http")
	fs.IntVar(&port, "port", 8080, "HTTP port (only used with --transport http)")
	fs.BoolVar(&enableOAuth, "oauth", false, "enable OAuth 2.1 authentication (only with http transport)")
	fs.StringVar(&oauthProvider, "provider", "okta", "OAuth provider: okta, google, azure, hmac")
	fs.StringVar(&oauthIssuer, "issuer", "", "OAuth issuer URL (required with --oauth)")
	fs.StringVar(&oauthAudience, "audience", "", "OAuth audience (required with --oauth)")
	fs.DurationVar(&sessionTimeout, "session-timeout", 30*time.Minute, "HTTP session timeout")
	fs.StringVar(&configPath, "config", "", "path to pagent config file")
	fs.BoolVar(&mcpVerbose, "v", false, "enable verbose logging")
	fs.BoolVar(&mcpVerbose, "verbose", false, "enable verbose logging")

	fs.Usage = func() {
		fmt.Print(`Usage: pagent mcp [flags]

Run pagent as an MCP (Model Context Protocol) server.

Transports:
  stdio    Standard input/output for CLI integration (default)
  http     Streamable HTTP transport for web integration

Examples:
  pagent mcp                                    # stdio mode
  pagent mcp --transport http --port 8080       # HTTP mode
  pagent mcp --transport http --oauth \
    --issuer https://company.okta.com \
    --audience api://pagent                     # HTTP with OAuth

Flags:
`)
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	log.Println("Starting Pagent MCP Server...")

	// Create handlers with configuration
	handlers := pagentmcp.NewHandlers()
	if configPath != "" {
		handlers.WithConfigPath(configPath)
	}
	if mcpVerbose {
		handlers.WithVerbose(true)
	}

	// Build server config
	cfg := &pagentmcp.ServerConfig{
		Version:        version,
		Handlers:       handlers,
		Port:           port,
		SessionTimeout: sessionTimeout,
	}

	if enableOAuth {
		if oauthIssuer == "" || oauthAudience == "" {
			return fmt.Errorf("--issuer and --audience are required with --oauth")
		}
		cfg.OAuth = &pagentmcp.OAuthConfig{
			Provider: oauthProvider,
			Issuer:   oauthIssuer,
			Audience: oauthAudience,
		}
	}

	// Create MCP server
	server := pagentmcp.NewServer(cfg)

	// Run based on transport mode
	log.Printf("Starting MCP server with %s transport...", transport)
	var err error
	switch transport {
	case "stdio":
		err = server.ServeStdio()
	case "http":
		if cfg.OAuth != nil {
			err = server.ServeHTTPWithOAuth()
		} else {
			err = server.ServeHTTP()
		}
	default:
		return fmt.Errorf("unknown transport: %s (use: stdio, http)", transport)
	}

	if err != nil {
		return err
	}

	log.Println("Server shutdown complete")
	return nil
}
