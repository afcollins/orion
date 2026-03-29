package config

import "strings"

// QuoteJinja2 pre-processes YAML content so that unquoted Jinja2 expressions
// ({{ ... }}) are wrapped in double quotes, making the content valid YAML.
// Values already wrapped in single or double quotes are left untouched.
//
// Example:
//
//	ocpVersion: {{ version }}         → ocpVersion: "{{ version }}"
//	jobType: "{{ jobtype }}"          → jobType: "{{ jobtype }}"   (unchanged)
//	name: node-density-{{ workers }}w → name: "node-density-{{ workers }}w"
func QuoteJinja2(content string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if !strings.Contains(line, "{{") {
			continue
		}
		colonIdx := strings.Index(line, ":")
		if colonIdx < 0 {
			continue
		}
		after := line[colonIdx+1:] // everything after the first colon
		trimmed := strings.TrimSpace(after)
		if trimmed == "" {
			continue
		}
		// Already quoted with double or single quotes — leave it alone.
		if (strings.HasPrefix(trimmed, `"`) && strings.HasSuffix(trimmed, `"`)) ||
			(strings.HasPrefix(trimmed, `'`) && strings.HasSuffix(trimmed, `'`)) {
			continue
		}
		// Preserve the whitespace between the colon and the value.
		leading := after[:len(after)-len(strings.TrimLeft(after, " \t"))]
		lines[i] = line[:colonIdx+1] + leading + `"` + trimmed + `"`
	}
	return strings.Join(lines, "\n")
}
