package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/tuannvm/pm-agent-workflow/internal/config"
)

var agentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "Manage agent definitions",
	Long:  `List and show agent definitions.`,
}

var agentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available agents",
	Long: `List all available agent types with their output files.

Example:
  pagent agents list`,
	RunE: agentsListCommand,
}

var agentsShowCmd = &cobra.Command{
	Use:   "show <agent>",
	Short: "Show agent prompt template",
	Long: `Show the prompt template for a specific agent.

Example:
  pagent agents show design
  pagent agents show tech`,
	Args: cobra.ExactArgs(1),
	RunE: agentsShowCommand,
}

func init() {
	rootCmd.AddCommand(agentsCmd)
	agentsCmd.AddCommand(agentsListCmd)
	agentsCmd.AddCommand(agentsShowCmd)
}

func agentsListCommand(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		cfg = config.Default()
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "AGENT\tOUTPUT\tDEPENDS ON")

	for _, name := range cfg.GetAgentNames() {
		agent := cfg.Agents[name]
		deps := "-"
		if len(agent.DependsOn) > 0 {
			deps = fmt.Sprintf("%v", agent.DependsOn)
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", name, agent.Output, deps)
	}

	_ = w.Flush()
	return nil
}

func agentsShowCommand(cmd *cobra.Command, args []string) error {
	agentName := args[0]

	cfg, err := config.Load(configPath)
	if err != nil {
		cfg = config.Default()
	}

	agent, ok := cfg.Agents[agentName]
	if !ok {
		return fmt.Errorf("unknown agent: %s (use 'pagent agents list' to see available agents)", agentName)
	}

	fmt.Printf("Agent: %s\n", agentName)
	fmt.Printf("Output: %s\n", agent.Output)
	if len(agent.DependsOn) > 0 {
		fmt.Printf("Depends on: %v\n", agent.DependsOn)
	}
	fmt.Println()
	fmt.Println("Prompt Template:")
	fmt.Println("----------------")
	fmt.Println(agent.Prompt)

	return nil
}
