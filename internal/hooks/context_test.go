package hooks

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/andrew8088/calvin/internal/calendar"
)

func TestWriteEventContextFile_RoundTrip(t *testing.T) {
	payload := calendar.HookPayload{
		SchemaVersion: 1,
		ID:            "evt-1",
		Title:         "Team Sync",
		Start:         "2026-04-17T10:00:00Z",
		End:           "2026-04-17T10:30:00Z",
		Calendar:      "primary",
		HookType:      "before-event-start",
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	path, cleanup, err := WriteEventContextFile(raw)
	if err != nil {
		t.Fatalf("WriteEventContextFile: %v", err)
	}
	defer cleanup()

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("event context file does not exist: %v", err)
	}

	loaded, err := LoadEventContextFile(path)
	if err != nil {
		t.Fatalf("LoadEventContextFile: %v", err)
	}

	if loaded.ID != payload.ID {
		t.Fatalf("loaded ID = %q, want %q", loaded.ID, payload.ID)
	}
	if loaded.Title != payload.Title {
		t.Fatalf("loaded Title = %q, want %q", loaded.Title, payload.Title)
	}
	if loaded.HookType != payload.HookType {
		t.Fatalf("loaded HookType = %q, want %q", loaded.HookType, payload.HookType)
	}
}

func TestWriteEventContextFile_CleanupRemovesFile(t *testing.T) {
	path, cleanup, err := WriteEventContextFile([]byte(`{"id":"evt-1"}`))
	if err != nil {
		t.Fatalf("WriteEventContextFile: %v", err)
	}

	cleanup()

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected file removal, stat err=%v", err)
	}
}

func TestLoadEventContextFile_InvalidJSON(t *testing.T) {
	f, err := os.CreateTemp("", "calvin-event-invalid-*.json")
	if err != nil {
		t.Fatalf("os.CreateTemp: %v", err)
	}
	path := f.Name()
	t.Cleanup(func() { _ = os.Remove(path) })

	if _, err := f.WriteString("{not-json"); err != nil {
		t.Fatalf("WriteString: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if _, err := LoadEventContextFile(path); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
