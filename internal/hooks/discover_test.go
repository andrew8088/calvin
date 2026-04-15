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
	createHook(t, base, "before-event-start", "notify")
	createHook(t, base, "on-event-start", "open-link")
	createHook(t, base, "on-event-end", "clear-status")

	result, err := DiscoverFrom(base)
	if err != nil {
		t.Fatalf("DiscoverFrom: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 hook types, got %d", len(result))
	}
	if len(result["before-event-start"]) != 1 {
		t.Errorf("expected 1 before-event-start hook, got %d", len(result["before-event-start"]))
	}
	if result["before-event-start"][0].Name != "notify" {
		t.Errorf("hook name = %q, want 'notify'", result["before-event-start"][0].Name)
	}
}

func TestDiscoverFrom_SkipsNonExecutable(t *testing.T) {
	base := setupHooksDir(t)

	path := filepath.Join(base, "before-event-start", "not-executable")
	os.WriteFile(path, []byte("#!/bin/sh\necho ok"), 0644)

	result, _ := DiscoverFrom(base)
	if len(result["before-event-start"]) != 0 {
		t.Error("non-executable files should be skipped")
	}
}

func TestDiscoverFrom_SkipsDotfiles(t *testing.T) {
	base := setupHooksDir(t)
	createHook(t, base, "before-event-start", ".hidden")

	result, _ := DiscoverFrom(base)
	if len(result["before-event-start"]) != 0 {
		t.Error("dotfiles should be skipped")
	}
}

func TestDiscoverFrom_SkipsDirectories(t *testing.T) {
	base := setupHooksDir(t)
	os.MkdirAll(filepath.Join(base, "before-event-start", "subdir"), 0755)

	result, _ := DiscoverFrom(base)
	if len(result["before-event-start"]) != 0 {
		t.Error("directories should be skipped")
	}
}

func TestDiscoverFrom_SortedAlphabetically(t *testing.T) {
	base := setupHooksDir(t)
	createHook(t, base, "before-event-start", "z-last")
	createHook(t, base, "before-event-start", "a-first")
	createHook(t, base, "before-event-start", "m-middle")

	result, _ := DiscoverFrom(base)
	hooks := result["before-event-start"]
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
	createHook(t, base, "on-event-start", "my-hook")

	result, _ := DiscoverFrom(base)
	hook := result["on-event-start"][0]

	expected := filepath.Join(base, "on-event-start", "my-hook")
	if hook.Path != expected {
		t.Errorf("Path = %q, want %q", hook.Path, expected)
	}
	if hook.Type != "on-event-start" {
		t.Errorf("Type = %q, want 'on-event-start'", hook.Type)
	}
}

func TestValidTypes(t *testing.T) {
	expected := map[string]bool{"before-event-start": true, "on-event-start": true, "on-event-end": true}
	for _, vt := range ValidTypes {
		if !expected[vt] {
			t.Errorf("unexpected valid type: %q", vt)
		}
	}
	if len(ValidTypes) != 3 {
		t.Errorf("expected 3 valid types, got %d", len(ValidTypes))
	}
}
