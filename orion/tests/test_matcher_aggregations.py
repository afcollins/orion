"""
Unit Test file for matcher aggregation functionality
"""

# pylint: disable = redefined-outer-name
# pylint: disable = missing-function-docstring
# pylint: disable = import-error, duplicate-code
from unittest.mock import patch
import pytest
from opensearch_dsl import Search
from opensearch_dsl.response import Response

# Import shared fixtures and helpers from test_matcher
from orion.cache import QueryCache
from orion.tests.test_matcher import make_matcher_fixture


@pytest.fixture
def matcher_instance():
    return make_matcher_fixture(index="perf-scale-ci")


@pytest.fixture
def uuid_matcher_instance():
    return make_matcher_fixture(
        index="krkn-telemetry",
        uuid_field="run_uuid",
        version_field="cluster_version",
    )


@pytest.mark.parametrize(
    "fixture_name,test_uuids,test_metrics,data_dict,expected",
    [
        # matcher_instance with values (single uuid agg, each bucket has time + value metric)
        (
            "matcher_instance",
            ["uuid1", "uuid2"],
            {
                "name": "apiserverCPU",
                "metricName": "containerCPU",
                "labels.namespace": "openshift-kube-apiserver",
                "metric_of_interest": "value",
                "agg": {"value": "cpu", "agg_type": "avg"},
            },
            {
                "aggregations": {
                    "uuid": {
                        "buckets": [
                            {
                                "key": "uuid1",
                                "time": {"value_as_string": "2024-02-09T12:00:00"},
                                "value": {"value": 42},
                            },
                            {
                                "key": "uuid2",
                                "time": {"value_as_string": "2024-02-09T13:00:00"},
                                "value": {"value": 56},
                            },
                        ]
                    },
                }
            },
            [
                {"uuid": "uuid1", "timestamp": "2024-02-09T12:00:00", "value_avg": 42},
                {"uuid": "uuid2", "timestamp": "2024-02-09T13:00:00", "value_avg": 56},
            ],
        ),
        # uuid_matcher_instance with values
        (
            "uuid_matcher_instance",
            ["uuid1", "uuid2"],
            {
                "name": "apiserverCPU",
                "metricName": "containerCPU",
                "labels.namespace": "openshift-kube-apiserver",
                "metric_of_interest": "value",
                "agg": {"value": "cpu", "agg_type": "avg"},
            },
            {
                "aggregations": {
                    "uuid": {
                        "buckets": [
                            {
                                "key": "uuid1",
                                "time": {"value_as_string": "2024-02-09T12:00:00"},
                                "value": {"value": 42},
                            },
                            {
                                "key": "uuid2",
                                "time": {"value_as_string": "2024-02-09T13:00:00"},
                                "value": {"value": 56},
                            },
                        ]
                    },
                }
            },
            [
                {"run_uuid": "uuid1", "timestamp": "2024-02-09T12:00:00", "value_avg": 42},
                {"run_uuid": "uuid2", "timestamp": "2024-02-09T13:00:00", "value_avg": 56},
            ],
        ),
        # matcher_instance with no agg values (empty uuid buckets)
        (
            "matcher_instance",
            ["uuid1", "uuid2"],
            {
                "name": "apiserverCPU",
                "metricName": "containerCPU",
                "labels.namespace": "openshift-kube-apiserver",
                "metric_of_interest": "value",
                "agg": {"value": "cpu", "agg_type": "avg"},
            },
            {
                "aggregations": {"uuid": {"buckets": []}},
            },
            [],
        ),
        # matcher_instance with count aggregation
        (
            "matcher_instance",
            ["uuid1", "uuid2"],
            {
                "name": "api_requests",
                "metricName": "apiCalls",
                "metric_of_interest": "request_id",
                "agg": {"value": "request_id", "agg_type": "count"},
            },
            {
                "aggregations": {
                    "uuid": {
                        "buckets": [
                            {
                                "key": "uuid1",
                                "time": {"value_as_string": "2024-02-09T12:00:00"},
                                "request_id": {"value": 1250},
                            },
                            {
                                "key": "uuid2",
                                "time": {"value_as_string": "2024-02-09T13:00:00"},
                                "request_id": {"value": 1520},
                            },
                        ]
                    },
                }
            },
            [
                {"uuid": "uuid1", "timestamp": "2024-02-09T12:00:00", "request_id_count": 1250},
                {"uuid": "uuid2", "timestamp": "2024-02-09T13:00:00", "request_id_count": 1520},
            ],
        ),
    ],
)
def test_get_agg_metric_query_variants(request,
                                       fixture_name,
                                       test_uuids,
                                       test_metrics,
                                       data_dict,
                                       expected):
    matcher = request.getfixturevalue(fixture_name)
    def mock_execute(self):
        return Response(response=data_dict, search=self)
    with patch.object(Search, "execute", mock_execute):
        result = matcher.get_agg_metric_query(test_uuids, test_metrics)
    assert result == expected


@pytest.mark.parametrize(
    "fixture_name,test_uuids,test_metrics,data_dict,expected",
    [
        # Test percentile aggregation with no target_percentile
        # Should return all percents
        (
            "matcher_instance",
            ["uuid1", "uuid2"],
            {
                "name": "api_latency_p95",
                "metricName": "api_latency",
                "metric_of_interest": "response_time_ms",
                "agg": {
                    "value": "response_time_ms",
                    "agg_type": "percentiles",
                    "percents": [50, 95, 99],
                },
            },
            {
                "aggregations": {
                    "uuid": {
                        "buckets": [
                            {
                                "key": "uuid1",
                                "time": {"value_as_string": "2024-02-09T12:00:00"},
                                "response_time_ms": {"values": {"50.0": 100.5, "95.0": 250.3, "99.0": 350.7}},
                            },
                            {
                                "key": "uuid2",
                                "time": {"value_as_string": "2024-02-09T13:00:00"},
                                "response_time_ms": {"values": {"50.0": 105.2, "95.0": 260.8, "99.0": 360.1}},
                            },
                        ]
                    },
                }
            },
            [
                {
                    "uuid": "uuid1",
                    "timestamp": "2024-02-09T12:00:00",
                    "response_time_ms_percentiles_50.0": 100.5,
                    "response_time_ms_percentiles_95.0": 250.3,
                    "response_time_ms_percentiles_99.0": 350.7,
                },
                {
                    "uuid": "uuid2",
                    "timestamp": "2024-02-09T13:00:00",
                    "response_time_ms_percentiles_50.0": 105.2,
                    "response_time_ms_percentiles_95.0": 260.8,
                    "response_time_ms_percentiles_99.0": 360.1,
                },
            ],
        ),
        # Test percentile aggregation with custom target (99th percentile)
        (
            "matcher_instance",
            ["uuid1", "uuid2"],
            {
                "name": "api_latency_p99",
                "metricName": "api_latency",
                "metric_of_interest": "response_time_ms",
                "agg": {
                    "value": "response_time_ms",
                    "agg_type": "percentiles",
                    "percents": [50, 95, 99],
                    "target_percentile": 99,
                },
            },
            {
                "aggregations": {
                    "uuid": {
                        "buckets": [
                            {
                                "key": "uuid1",
                                "time": {"value_as_string": "2024-02-09T12:00:00"},
                                "response_time_ms": {"values": {"50.0": 100.5, "95.0": 250.3, "99.0": 350.7}},
                            },
                            {
                                "key": "uuid2",
                                "time": {"value_as_string": "2024-02-09T13:00:00"},
                                "response_time_ms": {"values": {"50.0": 105.2, "95.0": 260.8, "99.0": 360.1}},
                            },
                        ]
                    },
                }
            },
            [
                {
                    "uuid": "uuid1",
                    "timestamp": "2024-02-09T12:00:00",
                    "response_time_ms_percentiles_99.0": 350.7,
                },
                {
                    "uuid": "uuid2",
                    "timestamp": "2024-02-09T13:00:00",
                    "response_time_ms_percentiles_99.0": 360.1,
                },
            ],
        ),
        # Test percentile aggregation with uuid_matcher_instance
        (
            "uuid_matcher_instance",
            ["uuid1", "uuid2"],
            {
                "name": "latency_p95",
                "metricName": "latency",
                "metric_of_interest": "value_ms",
                "agg": {
                    "value": "value_ms",
                    "agg_type": "percentiles",
                    "percents": [95, 99],
                    "target_percentile": 95,
                },
            },
            {
                "aggregations": {
                    "uuid": {
                        "buckets": [
                            {
                                "key": "uuid1",
                                "time": {"value_as_string": "2024-02-09T12:00:00"},
                                "value_ms": {"values": {"95.0": 150.2, "99.0": 200.5}},
                            },
                            {
                                "key": "uuid2",
                                "time": {"value_as_string": "2024-02-09T13:00:00"},
                                "value_ms": {"values": {"95.0": 155.8, "99.0": 205.3}},
                            },
                        ]
                    },
                }
            },
            [
                {"run_uuid": "uuid1", "timestamp": "2024-02-09T12:00:00", "value_ms_percentiles_95.0": 150.2},
                {"run_uuid": "uuid2", "timestamp": "2024-02-09T13:00:00", "value_ms_percentiles_95.0": 155.8},
            ],
        ),
    ],
)
def test_percentile_agg_metric_query(request,
                                    fixture_name,
                                    test_uuids,
                                    test_metrics,
                                    data_dict,
                                    expected):
    """Test percentile aggregation queries."""
    matcher = request.getfixturevalue(fixture_name)
    def mock_execute(self):
        return Response(response=data_dict, search=self)
    with patch.object(Search, "execute", mock_execute):
        result = matcher.get_agg_metric_query(test_uuids, test_metrics)
    assert result == expected


class TestAggMetricQueryCaching:
    """Verify get_agg_metric_query uses per-uuid cache correctly."""

    def test_partial_cache_hit_only_queries_missing_uuids(self, tmp_path):
        """Call with [a, b] to populate cache, then call with [a, b, c].
        Assert the second ES call only queries uuid 'c', not 'a' or 'b',
        and the merged result contains rows for all three."""
        cache = QueryCache(db_path=str(tmp_path / "cache.db"))
        matcher = make_matcher_fixture(index="perf-scale-ci")
        matcher.cache = cache

        test_metrics = {
            "name": "apiserverCPU",
            "metricName": "containerCPU",
            "labels.namespace": "openshift-kube-apiserver",
            "metric_of_interest": "value",
            "agg": {"value": "cpu", "agg_type": "avg"},
        }

        # First call: uuids [a, b] - both are cache misses
        data_ab = {
            "aggregations": {
                "uuid": {
                    "buckets": [
                        {
                            "key": "a",
                            "time": {"value_as_string": "2024-02-09T12:00:00"},
                            "value": {"value": 42},
                        },
                        {
                            "key": "b",
                            "time": {"value_as_string": "2024-02-09T13:00:00"},
                            "value": {"value": 56},
                        },
                    ]
                }
            }
        }

        def mock_execute_ab(self):
            return Response(response=data_ab, search=self)

        with patch.object(Search, "execute", mock_execute_ab):
            result_ab = matcher.get_agg_metric_query(["a", "b"], test_metrics)

        assert len(result_ab) == 2

        # Second call: uuids [a, b, c] - a and b should be cached,
        # only c should be queried
        data_c = {
            "aggregations": {
                "uuid": {
                    "buckets": [
                        {
                            "key": "c",
                            "time": {"value_as_string": "2024-02-09T14:00:00"},
                            "value": {"value": 99},
                        },
                    ]
                }
            }
        }

        executed_searches = []

        def mock_execute_c(self):
            executed_searches.append(self.to_dict())
            return Response(response=data_c, search=self)

        with patch.object(Search, "execute", mock_execute_c):
            result_abc = matcher.get_agg_metric_query(
                ["a", "b", "c"], test_metrics
            )

        # The second ES call should only have uuid 'c' in its terms filter
        assert len(executed_searches) == 1
        search_dict = executed_searches[0]
        terms_filter = search_dict["query"]["bool"]["must"][0]["terms"]
        queried_uuids = terms_filter["uuid.keyword"]
        assert queried_uuids == ["c"], (
            f"Expected only ['c'] queried, got {queried_uuids}"
        )

        # Merged result should have all three uuids
        assert len(result_abc) == 3
        result_uuids = {row["uuid"] for row in result_abc}
        assert result_uuids == {"a", "b", "c"}


class TestMetadataContextCacheIsolation:
    """Verify that different metadata_context values produce separate cache entries."""

    @staticmethod
    def _make_agg_response(uuid_val, metric_value):
        return {
            "aggregations": {
                "uuid": {
                    "buckets": [
                        {
                            "key": uuid_val,
                            "time": {"value_as_string": "2024-02-09T12:00:00"},
                            "value": {"value": metric_value},
                        },
                    ]
                }
            }
        }

    def test_different_metadata_contexts_no_shared_cache(self, tmp_path):
        """Two calls with the same metric+uuid but different metadata_context
        must both hit ES (no cache sharing across metadata contexts)."""
        cache = QueryCache(db_path=str(tmp_path / "cache.db"))
        matcher = make_matcher_fixture(index="perf-scale-ci")
        matcher.cache = cache

        test_metrics = {
            "name": "cpu",
            "metricName": "containerCPU",
            "metric_of_interest": "value",
            "agg": {"value": "cpu", "agg_type": "avg"},
        }

        execute_calls = []

        def mock_exec_a(self):
            execute_calls.append("A")
            return Response(
                response=TestMetadataContextCacheIsolation._make_agg_response("u1", 10),
                search=self,
            )

        def mock_exec_b(self):
            execute_calls.append("B")
            return Response(
                response=TestMetadataContextCacheIsolation._make_agg_response("u1", 20),
                search=self,
            )

        # Call under metadata context A
        matcher.set_metadata_context({"platform": "AWS", "benchmark.keyword": "node-density"})
        with patch.object(Search, "execute", mock_exec_a):
            result_a = matcher.get_agg_metric_query(["u1"], test_metrics)

        # Call under metadata context B (different platform)
        matcher.set_metadata_context({"platform": "GCP", "benchmark.keyword": "node-density"})
        with patch.object(Search, "execute", mock_exec_b):
            result_b = matcher.get_agg_metric_query(["u1"], test_metrics)

        # Both should have triggered ES calls
        assert len(execute_calls) == 2, (
            f"Expected 2 ES calls (one per context), got {len(execute_calls)}"
        )
        assert result_a[0]["value_avg"] == 10
        assert result_b[0]["value_avg"] == 20

    def test_same_metadata_context_uses_cache(self, tmp_path):
        """Two calls with the same metric+uuid+metadata_context
        must use cache on the second call (only one ES hit)."""
        cache = QueryCache(db_path=str(tmp_path / "cache.db"))
        matcher = make_matcher_fixture(index="perf-scale-ci")
        matcher.cache = cache

        test_metrics = {
            "name": "cpu",
            "metricName": "containerCPU",
            "metric_of_interest": "value",
            "agg": {"value": "cpu", "agg_type": "avg"},
        }

        execute_calls = []

        def mock_exec(self):
            execute_calls.append("hit")
            return Response(
                response=TestMetadataContextCacheIsolation._make_agg_response("u1", 42),
                search=self,
            )

        matcher.set_metadata_context({"platform": "AWS"})

        with patch.object(Search, "execute", mock_exec):
            result1 = matcher.get_agg_metric_query(["u1"], test_metrics)

        with patch.object(Search, "execute", mock_exec):
            result2 = matcher.get_agg_metric_query(["u1"], test_metrics)

        # Only one ES call expected -- second should be a cache hit
        assert len(execute_calls) == 1, (
            f"Expected 1 ES call (second should cache-hit), got {len(execute_calls)}"
        )
        assert result1 == result2

    def test_get_results_different_metadata_contexts(self, tmp_path):
        """get_results with different metadata_context should not share cache."""
        cache = QueryCache(db_path=str(tmp_path / "cache.db"))
        matcher = make_matcher_fixture(index="perf-scale-ci")
        matcher.cache = cache

        test_metrics = {"metricName": "containerCPU", "name": "cpu"}
        uuids = ["u1"]

        hits_a = [{"_source": {"uuid": "u1", "value": 100}}]
        hits_b = [{"_source": {"uuid": "u1", "value": 200}}]

        execute_calls = []

        class FakeHit:
            def __init__(self, doc):
                self._doc = doc
            def to_dict(self):
                return self._doc

        def make_mock(hits, label):
            def mock_exec(self):
                execute_calls.append(label)
                resp_data = {"hits": {"hits": hits}}
                return Response(response=resp_data, search=self)
            return mock_exec

        matcher.set_metadata_context({"platform": "AWS"})
        with patch.object(Search, "execute", make_mock(hits_a, "A")):
            with patch.object(matcher, "query_index", return_value=[FakeHit(h) for h in hits_a]):
                res_a = matcher.get_results("u1", uuids.copy(), test_metrics)

        matcher.set_metadata_context({"platform": "GCP"})
        with patch.object(Search, "execute", make_mock(hits_b, "B")):
            with patch.object(matcher, "query_index", return_value=[FakeHit(h) for h in hits_b]):
                res_b = matcher.get_results("u1", uuids.copy(), test_metrics)

        assert len(execute_calls) == 0  # query_index is mocked, not execute
        assert res_a != res_b, "Different metadata contexts should yield different results"

    def test_default_metadata_context_is_deterministic(self, tmp_path):
        """A Matcher that never calls set_metadata_context should still
        cache correctly (default empty dict is stable)."""
        cache = QueryCache(db_path=str(tmp_path / "cache.db"))
        matcher = make_matcher_fixture(index="perf-scale-ci")
        matcher.cache = cache

        test_metrics = {
            "name": "cpu",
            "metricName": "containerCPU",
            "metric_of_interest": "value",
            "agg": {"value": "cpu", "agg_type": "avg"},
        }

        execute_calls = []

        def mock_exec(self):
            execute_calls.append("hit")
            return Response(
                response=TestMetadataContextCacheIsolation._make_agg_response("u1", 5),
                search=self,
            )

        # Never call set_metadata_context -- default {} should work
        with patch.object(Search, "execute", mock_exec):
            matcher.get_agg_metric_query(["u1"], test_metrics)

        with patch.object(Search, "execute", mock_exec):
            matcher.get_agg_metric_query(["u1"], test_metrics)

        assert len(execute_calls) == 1, (
            "Default metadata_context should be deterministic for caching"
        )
