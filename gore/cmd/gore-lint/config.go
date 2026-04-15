package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents gore-lint configuration.
type Config struct {
	Rules   RulesConfig  `yaml:"rules"`
	Exclude []string     `yaml:"exclude"`
	Output  OutputConfig `yaml:"output"`
}

// RulesConfig configures rule severities.
type RulesConfig map[string]string

// OutputConfig configures output behavior.
type OutputConfig struct {
	Format string `yaml:"format"`
	Color  *bool  `yaml:"color"` // pointer to distinguish unset from false
}

// LoadConfig loads configuration from a YAML file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no config file is OK
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}

// RuleSeverity returns the severity for a rule, or default if not configured.
func (c *Config) RuleSeverity(ruleID, defaultSev string) string {
	if c == nil {
		return defaultSev
	}
	if sev, ok := c.Rules[ruleID]; ok {
		return sev
	}
	return defaultSev
}

// IsExcluded checks if a path should be excluded.
func (c *Config) IsExcluded(path string) bool {
	if c == nil {
		return false
	}
	for _, pattern := range c.Exclude {
		if matchPath(pattern, path) {
			return true
		}
	}
	return false
}

func matchPath(pattern, path string) bool {
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(path) >= len(prefix) && path[:len(prefix)] == prefix
	}
	return pattern == path
}
