package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/tuannvm/pagent/internal/agent"
	"github.com/tuannvm/pagent/internal/api"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check status of running agents",
	Long: `Check the status of all running agents.

Shows each agent's current state (running/stable/not running)
and port number.

Example:
  pagent status`,
	RunE: statusCommand,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func statusCommand(cmd *cobra.Command, args []string) error {
	// Read state file to find running agents
	state, err := agent.LoadState()
	if err != nil {
		if os.IsNotExist(err) {
			logInfo("No agents currently running")
			return nil
		}
		return fmt.Errorf("failed to read state: %w", err)
	}

	if len(state) == 0 {
		logInfo("No agents currently running")
		return nil
	}

	// Check status of each agent
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "AGENT\tPORT\tSTATUS")

	for name, port := range state {
		client := api.NewClient(port)
		status, err := client.GetStatus()

		statusStr := "not responding"
		if err == nil {
			statusStr = status.Status
		}

		_, _ = fmt.Fprintf(w, "%s\t%d\t%s\n", name, port, statusStr)
	}

	_ = w.Flush()
	return nil
}

