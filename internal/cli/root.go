package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	appVersion = "dev"
	appCommit  = "none"
	jsonOutput bool
)

func SetVersion(version, commit string) {
	appVersion = version
	appCommit = commit
}

func Execute() error {
	return rootCmd.Execute()
}

var rootCmd = &cobra.Command{
	Use:   "calvin",
	Short: "Programmable calendar hooks for your Mac",
	Long: `Calvin watches your Google Calendar and fires shell scripts
at event lifecycle moments. Drop scripts in your hooks directory,
and Calvin handles the rest.

Quick start:
  calvin init          Scaffold config and example hooks
  calvin auth          Authenticate with Google Calendar
  calvin test example-notify   Test an example hook
  calvin start --background    Start the daemon`,
}

func printJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(nextCmd)
	rootCmd.AddCommand(eventsCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(hooksCmd)
	rootCmd.AddCommand(testCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(weekCmd)
}
