package config

// Golden file tests for the config generator.
//
// Each test encodes an example config as a Go recipe (TestSpec/MetricSpec),
// generates the config, and compares it against the marshaled output of parsing
// the original example file. Both go through the same Marshal path so field
// ordering is canonical and comparable.
//
// To regenerate golden files after an intentional change:
//
//	go test ./pkg/config/... -run TestGolden -update
//
// Golden files live in testdata/golden/ and should be committed to source control.

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "regenerate golden files")

// goldenTest is the shared harness: it compares `got` (recipe output) against
// the golden file at testdata/golden/<name>.yaml.
// If -update is set, it regenerates the golden file by parsing examplePath and
// marshaling the result — making the example file the source of truth.
func goldenTest(t *testing.T, name string, examplePath string, got []byte) {
	t.Helper()
	goldenPath := filepath.Join("testdata", "golden", name+".yaml")

	if *update {
		golden := goldenFromExample(t, examplePath)
		if err := os.WriteFile(goldenPath, golden, 0644); err != nil {
			t.Fatalf("writing golden file: %v", err)
		}
		t.Logf("updated %s from %s", goldenPath, examplePath)
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("reading golden file %s (run with -update to generate): %v", goldenPath, err)
	}

	if normalizeWhitespace(string(got)) == normalizeWhitespace(string(want)) {
		return
	}

	// Print a diff using the system diff command if available, otherwise show both.
	t.Errorf("output does not match golden file %s", goldenPath)
	diff := systemDiff(want, got)
	if diff != "" {
		t.Logf("diff (want → got):\n%s", diff)
	} else {
		t.Logf("want:\n%s\ngot:\n%s", want, got)
	}
}

// normalizeWhitespace trims trailing whitespace from each line and collapses
// blank lines, matching git diff -w semantics for comparison purposes.
func normalizeWhitespace(s string) string {
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		trimmed := strings.TrimRight(l, " \t")
		out = append(out, trimmed)
	}
	// Trim leading/trailing blank lines.
	return strings.TrimSpace(strings.Join(out, "\n"))
}

// systemDiff runs the system `diff` command between want and got bytes.
// Returns empty string if diff is not available.
func systemDiff(want, got []byte) string {
	wantFile, err := os.CreateTemp("", "taurus-want-*.yaml")
	if err != nil {
		return ""
	}
	defer os.Remove(wantFile.Name())
	wantFile.Write(want)
	wantFile.Close()

	gotFile, err := os.CreateTemp("", "taurus-got-*.yaml")
	if err != nil {
		return ""
	}
	defer os.Remove(gotFile.Name())
	gotFile.Write(got)
	gotFile.Close()

	out, _ := exec.Command("diff", "-u", "--label", "want", "--label", "got",
		wantFile.Name(), gotFile.Name()).Output()
	return string(out)
}

// goldenFromExample parses an example file and marshals it to produce the
// canonical golden bytes. Used to generate golden files from existing examples.
func goldenFromExample(t *testing.T, examplePath string) []byte {
	t.Helper()
	cfg, err := LoadFile(examplePath)
	if err != nil {
		t.Fatalf("loading example %s: %v", examplePath, err)
	}
	data, err := Marshal(cfg)
	if err != nil {
		t.Fatalf("marshaling example %s: %v", examplePath, err)
	}
	return data
}
