package hooks

import (
	"bytes"
	"strings"
	"testing"

	"github.com/andrew8088/calvin/internal/config"
)

func TestLimitedWriter_UnderLimit(t *testing.T) {
	var buf bytes.Buffer
	w := &limitedWriter{buf: &buf, max: 100}

	n, err := w.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n != 5 {
		t.Errorf("n = %d, want 5", n)
	}
	if buf.String() != "hello" {
		t.Errorf("buf = %q", buf.String())
	}
}

func TestLimitedWriter_AtLimit(t *testing.T) {
	var buf bytes.Buffer
	w := &limitedWriter{buf: &buf, max: 5}

	w.Write([]byte("hel"))
	w.Write([]byte("lo world"))

	if buf.String() != "hello" {
		t.Errorf("buf = %q, want 'hello'", buf.String())
	}
}

func TestLimitedWriter_OverLimit(t *testing.T) {
	var buf bytes.Buffer
	w := &limitedWriter{buf: &buf, max: 3}

	n, err := w.Write([]byte("abcdef"))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n != 3 {
		t.Errorf("n = %d, want 3 (capped at remaining)", n)
	}
	if buf.String() != "abc" {
		t.Errorf("buf = %q, want 'abc'", buf.String())
	}
}

func TestLimitedWriter_ExactLimit(t *testing.T) {
	var buf bytes.Buffer
	w := &limitedWriter{buf: &buf, max: 5}

	w.Write([]byte("12345"))
	if buf.String() != "12345" {
		t.Errorf("buf = %q, want '12345'", buf.String())
	}

	w.Write([]byte("more"))
	if buf.String() != "12345" {
		t.Errorf("buf after overflow = %q, want '12345'", buf.String())
	}
}

func TestLimitedWriter_ZeroMax(t *testing.T) {
	var buf bytes.Buffer
	w := &limitedWriter{buf: &buf, max: 0}

	n, _ := w.Write([]byte("anything"))
	if n != 8 {
		t.Errorf("n = %d, want 8", n)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty buf with max=0")
	}
}

func TestBuildEnv_ContainsCalvinVars(t *testing.T) {
	env := buildEnv("evt-1", "pre_event")

	found := map[string]bool{}
	for _, e := range env {
		if strings.HasPrefix(e, "CALVIN_") {
			parts := strings.SplitN(e, "=", 2)
			found[parts[0]] = true
		}
	}

	for _, key := range []string{"CALVIN_EVENT_ID", "CALVIN_HOOK_TYPE", "CALVIN_CONFIG_DIR", "CALVIN_DATA_DIR"} {
		if !found[key] {
			t.Errorf("missing env var %s", key)
		}
	}
}

func TestBuildEnv_EventIDValue(t *testing.T) {
	env := buildEnv("my-event", "event_start")

	for _, e := range env {
		if e == "CALVIN_EVENT_ID=my-event" {
			return
		}
	}
	t.Error("CALVIN_EVENT_ID not set correctly")
}

func TestBuildEnv_PathIncludes(t *testing.T) {
	env := buildEnv("evt-1", "pre_event")

	for _, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			path := strings.TrimPrefix(e, "PATH=")
			if !strings.Contains(path, "/usr/local/bin") {
				t.Error("PATH missing /usr/local/bin")
			}
			if !strings.Contains(path, "/opt/homebrew/bin") {
				t.Error("PATH missing /opt/homebrew/bin")
			}
			return
		}
	}
	t.Error("PATH not found in env")
}

func TestBuildEnv_ConfigDirValue(t *testing.T) {
	env := buildEnv("evt-1", "pre_event")
	expected := "CALVIN_CONFIG_DIR=" + config.ConfigDir()

	for _, e := range env {
		if e == expected {
			return
		}
	}
	t.Errorf("expected %q in env", expected)
}
