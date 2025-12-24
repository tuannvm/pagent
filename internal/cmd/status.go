package cmd

import (
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/tuannvm/pagent/internal/agent"
	"github.com/tuannvm/pagent/internal/api"
)

func statusMain(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	parseGlobalFlags(fs)

	fs.Usage = func() {
		fmt.Print(`Usage: pagent status

Check the status of all running agents.

Shows each agent's current state (running/stable/not running)
and port number.
`)
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

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
