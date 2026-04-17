package cli

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

var commandsCmd = &cobra.Command{
	Use:   "commands",
	Short: "List command metadata for humans and agents",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCommands()
	},
}

func runCommands() error {
	entries := buildCommandCatalog(rootCmd)
	paths := make([]string, 0, len(entries))
	for path := range entries {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	if wantsJSON() {
		payload := make([]commandMetadata, 0, len(paths))
		for _, path := range paths {
			payload = append(payload, entries[path])
		}
		return printJSON(payload)
	}

	for _, path := range paths {
		fmt.Printf("%s\t%s\n", path, entries[path].Summary)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(commandsCmd)
}
