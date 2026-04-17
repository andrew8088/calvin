package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var describeCmd = &cobra.Command{
	Use:   "describe <command-path>",
	Short: "Describe a command for humans and agents",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDescribe(strings.Join(args, " "))
	},
}

func runDescribe(path string) error {
	entry, ok := buildCommandCatalog(rootCmd)[path]
	if !ok {
		return newExitError(2, "describe", "unknown_command_path", fmt.Sprintf("unknown command path: %s", path), nil, nil)
	}

	if wantsJSON() {
		return printJSON(entry)
	}

	fmt.Printf("%s\n\n", entry.Path)
	fmt.Printf("%s\n", entry.Summary)
	for _, flag := range entry.Flags {
		fmt.Printf("flag\t%s\t%s\t%s\n", flag.Scope, flag.Name, flag.Type)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(describeCmd)
}
