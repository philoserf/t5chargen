package chargen_test

import (
	"testing"

	"github.com/philoserf/t5chargen/chargen"
)

// repeater takes the named program at every choice point that offers it,
// which is how the pump was found: each College graduation past the first
// awards Edu+1 under chart C's "(If Edu already at this level, award
// Edu+1)", so twenty-three of them reached Edu-F at age 110.
type repeater struct{ program string }

func (d repeater) Choose(c chargen.Choice) (int, error) {
	for i, option := range c.Options {
		if option == d.program {
			return i, nil
		}
	}

	return autoPolicy(c)
}

func (repeater) Kind() chargen.DeciderKind { return chargen.DeciderPlayer }

// repeatable names every program a player can reach through step C or
// Later Education. Assigned schools are deliberately absent: they are
// sited inside a term by the career rather than applied for, and a second
// promotion may well site a second one.
var repeatable = []string{"ED5", "Trade School", "Apprenticeship", "College", "University"}

// TestNoProgramIsAttemptedTwice is interpretation I-100 measured at the
// record. ED5 is the case Book 1 states outright — "The process can be
// attempted once" (p. 61) — and the others follow the same reading.
func TestNoProgramIsAttemptedTwice(t *testing.T) {
	for _, program := range repeatable {
		t.Run(program, func(t *testing.T) {
			reached := 0

			for seed := uint64(1); seed <= 40; seed++ {
				character, err := chargen.Generate(chargen.Options{
					Seed: seed, CurrentYear: 1105, Decider: repeater{program: program},
				})
				if err != nil {
					t.Fatalf("seed %d: %v", seed, err)
				}

				attempts := 0

				for _, record := range character.Education {
					if record.Program == program {
						attempts++
					}
				}

				if attempts > 1 {
					t.Errorf("seed %d: attempted %s %d times", seed, program, attempts)
				}

				reached += attempts
			}

			if reached == 0 {
				t.Fatalf("no character reached %s across the sweep; it is asserting nothing", program)
			}
		})
	}
}

// TestRepeatSchoolingCannotPumpEdu is the same rule measured at the damage
// it was doing. Chart C's graduation values are fixed — "Edu=8 BA" — and
// the parenthetical that softens them, "(If Edu already at this level,
// award Edu+1)", is there for a character who already exceeds the value,
// not to be farmed a degree at a time.
//
// Edu-15 is the characteristic maximum for Humans (p. 68), so a character
// who reached it through schooling alone had found the ratchet.
func TestRepeatSchoolingCannotPumpEdu(t *testing.T) {
	for _, program := range repeatable {
		for seed := uint64(1); seed <= 40; seed++ {
			character, err := chargen.Generate(chargen.Options{
				Seed: seed, CurrentYear: 1105, Decider: repeater{program: program},
			})
			if err != nil {
				t.Fatalf("seed %d: %v", seed, err)
			}

			if edu := character.Characteristics.Edu; edu >= 15 {
				t.Errorf("%s seed %d: Edu reached %d by schooling", program, seed, edu)
			}
		}
	}
}
