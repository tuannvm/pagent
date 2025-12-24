package cmd

import (
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/tuannvm/pagent/internal/config"
)

func agentsMain(args []string) error {
	if len(args) == 0 {
		printAgentsUsage()
		return nil
	}

	subcmd := args[0]
	switch subcmd {
	case "list":
		return agentsListMain(args[1:])
	case "show":
		return agentsShowMain(args[1:])
	case "-h", "-help", "help":
		printAgentsUsage()
		return nil
	default:
		printAgentsUsage()
		return fmt.Errorf("unknown agents subcommand: %s", subcmd)
	}
}

func printAgentsUsage() {
	fmt.Print(`Usage: pagent agents <command>

Manage agent definitions.

Commands:
  list    List available agents
  show    Show agent prompt template

Examples:
  pagent agents list
  pagent agents show architect
`)
}

func agentsListMain(args []string) error {
	fs := flag.NewFlagSet("agents list", flag.ContinueOnError)
	var configPath string
	fs.StringVar(&configPath, "c", "", "config file path")
	fs.StringVar(&configPath, "config", "", "config file path")
	parseGlobalFlags(fs)

	fs.Usage = func() {
		fmt.Print(`Usage: pagent agents list [flags]

List all available agent types with their output files.

Flags:
  -c, -config string    Config file path
`)
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		cfg = config.Default()
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "AGENT\tOUTPUT\tDEPENDS ON")

	for _, name := range cfg.GetAgentNames() {
		agentCfg := cfg.Agents[name]
		deps := "-"
		if len(agentCfg.DependsOn) > 0 {
			deps = fmt.Sprintf("%v", agentCfg.DependsOn)
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", name, agentCfg.Output, deps)
	}

	_ = w.Flush()
	return nil
}

func agentsShowMain(args []string) error {
	fs := flag.NewFlagSet("agents show", flag.ContinueOnError)
	var configPath string
	fs.StringVar(&configPath, "c", "", "config file path")
	fs.StringVar(&configPath, "config", "", "config file path")
	parseGlobalFlags(fs)

	fs.Usage = func() {
		fmt.Print(`Usage: pagent agents show <agent> [flags]

Show the prompt template for a specific agent.

Arguments:
  <agent>    Name of the agent

Flags:
  -c, -config string    Config file path

Examples:
  pagent agents show architect
  pagent agents show implementer
`)
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		fs.Usage()
		return fmt.Errorf("missing required argument: agent name")
	}

	agentName := fs.Arg(0)

	cfg, err := config.Load(configPath)
	if err != nil {
		cfg = config.Default()
	}

	agentCfg, ok := cfg.Agents[agentName]
	if !ok {
		return fmt.Errorf("unknown agent: %s (use 'pagent agents list' to see available agents)", agentName)
	}

	fmt.Printf("Agent: %s\n", agentName)
	fmt.Printf("Output: %s\n", agentCfg.Output)
	if len(agentCfg.DependsOn) > 0 {
		fmt.Printf("Depends on: %v\n", agentCfg.DependsOn)
	}
	fmt.Println()
	fmt.Println("Prompt Template:")
	fmt.Println("----------------")
	fmt.Println(agentCfg.Prompt)

	return nil
}
