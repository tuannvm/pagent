package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tuannvm/pagent/internal/config"
	"gopkg.in/yaml.v3"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize pagent configuration",
	Long: `Create a .pagent/config.yaml file in the current directory
with default agent configurations.

You can customize the prompts and settings after initialization.

Example:
  pagent init`,
	RunE: initCommand,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func initCommand(cmd *cobra.Command, args []string) error {
	configDir := ".pagent"
	configFile := filepath.Join(configDir, "config.yaml")

	// Check if already exists
	if _, err := os.Stat(configFile); err == nil {
		return fmt.Errorf("config file already exists: %s", configFile)
	}

	// Create directory
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Get default config
	cfg := config.Default()

	// Marshal to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Add header comment
	header := `# Pagent Configuration
# Customize agent prompts and settings below
# Documentation: https://github.com/tuannvm/pagent

`

	// Write file
	if err := os.WriteFile(configFile, []byte(header+string(data)), 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	logInfo("Created %s", configFile)
	logInfo("")
	logInfo("You can now customize agent prompts and run:")
	logInfo("  pagent run ./prd.md")

	return nil
}
