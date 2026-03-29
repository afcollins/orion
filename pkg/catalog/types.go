package catalog

import (
	"fmt"
	"os"

	kbapi "github.com/kube-burner/kube-burner/v2/pkg/prometheus/api"
	"gopkg.in/yaml.v3"
)

// LoadMetricsProfile reads a kube-burner metrics profile YAML file and
// returns the metric definitions it contains.
func LoadMetricsProfile(path string) ([]kbapi.MetricDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading metrics profile %s: %w", path, err)
	}
	return ParseMetricsProfile(data)
}

// ParseMetricsProfile unmarshals raw YAML bytes into metric definitions.
func ParseMetricsProfile(data []byte) ([]kbapi.MetricDefinition, error) {
	var metrics []kbapi.MetricDefinition
	if err := yaml.Unmarshal(data, &metrics); err != nil {
		return nil, fmt.Errorf("parsing metrics profile: %w", err)
	}
	return metrics, nil
}
