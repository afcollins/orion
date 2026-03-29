package config

import (
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"
)

// OrionConfig represents the top-level orion configuration file.
type OrionConfig struct {
	// ParentConfig is the path to a parent config for metadata inheritance.
	ParentConfig string `yaml:"parentConfig,omitempty"`
	// MetricsFile is the path to an external metrics file to merge.
	MetricsFile string `yaml:"metricsFile,omitempty"`
	// Tests is the list of test configurations.
	Tests []Test `yaml:"tests"`
}

// Test represents a single test configuration within an OrionConfig.
type Test struct {
	// Name of the test. Supports Jinja2 template syntax when rendered by Python.
	Name string `yaml:"name"`
	// UUIDField is the field name used to identify unique runs (default: "uuid").
	UUIDField string `yaml:"uuid_field,omitempty"`
	// VersionField is the field name for the cluster version (default: "ocpVersion").
	VersionField string `yaml:"version_field,omitempty"`
	// Threshold is a global percentage threshold for regression alerts.
	Threshold *float64 `yaml:"threshold,omitempty"`
	// Direction is a global direction for all metrics (-1, 0, or 1).
	Direction *int `yaml:"direction,omitempty"`
	// IgnoreGlobal skips parent config metadata inheritance when true.
	IgnoreGlobal bool `yaml:"IgnoreGlobal,omitempty"`
	// IgnoreGlobalMetrics skips parent metrics inheritance when true.
	IgnoreGlobalMetrics bool `yaml:"IgnoreGlobalMetrics,omitempty"`
	// LocalConfig is the path to a local metadata config to merge.
	LocalConfig string `yaml:"local_config,omitempty"`
	// LocalMetrics is the path to a local metrics file to merge.
	LocalMetrics string `yaml:"local_metrics,omitempty"`
	// Metadata holds arbitrary key-value pairs for filtering/organizing results.
	Metadata Metadata `yaml:"metadata"`
	// Metrics is the list of metric definitions.
	Metrics []Metric `yaml:"metrics"`
}

// Metadata holds test metadata as ordered key-value pairs, preserving
// the flexibility of arbitrary fields (platform, clusterType, ocpVersion, etc.)
// while supporting nested "not" filters for exclusion queries.
type Metadata struct {
	// Entries holds all metadata key-value pairs in order.
	Entries []MetadataEntry `yaml:"-"`
}

// MetadataEntry is a single key-value pair in the metadata.
type MetadataEntry struct {
	Key   string
	Value interface{}
}

// Get returns the value for a key, or nil if not found.
func (m *Metadata) Get(key string) interface{} {
	for _, e := range m.Entries {
		if e.Key == key {
			return e.Value
		}
	}
	return nil
}

// Set sets or updates a key-value pair.
func (m *Metadata) Set(key string, value interface{}) {
	for i, e := range m.Entries {
		if e.Key == key {
			m.Entries[i].Value = value
			return
		}
	}
	m.Entries = append(m.Entries, MetadataEntry{Key: key, Value: value})
}

func (m Metadata) MarshalYAML() (interface{}, error) {
	node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for _, e := range m.Entries {
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: e.Key, Tag: "!!str"}
		valNode := &yaml.Node{}
		val, err := yaml.Marshal(e.Value)
		if err != nil {
			return nil, fmt.Errorf("marshaling metadata key %q: %w", e.Key, err)
		}
		if err := yaml.Unmarshal(val, valNode); err != nil {
			return nil, fmt.Errorf("re-encoding metadata key %q: %w", e.Key, err)
		}
		// yaml.Unmarshal wraps in a document node; unwrap it.
		if valNode.Kind == yaml.DocumentNode && len(valNode.Content) > 0 {
			valNode = valNode.Content[0]
		}
		node.Content = append(node.Content, keyNode, valNode)
	}
	return node, nil
}

func (m *Metadata) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("metadata must be a mapping, got %d", value.Kind)
	}
	m.Entries = nil
	for i := 0; i+1 < len(value.Content); i += 2 {
		key := value.Content[i].Value
		var val interface{}
		if err := value.Content[i+1].Decode(&val); err != nil {
			return fmt.Errorf("decoding metadata key %q: %w", key, err)
		}
		m.Entries = append(m.Entries, MetadataEntry{Key: key, Value: val})
	}
	return nil
}

// Metric represents a metric to analyze. It uses custom YAML marshaling to
// handle the mix of well-known fields and arbitrary Elasticsearch filter fields.
type Metric struct {
	// Name of the metric (must be unique within a test).
	Name string `yaml:"-"`
	// MetricOfInterest is the field to extract (e.g., "value", "P99", "raw").
	MetricOfInterest string `yaml:"-"`
	// Agg defines aggregation settings.
	Agg *Aggregation `yaml:"-"`
	// Direction indicates regression direction (-1, 0, or 1).
	Direction *int `yaml:"-"`
	// Threshold is the percentage change threshold for regression.
	Threshold *float64 `yaml:"-"`
	// Labels are Jira/tag labels (e.g., ["[Jira: etcd]"]).
	Labels []string `yaml:"-"`
	// Not holds exclusion filters.
	Not map[string]interface{} `yaml:"-"`
	// Correlation names a correlated metric.
	Correlation string `yaml:"-"`
	// Context is an integer context value.
	Context *int `yaml:"-"`
	// Filters holds arbitrary Elasticsearch filter fields like
	// "metricName.keyword", "labels.namespace.keyword", "quantileName", etc.
	Filters map[string]interface{} `yaml:"-"`
}

// well-known metric field names that get their own struct fields
var metricKnownFields = map[string]bool{
	"name":               true,
	"metric_of_interest": true,
	"agg":                true,
	"direction":          true,
	"threshold":          true,
	"labels":             true,
	"not":                true,
	"correlation":        true,
	"context":            true,
}

func (m Metric) MarshalYAML() (interface{}, error) {
	node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}

	addScalar := func(key, val string) {
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: key, Tag: "!!str"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: val, Tag: "!!str"},
		)
	}

	addNode := func(key string, val interface{}) error {
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: key, Tag: "!!str"}
		raw, err := yaml.Marshal(val)
		if err != nil {
			return err
		}
		valNode := &yaml.Node{}
		if err := yaml.Unmarshal(raw, valNode); err != nil {
			return err
		}
		if valNode.Kind == yaml.DocumentNode && len(valNode.Content) > 0 {
			valNode = valNode.Content[0]
		}
		node.Content = append(node.Content, keyNode, valNode)
		return nil
	}

	// Emit well-known fields first in a logical order.
	addScalar("name", m.Name)

	// Emit filter fields sorted for deterministic output.
	keys := make([]string, 0, len(m.Filters))
	for k := range m.Filters {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if err := addNode(k, m.Filters[k]); err != nil {
			return nil, err
		}
	}

	if m.MetricOfInterest != "" {
		addScalar("metric_of_interest", m.MetricOfInterest)
	}
	if m.Not != nil {
		if err := addNode("not", m.Not); err != nil {
			return nil, err
		}
	}
	if m.Agg != nil {
		if err := addNode("agg", m.Agg); err != nil {
			return nil, err
		}
	}
	if m.Direction != nil {
		if err := addNode("direction", *m.Direction); err != nil {
			return nil, err
		}
	}
	if m.Threshold != nil {
		if err := addNode("threshold", *m.Threshold); err != nil {
			return nil, err
		}
	}
	if m.Labels != nil {
		if err := addNode("labels", m.Labels); err != nil {
			return nil, err
		}
	}
	if m.Correlation != "" {
		addScalar("correlation", m.Correlation)
	}
	if m.Context != nil {
		if err := addNode("context", *m.Context); err != nil {
			return nil, err
		}
	}

	return node, nil
}

func (m *Metric) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("metric must be a mapping, got %d", value.Kind)
	}

	m.Filters = make(map[string]interface{})

	for i := 0; i+1 < len(value.Content); i += 2 {
		key := value.Content[i].Value
		valNode := value.Content[i+1]

		switch key {
		case "name":
			m.Name = valNode.Value
		case "metric_of_interest":
			m.MetricOfInterest = valNode.Value
		case "agg":
			var agg Aggregation
			if err := valNode.Decode(&agg); err != nil {
				return fmt.Errorf("decoding agg: %w", err)
			}
			m.Agg = &agg
		case "direction":
			var d int
			if err := valNode.Decode(&d); err != nil {
				return fmt.Errorf("decoding direction: %w", err)
			}
			m.Direction = &d
		case "threshold":
			var t float64
			if err := valNode.Decode(&t); err != nil {
				return fmt.Errorf("decoding threshold: %w", err)
			}
			m.Threshold = &t
		case "labels":
			var labels []string
			if err := valNode.Decode(&labels); err != nil {
				return fmt.Errorf("decoding labels: %w", err)
			}
			m.Labels = labels
		case "not":
			var notMap map[string]interface{}
			if err := valNode.Decode(&notMap); err != nil {
				return fmt.Errorf("decoding not: %w", err)
			}
			m.Not = notMap
		case "correlation":
			m.Correlation = valNode.Value
		case "context":
			var c int
			if err := valNode.Decode(&c); err != nil {
				return fmt.Errorf("decoding context: %w", err)
			}
			m.Context = &c
		default:
			// Arbitrary ES filter field
			var val interface{}
			if err := valNode.Decode(&val); err != nil {
				return fmt.Errorf("decoding filter %q: %w", key, err)
			}
			m.Filters[key] = val
		}
	}
	return nil
}

// Aggregation defines how to aggregate metric values.
type Aggregation struct {
	// AggType is the aggregation function (avg, sum, max, min, count, percentiles).
	AggType string `yaml:"agg_type"`
	// Percents lists the percentile values to compute (for agg_type: percentiles).
	Percents []float64 `yaml:"percents,omitempty"`
	// TargetPercentile selects which percentile to use as the result.
	TargetPercentile interface{} `yaml:"target_percentile,omitempty"`
}
