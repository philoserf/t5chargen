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
// One check reaches past the repository entirely. ERRATA.md quotes the
// printed rules on every page it cites, and citations_test.go holds the
// quotations to the pages — against a PDF this repository does not
// contain and cannot, the rules being a purchased artifact. It runs from
// `task citations` and skips everywhere else, which is the most a gate
// on an artifact nobody else has can honestly do.
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
// COVERAGE.md's Status column is gated on its vocabulary and on its
// evidence, not on its truth. Whether a rule is really implemented is
// not mechanically checkable; what is checkable is that the cell opens
// with one of six words and that the two words claiming implementation
// name a test or a gate, which TestEveryStatusIsInTheVocabulary
// enforces. The other four words are the reasons a rule has no test, and
// exempting them is what keeps the rule from demanding meaningless tests
// on rows like Many Dice. TestNoRowDefersToAClosedMilestone survives
// from when the column still carried deferrals.
//
// ERRATA.md's long historical narratives stay where they are. The
// proposal was to move them to a separate decision-history document,
// leaving each entry a short ruling. It was declined: reasoning at the
// site is deliberate here — CLAUDE.md asks for the governing rule quoted
// at the implementation — and splitting three thousand lines produces
// two documents that can disagree about the same decision, which is the
// failure this package exists to prevent. The classification on each
// heading gives a reader the ruling without the split.
//
// ERRATA.md entries do not carry test references. The proposal was to
// name a test in each implemented entry, some fifty of them. Instead
// TestEveryImplementedEntryRestsOnATest requires the COVERAGE.md row
// citing an entry to name one: the same guarantee, kept in the document
// that already holds test names, rather than copied into a second place
// that can rot against the first.
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

// statuses is the fixed vocabulary a COVERAGE.md row's Status cell may
// open with, and what each promises.
//
// The document used to name three and use nine, which is the drift a
// prerelease review caught: a vocabulary nobody enforces stops meaning
// anything, and "covered", "transcribed" and "implemented; unexercised"
// were three ways of saying different things under one heading.
//
// The list is closed on purpose. A row that fits none of these is a row
// whose author has not decided what it is claiming.
var statuses = map[string]string{
	"covered":            "implemented, and names a test or a gate",
	"interpretation":     "covered under the cited ERRATA.md reading",
	"accepted exception": "in v1 scope and deliberately incomplete",
	"out of scope":       "excluded by a docs/PRD.md non-goal",
	"unreachable":        "cannot occur for a v1 human",
	"play-time rule":     "a real rule, outside character generation",
}

// evidenced are the statuses that must name a test or a gate. The other
// three describe rules the engine does not run, and demanding a test of
// those would be demanding a test of nothing.
var evidenced = []string{"covered", "interpretation"}

// TestEveryStatusIsInTheVocabulary verifies COVERAGE.md's Status column
// against the list its own header prints, and holds the two statuses
// that claim implementation to naming their evidence.
//
// Both halves come from the same failure. A status column with an open
// vocabulary cannot be read mechanically, so nothing checked it, so rows
// accumulated statuses like "transcribed" and "implemented; unexercised"
// that say something real and say it in a way no gate can see. Fixing
// the vocabulary is what makes the second half — every claim of
// implementation names its evidence — checkable at all.
func TestEveryStatusIsInTheVocabulary(t *testing.T) {
	rows := coverageRows(t)

	for _, row := range rows {
		opening := statusOpening(row)
		if opening == "" {
			t.Errorf("%s: status opens %q, which is not in the vocabulary",
				firstCell(row), trimStatus(lastCell(row)))

			continue
		}

		if slices.Contains(evidenced, opening) && testCell(row) == "" {
			t.Errorf("%s: status is %q and names no test", firstCell(row), opening)
		}
	}
}

// coverageRows returns COVERAGE.md's rule rows, without its headers,
// separators, or the tables that are not rule maps.
func coverageRows(t *testing.T) []string {
	t.Helper()

	var rows []string

	for _, row := range regexp.MustCompile(`(?m)^\|.*\|`).FindAllString(read(t, docsDir+"COVERAGE.md"), -1) {
		cells := strings.Split(strings.Trim(row, "|"), "|")
		if len(cells) < 5 || strings.HasPrefix(strings.TrimSpace(cells[0]), "---") {
			continue
		}

		if first := strings.TrimSpace(cells[0]); first == "" || first == "Rule" || first == "Status" {
			continue
		}

		rows = append(rows, row)
	}

	if len(rows) < 50 {
		t.Fatalf("found only %d COVERAGE.md rule rows; the scan is not working", len(rows))
	}

	return rows
}

// testCell returns a row's Test column, or "" where it names none.
// statusOpening returns the vocabulary word a row's Status opens with,
// longest match first so "out of scope" is not read as an unknown word
// beginning with "out". It returns "" for a status outside the six.
func statusOpening(row string) string {
	status := strings.ToLower(lastCell(row))
	opening := ""

	for name := range statuses {
		if strings.HasPrefix(status, name) && len(name) > len(opening) {
			opening = name
		}
	}

	return opening
}

func testCell(row string) string {
	cells := strings.Split(strings.Trim(row, "|"), "|")

	got := strings.TrimSpace(cells[3])
	if got == "—" || got == "-" {
		return ""
	}

	return got
}

// trimStatus shortens a status for a failure message.
func trimStatus(status string) string {
	if len(status) <= 60 {
		return status
	}

	return status[:60] + "…"
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

// coverageColumns is the shape every COVERAGE.md table has. The document
// is one table repeated, and the gates below read the Test and Status
// cells by position.
var coverageColumns = []string{"Rule", "Cite", "Implementation", "Test", "Status"}

// TestEveryCoverageRowFitsItsHeader verifies that every table in
// COVERAGE.md has the five columns named above and that no row carries
// more or fewer cells than its header.
//
// A row with extra cells is not a cosmetic fault. Markdown drops the
// surplus, so the Implementation, Test and Status a reader sees are not
// the ones the source holds — three rows carried two spare "—" cells and
// rendered as though they named nothing, while a positional gate read a
// cell over and reported them as untested. The document lied in both
// directions at once, and neither was visible in the rendered page.
func TestEveryCoverageRowFitsItsHeader(t *testing.T) {
	var (
		columns int
		header  int
		tables  int
	)

	for i, line := range strings.Split(read(t, docsDir+"COVERAGE.md"), "\n") {
		if !strings.HasPrefix(line, "|") {
			continue
		}

		cells := strings.Split(strings.Trim(strings.TrimSpace(line), "|"), "|")
		for j := range cells {
			cells[j] = strings.TrimSpace(cells[j])
		}

		switch {
		case cells[0] == "Rule":
			if !slices.Equal(cells, coverageColumns) {
				t.Errorf("line %d: header is %v, want %v", i+1, cells, coverageColumns)
			}

			columns, header, tables = len(cells), i+1, tables+1
		case strings.HasPrefix(cells[0], "---"):
		case columns != 0 && len(cells) != columns:
			t.Errorf("line %d: %d cells, but the header on line %d has %d: %q",
				i+1, len(cells), header, columns, cells[0])
		}
	}

	if tables < 20 {
		t.Fatalf("found only %d tables; the scan is not working", tables)
	}
}

// classifications are what an ERRATA.md entry may be. Every entry
// carries exactly one, on its heading line.
//
// The four are not decoration. An Interpretation reads ambiguous text
// and stays inside the printed rule; a Deviation knowingly departs from
// it and therefore owes the record an `errata` stamp (docs/PRD.md, "any
// applied ERRATA.md deviations"); an Accepted exception is a rule in v1
// scope this tool declines to implement; an Open question is undecided.
// Those obligations differ, and an entry that does not say which it is
// cannot be held to any of them — I-47 sat unclassified for four
// milestones while the code silently picked a reading.
var classifications = []string{
	"Interpretation",
	"Deviation",
	"Accepted exception",
	"Open question",
}

// errataHeading matches a classified entry heading:
//
//	### I-1: Interpretation — Citizen Job roll landing on ... (p. 78)
var errataHeading = regexp.MustCompile(`(?m)^### (I-\d+): ([A-Z][a-z]+(?: [a-z]+)*) — (.+)$`)

// TestEveryInterpretationIsClassified verifies that every ERRATA.md
// entry names one of the four classifications on its heading line.
func TestEveryInterpretationIsClassified(t *testing.T) {
	errata := read(t, docsDir+"ERRATA.md")

	all := regexp.MustCompile(`(?m)^### (I-\d+):.*$`).FindAllString(errata, -1)
	if len(all) < 100 {
		t.Fatalf("found only %d ERRATA.md entries; the scan is not working", len(all))
	}

	for _, heading := range all {
		match := errataHeading.FindStringSubmatch(heading)
		if match == nil {
			t.Errorf("unclassified: %s", heading)

			continue
		}

		if !slices.Contains(classifications, match[2]) {
			t.Errorf("%s: classification %q is not one of %v", match[1], match[2], classifications)
		}
	}
}

// TestNoOpenQuestionIsCalledCovered verifies that no ERRATA.md entry
// classified as an Open question is cited from a COVERAGE.md row that
// calls itself covered or an interpretation.
//
// This is the I-47 failure in gate form. That entry recorded a
// disagreement and left the reading open while the engine had already
// chosen one, and the coverage map called the rule covered on its
// strength. A question the map answers is not open, and a rule resting
// on an open question is not covered; the two documents must not be able
// to hold both positions at once.
func TestNoOpenQuestionIsCalledCovered(t *testing.T) {
	rows := coverageRows(t)

	for _, match := range errataHeading.FindAllStringSubmatch(read(t, docsDir+"ERRATA.md"), -1) {
		if match[2] != "Open question" {
			continue
		}

		cites := regexp.MustCompile(`\b` + regexp.QuoteMeta(match[1]) + `\b`)

		for _, row := range rows {
			opening := statusOpening(row)
			if !cites.MatchString(row) || !slices.Contains(evidenced, opening) {
				continue
			}

			t.Errorf("%s is an Open question, but COVERAGE.md calls %q %s",
				match[1], firstCell(row), opening)
		}
	}
}

// citingRows returns the COVERAGE.md rows that cite an ERRATA.md entry,
// anchored so I-4 is not matched by a mention of I-44.
func citingRows(rows []string, entry string) []string {
	cites := regexp.MustCompile(`\b` + regexp.QuoteMeta(entry) + `\b`)

	var citing []string

	for _, row := range rows {
		if cites.MatchString(row) {
			citing = append(citing, row)
		}
	}

	return citing
}

// TestAcceptedExceptionsAgree verifies that the two documents name the
// same accepted exceptions. An ERRATA.md entry classified as one must be
// cited from a COVERAGE.md row that calls the rule an accepted
// exception, and every such row must cite such an entry.
//
// Both directions, because each half fails differently: a coverage row
// claiming an exception nothing explains is a rule quietly dropped, and
// an entry declining a rule the coverage map calls covered is the I-47
// shape again.
func TestAcceptedExceptionsAgree(t *testing.T) {
	rows := coverageRows(t)
	errata := read(t, docsDir+"ERRATA.md")

	declined := map[string]bool{}

	for _, match := range errataHeading.FindAllStringSubmatch(errata, -1) {
		if match[2] != "Accepted exception" {
			continue
		}

		declined[match[1]] = true

		if !slices.ContainsFunc(citingRows(rows, match[1]), func(row string) bool {
			return statusOpening(row) == "accepted exception"
		}) {
			t.Errorf("%s is an Accepted exception, which no COVERAGE.md row calls one", match[1])
		}
	}

	for _, row := range rows {
		if statusOpening(row) != "accepted exception" {
			continue
		}

		named := false

		for entry := range declined {
			if regexp.MustCompile(`\b` + regexp.QuoteMeta(entry) + `\b`).MatchString(row) {
				named = true
			}
		}

		if !named {
			t.Errorf("%q is an accepted exception citing no ERRATA.md entry classified as one", firstCell(row))
		}
	}
}

// TestEveryImplementedEntryRestsOnATest verifies that an ERRATA.md entry
// naming an implementation site is cited from at least one COVERAGE.md
// row that names a test.
//
// This is the adaptation recorded in the audit package doc. The review
// asked for a test reference on each implemented interpretation, which
// would copy some fifty test names into a second document that can rot
// against the first. Requiring the citing coverage row to carry one puts
// the same guarantee in the place that already holds it.
func TestEveryImplementedEntryRestsOnATest(t *testing.T) {
	rows := coverageRows(t)
	implemented := regexp.MustCompile(`(?m)^Implemented at `)

	// Split on the headings rather than matching whole entries: a
	// non-greedy match that stops at the next "### " consumes it, and so
	// returns every other entry.
	parts := regexp.MustCompile(`(?m)^### (I-\d+):`).
		Split(read(t, docsDir+"ERRATA.md"), -1)
	names := regexp.MustCompile(`(?m)^### (I-\d+):`).
		FindAllStringSubmatch(read(t, docsDir+"ERRATA.md"), -1)

	if len(names) < 100 || len(parts) != len(names)+1 {
		t.Fatalf("found %d ERRATA.md entries in %d parts; the scan is not working", len(names), len(parts))
	}

	for i, name := range names {
		body := parts[i+1]
		if !implemented.MatchString(body) {
			continue
		}

		if !slices.ContainsFunc(citingRows(rows, name[1]), func(row string) bool {
			return testCell(row) != ""
		}) {
			t.Errorf("%s names an implementation site, but no COVERAGE.md row citing it names a test", name[1])
		}
	}
}

// TestEveryDeviationIsDeclared holds the engine's deviation set equal to
// ERRATA.md's Deviation-classified headings, both directions.
//
// This is the gate the stamping gap needed and did not have. I-82 and
// I-112 both said plainly that they departed from the printed rule while
// docs/PRD.md required every record to carry "any applied ERRATA.md
// deviations", and `Character.Errata` went unwritten for as long as both
// entries existed. Nothing connected the classification to the engine,
// so the next entry classified as a Deviation would have failed to stamp
// exactly as silently.
func TestEveryDeviationIsDeclared(t *testing.T) {
	var classified []string

	for _, match := range errataHeading.FindAllStringSubmatch(read(t, docsDir+"ERRATA.md"), -1) {
		if match[2] == "Deviation" {
			classified = append(classified, match[1])
		}
	}

	if len(classified) == 0 {
		t.Fatal("ERRATA.md classifies no entry as a Deviation; the scan is not working")
	}

	for _, entry := range classified {
		if !slices.Contains(chargen.Deviations, entry) {
			t.Errorf("ERRATA.md classifies %s as a Deviation, which the engine never stamps", entry)
		}
	}

	for _, entry := range chargen.Deviations {
		if !slices.Contains(classified, entry) {
			t.Errorf("the engine stamps %s, which ERRATA.md does not classify as a Deviation", entry)
		}
	}
}
