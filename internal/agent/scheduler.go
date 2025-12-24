package agent

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
