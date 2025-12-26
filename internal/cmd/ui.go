package cmd

import (
	"context"
	"flag"
	"fmt"

	"github.com/tuannvm/pagent/internal/config"
	"github.com/tuannvm/pagent/internal/runner"
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

	// Run the dashboard - returns *config.RunOptions directly
	opts, err := tui.RunDashboard(tui.DashboardOptions{
		PrefilledInput: prefilledInput,
		Config:         cfg,
		Accessible:     accessible,
	})
	if err != nil {
		return err
	}

	// nil opts means user cancelled
	if opts == nil {
		logInfo("Cancelled")
		return nil
	}

	// Validate input
	if opts.InputPath == "" {
		return fmt.Errorf("no input file specified")
	}

	// Validate agent selection
	if len(opts.Agents) == 0 {
		return fmt.Errorf("no agents selected")
	}

	// Display what we're about to run
	logInfo("Running pagent with %d agents...", len(opts.Agents))
	logInfo("")

	// Execute directly using the shared runner - NO TRANSLATION LAYER!
	logger := runner.NewStdLogger(opts.IsVerbose(), opts.IsQuiet())
	return runner.Execute(context.Background(), *opts, logger)
}
