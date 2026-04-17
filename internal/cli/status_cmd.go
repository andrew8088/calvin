package cli

import (
	"fmt"
	"os"
	"strconv"
	"syscall"
	"time"

	"github.com/andrew8088/calvin/internal/auth"
	"github.com/andrew8088/calvin/internal/config"
	"github.com/andrew8088/calvin/internal/db"
	"github.com/andrew8088/calvin/internal/hooks"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:     "status",
	Short:   "Show daemon health dashboard",
	Example: "  calvin status",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStatus()
	},
}

func runStatus() error {
	dbPath := config.DBPath()
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		errMsg("No event data", "Calvin hasn't been started yet.", "calvin start")
		return fmt.Errorf("no database")
	}

	database, err := db.Open(dbPath, true)
	if err != nil {
		return err
	}
	defer database.Close()

	running, pid, uptime := daemonStatus()
	syncToken, _ := database.GetSyncToken("primary")
	eventCount, _ := database.EventCount()
	hookCounts, _ := hooks.CountByType()
	success, failed, timeout, _ := database.GetHookStats()
	tokenStatus := "missing"
	if auth.HasToken() {
		cfg, _ := config.Load()
		if err := auth.CheckTokenValid(cfg); err != nil {
			tokenStatus = "invalid"
		} else {
			tokenStatus = "valid"
		}
	}

	if wantsJSON() {
		return writeCommandJSON("status", map[string]any{
			"running":               running,
			"pid":                   pid,
			"uptime_seconds":        uptime,
			"sync_token":            syncToken != "",
			"events_today":          eventCount,
			"sync_interval_seconds": config.Default().SyncIntervalSeconds,
			"hooks_registered":      hookCounts,
			"hooks_success_today":   success,
			"hooks_failed_today":    failed,
			"hooks_timeout_today":   timeout,
			"auth_status":           tokenStatus,
		})
	}

	fmt.Println()

	if running {
		fmt.Printf("  %s Calvin is running (uptime: %s)\n", symRun(), humanDuration(int64(uptime)))
	} else {
		fmt.Printf("  %s Calvin is not running\n", symStop())
		if pid > 0 {
			fmt.Printf("      Last PID: %d\n", pid)
		}
	}
	fmt.Println()

	fmt.Printf("  %s\n", bold("Sync"))
	if syncToken != "" {
		fmt.Printf("    last sync:    %s\n", dim("token present"))
	} else {
		fmt.Printf("    last sync:    %s\n", dim("never"))
	}
	fmt.Printf("    events today: %d\n", eventCount)
	fmt.Printf("    sync interval: %ds\n", config.Default().SyncIntervalSeconds)
	fmt.Println()

	fmt.Printf("  %s\n", bold("Hooks"))
	parts := []string{}
	for _, t := range []string{"before-event-start", "on-event-start", "on-event-end"} {
		if c, ok := hookCounts[t]; ok && c > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", c, t))
		}
	}
	if len(parts) > 0 {
		fmt.Printf("    registered:  %s\n", joinParts(parts))
	} else {
		fmt.Printf("    registered:  %s\n", dim("none"))
	}
	hookStats := fmt.Sprintf("%d %s", success, green("✓"))
	if failed > 0 {
		hookStats += fmt.Sprintf("  %d %s", failed, red("✗"))
	}
	if timeout > 0 {
		hookStats += fmt.Sprintf("  %d %s", timeout, yellow("△"))
	}
	fmt.Printf("    fired today: %s\n", hookStats)
	fmt.Println()

	fmt.Printf("  %s\n", bold("Auth"))
	if auth.HasToken() {
		cfg, _ := config.Load()
		if err := auth.CheckTokenValid(cfg); err != nil {
			fmt.Printf("    token: %s\n", red("invalid ("+err.Error()+")"))
		} else {
			fmt.Printf("    token: %s\n", green("valid"))
		}
	} else {
		fmt.Printf("    token: %s\n", red("missing"))
		fmt.Printf("           Fix: %s\n", cyan("calvin auth"))
	}
	fmt.Println()

	return nil
}

func daemonStatus() (running bool, pid int, uptimeSeconds float64) {
	data, err := os.ReadFile(config.PIDPath())
	if err != nil {
		return false, 0, 0
	}
	pid, err = strconv.Atoi(string(data))
	if err != nil {
		return false, 0, 0
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false, pid, 0
	}
	if err := process.Signal(syscall.Signal(0)); err != nil {
		return false, pid, 0
	}

	info, err := os.Stat(config.PIDPath())
	if err == nil {
		uptimeSeconds = time.Since(info.ModTime()).Seconds()
	}

	return true, pid, uptimeSeconds
}
