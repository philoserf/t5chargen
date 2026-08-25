package chargen_test

import (
	"testing"

	"github.com/philoserf/t5chargen/chargen"
)

// greedyStudent takes whatever school is on offer, in chart order, at
// every opportunity. It is how the residual ratchet was found: seed 53
// rolls Edu 12 and then took ED5, a programme for a character with Edu
// under 5, for a level.
type greedyStudent struct {
	edu    int
	offers []eduOffer
}

// eduOffer is one offer, with the Edu the character held when it was put.
type eduOffer struct {
	options []string
	edu     int
}

// Watch tracks Edu as the engine records it, so an offer can be judged
// against the character at the moment it was made rather than against the
// character he later became. ED5 raises Edu to 5 itself, so his final
// value says nothing about whether he was eligible.
func (d *greedyStudent) Watch(event chargen.Event) {
	if event.Kind != chargen.EventConsequence || event.Consequence == nil {
		return
	}

	if event.Consequence.Characteristic == "Edu" {
		d.edu = event.Consequence.Value
	}
}

//nolint:exhaustive // Deliberately partitioned: the rest defer to the auto policy.
func (d *greedyStudent) Choose(c chargen.Choice) (int, error) {
	switch c.ID {
	case chargen.ChooseEducation, chargen.ChooseLaterEducation:
		d.offers = append(d.offers, eduOffer{options: c.Options, edu: d.edu})

		return d.climb(c), nil
	}

	return autoPolicy(c)
}

func (*greedyStudent) Kind() chargen.DeciderKind { return chargen.DeciderPlayer }

// climb reaches for the highest rung on offer.
//
// Chart order would take the Basic rows first, and with each programme
// limited to one attempt (I-100) they would all be spent before any degree
// was held — so a sweep driven in chart order can never present the case
// this file is about, and a test written on it passes while asserting
// nothing. Ask for the degree first, and the Basic rows are still
// unspent when it lands.
func (d *greedyStudent) climb(c chargen.Choice) int {
	for _, want := range []string{"University", "College"} {
		for i, option := range c.Options {
			if option == want {
				return i
			}
		}
	}

	for i, option := range c.Options {
		if option == "ED5" || option == "Trade School" || option == "Apprenticeship" {
			return i
		}
	}

	return 0
}

// TestED5IsNotOfferedAboveItsCeiling is the first half of I-102. ED5's
// "Edu 4 -" is the one prerequisite in chart C that is a ceiling, and a
// waiver overturns an adverse decision (p. 59) — being better educated
// than a remedial programme requires is not one.
func TestED5IsNotOfferedAboveItsCeiling(t *testing.T) {
	offers, seen := 0, 0

	for seed := uint64(1); seed <= 60; seed++ {
		student := &greedyStudent{}

		if _, err := chargen.Generate(chargen.Options{
			Seed: seed, CurrentYear: 1105, Decider: student,
		}); err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}

		for _, offer := range student.offers {
			offers++

			for _, option := range offer.options {
				if option != "ED5" {
					continue
				}

				seen++

				if offer.edu > 4 {
					t.Errorf("seed %d: ED5 offered at Edu %d, above its Edu 4 - ceiling", seed, offer.edu)
				}
			}
		}
	}

	if offers < 30 {
		t.Fatalf("only %d education offers across the sweep; it is asserting nothing", offers)
	}

	if seen == 0 {
		t.Fatal("ED5 was never offered at all; the sweep cannot tell a closed ceiling from a missing programme")
	}
}

// TestNoBasicSchoolingAfterADegree is the second half: chart C prints
// Basic and Higher Education as separate blocks, and p. 61 gives the
// ladder — ED5 exists "Because Edu-5 is the minimum prerequisite for Trade
// Schools". A graduate has climbed past the rung.
func TestNoBasicSchoolingAfterADegree(t *testing.T) {
	graduates := 0

	for seed := uint64(1); seed <= 60; seed++ {
		student := &greedyStudent{}

		c, err := chargen.Generate(chargen.Options{
			Seed: seed, CurrentYear: 1105, Decider: student,
		})
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}

		degreeAt := indexOfFirstDegree(c)
		if degreeAt < 0 {
			continue
		}

		graduates++

		for _, record := range c.Education[degreeAt+1:] {
			if record.Program == "ED5" || record.Program == "Trade School" ||
				record.Program == "Apprenticeship" {
				t.Errorf("seed %d: took %s after graduating a Higher Education programme",
					seed, record.Program)
			}
		}
	}

	if graduates == 0 {
		t.Fatal("no character graduated a Higher Education programme; the sweep is asserting nothing")
	}
}

// indexOfFirstDegree reports where the character first graduated College
// or University, or -1.
func indexOfFirstDegree(c chargen.Character) int {
	for i, record := range c.Education {
		if record.Graduated && (record.Program == "College" || record.Program == "University") {
			return i
		}
	}

	return -1
}
