package render_test

import (
	"os"
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/chargen"
	"github.com/philoserf/t5chargen/render"
)

// golden compares got against the named testdata file.
func golden(t *testing.T, got, file string) {
	t.Helper()

	want, err := os.ReadFile(file) //nolint:gosec // G304: fixed test-owned paths under testdata/.
	if err != nil {
		t.Fatal(err)
	}

	if got != string(want) {
		t.Errorf("output differs from %s:\n%s", file, got)
	}
}

// generate builds a character for render tests with the auto policy.
func generate(t *testing.T, opts chargen.Options) chargen.Character {
	t.Helper()

	opts.Decider = chargen.DefaultPolicy{}

	c, err := chargen.Generate(opts)
	if err != nil {
		t.Fatal(err)
	}

	return c
}

func TestSheetGolden(t *testing.T) {
	golden(t, render.Sheet(generate(t, chargen.Options{Seed: 1})), "testdata/seed1_sheet.md")
}

func TestHistoryGolden(t *testing.T) {
	golden(t, render.History(generate(t, chargen.Options{Seed: 1})), "testdata/seed1_history.md")
}

func TestSheetWithName(t *testing.T) {
	sheet := render.Sheet(generate(t, chargen.Options{Seed: 1, Name: "Eneri Dinsha"}))

	if !strings.Contains(sheet, "**Name**: Eneri Dinsha\n") {
		t.Errorf("sheet missing name line:\n%s", sheet)
	}

	if strings.Contains(sheet, " \n") {
		t.Error("sheet contains trailing whitespace")
	}
}

// TestHistoryMalformedEvents verifies malformed records render marked
// lines instead of panicking: kind without payload, and an out-of-range
// chosen index.
func TestHistoryMalformedEvents(t *testing.T) {
	c := chargen.Character{Events: []chargen.Event{
		{Seq: 1, Kind: chargen.EventStep},
		{Seq: 2, Kind: chargen.EventThrow},
		{Seq: 3, Kind: chargen.EventConsequence},
		{Seq: 4, Kind: chargen.EventChoice, Choice: &chargen.ChoiceEvent{
			Decider: chargen.DeciderPolicy,
			Options: []string{"a"},
			Chosen:  9,
		}},
	}}

	got := render.History(c)

	for _, want := range []string{
		"- #1 (step) [malformed event]\n",
		"- #2 (throw) [malformed event]\n",
		"- #3 (consequence) [malformed event]\n",
		"[chosen 9 out of range]",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("History() missing %q:\n%s", want, got)
		}
	}
}

// TestHistoryConsequenceTexts covers the consequence renderings the seed-1
// golden does not pin: the ERRATA I-1 job_undetermined line and the
// cap-absorbed no_award carrying the skill name.
func TestHistoryConsequenceTexts(t *testing.T) {
	c := chargen.Character{Events: []chargen.Event{
		{Seq: 1, Kind: chargen.EventConsequence, Consequence: &chargen.ConsequenceEvent{
			Cause: 1, Kind: chargen.ConsequenceJobUndetermined,
		}},
		{Seq: 2, Kind: chargen.EventConsequence, Consequence: &chargen.ConsequenceEvent{
			Cause: 1, Kind: chargen.ConsequenceNoAward, Skill: "Admin",
		}},
	}}

	got := render.History(c)

	for _, want := range []string{
		"Job undetermined (No Skill); retries next success — ERRATA I-1",
		"no award (Admin at the Skill-15 cap)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("History() missing %q:\n%s", want, got)
		}
	}
}

// TestHistoryThrowLine covers the targeted-throw rendering (target, mods,
// success) that seed-1 characteristics generation does not yet exercise.
func TestHistoryThrowLine(t *testing.T) {
	target := 8
	success := true

	c := chargen.Character{Events: []chargen.Event{
		{Seq: 1, Kind: chargen.EventThrow, Throw: &chargen.ThrowEvent{
			Expr:    "2D",
			Dice:    []int{3, 4},
			Total:   7,
			Target:  &target,
			Success: &success,
			Mods:    []chargen.Mod{{Name: "homeworld", Value: 2}},
			Cite:    "test cite",
		}},
	}}

	want := "# Generation Record\n- #1 2D = 3+4 = 7 vs 8 (homeworld +2): success — test cite\n"
	if got := render.History(c); got != want {
		t.Errorf("History() =\n%q\nwant\n%q", got, want)
	}
}

// TestHistoryChoiceLine covers choice rendering ahead of the first engine
// choice point.
func TestHistoryChoiceLine(t *testing.T) {
	c := chargen.Character{Events: []chargen.Event{
		{Seq: 1, Kind: chargen.EventChoice, Choice: &chargen.ChoiceEvent{
			Decider: chargen.DeciderPolicy,
			Prompt:  "Select career",
			Options: []string{"Citizen", "Scout"},
			Chosen:  0,
			Cite:    "test cite",
		}},
	}}

	want := "# Generation Record\n- #1 policy chose \"Citizen\" of [Citizen, Scout]: Select career — test cite\n"
	if got := render.History(c); got != want {
		t.Errorf("History() =\n%q\nwant\n%q", got, want)
	}
}
