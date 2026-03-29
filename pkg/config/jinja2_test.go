package config

import "testing"

func TestQuoteJinja2(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "unquoted simple template",
			input:  "ocpVersion: {{ version }}",
			expect: `ocpVersion: "{{ version }}"`,
		},
		{
			name:   "already double-quoted — unchanged",
			input:  `ocpVersion: "{{ version }}"`,
			expect: `ocpVersion: "{{ version }}"`,
		},
		{
			name:   "already single-quoted — unchanged",
			input:  "ocpVersion: '{{ version }}'",
			expect: "ocpVersion: '{{ version }}'",
		},
		{
			name:   "template with filter and default single-quote inside",
			input:  "jobType: {{ jobtype | default('periodic') }}",
			expect: `jobType: "{{ jobtype | default('periodic') }}"`,
		},
		{
			name:   "partial template in value",
			input:  "name: node-density-{{ workers }}w-5m",
			expect: `name: "node-density-{{ workers }}w-5m"`,
		},
		{
			name:   "no template — unchanged",
			input:  "platform: AWS",
			expect: "platform: AWS",
		},
		{
			name:   "empty value — unchanged",
			input:  "key:",
			expect: "key:",
		},
		{
			name:   "indented with leading spaces preserved",
			input:  "  ocpVersion: {{ version }}",
			expect: `  ocpVersion: "{{ version }}"`,
		},
		{
			name: "multiline — only template lines quoted",
			input: "platform: AWS\nocpVersion: {{ version }}\nworkers: 24",
			expect: "platform: AWS\nocpVersion: \"{{ version }}\"\nworkers: 24",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := QuoteJinja2(tc.input)
			if got != tc.expect {
				t.Errorf("\ngot:  %s\nwant: %s", got, tc.expect)
			}
		})
	}
}
