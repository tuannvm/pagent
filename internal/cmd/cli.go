// Package cmd provides the CLI implementation using stdlib flag.
package cmd

import (
	"flag"
	"fmt"
	"os"
)

var (
	verbose bool
	quiet   bool
	version = "dev"
)

// SetVersion sets the version string
func SetVersion(v string) {
	version = v
}

// Execute runs the CLI
func Execute() error {
	if len(os.Args) < 2 {
		printUsage()
		return nil
	}

	cmd := os.Args[1]

	// Handle flags that come before command (e.g., pagent -v run)
	// For simplicity, we expect: pagent <command> [flags] [args]

	switch cmd {
	case "run":
		return runMain(os.Args[2:])
	case "ui":
		return uiMain(os.Args[2:])
	case "init":
		return initMain(os.Args[2:])
	case "status":
		return statusMain(os.Args[2:])
	case "logs":
		return logsMain(os.Args[2:])
	case "message":
		return messageMain(os.Args[2:])
	case "stop":
		return stopMain(os.Args[2:])
	case "agents":
		return agentsMain(os.Args[2:])
	case "version", "-v", "--version":
		fmt.Printf("pagent version %s\n", version)
		return nil
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		printUsage()
		return fmt.Errorf("unknown command: %s", cmd)
	}
}

func printUsage() {
	fmt.Print(`Pagent - Orchestrate specialist agents from PRD

Usage:
  pagent <command> [flags] [args]

Commands:
  run <input>       Run specialist agents on input files
  ui [input]        Interactive dashboard for running agents
  init              Initialize pagent configuration
  status            Check status of running agents
  logs <agent>      View agent conversation history
  message <agent>   Send a message to an agent
  stop [agent]      Stop running agents
  agents            Manage agent definitions
  version           Print version information
  help              Show this help

Examples:
  pagent run ./prd.md
  pagent ui
  pagent ui ./prd.md
  pagent run ./prd.md -a architect,qa -s
  pagent init
  pagent status

Run 'pagent <command> -h' for command-specific help.
`)
}

// Helper functions for logging
func logInfo(format string, args ...interface{}) {
	if !quiet {
		_, _ = fmt.Fprintf(os.Stdout, format+"\n", args...)
	}
}

func logVerbose(format string, args ...interface{}) {
	if verbose && !quiet {
		_, _ = fmt.Fprintf(os.Stdout, "[DEBUG] "+format+"\n", args...)
	}
}

// parseGlobalFlags extracts -v and -q from args, returns remaining args
func parseGlobalFlags(fs *flag.FlagSet) {
	fs.BoolVar(&verbose, "v", false, "verbose output")
	fs.BoolVar(&verbose, "verbose", false, "verbose output")
	fs.BoolVar(&quiet, "q", false, "quiet output (errors only)")
	fs.BoolVar(&quiet, "quiet", false, "quiet output (errors only)")
}
