package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/andrew8088/calvin/internal/config"
	"github.com/andrew8088/calvin/internal/hooks"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Scaffold config directories and example hooks",
	Example: "  calvin init",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInit()
	},
}

func runInit() error {
	dirs := []string{
		config.ConfigDir(),
		config.DataDir(),
		config.StateDir(),
	}
	for _, hookType := range hooks.ValidTypes {
		dirs = append(dirs, filepath.Join(config.HooksDir(), hookType))
	}

	allExisted := true
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			allExisted = false
			if err := os.MkdirAll(dir, 0755); err != nil {
				errMsg(
					fmt.Sprintf("Failed to create directory: %s", dir),
					err.Error(),
					fmt.Sprintf("Check permissions on %s", filepath.Dir(dir)),
				)
				return err
			}
		}
	}

	configPath := filepath.Join(config.ConfigDir(), "config.toml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		allExisted = false
		if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
			return err
		}
	}

	examplesCreated := 0
	for name, content := range exampleHooks {
		parts := filepath.SplitList(name)
		_ = parts
		hookPath := filepath.Join(config.HooksDir(), name)
		if _, err := os.Stat(hookPath); os.IsNotExist(err) {
			allExisted = false
			if err := os.WriteFile(hookPath, []byte(content), 0755); err != nil {
				return err
			}
			examplesCreated++
		}
	}

	if allExisted {
		fmt.Println("  Already initialized. All directories and files exist.")
		return nil
	}

	fmt.Println(bold("  Calvin initialized!"))
	fmt.Println()
	if examplesCreated > 0 {
		fmt.Printf("  Created %d example hooks. Try one now:\n", examplesCreated)
	}
	fmt.Println()
	fmt.Printf("    %s\n", cyan("calvin auth"))
	fmt.Printf("    %s\n", cyan("calvin test example-notify"))
	fmt.Println()
	fmt.Println("  Or create your own:")
	fmt.Printf("    %s\n", cyan("calvin hooks new before-event-start my-hook"))
	fmt.Println()
	fmt.Println("  When ready:")
	fmt.Printf("    %s\n", cyan("calvin start --background"))
	fmt.Println()

	return nil
}

const defaultConfig = `# Calvin configuration
# See: calvin hooks schema for the JSON input format

sync_interval_seconds = 60
pre_event_minutes = 5
hook_timeout_seconds = 30
max_concurrent_hooks = 10
hook_output_max_bytes = 65536
hook_execution_retention_days = 30

# Uncomment to use your own Google Cloud OAuth credentials:
# oauth_client_id = ""
# oauth_client_secret = ""
# auth_port = 8085

# Calendars to watch (defaults to "primary" if omitted):
# [[calendars]]
# id = "primary"
#
# [[calendars]]
# id = "personal@gmail.com"
`

var exampleHooks = map[string]string{
	"before-event-start/example-notify": `#!/usr/bin/env bash
# Example Calvin hook: desktop notification before meetings
# This hook receives event JSON on stdin. Use jq to extract fields.
#
# Hook type: before-event-start (fires before the event starts)
# Run: calvin hooks schema  to see all available JSON fields

INPUT=$(cat /dev/stdin)
TITLE=$(echo "$INPUT" | jq -r '.title')
START=$(echo "$INPUT" | jq -r '.start')
LINK=$(echo "$INPUT" | jq -r '.meeting_link // empty')

MSG="Meeting in 5 min: $TITLE"
if [ -n "$LINK" ]; then
  MSG="$MSG\n$LINK"
fi

# macOS notification
osascript -e "display notification \"$MSG\" with title \"Calvin\""
echo "Notified: $TITLE at $START"
`,
	"on-event-start/example-open-link": `#!/usr/bin/env bash
# Example Calvin hook: auto-open meeting links when events start
# Hook type: on-event-start (fires when the event begins)

INPUT=$(cat /dev/stdin)
LINK=$(echo "$INPUT" | jq -r '.meeting_link // empty')
TITLE=$(echo "$INPUT" | jq -r '.title')

if [ -n "$LINK" ]; then
  open "$LINK"
  echo "Opened: $LINK for $TITLE"
else
  echo "No meeting link for: $TITLE"
fi
`,
	"on-event-end/example-log": `#!/usr/bin/env bash
# Example Calvin hook: log when meetings end
# Hook type: on-event-end (fires when the event ends)

INPUT=$(cat /dev/stdin)
TITLE=$(echo "$INPUT" | jq -r '.title')
DURATION_START=$(echo "$INPUT" | jq -r '.start')
DURATION_END=$(echo "$INPUT" | jq -r '.end')

echo "Meeting ended: $TITLE ($DURATION_START to $DURATION_END)"
`,
}
