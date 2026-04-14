package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/andrew8088/calvin/internal/calendar"
	"github.com/andrew8088/calvin/internal/config"
	"github.com/andrew8088/calvin/internal/db"
	"github.com/andrew8088/calvin/internal/logging"
)

type Executor struct {
	cfg       *config.Config
	database  *db.DB
	semaphore chan struct{}
	dbmu      sync.Mutex
}

func NewExecutor(cfg *config.Config, database *db.DB) *Executor {
	return &Executor{
		cfg:       cfg,
		database:  database,
		semaphore: make(chan struct{}, cfg.MaxConcurrentHooks),
	}
}

type Result struct {
	HookName   string
	HookType   string
	EventID    string
	Status     string
	Stdout     string
	Stderr     string
	DurationMs int64
	Err        error
}

func (e *Executor) FireHooks(ctx context.Context, event calendar.Event, hookType string, hooks []Hook, prev, next *calendar.Event) []Result {
	payload := calendar.EventToPayload(event, hookType, prev, next)
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log := logging.Get()
		log.Error("hooks", fmt.Sprintf("failed to marshal hook payload: %v", err))
		return nil
	}

	var wg sync.WaitGroup
	results := make([]Result, len(hooks))

	for i, hook := range hooks {
		wg.Add(1)
		go func(idx int, h Hook) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					log := logging.Get()
					log.Error("hooks", fmt.Sprintf("panic in hook %s: %v", h.Name, r))
					results[idx] = Result{
						HookName: h.Name,
						HookType: hookType,
						EventID:  event.ID,
						Status:   "failed",
						Stderr:   fmt.Sprintf("panic: %v", r),
					}
				}
			}()
			results[idx] = e.executeOne(ctx, h, hookType, event.ID, payloadBytes)
		}(i, hook)
	}

	wg.Wait()
	return results
}

func (e *Executor) executeOne(ctx context.Context, hook Hook, hookType, eventID string, payload []byte) Result {
	log := logging.Get()

	select {
	case e.semaphore <- struct{}{}:
		defer func() { <-e.semaphore }()
	case <-time.After(time.Duration(e.cfg.HookTimeoutSeconds) * time.Second):
		log.Warn("hooks", fmt.Sprintf("semaphore acquire timeout for %s, skipping", hook.Name))
		return Result{
			HookName: hook.Name,
			HookType: hookType,
			EventID:  eventID,
			Status:   "skipped",
			Stderr:   "semaphore acquire timeout",
		}
	}

	e.dbmu.Lock()
	executed, err := e.database.HasHookExecuted(eventID, hook.Name, hookType)
	e.dbmu.Unlock()
	if err != nil {
		log.Error("hooks", fmt.Sprintf("dedup check failed for %s: %v", hook.Name, err))
	}
	if executed {
		return Result{
			HookName: hook.Name,
			HookType: hookType,
			EventID:  eventID,
			Status:   "skipped",
			Stderr:   "already executed",
		}
	}

	timeout := time.Duration(e.cfg.HookTimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, hook.Path)
	cmd.Stdin = bytes.NewReader(payload)
	cmd.Dir = config.ConfigDir()
	cmd.Env = buildEnv(eventID, hookType)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	cmd.WaitDelay = 3 * time.Second

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &limitedWriter{buf: &stdoutBuf, max: e.cfg.HookOutputMaxBytes}
	cmd.Stderr = &limitedWriter{buf: &stderrBuf, max: e.cfg.HookOutputMaxBytes}

	start := time.Now()
	err = cmd.Run()
	duration := time.Since(start)
	durationMs := duration.Milliseconds()

	result := Result{
		HookName:   hook.Name,
		HookType:   hookType,
		EventID:    eventID,
		DurationMs: durationMs,
	}

	stdout := stdoutBuf.String()
	stderr := stderrBuf.String()
	if len(stdout) >= e.cfg.HookOutputMaxBytes {
		stdout = stdout[:e.cfg.HookOutputMaxBytes-len("[truncated]")] + "[truncated]"
	}
	if len(stderr) >= e.cfg.HookOutputMaxBytes {
		stderr = stderr[:e.cfg.HookOutputMaxBytes-len("[truncated]")] + "[truncated]"
	}
	result.Stdout = stdout
	result.Stderr = stderr

	if ctx.Err() != nil {
		result.Status = "timeout"
		result.Err = fmt.Errorf("hook timed out after %s", timeout)
		log.HookEvent(logging.LevelWarn, hook.Name, hookType, eventID, "timeout",
			fmt.Sprintf("Hook timed out after %dms", durationMs), durationMs)
	} else if err != nil {
		result.Status = "failed"
		result.Err = err
		log.HookEvent(logging.LevelError, hook.Name, hookType, eventID, "failed",
			fmt.Sprintf("Hook failed: %v", err), durationMs)
	} else {
		result.Status = "success"
		log.HookEvent(logging.LevelInfo, hook.Name, hookType, eventID, "success",
			fmt.Sprintf("Hook completed in %dms", durationMs), durationMs)
	}

	e.dbmu.Lock()
	dbErr := e.database.RecordHookExecution(eventID, hook.Name, hookType, result.Status, result.Stdout, result.Stderr, durationMs)
	e.dbmu.Unlock()
	if dbErr != nil {
		log.Error("hooks", fmt.Sprintf("failed to record hook execution: %v", dbErr))
	}

	return result
}

func buildEnv(eventID, hookType string) []string {
	env := os.Environ()
	env = append(env,
		"CALVIN_EVENT_ID="+eventID,
		"CALVIN_HOOK_TYPE="+hookType,
		"CALVIN_CONFIG_DIR="+config.ConfigDir(),
		"CALVIN_DATA_DIR="+config.DataDir(),
	)

	hasPath := false
	for i, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			hasPath = true
			path := strings.TrimPrefix(e, "PATH=")
			if !strings.Contains(path, "/usr/local/bin") {
				path = "/usr/local/bin:" + path
			}
			if !strings.Contains(path, "/opt/homebrew/bin") {
				path = "/opt/homebrew/bin:" + path
			}
			env[i] = "PATH=" + path
			break
		}
	}
	if !hasPath {
		env = append(env, "PATH=/usr/local/bin:/opt/homebrew/bin:/usr/bin:/bin")
	}

	return env
}

type limitedWriter struct {
	buf     *bytes.Buffer
	max     int
	written int
}

func (w *limitedWriter) Write(p []byte) (n int, err error) {
	remaining := w.max - w.written
	if remaining <= 0 {
		return len(p), nil
	}
	if len(p) > remaining {
		p = p[:remaining]
	}
	n, err = w.buf.Write(p)
	w.written += n
	return len(p), err
}

func RunTest(hookPath string, payload []byte) (string, string, int, error) {
	cmd := exec.Command(hookPath)
	cmd.Stdin = bytes.NewReader(payload)
	cmd.Dir = config.ConfigDir()
	cmd.Env = buildEnv("test-event", "test")

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return "", "", -1, err
		}
	}

	return stdoutBuf.String(), stderrBuf.String(), exitCode, nil
}
