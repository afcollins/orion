package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAllExampleConfigs(t *testing.T) {
	examplesDir := filepath.Join("..", "..", "examples")
	entries, err := os.ReadDir(examplesDir)
	if err != nil {
		t.Fatalf("reading examples dir: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			path := filepath.Join(examplesDir, entry.Name())
			cfg, err := LoadFile(path)
			if err != nil {
				data, _ := os.ReadFile(path)
				_, metricsErr := ParseMetricsFile(data)
				if metricsErr != nil {
					t.Fatalf("failed to parse as config or metrics file: config=%v, metrics=%v", err, metricsErr)
				}
				return
			}
			if cfg.Tests == nil {
				return
			}
			for _, test := range cfg.Tests {
				if test.Name == "" {
					t.Error("test has empty name")
				}
				for _, metric := range test.Metrics {
					if metric.Name == "" {
						t.Errorf("metric in test %q has empty name", test.Name)
					}
				}
			}
		})
	}
}

func TestLoadSubdirExampleConfigs(t *testing.T) {
	examplesDir := filepath.Join("..", "..", "examples")
	entries, err := os.ReadDir(examplesDir)
	if err != nil {
		t.Fatalf("reading examples dir: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		subDir := filepath.Join(examplesDir, entry.Name())
		subEntries, err := os.ReadDir(subDir)
		if err != nil {
			continue
		}
		for _, subEntry := range subEntries {
			if subEntry.IsDir() {
				continue
			}
			if !strings.HasSuffix(subEntry.Name(), ".yaml") && !strings.HasSuffix(subEntry.Name(), ".yml") {
				continue
			}
			t.Run(entry.Name()+"/"+subEntry.Name(), func(t *testing.T) {
				path := filepath.Join(subDir, subEntry.Name())
				cfg, err := LoadFile(path)
				if err != nil {
					data, _ := os.ReadFile(path)
					_, metricsErr := ParseMetricsFile(data)
					if metricsErr != nil {
						t.Fatalf("failed to parse as config or metrics: config=%v, metrics=%v", err, metricsErr)
					}
					return
				}
				if cfg.Tests == nil {
					return
				}
				for _, test := range cfg.Tests {
					if test.Name == "" {
						t.Error("test has empty name")
					}
				}
			})
		}
	}
}

func TestParseSmallScaleConfig(t *testing.T) {
	cfg, err := LoadFile(filepath.Join("..", "..", "examples", "label-small-scale-cluster-density.yaml"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(cfg.Tests) != 1 {
		t.Fatalf("expected 1 test, got %d", len(cfg.Tests))
	}
	test := cfg.Tests[0]
	if test.Name != "small-scale-cluster-density-v2" {
		t.Errorf("name = %q, want small-scale-cluster-density-v2", test.Name)
	}
	if got := test.Metadata.Get("platform"); got != "AWS" {
		t.Errorf("platform = %v, want AWS", got)
	}

	if len(test.Metrics) != 5 {
		t.Fatalf("expected 5 metrics, got %d", len(test.Metrics))
	}

	m := test.Metrics[0]
	if m.Name != "podReadyLatency" {
		t.Errorf("metric[0].Name = %q", m.Name)
	}
	if m.Filters["metricName.keyword"] != "podLatencyQuantilesMeasurement" {
		t.Errorf("metricName.keyword = %v", m.Filters["metricName.keyword"])
	}
	if m.Filters["quantileName"] != "Ready" {
		t.Errorf("quantileName = %v", m.Filters["quantileName"])
	}
	if m.MetricOfInterest != "P99" {
		t.Errorf("metric_of_interest = %q", m.MetricOfInterest)
	}
	if m.Not == nil || m.Not["jobConfig.name"] != "garbage-collection" {
		t.Errorf("not = %v", m.Not)
	}

	m = test.Metrics[1]
	if m.Name != "apiserverCPU" {
		t.Errorf("metric[1].Name = %q", m.Name)
	}
	if m.Filters["labels.namespace.keyword"] != "openshift-kube-apiserver" {
		t.Errorf("labels.namespace.keyword = %v", m.Filters["labels.namespace.keyword"])
	}
	if m.Agg == nil {
		t.Fatal("agg is nil")
	}
	if m.Agg.AggType != "avg" {
		t.Errorf("agg = %+v", m.Agg)
	}
}

func TestParseChaosConfig(t *testing.T) {
	cfg, err := LoadFile(filepath.Join("..", "..", "examples", "chaos_tests.yaml"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	test := cfg.Tests[0]
	if test.UUIDField != "run_uuid" {
		t.Errorf("uuid_field = %q, want run_uuid", test.UUIDField)
	}
	if test.VersionField != "cluster_version" {
		t.Errorf("version_field = %q, want cluster_version", test.VersionField)
	}
	if cfg.MetricsFile != "metrics/chaos-scenarios-metrics.yaml" {
		t.Errorf("metricsFile = %q", cfg.MetricsFile)
	}

	m := test.Metrics[0]
	if m.Name != "health_check_recovery" {
		t.Errorf("metric[0].Name = %q", m.Name)
	}
	if m.Direction == nil || *m.Direction != 1 {
		t.Errorf("direction = %v", m.Direction)
	}
	if m.Threshold == nil || *m.Threshold != 15 {
		t.Errorf("threshold = %v", m.Threshold)
	}
	if m.Not == nil {
		t.Fatal("not is nil")
	}
	if m.Not["status_code"] != 200 {
		t.Errorf("not.status_code = %v (type %T)", m.Not["status_code"], m.Not["status_code"])
	}
	if m.Agg == nil || m.Agg.AggType != "sum" {
		t.Errorf("agg = %v", m.Agg)
	}
}

func TestParseRhosoConfig(t *testing.T) {
	cfg, err := LoadFile(filepath.Join("..", "..", "examples", "rhoso", "rhoso-keystone.yaml"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	test := cfg.Tests[0]
	if test.UUIDField != "browbeat_uuid" {
		t.Errorf("uuid_field = %q", test.UUIDField)
	}
	if test.Threshold == nil || *test.Threshold != 10 {
		t.Errorf("threshold = %v", test.Threshold)
	}
	if test.Direction == nil || *test.Direction != 1 {
		t.Errorf("direction = %v", test.Direction)
	}

	m := test.Metrics[0]
	if m.Filters["action"] != "authenticate.keystone" {
		t.Errorf("action = %v", m.Filters["action"])
	}
	if m.Filters["doc_type"] != "result" {
		t.Errorf("doc_type = %v", m.Filters["doc_type"])
	}
	if m.Agg == nil {
		t.Fatal("agg is nil")
	}
	if m.Agg.AggType != "percentiles" {
		t.Errorf("agg_type = %q", m.Agg.AggType)
	}
	if len(m.Agg.Percents) != 1 || m.Agg.Percents[0] != 95 {
		t.Errorf("percents = %v", m.Agg.Percents)
	}
}

func TestParseInheritsConfig(t *testing.T) {
	cfg, err := LoadFile(filepath.Join("..", "..", "examples", "trt-external-payload-node-density-inherits.yaml"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if cfg.ParentConfig != "parent.yaml" {
		t.Errorf("parentConfig = %q", cfg.ParentConfig)
	}
	if cfg.MetricsFile != "metrics.yaml" {
		t.Errorf("metricsFile = %q", cfg.MetricsFile)
	}

	notVal := cfg.Tests[0].Metadata.Get("not")
	if notVal == nil {
		t.Fatal("metadata.not is nil")
	}
	notMap, ok := notVal.(map[string]interface{})
	if !ok {
		t.Fatalf("metadata.not is %T, want map", notVal)
	}
	if notMap["stream"] != "okd" {
		t.Errorf("metadata.not.stream = %v", notMap["stream"])
	}
}

func TestParseParentConfig(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "examples", "parent.yaml"))
	if err != nil {
		t.Fatalf("reading: %v", err)
	}
	meta, err := ParseParentConfig(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if meta.Get("platform") != "AWS" {
		t.Errorf("platform = %v", meta.Get("platform"))
	}
}

func TestRoundTrip(t *testing.T) {
	dir := 1
	thresh := 10.0
	cfg := &OrionConfig{
		Tests: []Test{
			{
				Name:      "test1",
				UUIDField: "uuid",
				Metadata: Metadata{
					Entries: []MetadataEntry{
						{Key: "platform", Value: "AWS"},
						{Key: "workerNodesCount", Value: 3},
					},
				},
				Metrics: []Metric{
					{
						Name:             "apiserverCPU",
						MetricOfInterest: "value",
						Direction:        &dir,
						Threshold:        &thresh,
						Filters: map[string]interface{}{
							"metricName.keyword":       "containerCPU",
							"labels.namespace.keyword": "openshift-kube-apiserver",
						},
						Agg: &Aggregation{
							Value:   "cpu",
							AggType: "avg",
						},
						Labels: []string{"[Jira: kube-apiserver]"},
					},
				},
			},
		},
	}

	data, err := Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	cfg2, err := Parse(data)
	if err != nil {
		t.Fatalf("parse roundtrip: %v", err)
	}

	m := cfg2.Tests[0].Metrics[0]
	if m.Name != "apiserverCPU" {
		t.Errorf("name = %q", m.Name)
	}
	if m.Filters["metricName.keyword"] != "containerCPU" {
		t.Errorf("metricName.keyword = %v", m.Filters["metricName.keyword"])
	}
	if m.Direction == nil || *m.Direction != 1 {
		t.Errorf("direction = %v", m.Direction)
	}
	if m.Agg == nil || m.Agg.AggType != "avg" {
		t.Errorf("agg = %v", m.Agg)
	}
}

func TestParseMetricsFile(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "metrics.yaml")
	metrics, err := LoadMetricsFile(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(metrics) == 0 {
		t.Fatal("expected at least one metric")
	}
	for _, m := range metrics {
		if m.Name == "" {
			t.Error("metric has empty name")
		}
	}
}

func TestMetadataNotFilter(t *testing.T) {
	yamlContent := `
tests:
  - name: test1
    metadata:
      platform: AWS
      not:
        stream: okd
    metrics:
      - name: m1
        metric_of_interest: value
`
	cfg, err := Parse([]byte(yamlContent))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	notVal := cfg.Tests[0].Metadata.Get("not")
	if notVal == nil {
		t.Fatal("not is nil")
	}
	notMap := notVal.(map[string]interface{})
	if notMap["stream"] != "okd" {
		t.Errorf("not.stream = %v", notMap["stream"])
	}
}

func TestBuildConfig(t *testing.T) {
	cfg := NewConfig().
		WithTest(NewTest("aws-small-scale-cluster-density-v2").
			WithPlatform("AWS").
			WithMasters(3, "m6a.xlarge").
			WithWorkers(24, "m6a.xlarge").
			WithBenchmark("cluster-density-v2").
			WithNetwork("OVNKubernetes").
			WithMetric(NewMetric("podReadyLatency").
				WithMetricName("podLatencyQuantilesMeasurement").
				WithFilter("quantileName", "Ready").
				WithInterest("P99").
				WithNot("jobConfig.name", "garbage-collection"),
			).
			WithMetric(NewMetric("apiserverCPU").
				WithMetricName("containerCPU").
				WithNamespace("openshift-kube-apiserver").
				WithAvg().
				WithLabel("[Jira: kube-apiserver]"),
			).
			WithMetric(NewMetric("etcdDisk").
				WithMetricName("99thEtcdDiskBackendCommitDurationSeconds").
				WithAvg(),
			).
			WithMetric(NewMetric("ovsCPU-irate-all").
				WithMetricName("cgroupCPU").
				WithFilter("labels.id.keyword", "/system.slice/ovs-vswitchd.service").
				WithAvg().
				WithDirection(1).
				WithThreshold(10),
			),
		).
		Build()

	data, err := Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	t.Logf("generated YAML:\n%s", data)

	got, err := Parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	test := got.Tests[0]
	if test.Name != "aws-small-scale-cluster-density-v2" {
		t.Errorf("name = %q", test.Name)
	}
	if test.Metadata.Get("platform") != "AWS" {
		t.Errorf("platform = %v", test.Metadata.Get("platform"))
	}
	if test.Metadata.Get("workerNodesCount") != 24 {
		t.Errorf("workerNodesCount = %v", test.Metadata.Get("workerNodesCount"))
	}

	if len(test.Metrics) != 4 {
		t.Fatalf("metrics count = %d, want 4", len(test.Metrics))
	}

	m := test.Metrics[0]
	if m.Name != "podReadyLatency" {
		t.Errorf("metric[0] name = %q", m.Name)
	}
	if m.Filters["metricName.keyword"] != "podLatencyQuantilesMeasurement" {
		t.Errorf("metricName.keyword = %v", m.Filters["metricName.keyword"])
	}
	if m.MetricOfInterest != "P99" {
		t.Errorf("metric_of_interest = %q", m.MetricOfInterest)
	}
	if m.Not["jobConfig.name"] != "garbage-collection" {
		t.Errorf("not = %v", m.Not)
	}

	m = test.Metrics[1]
	if m.Agg == nil || m.Agg.AggType != "avg" {
		t.Errorf("agg = %v", m.Agg)
	}
	if len(m.Labels) != 1 || m.Labels[0] != "[Jira: kube-apiserver]" {
		t.Errorf("labels = %v", m.Labels)
	}

	m = test.Metrics[3]
	if m.Direction == nil || *m.Direction != 1 {
		t.Errorf("direction = %v", m.Direction)
	}
	if m.Threshold == nil || *m.Threshold != 10 {
		t.Errorf("threshold = %v", m.Threshold)
	}
}

func readTestFile(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("..", "..", "examples", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}
	return string(data)
}
