package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tuannvm/pagent/internal/agent"
	"github.com/tuannvm/pagent/internal/api"
	"github.com/tuannvm/pagent/internal/config"
)

// AgentDescriptions maps agent names to their descriptions.
var AgentDescriptions = map[string]string{
	"architect":   "Analyzes PRD and creates technical architecture document",
	"qa":          "Creates comprehensive test plan based on architecture",
	"security":    "Performs security assessment and threat modeling",
	"implementer": "Implements the code based on architecture and security specs",
	"verifier":    "Verifies implementation against test plan and requirements",
}

// Handlers provides the business logic for MCP tool handlers.
// It can be used standalone or injected into the MCP server.
type Handlers struct {
	configPath string // Optional config file path
	verbose    bool
}

// NewHandlers creates a new Handlers instance.
func NewHandlers() *Handlers {
	return &Handlers{}
}

// WithConfigPath sets the config file path.
func (h *Handlers) WithConfigPath(path string) *Handlers {
	h.configPath = path
	return h
}

// WithVerbose enables verbose logging.
func (h *Handlers) WithVerbose(verbose bool) *Handlers {
	h.verbose = verbose
	return h
}

// loadConfig loads the config file or returns defaults.
func (h *Handlers) loadConfig() *config.Config {
	cfg, err := config.Load(h.configPath)
	if err != nil {
		return config.Default()
	}
	return cfg
}

// RunAgent executes a single agent.
func (h *Handlers) RunAgent(ctx context.Context, input RunAgentInput) RunAgentOutput {
	// Validate input
	if input.PRDPath == "" {
		return RunAgentOutput{Success: false, Error: "prd_path is required"}
	}
	if input.AgentName == "" {
		return RunAgentOutput{Success: false, Error: "agent_name is required"}
	}

	// Check PRD file exists
	absPath, err := filepath.Abs(input.PRDPath)
	if err != nil {
		return RunAgentOutput{Success: false, Error: fmt.Sprintf("invalid path: %v", err)}
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return RunAgentOutput{Success: false, Error: fmt.Sprintf("PRD file not found: %s", absPath)}
	}

	// Load config
	cfg := h.loadConfig()

	// Apply overrides
	if input.OutputDir != "" {
		cfg.OutputDir = input.OutputDir
	}
	if input.Persona != "" {
		if !config.IsValidPersona(input.Persona) {
			return RunAgentOutput{Success: false, Error: fmt.Sprintf("invalid persona: %s", input.Persona)}
		}
		cfg.Persona = input.Persona
	}

	// Validate agent exists
	if _, ok := cfg.Agents[input.AgentName]; !ok {
		return RunAgentOutput{Success: false, Error: fmt.Sprintf("unknown agent: %s", input.AgentName)}
	}

	// Create manager and run agent
	verbose := input.Verbose || h.verbose
	manager := agent.NewManager(cfg, absPath, verbose)
	result := manager.RunAgent(ctx, input.AgentName)

	output := RunAgentOutput{
		Agent:      result.Agent,
		OutputPath: result.OutputPath,
		Duration:   result.Duration.String(),
		Success:    result.Error == nil,
	}
	if result.Error != nil {
		output.Error = result.Error.Error()
	}

	return output
}

// RunPipeline executes the full agent pipeline.
func (h *Handlers) RunPipeline(ctx context.Context, input RunPipelineInput) (RunPipelineOutput, error) {
	// Validate input
	if input.PRDPath == "" {
		return RunPipelineOutput{}, fmt.Errorf("prd_path is required")
	}

	// Check PRD file exists
	absPath, err := filepath.Abs(input.PRDPath)
	if err != nil {
		return RunPipelineOutput{}, fmt.Errorf("invalid path: %v", err)
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return RunPipelineOutput{}, fmt.Errorf("PRD file not found: %s", absPath)
	}

	// Load config
	cfg := h.loadConfig()

	// Apply overrides
	if input.OutputDir != "" {
		cfg.OutputDir = input.OutputDir
	}
	if input.Persona != "" {
		if !config.IsValidPersona(input.Persona) {
			return RunPipelineOutput{}, fmt.Errorf("invalid persona: %s", input.Persona)
		}
		cfg.Persona = input.Persona
	}

	// Determine which agents to run
	agentsToRun := input.Agents
	if len(agentsToRun) == 0 {
		agentsToRun = cfg.GetAgentNames()
	}

	// Validate all agents exist
	for _, name := range agentsToRun {
		if _, ok := cfg.Agents[name]; !ok {
			return RunPipelineOutput{}, fmt.Errorf("unknown agent: %s", name)
		}
	}

	// Create manager
	verbose := input.Verbose || h.verbose
	manager := agent.NewManager(cfg, absPath, verbose)

	// Run agents based on execution mode
	var results []RunAgentOutput
	var successful, failed int

	if input.Sequential {
		// Sequential execution
		order := manager.TopologicalSort(agentsToRun)
		for _, name := range order {
			result := manager.RunAgent(ctx, name)
			output := RunAgentOutput{
				Agent:      result.Agent,
				OutputPath: result.OutputPath,
				Duration:   result.Duration.String(),
				Success:    result.Error == nil,
			}
			if result.Error != nil {
				output.Error = result.Error.Error()
				failed++
			} else {
				successful++
			}
			results = append(results, output)
		}
	} else {
		// Parallel by dependency level
		levels := manager.GetDependencyLevels(agentsToRun)
		for _, level := range levels {
			levelResults := make(chan agent.Result, len(level))
			for _, name := range level {
				go func(agentName string) {
					levelResults <- manager.RunAgent(ctx, agentName)
				}(name)
			}

			for range level {
				result := <-levelResults
				output := RunAgentOutput{
					Agent:      result.Agent,
					OutputPath: result.OutputPath,
					Duration:   result.Duration.String(),
					Success:    result.Error == nil,
				}
				if result.Error != nil {
					output.Error = result.Error.Error()
					failed++
				} else {
					successful++
				}
				results = append(results, output)
			}
		}
	}

	return RunPipelineOutput{
		Results:     results,
		TotalAgents: len(agentsToRun),
		Successful:  successful,
		Failed:      failed,
	}, nil
}

// ListAgents returns all available agents.
func (h *Handlers) ListAgents(_ context.Context, _ ListAgentsInput) ListAgentsOutput {
	cfg := h.loadConfig()

	var agents []AgentInfo
	for name, agentCfg := range cfg.Agents {
		agents = append(agents, AgentInfo{
			Name:        name,
			Output:      agentCfg.Output,
			DependsOn:   agentCfg.DependsOn,
			Description: AgentDescriptions[name],
		})
	}

	return ListAgentsOutput{Agents: agents}
}

// GetStatus returns the status of running agents.
func (h *Handlers) GetStatus(_ context.Context, input GetStatusInput) GetStatusOutput {
	state, err := agent.LoadState()
	if err != nil {
		return GetStatusOutput{Agents: []AgentStatus{}}
	}

	var agents []AgentStatus
	for name, port := range state {
		if input.AgentName != "" && name != input.AgentName {
			continue
		}

		client := api.NewClient(port)
		status, err := client.GetStatus()
		statusStr := "unknown"
		if err == nil {
			statusStr = status.Status
		}

		agents = append(agents, AgentStatus{
			Name:   name,
			Port:   port,
			Status: statusStr,
		})
	}

	return GetStatusOutput{Agents: agents}
}

// SendMessage sends a message to a running agent.
func (h *Handlers) SendMessage(_ context.Context, input SendMessageInput) SendMessageOutput {
	if input.AgentName == "" {
		return SendMessageOutput{Success: false, Error: "agent_name is required"}
	}
	if input.Message == "" {
		return SendMessageOutput{Success: false, Error: "message is required"}
	}

	state, err := agent.LoadState()
	if err != nil {
		return SendMessageOutput{Success: false, Error: "no running agents found"}
	}

	port, ok := state[input.AgentName]
	if !ok {
		available := make([]string, 0, len(state))
		for name := range state {
			available = append(available, name)
		}
		return SendMessageOutput{
			Success: false,
			Error:   fmt.Sprintf("agent %q not running. Available: %s", input.AgentName, strings.Join(available, ", ")),
		}
	}

	client := api.NewClient(port)
	if err := client.SendMessage(input.Message, "user"); err != nil {
		return SendMessageOutput{Success: false, Error: err.Error()}
	}

	return SendMessageOutput{Success: true}
}

// StopAgents stops running agents.
func (h *Handlers) StopAgents(_ context.Context, input StopAgentsInput) StopAgentsOutput {
	state, err := agent.LoadState()
	if err != nil {
		return StopAgentsOutput{Stopped: []string{}, Success: true}
	}

	var stopped []string
	if input.AgentName != "" {
		if _, ok := state[input.AgentName]; ok {
			stopped = append(stopped, input.AgentName)
		}
	} else {
		for name := range state {
			stopped = append(stopped, name)
		}
	}

	agent.ClearState()

	return StopAgentsOutput{Stopped: stopped, Success: true}
}
