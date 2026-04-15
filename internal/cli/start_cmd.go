package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/andrew8088/calvin/internal/auth"
	"github.com/andrew8088/calvin/internal/calendar"
	"github.com/andrew8088/calvin/internal/config"
	"github.com/andrew8088/calvin/internal/db"
	"github.com/andrew8088/calvin/internal/hooks"
	"github.com/andrew8088/calvin/internal/logging"
	"github.com/andrew8088/calvin/internal/scheduler"
	"github.com/spf13/cobra"
)

var startBackground bool

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Calvin daemon",
	Example: "  calvin start\n  calvin start --background",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStart()
	},
}

func init() {
	startCmd.Flags().BoolVar(&startBackground, "background", false, "Run daemon in background")
}

func runStart() error {
	cfg, err := config.Load()
	if err != nil {
		errMsg("Failed to load config", err.Error(), "calvin init")
		return err
	}

	if !auth.HasToken() {
		errMsg("No authentication token found",
			"Calvin needs access to your Google Calendar to work.",
			"calvin auth")
		return fmt.Errorf("no auth token")
	}

	if _, err := os.Stat(config.ConfigDir()); os.IsNotExist(err) {
		errMsg("Config directory not found",
			"Calvin hasn't been initialized yet.",
			"calvin init")
		return fmt.Errorf("not initialized")
	}

	if err := checkExistingPID(); err != nil {
		return err
	}

	if startBackground {
		return runBackground()
	}

	return runForeground(cfg)
}

func runBackground() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable: %w", err)
	}

	logPath := config.LogPath()
	logDir := config.StateDir()
	os.MkdirAll(logDir, 0755)

	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}

	attr := &os.ProcAttr{
		Dir:   "/",
		Env:   os.Environ(),
		Files: []*os.File{os.Stdin, logFile, logFile},
		Sys:   &syscall.SysProcAttr{Setsid: true},
	}

	proc, err := os.StartProcess(exe, []string{exe, "start"}, attr)
	if err != nil {
		logFile.Close()
		return fmt.Errorf("starting background process: %w", err)
	}

	logFile.Close()
	proc.Release()

	fmt.Printf("  %s Calvin running in background (PID: %d)\n", symRun(), proc.Pid)
	fmt.Printf("  Logs: %s\n", dim(logPath))

	return nil
}

func runForeground(cfg *config.Config) error {
	if err := logging.Init(); err != nil {
		return fmt.Errorf("initializing logging: %w", err)
	}
	log := logging.Get()

	if err := writePID(); err != nil {
		return err
	}
	defer removePID()

	database, err := db.Open(config.DBPath(), false)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer func() {
		database.Checkpoint()
		database.Close()
	}()

	if err := database.PruneOldExecutions(cfg.HookExecutionRetentionDays); err != nil {
		log.Warn("daemon", fmt.Sprintf("Failed to prune old executions: %v", err))
	}

	ts, err := auth.TokenSource(cfg)
	if err != nil {
		return fmt.Errorf("loading token: %w", err)
	}

	syncer := calendar.NewSyncer(ts)
	executor := hooks.NewExecutor(cfg, database)
	sched := scheduler.New(cfg, database, executor)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	printBanner(cfg, database)

	var hookWg sync.WaitGroup

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	syncCh := make(chan os.Signal, 1)
	signal.Notify(syncCh, syscall.SIGUSR1)

	ticker := time.NewTicker(time.Duration(cfg.SyncIntervalSeconds) * time.Second)
	defer ticker.Stop()

	failCount := 0
	calendars := cfg.ResolvedCalendars()

	doSync := func() {
		syncGen, _ := database.GetSyncGeneration()
		newGen := syncGen + 1

		var allEvents []calendar.Event
		var allFullSync bool

		for _, cal := range calendars {
			syncToken, _ := database.GetSyncToken(cal.ID)
			events, newToken, fullSync, err := syncer.Sync(ctx, cal.ID, syncToken)
			if err != nil {
				failCount++
				if failCount >= 5 {
					log.Error("sync", fmt.Sprintf("Calendar %s sync failed %d times. Check your network connection.", cal.ID, failCount))
				} else {
					log.Warn("sync", fmt.Sprintf("Sync %s failed (attempt %d): %v", cal.ID, failCount, err))
				}
				continue
			}
			failCount = 0

			if err := database.WithTransaction(ctx, func() error {
				for _, event := range events {
					if err := database.UpsertEvent(event, newGen); err != nil {
						return fmt.Errorf("upserting event: %w", err)
					}
				}

				if newToken != "" {
					if err := database.SetSyncToken(cal.ID, newToken); err != nil {
						return fmt.Errorf("saving sync token: %w", err)
					}
				}
				return nil
			}); err != nil {
				log.Error("sync", fmt.Sprintf("Transaction failed for %s: %v", cal.ID, err))
				continue
			}

			allEvents = append(allEvents, events...)
			if fullSync {
				allFullSync = true
			}
		}

		if allFullSync {
			deleted, err := database.DeleteStaleSyncGeneration(newGen)
			if err != nil {
				log.Error("sync", fmt.Sprintf("Failed to clean stale events: %v", err))
			} else if len(deleted) > 0 {
				log.Info("sync", fmt.Sprintf("Removed %d stale events", len(deleted)))
			}
		}

		dbEvents, _ := database.ListUpcomingEvents(time.Now().Add(-1*time.Hour), 200)
		diffs := calendar.Diff(dbEvents, allEvents, newGen)
		sched.ProcessDiff(ctx, diffs)

		if err := sched.ScheduleFromDB(ctx); err != nil {
			log.Error("scheduler", fmt.Sprintf("Failed to schedule: %v", err))
		}

		log.Info("sync", fmt.Sprintf("Synced %d events across %d calendars, %d diffs, %d timers active",
			len(allEvents), len(calendars), len(diffs), sched.TimerCount()))
	}

	doSync()

	for {
		select {
		case <-ticker.C:
			doSync()
		case <-syncCh:
			log.Info("daemon", "Received SIGUSR1, forcing sync")
			doSync()
			ticker.Reset(time.Duration(cfg.SyncIntervalSeconds) * time.Second)
		case sig := <-sigCh:
			log.Info("daemon", fmt.Sprintf("Received %s, shutting down...", sig))
			ticker.Stop()
			sched.CancelAll()

			done := make(chan struct{})
			go func() {
				hookWg.Wait()
				close(done)
			}()
			select {
			case <-done:
				log.Info("daemon", "All hooks completed, exiting cleanly")
			case <-time.After(5 * time.Second):
				log.Warn("daemon", "Shutdown timeout, force exiting with unfinished hooks")
			}
			return nil
		}
	}
}

func printBanner(cfg *config.Config, database *db.DB) {
	if !isTerminal() {
		return
	}

	eventCount, _ := database.EventCount()
	hookCounts, _ := hooks.CountByType()
	events, _ := database.ListUpcomingEvents(time.Now(), 1)

	fmt.Println()
	fmt.Printf("  %s Calvin %s\n", bold("Calvin"), dim(appVersion))
	fmt.Printf("  Config:  %s\n", dim(config.ConfigDir()))
	fmt.Printf("  Data:    %s\n", dim(config.DataDir()))
	fmt.Printf("  Logs:    %s\n", dim(config.LogPath()))

	if eventCount > 0 {
		line := fmt.Sprintf("  Events:  %d today", eventCount)
		if len(events) > 0 {
			until := time.Until(events[0].Start).Seconds()
			line += fmt.Sprintf(", next: %s (%s)", events[0].Title, humanCountdown(int64(until)))
		}
		fmt.Println(line)
	}

	hookLine := "  Hooks:   "
	parts := []string{}
	for _, t := range []string{"pre_event", "event_start", "event_end"} {
		if c, ok := hookCounts[t]; ok && c > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", c, t))
		}
	}
	if len(parts) > 0 {
		fmt.Println(hookLine + joinParts(parts))
	} else {
		fmt.Println(hookLine + dim("none"))
	}

	fmt.Printf("  Sync:    every %ds\n", cfg.SyncIntervalSeconds)
	fmt.Println()
}

func joinParts(parts []string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += ", "
		}
		result += p
	}
	return result
}

func checkExistingPID() error {
	data, err := os.ReadFile(config.PIDPath())
	if err != nil {
		return nil
	}
	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return nil
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return nil
	}
	if err := process.Signal(syscall.Signal(0)); err == nil {
		errMsg(
			fmt.Sprintf("Daemon already running (PID: %d)", pid),
			"Another instance of Calvin is already running.",
			"calvin stop  (or kill "+strconv.Itoa(pid)+")",
		)
		return fmt.Errorf("daemon already running")
	}
	log := logging.Get()
	log.Warn("daemon", fmt.Sprintf("Stale PID file found (PID %d not running), overwriting", pid))
	return nil
}

func writePID() error {
	dir := config.StateDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(config.PIDPath(), []byte(strconv.Itoa(os.Getpid())), 0644)
}

func removePID() {
	os.Remove(config.PIDPath())
}
