package cli

import (
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
			return auth.Revoke()
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
