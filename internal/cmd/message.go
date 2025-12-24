package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/tuannvm/pm-agent-workflow/internal/agent"
	"github.com/tuannvm/pm-agent-workflow/internal/api"
)

var messageCmd = &cobra.Command{
	Use:   "message <agent> <message>",
	Short: "Send a message to an agent",
	Long: `Send a message to a specific agent when it's in stable (idle) state.

The command will wait for the agent to become stable before sending.
Use this to provide guidance or additional instructions.

Example:
  pagent message design "Focus more on mobile UX"
  pagent message tech "Use REST, not GraphQL"`,
	Args: cobra.ExactArgs(2),
	RunE: messageCommand,
}

func init() {
	rootCmd.AddCommand(messageCmd)
}

func messageCommand(cmd *cobra.Command, args []string) error {
	agentName := args[0]
	message := args[1]

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
