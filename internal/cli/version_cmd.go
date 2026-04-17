package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print Calvin version",
	Run: func(cmd *cobra.Command, args []string) {
		if wantsJSON() {
			_ = writeCommandJSON("version", map[string]any{
				"version": appVersion,
				"commit":  appCommit,
			})
			return
		}
		fmt.Printf("calvin %s (%s)\n", appVersion, appCommit)
	},
}
