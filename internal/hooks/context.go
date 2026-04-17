package hooks

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/andrew8088/calvin/internal/calendar"
)

func WriteEventContextFile(payload []byte) (string, func(), error) {
	f, err := os.CreateTemp("", "calvin-event-*.json")
	if err != nil {
		return "", func() {}, fmt.Errorf("create event context file: %w", err)
	}

	path := f.Name()
	cleanup := func() {
		_ = os.Remove(path)
	}

	if _, err := f.Write(payload); err != nil {
		_ = f.Close()
		cleanup()
		return "", func() {}, fmt.Errorf("write event context file: %w", err)
	}

	if err := f.Close(); err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("close event context file: %w", err)
	}

	return path, cleanup, nil
}

func LoadEventContextFile(path string) (calendar.HookPayload, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return calendar.HookPayload{}, fmt.Errorf("read event context file %q: %w", path, err)
	}

	var payload calendar.HookPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return calendar.HookPayload{}, fmt.Errorf("parse event context file %q: %w", path, err)
	}

	return payload, nil
}
