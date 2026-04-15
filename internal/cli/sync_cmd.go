package cli

import (
	"fmt"
	"syscall"

	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:     "sync",
	Short:   "Force an immediate calendar sync",
	Long:    "Sends SIGUSR1 to the running daemon to trigger an immediate sync cycle.",
	Example: "  calvin sync",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSync()
	},
}

func runSync() error {
	pid, process, err := findDaemonProcess()
	if err != nil {
		return err
	}
	if process == nil {
		return nil
	}

	if err := process.Signal(syscall.SIGUSR1); err != nil {
		return fmt.Errorf("sending SIGUSR1 to PID %d: %w", pid, err)
	}

	fmt.Printf("  %s Sync triggered (PID: %d)\n", symPass(), pid)
	return nil
}
