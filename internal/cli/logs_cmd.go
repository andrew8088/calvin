package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/andrew8088/calvin/internal/config"
	"github.com/andrew8088/calvin/internal/logging"
	"github.com/spf13/cobra"
)

var (
	logsHook   string
	logsEvent  string
	logsType   string
	logsLevel  string
	logsSince  string
	logsLines  int
	logsFollow bool
)

const logsFollowPollInterval = 100 * time.Millisecond

var logsCmd = &cobra.Command{
	Use:     "logs",
	Short:   "Show Calvin daemon logs",
	Example: "  calvin logs\n  calvin logs --hook my-hook\n  calvin logs --level error\n  calvin logs -n 50\n  calvin logs -f",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLogs(cmd.Context())
	},
}

func init() {
	logsCmd.Flags().StringVar(&logsHook, "hook", "", "Filter by hook name")
	logsCmd.Flags().StringVar(&logsEvent, "event", "", "Filter by event ID")
	logsCmd.Flags().StringVar(&logsType, "type", "", "Filter by hook type")
	logsCmd.Flags().StringVar(&logsLevel, "level", "", "Filter by log level (info, warn, error)")
	logsCmd.Flags().StringVar(&logsSince, "since", "", "Filter entries since timestamp")
	logsCmd.Flags().IntVarP(&logsLines, "lines", "n", 100, "Number of lines to show")
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow new matching log entries")
}

func runLogs(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if wantsJSON() && logsFollow {
		return newExitError(2, "logs", "unsupported_follow_mode", "logs --json does not support --follow", nil, nil)
	}
	if err := validateSinceTimestamp(logsSince); err != nil {
		return newExitError(2, "logs", "invalid_since", err.Error(), nil, err)
	}

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

	entries, offset, err := readLogEntriesFromOffset(logPath, 0)
	if err != nil {
		return err
	}
	entries, err = filterMatchingEntries(entries)
	if err != nil {
		return newExitError(2, "logs", "invalid_since", err.Error(), nil, err)
	}

	if len(entries) == 0 {
		if logsFollow {
			return followLogs(ctx, logPath, offset)
		}
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

	if logsFollow {
		return followLogs(ctx, logPath, offset)
	}

	return nil
}

func readLogEntriesFromOffset(path string, offset int64) ([]logging.Entry, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, offset, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, offset, err
	}
	if info.Size() < offset {
		offset = 0
	}
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return nil, offset, err
	}

	entries, err := scanLogEntries(f)
	if err != nil {
		return nil, offset, err
	}
	nextOffset, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, offset, err
	}

	return entries, nextOffset, nil
}

func scanLogEntries(r io.Reader) ([]logging.Entry, error) {
	var entries []logging.Entry
	scanner := bufio.NewScanner(r)
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
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func filterMatchingEntries(entries []logging.Entry) ([]logging.Entry, error) {
	entries, err := filterLogsSince(entries, logsSince)
	if err != nil {
		return nil, err
	}
	filtered := make([]logging.Entry, 0, len(entries))
	for _, entry := range entries {
		if !matchesFilter(entry) {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered, nil
}

func followLogs(ctx context.Context, logPath string, offset int64) error {
	ticker := time.NewTicker(logsFollowPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			entries, nextOffset, err := readLogEntriesFromOffset(logPath, offset)
			if err != nil {
				return err
			}
			offset = nextOffset
			entries, err = filterMatchingEntries(entries)
			if err != nil {
				return newExitError(2, "logs", "invalid_since", err.Error(), nil, err)
			}
			for _, entry := range entries {
				printLogEntry(entry)
			}
		}
	}
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
