package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/tuannvm/pagent/internal/agent"
	"github.com/tuannvm/pagent/internal/config"
	"github.com/tuannvm/pagent/internal/input"
	"github.com/tuannvm/pagent/internal/postprocess"
)

func runMain(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)

	// Define flags
	var (
		agentsFlag     string
		outputDir      string
		sequential     bool
		configPath     string
		timeoutSeconds int
		resumeMode     bool
		forceMode      bool
		personaFlag    string
		stateless      bool
		noStateless    bool
	)

	fs.StringVar(&agentsFlag, "a", "", "comma-separated list of agents (default: all)")
	fs.StringVar(&agentsFlag, "agents", "", "comma-separated list of agents (default: all)")
	fs.StringVar(&outputDir, "o", "./outputs", "output directory")
	fs.StringVar(&outputDir, "output", "./outputs", "output directory")
	fs.BoolVar(&sequential, "s", false, "run agents in dependency order")
	fs.BoolVar(&sequential, "sequential", false, "run agents in dependency order")
	fs.StringVar(&configPath, "c", "", "config file path")
	fs.StringVar(&configPath, "config", "", "config file path")
	fs.IntVar(&timeoutSeconds, "t", 0, "timeout per agent in seconds (0=infinite)")
	fs.IntVar(&timeoutSeconds, "timeout", 0, "timeout per agent in seconds (0=infinite)")
	fs.BoolVar(&resumeMode, "r", false, "skip agents whose outputs are up-to-date")
	fs.BoolVar(&resumeMode, "resume", false, "skip agents whose outputs are up-to-date")
	fs.BoolVar(&forceMode, "f", false, "force regeneration, ignore existing outputs")
	fs.BoolVar(&forceMode, "force", false, "force regeneration, ignore existing outputs")
	fs.StringVar(&personaFlag, "p", "", "implementation style: minimal, balanced, production")
	fs.StringVar(&personaFlag, "persona", "", "implementation style: minimal, balanced, production")
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

	inputPath := fs.Arg(0)

	// Discover input files (supports both single file and directory)
	inp, err := input.Discover(inputPath)
	if err != nil {
		return fmt.Errorf("input error: %w", err)
	}

	// Load config
	cfg, err := config.Load(configPath)
	if err != nil {
		logVerbose("Using default config: %v", err)
		cfg = config.Default()
	}

	// Override with flags
	if outputDir != "" && outputDir != "./outputs" {
		cfg.OutputDir = outputDir
	}
	cfg.Timeout = timeoutSeconds
	cfg.ResumeMode = resumeMode
	cfg.ForceMode = forceMode

	// Force mode overrides resume mode
	if cfg.ForceMode {
		cfg.ResumeMode = false
	}

	// Override persona from CLI flag
	if personaFlag != "" {
		if !config.IsValidPersona(personaFlag) {
			return fmt.Errorf("invalid persona %q: must be one of %v", personaFlag, config.ValidPersonas)
		}
		cfg.Persona = personaFlag
	}

	// Override stateless preference from CLI flags
	if stateless {
		cfg.Preferences.Stateless = true
	} else if noStateless {
		cfg.Preferences.Stateless = false
	}

	// Ensure output directory exists
	if err = os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Determine which agents to run
	selectedAgents := cfg.GetAgentNames()
	if agentsFlag != "" {
		selectedAgents = strings.Split(agentsFlag, ",")
		for i := range selectedAgents {
			selectedAgents[i] = strings.TrimSpace(selectedAgents[i])
		}
	}

	// Validate agent names
	for _, name := range selectedAgents {
		if _, ok := cfg.Agents[name]; !ok {
			return fmt.Errorf("unknown agent: %s (available: %s)", name, strings.Join(cfg.GetAgentNames(), ", "))
		}
	}

	logInfo("Starting Pagent")
	logInfo("%s", inp.Summary())
	if inp.IsDirectory {
		logVerbose("Input files:")
		for _, f := range inp.RelativePaths() {
			logVerbose("  - %s", f)
		}
	}

	// Display mode information
	executionMode := "create"
	if cfg.IsModifyMode() {
		executionMode = "modify"
		logInfo("Mode: %s (targeting: %s)", executionMode, cfg.TargetCodebase)
		logInfo("Specs output: %s", cfg.GetEffectiveSpecsOutputDir())
		logInfo("Code output: %s", cfg.GetEffectiveCodeOutputDir())
	} else {
		logInfo("Mode: %s", executionMode)
		logInfo("Output: %s", cfg.OutputDir)
	}

	logInfo("Agents: %s", strings.Join(selectedAgents, ", "))
	logInfo("Persona: %s", cfg.Persona)
	logInfo("Architecture: %s", map[bool]string{true: "stateless", false: "database-backed"}[cfg.Preferences.Stateless])
	logInfo("Execution: %s", map[bool]string{true: "sequential", false: "parallel"}[sequential])
	logInfo("")

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		logInfo("\nReceived interrupt, shutting down agents...")
		cancel()
	}()

	// Create agent manager with input files
	var manager *agent.Manager
	if inp.IsDirectory {
		manager = agent.NewManagerWithInputs(cfg, inp.PrimaryFile, inp.Files, inp.Path, verbose)
	} else {
		manager = agent.NewManager(cfg, inp.PrimaryFile, verbose)
	}

	// Run agents
	var results []agent.Result
	if sequential {
		results, err = runSequential(ctx, manager, selectedAgents)
	} else {
		results, err = runParallel(ctx, manager, selectedAgents)
	}

	// Print summary
	printSummary(results)

	if err != nil {
		return err
	}

	// Check for any failures
	for _, r := range results {
		if r.Error != nil {
			return fmt.Errorf("some agents failed")
		}
	}

	// Run post-processing (only in modify mode)
	if cfg.IsModifyMode() && hasPostProcessing(cfg) {
		logInfo("")
		logInfo("=== Post-Processing ===")

		pp := postprocess.NewRunner(cfg, verbose)
		ppResults := pp.Run()

		for _, r := range ppResults {
			if r.Success {
				logInfo("✓ %s: %s", r.Step, r.Output)
			} else {
				logError("✗ %s: %v", r.Step, r.Error)
			}
		}

		// Check for post-processing failures
		for _, r := range ppResults {
			if !r.Success {
				return fmt.Errorf("post-processing failed: %s", r.Step)
			}
		}
	}

	return nil
}

// hasPostProcessing returns true if any post-processing is configured
func hasPostProcessing(cfg *config.Config) bool {
	pp := cfg.PostProcessing
	return pp.GenerateDiffSummary || pp.GeneratePRDescription || len(pp.ValidationCommands) > 0
}

// runParallel runs agents in parallel, but respects dependency levels.
func runParallel(ctx context.Context, manager *agent.Manager, agents []string) ([]agent.Result, error) {
	levels := manager.GetDependencyLevels(agents)
	var allResults []agent.Result

	for levelIdx, level := range levels {
		if len(level) == 0 {
			continue
		}

		logVerbose("Running level %d: %s", levelIdx+1, strings.Join(level, ", "))

		var wg sync.WaitGroup
		resultCh := make(chan agent.Result, len(level))

		for _, name := range level {
			wg.Add(1)
			go func(agentName string) {
				defer wg.Done()
				result := manager.RunAgent(ctx, agentName)
				resultCh <- result
			}(name)
		}

		go func() {
			wg.Wait()
			close(resultCh)
		}()

		levelFailed := false
		for result := range resultCh {
			allResults = append(allResults, result)
			printAgentStatus(result)
			if result.Error != nil {
				levelFailed = true
			}
		}

		select {
		case <-ctx.Done():
			return allResults, ctx.Err()
		default:
		}

		if levelFailed {
			logError("Level %d had failures, stopping execution", levelIdx+1)
			return allResults, fmt.Errorf("agents in level %d failed", levelIdx+1)
		}
	}

	return allResults, nil
}

func runSequential(ctx context.Context, manager *agent.Manager, agents []string) ([]agent.Result, error) {
	sorted := manager.TopologicalSort(agents)
	results := make([]agent.Result, 0, len(sorted))

	for _, name := range sorted {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		result := manager.RunAgent(ctx, name)
		results = append(results, result)
		printAgentStatus(result)

		if result.Error != nil {
			logError("Agent %s failed, stopping sequential execution", name)
			return results, result.Error
		}
	}

	return results, nil
}

func printAgentStatus(result agent.Result) {
	if result.Error != nil {
		logInfo("✗ %s: failed (%v)", result.Agent, result.Error)
	} else {
		logInfo("✓ %s: completed → %s", result.Agent, result.OutputPath)
	}
}

func printSummary(results []agent.Result) {
	logInfo("")
	logInfo("=== Summary ===")

	succeeded := 0
	failed := 0

	for _, r := range results {
		if r.Error != nil {
			failed++
		} else {
			succeeded++
		}
	}

	logInfo("%d/%d agents succeeded", succeeded, len(results))

	if failed > 0 {
		logInfo("Partial results saved.")
	}
}
