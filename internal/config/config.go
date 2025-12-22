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
	OutputDir string                 `yaml:"output_dir"`
	Timeout   int                    `yaml:"timeout"`
	Agents    map[string]AgentConfig `yaml:"agents"`
}

// AgentConfig represents a single agent's configuration
type AgentConfig struct {
	Prompt    string   `yaml:"prompt"`
	Output    string   `yaml:"output"`
	DependsOn []string `yaml:"depends_on"`
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
func Default() *Config {
	return &Config{
		OutputDir: "./outputs",
		Timeout:   300,
		Agents: map[string]AgentConfig{
			"design": {
				Prompt: `You are a Design Lead. Read the PRD at {prd_path} and create a design specification.
Write your output to {output_path}.

Include:
- UI/UX requirements
- User flows (mermaid diagrams)
- Component specifications
- Accessibility requirements
- Design system considerations

Be specific and actionable. Reference existing patterns when possible.`,
				Output:    "design-spec.md",
				DependsOn: []string{},
			},
			"tech": {
				Prompt: `You are a Tech Lead. Read the PRD at {prd_path} and any existing design docs in the outputs directory.
Write your output to {output_path}.

Include:
- Architecture decisions with rationale
- API specifications (OpenAPI format if applicable)
- Data models and database schema
- Technical constraints and trade-offs
- Integration points
- Security considerations

Be specific and actionable. Reference existing code patterns when possible.`,
				Output:    "technical-requirements.md",
				DependsOn: []string{"design"},
			},
			"qa": {
				Prompt: `You are a QA Lead. Read the PRD at {prd_path} and existing specs in the outputs directory.
Write your output to {output_path}.

Include:
- Test strategy overview
- Test cases organized by feature
- Acceptance criteria
- Edge cases and error scenarios
- Performance test requirements
- Regression test considerations

Be thorough and cover both happy paths and edge cases.`,
				Output:    "test-plan.md",
				DependsOn: []string{"tech"},
			},
			"security": {
				Prompt: `You are a Security Reviewer. Read the PRD at {prd_path} and technical requirements in the outputs directory.
Write your output to {output_path}.

Include:
- Threat model (STRIDE or similar)
- Security requirements
- Authentication/Authorization considerations
- Data protection requirements
- Risk assessment with severity levels
- Recommended mitigations

Focus on practical, actionable security guidance.`,
				Output:    "security-assessment.md",
				DependsOn: []string{"tech"},
			},
			"infra": {
				Prompt: `You are an Infrastructure Lead. Read the PRD at {prd_path} and technical requirements in the outputs directory.
Write your output to {output_path}.

Include:
- Infrastructure requirements
- Resource sizing estimates
- Deployment strategy
- Scaling considerations
- Monitoring and alerting requirements
- Disaster recovery considerations
- Cost estimates if possible

Be practical and consider both development and production environments.`,
				Output:    "infrastructure-plan.md",
				DependsOn: []string{"tech"},
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
