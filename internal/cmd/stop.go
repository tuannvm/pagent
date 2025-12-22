package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/tuannvm/pm-agent-workflow/internal/agent"
)

var (
	stopAll bool
)

var stopCmd = &cobra.Command{
	Use:   "stop [agent]",
	Short: "Stop running agents",
	Long: `Stop one or all running agents.

Examples:
  pm-agents stop tech         # Stop specific agent
  pm-agents stop --all        # Stop all agents`,
	RunE: stopCommand,
}

func init() {
	rootCmd.AddCommand(stopCmd)
	stopCmd.Flags().BoolVarP(&stopAll, "all", "a", false, "stop all agents")
}

func stopCommand(cmd *cobra.Command, args []string) error {
	if !stopAll && len(args) == 0 {
		return fmt.Errorf("specify an agent name or use --all")
	}

	// Read state file
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

	if stopAll {
		for name, port := range state {
			stopAgentByPort(name, port)
		}
		agent.ClearState()
		logInfo("All agents stopped")
	} else {
		agentName := args[0]
		port, ok := state[agentName]
		if !ok {
			return fmt.Errorf("agent '%s' not found", agentName)
		}

		stopAgentByPort(agentName, port)
		logInfo("Agent %s stopped", agentName)
	}

	return nil
}

func stopAgentByPort(name string, port int) {
	// Find process by port using lsof (macOS/Linux)
	// This is a best-effort approach since we don't have direct process control
	logVerbose("Attempting to stop agent %s on port %d", name, port)

	// First find the PID
	out, err := exec.Command("lsof", "-ti", fmt.Sprintf(":%d", port)).Output()
	if err != nil {
		logVerbose("Could not find process for agent %s on port %d: %v", name, port, err)
		return
	}

	pid := string(out)
	if pid == "" {
		logVerbose("No process found on port %d", port)
		return
	}

	// Kill the process
	killCmd := exec.Command("kill", "-TERM", pid[:len(pid)-1]) // Remove trailing newline
	if err := killCmd.Run(); err != nil {
		logVerbose("Could not kill process %s: %v", pid, err)
	}
}
