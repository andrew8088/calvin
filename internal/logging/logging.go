package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/andrew8088/calvin/internal/config"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Level string

const (
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

type Entry struct {
	Timestamp string `json:"ts"`
	Level     Level  `json:"level"`
	Component string `json:"component"`
	EventID   string `json:"event_id,omitempty"`
	HookName  string `json:"hook_name,omitempty"`
	HookType  string `json:"hook_type,omitempty"`
	Status    string `json:"status,omitempty"`
	DurationMs int64 `json:"duration_ms,omitempty"`
	Message   string `json:"message"`
}

type Logger struct {
	mu     sync.Mutex
	writer io.Writer
}

var defaultLogger *Logger

func Init() error {
	logPath := config.LogPath()
	w := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    10,
		MaxBackups: 3,
		Compress:   false,
	}
	defaultLogger = &Logger{writer: w}
	return nil
}

func InitStdout() {
	defaultLogger = &Logger{writer: os.Stdout}
}

func Get() *Logger {
	if defaultLogger == nil {
		InitStdout()
	}
	return defaultLogger
}

func (l *Logger) Log(level Level, component, message string) {
	l.log(Entry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level,
		Component: component,
		Message:   message,
	})
}

func (l *Logger) Info(component, message string) {
	l.Log(LevelInfo, component, message)
}

func (l *Logger) Warn(component, message string) {
	l.Log(LevelWarn, component, message)
}

func (l *Logger) Error(component, message string) {
	l.Log(LevelError, component, message)
}

func (l *Logger) HookEvent(level Level, hookName, hookType, eventID, status, message string, durationMs int64) {
	l.log(Entry{
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Level:      level,
		Component:  "hooks",
		EventID:    eventID,
		HookName:   hookName,
		HookType:   hookType,
		Status:     status,
		DurationMs: durationMs,
		Message:    message,
	})
}

func (l *Logger) log(e Entry) {
	data, err := json.Marshal(e)
	if err != nil {
		fmt.Fprintf(os.Stderr, "calvin: failed to marshal log entry: %v\n", err)
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.writer.Write(data)
	l.writer.Write([]byte("\n"))
}
