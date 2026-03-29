package config

// Generator recipes for existing example configs.
// Each recipe encodes an example config as Go code using TestSpec/MetricSpec,
// generates it, marshals it, and compares against the golden file derived
// from parsing the original example.

import (
	"path/filepath"
	"testing"
)


func examplePath(name string) string {
	return filepath.Join("..", "..", "examples", name)
}

func TestGoldenSmallScaleClusterDensity(t *testing.T) {
	cfg := &OrionConfig{
		Tests: []Test{
			{
				Name: "small-scale-cluster-density-v2",
				Metadata: Metadata{Entries: []MetadataEntry{
					{Key: "platform", Value: "AWS"},
					{Key: "masterNodesType", Value: "m6a.xlarge"},
					{Key: "masterNodesCount", Value: 3},
					{Key: "workerNodesType", Value: "m6a.xlarge"},
					{Key: "workerNodesCount", Value: 24},
					{Key: "benchmark.keyword", Value: "cluster-density-v2"},
					{Key: "ocpVersion", Value: "{{ version }}"},
					{Key: "not", Value: map[string]interface{}{"stream": "okd"}},
				}},
				Metrics: []Metric{
					{
						Name:             "podReadyLatency",
						MetricOfInterest: "P99",
						Filters: map[string]interface{}{
							"metricName.keyword": "podLatencyQuantilesMeasurement",
							"quantileName":       "Ready",
						},
						Not:    map[string]interface{}{"jobConfig.name": "garbage-collection"},
						Labels: []string{"[Jira: PodLatency]"},
					},
					{
						Name:             "apiserverCPU",
						MetricOfInterest: "value",
						Filters: map[string]interface{}{
							"metricName.keyword":       "containerCPU",
							"labels.namespace.keyword": "openshift-kube-apiserver",
						},
						Agg:    &Aggregation{Value: "cpu", AggType: "avg"},
						Labels: []string{"[Jira: kube-apiserver]"},
					},
					{
						Name:             "ovnCPU",
						MetricOfInterest: "value",
						Filters: map[string]interface{}{
							"metricName.keyword":       "containerCPU",
							"labels.namespace.keyword": "openshift-ovn-kubernetes",
						},
						Agg:    &Aggregation{Value: "cpu", AggType: "avg"},
						Labels: []string{"[Jira: ovnKubernetes]"},
					},
					{
						Name:             "etcdCPU",
						MetricOfInterest: "value",
						Filters: map[string]interface{}{
							"metricName.keyword":       "containerCPU",
							"labels.namespace.keyword": "openshift-etcd",
						},
						Agg:    &Aggregation{Value: "cpu", AggType: "avg"},
						Labels: []string{"[Jira: etcd]"},
					},
					{
						Name:             "etcdDisk",
						MetricOfInterest: "value",
						Filters: map[string]interface{}{
							"metricName.keyword": "99thEtcdDiskBackendCommitDurationSeconds",
						},
						Agg:    &Aggregation{Value: "duration", AggType: "avg"},
						Labels: []string{"[Jira: etcd]"},
					},
				},
			},
		},
	}

	got, err := Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	goldenTest(t, "label-small-scale-cluster-density", examplePath("label-small-scale-cluster-density.yaml"), got)
}

func TestGoldenRhosoKeystone(t *testing.T) {
	thresh := 10.0
	dir := 1

	newKeystoneMetric := func(name, action string) Metric {
		return Metric{
			Name:             name,
			MetricOfInterest: "raw",
			Filters: map[string]interface{}{
				"action":   action,
				"doc_type": "result",
			},
			Agg: &Aggregation{
				Value:            "duration",
				AggType:          "percentiles",
				Percents:         []float64{95},
				TargetPercentile: 95,
			},
			Labels: []string{"[Jira: KEYSTONE]"},
		}
	}

	cfg := &OrionConfig{
		Tests: []Test{
			{
				Name:      "keystone-performance-test",
				UUIDField: "browbeat_uuid",
				Threshold: &thresh,
				Direction: &dir,
				Metadata: Metadata{Entries: []MetadataEntry{
					{Key: "jobName.keyword", Value: "periodic-ci-openshift-eng-ocp-qe-perfscale-ci-main-metal-rhoso-x86-weekly-rhoso-keystone"},
					{Key: "jobType", Value: "periodic"},
					{Key: "jobStatus", Value: "pass"},
				}},
				Metrics: []Metric{
					newKeystoneMetric("keystone", "authenticate.keystone"),
					newKeystoneMetric("validateNeutron", "authenticate.validate_neutron"),
					newKeystoneMetric("validateNova", "authenticate.validate_nova"),
					newKeystoneMetric("createProject", "keystone_v3.create_project"),
					newKeystoneMetric("createUser", "keystone_v3.create_user"),
					newKeystoneMetric("listProjects", "keystone_v3.list_projects"),
					newKeystoneMetric("listUsers", "keystone_v3.list_users"),
				},
			},
		},
	}

	got, err := Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	goldenTest(t, "rhoso-keystone", examplePath(filepath.Join("rhoso", "rhoso-keystone.yaml")), got)
}
