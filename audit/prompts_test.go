package audit_test

// The gate on identifiers reaching a player.
//
// A choice's Prompt and Options are two things at once: the words the
// interactive front end puts on screen, and content recorded in the event
// log and compared byte for byte on replay. That makes a careless prompt
// both a user-facing defect and a versioned one — which is how "How much
// of the scholar_rank DM to apply?" survived: it read as a field name to
// anyone who knew the data, and nothing was checking that a player did
// not have to.
//
// Every kind, id and column in this repo's chart data is snake_case, and
// nothing Book 1 prints contains an underscore. So an underscore in a
// player-facing string is not a judgement call about wording: it is a
// transcription of an identifier, and it is always wrong.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// identifier matches the snake_case shape every id in the chart data has.
var identifier = regexp.MustCompile(`\b[a-z][a-z0-9]*(_[a-z0-9]+)+\b`)

// choiceText is the part of a record this gate reads: the strings a
// deciding human is shown.
type choiceText struct {
	Events []struct {
		Seq    int `json:"seq"`
		Choice *struct {
			Prompt  string   `json:"prompt"`
			Options []string `json:"options"`
		} `json:"choice"`
	} `json:"events"`
}

// TestNoPromptShowsAnIdentifier reads every golden record and requires
// that nothing a player is asked to answer names a field instead of a
// rule.
//
// Two sources, because neither covers the other.
//
// The fixtures are the engine's own output over all thirteen careers, so a
// prompt built by string concatenation from chart data is caught wherever
// the lifepath actually reaches it. What they cannot see is a path the
// auto policy declines: "Select a service" and "Choose Honors" appear in
// none of the fourteen records, because the policy never takes the Service
// Academy. That blindness is not hypothetical — it is where the Academy's
// own defects survived until they were reported from play.
//
// So the prompt literals in the source are scanned too. That reaches every
// static prompt whether or not a policy run arrives at it, and leaves only
// the interpolated half to the fixtures.
func TestNoPromptShowsAnIdentifier(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "chargen", "testdata", "*.json"))
	if err != nil {
		t.Fatal(err)
	}

	if len(files) == 0 {
		t.Fatal("no fixtures found; this gate is asserting nothing")
	}

	checked := 0
	for _, file := range files {
		checked += scanChoices(t, file)
	}

	// The fixtures are regenerated, so a scan that silently found no
	// choices at all would pass while checking nothing.
	if checked < 100 {
		t.Fatalf("only %d choices scanned; the fixtures are not being read", checked)
	}

	scanPromptLiterals(t)
}

// scanPromptLiterals checks the prompt strings written in the engine's
// source, which covers the choice points no auto-mode fixture reaches.
//
// Only the literal part of an interpolated prompt is visible here; the
// fixtures carry the rest, where a run gets to it.
func scanPromptLiterals(t *testing.T) {
	t.Helper()

	found, _ := locate(t, inPackage("chargen"), regexp.MustCompile(`Prompt:\s*"([^"]*)"`))

	for _, h := range found {
		if shown := identifier.FindString(h.match[1]); shown != "" {
			t.Errorf("%s writes the prompt %q, which shows the identifier %q",
				filepath.Base(h.path), h.match[1], shown)
		}
	}

	if len(found) < 30 {
		t.Fatalf("only %d prompt literals found; the source scan is not working", len(found))
	}
}

// scanChoices reports every identifier shown by one record's choices, and
// returns how many choices it read.
func scanChoices(t *testing.T, file string) int {
	t.Helper()

	data, err := os.ReadFile(file) //nolint:gosec // a fixture path the caller globbed
	if err != nil {
		t.Fatal(err)
	}

	var record choiceText
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}

	checked := 0

	for _, event := range record.Events {
		if event.Choice == nil {
			continue
		}

		checked++

		shown := append([]string{event.Choice.Prompt}, event.Choice.Options...)
		for _, text := range shown {
			if found := identifier.FindString(text); found != "" {
				t.Errorf("%s event %d shows %q in %q: that is an identifier, not something the book prints",
					filepath.Base(file), event.Seq, found, strings.TrimSpace(text))
			}
		}
	}

	return checked
}
