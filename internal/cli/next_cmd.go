package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/andrew8088/calvin/internal/config"
	"github.com/andrew8088/calvin/internal/db"
	"github.com/andrew8088/calvin/internal/hooks"
	"github.com/spf13/cobra"
)

var nextCmd = &cobra.Command{
	Use:   "next",
	Short: "Show next upcoming event with countdown",
	Example: "  calvin next",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runNext()
	},
}

func runNext() error {
	dbPath := config.DBPath()
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		fmt.Println("  No upcoming events today. Enjoy the quiet.")
		return nil
	}

	database, err := db.Open(dbPath, true)
	if err != nil {
		return err
	}
	defer database.Close()

	events, err := database.ListUpcomingEvents(time.Now(), 1)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		if jsonOutput {
			return printJSON(nil)
		}
		fmt.Println("  No upcoming events today. Enjoy the quiet.")
		return nil
	}

	e := events[0]

	if jsonOutput {
		return printJSON(e)
	}

	until := time.Until(e.Start).Seconds()

	fmt.Println()
	fmt.Printf("  %s\n", bold(e.Title))
	fmt.Printf("  starts %s\n", humanCountdown(int64(until)))
	fmt.Println()

	if e.MeetingLink != "" {
		fmt.Printf("  %s\n", blue(e.MeetingLink))
	}

	if len(e.Attendees) > 0 {
		accepted := 0
		tentative := 0
		declined := 0
		for _, a := range e.Attendees {
			switch a.Response {
			case "accepted":
				accepted++
			case "tentative":
				tentative++
			case "declined":
				declined++
			}
		}
		parts := []string{}
		if accepted > 0 {
			parts = append(parts, fmt.Sprintf("%d accepted", accepted))
		}
		if tentative > 0 {
			parts = append(parts, fmt.Sprintf("%d tentative", tentative))
		}
		if declined > 0 {
			parts = append(parts, fmt.Sprintf("%d declined", declined))
		}
		fmt.Printf("  %d attendees (%s)\n", len(e.Attendees), joinParts(parts))
	}

	allHooks, _ := hooks.Discover()
	hookNames := []string{}
	for _, t := range []string{"pre_event", "event_start", "event_end"} {
		for _, h := range allHooks[t] {
			hookNames = append(hookNames, t+"/"+h.Name)
		}
	}
	if len(hookNames) > 0 {
		fmt.Printf("  hooks: %s\n", dim(joinParts(hookNames)))
	}
	fmt.Println()

	return nil
}
