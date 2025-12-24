package cmd

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/tuannvm/pagent/internal/agent"
)

func stopMain(args []string) error {
	fs := flag.NewFlagSet("stop", flag.ContinueOnError)
	var stopAll bool
	fs.BoolVar(&stopAll, "a", false, "stop all agents")
	fs.BoolVar(&stopAll, "all", false, "stop all agents")
	parseGlobalFlags(fs)

	fs.Usage = func() {
		fmt.Print(`Usage: pagent stop [agent] [flags]

Stop one or all running agents.

Arguments:
  [agent]    Name of the agent to stop (optional if using -all)

Flags:
  -a, -all    Stop all agents

Examples:
  pagent stop tech
  pagent stop -all
`)
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	if !stopAll && fs.NArg() == 0 {
		fs.Usage()
		return fmt.Errorf("specify an agent name or use -all")
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
		agentName := fs.Arg(0)
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
	logVerbose("Attempting to stop agent %s on port %d", name, port)

	out, err := exec.Command("lsof", "-ti", fmt.Sprintf(":%d", port)).Output()
	if err != nil {
		logVerbose("Could not find process for agent %s on port %d: %v", name, port, err)
		return
	}

	pidStr := strings.TrimSpace(string(out))
	if pidStr == "" {
		logVerbose("No process found on port %d", port)
		return
	}

	pids := strings.Split(pidStr, "\n")
	for _, pid := range pids {
		pid = strings.TrimSpace(pid)
		if pid == "" {
			continue
		}
		logVerbose("Killing process %s", pid)
		killCmd := exec.Command("kill", "-TERM", pid)
		if err := killCmd.Run(); err != nil {
			logVerbose("Could not kill process %s: %v", pid, err)
		}
	}
}
