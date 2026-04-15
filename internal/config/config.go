package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type CalendarConfig struct {
	ID string `toml:"id"`
}

type Config struct {
	SyncIntervalSeconds        int              `toml:"sync_interval_seconds"`
	PreEventMinutes            int              `toml:"pre_event_minutes"`
	HookTimeoutSeconds         int              `toml:"hook_timeout_seconds"`
	MaxConcurrentHooks         int              `toml:"max_concurrent_hooks"`
	HookOutputMaxBytes         int              `toml:"hook_output_max_bytes"`
	HookExecutionRetentionDays int              `toml:"hook_execution_retention_days"`
	OAuthClientID              string           `toml:"oauth_client_id"`
	OAuthClientSecret          string           `toml:"oauth_client_secret"`
	AuthPort                   int              `toml:"auth_port"`
	Calendars                  []CalendarConfig `toml:"calendars"`
}

func Default() *Config {
	return &Config{
		SyncIntervalSeconds:        60,
		PreEventMinutes:            5,
		HookTimeoutSeconds:         30,
		MaxConcurrentHooks:         10,
		HookOutputMaxBytes:         65536,
		HookExecutionRetentionDays: 30,
		AuthPort:                   8085,
	}
}

func Load() (*Config, error) {
	cfg := Default()
	path := filepath.Join(ConfigDir(), "config.toml")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	if _, err := toml.Decode(string(data), cfg); err != nil {
		return nil, fmt.Errorf("parsing config.toml: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.SyncIntervalSeconds <= 0 {
		return fmt.Errorf("invalid config: sync_interval_seconds=%d (must be > 0)", c.SyncIntervalSeconds)
	}
	if c.PreEventMinutes < 0 {
		return fmt.Errorf("invalid config: pre_event_minutes=%d (must be >= 0)", c.PreEventMinutes)
	}
	if c.HookTimeoutSeconds <= 0 {
		return fmt.Errorf("invalid config: hook_timeout_seconds=%d (must be > 0)", c.HookTimeoutSeconds)
	}
	if c.MaxConcurrentHooks <= 0 {
		return fmt.Errorf("invalid config: max_concurrent_hooks=%d (must be > 0)", c.MaxConcurrentHooks)
	}
	if c.HookOutputMaxBytes <= 0 {
		return fmt.Errorf("invalid config: hook_output_max_bytes=%d (must be > 0)", c.HookOutputMaxBytes)
	}
	if c.HookExecutionRetentionDays <= 0 {
		return fmt.Errorf("invalid config: hook_execution_retention_days=%d (must be > 0)", c.HookExecutionRetentionDays)
	}
	if c.AuthPort <= 0 || c.AuthPort > 65535 {
		return fmt.Errorf("invalid config: auth_port=%d (must be 1-65535)", c.AuthPort)
	}
	return nil
}

func (c *Config) ResolvedCalendars() []CalendarConfig {
	if len(c.Calendars) > 0 {
		return c.Calendars
	}
	return []CalendarConfig{{ID: "primary"}}
}

func xdgDir(envVar, defaultSub string) string {
	if dir := os.Getenv(envVar); dir != "" && filepath.IsAbs(dir) {
		return filepath.Join(dir, "calvin")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, defaultSub, "calvin")
}

func ConfigDir() string {
	return xdgDir("XDG_CONFIG_HOME", ".config")
}

func DataDir() string {
	return xdgDir("XDG_DATA_HOME", filepath.Join(".local", "share"))
}

func StateDir() string {
	return xdgDir("XDG_STATE_HOME", filepath.Join(".local", "state"))
}

func HooksDir() string {
	return filepath.Join(ConfigDir(), "hooks")
}

func DBPath() string {
	return filepath.Join(DataDir(), "events.db")
}

func TokenPath() string {
	return filepath.Join(DataDir(), "token.json")
}

func LogPath() string {
	return filepath.Join(StateDir(), "calvin.log")
}

func PIDPath() string {
	return filepath.Join(StateDir(), "calvin.pid")
}
