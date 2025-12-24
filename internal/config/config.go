package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/tuannvm/pm-agent-workflow/internal/types"
	"gopkg.in/yaml.v3"
)

// Persona constants define implementation styles
const (
	PersonaMinimal    = "minimal"    // MVP, prototype - ship fast, iterate later
	PersonaBalanced   = "balanced"   // Pragmatic defaults - essential quality
	PersonaProduction = "production" // Enterprise-grade - comprehensive, scalable
)

// ValidPersonas lists all valid persona values
var ValidPersonas = []string{PersonaMinimal, PersonaBalanced, PersonaProduction}

// Mode constants define execution modes
const (
	ModeCreate = "create" // Create a new codebase from scratch (default)
	ModeModify = "modify" // Modify an existing codebase
)

// ValidModes lists all valid mode values
var ValidModes = []string{ModeCreate, ModeModify}

// Type aliases for backward compatibility and convenience
// These reference the canonical types in the types package
type (
	TechStack               = types.TechStack
	ArchitecturePreferences = types.ArchitecturePreferences
)

// Config represents the pm-agents configuration
type Config struct {
	OutputDir   string                  `yaml:"output_dir"`
	Timeout     int                     `yaml:"timeout"`
	Persona     string                  `yaml:"persona"`     // Implementation style: minimal, balanced, production
	Stack       TechStack               `yaml:"stack"`       // Technology stack preferences
	Preferences ArchitecturePreferences `yaml:"preferences"` // Architectural style preferences
	ResumeMode  bool                    `yaml:"-"`           // Set via CLI flag, not config file
	ForceMode   bool                    `yaml:"-"`           // Set via CLI flag, not config file
	Agents      map[string]AgentConfig  `yaml:"agents"`

	// Mode-specific configuration for existing codebase modifications
	Mode           string   `yaml:"mode"`            // "create" (default) or "modify"
	TargetCodebase string   `yaml:"target_codebase"` // Path to existing codebase (required for modify mode)
	InputFiles     []string `yaml:"input_files"`     // Multiple input files (TRD, requirements, etc.)
	SpecsOutputDir string   `yaml:"specs_output_dir"` // Directory for spec outputs (default: output_dir)

	// Post-processing options
	PostProcessing PostProcessingConfig `yaml:"post_processing"`
}

// PostProcessingConfig contains options for post-execution actions
type PostProcessingConfig struct {
	GenerateDiffSummary    bool     `yaml:"generate_diff_summary"`    // Generate git diff summary
	GeneratePRDescription  bool     `yaml:"generate_pr_description"`  // Generate PR description from changes
	ValidationCommands     []string `yaml:"validation_commands"`      // Custom commands to run after implementation
}

// IsValidPersona checks if a persona string is valid
func IsValidPersona(p string) bool {
	for _, valid := range ValidPersonas {
		if p == valid {
			return true
		}
	}
	return false
}

// IsValidMode checks if a mode string is valid
func IsValidMode(m string) bool {
	if m == "" {
		return true // Empty defaults to create
	}
	for _, valid := range ValidModes {
		if m == valid {
			return true
		}
	}
	return false
}

// IsModifyMode returns true if the config is set to modify an existing codebase
func (c *Config) IsModifyMode() bool {
	return c.Mode == ModeModify
}

// GetEffectiveCodeOutputDir returns the directory where code should be written
// In modify mode, this is the target codebase; in create mode, it's output_dir/code
func (c *Config) GetEffectiveCodeOutputDir() string {
	if c.IsModifyMode() && c.TargetCodebase != "" {
		return c.TargetCodebase
	}
	return filepath.Join(c.OutputDir, "code")
}

// GetEffectiveSpecsOutputDir returns the directory where specs should be written
func (c *Config) GetEffectiveSpecsOutputDir() string {
	if c.SpecsOutputDir != "" {
		return c.SpecsOutputDir
	}
	if c.IsModifyMode() && c.TargetCodebase != "" {
		// In modify mode, default specs to a .pm-agents subdirectory
		return filepath.Join(c.TargetCodebase, ".pm-agents", "specs")
	}
	return c.OutputDir
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
	if cfg.Persona == "" {
		cfg.Persona = PersonaBalanced
	}
	if cfg.Mode == "" {
		cfg.Mode = ModeCreate
	}

	// Validate persona
	if !IsValidPersona(cfg.Persona) {
		return nil, fmt.Errorf("invalid persona %q: must be one of %v", cfg.Persona, ValidPersonas)
	}

	// Validate mode
	if !IsValidMode(cfg.Mode) {
		return nil, fmt.Errorf("invalid mode %q: must be one of %v", cfg.Mode, ValidModes)
	}

	// Validate modify mode requirements
	if cfg.Mode == ModeModify {
		if cfg.TargetCodebase == "" {
			return nil, fmt.Errorf("target_codebase is required when mode is %q", ModeModify)
		}
		// Verify target codebase exists
		if _, err := os.Stat(cfg.TargetCodebase); os.IsNotExist(err) {
			return nil, fmt.Errorf("target_codebase %q does not exist", cfg.TargetCodebase)
		}
	}

	// Apply default agents if none specified
	if len(cfg.Agents) == 0 {
		cfg.Agents = Default().Agents
	}

	// Apply default stack if not specified
	if cfg.Stack.Cloud == "" {
		cfg.Stack = DefaultStack()
	}

	// Apply default preferences where not specified
	defaultPrefs := DefaultPreferences()
	if cfg.Preferences.Language == "" {
		cfg.Preferences.Language = defaultPrefs.Language
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

// DefaultPreferences returns the default architecture preferences
// These are neutral defaults that work for most projects
func DefaultPreferences() ArchitecturePreferences {
	return types.DefaultPreferences()
}

// DefaultStack returns the default technology stack preferences
// These are widely-used, well-documented technologies
func DefaultStack() TechStack {
	return types.DefaultStack()
}

// Default returns the default configuration
// Prompts are loaded from embedded templates (internal/prompt/templates/)
// Users can override by specifying prompt_file or inline prompt in config
func Default() *Config {
	return &Config{
		OutputDir:   "./outputs",
		Timeout:     0,               // 0 = no timeout (poll until completion). Set via --timeout for safety net.
		Persona:     PersonaBalanced, // Default to pragmatic middle-ground
		Mode:        ModeCreate,      // Default to creating new codebase
		Stack:       DefaultStack(),
		Preferences: DefaultPreferences(),
		PostProcessing: PostProcessingConfig{
			GenerateDiffSummary:   false,
			GeneratePRDescription: false,
		},
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
