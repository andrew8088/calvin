package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/andrew8088/calvin/internal/calendar"
	"github.com/andrew8088/calvin/internal/db"
	"github.com/andrew8088/calvin/internal/logging"
)

func TestVersionCommandJSONMode(t *testing.T) {
	stdout, stderr, err := runCLI(t, "version", "--json")
	if err != nil {
		t.Fatalf("runCLI: %v, stderr=%s", err, stderr)
	}
	if !json.Valid([]byte(stdout)) || stderr != "" {
		t.Fatalf("stdout=%s stderr=%s", stdout, stderr)
	}
}

func TestVersionEnvJSONEmitsJSON(t *testing.T) {
	stdout, stderr, err := runCLIWithTempEnv(t, []string{"CALVIN_OUTPUT=json"}, "version")
	if err != nil || !json.Valid([]byte(stdout)) || stderr != "" {
		t.Fatalf("stdout=%s stderr=%s err=%v", stdout, stderr, err)
	}
}

func TestRootJSONModeUnknownCommandWritesStructuredError(t *testing.T) {
	stdout, stderr, err := runCLI(t, "nope", "--json")
	if exitCode(err) == 0 || stdout != "" || !json.Valid([]byte(stderr)) {
		t.Fatalf("stdout=%s stderr=%s", stdout, stderr)
	}
}

func TestRootEnvJSONUnknownCommandWritesStructuredError(t *testing.T) {
	stdout, stderr, err := runCLIWithTempEnv(t, []string{"CALVIN_OUTPUT=json"}, "nope")
	if exitCode(err) == 0 || stdout != "" || !json.Valid([]byte(stderr)) {
		t.Fatalf("stdout=%s stderr=%s", stdout, stderr)
	}
}

func TestRootJSONModeInvalidOutputWritesStructuredError(t *testing.T) {
	stdout, stderr, err := runCLI(t, "version", "--json", "--output", "yaml")
	if exitCode(err) == 0 || stdout != "" || !json.Valid([]byte(stderr)) {
		t.Fatalf("stdout=%s stderr=%s", stdout, stderr)
	}
}

func TestRootJSONModeUsageErrorWritesStructuredError(t *testing.T) {
	stdout, stderr, err := runCLI(t, "hooks", "new", "--json")
	if exitCode(err) == 0 || stdout != "" || !json.Valid([]byte(stderr)) {
		t.Fatalf("stdout=%s stderr=%s", stdout, stderr)
	}
}

func TestAuthJSONModeUnsupportedInteractiveFlow(t *testing.T) {
	stdout, stderr, err := runCLI(t, "auth", "--json")
	if exitCode(err) == 0 || stdout != "" || !json.Valid([]byte(stderr)) {
		t.Fatalf("stdout=%s stderr=%s", stdout, stderr)
	}
}

func TestStartJSONModeUnsupportedForegroundMode(t *testing.T) {
	stdout, stderr, err := runCLIWithTempEnv(t, nil, "start", "--json")
	if exitCode(err) == 0 || stdout != "" || !json.Valid([]byte(stderr)) {
		t.Fatalf("stdout=%s stderr=%s", stdout, stderr)
	}
}

func TestStopJSONModeNoDaemon(t *testing.T) {
	stdout, stderr, err := runCLIWithTempEnv(t, nil, "stop", "--json")
	if err != nil || !json.Valid([]byte(stdout)) || stderr != "" {
		t.Fatalf("stdout=%s stderr=%s err=%v", stdout, stderr, err)
	}
}

func TestSyncJSONModeNoDaemon(t *testing.T) {
	stdout, stderr, err := runCLIWithTempEnv(t, nil, "sync", "--json")
	if err != nil || !json.Valid([]byte(stdout)) || stderr != "" {
		t.Fatalf("stdout=%s stderr=%s err=%v", stdout, stderr, err)
	}
}

func TestStatusJSONModeReportsConfiguredCalendarsAndInterval(t *testing.T) {
	temp, env := isolatedEnv(t)

	configDir := filepath.Join(temp, "config", "calvin")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll: %v", err)
	}
	configData := `sync_interval_seconds = 123

[[calendars]]
id = "work@company.com"
`
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(configData), 0o644); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}

	dataDir := filepath.Join(temp, "data", "calvin")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll: %v", err)
	}
	database, err := db.Open(filepath.Join(dataDir, "events.db"), false)
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	if err := database.SetSyncToken("work@company.com", "token-work"); err != nil {
		t.Fatalf("database.SetSyncToken: %v", err)
	}
	if err := database.Close(); err != nil {
		t.Fatalf("database.Close: %v", err)
	}

	stdout, stderr, err := runCLIWithEnv(t, env, "status", "--json")
	if err != nil || stderr != "" {
		t.Fatalf("stdout=%s stderr=%s err=%v", stdout, stderr, err)
	}

	var result struct {
		OK   bool `json:"ok"`
		Data struct {
			SyncToken           bool `json:"sync_token"`
			SyncIntervalSeconds int  `json:"sync_interval_seconds"`
			SyncCalendars       []struct {
				ID        string `json:"id"`
				SyncToken bool   `json:"sync_token"`
			} `json:"sync_calendars"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected ok result, got %s", stdout)
	}
	if !result.Data.SyncToken {
		t.Fatalf("expected aggregate sync_token=true, got false in %s", stdout)
	}
	if result.Data.SyncIntervalSeconds != 123 {
		t.Fatalf("sync_interval_seconds = %d, want 123", result.Data.SyncIntervalSeconds)
	}
	if len(result.Data.SyncCalendars) != 1 {
		t.Fatalf("expected 1 sync calendar, got %d", len(result.Data.SyncCalendars))
	}
	if result.Data.SyncCalendars[0].ID != "work@company.com" {
		t.Fatalf("calendar id = %q, want work@company.com", result.Data.SyncCalendars[0].ID)
	}
	if !result.Data.SyncCalendars[0].SyncToken {
		t.Fatalf("expected calendar sync token to be true, got false in %s", stdout)
	}
}

func TestLogsJSONModeReturnsStructuredEntries(t *testing.T) {
	temp, env := isolatedEnv(t)
	seedLogs(t, temp, []logging.Entry{{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     logging.LevelInfo,
		Component: "sync",
		Message:   "seed log",
	}})

	stdout, stderr, err := runCLIWithEnv(t, env, "logs", "--json")
	if err != nil || !json.Valid([]byte(stdout)) || stderr != "" || !containsJSONKey(stdout, "ok") {
		t.Fatalf("stdout=%s stderr=%s err=%v", stdout, stderr, err)
	}
}

func TestEventDetailJSONModeReturnsStructuredDetail(t *testing.T) {
	temp, env := isolatedEnv(t)
	seedEventDB(t, temp, calendar.Event{
		ID:       "evt-123",
		Title:    "Weekly Standup",
		Start:    time.Now().Add(30 * time.Minute),
		End:      time.Now().Add(60 * time.Minute),
		Calendar: "primary",
		Status:   "confirmed",
	})

	stdout, stderr, err := runCLIWithEnv(t, env, "events", "evt-123", "--json")
	if err != nil || !json.Valid([]byte(stdout)) || stderr != "" || !containsJSONKey(stdout, "ok") {
		t.Fatalf("stdout=%s stderr=%s err=%v", stdout, stderr, err)
	}
}

func TestDoctorJSONModeReturnsStructuredChecks(t *testing.T) {
	stdout, stderr, err := runCLIWithTempEnv(t, nil, "doctor", "--json")
	if err != nil || !json.Valid([]byte(stdout)) || stderr != "" || !containsJSONKey(stdout, "ok") {
		t.Fatalf("stdout=%s stderr=%s err=%v", stdout, stderr, err)
	}
}

func TestHooksNewJSONModeInvalidTypeWritesStructuredError(t *testing.T) {
	stdout, stderr, err := runCLIWithTempEnv(t, nil, "hooks", "new", "before-start", "my-hook", "--json")
	if exitCode(err) == 0 || stdout != "" || !json.Valid([]byte(stderr)) {
		t.Fatalf("stdout=%s stderr=%s", stdout, stderr)
	}
}

func TestHooksNewJSONModeRejectsTraversalName(t *testing.T) {
	stdout, stderr, err := runCLIWithTempEnv(t, nil, "hooks", "new", "before-event-start", "../../.ssh", "--json")
	if exitCode(err) == 0 || stdout != "" || !json.Valid([]byte(stderr)) {
		t.Fatalf("stdout=%s stderr=%s", stdout, stderr)
	}
}

func TestIgnoreJSONModePreservesExitSemantics(t *testing.T) {
	stdout, stderr, err := runCLIWithEventContext(t, testHookPayload(), "ignore", "--title", "*retro*", "--json")
	if exitCode(err) != 1 || stderr != "" || !json.Valid([]byte(stdout)) || !containsJSONKey(stdout, "exit_code") {
		t.Fatalf("stdout=%s stderr=%s", stdout, stderr)
	}
}

func TestTestCommandJSONModeReturnsStructuredResult(t *testing.T) {
	temp, env := isolatedEnv(t)
	seedHook(t, temp, "before-event-start", "example-json", "#!/bin/sh\necho hook-ran\n")

	stdout, stderr, err := runCLIWithEnv(t, env, "test", "example-json", "--json")
	if err != nil || stderr != "" || !json.Valid([]byte(stdout)) || !containsJSONKey(stdout, "ok") {
		t.Fatalf("stdout=%s stderr=%s err=%v", stdout, stderr, err)
	}
}

func containsJSONKey(s, key string) bool {
	return json.Valid([]byte(s)) && strings.Contains(s, `"`+key+`"`)
}

func seedLogs(t *testing.T, temp string, entries []logging.Entry) {
	t.Helper()
	logDir := filepath.Join(temp, "state", "calvin")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll: %v", err)
	}
	path := filepath.Join(logDir, "calvin.log")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("os.Create: %v", err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, entry := range entries {
		if err := enc.Encode(entry); err != nil {
			t.Fatalf("enc.Encode: %v", err)
		}
	}
}

func seedEventDB(t *testing.T, temp string, event calendar.Event) {
	t.Helper()
	dataDir := filepath.Join(temp, "data", "calvin")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll: %v", err)
	}
	database, err := db.Open(filepath.Join(dataDir, "events.db"), false)
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	defer database.Close()
	if err := database.UpsertEvent(event, 1); err != nil {
		t.Fatalf("database.UpsertEvent: %v", err)
	}
}

func seedHook(t *testing.T, temp, hookType, name, content string) {
	t.Helper()
	hookDir := filepath.Join(temp, "config", "calvin", "hooks", hookType)
	if err := os.MkdirAll(hookDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll: %v", err)
	}
	hookPath := filepath.Join(hookDir, name)
	if err := os.WriteFile(hookPath, []byte(content), 0o755); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}
}
