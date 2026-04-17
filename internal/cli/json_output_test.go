package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestResolveOutputMode_JSONFlagWins(t *testing.T) {
	mode := resolveOutputMode("", true, "")
	if mode != outputModeJSON {
		t.Fatalf("mode = %q, want %q", mode, outputModeJSON)
	}
}

func TestResolveOutputMode_EnvEnablesJSON(t *testing.T) {
	mode := resolveOutputMode("", false, "json")
	if mode != outputModeJSON {
		t.Fatalf("mode = %q, want %q", mode, outputModeJSON)
	}
}

func TestResolveOutputMode_FlagBeatsEnv(t *testing.T) {
	mode := resolveOutputMode("", true, "text")
	if mode != outputModeJSON {
		t.Fatalf("mode = %q, want %q", mode, outputModeJSON)
	}
}

func TestResolveOutputMode_OutputFlagWinsEnv(t *testing.T) {
	mode := resolveOutputMode("text", false, "json")
	if mode != outputModeText {
		t.Fatalf("mode = %q, want %q", mode, outputModeText)
	}
}

func TestResolveOutputMode_OutputFlagWinsJSONFlag(t *testing.T) {
	mode := resolveOutputMode("text", true, "json")
	if mode != outputModeText {
		t.Fatalf("mode = %q, want %q", mode, outputModeText)
	}
}

func TestWriteJSONResult_StableEnvelope(t *testing.T) {
	buf := new(bytes.Buffer)
	err := writeJSONResult(buf, commandResult{
		OK:      true,
		Command: "version",
		Data:    map[string]any{"version": "dev"},
	})
	if err != nil {
		t.Fatalf("writeJSONResult: %v", err)
	}
	if !strings.Contains(buf.String(), `"command": "version"`) {
		t.Fatalf("stdout = %s", buf.String())
	}
}

func TestWriteJSONError_StableEnvelope(t *testing.T) {
	buf := new(bytes.Buffer)
	err := writeJSONError(buf, commandErrorResult{
		OK:      false,
		Command: "hooks new",
		Error: commandError{
			Code:    "invalid_hook_type",
			Message: "invalid hook type: before-start",
		},
	})
	if err != nil {
		t.Fatalf("writeJSONError: %v", err)
	}
	if !strings.Contains(buf.String(), `"ok": false`) {
		t.Fatalf("stderr = %s", buf.String())
	}
}

func TestValidateOutputMode_Invalid(t *testing.T) {
	if err := validateOutputMode("yaml"); err == nil {
		t.Fatal("expected invalid output mode to fail")
	}
}
