// Package docs_test enforces the milestone-3 exit criterion
// (docs/PRD.md): "a living COVERAGE.md in the tool repo mapping every E1
// step and career rule to its page cite, implementation, and golden test
// — no career is 'done' until its uncommon branches are listed there as
// covered or explicitly deferred."
//
// A document is only living if it cannot drift, so the invariants that
// make it trustworthy are checked here rather than by eye: that every
// test COVERAGE.md names exists, that every interpretation ERRATA.md
// records is cited from it, that every choice point the engine presents
// has a POLICY.md rule, and that every career the engine can run has a
// section.
package docs_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// read returns a repository file relative to the module root.
func read(t *testing.T, name string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("..", name)) //nolint:gosec // G304: fixed repository files.
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}

	return string(data)
}

// TestCoverageNamesRealTests verifies every test COVERAGE.md cites as
// evidence exists. A renamed or deleted test would otherwise leave the
// document claiming a rule is covered by nothing.
func TestCoverageNamesRealTests(t *testing.T) {
	named := regexp.MustCompile("`(Test[A-Za-z0-9_]+)`").FindAllStringSubmatch(read(t, "COVERAGE.md"), -1)
	if len(named) == 0 {
		t.Fatal("COVERAGE.md cites no tests")
	}

	declared := declaredTests(t)

	for _, match := range named {
		if !slices.Contains(declared, match[1]) {
			t.Errorf("COVERAGE.md cites %s, which no test declares", match[1])
		}
	}
}

// declaredTests lists every test function in the module.
func declaredTests(t *testing.T) []string {
	t.Helper()

	var names []string

	declaration := regexp.MustCompile(`(?m)^func (Test[A-Za-z0-9_]+)\(`)

	err := filepath.WalkDir("..", func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return err
		}

		data, err := os.ReadFile(path) //nolint:gosec // G304: walking the module's own tree.
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		for _, match := range declaration.FindAllStringSubmatch(string(data), -1) {
			names = append(names, match[1])
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walking the module: %v", err)
	}

	return names
}

// TestEveryInterpretationIsCited verifies every ERRATA.md entry is
// referenced from COVERAGE.md, so a deliberate deviation cannot be
// recorded and then lost track of.
func TestEveryInterpretationIsCited(t *testing.T) {
	entries := regexp.MustCompile(`(?m)^### (I-\d+):`).FindAllStringSubmatch(read(t, "ERRATA.md"), -1)
	if len(entries) == 0 {
		t.Fatal("ERRATA.md records no interpretations")
	}

	coverage := read(t, "COVERAGE.md")

	for _, entry := range entries {
		// Anchored, so I-4 is not satisfied by a mention of I-44.
		cited := regexp.MustCompile(`\b` + regexp.QuoteMeta(entry[1]) + `\b`)
		if !cited.MatchString(coverage) {
			t.Errorf("ERRATA.md records %s, which COVERAGE.md never cites", entry[1])
		}
	}
}

// TestEveryChoicePointHasAPolicy verifies every choice the engine can
// present has a POLICY.md rule. The auto policy is required to be total
// (docs/PRD.md, CLI sketch), so a choice point with no rule is a gap in
// the decision table even when the code happens to answer it.
func TestEveryChoicePointHasAPolicy(t *testing.T) {
	ids := regexp.MustCompile(`ChoiceID = "([a-z_]+)"`).FindAllStringSubmatch(read(t, "chargen/decider.go"), -1)
	if len(ids) == 0 {
		t.Fatal("no choice points found")
	}

	policy := read(t, "POLICY.md")

	for _, id := range ids {
		if !strings.Contains(policy, "`"+id[1]+"`") {
			t.Errorf("choice point %q has no POLICY.md rule", id[1])
		}
	}
}

// TestEveryCareerHasCoverage verifies every career the engine can run has
// a COVERAGE.md section, which is where its uncommon branches are listed
// as covered or deferred.
func TestEveryCareerHasCoverage(t *testing.T) {
	registered := regexp.MustCompile(`"([A-Z][a-z]+)":\s+new`).
		FindAllStringSubmatch(read(t, "chargen/careerrun.go"), -1)
	if len(registered) == 0 {
		t.Fatal("no careers are registered")
	}

	coverage := read(t, "COVERAGE.md")

	for _, career := range registered {
		if !strings.Contains(coverage, "— "+career[1]+" (chart") {
			t.Errorf("career %s has no COVERAGE.md section", career[1])
		}
	}
}

// TestCareerSectionsAreInChartOrder keeps the document readable against
// the book: its career sections run in Book 1 chart order.
func TestCareerSectionsAreInChartOrder(t *testing.T) {
	found := regexp.MustCompile(`(?m)^## Career (\d+) —`).FindAllStringSubmatch(read(t, "COVERAGE.md"), -1)

	numbers := make([]string, 0, len(found))
	for _, match := range found {
		numbers = append(numbers, match[1])
	}

	if !slices.IsSorted(numbers) {
		t.Errorf("career sections run %v, want Book 1 chart order", numbers)
	}
}
