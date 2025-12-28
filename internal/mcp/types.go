// Package mcp provides MCP (Model Context Protocol) server functionality for pagent.
// It exposes pagent's agent orchestration capabilities as MCP tools.
package mcp

// RunAgentInput defines parameters for running a single agent.
type RunAgentInput struct {
	PRDPath   string `json:"prd_path" jsonschema:"Absolute path to the PRD or requirements file"`
	AgentName string `json:"agent_name" jsonschema:"Name of the agent to run (architect/qa/security/implementer/verifier)"`
	OutputDir string `json:"output_dir,omitempty" jsonschema:"Output directory for generated files (default: ./outputs)"`
	Persona   string `json:"persona,omitempty" jsonschema:"Implementation style: minimal/balanced/production (default: balanced)"`
	Verbose   bool   `json:"verbose,omitempty" jsonschema:"Enable verbose debug output"`
}

// RunAgentOutput contains the result of running an agent.
type RunAgentOutput struct {
	Agent      string `json:"agent"`
	OutputPath string `json:"output_path"`
	Duration   string `json:"duration"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
}

// RunPipelineInput defines parameters for running the full agent pipeline.
type RunPipelineInput struct {
	PRDPath    string   `json:"prd_path" jsonschema:"Absolute path to the PRD or requirements file"`
	Agents     []string `json:"agents,omitempty" jsonschema:"Specific agents to run (default: all agents in dependency order)"`
	OutputDir  string   `json:"output_dir,omitempty" jsonschema:"Output directory for generated files (default: ./outputs)"`
	Persona    string   `json:"persona,omitempty" jsonschema:"Implementation style: minimal/balanced/production (default: balanced)"`
	Sequential bool     `json:"sequential,omitempty" jsonschema:"Run agents sequentially instead of parallel-by-level"`
	Verbose    bool     `json:"verbose,omitempty" jsonschema:"Enable verbose debug output"`
}

// RunPipelineOutput contains the results of running the pipeline.
type RunPipelineOutput struct {
	Results       []RunAgentOutput `json:"results"`
	TotalAgents   int              `json:"total_agents"`
	Successful    int              `json:"successful"`
	Failed        int              `json:"failed"`
	TotalDuration string           `json:"total_duration"`
}

// ListAgentsInput defines parameters for listing agents.
type ListAgentsInput struct{}

// AgentInfo describes an available agent.
type AgentInfo struct {
	Name        string   `json:"name"`
	Output      string   `json:"output"`
	DependsOn   []string `json:"depends_on"`
	Description string   `json:"description"`
}

// ListAgentsOutput contains available agents.
type ListAgentsOutput struct {
	Agents []AgentInfo `json:"agents"`
}

// GetStatusInput defines parameters for getting agent status.
type GetStatusInput struct {
	AgentName string `json:"agent_name,omitempty" jsonschema:"Specific agent to check (empty for all running agents)"`
}

// AgentStatus describes the status of a running agent.
type AgentStatus struct {
	Name      string `json:"name"`
	Port      int    `json:"port"`
	Status    string `json:"status"` // "running" or "stable"
	StartedAt string `json:"started_at,omitempty"`
}

// GetStatusOutput contains agent status information.
type GetStatusOutput struct {
	Agents []AgentStatus `json:"agents"`
}

// SendMessageInput defines parameters for sending a message to a running agent.
type SendMessageInput struct {
	AgentName string `json:"agent_name" jsonschema:"Name of the running agent to message"`
	Message   string `json:"message" jsonschema:"Message content to send to the agent"`
}

// SendMessageOutput contains the result of sending a message.
type SendMessageOutput struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// StopAgentsInput defines parameters for stopping agents.
type StopAgentsInput struct {
	AgentName string `json:"agent_name,omitempty" jsonschema:"Specific agent to stop (empty to stop all)"`
}

// StopAgentsOutput contains the result of stopping agents.
type StopAgentsOutput struct {
	Stopped []string `json:"stopped"`
	Success bool     `json:"success"`
	Error   string   `json:"error,omitempty"`
}
