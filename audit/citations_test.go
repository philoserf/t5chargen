package audit_test

// The gate on ERRATA.md's page citations: that every sentence it quotes
// from the printed rules is on a page the entry citing it names.
//
// It cannot run in CI and does not run in `task`. The rules are a
// purchased, watermarked PDF that this repository does not redistribute
// (README, Rules source), so the gate is skipped unless the caller says
// where their copy is — `task citations` does, and CLAUDE.md names the
// collection it expects.
//
// What it checks is provenance, not interpretation. That a quote appears
// where it is cited says nothing about whether the reading drawn from it
// is right, and no test can say that.

import (
	"context"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// rulesPDFEnv names the environment variable holding the path to Book 1.
const rulesPDFEnv = "T5_RULES_PDF"

// quoteBounds are the shortest and longest quotations worth checking. A
// short one is usually a term rather than a sentence and matches
// everywhere; a very long one is usually several sentences joined, and
// the fragments below handle those.
// pdfExtractTimeout bounds the one extraction run. Book 1 takes well
// under a second; a minute means pdftotext is stuck on something.
const pdfExtractTimeout = time.Minute

const (
	minQuote = 25
	maxQuote = 400
)

// TestEveryQuotationIsOnThePageItCites verifies ERRATA.md against the
// printed rules.
//
// A sweep on 2026-08-28 found thirteen citations wrong or missing, all
// the same shape: an entry quoting chart text, naming the chart by
// number, and never giving its page. A reader who does not already know
// where a chart sits cannot check those, which is what the ground rule
// about verifiable citations exists to prevent.
func TestEveryQuotationIsOnThePageItCites(t *testing.T) {
	pages := rulesPages(t)

	checked, unmatched := 0, 0

	for _, entry := range interpretations(t) {
		for _, quote := range quotations(entry.body) {
			runs := fragments(quote)
			if len(runs) == 0 {
				continue
			}

			checked++

			missing := firstUnmatched(runs, entry.pages, pages)
			if missing == "" {
				continue
			}

			// A run the book prints somewhere else is a citation
			// pointing at the wrong page, which is the fault this gate
			// exists to find. A run the extraction cannot locate at all
			// is usually the extraction: the raw layout joins words
			// across column boundaries and reorders chart cells, and
			// some quotations are of this repository's own prose rather
			// than the book's.
			if elsewhere := printedOn(missing, pages); len(elsewhere) > 0 {
				t.Errorf("%s cites %v, but %q is printed on %v",
					entry.id, entry.pages, trim(missing), elsewhere)

				continue
			}

			unmatched++
		}
	}

	if checked == 0 {
		t.Fatal("no quotations found in ERRATA.md; the sweep is asserting nothing")
	}

	t.Logf("%d quotations checked; %d could not be located in the extraction and were "+
		"not held against the citation", checked, unmatched)
}

// entry is one interpretation: its id, its body, and every page it cites.
type entry struct {
	id    string
	body  string
	pages []int
}

var (
	heading     = regexp.MustCompile(`(?m)^### (I-\d+):`)
	cite        = regexp.MustCompile(`pp?\.\s*(\d+)(?:\s*[-–]\s*(\d+))?`)
	quoted      = regexp.MustCompile(`"([^"]*)"`)
	emphasis    = regexp.MustCompile(`\*\*|__`)
	editorial   = regexp.MustCompile(`\[(\w+)\]`)
	elision     = regexp.MustCompile(`\s*(?:\.\.\.|…)\s*|\s+/\s+`)
	hyphenBreak = regexp.MustCompile(`-\s*\n\s*`)
	spaces      = regexp.MustCompile(`\s+`)
)

// interpretations splits ERRATA.md into its entries.
func interpretations(t *testing.T) []entry {
	t.Helper()

	src := read(t, docsDir+"ERRATA.md")

	marks := heading.FindAllStringSubmatchIndex(src, -1)
	if len(marks) == 0 {
		t.Fatal("ERRATA.md records no interpretations")
	}

	entries := make([]entry, 0, len(marks))

	for i, m := range marks {
		end := len(src)
		if i+1 < len(marks) {
			end = marks[i+1][0]
		}

		body := src[m[1]:end]
		entries = append(entries, entry{id: src[m[2]:m[3]], body: body, pages: citedPages(body)})
	}

	return entries
}

// citedPages lists every page an entry names, expanding a printed range.
func citedPages(body string) []int {
	seen := map[int]bool{}

	var pages []int

	for _, m := range cite.FindAllStringSubmatch(body, -1) {
		first, _ := strconv.Atoi(m[1])
		last := first

		if m[2] != "" {
			last, _ = strconv.Atoi(m[2])
		}

		for p := first; p <= last && p-first < 20; p++ {
			if !seen[p] {
				seen[p] = true
				pages = append(pages, p)
			}
		}
	}

	return pages
}

// quotations returns the quoted spans worth checking. Every pair is taken
// before any is filtered: dropping a short quote first would leave its
// closing mark to open the next, and every quotation after it would be
// read as the gap between two.
func quotations(body string) []string {
	var out []string

	for _, m := range quoted.FindAllStringSubmatch(body, -1) {
		if q := m[1]; len(q) >= minQuote && len(q) <= maxQuote {
			out = append(out, q)
		}
	}

	return out
}

// fragments splits one quotation into the contiguous runs the book must
// print, each as the spellings that would satisfy it.
//
// An ellipsis marks elided text and " / " joins chart cells printed
// apart, so neither can be matched as one string. An editorial "[s]" may
// be a letter the book lacks or one it has, so both are accepted.
func fragments(q string) [][]string {
	var runs [][]string

	for _, part := range elision.Split(emphasis.ReplaceAllString(q, ""), -1) {
		added := normalize(editorial.ReplaceAllString(part, "$1"))
		dropped := normalize(editorial.ReplaceAllString(part, ""))

		alternatives := []string{added}
		if dropped != added {
			alternatives = append(alternatives, dropped)
		}

		if len(added) >= 12 {
			runs = append(runs, alternatives)
		}
	}

	return runs
}

// firstUnmatched reports the first run of a quotation that no cited page
// prints, or "" when every run is accounted for.
//
// Consecutive cited pages are also searched joined, because a quotation
// may run over a page break.
func firstUnmatched(runs [][]string, pages []int, book map[int]string) string {
	haystacks := make([]string, 0, len(pages)*2)

	for _, p := range pages {
		if text, ok := book[p]; ok {
			haystacks = append(haystacks, text)

			if next, ok := book[p+1]; ok {
				haystacks = append(haystacks, text+" "+next)
			}
		}
	}

	for _, alternatives := range runs {
		if !printedAnywhere(alternatives, haystacks) {
			return alternatives[0]
		}
	}

	return ""
}

// printedOn lists the pages that print a run, for a run no cited page
// does. At most three: the message is a pointer, not a concordance.
func printedOn(run string, book map[int]string) []int {
	var found []int

	for page := 1; page <= len(book); page++ {
		if strings.Contains(book[page], run) {
			found = append(found, page)

			if len(found) == 3 {
				break
			}
		}
	}

	return found
}

// printedAnywhere reports whether any spelling of a run appears in any of
// the pages.
func printedAnywhere(alternatives, haystacks []string) bool {
	for _, run := range alternatives {
		for _, page := range haystacks {
			if strings.Contains(page, run) {
				return true
			}
		}
	}

	return false
}

// normalize renders text comparable across the gap between a Markdown
// document and a PDF extraction: the quotation marks and dashes differ by
// character, the extraction breaks words across lines with a hyphen, and
// it wraps wherever the column ended.
func normalize(text string) string {
	for _, pair := range [][2]string{
		{"\u2019", "'"},
		{"\u2018", "'"},
		{"\u201c", `"`},
		{"\u201d", `"`},
		{"\u2014", "-"},
		{"\u2013", "-"},
		{"\u2212", "-"},
		{"\u00a0", " "},
		{"\ufb01", "fi"},
		{"\ufb02", "fl"},
	} {
		text = strings.ReplaceAll(text, pair[0], pair[1])
	}

	text = hyphenBreak.ReplaceAllString(text, "")
	text = spaces.ReplaceAllString(text, " ")

	return strings.ToLower(strings.TrimSpace(text))
}

// trim shortens a run for a failure message.
func trim(run string) string {
	if len(run) <= 72 {
		return run
	}

	return run[:72] + "…"
}

// rulesPages extracts Book 1 a page at a time, keyed by printed page
// number.
//
// One pdftotext run rather than one per page: the tool separates pages
// with a form feed, so splitting on it is both faster and the only thing
// that keeps the numbering honest.
func rulesPages(t *testing.T) map[int]string {
	t.Helper()

	pdf := os.Getenv(rulesPDFEnv)
	if pdf == "" {
		t.Skipf("set %s to sweep ERRATA.md's citations against the printed rules; "+
			"`task citations` does. The rules are not redistributed with this repository.",
			rulesPDFEnv)
	}

	// G703/G204: the path is the caller's own, named on their own machine
	// through an environment variable they set, and the file it points at
	// is a book they bought. There is no untrusted input here to traverse
	// with, and the alternative — hard-coding one path — would make the
	// gate unusable for anyone whose collection sits elsewhere.
	if _, err := os.Stat(pdf); err != nil { //nolint:gosec // G703: the path is the caller's own.
		t.Fatalf("%s names %q, which cannot be read: %v", rulesPDFEnv, pdf, err)
	}

	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Skip("pdftotext is not installed; it comes with poppler")
	}

	ctx, cancel := context.WithTimeout(t.Context(), pdfExtractTimeout)
	defer cancel()

	//nolint:gosec // G204: as above — the path is the caller's own.
	out, err := exec.CommandContext(ctx, "pdftotext", "-raw", pdf, "-").Output()
	if err != nil {
		t.Fatalf("extracting %q: %v", pdf, err)
	}

	pages := map[int]string{}
	for i, page := range strings.Split(string(out), "\f") {
		pages[i+1] = normalize(page)
	}

	if len(pages) < 100 {
		t.Fatalf("extracted only %d pages from %q; that is not Book 1", len(pages), pdf)
	}

	return pages
}
