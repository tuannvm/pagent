package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/tuannvm/pm-agent-workflow/internal/agent"
	"github.com/tuannvm/pm-agent-workflow/internal/config"
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
)

var runCmd = &cobra.Command{
	Use:   "run <prd-file>",
	Short: "Run specialist agents on a PRD",
	Long: `Run specialist agents to transform a PRD into deliverables.

By default, all agents run in parallel. Use --sequential to run
agents in dependency order.

Examples:
  pm-agents run ./prd.md
  pm-agents run ./prd.md --agents design,tech
  pm-agents run ./prd.md --sequential
  pm-agents run ./prd.md --output ./docs/`,
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
}

func runCommand(cmd *cobra.Command, args []string) error {
	prdPath := args[0]

	// Validate PRD file exists
	absPath, err := filepath.Abs(prdPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("PRD file not found: %s", absPath)
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
	logInfo("PRD: %s", absPath)
	logInfo("Output: %s", cfg.OutputDir)
	logInfo("Agents: %s", strings.Join(selectedAgents, ", "))
	logInfo("Persona: %s", cfg.Persona)
	logInfo("Mode: %s", map[bool]string{true: "sequential", false: "parallel"}[sequential])
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

	// Create agent manager
	manager := agent.NewManager(cfg, absPath, verbose)

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

	return nil
}

func runParallel(ctx context.Context, manager *agent.Manager, agents []string) ([]agent.Result, error) {
	var wg sync.WaitGroup
	results := make([]agent.Result, len(agents))
	resultCh := make(chan agent.Result, len(agents))

	for _, name := range agents {
		wg.Add(1)
		go func(agentName string) {
			defer wg.Done()
			result := manager.RunAgent(ctx, agentName)
			resultCh <- result
		}(name)
	}

	// Wait for all agents in a separate goroutine
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Collect results
	i := 0
	for result := range resultCh {
		results[i] = result
		i++
		printAgentStatus(result)
	}

	return results, nil
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
