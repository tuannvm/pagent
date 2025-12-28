package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/tuannvm/pagent/internal/api"
	"github.com/tuannvm/pagent/internal/config"
	"github.com/tuannvm/pagent/internal/prompt"
	"github.com/tuannvm/pagent/internal/state"
)

const (
	basePort      = 3284
	healthTimeout = 120 * time.Second // 2 min for Claude Code to fully initialize
)

// State file paths
var (
	StateFile = filepath.Join(os.TempDir(), "pagent-state.json")
)

// Result represents the result of running an agent
type Result struct {
	Agent      string
	OutputPath string
	Error      error
	Duration   time.Duration
}

// RunningAgent tracks a running agent
type RunningAgent struct {
	Name      string
	Port      int
	Client    *api.Client // HTTP client for status polling
	LibClient *LibClient  // Library client for agent management
	StartedAt time.Time
}

// Manager manages agent lifecycle
type Manager struct {
	config       *config.Config
	prdPath      string   // Primary input file (backward compatible)
	inputFiles   []string // All input files
	inputDir     string   // Input directory (empty if single file)
	verbose      bool
	agents       map[string]*RunningAgent
	portAlloc    int
	mu           sync.Mutex
	promptLoader *prompt.Loader
	stateManager *state.Manager // Tracks resume state for incremental execution
}

// NewManager creates a new agent manager
func NewManager(cfg *config.Config, prdPath string, verbose bool) *Manager {
	m := &Manager{
		config:       cfg,
		prdPath:      prdPath,
		inputFiles:   []string{prdPath}, // Single file as default
		verbose:      verbose,
		agents:       make(map[string]*RunningAgent),
		portAlloc:    basePort,
		promptLoader: prompt.NewLoader("prompts"), // Load from ./prompts if exists
		stateManager: state.NewManager(cfg.OutputDir),
	}
	m.initializeState()
	return m
}

// NewManagerWithInputs creates a manager with multiple input files
func NewManagerWithInputs(cfg *config.Config, primaryFile string, inputFiles []string, inputDir string, verbose bool) *Manager {
	m := &Manager{
		config:       cfg,
		prdPath:      primaryFile,
		inputFiles:   inputFiles,
		inputDir:     inputDir,
		verbose:      verbose,
		agents:       make(map[string]*RunningAgent),
		portAlloc:    basePort,
		promptLoader: prompt.NewLoader("prompts"),
		stateManager: state.NewManager(cfg.OutputDir),
	}
	m.initializeState()
	return m
}

// initializeState loads existing resume state and updates input/config hashes.
func (m *Manager) initializeState() {
	// Load existing state (if any)
	if err := m.stateManager.Load(); err != nil && m.verbose {
		fmt.Printf("[DEBUG] Failed to load resume state: %v\n", err)
	}

	// Update input hash
	if err := m.stateManager.UpdateInputHash(m.inputFiles); err != nil && m.verbose {
		fmt.Printf("[DEBUG] Failed to update input hash: %v\n", err)
	}

	// Update config hash
	if err := m.stateManager.UpdateConfigHash(m.config.Persona, m.config.Stack, m.config.Preferences); err != nil && m.verbose {
		fmt.Printf("[DEBUG] Failed to update config hash: %v\n", err)
	}
}

// RunAgent spawns and runs a single agent
func (m *Manager) RunAgent(ctx context.Context, name string) Result {
	start := time.Now()

	agentCfg, ok := m.config.Agents[name]
	if !ok {
		return Result{
			Agent: name,
			Error: fmt.Errorf("unknown agent: %s", name),
		}
	}

	// Determine output path based on agent type and mode
	// Spec outputs (architect, qa, security) go to SpecsOutputDir
	// Code outputs (implementer, verifier) go to CodeOutputDir in modify mode,
	// but in create mode, the agent output already includes the code/ prefix
	var outputPath string
	if isSpecAgent(name) {
		outputPath = filepath.Join(m.config.GetEffectiveSpecsOutputDir(), agentCfg.Output)
	} else if isCodeAgent(name) && m.config.IsModifyMode() {
		// In modify mode, code goes to target codebase
		outputPath = filepath.Join(m.config.GetEffectiveCodeOutputDir(), agentCfg.Output)
	} else {
		// In create mode or for non-code agents, use OutputDir directly
		// (agent output like "code/.complete" already has the code/ prefix)
		outputPath = filepath.Join(m.config.OutputDir, agentCfg.Output)
	}
	absOutputPath, _ := filepath.Abs(outputPath)

	// Resume mode: use content hashing to determine if regeneration is needed
	if m.config.ResumeMode {
		deps := m.config.GetDependencies(name)
		shouldRegen, reason := m.stateManager.ShouldRegenerate(name, absOutputPath, deps)
		if !shouldRegen {
			if m.verbose {
				fmt.Printf("[DEBUG] Skipping agent %s - %s: %s\n", name, reason, absOutputPath)
			}
			return Result{
				Agent:      name,
				OutputPath: absOutputPath,
				Duration:   time.Since(start),
			}
		}
		if m.verbose {
			fmt.Printf("[DEBUG] Regenerating %s - %s\n", name, reason)
		}
	}

	// Allocate port
	port := m.allocatePort()

	if m.verbose {
		fmt.Printf("[DEBUG] Starting agent %s on port %d\n", name, port)
	}

	// Build the prompt using template loader
	absOutputDir, _ := filepath.Abs(m.config.OutputDir)

	// In force mode, don't pass existing files (treat as fresh generation)
	var existingFiles []string
	if !m.config.ForceMode {
		existingFiles = m.listExistingFiles(absOutputDir)
	}

	// Determine effective output directories based on mode
	specsOutputDir := m.config.GetEffectiveSpecsOutputDir()
	codeOutputDir := m.config.GetEffectiveCodeOutputDir()
	absSpecsOutputDir, _ := filepath.Abs(specsOutputDir)
	absCodeOutputDir, _ := filepath.Abs(codeOutputDir)

	promptVars := prompt.Variables{
		PRDPath:       m.prdPath,
		InputFiles:    m.inputFiles,
		InputDir:      m.inputDir,
		HasMultiInput: len(m.inputFiles) > 1,
		OutputDir:     absOutputDir,
		OutputPath:    absOutputPath,
		AgentName:     name,
		ExistingFiles: existingFiles,
		HasExisting:   len(existingFiles) > 0 && !m.config.ForceMode,
		Persona:       m.config.Persona,
		// Stack and Preferences are now the same type in config and prompt packages
		// (both alias types.TechStack and types.ArchitecturePreferences)
		Stack:       m.config.Stack,
		Preferences: m.config.Preferences,
		// Mode-specific variables
		Mode:           m.config.Mode,
		TargetCodebase: m.config.TargetCodebase,
		SpecsOutputDir: absSpecsOutputDir,
		CodeOutputDir:  absCodeOutputDir,
	}

	renderedPrompt, err := m.promptLoader.LoadAndRender(name, agentCfg.Prompt, agentCfg.PromptFile, promptVars)
	if err != nil {
		return Result{
			Agent:    name,
			Error:    fmt.Errorf("failed to load prompt: %w", err),
			Duration: time.Since(start),
		}
	}

	// Start AgentAPI process
	agent, err := m.spawnAgent(ctx, name, port)
	if err != nil {
		return Result{
			Agent:    name,
			Error:    fmt.Errorf("failed to spawn agent: %w", err),
			Duration: time.Since(start),
		}
	}

	m.mu.Lock()
	m.agents[name] = agent
	m.mu.Unlock()

	// Save state for monitoring commands
	_ = m.saveState()

	defer func() {
		m.stopAgent(name)
		_ = m.saveState() // Update state after stopping
	}()

	// Wait for agent API to be healthy
	if err := agent.Client.WaitForHealthy(healthTimeout); err != nil {
		return Result{
			Agent:    name,
			Error:    fmt.Errorf("agent failed to start: %w", err),
			Duration: time.Since(start),
		}
	}

	if m.verbose {
		fmt.Printf("[DEBUG] Agent %s API is healthy, waiting for stable state\n", name)
	}

	// Wait for agent to be ready for input (stable state)
	// Claude Code starts in "running" state while loading
	if err := agent.Client.WaitForStable(healthTimeout); err != nil {
		return Result{
			Agent:    name,
			Error:    fmt.Errorf("agent failed to become stable: %w", err),
			Duration: time.Since(start),
		}
	}

	if m.verbose {
		fmt.Printf("[DEBUG] Agent %s is stable, sending task\n", name)
	}

	// Send the task prompt
	if err := agent.Client.SendMessage(renderedPrompt, "user"); err != nil {
		return Result{
			Agent:    name,
			Error:    fmt.Errorf("failed to send task: %w", err),
			Duration: time.Since(start),
		}
	}

	// Wait for agent to complete (become stable after being running)
	timeout := time.Duration(m.config.Timeout) * time.Second
	if err := m.waitForCompletion(ctx, agent, timeout); err != nil {
		return Result{
			Agent:    name,
			Error:    err,
			Duration: time.Since(start),
		}
	}

	// Verify output file was created
	if _, err := os.Stat(absOutputPath); os.IsNotExist(err) {
		return Result{
			Agent:    name,
			Error:    fmt.Errorf("output file not created: %s", absOutputPath),
			Duration: time.Since(start),
		}
	}

	// Record successful output for resume state tracking
	deps := m.config.GetDependencies(name)
	if err := m.stateManager.RecordAgentOutput(name, absOutputPath, deps); err != nil && m.verbose {
		fmt.Printf("[DEBUG] Failed to record agent output state: %v\n", err)
	}
	if err := m.stateManager.Save(); err != nil && m.verbose {
		fmt.Printf("[DEBUG] Failed to save resume state: %v\n", err)
	}

	return Result{
		Agent:      name,
		OutputPath: absOutputPath,
		Duration:   time.Since(start),
	}
}

// allocatePort returns the next available port
func (m *Manager) allocatePort() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	port := m.portAlloc
	m.portAlloc++
	return port
}

// saveState persists agent state to disk for monitoring commands
func (m *Manager) saveState() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state := make(map[string]int)
	for name, agent := range m.agents {
		state[name] = agent.Port
	}

	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	return os.WriteFile(StateFile, data, 0644)
}

// ClearState removes the state file
func ClearState() {
	_ = os.Remove(StateFile)
}

// RemoveAgentFromState removes a specific agent from the state file
func RemoveAgentFromState(agentName string) error {
	state, err := LoadState()
	if err != nil {
		return err
	}

	delete(state, agentName)

	if len(state) == 0 {
		ClearState()
		return nil
	}

	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	return os.WriteFile(StateFile, data, 0644)
}

// LoadState loads agent state from disk
func LoadState() (map[string]int, error) {
	data, err := os.ReadFile(StateFile)
	if err != nil {
		return nil, err
	}

	var state map[string]int
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return state, nil
}

// isSpecAgent returns true if the agent produces specification documents
func isSpecAgent(name string) bool {
	specAgents := map[string]bool{
		"architect": true,
		"qa":        true,
		"security":  true,
	}
	return specAgents[name]
}

// isCodeAgent returns true if the agent produces or modifies code
func isCodeAgent(name string) bool {
	codeAgents := map[string]bool{
		"implementer": true,
		"verifier":    true,
	}
	return codeAgents[name]
}

// listExistingFiles returns a list of files in the output directory
func (m *Manager) listExistingFiles(outputDir string) []string {
	var files []string

	err := filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Ignore errors
		}
		if info.IsDir() {
			return nil
		}
		// Get relative path from output dir
		relPath, err := filepath.Rel(outputDir, path)
		if err != nil {
			return nil
		}
		files = append(files, relPath)
		return nil
	})

	if err != nil {
		return nil
	}

	return files
}
