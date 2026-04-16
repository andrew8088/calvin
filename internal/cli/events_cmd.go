package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/andrew8088/calvin/internal/calendar"
	"github.com/andrew8088/calvin/internal/config"
	"github.com/andrew8088/calvin/internal/db"
	"github.com/spf13/cobra"
)

var eventsCmd = &cobra.Command{
	Use:   "events [event-id]",
	Short: "List today's events or show event detail",
	Example: "  calvin events\n  calvin events abc123",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return runEventDetail(args[0])
		}
		return runEvents()
	},
}

func runEvents() error {
	dbPath := config.DBPath()
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		fmt.Println("  No cached events.")
		fmt.Printf("  Run: %s\n", cyan("calvin start"))
		return nil
	}

	database, err := db.Open(dbPath, true)
	if err != nil {
		return err
	}
	defer database.Close()

	if _, err := os.Stat(config.PIDPath()); os.IsNotExist(err) {
		fmt.Printf("  %s daemon not running, showing cached events\n", symWarn())
		fmt.Println()
	}

	events, err := database.ListEventsForDay(time.Now())
	if err != nil {
		return err
	}

	if len(events) == 0 {
		if jsonOutput {
			return printJSON([]calendar.Event{})
		}
		fmt.Println("  No events today.")
		return nil
	}

	if jsonOutput {
		return printJSON(events)
	}

	fmt.Println()
	fmt.Printf("  %s (%d events)\n", bold("Today"), len(events))
	fmt.Println()

	for _, e := range events {
		timeStr := e.Start.Local().Format("15:04")
		if e.AllDay {
			timeStr = "  ✦  "
		}
		title := truncate(e.Title, 40)

		execs, _ := database.GetHookExecutions(e.ID)
		hookStr := ""
		if len(execs) > 0 {
			success := 0
			failed := 0
			for _, ex := range execs {
				if ex.Status == "success" {
					success++
				} else {
					failed++
				}
			}
			if failed > 0 {
				hookStr = fmt.Sprintf("%s %d hooks fired, %s %d failed", symPass(), success, symFail(), failed)
			} else {
				hookStr = fmt.Sprintf("%s %d hooks fired", symPass(), success)
			}
		} else {
			now := time.Now()
			if e.Start.After(now) {
				until := time.Until(e.Start).Seconds()
				hookStr = fmt.Sprintf("%s pending (%s)", symStop(), humanCountdown(int64(until)))
			}
		}

		fmt.Printf("  %s  %-40s %s\n", dim(timeStr), title, hookStr)
	}
	fmt.Println()

	return nil
}

func runEventDetail(eventID string) error {
	dbPath := config.DBPath()
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		errMsg("No event data", "Calvin hasn't synced yet.", "calvin start")
		return fmt.Errorf("no database")
	}

	database, err := db.Open(dbPath, true)
	if err != nil {
		return err
	}
	defer database.Close()

	event, err := database.GetEvent(eventID)
	if err != nil {
		return err
	}
	if event == nil {
		errMsg(
			fmt.Sprintf("Event %s not in cache", eventID),
			"The event ID wasn't found in Calvin's database.",
			"calvin events  (to see cached event IDs)",
		)
		return fmt.Errorf("event not found")
	}

	execs, err := database.GetHookExecutions(eventID)
	if err != nil {
		return err
	}

	if jsonOutput {
		type execJSON struct {
			HookName   string `json:"hook_name"`
			HookType   string `json:"hook_type"`
			Status     string `json:"status"`
			Stdout     string `json:"stdout,omitempty"`
			Stderr     string `json:"stderr,omitempty"`
			DurationMs int64  `json:"duration_ms"`
			ExecutedAt string `json:"executed_at"`
		}
		type detailJSON struct {
			calendar.Event
			HookExecutions []execJSON `json:"hook_executions"`
		}
		je := []execJSON{}
		for _, ex := range execs {
			je = append(je, execJSON{
				HookName:   ex.HookName,
				HookType:   ex.HookType,
				Status:     ex.Status,
				Stdout:     ex.Stdout,
				Stderr:     ex.Stderr,
				DurationMs: ex.DurationMs,
				ExecutedAt: ex.ExecutedAt,
			})
		}
		return printJSON(detailJSON{Event: *event, HookExecutions: je})
	}

	fmt.Println()
	fmt.Printf("  %s\n", bold(event.Title))
	if event.AllDay {
		fmt.Printf("  %s\n", dim(event.Start.Local().Format("Mon Jan 2")+" (all day)"))
	} else {
		fmt.Printf("  %s - %s\n", dim(event.Start.Local().Format("Mon Jan 2 15:04")), dim(event.End.Local().Format("15:04")))
	}
	if event.MeetingLink != "" {
		fmt.Printf("  %s\n", blue(event.MeetingLink))
	}
	fmt.Printf("  ID: %s\n", dim(event.ID))
	fmt.Println()

	if len(execs) == 0 {
		fmt.Println("  No hook executions recorded.")
	} else {
		fmt.Printf("  %s\n", bold("Hook Executions"))
		fmt.Println()
		for _, ex := range execs {
			sym := symPass()
			if ex.Status == "failed" {
				sym = symFail()
			} else if ex.Status == "timeout" {
				sym = symWarn()
			} else if ex.Status == "skipped" {
				sym = symStop()
			}
			fmt.Printf("  %s %s/%s  %s  %s\n",
				sym, ex.HookType, ex.HookName,
				dim(fmt.Sprintf("%dms", ex.DurationMs)),
				dim(ex.ExecutedAt))

			if ex.Stdout != "" {
				lines := truncate(ex.Stdout, 200)
				fmt.Printf("      stdout: %s\n", dim(lines))
			}
			if ex.Stderr != "" && ex.Status != "success" {
				lines := truncate(ex.Stderr, 200)
				fmt.Printf("      stderr: %s\n", red(lines))
			}
		}
	}
	fmt.Println()

	return nil
}
