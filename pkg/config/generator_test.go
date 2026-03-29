package config

import (
	"path/filepath"
	"testing"

	"github.com/cloud-bulldozer/orion/pkg/catalog"
)


func TestGenerateConfigSimple(t *testing.T) {
	profile, err := catalog.LoadMetricsProfile(filepath.Join("..", "catalog", "testdata", "metrics.yml"))
	if err != nil {
		t.Fatalf("loading profile: %v", err)
	}

	dir := 1
	thresh := 10.0
	cfg, err := GenerateConfig(profile, []TestSpec{
		{
			Name: "my-cluster-density",
			Metadata: Metadata{
				Entries: []MetadataEntry{
					{Key: "platform", Value: "AWS"},
					{Key: "benchmark.keyword", Value: "cluster-density-v2"},
				},
			},
			Metrics: []MetricSpec{
				{
					Name:              "apiserverCPU",
					ProfileMetricName: "containerCPU",
					LabelFilters:      map[string]string{"namespace": "openshift-kube-apiserver"},
					Agg:               &Aggregation{Value: "cpu", AggType: "avg"},
					Direction:         &dir,
					Threshold:         &thresh,
					Labels:            []string{"[Jira: kube-apiserver]"},
				},
				{
					Name:              "etcdCPU",
					ProfileMetricName: "containerCPU",
					LabelFilters:      map[string]string{"namespace": "openshift-etcd"},
					Agg:               &Aggregation{Value: "cpu", AggType: "avg"},
					Labels:            []string{"[Jira: etcd]"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("GenerateConfig: %v", err)
	}

	if len(cfg.Tests) != 1 {
		t.Fatalf("expected 1 test, got %d", len(cfg.Tests))
	}
	test := cfg.Tests[0]
	if test.Name != "my-cluster-density" {
		t.Errorf("name = %q", test.Name)
	}
	if len(test.Metrics) != 2 {
		t.Fatalf("expected 2 metrics, got %d", len(test.Metrics))
	}

	m0 := test.Metrics[0]
	if m0.Name != "apiserverCPU" {
		t.Errorf("metric[0].Name = %q", m0.Name)
	}
	if m0.Filters["metricName.keyword"] != "containerCPU" {
		t.Errorf("metricName.keyword = %v", m0.Filters["metricName.keyword"])
	}
	if m0.Filters["labels.namespace.keyword"] != "openshift-kube-apiserver" {
		t.Errorf("labels.namespace.keyword = %v", m0.Filters["labels.namespace.keyword"])
	}
	if m0.MetricOfInterest != "value" {
		t.Errorf("metric_of_interest = %q", m0.MetricOfInterest)
	}
	if m0.Agg == nil || m0.Agg.AggType != "avg" {
		t.Errorf("agg = %v", m0.Agg)
	}
	if m0.Direction == nil || *m0.Direction != 1 {
		t.Errorf("direction = %v", m0.Direction)
	}
}

func TestGenerateConfigDefaultMetricOfInterest(t *testing.T) {
	profile, err := catalog.LoadMetricsProfile(filepath.Join("..", "catalog", "testdata", "metrics.yml"))
	if err != nil {
		t.Fatalf("loading profile: %v", err)
	}

	cfg, err := GenerateConfig(profile, []TestSpec{
		{
			Name: "test",
			Metrics: []MetricSpec{
				{
					Name:              "m1",
					ProfileMetricName: "containerCPU",
					// MetricOfInterest intentionally omitted — should default to "value"
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("GenerateConfig: %v", err)
	}
	if cfg.Tests[0].Metrics[0].MetricOfInterest != "value" {
		t.Errorf("default metric_of_interest should be 'value', got %q",
			cfg.Tests[0].Metrics[0].MetricOfInterest)
	}
}

func TestGenerateConfigAutoAgg(t *testing.T) {
	profile, err := catalog.LoadMetricsProfile(filepath.Join("..", "catalog", "testdata", "metrics.yml"))
	if err != nil {
		t.Fatalf("loading profile: %v", err)
	}

	cfg, err := GenerateConfig(profile, []TestSpec{
		{
			Name: "test",
			Metrics: []MetricSpec{
				{
					Name:              "cpu",
					ProfileMetricName: "containerCPU",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("GenerateConfig: %v", err)
	}

	m := cfg.Tests[0].Metrics[0]
	if m.Agg == nil {
		t.Fatal("expected auto-generated agg for containerCPU, got nil")
	}
	if m.Agg.AggType != "avg" {
		t.Errorf("auto agg_type = %q, want avg", m.Agg.AggType)
	}
}

func TestGenerateConfigNoAggForPrefixedMetrics(t *testing.T) {
	profile, err := catalog.LoadMetricsProfile(filepath.Join("..", "catalog", "testdata", "metrics.yml"))
	if err != nil {
		t.Fatalf("loading profile: %v", err)
	}

	// Find a metric that starts with "99th" — these don't have max/avg prefix
	// but let's verify the ones that DO have the prefix are skipped.
	// We need metrics starting with "max" or "avg" in the profile.
	// The test profile may not have them, so let's test with needsAgg directly.
	if needsAgg("containerCPU") != true {
		t.Error("containerCPU should need agg")
	}
	if needsAgg("max-cpu-etcd") != false {
		t.Error("max-cpu-etcd should not need agg")
	}
	if needsAgg("avg-ro-apicalls-latency") != false {
		t.Error("avg-ro-apicalls-latency should not need agg")
	}
	if needsAgg("99thEtcdDiskBackendCommitDurationSeconds") != true {
		t.Error("99thEtcdDisk... should need agg")
	}

	// Verify explicit Agg is preserved (not overwritten).
	cfg, err := GenerateConfig(profile, []TestSpec{
		{
			Name: "test",
			Metrics: []MetricSpec{
				{
					Name:              "cpu",
					ProfileMetricName: "containerCPU",
					Agg:               &Aggregation{AggType: "max"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("GenerateConfig: %v", err)
	}
	if cfg.Tests[0].Metrics[0].Agg.AggType != "max" {
		t.Errorf("explicit agg should be preserved, got %q", cfg.Tests[0].Metrics[0].Agg.AggType)
	}
}

func TestGenerateConfigUnknownMetric(t *testing.T) {
	profile, err := catalog.LoadMetricsProfile(filepath.Join("..", "catalog", "testdata", "metrics.yml"))
	if err != nil {
		t.Fatalf("loading profile: %v", err)
	}

	_, err = GenerateConfig(profile, []TestSpec{
		{
			Name: "test",
			Metrics: []MetricSpec{
				{
					Name:              "bad",
					ProfileMetricName: "doesNotExist",
				},
			},
		},
	})
	if err == nil {
		t.Fatal("expected error for unknown profile metric, got nil")
	}
}

func TestGenerateConfigInvalidLabelFilter(t *testing.T) {
	profile, err := catalog.LoadMetricsProfile(filepath.Join("..", "catalog", "testdata", "metrics.yml"))
	if err != nil {
		t.Fatalf("loading profile: %v", err)
	}

	_, err = GenerateConfig(profile, []TestSpec{
		{
			Name: "test",
			Metrics: []MetricSpec{
				{
					Name:              "m1",
					ProfileMetricName: "containerCPU",
					// "region" is not a label in containerCPU's by() clause
					LabelFilters: map[string]string{"region": "us-east-1"},
				},
			},
		},
	})
	if err == nil {
		t.Fatal("expected error for invalid label filter, got nil")
	}
}

func TestGenerateConfigRoundTrip(t *testing.T) {
	profile, err := catalog.LoadMetricsProfile(filepath.Join("..", "catalog", "testdata", "metrics.yml"))
	if err != nil {
		t.Fatalf("loading profile: %v", err)
	}

	cfg, err := GenerateConfig(profile, []TestSpec{
		{
			Name: "round-trip-test",
			Metadata: Metadata{
				Entries: []MetadataEntry{{Key: "platform", Value: "AWS"}},
			},
			Metrics: []MetricSpec{
				{
					Name:              "etcdCPU",
					ProfileMetricName: "containerCPU",
					LabelFilters:      map[string]string{"namespace": "openshift-etcd"},
					Agg:               &Aggregation{Value: "cpu", AggType: "avg"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	// Marshal to YAML then parse back — must survive the round trip.
	data, err := Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	cfg2, err := Parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	m := cfg2.Tests[0].Metrics[0]
	if m.Filters["metricName.keyword"] != "containerCPU" {
		t.Errorf("metricName.keyword = %v", m.Filters["metricName.keyword"])
	}
	if m.Filters["labels.namespace.keyword"] != "openshift-etcd" {
		t.Errorf("labels.namespace.keyword = %v", m.Filters["labels.namespace.keyword"])
	}
}
