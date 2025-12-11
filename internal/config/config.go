package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the main application configuration
type Config struct {
	Server    ServerConfig              `yaml:"server"`
	Auth      AuthConfig                `yaml:"auth"`
	Plugins   []PluginConfig            `yaml:"plugins"`
	Calendars []CalendarConfig          `yaml:"calendars"`
	Scheduler SchedulerConfig           `yaml:"scheduler"`
}

// ServerConfig contains HTTP server settings
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// AuthConfig contains authentication settings
type AuthConfig struct {
	Method string `yaml:"method"` // "none" or "apikey"
	APIKey string `yaml:"apiKey,omitempty"`
}

// PluginConfig represents a configured plugin instance
type PluginConfig struct {
	ID     string                 `yaml:"id"`
	Type   string                 `yaml:"type"`
	Config map[string]interface{} `yaml:"config,omitempty"`
}

// CalendarConfig represents a calendar that aggregates plugin events
type CalendarConfig struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	PluginIDs   []string `yaml:"plugins"`
}

// SchedulerConfig contains event fetching schedule settings
type SchedulerConfig struct {
	Interval time.Duration `yaml:"interval"`
}

// LoadFromFile loads configuration from a YAML file
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Set defaults
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}
	if cfg.Scheduler.Interval == 0 {
		cfg.Scheduler.Interval = 15 * time.Minute
	}
	if cfg.Auth.Method == "" {
		cfg.Auth.Method = "none"
	}

	return &cfg, nil
}
