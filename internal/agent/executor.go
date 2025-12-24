package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/tuannvm/pagent/internal/api"
)

// spawnAgent starts an agent using the agentapi library
func (m *Manager) spawnAgent(ctx context.Context, name string, port int) (*RunningAgent, error) {
	libClient, err := NewLibClient(ctx, LibClientConfig{
		Port:     port,
		Verbose:  m.verbose,
		AgentCmd: "claude",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create lib client: %w", err)
	}

	// Start the HTTP server
	if err := libClient.Start(); err != nil {
		_ = libClient.Close(ctx)
		return nil, fmt.Errorf("failed to start lib server: %w", err)
	}

	return &RunningAgent{
		Name:      name,
		Port:      port,
		LibClient: libClient,
		Client:    api.NewClient(port), // HTTP client for status polling
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

	if agent.LibClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = agent.LibClient.Close(ctx)
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
