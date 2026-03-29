package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadFile reads an orion config YAML file and unmarshals it.
// Jinja2 template expressions ({{ ... }}) are quoted before parsing so they
// survive YAML parsing as literal strings. Config inheritance and template
// rendering are handled by the Python runtime at execution time.
func LoadFile(path string) (*OrionConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}
	return Parse(data)
}

// Parse unmarshals raw YAML bytes into an OrionConfig.
// Jinja2 template expressions are quoted automatically before parsing.
func Parse(data []byte) (*OrionConfig, error) {
	quoted := QuoteJinja2(string(data))
	var cfg OrionConfig
	if err := yaml.Unmarshal([]byte(quoted), &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}

// Marshal serializes an OrionConfig to YAML bytes.
func Marshal(cfg *OrionConfig) ([]byte, error) {
	return yaml.Marshal(cfg)
}

// ParseMetricsFile parses a standalone metrics YAML file (a list of metrics).
func ParseMetricsFile(data []byte) ([]Metric, error) {
	var metrics []Metric
	if err := yaml.Unmarshal(data, &metrics); err != nil {
		return nil, fmt.Errorf("parsing metrics file: %w", err)
	}
	return metrics, nil
}

// LoadMetricsFile reads and parses a standalone metrics YAML file.
func LoadMetricsFile(path string) ([]Metric, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading metrics file %s: %w", path, err)
	}
	return ParseMetricsFile(data)
}

// ParseParentConfig parses a parent config file (metadata-only, no tests).
func ParseParentConfig(data []byte) (*Metadata, error) {
	var raw struct {
		Metadata Metadata `yaml:"metadata"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing parent config: %w", err)
	}
	return &raw.Metadata, nil
}
