package tui

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// Colors matching gum's aesthetic
var (
	ColorPrimary   = lipgloss.Color("6")   // Teal
	ColorSecondary = lipgloss.Color("14")  // Bright cyan
	ColorMuted     = lipgloss.Color("241") // Gray
	ColorSuccess   = lipgloss.Color("42")  // Green
)

// Banner returns the styled app banner
func Banner() string {
	logo := ` ██████╗  █████╗  ██████╗ ███████╗███╗   ██╗████████╗
 ██╔══██╗██╔══██╗██╔════╝ ██╔════╝████╗  ██║╚══██╔══╝
 ██████╔╝███████║██║  ███╗█████╗  ██╔██╗ ██║   ██║
 ██╔═══╝ ██╔══██║██║   ██║██╔══╝  ██║╚██╗██║   ██║
 ██║     ██║  ██║╚██████╔╝███████╗██║ ╚████║   ██║
 ╚═╝     ╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═╝  ╚═══╝   ╚═╝
`
	logoStyle := lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true)

	tagline := lipgloss.NewStyle().
		Foreground(ColorMuted).
		Italic(true).
		Render(" From idea to implementation, orchestrated.")

	return logoStyle.Render(logo) + "\n" + tagline + "\n"
}

// PagentTheme returns a gum-inspired theme for forms
func PagentTheme() *huh.Theme {
	t := huh.ThemeCharm()

	// Title styling - bright magenta, bold
	t.Focused.Title = t.Focused.Title.
		Foreground(ColorPrimary).
		Bold(true)

	// Selected option - green
	t.Focused.SelectedOption = t.Focused.SelectedOption.
		Foreground(ColorSuccess)

	// Description text - muted gray
	t.Focused.Description = t.Focused.Description.
		Foreground(ColorMuted)

	// Blurred state - more subtle
	t.Blurred.Title = t.Blurred.Title.
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

// TitleStyle returns style for section titles
func TitleStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary)
}

// SuccessStyle returns style for success messages
func SuccessStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(ColorSuccess)
}

// MutedStyle returns style for muted/secondary text
func MutedStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(ColorMuted)
}
