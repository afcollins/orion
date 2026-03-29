package catalog

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestExtractLabels(t *testing.T) {
	tests := []struct {
		name   string
		query  string
		expect []string
	}{
		{
			name:   "single label",
			query:  `sum(kube_namespace_status_phase) by (phase) > 0`,
			expect: []string{"phase"},
		},
		{
			name:   "multiple labels",
			query:  `sum(irate(container_cpu_usage_seconds_total{name!=""}[2m])) by (container, pod, namespace, node)`,
			expect: []string{"container", "pod", "namespace", "node"},
		},
		{
			name:   "no spaces",
			query:  `sum(rate(x[2m])) by(verb,resource,code)`,
			expect: []string{"verb", "resource", "code"},
		},
		{
			name:   "histogram quantile excludes le",
			query:  `histogram_quantile(0.99, sum(rate(apiserver_request_duration_seconds_bucket[2m])) by (le, resource, verb, scope))`,
			expect: []string{"resource", "verb", "scope"},
		},
		{
			name:   "no by clause",
			query:  `count(kube_secret_info{})`,
			expect: nil,
		},
		{
			name:   "multiple by clauses",
			query:  `(sum(irate(node_cpu_seconds_total[2m])) by (mode,instance) and on (instance) label_replace(kube_node_role{role="worker"}, "instance", "$1", "node", "(.+)"))`,
			expect: []string{"mode", "instance"},
		},
		{
			name:   "single label mode",
			query:  `sum(node_cpu_seconds_total) by (mode)`,
			expect: []string{"mode"},
		},
		{
			name:   "with gt filter",
			query:  `sum(irate(apiserver_request_total{verb!="WATCH"}[2m])) by (verb,resource,code) > 0`,
			expect: []string{"verb", "resource", "code"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractLabels(tc.query)
			if !reflect.DeepEqual(got, tc.expect) {
				t.Errorf("ExtractLabels() = %v, want %v", got, tc.expect)
			}
		})
	}
}

func TestExtractLabelsFromProfile(t *testing.T) {
	path := filepath.Join("testdata", "metrics.yml")
	metrics, err := LoadMetricsProfile(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	// containerCPU should have container, pod, namespace, node
	for _, m := range metrics {
		if m.MetricName == "containerCPU" {
			labels := ExtractLabels(m.Query)
			if len(labels) == 0 {
				t.Fatal("expected labels for containerCPU")
			}
			expect := map[string]bool{"container": true, "pod": true, "namespace": true, "node": true}
			for _, l := range labels {
				if !expect[l] {
					t.Errorf("unexpected label %q for containerCPU", l)
				}
			}
			for l := range expect {
				found := false
				for _, got := range labels {
					if got == l {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("missing expected label %q for containerCPU", l)
				}
			}
			return
		}
	}
	t.Fatal("containerCPU not found in profile")
}
