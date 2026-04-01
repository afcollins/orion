# Orion: Current State, Gaps, and Future
Andrew's take

## Current State of Orion

Python code:
* ~3,750 lines (orion/ source, excluding tests)
* 43 unit tests, ~88% coverage

61 config files:
* Formatting was only recently linted
* Configs are heavily repeated, error-prone, and have no validation

## Gaps

**Language:** Orion is written in Python. The rest of our applications are in Go.

**Scale of metrics:** kube-burner captures millions of metrics for larger runs. These metrics form hundreds of individual time series, split by metricName and label values.

We currently track (across all config files):
* ~204 unique metrics
* ~19 namespaces
* ~5 container types (OVN-related)
* 2 cgroup slices (system.slice, kubepods.slice)

A standard cluster has:
* ~90 namespaces
* 100+ unique containers
* Additional metric types beyond CPU and Memory: disk, network, latencies

**Claim:** We cannot reach full coverage with hand-crafted config files. We could write ~10 lines of YAML for each metric time series, but this creates heavy duplication.

**Query architecture:** Orion uses a two-phase query pattern:

1. Query the metadata index for runs matching config criteria, extract UUIDs
2. For each metric in the config, query the metrics index filtered by those UUIDs

This results in N+1 sequential OpenSearch round trips for N metrics. Results are merged client-side in Python via pandas. This pattern scales poorly as we add metrics.

## Proposal: taurus - Generate Orion Configs from kube-burner Metrics Profiles

Goal: Create an Orion config that checks 100% of collected metrics for regressions.

* Report-mode metrics (metrics-report.yml) are already stored as individual time series
* containerCPU/Memory produces the most metrics and is our biggest untapped resource for analysis

See analysis reports: [pod latencies](/Users/ancollin/go/src/github.com/afcollins/kb-parser/pod_latency_stats.py) and [container CPU](/Users/ancollin/go/src/github.com/afcollins/kb-parser/containerCPU-analysis-report.md)

**Approach:** Two phases:
1. Generate a shared `metrics.yaml` covering all kube-burner metrics with sensible defaults for `agg_type`, `threshold`, and `direction`
2. Generate per-benchmark configs that reference it, varying only metadata

This aligns with the existing inheritance model (`parentConfig`, `metricsFile`) and is incrementally adoptable.

**Considerations:**
* Generated configs should be human-readable and committed to the repo, not generated at runtime. When a regression fires, someone needs to trace it back to a config.
* Not all metrics should use the same threshold or direction. taurus needs an override mechanism on top of generated defaults.
* kube-burner metrics profiles define *what* to collect, but Orion configs also need `agg_type`, `metric_of_interest`, `threshold`, and `direction`. taurus must provide sensible defaults for these.
* Maintenance shifts from 61 YAML files to maintaining the generator. This is a net improvement, but the generator itself becomes a dependency.

## Proposal: Use AI Tools for Test Coverage and Refactoring

* "Legacy code" can be defined as code without unit tests.
* Without tests, it is harder to change code, find bugs, fix bugs, add features, and review changes.
* Tests give us a safety net: they catch unexpected changes in behavior before they reach production.
* In progress: I have used Claude to generate tests and am validating them through mutation testing. Results are positive so far.

## Proposal: Reduce Query Latency

Orion is slow at fetching data points. The per-metric query loop (`utils.py:59-93`) and the client-side metadata/metrics join (`utils.py:342-375`) are the main bottlenecks.

### Quick win: `_msearch` batching

Batch all per-metric queries into a single OpenSearch `_msearch` request. This eliminates N-1 HTTP round trips with minimal code change.

### Caching

Once we compute data points for a given UUID, we should never need to compute them again. Completed run data is immutable, so cache invalidation is not a concern.

Possible approaches:
* **Local file cache (parquet/SQLite):** Low complexity, but not shared across CI jobs. Best for interactive and iterative analysis.
* **Shared cache (Redis/memcached):** Shared across CI, but adds infrastructure to maintain.
* **Pre-materialized results at ingest:** Fastest reads, but requires changes to the ingest pipeline.

### Longer-term: Denormalized index

The two-index pattern (metadata index + metrics index joined client-side by UUID) could be replaced by a single denormalized index containing both metadata and metrics. This eliminates the join entirely but requires changes to the ingest pipeline outside of Orion.

## Increasing the Number of Tracked Series

* kube-burner's report mode helps by producing one data point per run. However, with a 1:1 mapping between metric definitions and data points, we still need to precompute all available data points.
* By contrast, containerCPU gives a full time series per pod, which lets us group and summarize data during post-processing.
  * However, containerCPU produces the largest documents in both size and data point count. It is the main cause of OpenSearch indexing slowness during large runs and would be better stored in a time-series database (TSDB).
  * Orion does not currently support TSDB. TSDB support is a larger scope change and should be evaluated separately from the other proposals.
* We could collect more summary statistics at kube-burner time, but this means more duplicated queries — better handled through code-driven config generation (see taurus proposal above).
