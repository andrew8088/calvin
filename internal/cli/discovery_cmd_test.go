package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCommandsCommandJSON(t *testing.T) {
	stdout, stderr, err := runCLI(t, "commands", "--json")
	if err != nil {
		t.Fatalf("runCLI: %v stderr=%s", err, stderr)
	}
	if !json.Valid([]byte(stdout)) || stderr != "" || !strings.Contains(stdout, `"path": "events"`) {
		t.Fatalf("stdout=%s stderr=%s", stdout, stderr)
	}
}

func TestDescribeCommandJSON(t *testing.T) {
	stdout, stderr, err := runCLI(t, "describe", "hooks", "new", "--json")
	if err != nil {
		t.Fatalf("runCLI: %v stderr=%s", err, stderr)
	}
	if !json.Valid([]byte(stdout)) || stderr != "" || !strings.Contains(stdout, `"mutates_state": true`) || !strings.Contains(stdout, `"name": "json"`) {
		t.Fatalf("stdout=%s stderr=%s", stdout, stderr)
	}
}

func TestDescribeCommandInvalidPathReturnsStructuredError(t *testing.T) {
	stdout, stderr, err := runCLI(t, "describe", "hooks", "missing", "--json")
	if exitCode(err) == 0 || stdout != "" || !json.Valid([]byte(stderr)) {
		t.Fatalf("stdout=%s stderr=%s", stdout, stderr)
	}
}

func TestSchemaCommandHookPayload(t *testing.T) {
	stdout, stderr, err := runCLI(t, "schema", "hook-payload")
	if err != nil {
		t.Fatalf("runCLI: %v stderr=%s", err, stderr)
	}
	if !json.Valid([]byte(stdout)) || stderr != "" || !strings.Contains(stdout, `"schema_version"`) {
		t.Fatalf("stdout=%s stderr=%s", stdout, stderr)
	}
}

func TestSchemaCommandInvalidNameReturnsStructuredError(t *testing.T) {
	stdout, stderr, err := runCLI(t, "schema", "nope", "--json")
	if exitCode(err) == 0 || stdout != "" || !json.Valid([]byte(stderr)) {
		t.Fatalf("stdout=%s stderr=%s", stdout, stderr)
	}
}

func TestHooksSchemaJSONUsesSharedSchema(t *testing.T) {
	stdout, stderr, err := runCLI(t, "hooks", "schema", "--json")
	if err != nil {
		t.Fatalf("runCLI: %v stderr=%s", err, stderr)
	}
	if !json.Valid([]byte(stdout)) || stderr != "" || !strings.Contains(stdout, `"schema_version"`) {
		t.Fatalf("stdout=%s stderr=%s", stdout, stderr)
	}
}
