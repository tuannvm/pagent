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

// DashboardResult contains the user's selections from the dashboard
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

	// === ALL ON ONE PAGE ===
	agentNames := cfg.GetAgentNames()
	result.Agents = agentNames // Default: all agents

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
	if result.InputPath != "" {
		found := false
		for _, opt := range inputOptions {
			if opt.Value == result.InputPath {
				found = true
				break
			}
		}
		if !found {
			inputOptions = append([]huh.Option[string]{huh.NewOption(result.InputPath, result.InputPath)}, inputOptions...)
		}
	}

	// Track selections
	var action string = "run"
	var executionMode string = "parallel"
	var timeoutStr string = strconv.Itoa(result.Timeout)

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
				Value(&result.InputPath),
		)

		mainFields = append(mainFields,
			huh.NewSelect[string]().
				Title("Persona").
				Options(
					huh.NewOption("Minimal - MVP focus", "minimal"),
					huh.NewOption("Balanced - Standard", "balanced"),
					huh.NewOption("Production - Enterprise", "production"),
				).
				Value(&result.Persona),
		)

		mainFields = append(mainFields,
			huh.NewInput().
				Title("Output").
				Placeholder("./outputs").
				Value(&result.OutputDir),
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
				result.Cancelled = true
				return result, nil
			}
			return nil, fmt.Errorf("form error: %w", err)
		}

		// Handle browse
		if result.InputPath == "__browse__" {
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
						Value(&result.InputPath),
				).Title("Pagent"),
			).WithTheme(PagentTheme()).WithAccessible(accessible)

			if err := browseForm.Run(); err != nil {
				if errors.Is(err, huh.ErrUserAborted) {
					result.InputPath = "" // Reset and go back to main
					continue
				}
				return nil, err
			}
			// Add the browsed file to options and continue
			if result.InputPath != "" {
				inputOptions = append([]huh.Option[string]{huh.NewOption(result.InputPath, result.InputPath)}, inputOptions...)
			}
			continue
		}

		// Validate input
		if result.InputPath == "" {
			continue
		}

		if action == "cancel" {
			result.Cancelled = true
			return result, nil
		}

		if action == "run" {
			break
		}

		// action == "advanced"
		var agentOptions []huh.Option[string]
		for _, name := range agentNames {
			selected := false
			for _, a := range result.Agents {
				if a == name {
					selected = true
					break
				}
			}
			agentOptions = append(agentOptions, huh.NewOption(name, name).Selected(selected))
		}

		advancedForm := huh.NewForm(
			huh.NewGroup(
				huh.NewMultiSelect[string]().
					Title("Agents").
					Description("Space=toggle").
					Options(agentOptions...).
					Value(&result.Agents),

				huh.NewSelect[string]().
					Title("Execution").
					Options(
						huh.NewOption("Parallel", "parallel"),
						huh.NewOption("Sequential", "sequential"),
					).
					Value(&executionMode),

				huh.NewSelect[string]().
					Title("Resume").
					Options(
						huh.NewOption("Normal", "normal"),
						huh.NewOption("Resume", "resume"),
						huh.NewOption("Force", "force"),
					).
					Value(&result.ResumeMode),

				huh.NewSelect[string]().
					Title("Architecture").
					Options(
						huh.NewOption("From config", "config"),
						huh.NewOption("Stateless", "stateless"),
						huh.NewOption("Database", "database"),
					).
					Value(&result.Architecture),

				huh.NewInput().
					Title("Timeout (sec)").
					Placeholder("0").
					Value(&timeoutStr),

				huh.NewInput().
					Title("Config file").
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
		result.Timeout = t
	}

	// Map execution mode to boolean
	result.Sequential = (executionMode == "sequential")

	// Determine if all agents are selected
	result.AllAgents = len(result.Agents) == len(agentNames)

	return result, nil
}

// isTerminal checks if stdout is a terminal
func isTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}
