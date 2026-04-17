package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/andrew8088/calvin/internal/logging"
)

func TestLogsFollowStreamsNewMatchingEntries(t *testing.T) {
	temp, env := isolatedEnv(t)
	seedLogs(t, temp, []logging.Entry{{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     logging.LevelInfo,
		Component: "hooks",
		HookName:  "my-hook",
		HookType:  "before-event-start",
		Message:   "initial match",
	}})

	binaryPath, repoRoot := buildCLIBinary(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, "logs", "-f", "--hook", "my-hook", "-n", "1")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), env...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("cmd.Start: %v", err)
	}

	waitForSubstring(t, &stdout, "initial match")
	appendLogs(t, temp, []logging.Entry{
		{
			Timestamp: time.Now().UTC().Add(time.Second).Format(time.RFC3339),
			Level:     logging.LevelInfo,
			Component: "hooks",
			HookName:  "other-hook",
			HookType:  "before-event-start",
			Message:   "ignored follow",
		},
		{
			Timestamp: time.Now().UTC().Add(2 * time.Second).Format(time.RFC3339),
			Level:     logging.LevelInfo,
			Component: "hooks",
			HookName:  "my-hook",
			HookType:  "before-event-start",
			Message:   "follow match",
		},
	})
	waitForSubstring(t, &stdout, "follow match")

	cancel()
	if err := cmd.Wait(); err != nil && ctx.Err() == nil {
		t.Fatalf("cmd.Wait: %v", err)
	}

	output := stdout.String()
	if strings.Contains(output, "ignored follow") {
		t.Fatalf("stdout = %q, did not expect non-matching follow entry", output)
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestLogsJSONFollowModeReturnsStructuredError(t *testing.T) {
	stdout, stderr, err := runCLI(t, "logs", "--json", "-f")
	if exitCode(err) == 0 {
		t.Fatalf("expected non-zero exit, stdout=%s stderr=%s", stdout, stderr)
	}
	if stdout != "" || !json.Valid([]byte(stderr)) || !strings.Contains(stderr, `"unsupported_follow_mode"`) {
		t.Fatalf("stdout=%s stderr=%s", stdout, stderr)
	}
}

func buildCLIBinary(t *testing.T) (string, string) {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "../.."))
	binaryPath := filepath.Join(t.TempDir(), "calvin-test")

	build := exec.Command("go", "build", "-o", binaryPath, ".")
	build.Dir = repoRoot
	build.Env = os.Environ()
	buildOutput, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("go build: %v\n%s", err, string(buildOutput))
	}

	return binaryPath, repoRoot
}

func appendLogs(t *testing.T, temp string, entries []logging.Entry) {
	t.Helper()

	path := filepath.Join(temp, "state", "calvin", "calvin.log")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("os.OpenFile: %v", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	for _, entry := range entries {
		if err := enc.Encode(entry); err != nil {
			t.Fatalf("enc.Encode: %v", err)
		}
	}
}

func waitForSubstring(t *testing.T, buf *bytes.Buffer, want string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(buf.String(), want) {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %q in %q", want, buf.String())
}
