package chargen_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/chargen"
)

// TestARefusalToBeginNamesAThrow holds the line between a prerequisite and
// a failed attempt, which p. 65 draws: "Pre-Requisites. Some Careers have
// requirements before a character may attempt to Begin", and separately
// "If both Begin and Retry fail, this career may not be used. Each failed
// attempt (both Begin or Retry) takes one year."
//
// A career that refuses a character must therefore have rolled for it. A
// refusal caused by the selection itself is a prerequisite enforced too
// late — the character was offered a career he could never have, chose it,
// and was told afterwards. That is how chart 11's "if Soc B+" hid until it
// was reported from play (I-28), and this is the guard that keeps the next
// one from doing the same.
func TestARefusalToBeginNamesAThrow(t *testing.T) {
	refusals := 0

	for _, name := range career.Available() {
		for seed := uint64(1); seed <= 40; seed++ {
			c, open := generateIfOpen(t, chargen.Options{Seed: seed, Career: name})
			if !open {
				continue // the rules deny the career outright; nothing was attempted
			}

			kinds := make(map[int]chargen.EventKind, len(c.Events))
			for _, e := range c.Events {
				kinds[e.Seq] = e.Kind
			}

			for _, e := range c.Events {
				if e.Kind != chargen.EventConsequence ||
					e.Consequence.Kind != chargen.ConsequenceCareerNotBegun {
					continue
				}

				refusals++

				if kind := kinds[e.Consequence.Cause]; kind != chargen.EventThrow {
					t.Errorf("%s seed %d: refused to begin on a %s at event %d, not a throw — "+
						"a prerequisite enforced after the character chose it",
						name, seed, kind, e.Consequence.Cause)
				}
			}
		}
	}

	if refusals == 0 {
		t.Fatal("no career refused anybody across the sweep; the guard is asserting nothing")
	}
}

// automaticPeek records the Select Career offer and its scores.
type automaticPeek struct {
	options []string
	scores  []int
}

func (d *automaticPeek) Choose(c chargen.Choice) (int, error) {
	if c.ID == chargen.ChooseCareer && d.options == nil {
		d.options = slices.Clone(c.Options)
		d.scores = slices.Clone(c.Scores)
	}

	return autoPolicy(c)
}

func (*automaticPeek) Kind() chargen.DeciderKind { return chargen.DeciderPlayer }

// TestAnAutomaticEntryNeverFails ties the declaration to the behaviour.
//
// "[automatic]" is a promise to a player weighing chart E1 step D: this
// career costs him nothing to try, where every other one risks the year
// p. 65 charges for a failed attempt. A career the data marks automatic
// that then refuses somebody would be the menu lying to him.
//
// Both directions are checked. A career scored 1 must begin for every
// character offered it; and at least one scored 0 must fail somewhere in
// the sweep, or the annotation would be marking a distinction the engine
// does not actually make.
func TestAnAutomaticEntryNeverFails(t *testing.T) {
	promised, refusedElsewhere := 0, 0

	for seed := uint64(1); seed <= 60; seed++ {
		peek := &automaticPeek{}

		c, err := chargen.Generate(chargen.Options{Seed: seed, CurrentYear: 1105, Decider: peek})
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}

		if len(peek.scores) != len(peek.options) {
			t.Fatalf("seed %d: %d careers offered but %d scores", seed, len(peek.options), len(peek.scores))
		}

		for i, name := range peek.options {
			kept, refused := checkEntryPromise(t, seed, name, peek.scores[i])
			promised += kept
			refusedElsewhere += refused
		}

		_ = c
	}

	if promised == 0 {
		t.Fatal("no career was marked automatic across the sweep; the promise is untested")
	}

	if refusedElsewhere == 0 {
		t.Fatal("no unmarked career ever refused anybody; the annotation marks no real distinction")
	}
}

// TestScholarThrowsOnlyBelowEduEight pins chart 02's conditional itself:
// "(Edu 8+) Automatic". Above it there is no To Begin throw, below it
// there is one.
//
// Needed because the condition now lives in the career data, and the data
// drives both the annotation and the entry — so a wrong condition is
// consistent with itself and neither the menu nor the behaviour would
// disagree. Falsifying it to "always" leaves TestAnAutomaticEntryNeverFails
// green, which is how this gap was found. Only a claim about what the
// chart says can catch it.
func TestScholarThrowsOnlyBelowEduEight(t *testing.T) {
	above, below := 0, 0

	for seed := uint64(1); seed <= 80; seed++ {
		c, open := generateIfOpen(t, chargen.Options{Seed: seed, Career: "Scholar", CurrentYear: 1105})
		if !open || len(c.Careers) == 0 {
			continue
		}

		threw := hasStep(c, "Scholar: To Begin") && beginThrew(c)

		if c.Characteristics.Edu >= 8 {
			above++

			if threw {
				t.Errorf("seed %d: a Scholar at Edu %d rolled To Begin, and chart 02 makes it automatic",
					seed, c.Characteristics.Edu)
			}

			continue
		}

		below++

		if !threw {
			t.Errorf("seed %d: a Scholar at Edu %d entered without a To Begin throw",
				seed, c.Characteristics.Edu)
		}
	}

	if above == 0 || below == 0 {
		t.Fatalf("the sweep saw %d above Edu 8 and %d below; it is asserting one side only", above, below)
	}
}

// beginThrew reports whether a throw was recorded against the To Begin
// cite, which is the entry roll and not the term's later ones.
func beginThrew(c chargen.Character) bool {
	for _, e := range c.Events {
		if e.Kind == chargen.EventThrow && e.Throw != nil &&
			strings.Contains(e.Throw.Cite, "To Begin") {
			return true
		}
	}

	return false
}

// checkEntryPromise forces one career and reports whether it kept an
// automatic promise, and whether an unmarked one refused the character.
func checkEntryPromise(t *testing.T, seed uint64, name string, score int) (int, int) {
	t.Helper()

	forced, open := generateIfOpen(t, chargen.Options{Seed: seed, Career: name, CurrentYear: 1105})
	if !open || len(forced.Careers) == 0 {
		return 0, 0
	}

	began := forced.Careers[0].Began

	if score != 1 {
		if !began {
			return 0, 1
		}

		return 0, 0
	}

	if !began {
		t.Errorf("seed %d: %s was marked automatic and then refused the character", seed, name)
	}

	return 1, 0
}
