package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	verbose bool
	quiet   bool
	version = "dev"
)

var rootCmd = &cobra.Command{
	Use:   "pagent",
	Short: "PM Agent Workflow - Orchestrate specialist agents from PRD",
	Long: `PM Agent Workflow is a CLI tool that spawns specialist agents
(Design, Tech, QA, Security, Infra) to transform a PRD into
actionable deliverables.

Example:
  pagent run ./prd.md
  pagent run ./prd.md --agents design,tech
  pagent status
  pagent message tech "Focus on REST API"`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("pagent version %s\n", version)
	},
}

// SetVersion sets the version string
func SetVersion(v string) {
	version = v
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "quiet output (errors only)")
	rootCmd.AddCommand(versionCmd)
}

func logInfo(format string, args ...interface{}) {
	if !quiet {
		_, _ = fmt.Fprintf(os.Stdout, format+"\n", args...)
	}
}

func logVerbose(format string, args ...interface{}) {
	if verbose && !quiet {
		_, _ = fmt.Fprintf(os.Stdout, "[DEBUG] "+format+"\n", args...)
	}
}

func logError(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
}
