"""
Local SQLite cache for OpenSearch query results.

Caches immutable performance-test data so repeated orion invocations
against the same index/uuids skip redundant OpenSearch round-trips.
Storage location respects XDG_DATA_HOME (default ~/.local/share/orion/cache.db).
"""

import hashlib
import json
import logging
import os
import sqlite3
import threading
import time
from typing import Any, Dict, List, Optional

logger = logging.getLogger("Orion")


_SCHEMA_SQL = """\
CREATE TABLE IF NOT EXISTS query_cache (
    cache_key   TEXT PRIMARY KEY,
    index_name  TEXT NOT NULL,
    query_type  TEXT NOT NULL,
    payload     TEXT NOT NULL,
    created_at  REAL NOT NULL
);

CREATE TABLE IF NOT EXISTS uuid_metric_cache (
    index_name  TEXT NOT NULL,
    namespace   TEXT NOT NULL,
    uuid        TEXT NOT NULL,
    payload     TEXT NOT NULL,
    created_at  REAL NOT NULL,
    PRIMARY KEY (index_name, namespace, uuid)
);
"""


def _default_db_path() -> str:
    """Return the default cache database path, respecting XDG_DATA_HOME."""
    data_home = os.environ.get("XDG_DATA_HOME") or os.path.join(
        os.path.expanduser("~"), ".local", "share"
    )
    return os.path.join(data_home, "orion", "cache.db")


class QueryCache:
    """SQLite-backed cache for OpenSearch query results.

    Stores two kinds of cached data:

    * **query_cache** -- whole-result blobs keyed by a content hash
      (used for ``discover_field_values``, ``get_metadata_by_uuid``).
    * **uuid_metric_cache** -- per-uuid metric rows keyed by
      ``(index, namespace, uuid)`` (used for batch metric fetchers).

    Thread-safety is achieved via one ``sqlite3`` connection per thread
    (``threading.local``), all pointing at the same on-disk database
    with WAL journaling enabled.

    Args:
        db_path: Filesystem path for the SQLite database.  Defaults to
            ``$XDG_DATA_HOME/orion/cache.db`` (or
            ``~/.local/share/orion/cache.db`` when the env var is unset).
        enabled: When *False*, every public method becomes a safe no-op
            and no database file is created.
    """

    def __init__(self, db_path: Optional[str] = None, enabled: bool = True):
        self._enabled = enabled
        self._db_path = db_path or _default_db_path()
        self._local = threading.local()

        if self._enabled:
            # Ensure parent directory exists before first connect.
            os.makedirs(os.path.dirname(self._db_path), exist_ok=True)
            # Eagerly initialise the connection for the constructing thread
            # so that the schema is guaranteed to exist before any get/put.
            self._conn()

    # ------------------------------------------------------------------
    # Internal helpers
    # ------------------------------------------------------------------

    def _conn(self) -> sqlite3.Connection:
        """Return a per-thread ``sqlite3.Connection``, creating it if needed."""
        conn: Optional[sqlite3.Connection] = getattr(
            self._local, "conn", None
        )
        if conn is None:
            conn = sqlite3.connect(self._db_path)
            conn.execute("PRAGMA journal_mode=WAL")
            conn.executescript(_SCHEMA_SQL)
            self._local.conn = conn
        return conn

    # ------------------------------------------------------------------
    # Whole-result cache
    # ------------------------------------------------------------------

    def get(self, cache_key: str) -> Optional[Any]:
        """Look up a cached query result.

        Args:
            cache_key: The content-addressed key (typically from
                :meth:`make_key`).

        Returns:
            The deserialised payload, or *None* if the key is not cached.
        """
        if not self._enabled:
            return None
        row = self._conn().execute(
            "SELECT payload FROM query_cache WHERE cache_key = ?",
            (cache_key,),
        ).fetchone()
        if row is None:
            logger.debug("Cache miss (query_cache) key=%s…", cache_key[:12])
            return None
        logger.debug("Cache hit  (query_cache) key=%s…", cache_key[:12])
        return json.loads(row[0])

    def put(
        self, cache_key: str, index: str, query_type: str, value: Any
    ) -> None:
        """Store (or update) a whole-result cache entry.

        Args:
            cache_key: Content-addressed key.
            index: OpenSearch index name (stored for diagnostics).
            query_type: Short label describing the query origin.
            value: Payload to cache (must be JSON-serialisable).
        """
        if not self._enabled:
            return
        conn = self._conn()
        logger.debug("Cache store (query_cache) key=%s… type=%s", cache_key[:12], query_type)
        conn.execute(
            "INSERT OR REPLACE INTO query_cache "
            "(cache_key, index_name, query_type, payload, created_at) "
            "VALUES (?, ?, ?, ?, ?)",
            (cache_key, index, query_type, json.dumps(value), time.time()),
        )
        conn.commit()

    # ------------------------------------------------------------------
    # Per-uuid metric cache
    # ------------------------------------------------------------------

    def get_uuid_rows(
        self, index: str, namespace: str, uuids: List[str]
    ) -> Dict[str, Any]:
        """Retrieve cached per-uuid metric rows.

        Args:
            index: OpenSearch index name.
            namespace: Cache namespace (hash of metric config).
            uuids: UUIDs to look up.

        Returns:
            Dict mapping each *found* uuid to its deserialised payload.
            UUIDs not present in the cache are simply absent from the dict.
        """
        if not self._enabled or not uuids:
            return {}
        conn = self._conn()
        placeholders = ",".join("?" for _ in uuids)
        rows = conn.execute(
            "SELECT uuid, payload FROM uuid_metric_cache "
            f"WHERE index_name = ? AND namespace = ? AND uuid IN ({placeholders})",
            [index, namespace, *uuids],
        ).fetchall()
        logger.debug(
            "Cache lookup (uuid_metric) ns=%s… requested=%d found=%d",
            namespace[:12], len(uuids), len(rows),
        )
        return {uuid: json.loads(payload) for uuid, payload in rows}

    def put_uuid_rows(
        self,
        index: str,
        namespace: str,
        uuid_field: str,
        rows: List[dict],
    ) -> None:
        """Cache per-uuid metric rows.

        Args:
            index: OpenSearch index name.
            namespace: Cache namespace (hash of metric config).
            uuid_field: Key to extract the uuid value from each row dict.
            rows: List of row dicts.  Rows missing *uuid_field* are skipped.
        """
        if not self._enabled or not rows:
            return
        conn = self._conn()
        stored = 0
        now = time.time()
        for row in rows:
            uuid = row.get(uuid_field)
            if uuid is None:
                continue
            conn.execute(
                "INSERT OR REPLACE INTO uuid_metric_cache "
                "(index_name, namespace, uuid, payload, created_at) "
                "VALUES (?, ?, ?, ?, ?)",
                (index, namespace, uuid, json.dumps(row), now),
            )
            stored += 1
        conn.commit()
        logger.debug("Cache store (uuid_metric) ns=%s… rows=%d", namespace[:12], stored)

    # ------------------------------------------------------------------
    # Maintenance
    # ------------------------------------------------------------------

    def clear(self) -> None:
        """Delete all cached data from both tables."""
        if not self._enabled:
            return
        conn = self._conn()
        conn.execute("DELETE FROM query_cache")
        conn.execute("DELETE FROM uuid_metric_cache")
        conn.commit()

    # ------------------------------------------------------------------
    # Key construction
    # ------------------------------------------------------------------

    @staticmethod
    def make_key(*parts) -> str:
        """Build a stable, content-addressed cache key.

        JSON-serialises *parts* with sorted keys and returns the SHA-256
        hex digest.  Callers should pre-sort any lists whose order is
        not semantically meaningful (e.g. ``sorted(uuids)``).

        Args:
            *parts: Arbitrary JSON-serialisable values that together
                identify a unique query.

        Returns:
            A 64-character lowercase hex string (SHA-256 digest).
        """
        blob = json.dumps(parts, sort_keys=True).encode()
        return hashlib.sha256(blob).hexdigest()
