package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.SyncIntervalSeconds != 60 {
		t.Errorf("SyncIntervalSeconds = %d, want 60", cfg.SyncIntervalSeconds)
	}
	if cfg.PreEventMinutes != 5 {
		t.Errorf("PreEventMinutes = %d, want 5", cfg.PreEventMinutes)
	}
	if cfg.HookTimeoutSeconds != 30 {
		t.Errorf("HookTimeoutSeconds = %d, want 30", cfg.HookTimeoutSeconds)
	}
	if cfg.MaxConcurrentHooks != 10 {
		t.Errorf("MaxConcurrentHooks = %d, want 10", cfg.MaxConcurrentHooks)
	}
	if cfg.HookOutputMaxBytes != 65536 {
		t.Errorf("HookOutputMaxBytes = %d, want 65536", cfg.HookOutputMaxBytes)
	}
	if cfg.HookExecutionRetentionDays != 30 {
		t.Errorf("HookExecutionRetentionDays = %d, want 30", cfg.HookExecutionRetentionDays)
	}
	if cfg.AuthPort != 8085 {
		t.Errorf("AuthPort = %d, want 8085", cfg.AuthPort)
	}
}

func TestValidate_Default(t *testing.T) {
	cfg := Default()
	if err := cfg.Validate(); err != nil {
		t.Errorf("default config should be valid: %v", err)
	}
}

func TestValidate_InvalidFields(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*Config)
	}{
		{"zero sync interval", func(c *Config) { c.SyncIntervalSeconds = 0 }},
		{"negative sync interval", func(c *Config) { c.SyncIntervalSeconds = -1 }},
		{"negative pre-event", func(c *Config) { c.PreEventMinutes = -1 }},
		{"zero hook timeout", func(c *Config) { c.HookTimeoutSeconds = 0 }},
		{"zero max concurrent", func(c *Config) { c.MaxConcurrentHooks = 0 }},
		{"zero output max bytes", func(c *Config) { c.HookOutputMaxBytes = 0 }},
		{"zero retention days", func(c *Config) { c.HookExecutionRetentionDays = 0 }},
		{"zero auth port", func(c *Config) { c.AuthPort = 0 }},
		{"auth port too high", func(c *Config) { c.AuthPort = 65536 }},
		{"negative auth port", func(c *Config) { c.AuthPort = -1 }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			tt.modify(cfg)
			if err := cfg.Validate(); err == nil {
				t.Error("expected validation error")
			}
		})
	}
}

func TestValidate_PreEventZeroIsValid(t *testing.T) {
	cfg := Default()
	cfg.PreEventMinutes = 0
	if err := cfg.Validate(); err != nil {
		t.Errorf("pre_event_minutes=0 should be valid: %v", err)
	}
}

func TestXDGPaths_CustomEnv(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("XDG_STATE_HOME", tmp)

	if got := ConfigDir(); got != filepath.Join(tmp, "calvin") {
		t.Errorf("ConfigDir() = %q, want %q", got, filepath.Join(tmp, "calvin"))
	}
	if got := DataDir(); got != filepath.Join(tmp, "calvin") {
		t.Errorf("DataDir() = %q, want %q", got, filepath.Join(tmp, "calvin"))
	}
	if got := StateDir(); got != filepath.Join(tmp, "calvin") {
		t.Errorf("StateDir() = %q, want %q", got, filepath.Join(tmp, "calvin"))
	}
}

func TestXDGPaths_RelativeEnvIgnored(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "relative/path")

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".config", "calvin")
	if got := ConfigDir(); got != expected {
		t.Errorf("ConfigDir() with relative XDG = %q, want %q", got, expected)
	}
}

func TestDerivedPaths(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("XDG_STATE_HOME", tmp)

	base := filepath.Join(tmp, "calvin")

	if got := HooksDir(); got != filepath.Join(base, "hooks") {
		t.Errorf("HooksDir() = %q", got)
	}
	if got := DBPath(); got != filepath.Join(base, "events.db") {
		t.Errorf("DBPath() = %q", got)
	}
	if got := TokenPath(); got != filepath.Join(base, "token.json") {
		t.Errorf("TokenPath() = %q", got)
	}
	if got := LogPath(); got != filepath.Join(base, "calvin.log") {
		t.Errorf("LogPath() = %q", got)
	}
	if got := PIDPath(); got != filepath.Join(base, "calvin.pid") {
		t.Errorf("PIDPath() = %q", got)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() with missing file should return defaults: %v", err)
	}
	if cfg.SyncIntervalSeconds != 60 {
		t.Errorf("expected default sync interval, got %d", cfg.SyncIntervalSeconds)
	}
}

func TestLoad_ValidFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	dir := filepath.Join(tmp, "calvin")
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, "config.toml"), []byte(`
sync_interval_seconds = 120
pre_event_minutes = 10
`), 0644)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if cfg.SyncIntervalSeconds != 120 {
		t.Errorf("SyncIntervalSeconds = %d, want 120", cfg.SyncIntervalSeconds)
	}
	if cfg.PreEventMinutes != 10 {
		t.Errorf("PreEventMinutes = %d, want 10", cfg.PreEventMinutes)
	}
	if cfg.HookTimeoutSeconds != 30 {
		t.Errorf("unset fields should keep defaults, got HookTimeoutSeconds=%d", cfg.HookTimeoutSeconds)
	}
}

func TestLoad_InvalidFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	dir := filepath.Join(tmp, "calvin")
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, "config.toml"), []byte(`sync_interval_seconds = -5`), 0644)

	_, err := Load()
	if err == nil {
		t.Error("expected validation error for invalid config")
	}
}
