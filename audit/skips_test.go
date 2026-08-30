package audit_test

import (
	"regexp"
	"strings"
	"testing"
)

// rulesSweep is the one test allowed to bow out at run time. It needs the
// printed rules, which are not redistributed with this repository, and
// pdftotext to read them — two facts about the machine the suite runs on
// rather than about the engine.
const rulesSweep = "citations_test.go"

// skipCall matches a call to testing.T's skip methods. Written so it
// cannot match its own source: the pattern's escaped dot means this file
// nowhere holds the literal text it looks for.
var skipCall = regexp.MustCompile(`\bt\.Skipf?\(`)

// TestOnlyTheRulesSweepBowsOut holds that no test outside the rules sweep
// may decline to run.
//
// The engine has no wall clock and no unseeded randomness (CLAUDE.md,
// Determinism), so whether a pinned fixture is disabled, or a bounded
// sweep reaches a branch, is a fact about this engine and not a condition
// to shrug at. A test that bows out when it cannot find its case reports
// green while asserting nothing, and the document that cites it goes on
// claiming the rule is covered.
//
// That is not hypothetical. TestDoubleBenefitsDoublesTheCount bowed out on
// every run while COVERAGE.md called p. 69 covered, and nothing recorded
// when it had stopped testing — docs_test.go checks that a cited test
// exists, never that it runs. This gate is the missing half.
//
// The repo states both replacements. For a pinned fixture, guard the
// premise and fail: "a fixture that stopped dying would make this test
// pass while testing nothing" (chargen/character_test.go). For a sweep,
// count what the sweep found and fail asking for a wider bound
// (chargen/scout_test.go).
func TestOnlyTheRulesSweepBowsOut(t *testing.T) {
	found, scanned := locate(t,
		func(path string) bool {
			return isGoTest(path) && !strings.HasSuffix(path, rulesSweep)
		},
		skipCall)

	// Files, not matches. The healthy state of this gate is zero matches,
	// so counting those would make a broken walk indistinguishable from a
	// clean tree.
	if scanned < 50 {
		t.Fatalf("walked %d test files, expected the module's own; the walk is broken", scanned)
	}

	for _, h := range found {
		t.Errorf("%s:%d declines to run:\n\t%s\n"+
			"Only %s may, and only because the printed rules are not redistributed. "+
			"Guard a pinned fixture's premise and fail, or count what a sweep found and "+
			"ask for a wider bound.",
			h.path, h.line, strings.TrimSpace(h.text), rulesSweep)
	}
}
