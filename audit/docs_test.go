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
// has a POLICY.md rule, that POLICY.md states the version the engine
// stamps, and that every career the engine can run has a section.
//
// One check reaches past the documents into the Go source. A doc comment
// is a claim about the code as much as COVERAGE.md is, and it rots the
// same way: twenty-six of them still deferred work to milestones 3 and 4
// long after those delivered it. TestNoCommentNamesAClosedMilestone
// gates that, and it belongs here rather than beside the code it reads,
// because what makes a milestone closed is a fact about docs/PRD.md.
//
// # What this package deliberately does not gate
//
// Named here so their absence reads as a decision rather than an
// oversight, because each was proposed and each proposal was wrong.
//
// README.md and docs/PRD.md are reviewed, not gated. Both drifted and
// both were corrected, but the obvious check — hold the README's --career
// list to career.Available() — demands the README advertise flag values
// the CLI refuses: Functionary is never a first career (p. 87) and
// Craftsman's automatic entry needs what a character leaving education
// never has (p. 75). The available set is character-dependent besides;
// Noble drops out on a low Soc.
//
// COVERAGE.md's Status column is half gated. "covered" is not
// mechanically checkable, and the deferred half cannot be narrowed to
// "names no implementation" — legitimately deferred rows name real code.
// What is checkable is that a deferral does not name a milestone that has
// already shipped, which TestNoRowDefersToAClosedMilestone enforces after
// that claim rotted twice in two days. The rest of the column is still
// reviewed.
//
// COVERAGE.md's prose is reviewed, not gated, and this is where the rot
// keeps going next. Three stale claims in a week lived in a note or a
// paragraph rather than a Status cell, each one invisible to the gate
// standing beside it. No mechanical rule was found for prose that would
// not either miss most of it or flag most of it, so the honest position
// is that the gates cover the table and a reader covers the rest.
//
// COVERAGE.md's Implementation column is reviewed, not gated, for the
// reason given at TestCoverageNamesRealTests.
//
// Load-time validation density is not gated. The proposal was that every
// chart package declare a validate function; medal is the counterexample,
// validating its table thoroughly inside the sync.OnceValues body with no
// such function. A gate keying on the name would demand a refactor for
// its own sake.
package audit_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/chargen"
)

// read returns a repository file relative to the module root.
// Where the documents these gates hold to account live. They sit together
// in one folder and the checks on them in another, so neither mixes prose
// with the code that reads it.
//
// Two spellings because there are two ways in: read joins the repository
// root itself, while the schema is opened directly.
const (
	docsDir  = "docs/"
	docsPath = ".." + string(filepath.Separator) + docsDir
)

func read(t *testing.T, name string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("..", name)) //nolint:gosec // G304: fixed repository files.
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}

	return string(data)
}

// scan returns the first submatch of every match of pattern across the
// module's source files that keep accepts.
func scan(t *testing.T, keep func(path string) bool, pattern *regexp.Regexp) []string {
	t.Helper()

	var found []string

	err := filepath.WalkDir("..", func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !keep(path) {
			return err
		}

		data, err := os.ReadFile(path) //nolint:gosec // G304: walking the module's own tree.
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		for _, match := range pattern.FindAllStringSubmatch(string(data), -1) {
			found = append(found, match[1])
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walking the module: %v", err)
	}

	return found
}

// TestCoverageNamesRealTests verifies every test COVERAGE.md cites as
// evidence exists. A renamed or deleted test would otherwise leave the
// document claiming a rule is covered by nothing.
//
// Only the Test column is checked, and the asymmetry is deliberate rather
// than an oversight. The Implementation column makes the same kind of
// claim and would rot the same way, but it is not one vocabulary: its 288
// backticked entries are Go functions and types, JSON field names
// (begin_automatic_if, knowledge_only), file paths (functionary.json),
// CLI flags (--current-year), package names, and choice ids. Nothing
// distinguishes them mechanically, so a gate over that column needs an
// exclusion list about as long as the set it would falsely flag — which
// is the shape of check this repo has rejected before.
//
// All 288 resolve as of this writing. Treat the Implementation column as
// reviewed rather than gated, and check it by hand when renaming.
func TestCoverageNamesRealTests(t *testing.T) {
	named := regexp.MustCompile("`(Test[A-Za-z0-9_]+)`").FindAllStringSubmatch(read(t, docsDir+"COVERAGE.md"), -1)
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

	return scan(t,
		func(path string) bool { return strings.HasSuffix(path, "_test.go") },
		regexp.MustCompile(`(?m)^func (Test[A-Za-z0-9_]+)\(`))
}

// TestEveryInterpretationIsCited verifies every ERRATA.md entry is
// referenced from COVERAGE.md, so a deliberate deviation cannot be
// recorded and then lost track of.
func TestEveryInterpretationIsCited(t *testing.T) {
	entries := regexp.MustCompile(`(?m)^### (I-\d+):`).FindAllStringSubmatch(read(t, docsDir+"ERRATA.md"), -1)
	if len(entries) == 0 {
		t.Fatal("ERRATA.md records no interpretations")
	}

	coverage := read(t, docsDir+"COVERAGE.md")

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
	ids := declaredChoiceIDs(t)
	if len(ids) == 0 {
		t.Fatal("no choice points found")
	}

	ruled := policyRuleIDs(t)

	for _, id := range ids {
		if !ruled[id] {
			t.Errorf("choice point %q has no POLICY.md decision-table rule", id)
		}
	}
}

// policyRuleIDs lists the choice points POLICY.md's decision table gives a
// rule, read from the table's first column rather than from the document.
//
// A substring search over the whole file is weaker than the rule this gate
// claims to enforce. POLICY.md's version history names choice points while
// explaining past bumps, so a choice point deleted from the table but
// still mentioned there would keep the gate green — and the gate would
// report a decision table as total when it had stopped being so.
//
// One row may cover two choice points that share a rule
// ("| `select_major` / `select_minor` |"), so the whole first cell is
// read. A cell holding anything but backticked ids and separators is not
// a rule row: that skips the header, the alignment row, and every prose
// cell that happens to quote an id.
func policyRuleIDs(t *testing.T) map[string]bool {
	t.Helper()

	firstCell := regexp.MustCompile(`(?m)^\|([^|]*)\|`)
	id := regexp.MustCompile("`([a-z0-9_]+)`")

	ids := make(map[string]bool)

	for _, row := range firstCell.FindAllStringSubmatch(read(t, docsDir+"POLICY.md"), -1) {
		named := id.FindAllStringSubmatch(row[1], -1)
		if len(named) == 0 {
			continue
		}

		if strings.Trim(id.ReplaceAllString(row[1], ""), " \t/") != "" {
			continue
		}

		for _, match := range named {
			ids[match[1]] = true
		}
	}

	return ids
}

// declaredChoiceIDs lists every ChoiceID constant in the module's
// non-test sources. The whole tree is scanned rather than decider.go
// alone, so a choice point declared beside the career that raises it
// cannot escape the gate.
func declaredChoiceIDs(t *testing.T) []string {
	t.Helper()

	return scan(t,
		func(path string) bool {
			return strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go")
		},
		regexp.MustCompile(`ChoiceID = "([a-z0-9_]+)"`))
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

	coverage := read(t, docsDir+"COVERAGE.md")

	for _, career := range registered {
		if !strings.Contains(coverage, "— "+career[1]+" (chart") {
			t.Errorf("career %s has no COVERAGE.md section", career[1])
		}
	}
}

// TestCareerSectionsAreInChartOrder keeps the document readable against
// the book: its career sections run in Book 1 chart order.
func TestCareerSectionsAreInChartOrder(t *testing.T) {
	found := regexp.MustCompile(`(?m)^## Career (\d+) —`).FindAllStringSubmatch(read(t, docsDir+"COVERAGE.md"), -1)
	if len(found) == 0 {
		t.Fatal("COVERAGE.md has no career sections")
	}

	numbers := make([]int, 0, len(found))

	for _, match := range found {
		// Compared as numbers: "9" does not sort after "10".
		chart, err := strconv.Atoi(match[1])
		if err != nil {
			t.Fatalf("career section %q: %v", match[1], err)
		}

		numbers = append(numbers, chart)
	}

	if !slices.IsSorted(numbers) {
		t.Errorf("career sections run %v, want Book 1 chart order", numbers)
	}
}

// closedMilestones are the docs/PRD.md milestones that have shipped.
//
// Update this when one closes. That is the point rather than the cost: a
// COVERAGE.md row deferring to a milestone already delivered is a claim
// that has rotted, and it rots silently, because the row still reads as a
// deliberate decision. Two cleanups of exactly this shipped in two days —
// five rows in #83 and eight here — before anything checked it.
//
// M6 closed with the Rogue's previous-career Scheme, the last row it
// held. Closing it here guards rows written from now on and catches
// nothing today: the one M6 claim left over was prose rather than a table
// row, so this gate could not see it and a person had to.
//
// M7 closed with the Career and World Knowledges, and closing it here did
// catch something: the Capital*** row still read "deferred (M7)" for a
// rule the milestone did not deliver and was never going to. That is the
// blind spot this gate has when a milestone is open — a row may defer to
// it truthfully right up to the moment it closes — and the only thing
// that shuts it is remembering to add the milestone here.
var closedMilestones = []string{"M1", "M2", "M3", "M4", "M5", "M6", "M7"}

// TestNoRowDefersToAClosedMilestone verifies COVERAGE.md's Status column
// names no milestone that has already shipped.
//
// This is the narrow half of a check the package otherwise declines. Which
// rules are "covered" is not mechanically decidable, and the deferred half
// cannot be narrowed to "names no implementation" — two legitimately
// deferred rows name real code. But a milestone number carries no
// information once that milestone is closed: the rule is either covered or
// waiting on something else, and saying "deferred (M4)" after M4 shipped
// is neither.
//
// Chart M1 and chart M2 are chart names, not milestones, and appear twenty
// times between them. They are excluded by the "chart" that precedes them
// rather than by listing the rows, so a new mention of either is safe.
func TestNoRowDefersToAClosedMilestone(t *testing.T) {
	rows := regexp.MustCompile(`(?m)^\|.*\|`).FindAllString(read(t, docsDir+"COVERAGE.md"), -1)
	if len(rows) < 50 {
		t.Fatalf("found only %d COVERAGE.md rows; the scan is not working", len(rows))
	}

	milestone := regexp.MustCompile(`(?i)(chart\s+)?\bM([1-9])\b`)

	for _, row := range rows {
		status := lastCell(row)

		// Only a deferral is checked. A milestone named in a covered
		// row's prose ("no-transfer becomes testable with M4 career
		// changes") is stale context, not a false status — and "M1" also
		// occurs as a Merchant rank code, in a row about a commission
		// from R2 to M1, which no amount of chart-prefix handling
		// separates from a milestone.
		if !strings.Contains(strings.ToLower(status), "deferred") {
			continue
		}

		for _, named := range milestone.FindAllStringSubmatch(status, -1) {
			if named[1] != "" { // "chart M1" — a chart, not a milestone
				continue
			}

			if slices.Contains(closedMilestones, "M"+named[2]) {
				t.Errorf("a row defers to M%s, which has shipped: %s", named[2], firstCell(row))
			}
		}
	}
}

// TestPolicyDocumentStatesItsVersion verifies POLICY.md's stated version
// is the one the engine stamps on every record.
//
// The document names the version in its own first paragraph, and nothing
// held the two together: PolicyVersion went to 0.22.0 with the Rogue's
// previous-career Scheme and POLICY.md's header stayed at 0.21.0, in the
// same pull request that added the rule. Every record written in between
// cited a decision table by a number the document disclaimed.
//
// This is the cheapest gate in the package and the one with the least
// judgement in it: two strings, and they either match or a record's
// provenance points at nothing.
func TestPolicyDocumentStatesItsVersion(t *testing.T) {
	stated := regexp.MustCompile(`(?m)^Version: \*\*([0-9]+\.[0-9]+\.[0-9]+)\*\*`).
		FindStringSubmatch(read(t, docsDir+"POLICY.md"))
	if stated == nil {
		t.Fatal("POLICY.md states no version; the header it is read from has moved")
	}

	if stated[1] != chargen.PolicyVersion {
		t.Errorf("POLICY.md states version %s; the engine stamps %s on every record",
			stated[1], chargen.PolicyVersion)
	}
}

// TestNoCommentNamesAClosedMilestone verifies no Go source comment cites
// a milestone that has shipped.
//
// It is the same claim TestNoRowDefersToAClosedMilestone makes about
// COVERAGE.md, pointed at the place the rot actually went. Twenty-six
// comments read as deliberate deferrals — "the muster-out table D
// (milestone 4)", "terms run long until aging lands", "v1 ships the
// Citizen career only" — long after milestone 4 delivered muster out and
// aging and milestone 3 delivered all thirteen careers. Four cleanups of
// this shape shipped in a week, each in a place the previous gate could
// not see.
//
// Unlike the COVERAGE check this needs no deferral word, and the stricter
// rule is the simpler one: once a milestone ships, its number tells a
// reader nothing the code does not. Where the work is done the citation
// is noise; where a gap remains, the gap is what to name. Historical
// attribution belongs to git history and docs/MILESTONE-*.md. Nothing
// then has to decide whether a given sentence is a deferral, which is the
// judgement the COVERAGE check has to make and the reason it can only
// gate half a column.
//
// Test files are excluded because a guard has to quote what it guards
// against: this comment names four closed milestones, and so does the one
// above it.
//
// The bare "M4" form COVERAGE.md uses is not matched here, and cannot be.
// In Go source a bare M1 or M2 means chart M1 or chart M2 some fifty
// times over — the muster-out chart is the whole subject of three
// packages — and "chart" does not always sit next to it: career.go says
// "chart M2 reprints it" and then "M2 is transcribed" two lines later.
// So this matches the spelled-out word and the parenthesised "(M4)",
// which never names a chart. A bare "M4" meaning a milestone would slip
// past, and that is the deliberate trade: the alternative flags fifty
// chart citations, and a gate that cries wolf gets read as noise, which
// is how the claims it guards rotted in the first place.
func TestNoCommentNamesAClosedMilestone(t *testing.T) {
	sources, scanned := locate(t,
		func(path string) bool {
			return strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go")
		},
		regexp.MustCompile(`(?i)\bmilestone\s+([1-9])\b|\(M([1-9])\)`))

	// Files, not matches. The healthy state of this gate is zero matches,
	// so counting those would make a broken walk indistinguishable from a
	// clean tree — and would have passed silently the moment the last
	// comment was fixed.
	if scanned < 20 {
		t.Fatalf("scanned only %d Go files; the walk is not working", scanned)
	}

	for _, found := range sources {
		named := found.match[1]
		if named == "" {
			named = found.match[2]
		}

		if slices.Contains(closedMilestones, "M"+named) {
			t.Errorf("%s:%d names milestone %s, which has shipped: %s",
				found.path, found.line, named, strings.TrimSpace(found.text))
		}
	}
}

// hit is one regexp match and where it was found, which is what a reader
// of the failure needs and what scan discards.
type hit struct {
	path  string
	line  int
	text  string
	match []string
}

// locate is scan with the location kept, and reports how many files it
// read. The two differ only in what they return: scan collects a
// vocabulary, locate reports places to go and fix.
func locate(t *testing.T, keep func(path string) bool, pattern *regexp.Regexp) ([]hit, int) {
	t.Helper()

	var (
		found   []hit
		scanned int
	)

	err := filepath.WalkDir("..", func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !keep(path) {
			return err
		}

		data, err := os.ReadFile(path) //nolint:gosec // G304: walking the module's own tree.
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		scanned++

		for i, text := range strings.Split(string(data), "\n") {
			for _, match := range pattern.FindAllStringSubmatch(text, -1) {
				found = append(found, hit{path: path, line: i + 1, text: text, match: match})
			}
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walking the module: %v", err)
	}

	return found, scanned
}

// firstCell and lastCell return a Markdown table row's first and last
// cells — the rule it names and the status it claims.
func firstCell(row string) string {
	cells := strings.Split(strings.Trim(row, "|"), "|")

	return strings.TrimSpace(cells[0])
}

func lastCell(row string) string {
	cells := strings.Split(strings.Trim(row, "|"), "|")

	return strings.TrimSpace(cells[len(cells)-1])
}
