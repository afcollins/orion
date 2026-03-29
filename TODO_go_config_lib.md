# Go Config Library TODO

## Phase 1: Config Types & Loader
- [x] Initialize Go module (`go.mod`)
- [x] Define orion config structs (`pkg/config/types.go`) — OrionConfig, Test, Metric, Aggregation, Metadata
- [x] YAML marshal/unmarshal with custom handling for mixed known/filter fields
- [x] Loader functions (`pkg/config/loader.go`) — LoadFile, Parse, Marshal, ParseMetricsFile, ParseParentConfig
- [x] Jinja2 template quoting (`pkg/config/jinja2.go`) — wrap unquoted `{{ }}` before YAML parse
- [x] Tests loading all 67+ existing example configs for backward compatibility
- [x] Focused tests for specific config types (chaos, RHOSO, inheritance, round-trip)

## Phase 2: Catalog — Metrics Profile Parsing
- [x] Import kube-burner `MetricDefinition` from leaf package (`pkg/prometheus/api`)
- [x] Parse metrics profile YAML into `[]kbapi.MetricDefinition` (`pkg/catalog/types.go`)
- [x] Tests parsing `testdata/metrics.yml` (56 metrics, instant/captureStart flags)
- [x] Label extractor — parse `by()` clauses from Prometheus queries (`pkg/catalog/extractor.go`)
- [x] Tests for label extraction (multi-label, histogram le exclusion, edge cases)

## Phase 3: Config Generator
- [x] Generator library (`pkg/config/generator.go`) — produce orion Metric structs from profile metrics + user options
- [x] Tests for generation (simple, default metric_of_interest, unknown metric, invalid label, round-trip)

## Phase 4: CLI Tool (taurus)
- [x] CLI entrypoint (`cmd/taurus/main.go`) — `validate` and `list` commands
- [ ] `generate` command — scaffold a full orion config from a metrics profile (all metrics, no label filters)

## Phase 5: Validation
- [x] Validator (`pkg/validator/validator.go`) — structural + profile cross-check
- [x] Tests for validation (valid config, unknown metric, invalid label, missing field, duplicate, no-profile mode)

## Phase 6: Golden File Tests (generate command)
**Requirements:**
- The primary way to generate orion configs is via Go code (library API), not by editing a spec file.
  A TUI will eventually wrap this for interactive use.
- Tests must prove the generator is expressive enough to recreate the existing `examples/` configs.
- For each example config, write Go code using `TestSpec`/`MetricSpec` that produces it, then
  compare the marshaled output against the example file. Whitespace differences acceptable if
  they are invisible to `git diff -w`, otherwise the output must match verbatim.
- These golden file tests serve a dual purpose:
  1. **Completeness** — proves generator API covers all real config patterns
  2. **Regression** — catches any future change to marshaling/serialization that alters output

**Tasks:**
- [x] Golden file test harness (`pkg/config/golden_test.go`) — diff on failure, `-update` flag
      regenerates goldens from example files (examples are source of truth)
- [x] Implement generator recipe for `examples/label-small-scale-cluster-density.yaml`
- [x] Implement generator recipe for `examples/rhoso/rhoso-keystone.yaml`
- [ ] Implement generator recipe for `examples/chaos_tests.yaml`
- [ ] Implement generator recipe for `examples/trt-external-payload-node-density-inherits.yaml`
- [ ] Expand to remaining example configs as coverage grows

## Phase 7: jobSummary.json Metadata Parsing
- [ ] Import jobSummary struct from kube-burner (same pattern as `pkg/prometheus/api.MetricDefinition`)
- [ ] Parser that extracts orion `Metadata` entries from a jobSummary.json
- [ ] Tests for parsing (valid doc, missing fields, edge cases)
- [ ] Wire into taurus CLI — e.g. `taurus generate --job-summary jobSummary.json` to
      auto-populate test metadata from a real kube-burner run

## Open Questions
- Where should metric analysis metadata (direction, threshold, agg defaults) live?
- How to specify which label filter combinations to generate?

## Notes
- kube-burner dependency uses local replace directive for now
- `MetricDefinition` extracted to `pkg/prometheus/api` leaf package in kube-burner (pending PR)
- CLI binary: `go build ./cmd/taurus` (NOT `go build cmd/taurus/main.go`, which produces `main`)
