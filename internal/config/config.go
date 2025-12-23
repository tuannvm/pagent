package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// Config represents the pm-agents configuration
type Config struct {
	OutputDir  string                 `yaml:"output_dir"`
	Timeout    int                    `yaml:"timeout"`
	ResumeMode bool                   `yaml:"-"` // Set via CLI flag, not config file
	ForceMode  bool                   `yaml:"-"` // Set via CLI flag, not config file
	Agents     map[string]AgentConfig `yaml:"agents"`
}

// AgentConfig represents a single agent's configuration
type AgentConfig struct {
	Prompt     string   `yaml:"prompt"`      // Inline prompt (takes precedence)
	PromptFile string   `yaml:"prompt_file"` // Path to prompt template file
	Output     string   `yaml:"output"`
	DependsOn  []string `yaml:"depends_on"`
}

// Load reads config from file, checking multiple locations
func Load(path string) (*Config, error) {
	var configPath string

	if path != "" {
		configPath = path
	} else {
		// Check standard locations
		locations := []string{
			".pm-agents/config.yaml",
			".pm-agents/config.yml",
			filepath.Join(os.Getenv("HOME"), ".pm-agents/config.yaml"),
		}

		for _, loc := range locations {
			if _, err := os.Stat(loc); err == nil {
				configPath = loc
				break
			}
		}
	}

	if configPath == "" {
		return nil, os.ErrNotExist
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Apply defaults
	if cfg.OutputDir == "" {
		cfg.OutputDir = "./outputs"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 300
	}

	// Apply environment variable overrides
	cfg.ApplyEnvOverrides()

	return &cfg, nil
}

// ApplyEnvOverrides applies environment variable overrides to config
func (c *Config) ApplyEnvOverrides() {
	if envDir := os.Getenv("PM_AGENTS_OUTPUT_DIR"); envDir != "" {
		c.OutputDir = envDir
	}
	if envTimeout := os.Getenv("PM_AGENTS_TIMEOUT"); envTimeout != "" {
		var timeout int
		if _, err := fmt.Sscanf(envTimeout, "%d", &timeout); err == nil && timeout > 0 {
			c.Timeout = timeout
		}
	}
}

// Default returns the default configuration
// Prompts are loaded from embedded templates (internal/prompt/templates/)
// Users can override by specifying prompt_file or inline prompt in config
func Default() *Config {
	return &Config{
		OutputDir: "./outputs",
		Timeout:   0, // 0 = no timeout (poll until completion). Set via --timeout for safety net.
		Agents: map[string]AgentConfig{
			// SPECIFICATION PHASE
			"architect": {
				Output:    "architecture.md",
				DependsOn: []string{},
			},
			"qa": {
				Output:    "test-plan.md",
				DependsOn: []string{"architect"},
			},
			"security": {
				Output:    "security-assessment.md",
				DependsOn: []string{"architect"},
			},
			// IMPLEMENTATION PHASE
			"implementer": {
				Output:    "code/.complete",
				DependsOn: []string{"architect", "security"},
			},
			"verifier": {
				Output:    "code/.verified",
				DependsOn: []string{"implementer", "qa"},
			},
		},
	}
}

// GetAgentNames returns sorted list of agent names
func (c *Config) GetAgentNames() []string {
	names := make([]string, 0, len(c.Agents))
	for name := range c.Agents {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetDependencies returns the dependencies for an agent
func (c *Config) GetDependencies(agentName string) []string {
	if agent, ok := c.Agents[agentName]; ok {
		return agent.DependsOn
	}
	return nil
}
