package cmd

import (
	"flag"
	"fmt"
	"strings"

	"github.com/tuannvm/pagent/internal/config"
	"github.com/tuannvm/pagent/internal/tui"
)

func uiMain(args []string) error {
	fs := flag.NewFlagSet("ui", flag.ContinueOnError)

	var accessible bool
	fs.BoolVar(&accessible, "accessible", false, "enable accessible mode for screen readers")

	fs.Usage = func() {
		fmt.Print(`Usage: pagent ui [input] [flags]

Launch interactive dashboard for running agents.

All pagent run options are available through the UI - no flags to memorize.
Smart defaults are pre-filled from your .pagent/config.yaml.

Arguments:
  [input]    Optional: pre-fill input file path

Flags:
  --accessible    Enable accessible mode for screen readers

Examples:
  pagent ui
  pagent ui ./prd.md
  pagent ui --accessible
`)
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Load config
	cfg, err := config.Load("")
	if err != nil {
		cfg = config.Default()
	}

	// Pre-fill input if provided as argument
	prefilledInput := ""
	if fs.NArg() > 0 {
		prefilledInput = fs.Arg(0)
	}

	// Run the dashboard
	result, err := tui.RunDashboard(tui.DashboardOptions{
		PrefilledInput: prefilledInput,
		Config:         cfg,
		Accessible:     accessible,
	})
	if err != nil {
		return err
	}

	if result.Cancelled {
		logInfo("Cancelled")
		return nil
	}

	// Validate input
	if result.InputPath == "" {
		return fmt.Errorf("no input file specified")
	}

	// Build args for runMain
	runArgs := []string{result.InputPath}

	if !result.AllAgents && len(result.Agents) > 0 {
		runArgs = append(runArgs, "-a", strings.Join(result.Agents, ","))
	}

	if result.Persona != "" && result.Persona != "balanced" {
		runArgs = append(runArgs, "-p", result.Persona)
	}

	if result.OutputDir != "" && result.OutputDir != "./outputs" {
		runArgs = append(runArgs, "-o", result.OutputDir)
	}

	if result.Sequential {
		runArgs = append(runArgs, "-s")
	}

	switch result.ResumeMode {
	case "resume":
		runArgs = append(runArgs, "-r")
	case "force":
		runArgs = append(runArgs, "-f")
	}

	switch result.Architecture {
	case "stateless":
		runArgs = append(runArgs, "--stateless")
	case "database":
		runArgs = append(runArgs, "--no-stateless")
	}

	if result.Timeout > 0 {
		runArgs = append(runArgs, "-t", fmt.Sprintf("%d", result.Timeout))
	}

	if result.ConfigPath != "" {
		runArgs = append(runArgs, "-c", result.ConfigPath)
	}

	switch result.Verbosity {
	case "verbose":
		runArgs = append(runArgs, "-v")
	case "quiet":
		runArgs = append(runArgs, "-q")
	}

	// Display what we're about to run
	logInfo("Running: pagent run %s", strings.Join(runArgs, " "))
	logInfo("")

	// Execute run command
	return runMain(runArgs)
}
