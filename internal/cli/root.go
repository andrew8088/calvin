package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	appVersion        = "dev"
	appCommit         = "none"
	jsonOutput        bool
	outputFlag        string
	currentOutputMode = outputModeText
)

func SetVersion(version, commit string) {
	appVersion = version
	appCommit = commit
}

func Execute() error {
	args := os.Args[1:]
	jsonIntent := rawOutputMode(args, os.Getenv("CALVIN_OUTPUT")) == outputModeJSON
	rootCmd.SilenceErrors = jsonIntent
	rootCmd.SilenceUsage = jsonIntent

	err := rootCmd.Execute()
	if err == nil {
		return nil
	}

	exitErr := wrapCLIError(err, args)
	if !jsonIntent {
		return exitErr
	}
	if writeErr := writeJSONError(os.Stderr, exitErr.Result); writeErr != nil {
		return fmt.Errorf("write json error: %w", writeErr)
	}
	return exitErr
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
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		currentOutputMode = resolveOutputMode(outputFlag, jsonOutput, os.Getenv("CALVIN_OUTPUT"))
		if err := validateOutputMode(outputFlag); err != nil {
			return newExitError(2, cmd.CommandPath(), "invalid_output_mode", err.Error(), map[string]any{
				"valid_output_modes": []string{string(outputModeText), string(outputModeJSON)},
			}, err)
		}
		return nil
	},
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
	rootCmd.PersistentFlags().StringVar(&outputFlag, "output", "", "Output format: text or json")
	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return newExitError(2, cmd.CommandPath(), "invalid_flag", err.Error(), nil, err)
	})
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(nextCmd)
	rootCmd.AddCommand(eventsCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(hooksCmd)
	rootCmd.AddCommand(matchCmd)
	rootCmd.AddCommand(ignoreCmd)
	rootCmd.AddCommand(testCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(weekCmd)
	rootCmd.AddCommand(freeCmd)
}
