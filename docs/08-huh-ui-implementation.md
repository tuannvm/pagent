# Huh TUI Implementation

**Status**: Planned
**Depends On**: [07-huh-ui-plan.md](./07-huh-ui-plan.md)
**Library**: [charmbracelet/huh](https://github.com/charmbracelet/huh) v2

---

## Architecture

```
internal/
├── cmd/
│   ├── cli.go           # Add "ui" case to command switch
│   ├── ui.go            # NEW: ui command entry point
│   └── ...
└── tui/                  # NEW: TUI components
    ├── dashboard.go      # Main single-screen form
    ├── theme.go          # Gum-like styling
    ├── fields.go         # Custom field helpers
    └── files.go          # File discovery utilities
```

## Implementation Tasks

### Task 1: Add huh Dependency

```bash
go get github.com/charmbracelet/huh/v2@latest
go get github.com/charmbracelet/lipgloss@latest
go get golang.org/x/term@latest
```

### Task 2: Create TUI Package

#### `internal/tui/theme.go`

```go
package tui

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// Colors matching gum's aesthetic
var (
	ColorPrimary   = lipgloss.Color("212") // Bright magenta
	ColorSecondary = lipgloss.Color("39")  // Cyan
	ColorMuted     = lipgloss.Color("241") // Gray
	ColorSuccess   = lipgloss.Color("42")  // Green
)

// PagentTheme returns a gum-inspired theme
func PagentTheme() *huh.Theme {
	t := huh.ThemeCharm()

	// Title styling
	t.Focused.Title = t.Focused.Title.
		Foreground(ColorPrimary).
		Bold(true)

	// Selected option
	t.Focused.SelectedOption = t.Focused.SelectedOption.
		Foreground(ColorSuccess)

	// Description text
	t.Focused.Description = t.Focused.Description.
		Foreground(ColorMuted)

	return t
}

// HeaderStyle returns styled header for the dashboard
func HeaderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(ColorMuted).
		Padding(0, 1).
		MarginBottom(1)
}
```

#### `internal/tui/files.go`

```go
package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DiscoverInputFiles finds potential input files in the current directory
func DiscoverInputFiles() []string {
	patterns := []string{
		"*.md",
		"*.yaml",
		"*.yml",
		"prd*",
		"PRD*",
		"requirements*",
	}

	fileSet := make(map[string]os.FileInfo)

	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		for _, m := range matches {
			if info, err := os.Stat(m); err == nil && !info.IsDir() {
				fileSet[m] = info
			}
		}
	}

	// Also check common subdirectories
	subdirs := []string{"docs", "examples", "specs"}
	for _, dir := range subdirs {
		for _, pattern := range patterns {
			matches, _ := filepath.Glob(filepath.Join(dir, pattern))
			for _, m := range matches {
				if info, err := os.Stat(m); err == nil && !info.IsDir() {
					fileSet[m] = info
				}
			}
		}
	}

	// Convert to slice and sort by modification time (recent first)
	type fileWithTime struct {
		path    string
		modTime int64
	}
	var files []fileWithTime
	for path, info := range fileSet {
		files = append(files, fileWithTime{path, info.ModTime().Unix()})
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime > files[j].modTime
	})

	// Return paths only, limit to 10
	result := make([]string, 0, 10)
	for i, f := range files {
		if i >= 10 {
			break
		}
		result = append(result, f.path)
	}
	return result
}

// IsMarkdownOrYAML checks if file has supported extension
func IsMarkdownOrYAML(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".md" || ext == ".yaml" || ext == ".yml"
}
```

#### `internal/tui/dashboard.go`

```go
package tui

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/tuannvm/pagent/internal/config"
	"golang.org/x/term"
)

// DashboardResult contains the user's selections
type DashboardResult struct {
	InputPath    string
	Agents       []string
	AllAgents    bool
	Persona      string
	OutputDir    string
	Sequential   bool
	ResumeMode   string // "normal", "resume", "force"
	Architecture string // "config", "stateless", "database"
	Timeout      int
	ConfigPath   string
	Verbosity    string // "normal", "verbose", "quiet"
	Cancelled    bool
}

// DashboardOptions configures the dashboard
type DashboardOptions struct {
	PrefilledInput string
	Config         *config.Config
	Accessible     bool
}

// RunDashboard displays the interactive single-screen form
func RunDashboard(opts DashboardOptions) (*DashboardResult, error) {
	cfg := opts.Config
	if cfg == nil {
		cfg = config.Default()
	}

	// Auto-enable accessible mode for non-terminals
	accessible := opts.Accessible || !isTerminal()

	// Initialize result with defaults from config
	result := &DashboardResult{
		InputPath:    opts.PrefilledInput,
		AllAgents:    true,
		Persona:      cfg.Persona,
		OutputDir:    cfg.OutputDir,
		Sequential:   false,
		ResumeMode:   "normal",
		Architecture: "config",
		Timeout:      cfg.Timeout,
		ConfigPath:   "",
		Verbosity:    "normal",
	}

	// Discover input files if not pre-filled
	var inputOptions []huh.Option[string]
	if result.InputPath == "" {
		files := DiscoverInputFiles()
		for _, f := range files {
			inputOptions = append(inputOptions, huh.NewOption(f, f))
		}
		inputOptions = append(inputOptions, huh.NewOption("Enter path manually...", "__manual__"))
	}

	// Build agent options from config
	agentNames := cfg.GetAgentNames()
	var agentOptions []huh.Option[string]
	for _, name := range agentNames {
		agentOptions = append(agentOptions, huh.NewOption(name, name).Selected(true))
	}

	// Track agent selection mode
	var agentMode string = "all"

	// Build the form
	var formGroups []*huh.Group

	// === Main Options Group ===
	mainFields := []huh.Field{}

	// Input field - select or text based on discovery
	if len(inputOptions) > 0 && result.InputPath == "" {
		mainFields = append(mainFields,
			huh.NewSelect[string]().
				Title("Input").
				Description("Select PRD or input file").
				Options(inputOptions...).
				Value(&result.InputPath),
		)
	} else {
		mainFields = append(mainFields,
			huh.NewInput().
				Title("Input").
				Description("Path to PRD or input file").
				Placeholder("./prd.md").
				Value(&result.InputPath),
		)
	}

	// Agent selection mode
	mainFields = append(mainFields,
		huh.NewSelect[string]().
			Title("Agents").
			Options(
				huh.NewOption("All agents", "all"),
				huh.NewOption("Select specific agents...", "select"),
			).
			Value(&agentMode),
	)

	// Persona
	mainFields = append(mainFields,
		huh.NewSelect[string]().
			Title("Persona").
			Description("Implementation style").
			Options(
				huh.NewOption("Minimal - MVP focus", "minimal"),
				huh.NewOption("Balanced - Standard implementation", "balanced"),
				huh.NewOption("Production - Enterprise-ready", "production"),
			).
			Value(&result.Persona),
	)

	// Output directory
	mainFields = append(mainFields,
		huh.NewInput().
			Title("Output").
			Description("Output directory for generated files").
			Placeholder("./outputs").
			Value(&result.OutputDir),
	)

	formGroups = append(formGroups, huh.NewGroup(mainFields...))

	// === Advanced Options Group ===
	advancedFields := []huh.Field{
		huh.NewSelect[string]().
			Title("Execution").
			Options(
				huh.NewOption("Parallel (faster)", "parallel"),
				huh.NewOption("Sequential (dependency order)", "sequential"),
			).
			Value(boolToString(&result.Sequential, "sequential", "parallel")),

		huh.NewSelect[string]().
			Title("Resume Mode").
			Options(
				huh.NewOption("Normal - regenerate all", "normal"),
				huh.NewOption("Resume - skip up-to-date", "resume"),
				huh.NewOption("Force - ignore existing", "force"),
			).
			Value(&result.ResumeMode),

		huh.NewSelect[string]().
			Title("Architecture").
			Options(
				huh.NewOption("From config", "config"),
				huh.NewOption("Stateless", "stateless"),
				huh.NewOption("Database-backed", "database"),
			).
			Value(&result.Architecture),

		huh.NewInput().
			Title("Timeout").
			Description("Seconds per agent (0 = unlimited)").
			Placeholder("0").
			Value(intToString(&result.Timeout)),

		huh.NewInput().
			Title("Config").
			Description("Custom config file path (optional)").
			Placeholder(".pagent/config.yaml").
			Value(&result.ConfigPath),

		huh.NewSelect[string]().
			Title("Verbosity").
			Options(
				huh.NewOption("Normal", "normal"),
				huh.NewOption("Verbose", "verbose"),
				huh.NewOption("Quiet", "quiet"),
			).
			Value(&result.Verbosity),
	}

	formGroups = append(formGroups, huh.NewGroup(advancedFields...).
		Title("Advanced Options").
		Description("Press Enter to skip with defaults"))

	// === Confirmation Group ===
	var confirmed bool
	formGroups = append(formGroups, huh.NewGroup(
		huh.NewConfirm().
			Title("Run agents?").
			Affirmative("Run").
			Negative("Cancel").
			Value(&confirmed),
	))

	// Create and run the form
	form := huh.NewForm(formGroups...).
		WithTheme(PagentTheme()).
		WithAccessible(accessible)

	err := form.Run()
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			result.Cancelled = true
			return result, nil
		}
		return nil, fmt.Errorf("form error: %w", err)
	}

	if !confirmed {
		result.Cancelled = true
		return result, nil
	}

	// Handle manual input if selected
	if result.InputPath == "__manual__" {
		var manualPath string
		manualForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Enter input file path").
					Value(&manualPath),
			),
		).WithTheme(PagentTheme()).WithAccessible(accessible)

		if err := manualForm.Run(); err != nil {
			return nil, err
		}
		result.InputPath = manualPath
	}

	// Handle agent selection if "select" mode chosen
	if agentMode == "select" {
		result.AllAgents = false
		agentForm := huh.NewForm(
			huh.NewGroup(
				huh.NewMultiSelect[string]().
					Title("Select agents to run").
					Options(agentOptions...).
					Value(&result.Agents),
			),
		).WithTheme(PagentTheme()).WithAccessible(accessible)

		if err := agentForm.Run(); err != nil {
			return nil, err
		}
	} else {
		result.AllAgents = true
		result.Agents = agentNames
	}

	return result, nil
}

// isTerminal checks if stdout is a terminal
func isTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// Helper to convert bool pointer to string value for select
func boolToString(b *bool, trueVal, falseVal string) *string {
	s := falseVal
	if *b {
		s = trueVal
	}
	return &s
}

// Helper to convert int to string for input
func intToString(i *int) *string {
	s := fmt.Sprintf("%d", *i)
	return &s
}
```

### Task 3: Add UI Command

#### `internal/cmd/ui.go`

```go
package cmd

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/tuannvm/pagent/internal/config"
	"github.com/tuannvm/pagent/internal/tui"
)

func uiMain(args []string) error {
	fs := flag.NewFlagSet("ui", flag.ContinueOnError)

	var accessible bool
	fs.BoolVar(&accessible, "accessible", false, "enable accessible mode for screen readers")

	fs.Usage = func() {
		fmt.Print(`Usage: pagent ui [input] [flags]

Launch interactive dashboard for running agents.

Arguments:
  [input]    Optional: pre-fill input file path

Flags:
  --accessible    Enable accessible mode for screen readers

Examples:
  pagent ui
  pagent ui ./prd.md
  pagent ui --accessible
`)
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Load config
	cfg, err := config.Load("")
	if err != nil {
		cfg = config.Default()
	}

	// Pre-fill input if provided as argument
	prefilledInput := ""
	if fs.NArg() > 0 {
		prefilledInput = fs.Arg(0)
	}

	// Run the dashboard
	result, err := tui.RunDashboard(tui.DashboardOptions{
		PrefilledInput: prefilledInput,
		Config:         cfg,
		Accessible:     accessible,
	})
	if err != nil {
		return err
	}

	if result.Cancelled {
		logInfo("Cancelled")
		return nil
	}

	// Validate input
	if result.InputPath == "" {
		return fmt.Errorf("no input file specified")
	}

	// Build args for runMain
	runArgs := []string{result.InputPath}

	if !result.AllAgents && len(result.Agents) > 0 {
		runArgs = append(runArgs, "-a", strings.Join(result.Agents, ","))
	}

	if result.Persona != "" {
		runArgs = append(runArgs, "-p", result.Persona)
	}

	if result.OutputDir != "" && result.OutputDir != "./outputs" {
		runArgs = append(runArgs, "-o", result.OutputDir)
	}

	if result.Sequential {
		runArgs = append(runArgs, "-s")
	}

	switch result.ResumeMode {
	case "resume":
		runArgs = append(runArgs, "-r")
	case "force":
		runArgs = append(runArgs, "-f")
	}

	switch result.Architecture {
	case "stateless":
		runArgs = append(runArgs, "--stateless")
	case "database":
		runArgs = append(runArgs, "--no-stateless")
	}

	if result.Timeout > 0 {
		runArgs = append(runArgs, "-t", fmt.Sprintf("%d", result.Timeout))
	}

	if result.ConfigPath != "" {
		runArgs = append(runArgs, "-c", result.ConfigPath)
	}

	switch result.Verbosity {
	case "verbose":
		runArgs = append(runArgs, "-v")
	case "quiet":
		runArgs = append(runArgs, "-q")
	}

	// Display what we're about to run
	logInfo("Running: pagent run %s", strings.Join(runArgs, " "))
	logInfo("")

	// Execute run command
	return runMain(runArgs)
}
```

### Task 4: Update CLI Dispatcher

#### `internal/cmd/cli.go` (add to switch)

```go
switch cmd {
case "run":
    return runMain(os.Args[2:])
case "ui":                           // ADD THIS
    return uiMain(os.Args[2:])       // ADD THIS
case "init":
    return initMain(os.Args[2:])
// ... rest of cases
}
```

Update `printUsage()`:

```go
func printUsage() {
	fmt.Print(`Pagent - Orchestrate specialist agents from PRD

Usage:
  pagent <command> [flags] [args]

Commands:
  run <input>       Run specialist agents on input files
  ui [input]        Interactive dashboard for running agents   // ADD THIS
  init              Initialize pagent configuration
  status            Check status of running agents
  logs <agent>      View agent conversation history
  message <agent>   Send a message to an agent
  stop [agent]      Stop running agents
  agents            Manage agent definitions
  version           Print version information
  help              Show this help

Examples:
  pagent run ./prd.md
  pagent ui                           // ADD THIS
  pagent ui ./prd.md                  // ADD THIS
  pagent run ./prd.md -a architect,qa -s
  pagent init
  pagent status

Run 'pagent <command> -h' for command-specific help.
`)
}
```

## Testing

### Manual Testing

```bash
# Basic launch
pagent ui

# With pre-filled input
pagent ui ./examples/sample-prd.md

# Accessible mode
pagent ui --accessible

# Verify it calls run correctly
pagent ui  # Select options, confirm, verify output
```

### Unit Tests

```go
// internal/tui/files_test.go
package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiscoverInputFiles(t *testing.T) {
	// Test in a directory with known files
	files := DiscoverInputFiles()
	// Should return slice (may be empty)
	assert.NotNil(t, files)
}

func TestIsMarkdownOrYAML(t *testing.T) {
	assert.True(t, IsMarkdownOrYAML("test.md"))
	assert.True(t, IsMarkdownOrYAML("test.yaml"))
	assert.True(t, IsMarkdownOrYAML("test.yml"))
	assert.False(t, IsMarkdownOrYAML("test.go"))
	assert.False(t, IsMarkdownOrYAML("test.txt"))
}
```

```go
// internal/tui/theme_test.go
package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPagentTheme(t *testing.T) {
	theme := PagentTheme()
	assert.NotNil(t, theme)
}

func TestHeaderStyle(t *testing.T) {
	style := HeaderStyle()
	assert.NotNil(t, style)
}
```

## Files Changed

| File | Change |
|------|--------|
| `go.mod` | Add `huh/v2`, `lipgloss`, `x/term` |
| `internal/tui/theme.go` | NEW: gum-like styling |
| `internal/tui/files.go` | NEW: file discovery |
| `internal/tui/dashboard.go` | NEW: main form |
| `internal/cmd/ui.go` | NEW: ui command |
| `internal/cmd/cli.go` | Add `ui` case, update help |

## Known Limitations

1. **Collapsible groups**: huh v2 doesn't have native collapsible sections. Advanced options are shown as a separate form page instead.
2. **File picker**: huh's FilePicker is basic. For v1, we use a select list of discovered files + manual entry option.
3. **Real-time validation**: Input validation happens after form submission, not during typing.

## Future Enhancements

- [ ] Custom file picker with directory navigation
- [ ] Remember last used selections
- [ ] Keyboard shortcuts (e.g., Ctrl+R to run immediately)
- [ ] Live preview of command that will be executed

## References

- [huh GitHub](https://github.com/charmbracelet/huh)
- [huh Examples](https://github.com/charmbracelet/huh/tree/main/examples)
- [gum GitHub](https://github.com/charmbracelet/gum) (style inspiration)
- [lipgloss](https://github.com/charmbracelet/lipgloss) (styling library)
