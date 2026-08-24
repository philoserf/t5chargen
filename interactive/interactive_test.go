package interactive_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/chargen"
	"github.com/philoserf/t5chargen/interactive"
)

// ask puts one choice to a decider reading the given script.
func ask(t *testing.T, script string, c chargen.Choice) (int, string, error) {
	t.Helper()

	var out strings.Builder

	index, err := interactive.New(strings.NewReader(script), &out).Choose(c)

	return index, out.String(), err
}

// threeWay is an ordinary choice: a handful of options and a real
// decision.
var threeWay = chargen.Choice{
	ID:      chargen.ChooseRiskMod,
	Prompt:  "Select Caution, Bravery, or No Mod",
	Options: []string{"Caution +1", "No Mod", "Bravery -9"},
	Cite:    "Book 1 p. 65",
}

// TestAnswersByNumber verifies the ordinary case: the options are listed
// and a number picks one. The number is the option's own 1-based
// position, which is what the engine records and what replay reapplies.
func TestAnswersByNumber(t *testing.T) {
	for answer, want := range map[string]int{"1": 0, "2": 1, "3": 2} {
		index, _, err := ask(t, answer+"\n", threeWay)
		if err != nil {
			t.Fatalf("answer %q: %v", answer, err)
		}

		if index != want {
			t.Errorf("answer %q chose index %d, want %d", answer, index, want)
		}
	}
}

// TestRejectsAnswersOutsideTheList verifies a number the list does not
// hold is not taken. Returning it would reach the engine as an
// out-of-range answer and end generation, when what happened is that
// somebody mistyped.
func TestRejectsAnswersOutsideTheList(t *testing.T) {
	index, out, err := ask(t, "0\n9\n-1\n2\n", threeWay)
	if err != nil {
		t.Fatal(err)
	}

	if index != 1 {
		t.Errorf("chose index %d, want the 2 that was eventually typed", index)
	}

	if !strings.Contains(out, "nothing matches") {
		t.Error("an out-of-range number was not reported back to the player")
	}
}

// TestSingleOptionIsNotAQuestion verifies a choice with one option is
// answered without asking. The engine presents some choices that way as a
// seam rather than a decision — an assigned homeworld is the standing
// example — and the session must not stop for them, nor consume a line of
// input answering one.
func TestSingleOptionIsNotAQuestion(t *testing.T) {
	only := chargen.Choice{
		ID:      chargen.ChooseHomeworld,
		Prompt:  "Select a homeworld",
		Options: []string{"Regina A788899-C (Ph Pa Ri)"},
	}

	index, out, err := ask(t, "", only) // no input at all
	if err != nil {
		t.Fatalf("a single-option choice asked for input: %v", err)
	}

	if index != 0 {
		t.Errorf("chose index %d, want 0", index)
	}

	if !strings.Contains(out, "Regina") {
		t.Error("the answer was not shown to the player")
	}
}

// TestSearchNarrowsWithoutRenumbering verifies the filter. The engine
// presents lists of over a hundred — the Master Skill List is one — which
// is not a menu, so typing text narrows what is shown.
//
// What it must not do is renumber. The numbers stay the option's own, so
// an answer means the same thing whether it was typed against the whole
// list or a filtered view of it, and a player who searches and then
// changes his mind is not silently choosing something else.
func TestSearchNarrowsWithoutRenumbering(t *testing.T) {
	long := chargen.Choice{ID: chargen.ChooseSkill, Prompt: "Select a skill"}
	for i := range 40 {
		long.Options = append(long.Options, "Skill"+string(rune('A'+i%26))+strings.Repeat("x", i/26))
	}

	long.Options[37] = "Navigator"

	index, out, err := ask(t, "navig\n38\n", long)
	if err != nil {
		t.Fatal(err)
	}

	if index != 37 {
		t.Errorf("chose index %d, want 37 — searching renumbered the list", index)
	}

	if !strings.Contains(out, " 38  Navigator") {
		t.Errorf("the match was not listed under its own number:\n%s", out)
	}
}

// TestLongListsAreNotDumped verifies the whole of a long list is not
// printed at once, and that the player is told how much was held back.
func TestLongListsAreNotDumped(t *testing.T) {
	long := chargen.Choice{ID: chargen.ChooseSkill, Prompt: "Select a skill"}
	for i := range 100 {
		long.Options = append(long.Options, "Option"+strings.Repeat("y", i))
	}

	_, out, err := ask(t, "1\n", long)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(out, "100  ") {
		t.Error("the hundredth option was printed; the list was dumped rather than paged")
	}

	if !strings.Contains(out, "more") {
		t.Error("the player was not told options were held back")
	}
}

// TestAbandonment verifies both ways a session ends without an answer.
// docs/PRD.md's CLI sketch requires an interrupted session to produce no
// output file, and the engine learns that from this error — which must
// stay distinguishable from an out-of-range answer, a decider that replied
// wrongly rather than one that stopped replying.
func TestAbandonment(t *testing.T) {
	for name, script := range map[string]string{
		"input ends": "",
		"quit":       "q\n",
		"quit upper": "Q\n",
	} {
		_, _, err := ask(t, script, threeWay)
		if !errors.Is(err, interactive.ErrAbandoned) {
			t.Errorf("%s: err = %v, want ErrAbandoned", name, err)
		}
	}
}

// TestAbandonmentReachesTheEngine verifies the error survives generation
// intact rather than arriving as something else. The Decider contract
// added an error return for exactly this, and a session abandoned at the
// first prompt must be reported as abandoned rather than as a bad answer.
func TestAbandonmentReachesTheEngine(t *testing.T) {
	_, err := chargen.Generate(chargen.Options{
		Seed:    1,
		Decider: interactive.New(strings.NewReader("q\n"), &strings.Builder{}),
	})

	if !errors.Is(err, interactive.ErrAbandoned) {
		t.Fatalf("generation reported %v, want ErrAbandoned", err)
	}
}

// TestGeneratesACharacter verifies the decider drives a whole lifepath.
// Answering 1 to everything is a real player's laziest run, and it must
// reach a finished character rather than stalling on some choice the
// prompt loop cannot present.
func TestGeneratesACharacter(t *testing.T) {
	// Enough answers for the longest run this reaches; the reader is
	// exhausted only if the engine asks for more, which fails as
	// abandonment and names the choice it stopped at.
	script := strings.Repeat("1\n", 4000)

	c, err := chargen.Generate(chargen.Options{
		Seed:    1,
		Decider: interactive.New(strings.NewReader(script), &strings.Builder{}),
	})
	if err != nil {
		t.Fatalf("an all-ones session did not finish: %v", err)
	}

	if c.UPP == "" || len(c.Events) == 0 {
		t.Error("the session produced no character")
	}

	// The record must attest who decided (FR10).
	if c.PolicyVersion != "none" {
		t.Errorf("policy_version is %q for a player-decided run, want %q", c.PolicyVersion, "none")
	}

	player := 0

	for _, event := range c.Events {
		if event.Kind == chargen.EventChoice && event.Choice.Decider == chargen.DeciderPlayer {
			player++
		}
	}

	if player == 0 {
		t.Error("no choice in the record attests the player decided it")
	}
}

// TestAnInteractiveRecordReplays verifies the whole point of answering
// with indices: a session a person drove is replayable, because the
// choices it recorded are the same kind of data the policy's are.
func TestAnInteractiveRecordReplays(t *testing.T) {
	c, err := chargen.Generate(chargen.Options{
		Seed:    7,
		Decider: interactive.New(strings.NewReader(strings.Repeat("1\n", 4000)), &strings.Builder{}),
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := chargen.Replay(c); err != nil {
		t.Fatalf("an interactive record does not replay: %v", err)
	}
}

// TestScoresAreShownOnlyWhenNamed verifies the decision aid is rendered
// only where the engine says what it means. "1" against a programme means
// "you meet its prerequisite", which no player would read off the digit —
// a number nobody can interpret looks like information and is not.
func TestScoresAreShownOnlyWhenNamed(t *testing.T) {
	labelled := chargen.Choice{
		ID:         chargen.ChooseEducation,
		Prompt:     "Select pre-career education",
		Options:    []string{"College", "University"},
		Scores:     []int{1, 0},
		ScoreLabel: "qualifies",
	}

	_, out, err := ask(t, "1\n", labelled)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out, "[qualifies 0]") {
		t.Errorf("a named score was not shown:\n%s", out)
	}

	bare := labelled
	bare.ScoreLabel = ""

	_, out, err = ask(t, "1\n", bare)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(out, "[") {
		t.Errorf("an unnamed score was shown as a bare number:\n%s", out)
	}
}
