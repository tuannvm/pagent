package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/tuannvm/pagent/internal/agent"
	"github.com/tuannvm/pagent/internal/api"
)

func logsMain(args []string) error {
	fs := flag.NewFlagSet("logs", flag.ContinueOnError)
	var followLogs bool
	fs.BoolVar(&followLogs, "f", false, "follow log output (not implemented)")
	fs.BoolVar(&followLogs, "follow", false, "follow log output (not implemented)")
	parseGlobalFlags(fs)

	fs.Usage = func() {
		fmt.Print(`Usage: pagent logs <agent> [flags]

View the conversation history for a specific agent.

Arguments:
  <agent>    Name of the agent

Flags:
  -f, -follow    Follow log output (not implemented)

Examples:
  pagent logs design
  pagent logs tech
`)
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		fs.Usage()
		return fmt.Errorf("missing required argument: agent name")
	}

	agentName := fs.Arg(0)

	// Read state file
	state, err := agent.LoadState()
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no agents running - start with 'pagent run'")
		}
		return fmt.Errorf("failed to read state: %w", err)
	}

	port, ok := state[agentName]
	if !ok {
		return fmt.Errorf("agent '%s' not found in running agents", agentName)
	}

	client := api.NewClient(port)

	// Get messages
	messages, err := client.GetMessages()
	if err != nil {
		return fmt.Errorf("failed to get messages: %w", err)
	}

	if len(messages) == 0 {
		logInfo("No messages yet for agent %s", agentName)
		return nil
	}

	// Print messages
	for _, msg := range messages {
		rolePrefix := "Agent"
		if msg.Role == "user" {
			rolePrefix = "User"
		}

		fmt.Printf("[%s]\n", rolePrefix)
		fmt.Printf("%s\n\n", msg.Content)
	}

	if followLogs {
		logInfo("Note: -follow is not yet implemented. Use status to check agent state.")
	}

	return nil
}
