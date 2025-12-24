package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tuannvm/pm-agent-workflow/internal/agent"
	"github.com/tuannvm/pm-agent-workflow/internal/api"
)

var (
	followLogs bool
)

var logsCmd = &cobra.Command{
	Use:   "logs <agent>",
	Short: "View agent conversation history",
	Long: `View the conversation history for a specific agent.

Shows all messages exchanged between the user and the agent.

Example:
  pagent logs design
  pagent logs tech --follow`,
	Args: cobra.ExactArgs(1),
	RunE: logsCommand,
}

func init() {
	rootCmd.AddCommand(logsCmd)
	logsCmd.Flags().BoolVarP(&followLogs, "follow", "f", false, "follow log output (not implemented)")
}

func logsCommand(cmd *cobra.Command, args []string) error {
	agentName := args[0]

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
		rolePrefix := "🤖"
		if msg.Role == "user" {
			rolePrefix = "👤"
		}

		fmt.Printf("%s [%s]\n", rolePrefix, msg.Role)
		fmt.Printf("%s\n\n", msg.Content)
	}

	if followLogs {
		logInfo("Note: --follow is not yet implemented. Use status to check agent state.")
	}

	return nil
}
