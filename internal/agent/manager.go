package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/tuannvm/pm-agent-workflow/internal/api"
	"github.com/tuannvm/pm-agent-workflow/internal/config"
)

const (
	basePort      = 3284
	healthTimeout = 30 * time.Second
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
	config    *config.Config
	prdPath   string
	verbose   bool
	agents    map[string]*RunningAgent
	portAlloc int
	mu        sync.Mutex
}

// NewManager creates a new agent manager
func NewManager(cfg *config.Config, prdPath string, verbose bool) *Manager {
	return &Manager{
		config:    cfg,
		prdPath:   prdPath,
		verbose:   verbose,
		agents:    make(map[string]*RunningAgent),
		portAlloc: basePort,
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

	// Allocate port
	port := m.allocatePort()
	outputPath := filepath.Join(m.config.OutputDir, agentCfg.Output)
	absOutputPath, _ := filepath.Abs(outputPath)

	if m.verbose {
		fmt.Printf("[DEBUG] Starting agent %s on port %d\n", name, port)
	}

	// Build the prompt with substitutions
	prompt := strings.ReplaceAll(agentCfg.Prompt, "{prd_path}", m.prdPath)
	prompt = strings.ReplaceAll(prompt, "{output_path}", absOutputPath)

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

	// Wait for agent to be healthy
	if err := agent.Client.WaitForHealthy(healthTimeout); err != nil {
		return Result{
			Agent:    name,
			Error:    fmt.Errorf("agent failed to start: %w", err),
			Duration: time.Since(start),
		}
	}

	if m.verbose {
		fmt.Printf("[DEBUG] Agent %s is healthy, sending task\n", name)
	}

	// Send the task prompt
	if err := agent.Client.SendMessage(prompt, "user"); err != nil {
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

	// Capture stderr for debugging
	if m.verbose {
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
	deadline := time.Now().Add(timeout)
	wasRunning := false

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		status, err := agent.Client.GetStatus()
		if err != nil {
			// Process might have crashed
			if agent.Process.ProcessState != nil && agent.Process.ProcessState.Exited() {
				return fmt.Errorf("agent process exited unexpectedly")
			}
			time.Sleep(1 * time.Second)
			continue
		}

		if status.Status == "running" {
			wasRunning = true
		}

		// Agent is done when it transitions from running to stable
		if wasRunning && status.Status == "stable" {
			return nil
		}

		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("timeout waiting for agent to complete")
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
	// Build adjacency list
	agentSet := make(map[string]bool)
	for _, a := range agents {
		agentSet[a] = true
	}

	// Kahn's algorithm
	inDegree := make(map[string]int)
	for _, a := range agents {
		inDegree[a] = 0
	}

	for _, a := range agents {
		deps := m.config.GetDependencies(a)
		for _, dep := range deps {
			if agentSet[dep] {
				inDegree[a]++
			}
		}
	}

	// Find all nodes with no incoming edges
	var queue []string
	for _, a := range agents {
		if inDegree[a] == 0 {
			queue = append(queue, a)
		}
	}

	var result []string
	for len(queue) > 0 {
		// Pop from queue
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)

		// For each agent that depends on this node
		for _, a := range agents {
			deps := m.config.GetDependencies(a)
			for _, dep := range deps {
				if dep == node {
					inDegree[a]--
					if inDegree[a] == 0 {
						queue = append(queue, a)
					}
				}
			}
		}
	}

	return result
}
