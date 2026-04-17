package cli

import (
	"os"

	"github.com/andrew8088/calvin/internal/auth"
	"github.com/andrew8088/calvin/internal/config"
	"github.com/spf13/cobra"
)

var authRevoke bool

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with Google Calendar (read-only access)",
	Long: `Calvin requests read-only access to your calendar.
It never creates, modifies, or deletes events.`,
	Example: "  calvin auth\n  calvin auth --revoke",
	RunE: func(cmd *cobra.Command, args []string) error {
		if authRevoke {
			if wantsJSON() {
				return runAuthRevokeJSON()
			}
			return auth.Revoke()
		}
		if wantsJSON() {
			return newExitError(1, "auth", "unsupported_interactive_flow", "auth --json does not support browser OAuth yet", nil, nil)
		}
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		return auth.RunFlow(cfg)
	},
}

func init() {
	authCmd.Flags().BoolVar(&authRevoke, "revoke", false, "Clear stored credentials")
}

func runAuthRevokeJSON() error {
	path := config.TokenPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return writeCommandJSON("auth", map[string]any{
			"revoked":       false,
			"token_present": false,
		})
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	return writeCommandJSON("auth", map[string]any{
		"revoked":       true,
		"token_present": true,
	})
}
