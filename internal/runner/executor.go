// Package runner provides the execution logic for running agents.
package runner

import (
	"context"
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

// Logger provides logging methods for the executor
type Logger interface {
	Info(format string, args ...interface{})
	Verbose(format string, args ...interface{})
	Error(format string, args ...interface{})
}

// Execute runs agents with the given options.
// This is the shared execution path for both CLI and TUI.
func Execute(ctx context.Context, opts config.RunOptions, logger Logger) error {
	// Discover input files
	inp, err := input.Discover(opts.InputPath)
	if err != nil {
		return fmt.Errorf("input error: %w", err)
	}

	// Load config
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		logger.Verbose("Using default config: %v", err)
		cfg = config.Default()
	}

	// Apply options to config
	if err := applyOptions(cfg, opts); err != nil {
		return err
	}

	// Ensure output directory exists
	if err = os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Determine which agents to run
	selectedAgents := opts.Agents
	if len(selectedAgents) == 0 {
		selectedAgents = cfg.GetAgentNames()
	}

	// Validate agent names
	for _, name := range selectedAgents {
		if _, ok := cfg.Agents[name]; !ok {
			return fmt.Errorf("unknown agent: %s (available: %s)", name, strings.Join(cfg.GetAgentNames(), ", "))
		}
	}

	// Log startup info
	logStartup(logger, inp, cfg, selectedAgents, opts.Sequential)

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		logger.Info("\nReceived interrupt, shutting down agents...")
		cancel()
	}()

	// Create agent manager with input files
	var manager *agent.Manager
	if inp.IsDirectory {
		manager = agent.NewManagerWithInputs(cfg, inp.PrimaryFile, inp.Files, inp.Path, opts.IsVerbose())
	} else {
		manager = agent.NewManager(cfg, inp.PrimaryFile, opts.IsVerbose())
	}

	// Run agents
	var results []agent.Result
	if opts.Sequential {
		results, err = runSequential(ctx, manager, selectedAgents, logger)
	} else {
		results, err = runParallel(ctx, manager, selectedAgents, logger)
	}

	// Print summary
	printSummary(results, logger)

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
		if err := runPostProcessing(cfg, opts.IsVerbose(), logger); err != nil {
			return err
		}
	}

	return nil
}

// applyOptions applies RunOptions to the config
func applyOptions(cfg *config.Config, opts config.RunOptions) error {
	// Override output directory if specified
	if opts.OutputDir != "" {
		cfg.OutputDir = opts.OutputDir
	}

	cfg.Timeout = opts.Timeout

	// Handle resume mode
	switch opts.ResumeMode {
	case config.ResumeModeResume:
		cfg.ResumeMode = true
		cfg.ForceMode = false
	case config.ResumeModeForce:
		cfg.ResumeMode = false
		cfg.ForceMode = true
	default:
		cfg.ResumeMode = false
		cfg.ForceMode = false
	}

	// Override persona if specified
	if opts.Persona != "" {
		if !config.IsValidPersona(opts.Persona) {
			return fmt.Errorf("invalid persona %q: must be one of %v", opts.Persona, config.ValidPersonas)
		}
		cfg.Persona = opts.Persona
	}

	// Override architecture preference
	switch opts.Architecture {
	case config.ArchitectureStateless:
		cfg.Preferences.Stateless = true
	case config.ArchitectureDatabase:
		cfg.Preferences.Stateless = false
	// "config" means use whatever is in config
	}

	return nil
}

func logStartup(logger Logger, inp *input.Input, cfg *config.Config, agents []string, sequential bool) {
	logger.Info("Starting Pagent")
	logger.Info("%s", inp.Summary())

	if inp.IsDirectory {
		logger.Verbose("Input files:")
		for _, f := range inp.RelativePaths() {
			logger.Verbose("  - %s", f)
		}
	}

	// Display mode information
	executionMode := "create"
	if cfg.IsModifyMode() {
		executionMode = "modify"
		logger.Info("Mode: %s (targeting: %s)", executionMode, cfg.TargetCodebase)
		logger.Info("Specs output: %s", cfg.GetEffectiveSpecsOutputDir())
		logger.Info("Code output: %s", cfg.GetEffectiveCodeOutputDir())
	} else {
		logger.Info("Mode: %s", executionMode)
		logger.Info("Output: %s", cfg.OutputDir)
	}

	logger.Info("Agents: %s", strings.Join(agents, ", "))
	logger.Info("Persona: %s", cfg.Persona)
	logger.Info("Architecture: %s", map[bool]string{true: "stateless", false: "database-backed"}[cfg.Preferences.Stateless])
	logger.Info("Execution: %s", map[bool]string{true: "sequential", false: "parallel"}[sequential])
	logger.Info("")
}

func runParallel(ctx context.Context, manager *agent.Manager, agents []string, logger Logger) ([]agent.Result, error) {
	levels := manager.GetDependencyLevels(agents)
	var allResults []agent.Result

	for levelIdx, level := range levels {
		if len(level) == 0 {
			continue
		}

		logger.Verbose("Running level %d: %s", levelIdx+1, strings.Join(level, ", "))

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
			printAgentStatus(result, logger)
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
			logger.Error("Level %d had failures, stopping execution", levelIdx+1)
			return allResults, fmt.Errorf("agents in level %d failed", levelIdx+1)
		}
	}

	return allResults, nil
}

func runSequential(ctx context.Context, manager *agent.Manager, agents []string, logger Logger) ([]agent.Result, error) {
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
		printAgentStatus(result, logger)

		if result.Error != nil {
			logger.Error("Agent %s failed, stopping sequential execution", name)
			return results, result.Error
		}
	}

	return results, nil
}

func printAgentStatus(result agent.Result, logger Logger) {
	if result.Error != nil {
		logger.Info("✗ %s: failed (%v)", result.Agent, result.Error)
	} else {
		logger.Info("✓ %s: completed → %s", result.Agent, result.OutputPath)
	}
}

func printSummary(results []agent.Result, logger Logger) {
	logger.Info("")
	logger.Info("=== Summary ===")

	succeeded := 0
	failed := 0

	for _, r := range results {
		if r.Error != nil {
			failed++
		} else {
			succeeded++
		}
	}

	logger.Info("%d/%d agents succeeded", succeeded, len(results))

	if failed > 0 {
		logger.Info("Partial results saved.")
	}
}

func hasPostProcessing(cfg *config.Config) bool {
	pp := cfg.PostProcessing
	return pp.GenerateDiffSummary || pp.GeneratePRDescription || len(pp.ValidationCommands) > 0
}

func runPostProcessing(cfg *config.Config, verbose bool, logger Logger) error {
	logger.Info("")
	logger.Info("=== Post-Processing ===")

	pp := postprocess.NewRunner(cfg, verbose)
	ppResults := pp.Run()

	for _, r := range ppResults {
		if r.Success {
			logger.Info("✓ %s: %s", r.Step, r.Output)
		} else {
			logger.Error("✗ %s: %v", r.Step, r.Error)
		}
	}

	// Check for post-processing failures
	for _, r := range ppResults {
		if !r.Success {
			return fmt.Errorf("post-processing failed: %s", r.Step)
		}
	}

	return nil
}
