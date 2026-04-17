package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/andrew8088/calvin/internal/config"
	"github.com/andrew8088/calvin/internal/logging"
	"github.com/spf13/cobra"
)

var (
	logsHook  string
	logsEvent string
	logsType  string
	logsLevel string
	logsSince string
	logsLines int
)

var logsCmd = &cobra.Command{
	Use:     "logs",
	Short:   "Show Calvin daemon logs",
	Example: "  calvin logs\n  calvin logs --hook my-hook\n  calvin logs --level error\n  calvin logs -n 50",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLogs()
	},
}

func init() {
	logsCmd.Flags().StringVar(&logsHook, "hook", "", "Filter by hook name")
	logsCmd.Flags().StringVar(&logsEvent, "event", "", "Filter by event ID")
	logsCmd.Flags().StringVar(&logsType, "type", "", "Filter by hook type")
	logsCmd.Flags().StringVar(&logsLevel, "level", "", "Filter by log level (info, warn, error)")
	logsCmd.Flags().StringVar(&logsSince, "since", "", "Filter entries since timestamp")
	logsCmd.Flags().IntVarP(&logsLines, "lines", "n", 100, "Number of lines to show")
}

func runLogs() error {
	logPath := config.LogPath()
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		if wantsJSON() {
			return writeCommandJSON("logs", map[string]any{"entries": []logging.Entry{}}, "no log file found")
		}
		fmt.Println("  No log file found.")
		fmt.Printf("  Expected at: %s\n", dim(logPath))
		fmt.Printf("  Start the daemon: %s\n", cyan("calvin start"))
		return nil
	}

	f, err := os.Open(logPath)
	if err != nil {
		return err
	}
	defer f.Close()

	var entries []logging.Entry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var entry logging.Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if !matchesFilter(entry) {
			continue
		}
		entries = append(entries, entry)
	}
	entries, err = filterLogsSince(entries, logsSince)
	if err != nil {
		return newExitError(2, "logs", "invalid_since", err.Error(), nil, err)
	}

	if len(entries) == 0 {
		if wantsJSON() {
			return writeCommandJSON("logs", map[string]any{"entries": []logging.Entry{}})
		}
		fmt.Println("  No matching log entries.")
		return nil
	}

	start := 0
	if len(entries) > logsLines {
		start = len(entries) - logsLines
	}

	if wantsJSON() {
		return writeCommandJSON("logs", map[string]any{"entries": entries[start:]})
	}

	fmt.Println()
	for _, entry := range entries[start:] {
		printLogEntry(entry)
	}
	fmt.Println()

	return nil
}

func matchesFilter(e logging.Entry) bool {
	if logsHook != "" && e.HookName != logsHook {
		return false
	}
	if logsEvent != "" && e.EventID != logsEvent {
		return false
	}
	if logsType != "" && e.HookType != logsType {
		return false
	}
	if logsLevel != "" && string(e.Level) != logsLevel {
		return false
	}
	return true
}

func validateSinceTimestamp(since string) error {
	if since == "" {
		return nil
	}
	if _, err := time.Parse(time.RFC3339, since); err != nil {
		return fmt.Errorf("invalid --since timestamp %q: %w", since, err)
	}
	return nil
}

func filterLogsSince(entries []logging.Entry, since string) ([]logging.Entry, error) {
	if err := validateSinceTimestamp(since); err != nil {
		return nil, err
	}
	if since == "" {
		return entries, nil
	}
	sinceTime, _ := time.Parse(time.RFC3339, since)
	filtered := make([]logging.Entry, 0, len(entries))
	for _, entry := range entries {
		entryTime, err := time.Parse(time.RFC3339, entry.Timestamp)
		if err != nil {
			continue
		}
		if entryTime.Before(sinceTime) {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered, nil
}

func printLogEntry(e logging.Entry) {
	ts := e.Timestamp
	if len(ts) > 19 {
		ts = ts[:19]
	}

	levelStr := string(e.Level)
	switch e.Level {
	case logging.LevelError:
		levelStr = red(strings.ToUpper(levelStr))
	case logging.LevelWarn:
		levelStr = yellow(strings.ToUpper(levelStr))
	default:
		levelStr = dim(strings.ToUpper(levelStr))
	}

	line := fmt.Sprintf("  %s %-5s [%s]", dim(ts), levelStr, e.Component)

	if e.HookName != "" {
		line += fmt.Sprintf(" %s/%s", e.HookType, e.HookName)
	}
	if e.EventID != "" {
		line += fmt.Sprintf(" event=%s", e.EventID)
	}
	if e.Status != "" {
		switch e.Status {
		case "success":
			line += " " + green(e.Status)
		case "failed":
			line += " " + red(e.Status)
		case "timeout":
			line += " " + yellow(e.Status)
		default:
			line += " " + e.Status
		}
	}
	if e.DurationMs > 0 {
		line += fmt.Sprintf(" %dms", e.DurationMs)
	}

	line += " " + e.Message
	fmt.Println(line)
}
