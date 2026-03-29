package validator

import (
	"fmt"

	kbapi "github.com/kube-burner/kube-burner/v2/pkg/prometheus/api"

	"github.com/cloud-bulldozer/orion/pkg/catalog"
	"github.com/cloud-bulldozer/orion/pkg/config"
)

// Validate checks an OrionConfig for structural correctness and, when a metrics
// profile is provided, that every metricName.keyword references a known metric
// and every labels.X.keyword is a valid label for that metric's by() clause.
// Returns a list of human-readable error strings (empty means valid).
func Validate(cfg *config.OrionConfig, profile []kbapi.MetricDefinition) []string {
	var errs []string

	// Build a lookup from the profile if provided.
	byName := make(map[string]kbapi.MetricDefinition, len(profile))
	for _, m := range profile {
		byName[m.MetricName] = m
	}

	for _, test := range cfg.Tests {
		if test.Name == "" {
			errs = append(errs, "test has empty name")
		}

		seen := make(map[string]bool)
		for _, m := range test.Metrics {
			if m.Name == "" {
				errs = append(errs, fmt.Sprintf("test %q: metric has empty name", test.Name))
				continue
			}
			if seen[m.Name] {
				errs = append(errs, fmt.Sprintf("test %q: duplicate metric name %q", test.Name, m.Name))
			}
			seen[m.Name] = true

			if m.MetricOfInterest == "" {
				errs = append(errs, fmt.Sprintf("test %q, metric %q: missing metric_of_interest", test.Name, m.Name))
			}

			// Profile-based validation — only when a profile was provided.
			if len(profile) == 0 {
				continue
			}
			metricName, ok := m.Filters["metricName.keyword"]
			if !ok {
				// metricName (without .keyword) is also valid
				metricName, ok = m.Filters["metricName"]
			}
			if !ok {
				continue // no metricName filter, can't cross-check
			}
			name, _ := metricName.(string)
			if isImplicitMetric(name) {
				continue
			}
			def, found := byName[name]
			if !found {
				errs = append(errs, fmt.Sprintf("test %q, metric %q: metricName %q not found in profile",
					test.Name, m.Name, name))
				continue
			}

			// Validate labels.X.keyword filters against the by() clause.
			validLabels := make(map[string]bool)
			for _, l := range catalog.ExtractLabels(def.Query) {
				validLabels[l] = true
			}
			for k := range m.Filters {
				label, ok := labelFromFilter(k)
				if !ok {
					continue
				}
				if !validLabels[label] {
					errs = append(errs, fmt.Sprintf(
						"test %q, metric %q: label filter %q is not a valid label for %q (valid: %v)",
						test.Name, m.Name, label, name, catalog.ExtractLabels(def.Query)))
				}
			}
		}
	}
	return errs
}

// isImplicitMetric returns true for metric names that kube-burner creates
// automatically and are not listed in any metrics profile.
func isImplicitMetric(name string) bool {
	switch name {
	case "podLatencyMeasurement", "podLatencyQuantilesMeasurement":
		return true
	}
	return false
}

// labelFromFilter extracts the label name from a "labels.X.keyword" filter key.
// Returns ("", false) if the key is not a label filter.
func labelFromFilter(key string) (string, bool) {
	const prefix = "labels."
	const suffix = ".keyword"
	if len(key) > len(prefix)+len(suffix) &&
		key[:len(prefix)] == prefix &&
		key[len(key)-len(suffix):] == suffix {
		return key[len(prefix) : len(key)-len(suffix)], true
	}
	return "", false
}
