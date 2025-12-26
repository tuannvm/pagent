// options.go provides shared option definitions for CLI and TUI.
package config

// Option represents a selectable option with value and label
type Option struct {
	Value       string
	Label       string
	Description string
}

// RunOptions contains all parameters for running agents.
// This is the single source of truth used by both CLI and TUI.
type RunOptions struct {
	InputPath    string
	Agents       []string
	Persona      string
	OutputDir    string
	Sequential   bool
	ResumeMode   string // "normal", "resume", "force"
	Architecture string // "config", "stateless", "database"
	Timeout      int
	ConfigPath   string
	Verbosity    string // "normal", "verbose", "quiet"
}

// Shared option definitions - SINGLE SOURCE OF TRUTH
var PersonaOptions = []Option{
	{Value: PersonaMinimal, Label: "Minimal", Description: "MVP focus"},
	{Value: PersonaBalanced, Label: "Balanced", Description: "Standard"},
	{Value: PersonaProduction, Label: "Production", Description: "Enterprise"},
}

// VerbosityNormal, VerbosityVerbose, VerbosityQuiet are verbosity constants
const (
	VerbosityNormal  = "normal"
	VerbosityVerbose = "verbose"
	VerbosityQuiet   = "quiet"
)

var VerbosityOptions = []Option{
	{Value: VerbosityNormal, Label: "Normal", Description: "Standard output"},
	{Value: VerbosityVerbose, Label: "Verbose", Description: "Debug info"},
	{Value: VerbosityQuiet, Label: "Quiet", Description: "Errors only"},
}

// ExecutionParallel, ExecutionSequential are execution mode constants
const (
	ExecutionParallel   = "parallel"
	ExecutionSequential = "sequential"
)

var ExecutionOptions = []Option{
	{Value: ExecutionParallel, Label: "Parallel", Description: "Faster, respects dependencies"},
	{Value: ExecutionSequential, Label: "Sequential", Description: "One at a time"},
}

// ResumeModeNormal, ResumeModeResume, ResumeModeForce are resume mode constants
const (
	ResumeModeNormal = "normal"
	ResumeModeResume = "resume"
	ResumeModeForce  = "force"
)

var ResumeModeOptions = []Option{
	{Value: ResumeModeNormal, Label: "Normal", Description: "Regenerate all"},
	{Value: ResumeModeResume, Label: "Resume", Description: "Skip existing"},
	{Value: ResumeModeForce, Label: "Force", Description: "Overwrite all"},
}

// ArchitectureConfig, ArchitectureStateless, ArchitectureDatabase are architecture constants
const (
	ArchitectureConfig    = "config"
	ArchitectureStateless = "stateless"
	ArchitectureDatabase  = "database"
)

var ArchitectureOptions = []Option{
	{Value: ArchitectureConfig, Label: "From config", Description: "Use config setting"},
	{Value: ArchitectureStateless, Label: "Stateless", Description: "Prefer stateless"},
	{Value: ArchitectureDatabase, Label: "Database", Description: "DB-backed"},
}

// DefaultRunOptions returns RunOptions with sensible defaults from config
func DefaultRunOptions(cfg *Config) RunOptions {
	if cfg == nil {
		cfg = Default()
	}
	return RunOptions{
		Persona:      cfg.Persona,
		OutputDir:    cfg.OutputDir,
		Timeout:      cfg.Timeout,
		ResumeMode:   ResumeModeNormal,
		Architecture: ArchitectureConfig,
		Verbosity:    VerbosityNormal,
		Sequential:   false,
	}
}

// IsVerbose returns true if verbosity is set to verbose
func (o RunOptions) IsVerbose() bool {
	return o.Verbosity == VerbosityVerbose
}

// IsQuiet returns true if verbosity is set to quiet
func (o RunOptions) IsQuiet() bool {
	return o.Verbosity == VerbosityQuiet
}
