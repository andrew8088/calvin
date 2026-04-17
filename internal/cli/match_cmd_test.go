package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/andrew8088/calvin/internal/calendar"
	"github.com/andrew8088/calvin/internal/hooks"
)

func writeEventContextFile(t *testing.T, payload calendar.HookPayload) string {
	t.Helper()

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	f, err := os.CreateTemp("", "calvin-cli-match-*.json")
	if err != nil {
		t.Fatalf("os.CreateTemp: %v", err)
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		t.Fatalf("Write: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(f.Name()) })

	return f.Name()
}

func testHookPayload() calendar.HookPayload {
	return calendar.HookPayload{
		Title:     "Weekly Standup",
		Calendar:  "primary",
		Organizer: "alice@example.com",
		HookType:  "before-event-start",
		Attendees: []calendar.Attendee{{Email: "a@example.com"}, {Email: "b@example.com"}},
	}
}

func TestRunHookFilter_MatchAndMismatch(t *testing.T) {
	eventFile := writeEventContextFile(t, testHookPayload())

	var stderr bytes.Buffer
	code := runHookFilter(hooks.MatchCriteria{TitlePatterns: []string{"*standup*"}}, eventFile, false, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0, stderr=%q", code, stderr.String())
	}

	stderr.Reset()
	code = runHookFilter(hooks.MatchCriteria{TitlePatterns: []string{"*retro*"}}, eventFile, false, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1, stderr=%q", code, stderr.String())
	}
}

func TestRunHookFilter_UsesEnvContextFile(t *testing.T) {
	eventFile := writeEventContextFile(t, testHookPayload())
	t.Setenv("CALVIN_EVENT_FILE", eventFile)

	var stderr bytes.Buffer
	code := runHookFilter(hooks.MatchCriteria{CalendarPatterns: []string{"primary"}}, "", false, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0, stderr=%q", code, stderr.String())
	}
}

func TestRunHookFilter_ErrorsReturnCode2(t *testing.T) {
	eventFile := writeEventContextFile(t, testHookPayload())

	t.Run("no criteria", func(t *testing.T) {
		var stderr bytes.Buffer
		code := runHookFilter(hooks.MatchCriteria{}, eventFile, false, &stderr)
		if code != 2 {
			t.Fatalf("code = %d, want 2", code)
		}
		if !strings.Contains(stderr.String(), "at least one filter") {
			t.Fatalf("stderr = %q", stderr.String())
		}
	})

	t.Run("missing context", func(t *testing.T) {
		t.Setenv("CALVIN_EVENT_FILE", "")
		var stderr bytes.Buffer
		code := runHookFilter(hooks.MatchCriteria{TitlePatterns: []string{"*standup*"}}, "", false, &stderr)
		if code != 2 {
			t.Fatalf("code = %d, want 2", code)
		}
		if !strings.Contains(stderr.String(), "event context") {
			t.Fatalf("stderr = %q", stderr.String())
		}
	})

	t.Run("invalid pattern", func(t *testing.T) {
		var stderr bytes.Buffer
		code := runHookFilter(hooks.MatchCriteria{TitlePatterns: []string{"["}}, eventFile, false, &stderr)
		if code != 2 {
			t.Fatalf("code = %d, want 2", code)
		}
		if !strings.Contains(stderr.String(), "invalid") {
			t.Fatalf("stderr = %q", stderr.String())
		}
	})
}

func TestRunHookFilter_WhyPrintsReasons(t *testing.T) {
	eventFile := writeEventContextFile(t, testHookPayload())

	var stderr bytes.Buffer
	code := runHookFilter(hooks.MatchCriteria{TitlePatterns: []string{"*retro*"}}, eventFile, true, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "title") {
		t.Fatalf("stderr = %q, expected title reason", stderr.String())
	}
}
