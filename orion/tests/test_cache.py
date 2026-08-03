"""
Unit tests for orion.cache.QueryCache.
"""

# pylint: disable = redefined-outer-name
# pylint: disable = missing-function-docstring
# pylint: disable = import-error

import os
import threading

from orion.cache import QueryCache


# ---- get / put round-trip ---------------------------------------------------

class TestGetPut:
    """Whole-result cache: get/put round-trip."""

    def test_miss_returns_none(self, tmp_path):
        cache = QueryCache(db_path=str(tmp_path / "cache.db"))
        assert cache.get("nonexistent_key") is None

    def test_put_then_get_returns_same_value(self, tmp_path):
        cache = QueryCache(db_path=str(tmp_path / "cache.db"))
        value = {"uuid": "abc", "metric": 42, "nested": [1, 2, 3]}
        cache.put("key1", "test-index", "test-query", value)
        result = cache.get("key1")
        assert result == value

    def test_put_overwrites_existing_key(self, tmp_path):
        cache = QueryCache(db_path=str(tmp_path / "cache.db"))
        cache.put("k", "idx", "qt", {"v": 1})
        cache.put("k", "idx", "qt", {"v": 2})
        assert cache.get("k") == {"v": 2}


# ---- get_uuid_rows / put_uuid_rows ------------------------------------------

class TestUuidRows:
    """Per-uuid metric cache: partial hit/miss splitting."""

    def test_put_and_get_all_present(self, tmp_path):
        cache = QueryCache(db_path=str(tmp_path / "cache.db"))
        rows = [
            {"uuid": "u1", "value": 10},
            {"uuid": "u2", "value": 20},
            {"uuid": "u3", "value": 30},
        ]
        cache.put_uuid_rows("idx", "ns", "uuid", rows)
        result = cache.get_uuid_rows("idx", "ns", ["u1", "u2", "u3"])
        assert len(result) == 3
        assert result["u1"] == {"uuid": "u1", "value": 10}
        assert result["u3"] == {"uuid": "u3", "value": 30}

    def test_partial_hit_miss(self, tmp_path):
        cache = QueryCache(db_path=str(tmp_path / "cache.db"))
        rows = [
            {"uuid": "u1", "value": 10},
            {"uuid": "u2", "value": 20},
        ]
        cache.put_uuid_rows("idx", "ns", "uuid", rows)
        result = cache.get_uuid_rows("idx", "ns", ["u1", "u3", "u4"])
        assert set(result.keys()) == {"u1"}
        assert result["u1"] == {"uuid": "u1", "value": 10}

    def test_empty_uuids_returns_empty(self, tmp_path):
        cache = QueryCache(db_path=str(tmp_path / "cache.db"))
        result = cache.get_uuid_rows("idx", "ns", [])
        assert result == {}

    def test_rows_without_uuid_field_skipped(self, tmp_path):
        cache = QueryCache(db_path=str(tmp_path / "cache.db"))
        rows = [
            {"uuid": "u1", "value": 10},
            {"no_uuid": "missing", "value": 20},
        ]
        cache.put_uuid_rows("idx", "ns", "uuid", rows)
        result = cache.get_uuid_rows("idx", "ns", ["u1"])
        assert len(result) == 1

    def test_different_namespaces_isolated(self, tmp_path):
        cache = QueryCache(db_path=str(tmp_path / "cache.db"))
        cache.put_uuid_rows("idx", "ns-a", "uuid", [{"uuid": "u1", "value": "a"}])
        cache.put_uuid_rows("idx", "ns-b", "uuid", [{"uuid": "u1", "value": "b"}])
        assert cache.get_uuid_rows("idx", "ns-a", ["u1"])["u1"]["value"] == "a"
        assert cache.get_uuid_rows("idx", "ns-b", ["u1"])["u1"]["value"] == "b"


# ---- clear -------------------------------------------------------------------

class TestClear:
    """Verify clear() empties both tables."""

    def test_clear_empties_query_cache(self, tmp_path):
        cache = QueryCache(db_path=str(tmp_path / "cache.db"))
        cache.put("k1", "idx", "qt", {"v": 1})
        cache.clear()
        assert cache.get("k1") is None

    def test_clear_empties_uuid_metric_cache(self, tmp_path):
        cache = QueryCache(db_path=str(tmp_path / "cache.db"))
        cache.put_uuid_rows("idx", "ns", "uuid", [{"uuid": "u1", "v": 1}])
        cache.clear()
        assert cache.get_uuid_rows("idx", "ns", ["u1"]) == {}

    def test_clear_empties_both_tables(self, tmp_path):
        cache = QueryCache(db_path=str(tmp_path / "cache.db"))
        cache.put("k1", "idx", "qt", "val")
        cache.put_uuid_rows("idx", "ns", "uuid", [{"uuid": "u1", "v": 1}])
        cache.clear()
        assert cache.get("k1") is None
        assert cache.get_uuid_rows("idx", "ns", ["u1"]) == {}


# ---- enabled=False (disabled mode) ------------------------------------------

class TestDisabledMode:
    """With enabled=False, all operations are safe no-ops."""

    def test_no_db_file_created(self, tmp_path):
        db_path = str(tmp_path / "should_not_exist.db")
        QueryCache(db_path=db_path, enabled=False)
        assert not os.path.exists(db_path)

    def test_get_returns_none(self, tmp_path):
        cache = QueryCache(db_path=str(tmp_path / "x.db"), enabled=False)
        assert cache.get("any_key") is None

    def test_put_is_noop(self, tmp_path):
        db_path = str(tmp_path / "x.db")
        cache = QueryCache(db_path=db_path, enabled=False)
        cache.put("k", "idx", "qt", {"v": 1})
        assert not os.path.exists(db_path)

    def test_get_uuid_rows_returns_empty(self, tmp_path):
        cache = QueryCache(db_path=str(tmp_path / "x.db"), enabled=False)
        assert cache.get_uuid_rows("idx", "ns", ["u1", "u2"]) == {}

    def test_put_uuid_rows_is_noop(self, tmp_path):
        db_path = str(tmp_path / "x.db")
        cache = QueryCache(db_path=db_path, enabled=False)
        cache.put_uuid_rows("idx", "ns", "uuid", [{"uuid": "u1", "v": 1}])
        assert not os.path.exists(db_path)

    def test_clear_is_noop(self, tmp_path):
        cache = QueryCache(db_path=str(tmp_path / "x.db"), enabled=False)
        cache.clear()  # should not raise


# ---- make_key ----------------------------------------------------------------

class TestMakeKey:
    """Verify determinism and collision resistance of make_key."""

    def test_same_inputs_same_key(self):
        k1 = QueryCache.make_key("index", "query", ["u1", "u2"])
        k2 = QueryCache.make_key("index", "query", ["u1", "u2"])
        assert k1 == k2

    def test_different_inputs_different_key(self):
        k1 = QueryCache.make_key("index", "query_a")
        k2 = QueryCache.make_key("index", "query_b")
        assert k1 != k2

    def test_key_is_hex_string(self):
        key = QueryCache.make_key("test")
        assert len(key) == 64
        assert all(c in "0123456789abcdef" for c in key)

    def test_nested_dicts_and_lists(self):
        metric_config = {
            "name": "apiserverCPU",
            "metricName": "containerCPU",
            "labels": {"namespace": "openshift-kube-apiserver"},
            "agg": {"value": "cpu", "agg_type": "avg"},
            "filters": [1, 2, 3],
        }
        key = QueryCache.make_key(metric_config)
        assert isinstance(key, str) and len(key) == 64

    def test_order_of_dict_keys_does_not_matter(self):
        k1 = QueryCache.make_key({"a": 1, "b": 2})
        k2 = QueryCache.make_key({"b": 2, "a": 1})
        assert k1 == k2


# ---- thread safety -----------------------------------------------------------

class TestThreadSafety:
    """Basic thread-safety smoke test: concurrent puts/gets don't crash."""

    def test_concurrent_puts_and_gets(self, tmp_path):
        cache = QueryCache(db_path=str(tmp_path / "cache.db"))
        errors = []

        def worker(thread_id):
            try:
                for i in range(20):
                    key = f"t{thread_id}_k{i}"
                    cache.put(key, "idx", "qt", {"thread": thread_id, "i": i})
                    result = cache.get(key)
                    assert result is not None, f"get returned None for {key}"
            except Exception as exc:  # pylint: disable=broad-except
                errors.append(exc)

        threads = [threading.Thread(target=worker, args=(t,)) for t in range(4)]
        for t in threads:
            t.start()
        for t in threads:
            t.join()

        assert not errors, f"Thread errors: {errors}"

        # Verify some data is retrievable afterward
        assert cache.get("t0_k0") is not None
        assert cache.get("t3_k19") is not None

    def test_concurrent_uuid_row_operations(self, tmp_path):
        cache = QueryCache(db_path=str(tmp_path / "cache.db"))
        errors = []

        def worker(thread_id):
            try:
                rows = [{"uuid": f"t{thread_id}_u{i}", "v": i} for i in range(10)]
                cache.put_uuid_rows("idx", f"ns_{thread_id}", "uuid", rows)
                result = cache.get_uuid_rows(
                    "idx", f"ns_{thread_id}",
                    [f"t{thread_id}_u{i}" for i in range(10)]
                )
                assert len(result) == 10
            except Exception as exc:  # pylint: disable=broad-except
                errors.append(exc)

        threads = [threading.Thread(target=worker, args=(t,)) for t in range(4)]
        for t in threads:
            t.start()
        for t in threads:
            t.join()

        assert not errors, f"Thread errors: {errors}"
