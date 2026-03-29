package catalog

import (
	"path/filepath"
	"testing"
)

func TestLoadMetricsProfile(t *testing.T) {
	path := filepath.Join("testdata", "metrics.yml")
	metrics, err := LoadMetricsProfile(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(metrics) == 0 {
		t.Fatal("expected at least one metric definition")
	}

	// Verify a known metric exists with correct fields
	found := false
	for _, m := range metrics {
		if m.MetricName == "containerCPU" {
			found = true
			if m.Query == "" {
				t.Error("containerCPU has empty query")
			}
			break
		}
	}
	if !found {
		t.Error("containerCPU metric not found in profile")
	}

	// Verify all metrics have required fields
	for _, m := range metrics {
		if m.MetricName == "" {
			t.Error("metric has empty MetricName")
		}
		if m.Query == "" {
			t.Errorf("metric %q has empty Query", m.MetricName)
		}
	}
	t.Logf("parsed %d metrics from profile", len(metrics))
}

func TestParseMetricsProfileInstant(t *testing.T) {
	path := filepath.Join("testdata", "metrics.yml")
	metrics, err := LoadMetricsProfile(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	// Find an instant metric and verify the flag is set
	hasInstant := false
	hasCaptureStart := false
	for _, m := range metrics {
		if m.Instant {
			hasInstant = true
		}
		if m.CaptureStart {
			hasCaptureStart = true
		}
	}
	if !hasInstant {
		t.Error("expected at least one instant metric")
	}
	if !hasCaptureStart {
		t.Error("expected at least one captureStart metric")
	}
}
