package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

type outputMode string

const (
	outputModeText outputMode = "text"
	outputModeJSON outputMode = "json"
)

type commandResult struct {
	OK       bool     `json:"ok"`
	Command  string   `json:"command"`
	Data     any      `json:"data,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

type commandError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

type commandErrorResult struct {
	OK      bool         `json:"ok"`
	Command string       `json:"command"`
	Error   commandError `json:"error"`
}

type ExitError struct {
	ExitCode int
	Result   commandErrorResult
	err      error
}

func (e *ExitError) Error() string {
	if e == nil {
		return ""
	}
	if e.Result.Error.Message != "" {
		return e.Result.Error.Message
	}
	if e.err != nil {
		return e.err.Error()
	}
	return fmt.Sprintf("exit %d", e.ExitCode)
}

func (e *ExitError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func newExitError(code int, command, errCode, message string, details map[string]any, err error) *ExitError {
	return &ExitError{
		ExitCode: code,
		Result: commandErrorResult{
			OK:      false,
			Command: command,
			Error: commandError{
				Code:    errCode,
				Message: message,
				Details: details,
			},
		},
		err: err,
	}
}

func resolveOutputMode(outputFlag string, jsonFlag bool, envValue string) outputMode {
	if outputFlag != "" {
		return outputMode(strings.ToLower(outputFlag))
	}
	if jsonFlag || strings.EqualFold(envValue, string(outputModeJSON)) {
		return outputModeJSON
	}
	return outputModeText
}

func validateOutputMode(v string) error {
	if v == "" || v == string(outputModeText) || v == string(outputModeJSON) {
		return nil
	}
	return fmt.Errorf("invalid output mode: %s", v)
}

func writeJSONResult(w io.Writer, v commandResult) error {
	return writeJSON(w, v)
}

func writeJSONError(w io.Writer, v commandErrorResult) error {
	return writeJSON(w, v)
}

func writeCommandJSON(command string, data any, warnings ...string) error {
	result := commandResult{OK: true, Command: command, Data: data}
	if len(warnings) > 0 {
		result.Warnings = warnings
	}
	return writeJSONResult(os.Stdout, result)
}

func writeJSON(w io.Writer, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if _, err := w.Write(append(data, '\n')); err != nil {
		return err
	}
	return nil
}

func wantsJSON() bool {
	return currentOutputMode == outputModeJSON
}

func rawOutputMode(args []string, envValue string) outputMode {
	outputFlag, ok := rawOutputFlag(args)
	if ok {
		switch strings.ToLower(outputFlag) {
		case string(outputModeJSON):
			return outputModeJSON
		case string(outputModeText):
			return outputModeText
		default:
			if rawJSONFlag(args) || strings.EqualFold(envValue, string(outputModeJSON)) {
				return outputModeJSON
			}
			return outputModeText
		}
	}
	if rawJSONFlag(args) || strings.EqualFold(envValue, string(outputModeJSON)) {
		return outputModeJSON
	}
	return outputModeText
}

func rawJSONFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--json" || arg == "--json=true" {
			return true
		}
	}
	return false
}

func rawOutputFlag(args []string) (string, bool) {
	for i, arg := range args {
		if arg == "--output" {
			if i+1 >= len(args) {
				return "", true
			}
			return args[i+1], true
		}
		if strings.HasPrefix(arg, "--output=") {
			return strings.TrimPrefix(arg, "--output="), true
		}
	}
	return "", false
}

func wrapCLIError(err error, args []string) *ExitError {
	var exitErr *ExitError
	if errors.As(err, &exitErr) {
		return exitErr
	}

	command := commandNameFromArgs(args)
	code := 1
	errCode := "command_failed"
	if isUsageError(err) {
		code = 2
		errCode = "usage_error"
	}

	return newExitError(code, command, errCode, err.Error(), nil, err)
}

func isUsageError(err error) bool {
	message := err.Error()
	return strings.Contains(message, "unknown command") ||
		strings.Contains(message, "accepts") ||
		strings.Contains(message, "required flag") ||
		strings.Contains(message, "flag needs an argument") ||
		strings.Contains(message, "unknown flag") ||
		strings.Contains(message, "invalid argument")
}

func commandNameFromArgs(args []string) string {
	path := make([]string, 0, 2)
	skipValue := false
	for _, arg := range args {
		if skipValue {
			skipValue = false
			continue
		}
		if arg == "--output" {
			skipValue = true
			continue
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}
		path = append(path, arg)
		if len(path) == 2 {
			break
		}
	}
	if len(path) == 0 {
		return "calvin"
	}
	return strings.Join(path, " ")
}
