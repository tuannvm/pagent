package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/tuannvm/pm-agent-workflow/internal/api"
	"github.com/tuannvm/pm-agent-workflow/internal/config"
	"github.com/tuannvm/pm-agent-workflow/internal/prompt"
	"github.com/tuannvm/pm-agent-workflow/internal/state"
)

const (
	basePort      = 3284
	healthTimeout = 120 * time.Second // 2 min for Claude Code to fully initialize
)

// State file paths
var (
	StateFile = filepath.Join(os.TempDir(), "pm-agents-state.json")
)

// Result represents the result of running an agent
type Result struct {
	Agent      string
	OutputPath string
	Error      error
	Duration   time.Duration
}

// RunningAgent tracks a running agent process
type RunningAgent struct {
	Name    string
	Port    int
	Process *exec.Cmd
	Client  *api.Client
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
	// Code outputs (implementer, verifier) go to CodeOutputDir (or TargetCodebase in modify mode)
	var outputPath string
	if isSpecAgent(name) {
		outputPath = filepath.Join(m.config.GetEffectiveSpecsOutputDir(), agentCfg.Output)
	} else if isCodeAgent(name) {
		outputPath = filepath.Join(m.config.GetEffectiveCodeOutputDir(), agentCfg.Output)
	} else {
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

// spawnAgent starts an AgentAPI process
func (m *Manager) spawnAgent(ctx context.Context, name string, port int) (*RunningAgent, error) {
	// Check if agentapi is available
	agentapiPath, err := exec.LookPath("agentapi")
	if err != nil {
		return nil, fmt.Errorf("agentapi not found in PATH: %w", err)
	}

	// Build command: agentapi server --port <port> -- claude
	cmd := exec.CommandContext(ctx, agentapiPath, "server", "--port", fmt.Sprintf("%d", port), "--", "claude")

	// Set process group so we can kill all children
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Capture stdout/stderr for debugging
	if m.verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start agentapi: %w", err)
	}

	return &RunningAgent{
		Name:      name,
		Port:      port,
		Process:   cmd,
		Client:    api.NewClient(port),
		StartedAt: time.Now(),
	}, nil
}

// waitForCompletion waits for agent to finish processing
func (m *Manager) waitForCompletion(ctx context.Context, agent *RunningAgent, timeout time.Duration) error {
	start := time.Now()
	wasRunning := false
	lastStatus := ""
	lastProgressLog := time.Now()
	pollInterval := 1 * time.Second
	consecutiveErrors := 0
	maxConsecutiveErrors := 30 // 30 consecutive failures (~30s) indicates dead agent

	for {
		// Check timeout (0 = no timeout, poll indefinitely)
		if timeout > 0 && time.Since(start) > timeout {
			return fmt.Errorf("timeout waiting for agent to complete")
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		status, err := agent.Client.GetStatus()
		if err != nil {
			consecutiveErrors++
			if consecutiveErrors >= maxConsecutiveErrors {
				return fmt.Errorf("agent API unreachable after %d consecutive failures - process likely crashed", consecutiveErrors)
			}
			if m.verbose && consecutiveErrors%10 == 0 {
				fmt.Printf("[DEBUG] Agent %s API error (attempt %d/%d): %v\n",
					agent.Name, consecutiveErrors, maxConsecutiveErrors, err)
			}
			time.Sleep(pollInterval)
			continue
		}
		consecutiveErrors = 0 // Reset on successful API call

		// Track status transitions
		if status.Status != lastStatus {
			if m.verbose {
				fmt.Printf("[DEBUG] Agent %s status: %s (elapsed: %s)\n",
					agent.Name, status.Status, time.Since(start).Round(time.Second))
			}
			lastStatus = status.Status
		}

		if status.Status == "running" {
			wasRunning = true
		}

		// Agent is done when it transitions from running to stable
		if wasRunning && status.Status == "stable" {
			if m.verbose {
				fmt.Printf("[DEBUG] Agent %s completed in %s\n",
					agent.Name, time.Since(start).Round(time.Second))
			}
			return nil
		}

		// Progress indicator every 30 seconds
		if m.verbose && time.Since(lastProgressLog) > 30*time.Second {
			fmt.Printf("[DEBUG] Agent %s still %s... (elapsed: %s)\n",
				agent.Name, status.Status, time.Since(start).Round(time.Second))
			lastProgressLog = time.Now()
		}

		time.Sleep(pollInterval)
	}
}

// stopAgent gracefully stops an agent
func (m *Manager) stopAgent(name string) {
	m.mu.Lock()
	agent, ok := m.agents[name]
	if !ok {
		m.mu.Unlock()
		return
	}
	delete(m.agents, name)
	m.mu.Unlock()

	if agent.Process != nil && agent.Process.Process != nil {
		// Kill the process group
		pgid, err := syscall.Getpgid(agent.Process.Process.Pid)
		if err == nil {
			_ = syscall.Kill(-pgid, syscall.SIGTERM)
		} else {
			_ = agent.Process.Process.Kill()
		}
		_ = agent.Process.Wait()
	}
}

// StopAll stops all running agents
func (m *Manager) StopAll() {
	m.mu.Lock()
	names := make([]string, 0, len(m.agents))
	for name := range m.agents {
		names = append(names, name)
	}
	m.mu.Unlock()

	for _, name := range names {
		m.stopAgent(name)
	}
}

// GetRunningAgents returns currently running agents
func (m *Manager) GetRunningAgents() []*RunningAgent {
	m.mu.Lock()
	defer m.mu.Unlock()

	agents := make([]*RunningAgent, 0, len(m.agents))
	for _, agent := range m.agents {
		agents = append(agents, agent)
	}
	return agents
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

// TopologicalSort returns agents in dependency order
func (m *Manager) TopologicalSort(agents []string) []string {
	levels := m.GetDependencyLevels(agents)
	var result []string
	for _, level := range levels {
		result = append(result, level...)
	}
	return result
}

// GetDependencyLevels groups agents by dependency level for parallel execution.
// Level 0: agents with no dependencies
// Level 1: agents whose dependencies are all in level 0
// Level N: agents whose dependencies are all in levels 0..N-1
// Returns a slice of levels, where each level is a slice of agent names.
func (m *Manager) GetDependencyLevels(agents []string) [][]string {
	// Build agent set for filtering
	agentSet := make(map[string]bool)
	for _, a := range agents {
		agentSet[a] = true
	}

	// Calculate in-degree for each agent (only counting dependencies in our set)
	inDegree := make(map[string]int)
	for _, a := range agents {
		inDegree[a] = 0
		deps := m.config.GetDependencies(a)
		for _, dep := range deps {
			if agentSet[dep] {
				inDegree[a]++
			}
		}
	}

	// Track which agents have been assigned to a level
	assigned := make(map[string]bool)
	var levels [][]string

	// Keep building levels until all agents are assigned
	for len(assigned) < len(agents) {
		var currentLevel []string

		// Find all agents whose dependencies are satisfied (in-degree == 0)
		for _, a := range agents {
			if !assigned[a] && inDegree[a] == 0 {
				currentLevel = append(currentLevel, a)
			}
		}

		// If no agents can be added, we have a cycle (shouldn't happen with valid config)
		if len(currentLevel) == 0 {
			break
		}

		// Mark agents in this level as assigned
		for _, a := range currentLevel {
			assigned[a] = true
		}

		// Reduce in-degree for agents that depend on this level
		for _, completed := range currentLevel {
			for _, a := range agents {
				if assigned[a] {
					continue
				}
				deps := m.config.GetDependencies(a)
				for _, dep := range deps {
					if dep == completed {
						inDegree[a]--
					}
				}
			}
		}

		levels = append(levels, currentLevel)
	}

	return levels
}

// GetTransitiveDependencies returns all dependencies for an agent, including transitive ones.
// This is useful for auto-including required agents when a user requests a specific agent.
func (m *Manager) GetTransitiveDependencies(agentName string) []string {
	visited := make(map[string]bool)
	var result []string

	var visit func(name string)
	visit = func(name string) {
		if visited[name] {
			return
		}
		visited[name] = true

		deps := m.config.GetDependencies(name)
		for _, dep := range deps {
			visit(dep)
		}
		// Add after visiting dependencies (reverse topological order)
		if name != agentName { // Don't include the agent itself
			result = append(result, name)
		}
	}

	visit(agentName)
	return result
}

// ExpandWithDependencies takes a list of agents and returns the list expanded
// to include all transitive dependencies. The returned list is in dependency order.
func (m *Manager) ExpandWithDependencies(agents []string) []string {
	// Collect all agents including dependencies
	agentSet := make(map[string]bool)
	for _, a := range agents {
		agentSet[a] = true
		for _, dep := range m.GetTransitiveDependencies(a) {
			agentSet[dep] = true
		}
	}

	// Convert to slice
	var allAgents []string
	for a := range agentSet {
		allAgents = append(allAgents, a)
	}

	// Return in topological order
	return m.TopologicalSort(allAgents)
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
