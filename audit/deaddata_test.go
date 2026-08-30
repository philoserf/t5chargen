package audit_test

// The gate on transcribed-but-unread chart data.
//
// Milestone 4 shipped three fields that were transcribed from the page,
// validated at load, and then read by nothing: chart F's "WB x1" with no
// award producing a WB, a `fame` benefit kind that resolved and never
// awarded, and chart M1's `minimum_terms` sitting beside a Go constant
// that did the work. Each was found by review rather than by a test,
// because a field nothing reads breaks nothing.
//
// The gate is not that every field must be read. Some are transcribed
// precisely so the data records what the page says, and some belong to a
// rule that is deliberately deferred. The gate is that a field is one of
// those two things *on purpose*, named here with a reason.
//
// Known limitation, demonstrated rather than hypothetical: the search is
// by field *name*, so a field passes whenever any type anywhere shares its
// name with a field that is read. career.Operation.Implemented sat unread
// through two milestones because education.Program.Implemented is read,
// and it took a code review rather than this gate to find it. Closing that
// needs type resolution rather than a name search, which is a bigger
// change than the gate is worth today — but a field with a common name is
// weakly covered here, and reviewing one should not stop at "the gate is
// green".
//
// Two more live examples, so the reach of that is concrete rather than
// hypothetical. calendar.Row.Dice is read by calendar's own validator and
// nothing else, and passes because career.MusterOutCell.Dice is read in
// chargen/musterout.go. lifestage.Stage.First is read by no production
// code at all — lifestage_test.go holds it to FirstYearOf — and passes
// because dice.FluxRoll.First is read in chargen/event.go. Neither can be
// listed in unreadOnPurpose: the name *is* read, so the reverse check
// below would fire.
//
// Second known limitation, and the reason the first cannot simply be
// fixed by resolving reachability: the gate blanks validate* bodies so a
// validator's read does not count, but it cannot see *which value* an
// argument takes. education.SkillRow.flag returns Law and Medical. flag
// has two callers: teaches, which only validateNamedAwards calls, and
// Majors, which chargen reaches with program.MajorsFrom — a value that is
// "school", "college" or "academy" and never "law" or "medical", because
// Medical School and Law School select no Major (I-104). So the only read
// that executes is the loader's own cross-check, and the gate counts the
// fields as read anyway.
//
// Neither available fix works. Listing them in unreadOnPurpose trips the
// reverse check, because .Law and .Medical are plainly present in flag,
// which is not a validator. A call-graph closure keeps flag, because
// Majors genuinely is production-reachable. What makes these two unread
// is a value, not a name and not a path, and no static test over this
// source can decide it. Recorded here rather than worked around: the
// gate's green is one claim short for a field reached through a shared
// helper with a selecting argument.

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// why a data field is not read in production. Every entry is either
// transcription — the page said it, so the data says it, and no rule turns
// on it — or a rule this repo has deliberately not implemented, in which
// case the reason names where that deferral is recorded.
var unreadOnPurpose = map[string]string{
	// Transcription: recorded because the page prints it.
	"Note":        "chart M1's gloss on a benefit, chart B's \"Born In Deep Space\"; prose, not a rule",
	"Instruction": "p. 263's \"Roll four consecutive dice\"; the procedure is in Go",
	// "Printed" was here for MusterOutCell, whose printed wording is kept
	// but read by nothing (I-70). It is gone because MusterOutDM.Printed
	// now names the DM column a player is looking for on his chart, and
	// this gate keys on the field name rather than the field: one Printed
	// being read is all it can see. The cell's rationale did not move —
	// it is on the struct, where it was always the better home.
	"TonsPerShareCite":    "the p. 90 sentence the tonnage invariant checks against",
	"Last":                "the last year chart A prints for a stage; Of derives it from First — I-48",
	"TraditionalLifespan": "\"The traditional lifespan for humans is 74 years\"; flavor, and the page says so",
	"QSP":                 "chart S's Quick Ship Profile; ship design is out of scope",
	"Hex":                 "chart B's world location; it identifies the world on the page, and no rule reads it",
	"Sector":              "chart B's world location; see Hex",
	"Jump":                "chart S's J column; no rule in chargen reads a ship's performance",
	"Maneuver":            "chart S's G column; as Jump",
	"CostMCr":             "chart S prices the ship in megacredits; a share has no credit value — interpretation I-84",

	// chart C's knowledge_only flag marks the container skills of p. 134.
	// It is orthogonal to the institution columns rather than a substitute
	// for them — Gunner is knowledge-only *and* offered by the Navy — so
	// it explains the matrix rather than driving it. Whether education can
	// award a bare container skill is a separate open question, noted in
	// COVERAGE.md.
	"KnowledgeOnly": "explains why a chart C row is shaped as it is; the matrix columns are what award a skill",

	// Transcription that exists so the data can check itself. Each of
	// these is read by a validator and nothing else, which is exactly the
	// shape this gate reports — and here it is the intent.
	"TonsPerShare": "chart S's \"one Share acquires 50 tons\"; the rate every priced row is checked against",
	"LoanShares":   "chart S's Loan? column; loaning a ship is play, not chargen — interpretation I-84",
	"Carried":      "marks the Lab Launch and Escort Gig, which come with a ship rather than being bought",
	"Eligibility":  "chart M1's eligibility prose; eligibleFor keys off the row id, not the text",
	"MusterOutM2":  "chart M2, transcribed so its disagreement with the career pages is visible — I-71",

	// Deferred rules, each recorded in COVERAGE.md.
	"QREBS":   "chart 01's Masterpiece qualities; QREBS allocation is deferred",
	"Maximum": "a QREBS quality's ceiling; as QREBS",
}

// TestNoChartDataIsTranscribedAndForgotten checks every field of the
// embedded chart data is either read by production code or listed above.
//
// It also checks the reverse, which is the half that rots: a field that
// becomes read must leave the list, or the list stops describing the code.
func TestNoChartDataIsTranscribedAndForgotten(t *testing.T) {
	fields := chartDataFields(t)
	if len(fields) < 100 {
		t.Fatalf("found only %d chart data fields; the scan is not working", len(fields))
	}

	source := productionSource(t)

	var unread []string

	for name, where := range fields {
		read := regexp.MustCompile(`\.` + regexp.QuoteMeta(name) + `\b`).MatchString(source)
		_, excused := unreadOnPurpose[name]

		switch {
		case read && excused:
			t.Errorf("%s (%s) is read in production but still listed as unread on purpose", name, where)
		case !read && !excused:
			unread = append(unread, name+" ("+where+")")
		}
	}

	for _, field := range unread {
		t.Errorf("%s is transcribed and never read; wire it, or say why not in unreadOnPurpose", field)
	}
}

// chartDataFields returns the exported, JSON-tagged fields declared in the
// packages that embed a chart, keyed by name and valued by where the first
// declaration is. Only those packages: a field of the character record is
// written and serialized, not read, and is not what this gate is about.
func chartDataFields(t *testing.T) map[string]string {
	t.Helper()

	// The tag class carries digits and is anchored to the closing quote:
	// career.muster_out_m2, world.d1 and world.d2 were otherwise reported
	// as muster_out_m and d, sending a reader after a tag that does not
	// exist and collapsing d1 and d2 into one label. Coverage was never
	// affected — the map keys on the Go field name — only the message.
	declaration := regexp.MustCompile("(?m)^\\s+([A-Z][A-Za-z0-9]*)\\s+[][*A-Za-z0-9_.]+\\s+`json:\"([a-z0-9_]+)\"")
	fields := map[string]string{}

	for _, pkg := range chartPackages(t) {
		entries, err := os.ReadDir(pkg)
		if err != nil {
			t.Fatalf("reading %s: %v", pkg, err)
		}

		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}

			body, err := os.ReadFile(filepath.Join(pkg, name)) //nolint:gosec // G304: the module's own tree.
			if err != nil {
				t.Fatalf("reading %s: %v", name, err)
			}

			for _, match := range declaration.FindAllStringSubmatch(string(body), -1) {
				if _, seen := fields[match[1]]; !seen {
					fields[match[1]] = filepath.Base(pkg) + "." + match[2]
				}
			}
		}
	}

	return fields
}

// chartPackages lists the module's directories that embed a chart, which
// is exactly the ones holding a data directory.
func chartPackages(t *testing.T) []string {
	t.Helper()

	entries, err := os.ReadDir("..")
	if err != nil {
		t.Fatalf("reading the module root: %v", err)
	}

	var packages []string

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dir := filepath.Join("..", entry.Name())
		if info, err := os.Stat(filepath.Join(dir, "data")); err == nil && info.IsDir() {
			packages = append(packages, dir)
		}
	}

	if len(packages) == 0 {
		t.Fatal("no package embeds a chart; the scan is not working")
	}

	return packages
}

// productionSource concatenates every non-test Go file in the module with
// the bodies of its validate* functions removed, so a field counts as read
// only when something other than its own validator reads it.
//
// Dropping the validators is the whole point. "Transcribed from the page,
// validated at load, and then read by nothing" is the bug class this file
// exists for, so a field mentioned only inside validateX is the bug, not a
// reader of it — and counting it as a reader let chart M2's entire
// transcription pass unremarked.
//
// Dropping whole data packages instead would be wrong: chart data is
// legitimately read by accessors that live beside it and that the rule
// packages call — ship.Ships by Largest, lifestage.Stages by Of,
// benefit.Kinds by For, education.College by flag. Excluding those packages
// reports thirty-six fields, nearly all of them false.
//
// The remaining known imprecision is the other way: the regex matches on
// the Go field name, so a chart field sharing a common name with a
// character-record field (Name, Value) counts as read wherever that other
// field is read. Matching on the JSON tag would close it and is a larger
// change than the hole it leaves.
func productionSource(t *testing.T) string {
	t.Helper()

	var source strings.Builder

	walkFiles(t, isGoSource, func(path string, body []byte) error {
		kept, err := readOutsideValidators(path, body)
		if err != nil {
			return err
		}

		source.WriteString(kept)

		return nil
	})

	return source.String()
}

// readOutsideValidators returns a file's source with every validate*
// function body blanked out. It parses rather than brace-matching, because
// a brace inside a string literal or a comment would defeat the latter.
func readOutsideValidators(path string, body []byte) (string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), path, body, parser.SkipObjectResolution)
	if err != nil {
		return "", fmt.Errorf("parsing %s: %w", path, err)
	}

	kept := make([]byte, len(body))
	copy(kept, body)

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil || !strings.HasPrefix(fn.Name.Name, "validate") {
			continue
		}

		// Blank the body in place, preserving offsets so a later decl's
		// positions stay valid.
		for i := fn.Body.Pos() - 1; i < fn.Body.End()-1 && int(i) < len(kept); i++ {
			if kept[i] != '\n' {
				kept[i] = ' '
			}
		}
	}

	return string(kept), nil
}

// TestFameAndMedalAgreeOnCodes holds the two transcriptions of chart F's
// medal codes to each other.
//
// The codes appear in fame/data/fame.json, which prices them, and in
// medal/data/medals.json, which awards them. Neither package can import
// the other without inverting the dependency, so nothing cross-checked
// them and a typo in either would load clean: chargen/fame.go's consumer
// reads an unpriced code as a medal worth nothing and skips it, so the
// character's Fame is simply lower and the record says nothing.
//
// WB is the one deliberate difference. A Wound Badge is not a Medal — the
// Risk failure awards it, not the Reward success (p. 91) — so it is
// counted on the record and priced by name, and appears in fame.json
// alone.
func TestFameAndMedalAgreeOnCodes(t *testing.T) {
	var priced struct {
		Medals map[string]int `json:"medals"`
	}

	var awarded struct {
		Rows []struct {
			Code string `json:"code"`
		} `json:"rows"`
	}

	readJSON(t, filepath.Join("..", "fame", "data", "fame.json"), &priced)
	readJSON(t, filepath.Join("..", "medal", "data", "medals.json"), &awarded)

	const woundBadge = "WB"

	want := map[string]bool{woundBadge: true}
	for _, row := range awarded.Rows {
		want[row.Code] = true
	}

	if len(want) < 3 {
		t.Fatalf("found only %d medal codes; the scan is not working", len(want))
	}

	for code := range priced.Medals {
		if !want[code] {
			t.Errorf("fame.json prices %q, which chart M1 does not award", code)
		}
	}

	for code := range want {
		if _, ok := priced.Medals[code]; !ok {
			t.Errorf("chart M1 awards %q, which fame.json does not price", code)
		}
	}
}

// readJSON decodes one of the module's data files into v.
func readJSON(t *testing.T, path string, v any) {
	t.Helper()

	data, err := os.ReadFile(path) //nolint:gosec // G304: the module's own data files.
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}

	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}
}
