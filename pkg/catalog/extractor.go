package catalog

import (
	"regexp"
	"strings"
)

// byClauseRe matches `by (label1, label2, ...)` or `by(label1,label2)` in Prometheus queries.
var byClauseRe = regexp.MustCompile(`\bby\s*\(([^)]+)\)`)

// ExtractLabels extracts label names from the by() clause(s) in a Prometheus query.
// For histogram_quantile queries, the "le" label is excluded since it's a bucket
// boundary, not a user-facing dimension.
func ExtractLabels(query string) []string {
	matches := byClauseRe.FindAllStringSubmatch(query, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]bool)
	var labels []string
	for _, match := range matches {
		parts := strings.Split(match[1], ",")
		for _, p := range parts {
			label := strings.TrimSpace(p)
			if label == "" || label == "le" {
				continue
			}
			if !seen[label] {
				seen[label] = true
				labels = append(labels, label)
			}
		}
	}
	return labels
}
