package cli

import (
	"fmt"
	"os"
	"strconv"
	"syscall"
	"time"

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
	pid, process, err := findDaemonProcess()
	if err != nil {
		return err
	}
	if process == nil {
		if wantsJSON() {
			return writeCommandJSON("stop", map[string]any{
				"running": false,
				"signal":  "SIGTERM",
			})
		}
		return nil
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("sending SIGTERM to PID %d: %w", pid, err)
	}

	if wantsJSON() {
		return writeCommandJSON("stop", map[string]any{
			"running": true,
			"pid":     pid,
			"signal":  "SIGTERM",
		})
	}

	fmt.Printf("  %s Sent SIGTERM to Calvin (PID: %d)\n", symPass(), pid)

	deadline := time.After(5 * time.Second)
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			if err := process.Signal(syscall.Signal(0)); err != nil {
				fmt.Printf("  %s Calvin stopped\n", symPass())
				os.Remove(config.PIDPath())
				return nil
			}
		case <-deadline:
			fmt.Printf("  %s Calvin still running after 5s, force killing\n", symWarn())
			process.Signal(syscall.SIGKILL)
			os.Remove(config.PIDPath())
			return nil
		}
	}
}

func findDaemonProcess() (int, *os.Process, error) {
	data, err := os.ReadFile(config.PIDPath())
	if err != nil {
		if os.IsNotExist(err) {
			if !wantsJSON() {
				fmt.Printf("  %s Calvin is not running (no PID file)\n", symStop())
			}
			return 0, nil, nil
		}
		return 0, nil, err
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		os.Remove(config.PIDPath())
		return 0, nil, fmt.Errorf("invalid PID file, removed")
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		os.Remove(config.PIDPath())
		if !wantsJSON() {
			fmt.Printf("  %s Process %d not found, cleaned up PID file\n", symWarn(), pid)
		}
		return 0, nil, nil
	}

	if err := process.Signal(syscall.Signal(0)); err != nil {
		os.Remove(config.PIDPath())
		if !wantsJSON() {
			fmt.Printf("  %s Process %d not running, cleaned up stale PID file\n", symWarn(), pid)
		}
		return 0, nil, nil
	}

	return pid, process, nil
}
