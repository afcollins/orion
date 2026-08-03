"""
Unit tests for orion/utils.py
"""

# pylint: disable = redefined-outer-name
# pylint: disable = missing-function-docstring
# pylint: disable = import-error
# pylint: disable = missing-class-docstring

import logging
from unittest.mock import patch, MagicMock

import pandas as pd
import pytest

from orion.cache import QueryCache
from orion.logger import SingletonLogger
from orion.utils import (
    Utils,
)


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------

@pytest.fixture(autouse=True)
def _init_logger():
    """Ensure the singleton logger exists for every test."""
    SingletonLogger(debug=logging.DEBUG, name="Orion")


@pytest.fixture
def utils():
    return Utils()


@pytest.fixture
def utils_custom_fields():
    return Utils(uuid_field="run_uuid", version_field="cluster_version")


# ---------------------------------------------------------------------------
# standardize_timestamp
# ---------------------------------------------------------------------------

class TestStandardizeTimestamp:
    def test_none_returns_none(self, utils):
        assert utils.standardize_timestamp(None) is None

    def test_integer_epoch_seconds(self, utils):
        # 2024-01-01 00:00:00 UTC
        result = utils.standardize_timestamp(1704067200)
        assert result == "2024-01-01T00:00:00"

    def test_numeric_string_epoch_seconds(self, utils):
        result = utils.standardize_timestamp("1704067200")
        assert result == "2024-01-01T00:00:00"

    def test_iso_string(self, utils):
        result = utils.standardize_timestamp("2024-06-15T12:30:45Z")
        assert result == "2024-06-15T12:30:45"

    def test_iso_string_with_offset(self, utils):
        result = utils.standardize_timestamp("2024-06-15T12:30:45+00:00")
        assert result == "2024-06-15T12:30:45"

    def test_float_epoch_not_treated_as_seconds(self, utils):
        # BEHAVIOR GUARD: floats take the else branch in standardize_timestamp,
        # where pd.to_datetime interprets them as nanoseconds, NOT seconds.
        # A float like 1.7e9 (valid epoch seconds for 2024) gets interpreted
        # as ~1.7 seconds after 1970-01-01.
        #
        # If this test fails it means the float handling changed — callers
        # that rely on passing int/str for epoch-seconds may now silently
        # get wrong results from floats, or vice-versa. Either way, audit
        # all call sites before accepting the new behavior.
        ts = 1704067200.123  # 2024-01-01 as epoch seconds
        result = utils.standardize_timestamp(ts)
        assert result.startswith("1970-01-01"), (
            f"Float timestamp handling changed! Got {result} — expected 1970 "
            f"(float treated as nanoseconds). If this is intentional, update "
            f"this test and audit callers of standardize_timestamp."
        )

    def test_pandas_timestamp(self, utils):
        ts = pd.Timestamp("2024-03-15 08:00:00", tz="UTC")
        result = utils.standardize_timestamp(ts)
        assert result == "2024-03-15T08:00:00"


# ---------------------------------------------------------------------------
# sippy_pr_search gating (Task A)
# ---------------------------------------------------------------------------

class TestSippyPrSearchGating:
    """When sippy_pr_search is False/absent, process_test must not make
    any HTTP calls to sippy — versions should still be populated but
    prs should be empty lists."""

    def test_map_prs_version_disabled_skips_http(self, utils):
        """map_prs_version with enabled=False returns empty PR lists and
        never calls sippy_pr_search."""
        mock_match = MagicMock()
        # get_version returns a uuid->version mapping
        mock_match.get_results.return_value = [
            {"uuid": "u1", "ocpVersion": "4.15.0"},
            {"uuid": "u2", "ocpVersion": "4.15.1"},
        ]
        with patch.object(utils, "sippy_pr_search") as mock_sippy:
            versions, prs = utils.map_prs_version(
                ["u1", "u2"], mock_match, enabled=False
            )
            mock_sippy.assert_not_called()
        assert versions == {"u1": "4.15.0", "u2": "4.15.1"}
        assert prs == {"u1": [], "u2": []}

    def test_map_prs_version_enabled_calls_sippy(self, utils):
        """map_prs_version with enabled=True (default) calls sippy_pr_search
        for each version."""
        mock_match = MagicMock()
        mock_match.get_results.return_value = [
            {"uuid": "u1", "ocpVersion": "4.15.0"},
        ]
        with patch.object(utils, "sippy_pr_search", return_value=["pr1"]) as mock_sippy:
            _, prs = utils.map_prs_version(
                ["u1"], mock_match, enabled=True
            )
            mock_sippy.assert_called_once_with("4.15.0")
        assert prs == {"u1": ["pr1"]}


# ---------------------------------------------------------------------------
# sippy_pr_search caching (Task B)
# ---------------------------------------------------------------------------

class TestSippyPrSearchCaching:
    """Cache behavior for sippy_pr_search."""

    @pytest.fixture
    def cached_utils(self, tmp_path):
        db_path = str(tmp_path / "test_cache.db")
        cache = QueryCache(db_path=db_path, enabled=True)
        return Utils(cache=cache)

    @patch("orion.utils.requests.get")
    def test_cache_hit_skips_http(self, mock_get, cached_utils):
        """First call fetches from HTTP and caches; second call uses cache."""
        mock_response = MagicMock()
        mock_response.status_code = 200
        mock_response.json.return_value = [
            {"url": "https://github.com/org/repo/pull/1"},
            {"url": "https://github.com/org/repo/pull/2"},
        ]
        mock_get.return_value = mock_response

        # First call — HTTP fetch
        result1 = cached_utils.sippy_pr_search("4.15.0")
        assert mock_get.call_count == 1
        assert result1 == [
            "https://github.com/org/repo/pull/1",
            "https://github.com/org/repo/pull/2",
        ]

        # Second call — should use cache, no new HTTP call
        result2 = cached_utils.sippy_pr_search("4.15.0")
        assert mock_get.call_count == 1  # still 1, not 2
        assert result2 == result1

    @patch("orion.utils.requests.get")
    def test_failed_response_not_cached(self, mock_get, cached_utils):
        """A non-200 response must NOT be cached — subsequent calls
        should retry the HTTP request."""
        mock_response = MagicMock()
        mock_response.status_code = 503
        mock_get.return_value = mock_response

        result1 = cached_utils.sippy_pr_search("4.15.0")
        assert result1 == []

        result2 = cached_utils.sippy_pr_search("4.15.0")
        assert result2 == []

        # Both calls should have hit the network
        assert mock_get.call_count == 2

    @patch("orion.utils.requests.get")
    def test_empty_success_is_cached(self, mock_get, cached_utils):
        """A 200 response with zero PRs should be cached (version
        legitimately has no PRs)."""
        mock_response = MagicMock()
        mock_response.status_code = 200
        mock_response.json.return_value = []
        mock_get.return_value = mock_response

        result1 = cached_utils.sippy_pr_search("4.15.0")
        assert result1 == []

        result2 = cached_utils.sippy_pr_search("4.15.0")
        assert result2 == []

        # Only one HTTP call; second was served from cache
        assert mock_get.call_count == 1
