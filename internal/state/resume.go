// Package state manages resume state for incremental agent execution.
// It tracks content hashes to detect when inputs change and regeneration is needed.
package state

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// ResumeState tracks the state of previous agent runs for resumability.
type ResumeState struct {
	// InputHash is the combined hash of all input files
	InputHash string `json:"input_hash"`

	// ConfigHash is the hash of relevant configuration (persona, preferences, stack)
	ConfigHash string `json:"config_hash"`

	// AgentOutputs maps agent names to their output state
	AgentOutputs map[string]AgentOutput `json:"agent_outputs"`
}

// AgentOutput tracks the output state of a single agent.
type AgentOutput struct {
	// OutputPath is the path to the agent's output file
	OutputPath string `json:"output_path"`

	// OutputHash is the hash of the output content
	OutputHash string `json:"output_hash"`

	// InputHashAtGeneration is the input hash when this output was generated
	InputHashAtGeneration string `json:"input_hash_at_generation"`

	// ConfigHashAtGeneration is the config hash when this output was generated
	ConfigHashAtGeneration string `json:"config_hash_at_generation"`

	// DependencyHashes maps dependency agent names to their output hashes when this was generated
	DependencyHashes map[string]string `json:"dependency_hashes"`
}

// StateFile is the default location for resume state
const StateFile = ".pm-agents/.resume-state.json"

// Manager handles resume state operations.
type Manager struct {
	outputDir string
	state     *ResumeState
	statePath string
}

// NewManager creates a new state manager for the given output directory.
func NewManager(outputDir string) *Manager {
	return &Manager{
		outputDir: outputDir,
		statePath: filepath.Join(outputDir, StateFile),
		state: &ResumeState{
			AgentOutputs: make(map[string]AgentOutput),
		},
	}
}

// Load loads the resume state from disk.
func (m *Manager) Load() error {
	data, err := os.ReadFile(m.statePath)
	if err != nil {
		if os.IsNotExist(err) {
			// No state file - fresh start
			m.state = &ResumeState{
				AgentOutputs: make(map[string]AgentOutput),
			}
			return nil
		}
		return fmt.Errorf("failed to read resume state: %w", err)
	}

	if err := json.Unmarshal(data, &m.state); err != nil {
		return fmt.Errorf("failed to parse resume state: %w", err)
	}

	return nil
}

// Save persists the resume state to disk.
func (m *Manager) Save() error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(m.statePath), 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	data, err := json.MarshalIndent(m.state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal resume state: %w", err)
	}

	if err := os.WriteFile(m.statePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write resume state: %w", err)
	}

	return nil
}

// UpdateInputHash computes and stores the hash of all input files.
func (m *Manager) UpdateInputHash(inputFiles []string) error {
	hash, err := hashFiles(inputFiles)
	if err != nil {
		return fmt.Errorf("failed to hash input files: %w", err)
	}
	m.state.InputHash = hash
	return nil
}

// UpdateConfigHash computes and stores a hash of the relevant config.
func (m *Manager) UpdateConfigHash(persona string, stack, preferences interface{}) error {
	// Create a deterministic representation of config
	configData := map[string]interface{}{
		"persona":     persona,
		"stack":       stack,
		"preferences": preferences,
	}

	data, err := json.Marshal(configData)
	if err != nil {
		return fmt.Errorf("failed to marshal config for hashing: %w", err)
	}

	m.state.ConfigHash = hashBytes(data)
	return nil
}

// RecordAgentOutput records the output of an agent for future resume checks.
func (m *Manager) RecordAgentOutput(agentName, outputPath string, dependencyAgents []string) error {
	// Hash the output file
	outputHash, err := hashFile(outputPath)
	if err != nil {
		return fmt.Errorf("failed to hash output file: %w", err)
	}

	// Collect dependency hashes
	depHashes := make(map[string]string)
	for _, dep := range dependencyAgents {
		if depOutput, ok := m.state.AgentOutputs[dep]; ok {
			depHashes[dep] = depOutput.OutputHash
		}
	}

	m.state.AgentOutputs[agentName] = AgentOutput{
		OutputPath:             outputPath,
		OutputHash:             outputHash,
		InputHashAtGeneration:  m.state.InputHash,
		ConfigHashAtGeneration: m.state.ConfigHash,
		DependencyHashes:       depHashes,
	}

	return nil
}

// ShouldRegenerate determines if an agent needs to be regenerated.
// Returns true if the agent should run, false if it can be skipped.
func (m *Manager) ShouldRegenerate(agentName, outputPath string, dependencyAgents []string) (bool, string) {
	agentOutput, exists := m.state.AgentOutputs[agentName]
	if !exists {
		return true, "no previous output recorded"
	}

	// Check if output file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return true, "output file does not exist"
	}

	// Check if output file changed externally (user edited it)
	currentHash, err := hashFile(outputPath)
	if err != nil {
		return true, fmt.Sprintf("failed to hash current output: %v", err)
	}
	if currentHash != agentOutput.OutputHash {
		return true, "output file was modified externally"
	}

	// Check if inputs changed
	if m.state.InputHash != agentOutput.InputHashAtGeneration {
		return true, "input files changed"
	}

	// Check if config changed
	if m.state.ConfigHash != agentOutput.ConfigHashAtGeneration {
		return true, "configuration changed"
	}

	// Check if dependencies changed
	for _, dep := range dependencyAgents {
		currentDepOutput, depExists := m.state.AgentOutputs[dep]
		if !depExists {
			return true, fmt.Sprintf("dependency %s has no recorded output", dep)
		}

		recordedDepHash, wasRecorded := agentOutput.DependencyHashes[dep]
		if !wasRecorded {
			return true, fmt.Sprintf("dependency %s was not recorded at generation time", dep)
		}

		if currentDepOutput.OutputHash != recordedDepHash {
			return true, fmt.Sprintf("dependency %s output changed", dep)
		}
	}

	return false, "up-to-date"
}

// Clear removes all resume state.
func (m *Manager) Clear() error {
	m.state = &ResumeState{
		AgentOutputs: make(map[string]AgentOutput),
	}
	return os.Remove(m.statePath)
}

// hashFiles computes a combined hash of multiple files.
func hashFiles(paths []string) (string, error) {
	// Sort for deterministic ordering
	sorted := make([]string, len(paths))
	copy(sorted, paths)
	sort.Strings(sorted)

	h := sha256.New()
	for _, path := range sorted {
		// Include the relative path in the hash (so renames are detected)
		h.Write([]byte(path))
		h.Write([]byte{0}) // separator

		content, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("failed to read %s: %w", path, err)
		}
		h.Write(content)
		h.Write([]byte{0}) // separator
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// hashFile computes the SHA-256 hash of a single file.
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// hashBytes computes the SHA-256 hash of bytes.
func hashBytes(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
