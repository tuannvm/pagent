package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/tuannvm/pm-agent-workflow/internal/agent"
	"github.com/tuannvm/pm-agent-workflow/internal/config"
	"github.com/tuannvm/pm-agent-workflow/internal/input"
	"github.com/tuannvm/pm-agent-workflow/internal/postprocess"
)

var (
	agentsFlag     string
	outputDir      string
	sequential     bool
	configPath     string
	timeoutSeconds int
	resumeMode     bool
	forceMode      bool
	personaFlag    string
	statelessFlag  *bool // nil = use config default, non-nil = explicit override
)

var runCmd = &cobra.Command{
	Use:   "run <input>",
	Short: "Run specialist agents on input files",
	Long: `Run specialist agents to transform input documents into deliverables.

Input can be a single file or a directory containing multiple files.
Supported file types: .md, .yaml, .yml, .json, .txt

By default, all agents run in parallel. Use --sequential to run
agents in dependency order.

Examples:
  pm-agents run ./prd.md                    # Single PRD file
  pm-agents run ./input/                    # Directory with multiple inputs
  pm-agents run ./specs/ --agents architect # Only run architect
  pm-agents run ./prd.md --persona minimal  # MVP implementation`,
	Args: cobra.ExactArgs(1),
	RunE: runCommand,
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringVarP(&agentsFlag, "agents", "a", "", "comma-separated list of agents (default: all)")
	runCmd.Flags().StringVarP(&outputDir, "output", "o", "./outputs", "output directory")
	runCmd.Flags().BoolVarP(&sequential, "sequential", "s", false, "run agents in dependency order")
	runCmd.Flags().StringVarP(&configPath, "config", "c", "", "config file path")
	runCmd.Flags().IntVarP(&timeoutSeconds, "timeout", "t", 0, "timeout per agent in seconds (0=infinite, polls until completion)")
	runCmd.Flags().BoolVarP(&resumeMode, "resume", "r", false, "skip agents whose outputs are up-to-date with PRD")
	runCmd.Flags().BoolVarP(&forceMode, "force", "f", false, "force regeneration, ignore existing outputs")
	runCmd.Flags().StringVarP(&personaFlag, "persona", "p", "", "implementation style: minimal, balanced, production (default: balanced)")

	// Stateless flag - use pointer to detect if flag was explicitly set
	statelessFlag = runCmd.Flags().Bool("stateless", false, "prefer stateless architecture (event-driven, no traditional DB)")
	runCmd.Flags().Bool("no-stateless", false, "prefer traditional database-backed architecture")
}

func runCommand(cmd *cobra.Command, args []string) error {
	inputPath := args[0]

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
	if outputDir != "" {
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
	if cmd.Flags().Changed("stateless") {
		cfg.Preferences.Stateless = true
	} else if cmd.Flags().Changed("no-stateless") {
		cfg.Preferences.Stateless = false
	}

	// Ensure output directory exists
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
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

	logInfo("Starting PM Agent Workflow")
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
// Agents within the same dependency level run concurrently.
// All agents in a level must complete before the next level starts.
func runParallel(ctx context.Context, manager *agent.Manager, agents []string) ([]agent.Result, error) {
	// Group agents by dependency level
	levels := manager.GetDependencyLevels(agents)
	var allResults []agent.Result

	for levelIdx, level := range levels {
		if len(level) == 0 {
			continue
		}

		logVerbose("Running level %d: %s", levelIdx+1, strings.Join(level, ", "))

		var wg sync.WaitGroup
		resultCh := make(chan agent.Result, len(level))

		// Run all agents in this level concurrently
		for _, name := range level {
			wg.Add(1)
			go func(agentName string) {
				defer wg.Done()
				result := manager.RunAgent(ctx, agentName)
				resultCh <- result
			}(name)
		}

		// Wait for all agents in this level
		go func() {
			wg.Wait()
			close(resultCh)
		}()

		// Collect results for this level
		levelFailed := false
		for result := range resultCh {
			allResults = append(allResults, result)
			printAgentStatus(result)
			if result.Error != nil {
				levelFailed = true
			}
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return allResults, ctx.Err()
		default:
		}

		// Stop if any agent in this level failed (dependencies for next level won't be satisfied)
		if levelFailed {
			logError("Level %d had failures, stopping execution", levelIdx+1)
			return allResults, fmt.Errorf("agents in level %d failed", levelIdx+1)
		}
	}

	return allResults, nil
}

func runSequential(ctx context.Context, manager *agent.Manager, agents []string) ([]agent.Result, error) {
	// Topological sort based on dependencies
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
