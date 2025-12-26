# Huh TUI Implementation

**Status**: Implemented
**Depends On**: [07-huh-ui-plan.md](./07-huh-ui-plan.md)
**Library**: [charmbracelet/huh](https://github.com/charmbracelet/huh)

---

## Architecture

```
internal/
├── cmd/
│   ├── cli.go           # Command dispatcher with "ui" case
│   ├── run.go           # CLI run command (uses runner.Execute)
│   ├── ui.go            # UI command entry point (uses runner.Execute)
│   └── ...
├── config/
│   ├── config.go        # Configuration loading
│   └── options.go       # Shared RunOptions for CLI and TUI
├── runner/
│   ├── executor.go      # Shared execution logic
│   └── logger.go        # Logger interface and StdLogger
└── tui/
    ├── dashboard.go     # Main single-screen form
    ├── theme.go         # Styling and ASCII banner
    └── files.go         # File discovery utilities
```

### Key Design: Shared Execution Path

Both CLI and TUI use the same execution path:

```
┌─────────────┐     ┌─────────────┐
│  pagent run │     │  pagent ui  │
│   (CLI)     │     │   (TUI)     │
└──────┬──────┘     └──────┬──────┘
       │                   │
       │  config.RunOptions│
       └─────────┬─────────┘
                 ▼
        ┌────────────────┐
        │ runner.Execute │
        │  (shared)      │
        └────────────────┘
```

No translation layer - both paths build `config.RunOptions` and call `runner.Execute()` directly.

## Implementation

### Shared Options (`config/options.go`)

Single source of truth for all run options:

```go
// RunOptions contains all parameters for running agents.
// Used by both CLI and TUI.
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

// Option definitions with labels for TUI
var PersonaOptions = []Option{
    {Value: PersonaMinimal, Label: "Minimal", Description: "MVP focus"},
    {Value: PersonaBalanced, Label: "Balanced", Description: "Standard"},
    {Value: PersonaProduction, Label: "Production", Description: "Enterprise"},
}
// ... VerbosityOptions, ExecutionOptions, ResumeModeOptions, ArchitectureOptions
```

### Dashboard (`tui/dashboard.go`)

Returns `*config.RunOptions` directly:

```go
func RunDashboard(dashOpts DashboardOptions) (*config.RunOptions, error) {
    // Initialize with defaults from config
    opts := config.DefaultRunOptions(cfg)

    // Build form using shared option definitions
    for _, o := range config.PersonaOptions {
        personaOpts = append(personaOpts, huh.NewOption(o.Label+" - "+o.Description, o.Value))
    }

    // Run form...

    return &opts, nil  // Returns RunOptions directly
}
```

### UI Command (`cmd/ui.go`)

Calls runner.Execute directly - no translation:

```go
func uiMain(args []string) error {
    opts, err := tui.RunDashboard(tui.DashboardOptions{...})
    if err != nil {
        return err
    }

    if opts == nil {
        return nil // Cancelled
    }

    // Execute directly - NO TRANSLATION LAYER
    logger := runner.NewStdLogger(opts.IsVerbose(), opts.IsQuiet())
    return runner.Execute(context.Background(), *opts, logger)
}
```

### Runner (`runner/executor.go`)

Shared execution logic:

```go
func Execute(ctx context.Context, opts config.RunOptions, logger Logger) error {
    // Load config, validate, run agents...
}
```

## Form Structure

### Main Screen
- **Input**: File selector with auto-discovery + browse option
- **Persona**: Minimal / Balanced / Production
- **Output**: Directory path
- **Action**: Run / Advanced / Cancel

### Advanced Screen (optional)
- **Agents**: Multi-select from config
- **Execution**: Parallel / Sequential
- **Resume**: Normal / Resume / Force
- **Architecture**: From config / Stateless / Database
- **Timeout**: Seconds (0 = unlimited)
- **Config**: Custom config file path
- **Verbosity**: Normal / Verbose / Quiet

## Files

| File | Purpose |
|------|---------|
| `config/options.go` | Shared RunOptions and option definitions |
| `runner/executor.go` | Shared execution logic |
| `runner/logger.go` | Logger interface |
| `tui/dashboard.go` | Interactive form |
| `tui/theme.go` | Styling and banner |
| `tui/files.go` | File/folder discovery |
| `cmd/ui.go` | UI command entry point |

## Usage

```bash
# Launch interactive dashboard
pagent ui

# Pre-fill with input file
pagent ui ./prd.md

# Accessible mode for screen readers
pagent ui --accessible
```

## References

- [huh GitHub](https://github.com/charmbracelet/huh)
- [lipgloss](https://github.com/charmbracelet/lipgloss) (styling library)
