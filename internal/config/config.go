package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type TrainConfig struct {
	From      string   `yaml:"from"`
	To        string   `yaml:"to"`
	Departure string   `yaml:"departure"`
	Days      []string `yaml:"days"` // e.g., ["monday", "wednesday", "friday"]
}

func (t TrainConfig) DepartureTime() (time.Time, error) {
	parsed, err := time.Parse("1504", t.Departure)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid departure time %q: %w", t.Departure, err)
	}
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), parsed.Hour(), parsed.Minute(), 0, 0, time.Local), nil
}

// IsActiveDay returns true if the given weekday is in the configured days list.
// If no days are configured, returns true (runs every day).
func (t TrainConfig) IsActiveDay(weekday time.Weekday) bool {
	if len(t.Days) == 0 {
		return true // No filter, run every day
	}
	dayName := strings.ToLower(weekday.String())
	for _, d := range t.Days {
		if strings.ToLower(d) == dayName {
			return true
		}
	}
	return false
}

// IsActiveToday returns true if today is an active day for this train config.
func (t TrainConfig) IsActiveToday() bool {
	return t.IsActiveDay(time.Now().Weekday())
}

type Config struct {
	MorningTrain TrainConfig `yaml:"morning_train"`
	EveningTrain TrainConfig `yaml:"evening_train"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.MorningTrain.From == "" || c.MorningTrain.To == "" || c.MorningTrain.Departure == "" {
		return fmt.Errorf("morning_train: from, to, and departure are required")
	}
	if c.EveningTrain.From == "" || c.EveningTrain.To == "" || c.EveningTrain.Departure == "" {
		return fmt.Errorf("evening_train: from, to, and departure are required")
	}

	if _, err := c.MorningTrain.DepartureTime(); err != nil {
		return fmt.Errorf("morning_train: %w", err)
	}
	if _, err := c.EveningTrain.DepartureTime(); err != nil {
		return fmt.Errorf("evening_train: %w", err)
	}

	return nil
}
