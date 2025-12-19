package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Config struct {
	Notification NotificationConfig `json:"notification"`
	Monitor      MonitorConfig      `json:"monitor"`
}

type NotificationConfig struct {
	Desktop   bool   `json:"desktop"`
	Sound     bool   `json:"sound"`
	SoundFile string `json:"sound_file,omitempty"`
}

type MonitorConfig struct {
	PollIntervalMs int `json:"poll_interval_ms"`
	IdleThresholdS int `json:"idle_threshold_s"`
	DebounceSecs   int `json:"debounce_secs"`
}

var (
	cfg     *Config
	cfgOnce sync.Once
	cfgDir  string
)

func DefaultConfig() *Config {
	return &Config{
		Notification: NotificationConfig{
			Desktop:   true,
			Sound:     true,
			SoundFile: "",
		},
		Monitor: MonitorConfig{
			PollIntervalMs: 500,
			IdleThresholdS: 2,
			DebounceSecs:   30,
		},
	}
}

func GetConfigDir() string {
	cfgOnce.Do(func() {
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			cfgDir = filepath.Join(xdg, "gclaude")
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				cfgDir = ".gclaude"
			} else {
				cfgDir = filepath.Join(home, ".config", "gclaude")
			}
		}
	})
	return cfgDir
}

func GetDataDir() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "gclaude")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".gclaude"
	}
	return filepath.Join(home, ".local", "share", "gclaude")
}

func EnsureConfigDir() error {
	return os.MkdirAll(GetConfigDir(), 0755)
}

func EnsureDataDir() error {
	return os.MkdirAll(GetDataDir(), 0755)
}

func configPath() string {
	return filepath.Join(GetConfigDir(), "config.json")
}

func Load() (*Config, error) {
	if cfg != nil {
		return cfg, nil
	}

	cfg = DefaultConfig()

	data, err := os.ReadFile(configPath())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func Save(c *Config) error {
	if err := EnsureConfigDir(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath(), data, 0644)
}

func Get() *Config {
	c, _ := Load()
	return c
}
