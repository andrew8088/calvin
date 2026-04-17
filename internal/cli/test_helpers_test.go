package cli

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/andrew8088/calvin/internal/calendar"
)

func runCLI(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	return runCLIWithTempEnv(t, nil, args...)
}

func runCLIWithTempEnv(t *testing.T, env []string, args ...string) (string, string, error) {
	t.Helper()
	temp, tempEnv := isolatedEnv(t)
	env = append(tempEnv, env...)
	_ = temp
	return runCLIWithEnv(t, env, args...)
}

func isolatedEnv(t *testing.T) (string, []string) {
	t.Helper()

	temp := t.TempDir()
	originalHome := os.Getenv("HOME")
	gopath := os.Getenv("GOPATH")
	if gopath == "" && originalHome != "" {
		gopath = filepath.Join(originalHome, "go")
	}
	gocache := os.Getenv("GOCACHE")
	if gocache == "" {
		if cacheDir, err := os.UserCacheDir(); err == nil {
			gocache = filepath.Join(cacheDir, "go-build")
		}
	}
	gomodcache := os.Getenv("GOMODCACHE")
	if gomodcache == "" && gopath != "" {
		gomodcache = filepath.Join(gopath, "pkg", "mod")
	}

	env := []string{}
	env = append(env,
		"HOME="+temp,
		"XDG_CONFIG_HOME="+filepath.Join(temp, "config"),
		"XDG_DATA_HOME="+filepath.Join(temp, "data"),
		"XDG_STATE_HOME="+filepath.Join(temp, "state"),
	)
	if gopath != "" {
		env = append(env, "GOPATH="+gopath)
	}
	if gomodcache != "" {
		env = append(env, "GOMODCACHE="+gomodcache)
	}
	if gocache != "" {
		env = append(env, "GOCACHE="+gocache)
	}

	return temp, env
}

func runCLIWithEnv(t *testing.T, env []string, args ...string) (string, string, error) {
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

	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), env...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	return stdout.String(), stderr.String(), err
}

func runCLIWithEventContext(t *testing.T, payload calendar.HookPayload, args ...string) (string, string, error) {
	t.Helper()
	eventFile := writeEventContextFile(t, payload)
	return runCLIWithTempEnv(t, []string{"CALVIN_EVENT_FILE=" + eventFile}, args...)
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if ok := errorAs(err, &exitErr); ok {
		return exitErr.ExitCode()
	}
	return 1
}

func errorAs(err error, target any) bool {
	return errors.As(err, target)
}
