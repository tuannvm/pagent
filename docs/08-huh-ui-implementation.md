# Huh TUI Implementation

**Status**: Planned
**Depends On**: [07-huh-ui-plan.md](./07-huh-ui-plan.md)
**Library**: [charmbracelet/huh](https://github.com/charmbracelet/huh) v2

---

## Architecture

```
internal/
├── cmd/
│   ├── cli.go           # Main dispatcher (add --accessible flag)
│   ├── init.go          # init command (add huh form)
│   ├── run.go           # run command (add huh prompts)
│   └── ...
└── tui/                  # NEW: TUI components
    ├── forms.go          # Reusable form builders
    ├── theme.go          # Custom theme
    └── prompts.go        # Confirmation dialogs
```

## Implementation Tasks

### Task 1: Add huh Dependency

```bash
go get github.com/charmbracelet/huh/v2@latest
```

### Task 2: Create TUI Package

#### `internal/tui/theme.go`

```go
package tui

import (
	"github.com/charmbracelet/huh/v2"
	"github.com/charmbracelet/lipgloss"
)

// PagentTheme returns a custom theme for pagent forms
func PagentTheme() *huh.Theme {
	t := huh.ThemeBase()

	// Customize colors
	t.Focused.Title = t.Focused.Title.Foreground(lipgloss.Color("39"))  // Cyan
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(lipgloss.Color("42"))  // Green

	return t
}
```

#### `internal/tui/forms.go`

```go
package tui

import (
	"github.com/charmbracelet/huh/v2"
	"github.com/tuannvm/pagent/internal/config"
)

// InitForm creates the interactive init configuration form
func InitForm(cfg *config.Config) *huh.Form {
	return huh.NewForm(
		// Page 1: Basic Settings
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Implementation Persona").
				Description("How comprehensive should the output be?").
				Options(
					huh.NewOption("Minimal - MVP focus, lean output", "minimal"),
					huh.NewOption("Balanced - Standard implementation", "balanced"),
					huh.NewOption("Production - Enterprise-ready, comprehensive", "production"),
				).
				Value(&cfg.Persona),

			huh.NewSelect[string]().
				Title("Primary Language").
				Options(
					huh.NewOption("Go", "go"),
					huh.NewOption("Python", "python"),
					huh.NewOption("TypeScript", "typescript"),
					huh.NewOption("Rust", "rust"),
					huh.NewOption("Java", "java"),
				).
				Value(&cfg.Preferences.Language),
		),

		// Page 2: Stack Configuration
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Cloud Provider").
				Options(
					huh.NewOption("AWS", "aws"),
					huh.NewOption("GCP", "gcp"),
					huh.NewOption("Azure", "azure"),
					huh.NewOption("None", "none"),
				).
				Value(&cfg.Stack.Cloud),

			huh.NewSelect[string]().
				Title("Compute Platform").
				Options(
					huh.NewOption("Kubernetes", "kubernetes"),
					huh.NewOption("ECS", "ecs"),
					huh.NewOption("Lambda/Serverless", "serverless"),
					huh.NewOption("VMs", "vm"),
					huh.NewOption("GitHub Actions", "github-actions"),
					huh.NewOption("None", "none"),
				).
				Value(&cfg.Stack.Compute),

			huh.NewSelect[string]().
				Title("Database").
				Options(
					huh.NewOption("PostgreSQL", "postgres"),
					huh.NewOption("MySQL", "mysql"),
					huh.NewOption("MongoDB", "mongodb"),
					huh.NewOption("DynamoDB", "dynamodb"),
					huh.NewOption("None (Stateless)", "none"),
				).
				Value(&cfg.Stack.Database),
		),

		// Page 3: Agent Selection
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Agents to Enable").
				Description("Select which specialist agents to run").
				Options(
					huh.NewOption("Architect - System design & API specs", "architect").Selected(true),
					huh.NewOption("Implementer - Code generation", "implementer").Selected(true),
					huh.NewOption("QA - Test plans & test code", "qa").Selected(true),
					huh.NewOption("Security - Security assessment", "security").Selected(true),
					huh.NewOption("Verifier - Build & validation", "verifier").Selected(true),
				).
				Value(&cfg.EnabledAgents),
		),
	).WithTheme(PagentTheme())
}

// InputSelectForm prompts for input file when not provided
// Uses FilePicker for better UX, falls back to manual input
func InputSelectForm(recentFiles []string, accessible bool) (string, error) {
	var selected string

	options := make([]huh.Option[string], 0, len(recentFiles)+1)
	options = append(options, huh.NewOption("Browse files...", "__browse__"))
	options = append(options, huh.NewOption("Enter path manually...", "__manual__"))
	for _, f := range recentFiles {
		options = append(options, huh.NewOption(f, f))
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select input file").
				Options(options...).
				Value(&selected),
		),
	).WithTheme(PagentTheme()).
		WithAccessible(accessible)

	if err := form.Run(); err != nil {
		return "", err
	}

	// Handle special selections
	switch selected {
	case "__browse__":
		// Use FilePicker for browsing
		var filepath string
		picker := huh.NewForm(
			huh.NewGroup(
				huh.NewFilePicker().
					Title("Select input file").
					CurrentDirectory(".").
					ShowHidden(false).
					FileAllowed(true).
					DirAllowed(true).
					Value(&filepath),
			),
		).WithTheme(PagentTheme()).
			WithAccessible(accessible)

		if err := picker.Run(); err != nil {
			return "", err
		}
		return filepath, nil

	case "__manual__":
		// Manual path entry
		var path string
		input := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Input file path").
					Placeholder("./prd.md").
					Value(&path),
			),
		).WithTheme(PagentTheme()).
			WithAccessible(accessible)

		if err := input.Run(); err != nil {
			return "", err
		}
		return path, nil
	}

	return selected, nil
}
```

#### `internal/tui/prompts.go`

```go
package tui

import "github.com/charmbracelet/huh/v2"

// Confirm shows a yes/no confirmation dialog
func Confirm(title, description string, accessible bool) (bool, error) {
	var confirmed bool

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(title).
				Description(description).
				Affirmative("Yes").
				Negative("No").
				Value(&confirmed),
		),
	).WithTheme(PagentTheme()).
		WithAccessible(accessible)

	if err := form.Run(); err != nil {
		return false, err
	}

	return confirmed, nil
}

// SelectAgents prompts user to select which agents to run
func SelectAgents(available []string, accessible bool) ([]string, error) {
	var selected []string

	options := make([]huh.Option[string], len(available))
	for i, a := range available {
		options[i] = huh.NewOption(a, a).Selected(true)
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select agents to run").
				Options(options...).
				Value(&selected),
		),
	).WithTheme(PagentTheme()).
		WithAccessible(accessible)

	if err := form.Run(); err != nil {
		return nil, err
	}

	return selected, nil
}
```

### Task 3: Update `init` Command

#### `internal/cmd/init.go` (Modified)

```go
package cmd

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh/v2"
	"github.com/tuannvm/pagent/internal/config"
	"github.com/tuannvm/pagent/internal/tui"
	"gopkg.in/yaml.v3"
)

func initMain(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	parseGlobalFlags(fs)

	noInteractive := fs.Bool("no-interactive", false, "skip interactive prompts")

	if err := fs.Parse(args); err != nil {
		return err
	}

	configDir := ".pagent"
	configFile := filepath.Join(configDir, "config.yaml")

	// Check if already exists
	if _, err := os.Stat(configFile); err == nil {
		if !*noInteractive {
			overwrite, err := tui.Confirm(
				"Config already exists",
				fmt.Sprintf("Overwrite %s?", configFile),
				accessible,  // accessible mode changes rendering, not behavior
			)
			if err != nil || !overwrite {
				return fmt.Errorf("config file already exists: %s", configFile)
			}
		} else {
			return fmt.Errorf("config file already exists: %s", configFile)
		}
	}

	// Get default config
	cfg := config.Default()

	// Run interactive form if not disabled
	if !*noInteractive {
		form := tui.InitForm(cfg).
			WithAccessible(accessible)  // Set accessible mode from global flag
		if err := form.Run(); err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				logInfo("Setup cancelled")
				return nil
			}
			return fmt.Errorf("form error: %w", err)
		}
	}

	// Create directory and write config
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	header := `# Pagent Configuration
# Generated by pagent init
# Documentation: https://github.com/tuannvm/pagent

`
	if err := os.WriteFile(configFile, []byte(header+string(data)), 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	logInfo("Created %s", configFile)
	return nil
}
```

### Task 4: Update `run` Command

Add these imports to `internal/cmd/run.go`:

```go
import (
	"errors"
	"path/filepath"
	// ... existing imports ...
	"github.com/charmbracelet/huh/v2"
	"github.com/tuannvm/pagent/internal/tui"
)
```

Add to `runMain` function after flag parsing:

```go
if fs.NArg() < 1 {
	// No input provided - try interactive selection
	if !noInteractive {
		recentFiles := findRecentPRDFiles()  // helper to find .md files
		selected, err := tui.InputSelectForm(recentFiles, accessible)
		if err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				return nil
			}
			fs.Usage()
			return fmt.Errorf("missing required argument: input file or directory")
		}
		inputPath = selected
	} else {
		fs.Usage()
		return fmt.Errorf("missing required argument: input file or directory")
	}
}

// Helper function to find recent PRD files
func findRecentPRDFiles() []string {
	var files []string
	patterns := []string{"*.md", "*.yaml", "*.yml", "prd*", "PRD*"}

	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		files = append(files, matches...)
	}

	// Deduplicate and limit to 5 most recent
	seen := make(map[string]bool)
	var unique []string
	for _, f := range files {
		if !seen[f] && len(unique) < 5 {
			seen[f] = true
			unique = append(unique, f)
		}
	}
	return unique
}
```

### Task 5: Add Global Flags for TUI Control

Update `internal/cmd/cli.go`:

```go
var (
	verbose       bool
	quiet         bool
	noInteractive bool  // NEW: disable all interactive prompts
	accessible    bool  // NEW: enable accessible mode for screen readers
	version       = "dev"
)

func parseGlobalFlags(fs *flag.FlagSet) {
	fs.BoolVar(&verbose, "v", false, "verbose output")
	fs.BoolVar(&verbose, "verbose", false, "verbose output")
	fs.BoolVar(&quiet, "q", false, "quiet output (errors only)")
	fs.BoolVar(&quiet, "quiet", false, "quiet output (errors only)")
	fs.BoolVar(&noInteractive, "no-interactive", false, "disable interactive prompts")  // NEW
	fs.BoolVar(&accessible, "accessible", false, "enable accessible mode for screen readers")  // NEW
}
```

**Note on Accessible Mode:**

The `--accessible` flag enables huh's accessible mode which:
- Disables fancy TUI rendering
- Uses simple text prompts compatible with screen readers
- Works in environments where ANSI escape codes aren't supported (e.g., `TERM=dumb`)

You can also auto-detect non-interactive terminals:

```go
import "golang.org/x/term"

// isTerminal returns true if stdout is a terminal
func isTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// In command functions, auto-enable accessible mode for non-terminals:
if !isTerminal() {
	accessible = true
}
```

## Testing

### Unit Tests

```go
// internal/tui/forms_test.go
package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tuannvm/pagent/internal/config"
)

func TestInitFormCreation(t *testing.T) {
	cfg := config.Default()
	form := InitForm(cfg)

	// Form should be created without error
	assert.NotNil(t, form)
}

func TestConfirmDialog(t *testing.T) {
	// Test that Confirm returns proper types
	// Note: Can't easily test interactive forms in unit tests
	// Use integration tests for full form testing
}

func TestPagentTheme(t *testing.T) {
	theme := PagentTheme()
	assert.NotNil(t, theme)
}
```

### Integration Tests

```bash
# Test non-interactive mode
pagent init --no-interactive
pagent run ./prd.md --no-interactive

# Test interactive mode (manual)
pagent init  # Should show form
pagent run   # Should prompt for input

# Test accessible mode
pagent init --accessible
```

### Accessibility Testing

```bash
# Force accessible mode via flag
pagent init --accessible

# Force accessible mode via environment
TERM=dumb pagent init
```

## Rollout

1. **Alpha**: Add huh to `init` command only
2. **Beta**: Add to `run` command input selection
3. **GA**: Add confirmation dialogs, error recovery prompts

## Files Changed

| File | Change |
|------|--------|
| `go.mod` | Add `github.com/charmbracelet/huh/v2`, `golang.org/x/term` |
| `internal/tui/theme.go` | NEW |
| `internal/tui/forms.go` | NEW |
| `internal/tui/prompts.go` | NEW |
| `internal/cmd/cli.go` | Add `--no-interactive` and `--accessible` flags |
| `internal/cmd/init.go` | Add interactive form |
| `internal/cmd/run.go` | Add input selection prompt, `findRecentPRDFiles()` helper |

## References

- [huh GitHub](https://github.com/charmbracelet/huh)
- [huh v2 Docs](https://pkg.go.dev/github.com/charmbracelet/huh/v2)
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) (underlying TUI framework)
