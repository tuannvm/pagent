package tui

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/tuannvm/pagent/internal/config"
	"golang.org/x/term"
)

// DashboardOptions configures the dashboard
type DashboardOptions struct {
	PrefilledInput string
	Config         *config.Config
	Accessible     bool
}

// RunDashboard displays the interactive single-screen form.
// Returns nil, nil if the user cancels.
func RunDashboard(dashOpts DashboardOptions) (*config.RunOptions, error) {
	cfg := dashOpts.Config
	if cfg == nil {
		cfg = config.Default()
	}

	// Auto-enable accessible mode for non-terminals
	accessible := dashOpts.Accessible || !isTerminal()

	// Initialize result with defaults from config
	opts := config.DefaultRunOptions(cfg)
	opts.InputPath = dashOpts.PrefilledInput

	// Get all agent names for multi-select
	agentNames := cfg.GetAgentNames()
	opts.Agents = agentNames // Default: all agents

	// Build input options
	discoveredFiles := DiscoverInputFiles()
	discoveredFolders := DiscoverInputFolders()
	var inputOptions []huh.Option[string]

	// Add folders first
	for _, f := range discoveredFolders {
		inputOptions = append(inputOptions, huh.NewOption("📁 "+f+"/", f))
	}
	// Add files
	for _, f := range discoveredFiles {
		inputOptions = append(inputOptions, huh.NewOption(f, f))
	}
	// Add browse option
	inputOptions = append(inputOptions, huh.NewOption("🔍 Browse...", "__browse__"))

	// If input was pre-filled, add it as first option if not already present
	if opts.InputPath != "" {
		found := false
		for _, opt := range inputOptions {
			if opt.Value == opts.InputPath {
				found = true
				break
			}
		}
		if !found {
			inputOptions = append([]huh.Option[string]{huh.NewOption(opts.InputPath, opts.InputPath)}, inputOptions...)
		}
	}

	// Track selections using local variables
	var action string
	var executionMode = config.ExecutionParallel
	var timeoutStr = strconv.Itoa(opts.Timeout)

	// Build persona options from shared definitions
	var personaOpts []huh.Option[string]
	for _, o := range config.PersonaOptions {
		personaOpts = append(personaOpts, huh.NewOption(o.Label+" - "+o.Description, o.Value))
	}

	// === Main loop ===
	for {
		action = "run" // Reset

		// Print banner
		fmt.Print("\033[H\033[2J") // Clear screen
		fmt.Println(Banner())

		// Build main form fields
		var mainFields []huh.Field

		// Input selector
		mainFields = append(mainFields,
			huh.NewSelect[string]().
				Title("Input").
				Description("Select PRD or input file").
				Options(inputOptions...).
				Height(8).
				Value(&opts.InputPath),
		)

		// Persona using shared options
		mainFields = append(mainFields,
			huh.NewSelect[string]().
				Title("Persona").
				Options(personaOpts...).
				Value(&opts.Persona),
		)

		mainFields = append(mainFields,
			huh.NewInput().
				Title("Output").
				Placeholder("./outputs").
				Value(&opts.OutputDir),
		)

		mainFields = append(mainFields,
			huh.NewSelect[string]().
				Title("Action").
				Description("Shift+Tab go back").
				Options(
					huh.NewOption("▶ Run", "run"),
					huh.NewOption("⚙ Advanced...", "advanced"),
					huh.NewOption("✕ Cancel", "cancel"),
				).
				Value(&action),
		)

		mainForm := huh.NewForm(
			huh.NewGroup(mainFields...),
		).WithTheme(PagentTheme()).WithAccessible(accessible)

		if err := mainForm.Run(); err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				return nil, nil // Cancelled
			}
			return nil, fmt.Errorf("form error: %w", err)
		}

		// Handle browse
		if opts.InputPath == "__browse__" {
			browseForm := huh.NewForm(
				huh.NewGroup(
					huh.NewFilePicker().
						Title("Browse").
						Description("Enter=open/select • .=hidden").
						Picking(true).
						DirAllowed(false).
						FileAllowed(true).
						CurrentDirectory(".").
						ShowHidden(false).
						ShowSize(true).
						ShowPermissions(false).
						Height(15).
						Value(&opts.InputPath),
				).Title("Pagent"),
			).WithTheme(PagentTheme()).WithAccessible(accessible)

			if err := browseForm.Run(); err != nil {
				if errors.Is(err, huh.ErrUserAborted) {
					opts.InputPath = "" // Reset and go back to main
					continue
				}
				return nil, err
			}
			// Add the browsed file to options and continue
			if opts.InputPath != "" {
				inputOptions = append([]huh.Option[string]{huh.NewOption(opts.InputPath, opts.InputPath)}, inputOptions...)
			}
			continue
		}

		// Validate input
		if opts.InputPath == "" {
			continue
		}

		if action == "cancel" {
			return nil, nil // Cancelled
		}

		if action == "run" {
			break
		}

		// action == "advanced"
		var agentOptions []huh.Option[string]
		for _, name := range agentNames {
			selected := false
			for _, a := range opts.Agents {
				if a == name {
					selected = true
					break
				}
			}
			agentOptions = append(agentOptions, huh.NewOption(name, name).Selected(selected))
		}

		// Build options from shared definitions
		var execOpts []huh.Option[string]
		for _, o := range config.ExecutionOptions {
			execOpts = append(execOpts, huh.NewOption(o.Label, o.Value))
		}

		var resumeOpts []huh.Option[string]
		for _, o := range config.ResumeModeOptions {
			resumeOpts = append(resumeOpts, huh.NewOption(o.Label, o.Value))
		}

		var archOpts []huh.Option[string]
		for _, o := range config.ArchitectureOptions {
			archOpts = append(archOpts, huh.NewOption(o.Label, o.Value))
		}

		var verbOpts []huh.Option[string]
		for _, o := range config.VerbosityOptions {
			verbOpts = append(verbOpts, huh.NewOption(o.Label, o.Value))
		}

		advancedForm := huh.NewForm(
			huh.NewGroup(
				huh.NewMultiSelect[string]().
					Title("Agents").
					Description("Space=toggle").
					Options(agentOptions...).
					Value(&opts.Agents),

				huh.NewSelect[string]().
					Title("Execution").
					Options(execOpts...).
					Value(&executionMode),

				huh.NewSelect[string]().
					Title("Resume").
					Options(resumeOpts...).
					Value(&opts.ResumeMode),

				huh.NewSelect[string]().
					Title("Architecture").
					Options(archOpts...).
					Value(&opts.Architecture),

				huh.NewInput().
					Title("Timeout (sec)").
					Placeholder("0").
					Value(&timeoutStr),

				huh.NewInput().
					Title("Config file").
					Placeholder(".pagent/config.yaml").
					Value(&opts.ConfigPath),

				huh.NewSelect[string]().
					Title("Verbosity").
					Options(verbOpts...).
					Value(&opts.Verbosity),
			).Title("Advanced").Description("Esc=back"),
		).WithTheme(PagentTheme()).WithAccessible(accessible)

		if err := advancedForm.Run(); err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				continue // Back to main
			}
			return nil, fmt.Errorf("form error: %w", err)
		}
	}

	// Parse timeout
	if t, err := strconv.Atoi(timeoutStr); err == nil {
		opts.Timeout = t
	}

	// Map execution mode to boolean
	opts.Sequential = (executionMode == config.ExecutionSequential)

	return &opts, nil
}

// isTerminal checks if stdout is a terminal
func isTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}
