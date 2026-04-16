package cli

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/andrew8088/calvin/internal/calendar"
	"github.com/andrew8088/calvin/internal/config"
	"github.com/andrew8088/calvin/internal/db"
	"github.com/spf13/cobra"
)

var freeCmd = &cobra.Command{
	Use:     "free",
	Short:   "Show today's free time between meetings",
	Example: "  calvin free",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runFree(time.Now())
	},
}

func runFree(now time.Time) error {
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
		fmt.Fprintf(os.Stderr, "  %s daemon not running, showing cached events\n\n", symWarn())
	}

	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := start.Add(24 * time.Hour)

	events, err := database.ListEventsOverlapping(start, end)
	if err != nil {
		return err
	}

	slots := calendar.FreeSlotsForWindow(start, end, events)
	if jsonOutput {
		type freeSlotJSON struct {
			Start           string `json:"start"`
			End             string `json:"end"`
			DurationSeconds int64  `json:"duration_seconds"`
		}
		payload := make([]freeSlotJSON, 0, len(slots))
		for _, slot := range slots {
			payload = append(payload, freeSlotJSON{
				Start:           slot.Start.In(now.Location()).Format(time.RFC3339),
				End:             slot.End.In(now.Location()).Format(time.RFC3339),
				DurationSeconds: slot.DurationSeconds,
			})
		}
		return printJSON(payload)
	}

	if len(slots) == 0 {
		fmt.Println("No free time today.")
		return nil
	}

	for _, line := range formatFreeSlotsText(slots, now.Location()) {
		fmt.Println(line)
	}

	return nil
}

func formatFreeSlotsText(slots []calendar.FreeSlot, loc *time.Location) []string {
	lines := make([]string, 0, len(slots))
	for _, slot := range slots {
		lines = append(lines,
			slot.Start.In(loc).Format(time.RFC3339)+"\t"+
				slot.End.In(loc).Format(time.RFC3339)+"\t"+
				strconv.FormatInt(slot.DurationSeconds, 10),
		)
	}
	return lines
}
