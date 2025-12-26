package cmd

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/tuannvm/pagent/internal/config"
	"github.com/tuannvm/pagent/internal/runner"
)

func runMain(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)

	// Initialize with defaults
	opts := config.DefaultRunOptions(nil)

	// Define flags
	var (
		agentsFlag  string
		resumeMode  bool
		forceMode   bool
		stateless   bool
		noStateless bool
	)

	fs.StringVar(&agentsFlag, "a", "", "comma-separated list of agents (default: all)")
	fs.StringVar(&agentsFlag, "agents", "", "comma-separated list of agents (default: all)")
	fs.StringVar(&opts.OutputDir, "o", opts.OutputDir, "output directory")
	fs.StringVar(&opts.OutputDir, "output", opts.OutputDir, "output directory")
	fs.BoolVar(&opts.Sequential, "s", false, "run agents in dependency order")
	fs.BoolVar(&opts.Sequential, "sequential", false, "run agents in dependency order")
	fs.StringVar(&opts.ConfigPath, "c", "", "config file path")
	fs.StringVar(&opts.ConfigPath, "config", "", "config file path")
	fs.IntVar(&opts.Timeout, "t", 0, "timeout per agent in seconds (0=infinite)")
	fs.IntVar(&opts.Timeout, "timeout", 0, "timeout per agent in seconds (0=infinite)")
	fs.BoolVar(&resumeMode, "r", false, "skip agents whose outputs are up-to-date")
	fs.BoolVar(&resumeMode, "resume", false, "skip agents whose outputs are up-to-date")
	fs.BoolVar(&forceMode, "f", false, "force regeneration, ignore existing outputs")
	fs.BoolVar(&forceMode, "force", false, "force regeneration, ignore existing outputs")
	fs.StringVar(&opts.Persona, "p", "", "implementation style: minimal, balanced, production")
	fs.StringVar(&opts.Persona, "persona", "", "implementation style: minimal, balanced, production")
	fs.BoolVar(&stateless, "stateless", false, "prefer stateless architecture")
	fs.BoolVar(&noStateless, "no-stateless", false, "prefer traditional database-backed architecture")
	parseGlobalFlags(fs)

	fs.Usage = func() {
		fmt.Print(`Usage: pagent run <input> [flags]

Run specialist agents to transform input documents into deliverables.

Arguments:
  <input>    Input file or directory (.md, .yaml, .yml, .json, .txt)

Flags:
  -a, -agents string     Comma-separated list of agents (default: all)
  -o, -output string     Output directory (default: ./outputs)
  -s, -sequential        Run agents in dependency order
  -c, -config string     Config file path
  -t, -timeout int       Timeout per agent in seconds (0=infinite)
  -r, -resume            Skip agents whose outputs are up-to-date
  -f, -force             Force regeneration, ignore existing outputs
  -p, -persona string    Implementation style: minimal, balanced, production
  -stateless             Prefer stateless architecture
  -no-stateless          Prefer traditional database-backed architecture
  -v, -verbose           Verbose output
  -q, -quiet             Quiet output (errors only)

Examples:
  pagent run ./prd.md
  pagent run ./prd.md -a architect,qa -s
  pagent run ./prd.md -p minimal
  pagent run ./input/ -o ./docs/specs/
`)
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		fs.Usage()
		return fmt.Errorf("missing required argument: input file or directory")
	}

	// Set input path
	opts.InputPath = fs.Arg(0)

	// Parse agents
	if agentsFlag != "" {
		agents := strings.Split(agentsFlag, ",")
		for i := range agents {
			agents[i] = strings.TrimSpace(agents[i])
		}
		opts.Agents = agents
	}

	// Map boolean flags to options
	if forceMode {
		opts.ResumeMode = config.ResumeModeForce
	} else if resumeMode {
		opts.ResumeMode = config.ResumeModeResume
	}

	if stateless {
		opts.Architecture = config.ArchitectureStateless
	} else if noStateless {
		opts.Architecture = config.ArchitectureDatabase
	}

	// Map verbosity
	if verbose {
		opts.Verbosity = config.VerbosityVerbose
	} else if quiet {
		opts.Verbosity = config.VerbosityQuiet
	}

	// Execute using the shared runner
	logger := runner.NewStdLogger(verbose, quiet)
	return runner.Execute(context.Background(), opts, logger)
}
