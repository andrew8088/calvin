package cli

import (
	"fmt"
	"os"
	"strconv"
	"syscall"

	"github.com/andrew8088/calvin/internal/config"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:     "stop",
	Short:   "Stop the Calvin daemon",
	Example: "  calvin stop",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStop()
	},
}

func runStop() error {
	data, err := os.ReadFile(config.PIDPath())
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("  %s Calvin is not running (no PID file)\n", symStop())
			return nil
		}
		return err
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		os.Remove(config.PIDPath())
		return fmt.Errorf("invalid PID file, removed")
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		os.Remove(config.PIDPath())
		fmt.Printf("  %s Process %d not found, cleaned up PID file\n", symWarn(), pid)
		return nil
	}

	if err := process.Signal(syscall.Signal(0)); err != nil {
		os.Remove(config.PIDPath())
		fmt.Printf("  %s Process %d not running, cleaned up stale PID file\n", symWarn(), pid)
		return nil
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("sending SIGTERM to PID %d: %w", pid, err)
	}

	fmt.Printf("  %s Sent SIGTERM to Calvin (PID: %d)\n", symPass(), pid)
	return nil
}
