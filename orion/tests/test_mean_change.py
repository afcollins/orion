"""
Unit tests for the Mean Change detection algorithm.
"""

# pylint: disable = redefined-outer-name
# pylint: disable = missing-function-docstring

import pytest
import pandas as pd
from orion.algorithms.meanchange import MeanChange


def _make_algorithm(values, direction=0, threshold=0, window=5):
    """Helper to build a MeanChange instance from a simple list of values."""
    n = len(values)
    df = pd.DataFrame({
        "timestamp": list(range(1_700_000_000, 1_700_000_000 + n)),
        "uuid": [f"uuid-{i}" for i in range(n)],
        "version": [f"v{i}" for i in range(n)],
        "metric_a": values,
    })
    test = {
        "name": "test-mean-change",
        "uuid_field": "uuid",
        "version_field": "version",
    }
    options = {
        "mean_change_window": window,
        "collapse": False,
    }
    metrics_config = {
        "metric_a": {
            "direction": direction,
            "threshold": threshold,
            "correlation": "",
            "context": 5,
            "labels": [],
            "metric_of_interest": "value",
        }
    }
    return MeanChange(df, test, options, metrics_config)


class TestMeanChangeDetection:
    """Tests for core mean change detection logic."""

    def test_clear_mean_shift(self):
        """A clear step from 1 to 5 should be detected."""
        algo = _make_algorithm(
            [1, 1, 1, 1, 1, 5, 5, 5, 5, 5],
            direction=0,
            threshold=0,
            window=5,
        )
        series, cp_by_metric = algo._analyze()
        cps = cp_by_metric["metric_a"]
        assert len(cps) == 1
        cp = cps[0]
        assert cp.index == 5
        assert cp.stats.mean_1 < cp.stats.mean_2

    def test_no_change(self):
        """Flat data should produce no changepoints."""
        algo = _make_algorithm(
            [3, 3, 3, 3, 3, 3, 3, 3, 3, 3],
            direction=0,
            threshold=0,
            window=5,
        )
        _, cp_by_metric = algo._analyze()
        assert len(cp_by_metric["metric_a"]) == 0

    def test_direction_up_is_bad(self):
        """direction=1 means increase is bad. A decrease should be ignored."""
        algo = _make_algorithm(
            [5, 5, 5, 5, 5, 1, 1, 1, 1, 1],
            direction=1,
            threshold=0,
            window=5,
        )
        _, cp_by_metric = algo._analyze()
        assert len(cp_by_metric["metric_a"]) == 0

    def test_direction_up_is_bad_detects_increase(self):
        """direction=1 should detect an increase."""
        algo = _make_algorithm(
            [1, 1, 1, 1, 1, 5, 5, 5, 5, 5],
            direction=1,
            threshold=0,
            window=5,
        )
        _, cp_by_metric = algo._analyze()
        assert len(cp_by_metric["metric_a"]) == 1

    def test_direction_down_is_bad(self):
        """direction=-1 means decrease is bad. An increase should be ignored."""
        algo = _make_algorithm(
            [1, 1, 1, 1, 1, 5, 5, 5, 5, 5],
            direction=-1,
            threshold=0,
            window=5,
        )
        _, cp_by_metric = algo._analyze()
        assert len(cp_by_metric["metric_a"]) == 0

    def test_direction_down_is_bad_detects_decrease(self):
        """direction=-1 should detect a decrease."""
        algo = _make_algorithm(
            [5, 5, 5, 5, 5, 1, 1, 1, 1, 1],
            direction=-1,
            threshold=0,
            window=5,
        )
        _, cp_by_metric = algo._analyze()
        assert len(cp_by_metric["metric_a"]) == 1

    def test_threshold_filters_small_change(self):
        """A 10% change should be filtered when threshold is 20."""
        algo = _make_algorithm(
            [100, 100, 100, 100, 100, 110, 110, 110, 110, 110],
            direction=0,
            threshold=20,
            window=5,
        )
        _, cp_by_metric = algo._analyze()
        assert len(cp_by_metric["metric_a"]) == 0

    def test_threshold_allows_large_change(self):
        """A 100% change should pass a 20% threshold."""
        algo = _make_algorithm(
            [100, 100, 100, 100, 100, 200, 200, 200, 200, 200],
            direction=0,
            threshold=20,
            window=5,
        )
        _, cp_by_metric = algo._analyze()
        assert len(cp_by_metric["metric_a"]) == 1

    def test_nearby_shifts_collapsed(self):
        """Two shifts within window distance get collapsed to the largest one."""
        algo = _make_algorithm(
            [1, 1, 1, 1, 1, 5, 5, 5, 5, 5, 20, 20, 20, 20, 20],
            direction=0,
            threshold=0,
            window=5,
        )
        _, cp_by_metric = algo._analyze()
        cps = cp_by_metric["metric_a"]
        # With window=5, the transitions at index 5 and 10 produce a continuous
        # chain of consecutive candidates, so they collapse to a single point.
        assert len(cps) == 1

    def test_multiple_changepoints_with_gap(self):
        """Two shifts separated by enough stable points yield two changepoints."""
        # Use a small window so the two shifts are far enough apart
        algo = _make_algorithm(
            [1, 1, 1, 5, 5, 5, 5, 5, 5, 20, 20, 20],
            direction=0,
            threshold=0,
            window=3,
        )
        _, cp_by_metric = algo._analyze()
        cps = cp_by_metric["metric_a"]
        assert len(cps) == 2
        indices = sorted([cp.index for cp in cps])
        assert indices[0] == 3
        assert indices[1] == 9

    def test_regression_flag_set(self):
        """regression_flag should be True when changepoints are found."""
        algo = _make_algorithm(
            [1, 1, 1, 1, 1, 5, 5, 5, 5, 5],
            direction=0,
            threshold=0,
            window=5,
        )
        algo._analyze()
        assert algo.regression_flag is True

    def test_regression_flag_not_set(self):
        """regression_flag should be False when no changepoints are found."""
        algo = _make_algorithm(
            [3, 3, 3, 3, 3, 3, 3, 3, 3, 3],
            direction=0,
            threshold=0,
            window=5,
        )
        algo._analyze()
        assert algo.regression_flag is False

    def test_small_window(self):
        """A small window size should still detect shifts."""
        algo = _make_algorithm(
            [1, 1, 5, 5],
            direction=0,
            threshold=0,
            window=2,
        )
        _, cp_by_metric = algo._analyze()
        assert len(cp_by_metric["metric_a"]) == 1

    def test_stats_populated(self):
        """ChangePoint stats should have correct mean and std values."""
        algo = _make_algorithm(
            [10, 10, 10, 10, 10, 20, 20, 20, 20, 20],
            direction=0,
            threshold=0,
            window=5,
        )
        _, cp_by_metric = algo._analyze()
        cp = cp_by_metric["metric_a"][0]
        assert cp.stats.mean_1 == pytest.approx(10.0)
        assert cp.stats.mean_2 == pytest.approx(20.0)
        assert cp.stats.std_1 == pytest.approx(0.0)
        assert cp.stats.std_2 == pytest.approx(0.0)

    def test_minimum_data(self):
        """Two data points should work without error."""
        algo = _make_algorithm(
            [1, 10],
            direction=0,
            threshold=0,
            window=5,
        )
        _, cp_by_metric = algo._analyze()
        assert len(cp_by_metric["metric_a"]) == 1

    def test_single_data_point(self):
        """A single data point should produce no changepoints."""
        algo = _make_algorithm(
            [5],
            direction=0,
            threshold=0,
            window=5,
        )
        _, cp_by_metric = algo._analyze()
        assert len(cp_by_metric["metric_a"]) == 0


class TestFilterConsecutive:
    """Tests for the _filter_consecutive static method."""

    def test_empty(self):
        assert MeanChange._filter_consecutive([]) == []

    def test_single_candidate(self):
        candidates = [(5, 100.0, 1.0, 2.0, 0.0, 0.0)]
        result = MeanChange._filter_consecutive(candidates)
        assert len(result) == 1
        assert result[0][0] == 5

    def test_consecutive_keeps_largest(self):
        candidates = [
            (3, 50.0, 1.0, 1.5, 0.0, 0.0),
            (4, 200.0, 1.0, 3.0, 0.0, 0.0),
            (5, 100.0, 1.0, 2.0, 0.0, 0.0),
        ]
        result = MeanChange._filter_consecutive(candidates)
        assert len(result) == 1
        assert result[0][0] == 4  # largest abs change

    def test_non_consecutive_kept_separately(self):
        candidates = [
            (2, 50.0, 1.0, 1.5, 0.0, 0.0),
            (5, 80.0, 1.0, 1.8, 0.0, 0.0),
            (9, 30.0, 1.0, 1.3, 0.0, 0.0),
        ]
        result = MeanChange._filter_consecutive(candidates)
        assert len(result) == 3
