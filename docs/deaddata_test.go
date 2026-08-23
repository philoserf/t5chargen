package docs_test

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

import (
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
	"Note":                "chart M1's gloss on each benefit; prose for a reader, not a rule",
	"Instruction":         "p. 263's \"Roll four consecutive dice\"; the procedure is in Go",
	"Printed":             "a cell's printed wording where it differs from the vocabulary — I-70 keeps both",
	"TonsPerShareCite":    "the p. 90 sentence the tonnage invariant checks against",
	"Last":                "the last year chart A prints for a stage; Of derives it from First — I-48",
	"TraditionalLifespan": "\"The traditional lifespan for humans is 74 years\"; flavor, and the page says so",
	"QSP":                 "chart S's Quick Ship Profile; ship design is out of scope",
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

	// Deferred rules, each recorded in COVERAGE.md.
	"Law":       "chart C's Law School column; the program is data-only (implemented: false)",
	"Medical":   "chart C's Medical School column; as Law",
	"QREBS":     "chart 01's Masterpiece qualities; QREBS allocation is deferred",
	"Minimum":   "a QREBS quality's floor; as QREBS",
	"Maximum":   "a QREBS quality's ceiling; as QREBS",
	"ValueName": "a chart C prerequisite's credential (BA, Honors BA, MA); only the deferred graduate programs need one",
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

	declaration := regexp.MustCompile("(?m)^\\s+([A-Z][A-Za-z0-9]*)\\s+[][*A-Za-z0-9_.]+\\s+`json:\"([a-z_]+)")
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

// productionSource concatenates every non-test Go file in the module, so a
// field read anywhere counts as read.
func productionSource(t *testing.T) string {
	t.Helper()

	var source strings.Builder

	err := filepath.WalkDir("..", func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		body, err := os.ReadFile(path) //nolint:gosec // G304: the module's own tree.
		if err != nil {
			return err //nolint:wrapcheck // walked back to the caller, which names the module.
		}

		source.Write(body)

		return nil
	})
	if err != nil {
		t.Fatalf("walking the module: %v", err)
	}

	return source.String()
}
