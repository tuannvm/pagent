// Package main provides the CLI entry point for the pagent MCP server.
//
// Supports multiple transport modes:
//   - stdio (default): Standard input/output for CLI integration
//   - http: Streamable HTTP transport for web integration
//   - http+oauth: HTTP with OAuth 2.1 authentication
//
// Usage:
//
//	pagent-mcp                           # stdio mode (default)
//	pagent-mcp --transport http --port 8080
//	pagent-mcp --transport http --port 8080 --oauth --issuer https://company.okta.com --audience api://pagent
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	pagentmcp "github.com/tuannvm/pagent/internal/mcp"
)

// Version is the server version, set by the build process.
var Version = "dev"

func main() {
	log.Println("Starting Pagent MCP Server...")

	// CLI flags
	transport := flag.String("transport", getEnv("MCP_TRANSPORT", "stdio"), "Transport mode: stdio, http")
	port := flag.Int("port", getEnvInt("MCP_PORT", 8080), "HTTP port (only used with --transport http)")
	enableOAuth := flag.Bool("oauth", false, "Enable OAuth 2.1 authentication (only with http transport)")
	oauthProvider := flag.String("provider", "okta", "OAuth provider: okta, google, azure, hmac")
	oauthIssuer := flag.String("issuer", "", "OAuth issuer URL (required with --oauth)")
	oauthAudience := flag.String("audience", "", "OAuth audience (required with --oauth)")
	sessionTimeout := flag.Duration("session-timeout", 30*time.Minute, "HTTP session timeout")
	configPath := flag.String("config", "", "Path to pagent config file")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	flag.Parse()

	// Create handlers with configuration
	handlers := pagentmcp.NewHandlers()
	if *configPath != "" {
		handlers.WithConfigPath(*configPath)
	}
	if *verbose {
		handlers.WithVerbose(true)
	}

	// Build server config
	cfg := &pagentmcp.ServerConfig{
		Version:        Version,
		Handlers:       handlers,
		Port:           *port,
		SessionTimeout: *sessionTimeout,
	}

	if *enableOAuth {
		if *oauthIssuer == "" || *oauthAudience == "" {
			log.Fatal("--issuer and --audience are required with --oauth")
		}
		cfg.OAuth = &pagentmcp.OAuthConfig{
			Provider: *oauthProvider,
			Issuer:   *oauthIssuer,
			Audience: *oauthAudience,
		}
	}

	// Create MCP server
	server := pagentmcp.NewServer(cfg)

	// Run based on transport mode
	log.Printf("Starting MCP server with %s transport...", *transport)
	var err error
	switch *transport {
	case "stdio":
		err = server.ServeStdio()
	case "http":
		if cfg.OAuth != nil {
			err = server.ServeHTTPWithOAuth()
		} else {
			err = server.ServeHTTP()
		}
	default:
		log.Fatalf("Unknown transport: %s (use: stdio, http)", *transport)
	}

	if err != nil {
		log.Fatal(err)
	}

	log.Println("Server shutdown complete")
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok {
		var i int
		if _, err := fmt.Sscanf(v, "%d", &i); err == nil {
			return i
		}
	}
	return def
}
