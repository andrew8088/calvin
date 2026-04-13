package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/andrew8088/calvin/internal/auth"
	"github.com/andrew8088/calvin/internal/calendar"
	"github.com/andrew8088/calvin/internal/config"
	"github.com/andrew8088/calvin/internal/db"
	"github.com/andrew8088/calvin/internal/hooks"
	"github.com/spf13/cobra"
)

var testEventID string

var testCmd = &cobra.Command{
	Use:   "test <hook-name>",
	Short: "Test a hook with real or mock event data",
	Args:  cobra.ExactArgs(1),
	Example: "  calvin test example-notify\n  calvin test my-hook --event abc123",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTest(args[0])
	},
}

func init() {
	testCmd.Flags().StringVar(&testEventID, "event", "", "Use a specific event ID from cache")
}

func runTest(hookName string) error {
	hookPath, hookType, err := findHook(hookName)
	if err != nil {
		return err
	}

	payload, err := buildTestPayload(hookType)
	if err != nil {
		return err
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("  %s Testing %s/%s\n", bold("▶"), hookType, hookName)
	fmt.Printf("  Event: %s\n", dim(payload.Title))
	fmt.Println()

	stdout, stderr, exitCode, err := hooks.RunTest(hookPath, payloadBytes)
	if err != nil {
		errMsg(
			fmt.Sprintf("Hook failed to execute: %s", hookName),
			err.Error(),
			fmt.Sprintf("Check permissions: ls -la %s", hookPath),
		)
		return err
	}

	if stdout != "" {
		fmt.Printf("  %s\n", bold("stdout"))
		fmt.Printf("  %s\n", stdout)
	}
	if stderr != "" {
		fmt.Printf("  %s\n", bold("stderr"))
		fmt.Printf("  %s\n", red(stderr))
	}

	if exitCode == 0 {
		fmt.Printf("  %s Hook passed (exit 0)\n", symPass())
	} else {
		fmt.Printf("  %s Hook failed (exit %d)\n", symFail(), exitCode)
	}
	fmt.Println()

	return nil
}

func findHook(name string) (string, string, error) {
	allHooks, err := hooks.Discover()
	if err != nil {
		return "", "", err
	}

	for _, hookType := range hooks.ValidTypes {
		for _, h := range allHooks[hookType] {
			if h.Name == name {
				return h.Path, hookType, nil
			}
		}
	}

	for _, hookType := range hooks.ValidTypes {
		path := filepath.Join(config.HooksDir(), hookType, name)
		if _, err := os.Stat(path); err == nil {
			errMsg(
				fmt.Sprintf("Hook found but not executable: %s", name),
				path,
				fmt.Sprintf("chmod +x %s", path),
			)
			return "", "", fmt.Errorf("hook not executable")
		}
	}

	errMsg(
		fmt.Sprintf("Hook not found: %s", name),
		"No hook with that name exists in any hook directory.",
		"calvin hooks list",
	)
	return "", "", fmt.Errorf("hook not found: %s", name)
}

func buildTestPayload(hookType string) (calendar.HookPayload, error) {
	if testEventID != "" {
		return payloadFromDB(testEventID, hookType)
	}

	if auth.HasToken() {
		payload, err := payloadFromCalendar(hookType)
		if err == nil {
			return payload, nil
		}
	}

	return mockPayload(hookType), nil
}

func payloadFromDB(eventID, hookType string) (calendar.HookPayload, error) {
	dbPath := config.DBPath()
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		errMsg(
			"No cached events",
			"Calvin hasn't synced yet.",
			"calvin start",
		)
		return calendar.HookPayload{}, fmt.Errorf("no database")
	}

	database, err := db.Open(dbPath, true)
	if err != nil {
		return calendar.HookPayload{}, err
	}
	defer database.Close()

	event, err := database.GetEvent(eventID)
	if err != nil {
		return calendar.HookPayload{}, err
	}
	if event == nil {
		errMsg(
			fmt.Sprintf("Event %s not found", eventID),
			"That event ID isn't in Calvin's cache.",
			"calvin events",
		)
		return calendar.HookPayload{}, fmt.Errorf("event not found")
	}

	return calendar.EventToPayload(*event, hookType, nil, nil), nil
}

func payloadFromCalendar(hookType string) (calendar.HookPayload, error) {
	cfg, err := config.Load()
	if err != nil {
		return calendar.HookPayload{}, err
	}

	ts, err := auth.TokenSource(cfg)
	if err != nil {
		return calendar.HookPayload{}, err
	}

	syncer := calendar.NewSyncer(ts)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	event, err := syncer.FetchNextEvent(ctx)
	if err != nil {
		return calendar.HookPayload{}, err
	}
	if event == nil {
		return calendar.HookPayload{}, fmt.Errorf("no upcoming events")
	}

	fmt.Printf("  %s Using real event from your calendar\n", dim("ℹ"))
	return calendar.EventToPayload(*event, hookType, nil, nil), nil
}

func mockPayload(hookType string) calendar.HookPayload {
	fmt.Printf("  %s Using mock event data\n", dim("ℹ"))
	now := time.Now()
	start := now.Add(5 * time.Minute)
	end := start.Add(30 * time.Minute)
	link := "https://meet.google.com/abc-defg-hij"
	return calendar.HookPayload{
		SchemaVersion:   1,
		ID:              "mock-event-001",
		Title:           "Weekly Team Standup",
		Start:           start.Format(time.RFC3339),
		End:             end.Format(time.RFC3339),
		Location:        "",
		Description:     "Weekly sync with the team",
		MeetingLink:     &link,
		MeetingProvider: "google_meet",
		Attendees: []calendar.Attendee{
			{Email: "you@example.com", Name: "You", Response: "accepted"},
			{Email: "teammate@example.com", Name: "Teammate", Response: "accepted"},
		},
		Organizer: "you@example.com",
		Calendar:  "primary",
		Status:    "confirmed",
		HookType:  hookType,
	}
}
