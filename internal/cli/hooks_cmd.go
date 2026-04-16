package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/andrew8088/calvin/internal/config"
	"github.com/andrew8088/calvin/internal/hooks"
	"github.com/spf13/cobra"
)

var hooksCmd = &cobra.Command{
	Use:   "hooks",
	Short: "Manage Calvin hooks",
	Example: "  calvin hooks list\n  calvin hooks new before-event-start my-hook\n  calvin hooks schema",
}

var hooksListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all discovered hooks",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runHooksList()
	},
}

var hooksNewCmd = &cobra.Command{
	Use:   "new <type> <name>",
	Short: "Create a new hook from template",
	Args:  cobra.ExactArgs(2),
	Example: "  calvin hooks new before-event-start my-notifier\n  calvin hooks new on-event-start open-zoom",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runHooksNew(args[0], args[1])
	},
}

var hooksSchemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Print the JSON schema that hooks receive on stdin",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runHooksSchema()
	},
}

func init() {
	hooksCmd.AddCommand(hooksListCmd)
	hooksCmd.AddCommand(hooksNewCmd)
	hooksCmd.AddCommand(hooksSchemaCmd)
}

func runHooksList() error {
	allHooks, err := hooks.Discover()
	if err != nil {
		return err
	}

	total := 0
	for _, list := range allHooks {
		total += len(list)
	}

	if total == 0 {
		fmt.Println("  No hooks found.")
		fmt.Printf("  Hooks directory: %s\n", dim(config.HooksDir()))
		fmt.Printf("  Create one: %s\n", cyan("calvin hooks new before-event-start my-hook"))
		return nil
	}

	fmt.Println()
	for _, hookType := range hooks.ValidTypes {
		list, ok := allHooks[hookType]
		if !ok || len(list) == 0 {
			continue
		}
		fmt.Printf("  %s\n", bold(hookType))
		for _, h := range list {
			fmt.Printf("    %s %s\n", symPass(), h.Name)
			fmt.Printf("      %s\n", dim(h.Path))
		}
		fmt.Println()
	}

	return nil
}

func runHooksNew(hookType, name string) error {
	valid := false
	for _, t := range hooks.ValidTypes {
		if hookType == t {
			valid = true
			break
		}
	}
	if !valid {
		errMsg(
			fmt.Sprintf("Invalid hook type: %s", hookType),
			fmt.Sprintf("Valid types: %s", joinParts(hooks.ValidTypes)),
			fmt.Sprintf("calvin hooks new %s %s", hooks.ValidTypes[0], name),
		)
		return fmt.Errorf("invalid hook type: %s", hookType)
	}

	dir := filepath.Join(config.HooksDir(), hookType)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	hookPath := filepath.Join(dir, name)
	if _, err := os.Stat(hookPath); err == nil {
		errMsg(
			fmt.Sprintf("Hook already exists: %s", name),
			hookPath,
			fmt.Sprintf("Edit the existing hook or choose a different name"),
		)
		return fmt.Errorf("hook exists")
	}

	template := fmt.Sprintf(`#!/usr/bin/env bash
# Calvin %s hook: %s
# Receives event JSON on stdin. Run: calvin hooks schema

EVENT=$(cat /dev/stdin)
TITLE=$(echo "$EVENT" | jq -r '.title')
START=$(echo "$EVENT" | jq -r '.start')
LINK=$(echo "$EVENT" | jq -r '.meeting_link // empty')

echo "Hook fired: $TITLE at $START"
`, hookType, name)

	if err := os.WriteFile(hookPath, []byte(template), 0755); err != nil {
		return err
	}

	fmt.Printf("  %s Created %s/%s\n", symPass(), hookType, name)
	fmt.Printf("  Path: %s\n", dim(hookPath))
	fmt.Println()
	fmt.Printf("  Test it: %s\n", cyan("calvin test "+name))
	fmt.Printf("  Edit it: %s\n", cyan("$EDITOR "+hookPath))
	fmt.Println()

	return nil
}

func runHooksSchema() error {
	schema := map[string]any{
		"schema_version":  1,
		"id":              "abc123xyz",
		"title":           "Weekly standup",
		"start":           "2024-01-15T10:00:00-08:00",
		"end":             "2024-01-15T10:30:00-08:00",
		"all_day":         false,
		"location":        "",
		"description":     "Team sync",
		"meeting_link":    "https://meet.google.com/abc-defg-hij",
		"meeting_provider": "google_meet",
		"attendees": []map[string]string{
			{"email": "alice@example.com", "name": "Alice", "response": "accepted"},
			{"email": "bob@example.com", "name": "Bob", "response": "tentative"},
		},
		"organizer":  "alice@example.com",
		"calendar":   "primary",
		"status":     "confirmed",
		"hook_type":  "before-event-start",
		"previous_event": map[string]string{
			"id": "prev123", "title": "Morning coffee", "start": "2024-01-15T09:00:00-08:00",
			"end": "2024-01-15T09:30:00-08:00", "meeting_link": "",
		},
		"next_event": map[string]string{
			"id": "next456", "title": "Lunch", "start": "2024-01-15T12:00:00-08:00",
			"end": "2024-01-15T13:00:00-08:00", "meeting_link": "",
		},
	}

	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("  %s\n", bold("Hook Input Schema (JSON on stdin)"))
	fmt.Println()
	fmt.Println(string(data))
	fmt.Println()
	fmt.Println("  Fields:")
	fmt.Printf("    schema_version   %s\n", dim("always 1 (will increment on breaking changes)"))
	fmt.Printf("    all_day          %s\n", dim("true for all-day events (start/end are date-only strings)"))
	fmt.Printf("    meeting_link     %s\n", dim("null when no meeting link exists"))
	fmt.Printf("    hook_type        %s\n", dim("before-event-start | on-event-start | on-event-end"))
	fmt.Printf("    previous_event   %s\n", dim("null if this is the first event"))
	fmt.Printf("    next_event       %s\n", dim("null if this is the last event"))
	fmt.Println()
	fmt.Printf("  Usage in bash: %s\n", dim("INPUT=$(cat /dev/stdin); TITLE=$(echo \"$INPUT\" | jq -r '.title')"))
	fmt.Println()

	return nil
}
