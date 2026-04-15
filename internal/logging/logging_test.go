package logging

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func newTestLogger() (*Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	l := &Logger{writer: &buf}
	return l, &buf
}

func TestLog_JSONFormat(t *testing.T) {
	l, buf := newTestLogger()
	l.Info("test", "hello world")

	var entry Entry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse log entry: %v\nraw: %s", err, buf.String())
	}

	if entry.Level != LevelInfo {
		t.Errorf("level = %q, want 'info'", entry.Level)
	}
	if entry.Component != "test" {
		t.Errorf("component = %q, want 'test'", entry.Component)
	}
	if entry.Message != "hello world" {
		t.Errorf("message = %q", entry.Message)
	}
	if entry.Timestamp == "" {
		t.Error("expected non-empty timestamp")
	}
}

func TestLog_Levels(t *testing.T) {
	tests := []struct {
		name  string
		fn    func(*Logger)
		level Level
	}{
		{"info", func(l *Logger) { l.Info("c", "m") }, LevelInfo},
		{"warn", func(l *Logger) { l.Warn("c", "m") }, LevelWarn},
		{"error", func(l *Logger) { l.Error("c", "m") }, LevelError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, buf := newTestLogger()
			tt.fn(l)

			var entry Entry
			json.Unmarshal(buf.Bytes(), &entry)
			if entry.Level != tt.level {
				t.Errorf("level = %q, want %q", entry.Level, tt.level)
			}
		})
	}
}

func TestHookEvent(t *testing.T) {
	l, buf := newTestLogger()
	l.HookEvent(LevelInfo, "my-hook", "before-event-start", "evt-1", "success", "completed", 150)

	var entry Entry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if entry.Component != "hooks" {
		t.Errorf("component = %q, want 'hooks'", entry.Component)
	}
	if entry.HookName != "my-hook" {
		t.Errorf("hook_name = %q", entry.HookName)
	}
	if entry.HookType != "before-event-start" {
		t.Errorf("hook_type = %q", entry.HookType)
	}
	if entry.EventID != "evt-1" {
		t.Errorf("event_id = %q", entry.EventID)
	}
	if entry.Status != "success" {
		t.Errorf("status = %q", entry.Status)
	}
	if entry.DurationMs != 150 {
		t.Errorf("duration_ms = %d, want 150", entry.DurationMs)
	}
}

func TestLog_OmitsEmptyFields(t *testing.T) {
	l, buf := newTestLogger()
	l.Info("test", "basic log")

	raw := buf.String()
	if strings.Contains(raw, "event_id") {
		t.Error("empty event_id should be omitted")
	}
	if strings.Contains(raw, "hook_name") {
		t.Error("empty hook_name should be omitted")
	}
	if strings.Contains(raw, "duration_ms") {
		t.Error("zero duration_ms should be omitted")
	}
}

func TestLog_MultipleEntries(t *testing.T) {
	l, buf := newTestLogger()
	l.Info("a", "first")
	l.Warn("b", "second")
	l.Error("c", "third")

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 log lines, got %d", len(lines))
	}

	for i, line := range lines {
		var entry Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Errorf("line %d: parse error: %v", i, err)
		}
	}
}

func TestGet_DefaultsToStdout(t *testing.T) {
	old := defaultLogger
	defaultLogger = nil
	defer func() { defaultLogger = old }()

	l := Get()
	if l == nil {
		t.Fatal("Get() returned nil")
	}
}
