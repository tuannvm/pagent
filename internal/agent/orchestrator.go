// Package agent provides agent lifecycle management and orchestration.
package agent

import "context"

// Orchestrator defines the interface for agent orchestration.
// This abstraction enables:
// - Unit testing with mock implementations
// - Alternative implementations (e.g., remote agent execution)
// - Dependency injection in calling code
type Orchestrator interface {
	// RunAgent executes a single agent and returns the result.
	RunAgent(ctx context.Context, name string) Result

	// TopologicalSort returns agents in dependency order.
	TopologicalSort(agents []string) []string

	// GetDependencyLevels groups agents by dependency level for parallel execution.
	// Level 0 agents have no dependencies, level 1 depends only on level 0, etc.
	GetDependencyLevels(agents []string) [][]string

	// ExpandWithDependencies takes a list of agents and returns the list expanded
	// to include all transitive dependencies in dependency order.
	ExpandWithDependencies(agents []string) []string

	// GetTransitiveDependencies returns all dependencies for an agent, including transitive ones.
	GetTransitiveDependencies(agentName string) []string

	// StopAll stops all running agents.
	StopAll()

	// GetRunningAgents returns currently running agents.
	GetRunningAgents() []*RunningAgent
}

// Verify Manager implements Orchestrator at compile time
var _ Orchestrator = (*Manager)(nil)

// OrchestratorConfig contains configuration for creating an orchestrator.
// This provides a cleaner factory interface than passing many parameters.
type OrchestratorConfig struct {
	// Config is the pagent configuration
	Config interface {
		GetAgentNames() []string
		GetDependencies(name string) []string
	}

	// PrimaryFile is the main input file
	PrimaryFile string

	// InputFiles is the list of all input files
	InputFiles []string

	// InputDir is the input directory (empty if single file)
	InputDir string

	// Verbose enables debug logging
	Verbose bool
}
