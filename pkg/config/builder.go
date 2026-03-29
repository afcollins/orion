package config

import "fmt"

// ConfigBuilder provides a fluent API for constructing OrionConfig values.
type ConfigBuilder struct {
	parentConfig string
	metricsFile  string
	tests        []Test
}

func NewConfig() *ConfigBuilder {
	return &ConfigBuilder{}
}

func (b *ConfigBuilder) WithParentConfig(path string) *ConfigBuilder {
	b.parentConfig = path
	return b
}

func (b *ConfigBuilder) WithMetricsFile(path string) *ConfigBuilder {
	b.metricsFile = path
	return b
}

func (b *ConfigBuilder) WithTest(tb *TestBuilder) *ConfigBuilder {
	b.tests = append(b.tests, tb.Build())
	return b
}

func (b *ConfigBuilder) Build() *OrionConfig {
	return &OrionConfig{
		ParentConfig: b.parentConfig,
		MetricsFile:  b.metricsFile,
		Tests:        b.tests,
	}
}

// TestBuilder provides a fluent API for constructing Test values.
type TestBuilder struct {
	name         string
	uuidField    string
	versionField string
	threshold    *float64
	direction    *int
	metadata     Metadata
	metrics      []Metric
}

func NewTest(name string) *TestBuilder {
	return &TestBuilder{name: name}
}

func (b *TestBuilder) WithUUIDField(field string) *TestBuilder {
	b.uuidField = field
	return b
}

func (b *TestBuilder) WithVersionField(field string) *TestBuilder {
	b.versionField = field
	return b
}

func (b *TestBuilder) WithThreshold(t float64) *TestBuilder {
	b.threshold = &t
	return b
}

func (b *TestBuilder) WithDirection(d int) *TestBuilder {
	b.direction = &d
	return b
}

func (b *TestBuilder) WithMeta(key string, value interface{}) *TestBuilder {
	b.metadata.Set(key, value)
	return b
}

func (b *TestBuilder) WithPlatform(platform string) *TestBuilder {
	return b.WithMeta("platform", platform)
}

func (b *TestBuilder) WithWorkers(count int, nodeType string) *TestBuilder {
	b.WithMeta("workerNodesCount", count)
	return b.WithMeta("workerNodesType", nodeType)
}

func (b *TestBuilder) WithMasters(count int, nodeType string) *TestBuilder {
	b.WithMeta("masterNodesCount", count)
	return b.WithMeta("masterNodesType", nodeType)
}

func (b *TestBuilder) WithBenchmark(name string) *TestBuilder {
	return b.WithMeta("benchmark.keyword", name)
}

func (b *TestBuilder) WithNetwork(networkType string) *TestBuilder {
	return b.WithMeta("networkType", networkType)
}

func (b *TestBuilder) WithMetric(mb *MetricBuilder) *TestBuilder {
	b.metrics = append(b.metrics, mb.Build())
	return b
}

func (b *TestBuilder) Build() Test {
	return Test{
		Name:         b.name,
		UUIDField:    b.uuidField,
		VersionField: b.versionField,
		Threshold:    b.threshold,
		Direction:    b.direction,
		Metadata:     b.metadata,
		Metrics:      b.metrics,
	}
}

// MetricBuilder provides a fluent API for constructing Metric values.
type MetricBuilder struct {
	name             string
	metricOfInterest string
	filters          map[string]interface{}
	agg              *Aggregation
	direction        *int
	threshold        *float64
	labels           []string
	not              map[string]interface{}
	correlation      string
	context          *int
}

func NewMetric(name string) *MetricBuilder {
	return &MetricBuilder{
		name:    name,
		filters: make(map[string]interface{}),
	}
}

func (b *MetricBuilder) WithMetricName(name string) *MetricBuilder {
	b.filters["metricName.keyword"] = name
	return b
}

func (b *MetricBuilder) WithFilter(key string, value interface{}) *MetricBuilder {
	b.filters[key] = value
	return b
}

func (b *MetricBuilder) WithNamespace(ns string) *MetricBuilder {
	b.filters["labels.namespace.keyword"] = ns
	return b
}

func (b *MetricBuilder) WithInterest(field string) *MetricBuilder {
	b.metricOfInterest = field
	return b
}

func (b *MetricBuilder) WithAgg(aggType string) *MetricBuilder {
	if b.agg == nil {
		b.agg = &Aggregation{}
	}
	b.agg.AggType = aggType
	return b
}

func (b *MetricBuilder) WithAvg() *MetricBuilder { return b.WithAgg("avg") }
func (b *MetricBuilder) WithSum() *MetricBuilder { return b.WithAgg("sum") }
func (b *MetricBuilder) WithMax() *MetricBuilder { return b.WithAgg("max") }
func (b *MetricBuilder) WithMin() *MetricBuilder { return b.WithAgg("min") }

func (b *MetricBuilder) WithPercentiles(percents []float64, target interface{}) *MetricBuilder {
	b.agg = &Aggregation{
		AggType:          "percentiles",
		Percents:         percents,
		TargetPercentile: target,
	}
	return b
}

func (b *MetricBuilder) WithDirection(d int) *MetricBuilder {
	b.direction = &d
	return b
}

func (b *MetricBuilder) WithThreshold(t float64) *MetricBuilder {
	b.threshold = &t
	return b
}

func (b *MetricBuilder) WithLabel(label string) *MetricBuilder {
	b.labels = append(b.labels, label)
	return b
}

func (b *MetricBuilder) WithNot(key string, value interface{}) *MetricBuilder {
	if b.not == nil {
		b.not = make(map[string]interface{})
	}
	b.not[key] = value
	return b
}

func (b *MetricBuilder) WithCorrelation(name string) *MetricBuilder {
	b.correlation = name
	return b
}

func (b *MetricBuilder) WithContext(ctx int) *MetricBuilder {
	b.context = &ctx
	return b
}

func (b *MetricBuilder) Build() Metric {
	moi := b.metricOfInterest
	if moi == "" {
		moi = "value"
	}
	return Metric{
		Name:             b.name,
		MetricOfInterest: moi,
		Filters:          b.filters,
		Agg:              b.agg,
		Direction:        b.direction,
		Threshold:        b.threshold,
		Labels:           b.labels,
		Not:              b.not,
		Correlation:      b.correlation,
		Context:          b.context,
	}
}

func (b *MetricBuilder) String() string {
	return fmt.Sprintf("Metric(%s)", b.name)
}
