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

var weekCmd = &cobra.Command{
	Use:     "week",
	Short:   "Show the next 7 days of events",
	Example: "  calvin week",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWeek(time.Now())
	},
}

func runWeek(now time.Time) error {
	dbPath := config.DBPath()
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		if wantsJSON() {
			return writeCommandJSON("week", map[string]any{"events": []calendar.Event{}}, "no cached events")
		}
		fmt.Println("  No cached events.")
		fmt.Printf("  Run: %s\n", cyan("calvin start"))
		return nil
	}

	database, err := db.Open(dbPath, true)
	if err != nil {
		return err
	}
	defer database.Close()

	warnings := []string{}
	if _, err := os.Stat(config.PIDPath()); os.IsNotExist(err) {
		if wantsJSON() {
			warnings = append(warnings, "daemon not running, showing cached events")
		} else {
			fmt.Printf("  %s daemon not running, showing cached events\n", symWarn())
			fmt.Println()
		}
	}

	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := start.AddDate(0, 0, 7)

	events, err := database.ListEventsBetween(start, end)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		if wantsJSON() {
			return writeCommandJSON("week", map[string]any{"events": []calendar.Event{}}, warnings...)
		}
		fmt.Println("  No events this week.")
		return nil
	}

	if wantsJSON() {
		return writeCommandJSON("week", map[string]any{"events": events}, warnings...)
	}

	grouped := groupByDay(events, now.Location())

	fmt.Println()
	totalEvents := len(events)
	fmt.Printf("  %s (%d events)\n", bold("Next 7 Days"), totalEvents)

	for day := 0; day < 7; day++ {
		date := start.AddDate(0, 0, day)
		key := date.Format("2006-01-02")
		dayEvents := grouped[key]

		fmt.Println()
		label := date.Format("Mon Jan 2")
		if day == 0 {
			label += " (today)"
		} else if day == 1 {
			label += " (tomorrow)"
		}
		fmt.Printf("  %s  %s\n", bold(label), dim(fmt.Sprintf("%d events", len(dayEvents))))

		if len(dayEvents) == 0 {
			fmt.Printf("    %s\n", dim("No events"))
			continue
		}

		for _, e := range dayEvents {
			timeStr := e.Start.Local().Format("15:04")
			if e.AllDay {
				timeStr = "  ✦  "
			}
			title := truncate(e.Title, 40)
			fmt.Printf("    %s  %s\n", dim(timeStr), title)
		}
	}
	fmt.Println()

	return nil
}

func groupByDay(events []calendar.Event, loc *time.Location) map[string][]calendar.Event {
	grouped := make(map[string][]calendar.Event)
	for _, e := range events {
		key := e.Start.In(loc).Format("2006-01-02")
		grouped[key] = append(grouped[key], e)
	}
	return grouped
}
