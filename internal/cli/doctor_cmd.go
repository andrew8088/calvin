package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/andrew8088/calvin/internal/auth"
	"github.com/andrew8088/calvin/internal/calendar"
	"github.com/andrew8088/calvin/internal/config"
	"github.com/andrew8088/calvin/internal/db"
	"github.com/andrew8088/calvin/internal/hooks"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:     "doctor",
	Short:   "Check Calvin's health",
	Example: "  calvin doctor",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDoctor()
	},
}

func runDoctor() error {
	fmt.Println()
	fmt.Printf("  %s\n", bold("Calvin Doctor"))
	fmt.Println()

	passed := 0
	failed := 0

	check := func(name string, fn func() error) {
		if err := fn(); err != nil {
			fmt.Printf("  %s %s\n", symFail(), name)
			fmt.Printf("      %s\n", red(err.Error()))
			failed++
		} else {
			fmt.Printf("  %s %s\n", symPass(), name)
			passed++
		}
	}

	check("Config directory exists", func() error {
		if _, err := os.Stat(config.ConfigDir()); os.IsNotExist(err) {
			return fmt.Errorf("not found. Fix: calvin init")
		}
		return nil
	})

	check("Hooks directory exists", func() error {
		if _, err := os.Stat(config.HooksDir()); os.IsNotExist(err) {
			return fmt.Errorf("not found. Fix: calvin init")
		}
		return nil
	})

	check("Config file loads", func() error {
		_, err := config.Load()
		return err
	})

	check("OAuth token present", func() error {
		if !auth.HasToken() {
			return fmt.Errorf("no token found. Fix: calvin auth")
		}
		return nil
	})

	check("OAuth token valid", func() error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		return auth.CheckTokenValid(cfg)
	})

	check("Google Calendar API accessible", func() error {
		if !auth.HasToken() {
			return fmt.Errorf("skipped (no token)")
		}
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		ts, err := auth.TokenSource(cfg)
		if err != nil {
			return err
		}
		syncer := calendar.NewSyncer(ts)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return syncer.CheckAPIAccess(ctx)
	})

	check("SQLite database", func() error {
		dbPath := config.DBPath()
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			return fmt.Errorf("no database yet (will be created on first sync)")
		}
		database, err := db.Open(dbPath, true)
		if err != nil {
			return err
		}
		defer database.Close()
		return database.IntegrityCheck()
	})

	check("Hooks discoverable", func() error {
		allHooks, err := hooks.Discover()
		if err != nil {
			return err
		}
		total := 0
		for _, list := range allHooks {
			total += len(list)
		}
		if total == 0 {
			return fmt.Errorf("no hooks found. Fix: calvin hooks new before-event-start my-hook")
		}
		return nil
	})

	check("Daemon running", func() error {
		running, pid, _ := daemonStatus()
		if !running {
			if pid > 0 {
				return fmt.Errorf("not running (last PID: %d). Fix: calvin start --background", pid)
			}
			return fmt.Errorf("not running. Fix: calvin start --background")
		}
		return nil
	})

	fmt.Println()
	if failed == 0 {
		fmt.Printf("  %s All %d checks passed\n", symPass(), passed)
	} else {
		fmt.Printf("  %d passed, %d failed\n", passed, failed)
	}
	fmt.Println()

	return nil
}
