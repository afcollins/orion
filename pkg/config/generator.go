package config

import (
	"fmt"
	"strings"

	kbapi "github.com/kube-burner/kube-burner/v2/pkg/prometheus/api"
	"github.com/cloud-bulldozer/orion/pkg/catalog"
)

// aggPrefixes lists metric name prefixes that already encode an aggregation
// (e.g. "max-cpu-etcd", "avg-ro-apicalls-latency") and don't need an agg block.
var aggPrefixes = []string{"max", "avg"}

// needsAgg returns true if the profile metric name does not already encode
// an aggregation function in its name.
func needsAgg(profileMetricName string) bool {
	lower := strings.ToLower(profileMetricName)
	for _, p := range aggPrefixes {
		if strings.HasPrefix(lower, p) {
			return false
		}
	}
	return true
}

// MetricSpec describes how to generate a single orion Metric from a profile metric.
type MetricSpec struct {
	// Name is the orion metric name (unique within the test).
	Name string
	// ProfileMetricName is the metricName from the kube-burner metrics profile.
	ProfileMetricName string
	// MetricOfInterest is the field to extract (default: "value").
	MetricOfInterest string
	// LabelFilters are values for specific by() labels, e.g. {"namespace": "openshift-etcd"}.
	// Each entry becomes a `labels.X.keyword` filter in the orion config.
	LabelFilters map[string]string
	// ExtraFilters are arbitrary additional ES filter fields (e.g. {"quantileName": "Ready"}).
	ExtraFilters map[string]interface{}
	// Agg defines aggregation settings. If nil, no agg block is emitted.
	Agg *Aggregation
	// Direction overrides the regression direction (-1, 0, or 1).
	Direction *int
	// Threshold overrides the regression percentage threshold.
	Threshold *float64
	// Labels are Jira/tag labels (e.g. ["[Jira: etcd]"]).
	Labels []string
	// Not holds exclusion filters.
	Not map[string]interface{}
}

// TestSpec describes how to generate a single orion Test.
type TestSpec struct {
	// Name is the test name.
	Name string
	// UUIDField overrides the default uuid field (default: "uuid").
	UUIDField string
	// VersionField overrides the default version field (default: "ocpVersion").
	VersionField string
	// Threshold sets a global threshold for all metrics in this test.
	Threshold *float64
	// Direction sets a global direction for all metrics in this test.
	Direction *int
	// Metadata holds the test metadata key-value pairs.
	Metadata Metadata
	// Metrics lists the metric specifications to generate.
	Metrics []MetricSpec
}

// GenerateConfig produces an OrionConfig from a list of TestSpecs, validated
// against the provided metrics profile. Returns an error if a MetricSpec
// references a ProfileMetricName not present in the profile, or if a
// LabelFilter key is not a valid label for that metric's by() clause.
func GenerateConfig(profile []kbapi.MetricDefinition, tests []TestSpec) (*OrionConfig, error) {
	cfg := &OrionConfig{}
	for _, ts := range tests {
		test, err := generateTest(profile, ts)
		if err != nil {
			return nil, fmt.Errorf("test %q: %w", ts.Name, err)
		}
		cfg.Tests = append(cfg.Tests, *test)
	}
	return cfg, nil
}

func generateTest(profile []kbapi.MetricDefinition, ts TestSpec) (*Test, error) {
	test := &Test{
		Name:         ts.Name,
		UUIDField:    ts.UUIDField,
		VersionField: ts.VersionField,
		Threshold:    ts.Threshold,
		Direction:    ts.Direction,
		Metadata:     ts.Metadata,
	}

	for _, ms := range ts.Metrics {
		m, err := generateMetric(profile, ms)
		if err != nil {
			return nil, err
		}
		test.Metrics = append(test.Metrics, *m)
	}
	return test, nil
}

func generateMetric(profile []kbapi.MetricDefinition, ms MetricSpec) (*Metric, error) {
	// Look up the profile metric for validation.
	var def *kbapi.MetricDefinition
	for i := range profile {
		if profile[i].MetricName == ms.ProfileMetricName {
			def = &profile[i]
			break
		}
	}
	if def == nil {
		return nil, fmt.Errorf("metric %q: profileMetricName %q not found in metrics profile",
			ms.Name, ms.ProfileMetricName)
	}

	// Validate label filters against the by() clause.
	if len(ms.LabelFilters) > 0 {
		validLabels := catalog.ExtractLabels(def.Query)
		valid := make(map[string]bool, len(validLabels))
		for _, l := range validLabels {
			valid[l] = true
		}
		for k := range ms.LabelFilters {
			if !valid[k] {
				return nil, fmt.Errorf("metric %q: label filter %q is not a valid label for %q (valid: %v)",
					ms.Name, k, ms.ProfileMetricName, validLabels)
			}
		}
	}

	moi := ms.MetricOfInterest
	if moi == "" {
		moi = "value"
	}

	filters := make(map[string]interface{})
	// Use metricName.keyword for exact ES matching.
	filters["metricName.keyword"] = ms.ProfileMetricName
	for k, v := range ms.LabelFilters {
		filters[fmt.Sprintf("labels.%s.keyword", k)] = v
	}
	for k, v := range ms.ExtraFilters {
		filters[k] = v
	}

	agg := ms.Agg
	if agg == nil && needsAgg(ms.ProfileMetricName) {
		agg = &Aggregation{AggType: "avg"}
	}

	return &Metric{
		Name:             ms.Name,
		MetricOfInterest: moi,
		Filters:          filters,
		Agg:              agg,
		Direction:        ms.Direction,
		Threshold:        ms.Threshold,
		Labels:           ms.Labels,
		Not:              ms.Not,
	}, nil
}
