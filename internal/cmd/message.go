package cmd

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/tuannvm/pagent/internal/agent"
	"github.com/tuannvm/pagent/internal/api"
)

func messageMain(args []string) error {
	fs := flag.NewFlagSet("message", flag.ContinueOnError)
	parseGlobalFlags(fs)

	fs.Usage = func() {
		fmt.Print(`Usage: pagent message <agent> <message>

Send a message to a specific agent when it's in stable (idle) state.

The command will wait for the agent to become stable before sending.
Use this to provide guidance or additional instructions.

Arguments:
  <agent>      Name of the agent
  <message>    Message to send (quote if contains spaces)

Examples:
  pagent message design "Focus more on mobile UX"
  pagent message tech "Use REST, not GraphQL"
`)
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() < 2 {
		fs.Usage()
		return fmt.Errorf("missing required arguments: agent name and message")
	}

	agentName := fs.Arg(0)
	message := strings.Join(fs.Args()[1:], " ")

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

	// Check current status
	status, err := client.GetStatus()
	if err != nil {
		return fmt.Errorf("failed to get agent status: %w", err)
	}

	if status.Status == "running" {
		logInfo("Agent is currently running. Waiting for stable state...")

		if err := client.WaitForStable(60 * time.Second); err != nil {
			return fmt.Errorf("timeout waiting for agent: %w", err)
		}
	}

	// Send message
	logInfo("Sending message to %s...", agentName)

	if err := client.SendMessage(message, "user"); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	logInfo("Message sent successfully")
	return nil
}
