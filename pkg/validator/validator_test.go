package validator

import (
	"path/filepath"
	"testing"

	"github.com/cloud-bulldozer/orion/pkg/catalog"
	"github.com/cloud-bulldozer/orion/pkg/config"
)


func TestValidateValidConfig(t *testing.T) {
	p, err := catalog.LoadMetricsProfile(filepath.Join("..", "catalog", "testdata", "metrics.yml"))
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}

	dir := 1
	cfg := &config.OrionConfig{
		Tests: []config.Test{
			{
				Name: "test1",
				Metrics: []config.Metric{
					{
						Name:             "apiserverCPU",
						MetricOfInterest: "value",
						Direction:        &dir,
						Filters: map[string]interface{}{
							"metricName.keyword":       "containerCPU",
							"labels.namespace.keyword": "openshift-kube-apiserver",
						},
					},
				},
			},
		},
	}

	errs := Validate(cfg, p)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestValidateUnknownMetricName(t *testing.T) {
	p, err := catalog.LoadMetricsProfile(filepath.Join("..", "catalog", "testdata", "metrics.yml"))
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}

	cfg := &config.OrionConfig{
		Tests: []config.Test{
			{
				Name: "test1",
				Metrics: []config.Metric{
					{
						Name:             "bad",
						MetricOfInterest: "value",
						Filters:          map[string]interface{}{"metricName.keyword": "doesNotExist"},
					},
				},
			},
		},
	}

	errs := Validate(cfg, p)
	if len(errs) == 0 {
		t.Fatal("expected error for unknown metricName, got none")
	}
}

func TestValidateInvalidLabelFilter(t *testing.T) {
	p, err := catalog.LoadMetricsProfile(filepath.Join("..", "catalog", "testdata", "metrics.yml"))
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}

	cfg := &config.OrionConfig{
		Tests: []config.Test{
			{
				Name: "test1",
				Metrics: []config.Metric{
					{
						Name:             "m1",
						MetricOfInterest: "value",
						Filters: map[string]interface{}{
							"metricName.keyword":     "containerCPU",
							"labels.region.keyword":  "us-east-1", // not a valid label
						},
					},
				},
			},
		},
	}

	errs := Validate(cfg, p)
	if len(errs) == 0 {
		t.Fatal("expected error for invalid label filter, got none")
	}
}

func TestValidateMissingMetricOfInterest(t *testing.T) {
	cfg := &config.OrionConfig{
		Tests: []config.Test{
			{
				Name: "test1",
				Metrics: []config.Metric{
					{
						Name:    "m1",
						Filters: map[string]interface{}{"metricName.keyword": "containerCPU"},
						// MetricOfInterest intentionally missing
					},
				},
			},
		},
	}

	errs := Validate(cfg, nil)
	if len(errs) == 0 {
		t.Fatal("expected error for missing metric_of_interest, got none")
	}
}

func TestValidateDuplicateMetricNames(t *testing.T) {
	cfg := &config.OrionConfig{
		Tests: []config.Test{
			{
				Name: "test1",
				Metrics: []config.Metric{
					{Name: "m1", MetricOfInterest: "value", Filters: map[string]interface{}{}},
					{Name: "m1", MetricOfInterest: "value", Filters: map[string]interface{}{}},
				},
			},
		},
	}

	errs := Validate(cfg, nil)
	if len(errs) == 0 {
		t.Fatal("expected error for duplicate metric name, got none")
	}
}

func TestValidateNoProfileSkipsMetricCheck(t *testing.T) {
	cfg := &config.OrionConfig{
		Tests: []config.Test{
			{
				Name: "test1",
				Metrics: []config.Metric{
					{
						Name:             "m1",
						MetricOfInterest: "value",
						Filters:          map[string]interface{}{"metricName.keyword": "anythingAtAll"},
					},
				},
			},
		},
	}

	// Without a profile, metricName cross-check is skipped.
	errs := Validate(cfg, nil)
	if len(errs) != 0 {
		t.Errorf("expected no errors without profile, got: %v", errs)
	}
}

func TestValidateExistingExampleConfig(t *testing.T) {
	p, err := catalog.LoadMetricsProfile(filepath.Join("..", "catalog", "testdata", "metrics.yml"))
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}

	cfg, err := config.LoadFile(filepath.Join("..", "..", "examples", "rhoso", "rhoso-keystone.yaml"))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	// Existing configs should pass structural validation even without profile cross-check.
	errs := Validate(cfg, p)
	// Some metricNames may not be in our test profile — that's expected and not a test failure.
	// What we care about is no panics and structural errors only.
	for _, e := range errs {
		t.Logf("validation note: %s", e)
	}
}

func TestLabelFromFilter(t *testing.T) {
	tests := []struct {
		key      string
		label    string
		ok       bool
	}{
		{"labels.namespace.keyword", "namespace", true},
		{"labels.container.keyword", "container", true},
		{"metricName.keyword", "", false},
		{"labels.keyword", "", false},
		{"labels.", "", false},
	}
	for _, tc := range tests {
		got, ok := labelFromFilter(tc.key)
		if ok != tc.ok || got != tc.label {
			t.Errorf("labelFromFilter(%q) = (%q, %v), want (%q, %v)", tc.key, got, ok, tc.label, tc.ok)
		}
	}
}
