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

// TestASignedAnswerIsNotAnIndex verifies the benefit DM menu, whose
// options are printed "+0", "+1", "+2". A player copying the option he
// wants must not silently get the one above it: strconv.Atoi reads "+2"
// as 2, which is the third option's index only by accident. A signed
// answer is a search term, and the search shows the option under its own
// number.
func TestASignedAnswerIsNotAnIndex(t *testing.T) {
	dm := chargen.Choice{
		ID:      chargen.ChooseBenefitDM,
		Prompt:  "How much of the DM to apply?",
		Options: []string{"+0", "+1", "+2", "+3"},
	}

	index, out, err := ask(t, "+2\n3\n", dm)
	if err != nil {
		t.Fatal(err)
	}

	if index != 2 {
		t.Errorf("chose index %d, want 2 — %q was read as an option number", index, "+2")
	}

	if !strings.Contains(out, "3  +2") {
		t.Errorf("the searched option was not listed under its own number:\n%s", out)
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
		"input ends":           "",
		"quit confirmed":       "q\ny\n",
		"quit confirmed upper": "Q\nY\n",
		"quit then input ends": "q\n",
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
		Decider: interactive.New(strings.NewReader("q\ny\n"), &strings.Builder{}),
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

	// The qualification score renders as what it costs, and only where it
	// costs something: "[qualifies 0]" read as a refusal when it meant
	// that picking the row would cost a waiver attempt (I-95).
	if !strings.Contains(out, "University  [needs a waiver]") {
		t.Errorf("the row he falls short of does not say what choosing it costs:\n%s", out)
	}

	if strings.Contains(out, "qualifies") {
		t.Errorf("the qualification is still rendered as a digit:\n%s", out)
	}

	if strings.Contains(out, "College  [") {
		t.Errorf("a row he qualifies for was annotated; only the exceptions carry one:\n%s", out)
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

// TestQuitIsConfirmed verifies a mistyped abandon does not throw the
// session away. The prompt invites text to search, single letters are a
// natural way to search a hundred entries, and abandoning leaves no output
// file at all — so it asks first, and anything but confirmation carries on.
func TestQuitIsConfirmed(t *testing.T) {
	index, out, err := ask(t, "q\nn\n2\n", threeWay)
	if err != nil {
		t.Fatalf("a declined abandon ended the session: %v", err)
	}

	if index != 1 {
		t.Errorf("chose index %d, want the 2 typed after declining to abandon", index)
	}

	if !strings.Contains(out, "Abandon?") {
		t.Errorf("no confirmation was asked:\n%s", out)
	}
}

// TestSignedAnswersAreNotOptionNumbers verifies an answer is read as an
// option number only when it is one.
//
// The muster-out DM menu prints its options as "+0 +1 +2 +3". A player
// copying the one he wants types "+2" — and strconv.Atoi accepts a leading
// sign, so that used to select index 2, the *third* option, handing him +1
// and recording it as his own choice with no error. A wrong answer
// attributed to the player is worse than a rejected one.
func TestSignedAnswersAreNotOptionNumbers(t *testing.T) {
	dm := chargen.Choice{
		ID:      chargen.ChooseBenefitDM,
		Prompt:  "How much of the DM to apply?",
		Options: []string{"+0", "+1", "+2", "+3"},
	}

	index, out, err := ask(t, "+2\n3\n", dm)
	if err != nil {
		t.Fatal(err)
	}

	if index != 2 {
		t.Errorf("chose index %d (%q), want the 3 typed after the search", index, dm.Options[index])
	}

	// "+2" is a search, and it matches its own option, listed under its
	// own number so the player can see which one it is.
	if !strings.Contains(out, "  3  +2") {
		t.Errorf("the signed answer was not shown as a search hit:\n%s", out)
	}
}

// sessionAnswers is more than any run below needs; the reader is
// exhausted only if the engine asks for more, which fails as abandonment.
const (
	sessionAnswers = 4000

	// sessionSeed is pinned: its Scholar runs long enough to reach a
	// career, its skills and muster out, which is the whole checklist.
	sessionSeed = 3
)

// session drives a whole generation and returns everything the player saw.
func session(t *testing.T) string {
	t.Helper()

	var out strings.Builder

	_, err := chargen.Generate(chargen.Options{
		Seed:    sessionSeed,
		Decider: interactive.New(strings.NewReader(strings.Repeat("1\n", sessionAnswers)), &out),
	})
	if err != nil {
		t.Fatalf("the session did not finish: %v", err)
	}

	return out.String()
}

// TestTheChecklistIsVisible verifies the session shows the checklist a
// player is following. Chart E1 is lettered A to D and the engine walks it
// in order; before this the session showed none of it, opening at the
// homeworld with no indication that anything had come before.
func TestTheChecklistIsVisible(t *testing.T) {
	shown := session(t)

	steps := []string{
		"Generate Characteristics", // E1 step A
		"Determine A Homeworld",    // E1 step B
		"Education and Training",   // E1 step C
		"Select Career",            // E1 step D
	}

	at := 0

	for _, step := range steps {
		found := strings.Index(shown[at:], step)
		if found < 0 {
			t.Fatalf("the session never shows %q", step)
		}

		at += found
	}
}

// TestCharacteristicsComeFirst verifies step A's result reaches the player
// before he is asked anything.
//
// The six are rolled before any decision and every decision depends on
// them — which education admits him, which characteristic to check — and
// the session used to show none of them, so a player chose an education
// without knowing his own Edu.
func TestCharacteristicsComeFirst(t *testing.T) {
	shown := session(t)

	upp := strings.Index(shown, "UPP ")
	if upp < 0 {
		t.Fatal("the session never shows the characteristics")
	}

	question := strings.Index(shown, "a number to choose")
	if question < 0 {
		t.Fatal("the session asks nothing")
	}

	if upp > question {
		t.Error("the first question is asked before the characteristics are shown")
	}

	// Under step A's own heading, not under whatever step happens to be
	// entered before the first question.
	stepA := strings.Index(shown, "Generate Characteristics")
	stepB := strings.Index(shown, "Determine A Homeworld")

	if upp < stepA || upp > stepB {
		t.Error("the characteristics are not shown under step A")
	}
}

// TestRepeatedQuestionsAreNumbered verifies a run of identical questions
// says which is which. A term's skills are the same question asked several
// times, and five of them in a row with nothing to tell them apart is how
// a player loses his place.
func TestRepeatedQuestionsAreNumbered(t *testing.T) {
	shown := session(t)

	if !strings.Contains(shown, "Skills column  (1 of ") {
		t.Error("a run of skill selections does not say where in the run it is")
	}

	// The last of a run must be reachable too, or the count is decorative.
	if !strings.Contains(shown, "Skills column  (2 of ") {
		t.Error("only the first of a run is numbered")
	}
}

// TestTheStatusLineCarriesTheCharacter verifies every question is asked
// with the character's state above it, which is what makes an answer an
// informed one.
func TestTheStatusLineCarriesTheCharacter(t *testing.T) {
	shown := session(t)

	for _, want := range []string{"age ", "UPP ", " skill"} {
		if !strings.Contains(shown, want) {
			t.Errorf("no question is asked with %q above it", want)
		}
	}

	// Age is what tells a player his career has run long — the thing a
	// session of answering "continue" hides until the record is read.
	if !strings.Contains(shown, "age 2") && !strings.Contains(shown, "age 3") {
		t.Error("the running age is never shown")
	}
}

// TestTheSessionAgreesWithTheRecord verifies the running header describes
// the character the engine is actually building.
//
// The session keeps a shadow of the character because there is none to
// read while it is being made, and a shadow can drift: this one showed a
// Str of 0 for a character whose Str was 1, because aging zeroes a
// characteristic and a separate consequence resets it to 1 (p. 89 chart
// A) and only the first was being folded in. Twelve of four hundred seeds
// were wrong and nothing compared the two.
//
// A sweep rather than a pinned seed, because the case needs aging deep
// enough to zero something, and which seeds do that is not a fact worth
// pinning.
func TestTheSessionAgreesWithTheRecord(t *testing.T) {
	checked, reset := 0, 0

	for seed := range uint64(sessionSeedSweep) {
		var out strings.Builder

		player := interactive.New(strings.NewReader(strings.Repeat("1\n", sessionAnswers)), &out)

		c, err := chargen.Generate(chargen.Options{Seed: seed, Decider: player})
		if err != nil {
			continue
		}

		checked++

		if !strings.Contains(lastStatus(out.String()), c.UPP) {
			t.Errorf("seed %d: the session's last header does not carry the record's UPP %q:\n  %s",
				seed, c.UPP, lastStatus(out.String()))
		}

		for _, event := range c.Events {
			if event.Kind == chargen.EventConsequence &&
				event.Consequence.Kind == chargen.ConsequenceCharacteristicReset {
				reset++

				break
			}
		}
	}

	if checked == 0 {
		t.Fatal("no session completed; the sweep is asserting nothing")
	}

	// The drift needed a characteristic reset to show itself, so a sweep
	// that reached none would pass without testing the thing it is for.
	if reset == 0 {
		t.Errorf("no seed under %d reset a characteristic; widen the sweep", sessionSeedSweep)
	}
}

// sessionSeedSweep is wide enough to reach the aging that zeroes a
// characteristic, which is what the drift depended on.
const sessionSeedSweep = 120

// lastStatus returns the final status line a session printed.
func lastStatus(shown string) string {
	last := ""

	for line := range strings.SplitSeq(shown, "\n") {
		if strings.HasPrefix(line, "age ") {
			last = line
		}
	}

	return last
}
