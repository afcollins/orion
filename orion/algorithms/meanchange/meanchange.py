"""Mean Change detection algorithm"""

# pylint: disable = line-too-long
import logging
import numpy as np
import pandas as pd
from otava.analysis import TTestStats
from otava.series import ChangePoint
from orion.algorithms.algorithm import Algorithm

logger = logging.getLogger("Orion")


class MeanChange(Algorithm):
    """Sliding window mean change detection.

    Detects points where the mean of a metric shifts significantly
    between consecutive segments of the time series.

    Args:
        Algorithm (Algorithm): Inherits
    """

    def _analyze(self):
        logger.info("Starting analysis using Mean Change")

        if not (pd.api.types.is_numeric_dtype(self.dataframe["timestamp"]) and self.dataframe["timestamp"].astype(int).min() > 1e9):
            self.dataframe["timestamp"] = pd.to_datetime(self.dataframe["timestamp"])
            self.dataframe["timestamp"] = self.dataframe["timestamp"].astype(int) // 10**9

        series = self.setup_series()
        metric_columns = list(self.metrics_config.keys())
        window_size = int(self.options.get("mean_change_window") or 5)

        change_points_by_metric = {k: [] for k in metric_columns}

        for metric in metric_columns:
            values = self.dataframe[metric].values
            n = len(values)
            if n < 2:
                continue

            candidates = []
            for i in range(1, n):
                start_before = max(0, i - window_size)
                end_after = min(n, i + window_size)

                segment_before = values[start_before:i]
                segment_after = values[i:end_after]

                mean_before = np.mean(segment_before)
                mean_after = np.mean(segment_after)
                std_before = float(np.std(segment_before, ddof=0))
                std_after = float(np.std(segment_after, ddof=0))

                if mean_before == 0:
                    continue

                pct_change = ((mean_after - mean_before) / mean_before) * 100
                threshold = self.metrics_config[metric].get("threshold", 0)
                direction = self.metrics_config[metric].get("direction", 0)

                if abs(pct_change) <= threshold:
                    continue

                if direction == 1 and pct_change <= 0:
                    continue
                if direction == -1 and pct_change >= 0:
                    continue

                candidates.append((i, abs(pct_change), mean_before, mean_after, std_before, std_after))

            # Among consecutive candidates, keep only the one with the largest change
            filtered = self._filter_consecutive(candidates)

            for idx, _, mean_before, mean_after, std_before, std_after in filtered:
                change_point = ChangePoint(
                    index=idx,
                    qhat=0.0,
                    metric=metric,
                    time=int(self.dataframe["timestamp"].iloc[idx]),
                    stats=TTestStats(
                        mean_1=mean_before,
                        mean_2=mean_after,
                        std_1=std_before,
                        std_2=std_after,
                        pvalue=1.0,
                    ),
                )
                change_points_by_metric[metric].append(change_point)

        if [val for li in change_points_by_metric.values() for val in li]:
            self.regression_flag = True

        return series, change_points_by_metric

    @staticmethod
    def _filter_consecutive(candidates):
        """Keep only the candidate with the largest absolute change
        among groups of consecutive indices."""
        if not candidates:
            return []

        groups = []
        current_group = [candidates[0]]

        for c in candidates[1:]:
            if c[0] == current_group[-1][0] + 1:
                current_group.append(c)
            else:
                groups.append(current_group)
                current_group = [c]
        groups.append(current_group)

        return [max(group, key=lambda x: x[1]) for group in groups]
