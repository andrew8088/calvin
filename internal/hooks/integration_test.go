package hooks

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/andrew8088/calvin/internal/calendar"
	"github.com/andrew8088/calvin/internal/config"
	"github.com/andrew8088/calvin/internal/db"
	"github.com/andrew8088/calvin/internal/logging"
)

func openTestDB(t *testing.T) *db.DB {
	t.Helper()
	d, err := db.Open(":memory:", false)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

func writeScript(t *testing.T, dir, name, content string) Hook {
	t.Helper()
	path := filepath.Join(dir, name)
	os.WriteFile(path, []byte(content), 0755)
	return Hook{Name: name, Type: "pre_event", Path: path}
}

func testCfg() *config.Config {
	cfg := config.Default()
	cfg.HookTimeoutSeconds = 5
	cfg.MaxConcurrentHooks = 2
	cfg.HookOutputMaxBytes = 1024
	return cfg
}

func testEvent() calendar.Event {
	return calendar.Event{
		ID:    "evt-1",
		Title: "Test Meeting",
		Start: time.Date(2026, 4, 14, 10, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 4, 14, 11, 0, 0, 0, time.UTC),
	}
}

func TestFireHooks_Success(t *testing.T) {
	logging.InitStdout()
	d := openTestDB(t)
	dir := t.TempDir()

	hook := writeScript(t, dir, "echo-hook", "#!/bin/sh\necho hello")
	executor := NewExecutor(testCfg(), d)

	results := executor.FireHooks(context.Background(), testEvent(), "pre_event", []Hook{hook}, nil, nil)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "success" {
		t.Errorf("status = %q, want 'success'", results[0].Status)
	}
	if results[0].Stdout != "hello\n" {
		t.Errorf("stdout = %q, want 'hello\\n'", results[0].Stdout)
	}
	if results[0].EventID != "evt-1" {
		t.Errorf("EventID = %q", results[0].EventID)
	}
}

func TestFireHooks_Failure(t *testing.T) {
	logging.InitStdout()
	d := openTestDB(t)
	dir := t.TempDir()

	hook := writeScript(t, dir, "fail-hook", "#!/bin/sh\necho oops >&2\nexit 1")
	executor := NewExecutor(testCfg(), d)

	results := executor.FireHooks(context.Background(), testEvent(), "pre_event", []Hook{hook}, nil, nil)

	if results[0].Status != "failed" {
		t.Errorf("status = %q, want 'failed'", results[0].Status)
	}
	if results[0].Stderr != "oops\n" {
		t.Errorf("stderr = %q, want 'oops\\n'", results[0].Stderr)
	}
	if results[0].Err == nil {
		t.Error("expected non-nil Err")
	}
}

func TestFireHooks_Timeout(t *testing.T) {
	logging.InitStdout()
	d := openTestDB(t)
	dir := t.TempDir()

	cfg := testCfg()
	cfg.HookTimeoutSeconds = 1

	hook := writeScript(t, dir, "slow-hook", "#!/bin/sh\nwhile true; do :; done")
	executor := NewExecutor(cfg, d)

	results := executor.FireHooks(context.Background(), testEvent(), "pre_event", []Hook{hook}, nil, nil)

	if results[0].Status != "timeout" {
		t.Errorf("status = %q, want 'timeout'", results[0].Status)
	}
}

func TestFireHooks_Dedup(t *testing.T) {
	logging.InitStdout()
	d := openTestDB(t)
	dir := t.TempDir()

	hook := writeScript(t, dir, "dedup-hook", "#!/bin/sh\necho first")
	executor := NewExecutor(testCfg(), d)
	event := testEvent()

	results := executor.FireHooks(context.Background(), event, "pre_event", []Hook{hook}, nil, nil)
	if results[0].Status != "success" {
		t.Fatalf("first run: status = %q", results[0].Status)
	}

	results = executor.FireHooks(context.Background(), event, "pre_event", []Hook{hook}, nil, nil)
	if results[0].Status != "skipped" {
		t.Errorf("second run: status = %q, want 'skipped'", results[0].Status)
	}
}

func TestFireHooks_ReceivesStdin(t *testing.T) {
	logging.InitStdout()
	d := openTestDB(t)
	dir := t.TempDir()

	hook := writeScript(t, dir, "stdin-hook", "#!/bin/sh\ncat | jq -r '.title'")
	executor := NewExecutor(testCfg(), d)

	results := executor.FireHooks(context.Background(), testEvent(), "pre_event", []Hook{hook}, nil, nil)

	if results[0].Status != "success" {
		t.Fatalf("status = %q, stderr = %q", results[0].Status, results[0].Stderr)
	}
	if results[0].Stdout != "Test Meeting\n" {
		t.Errorf("stdout = %q, want 'Test Meeting\\n'", results[0].Stdout)
	}
}

func TestFireHooks_MultipleHooks(t *testing.T) {
	logging.InitStdout()
	d := openTestDB(t)
	dir := t.TempDir()

	var hks []Hook
	for i := 0; i < 4; i++ {
		name := "hook-" + string(rune('a'+i))
		hks = append(hks, writeScript(t, dir, name, "#!/bin/sh\necho ok"))
	}

	cfg := testCfg()
	cfg.MaxConcurrentHooks = 1
	executor := NewExecutor(cfg, d)

	results := executor.FireHooks(context.Background(), testEvent(), "pre_event", hks, nil, nil)

	successCount := 0
	for _, r := range results {
		if r.Status == "success" {
			successCount++
		}
	}
	if successCount != 4 {
		t.Errorf("expected 4 successes, got %d", successCount)
	}
}

func TestFireHooks_RecordsExecution(t *testing.T) {
	logging.InitStdout()
	d := openTestDB(t)
	dir := t.TempDir()

	hook := writeScript(t, dir, "record-hook", "#!/bin/sh\necho recorded")
	executor := NewExecutor(testCfg(), d)

	executor.FireHooks(context.Background(), testEvent(), "pre_event", []Hook{hook}, nil, nil)

	execs, err := d.GetHookExecutions("evt-1")
	if err != nil {
		t.Fatalf("GetHookExecutions: %v", err)
	}
	if len(execs) != 1 {
		t.Fatalf("expected 1 execution record, got %d", len(execs))
	}
	if execs[0].Status != "success" {
		t.Errorf("recorded status = %q", execs[0].Status)
	}
	if execs[0].HookName != "record-hook" {
		t.Errorf("recorded hook name = %q", execs[0].HookName)
	}
}
