package audit_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The audit gates all ask the same question of the module's own tree —
// which files, and what in them — and each used to carry its own walk to
// ask it. This file holds the one walk and the vocabulary the gates
// compose over it: a predicate saying which files to read, and a pattern
// saying what to look for.

// walkFiles reads every file under the module root the keep predicate
// accepts, hands each to visit, and reports how many it read.
//
// The count is what lets a gate tell a clean tree from a broken walk. A
// gate whose healthy state is zero matches would otherwise pass just as
// silently when the walk stopped finding anything at all.
func walkFiles(t *testing.T, keep func(path string) bool, visit func(path string, body []byte) error) int {
	t.Helper()

	scanned := 0

	err := filepath.WalkDir("..", func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !keep(path) {
			return err
		}

		body, err := os.ReadFile(path) //nolint:gosec // G304: walking the module's own tree.
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		scanned++

		return visit(path, body)
	})
	if err != nil {
		t.Fatalf("walking the module: %v", err)
	}

	return scanned
}

// hit is one pattern match, with enough of its surroundings to name in a
// failure message.
type hit struct {
	path  string
	line  int
	text  string
	match []string
}

// locate finds every match of pattern in the files keep accepts, and
// reports how many files it read. Matching is line by line, so a hit
// carries the line it sits on.
func locate(t *testing.T, keep func(path string) bool, pattern *regexp.Regexp) ([]hit, int) {
	t.Helper()

	var found []hit

	scanned := walkFiles(t, keep, func(path string, body []byte) error {
		for i, text := range strings.Split(string(body), "\n") {
			for _, match := range pattern.FindAllStringSubmatch(text, -1) {
				found = append(found, hit{path: path, line: i + 1, text: text, match: match})
			}
		}

		return nil
	})

	return found, scanned
}

// scan is locate projected onto each match's first submatch, for the gates
// that want the values a pattern captured rather than where it found them.
func scan(t *testing.T, keep func(path string) bool, pattern *regexp.Regexp) []string {
	t.Helper()

	found, _ := locate(t, keep, pattern)

	values := make([]string, 0, len(found))
	for _, h := range found {
		values = append(values, h.match[1])
	}

	return values
}

// isGoSource keeps the engine's own source, excluding tests.
func isGoSource(path string) bool {
	return strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go")
}

// isGoTest keeps the module's test files.
func isGoTest(path string) bool {
	return strings.HasSuffix(path, "_test.go")
}

// inPackage keeps one package's own source, excluding tests.
func inPackage(dir string) func(string) bool {
	return func(path string) bool {
		return strings.Contains(path, "/"+dir+"/") && isGoSource(path)
	}
}
