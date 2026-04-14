package hooks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andrew8088/calvin/internal/logging"
)

func init() {
	logging.InitStdout()
}

func setupHooksDir(t *testing.T) string {
	t.Helper()
	base := t.TempDir()

	for _, ht := range ValidTypes {
		os.MkdirAll(filepath.Join(base, ht), 0755)
	}
	return base
}

func createHook(t *testing.T, base, hookType, name string) string {
	t.Helper()
	path := filepath.Join(base, hookType, name)
	os.WriteFile(path, []byte("#!/bin/sh\necho ok"), 0755)
	return path
}

func TestDiscoverFrom_Empty(t *testing.T) {
	base := setupHooksDir(t)
	result, err := DiscoverFrom(base)
	if err != nil {
		t.Fatalf("DiscoverFrom: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d types", len(result))
	}
}

func TestDiscoverFrom_FindsHooks(t *testing.T) {
	base := setupHooksDir(t)
	createHook(t, base, "pre_event", "notify")
	createHook(t, base, "event_start", "open-link")
	createHook(t, base, "event_end", "clear-status")

	result, err := DiscoverFrom(base)
	if err != nil {
		t.Fatalf("DiscoverFrom: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 hook types, got %d", len(result))
	}
	if len(result["pre_event"]) != 1 {
		t.Errorf("expected 1 pre_event hook, got %d", len(result["pre_event"]))
	}
	if result["pre_event"][0].Name != "notify" {
		t.Errorf("hook name = %q, want 'notify'", result["pre_event"][0].Name)
	}
}

func TestDiscoverFrom_SkipsNonExecutable(t *testing.T) {
	base := setupHooksDir(t)

	path := filepath.Join(base, "pre_event", "not-executable")
	os.WriteFile(path, []byte("#!/bin/sh\necho ok"), 0644)

	result, _ := DiscoverFrom(base)
	if len(result["pre_event"]) != 0 {
		t.Error("non-executable files should be skipped")
	}
}

func TestDiscoverFrom_SkipsDotfiles(t *testing.T) {
	base := setupHooksDir(t)
	createHook(t, base, "pre_event", ".hidden")

	result, _ := DiscoverFrom(base)
	if len(result["pre_event"]) != 0 {
		t.Error("dotfiles should be skipped")
	}
}

func TestDiscoverFrom_SkipsDirectories(t *testing.T) {
	base := setupHooksDir(t)
	os.MkdirAll(filepath.Join(base, "pre_event", "subdir"), 0755)

	result, _ := DiscoverFrom(base)
	if len(result["pre_event"]) != 0 {
		t.Error("directories should be skipped")
	}
}

func TestDiscoverFrom_SortedAlphabetically(t *testing.T) {
	base := setupHooksDir(t)
	createHook(t, base, "pre_event", "z-last")
	createHook(t, base, "pre_event", "a-first")
	createHook(t, base, "pre_event", "m-middle")

	result, _ := DiscoverFrom(base)
	hooks := result["pre_event"]
	if len(hooks) != 3 {
		t.Fatalf("expected 3 hooks, got %d", len(hooks))
	}
	if hooks[0].Name != "a-first" || hooks[1].Name != "m-middle" || hooks[2].Name != "z-last" {
		t.Errorf("hooks not sorted: %v, %v, %v", hooks[0].Name, hooks[1].Name, hooks[2].Name)
	}
}

func TestDiscoverFrom_MissingTypeDir(t *testing.T) {
	base := t.TempDir()

	result, err := DiscoverFrom(base)
	if err != nil {
		t.Fatalf("DiscoverFrom with missing dirs: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result for missing dirs, got %d", len(result))
	}
}

func TestDiscoverFrom_HookPath(t *testing.T) {
	base := setupHooksDir(t)
	createHook(t, base, "event_start", "my-hook")

	result, _ := DiscoverFrom(base)
	hook := result["event_start"][0]

	expected := filepath.Join(base, "event_start", "my-hook")
	if hook.Path != expected {
		t.Errorf("Path = %q, want %q", hook.Path, expected)
	}
	if hook.Type != "event_start" {
		t.Errorf("Type = %q, want 'event_start'", hook.Type)
	}
}

func TestValidTypes(t *testing.T) {
	expected := map[string]bool{"pre_event": true, "event_start": true, "event_end": true}
	for _, vt := range ValidTypes {
		if !expected[vt] {
			t.Errorf("unexpected valid type: %q", vt)
		}
	}
	if len(ValidTypes) != 3 {
		t.Errorf("expected 3 valid types, got %d", len(ValidTypes))
	}
}
