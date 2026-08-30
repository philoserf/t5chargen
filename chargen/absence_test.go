package chargen_test

import (
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/chargen"
)

// The rules a chart states by leaving something out, and which this
// engine therefore models by absence.
//
// Absence is what rots quietly. A rank ladder deleted by accident, or a
// Caution/Bravery selection added to a career whose box never printed
// one, changes a generated character and moves no row of any document —
// so these are the assertions that a thing is still missing.

// TestOnlySixCareersHaveNoRank verifies p. 65 by name: "The Citizen,
// Entertainer, Craftsman, Scout, Agent, and Rogue careers have no rank."
//
// Both directions. Six careers must have no ladder, and the other seven
// must have one — a test that only checked the six would pass on a
// transcription that gave every career none.
func TestOnlySixCareersHaveNoRank(t *testing.T) {
	rankless := map[string]bool{
		"Citizen": true, "Entertainer": true, "Craftsman": true,
		"Scout": true, "Agent": true, "Rogue": true,
	}

	names := career.Available()
	if len(names) != 13 {
		t.Fatalf("chart 01-13 is thirteen careers, and %d are registered", len(names))
	}

	for _, name := range names {
		def, err := career.ByName(name)
		if err != nil {
			t.Fatalf("loading %s: %v", name, err)
		}

		if rankless[name] && len(def.Ranks) != 0 {
			t.Errorf("%s has %d ranks; p. 65 says it has none", name, len(def.Ranks))
		}

		if !rankless[name] && len(def.Ranks) == 0 {
			t.Errorf("%s has no ranks; p. 65 lists only six careers without them", name)
		}
	}
}

// TestTheNobleSelectsNoRiskMod verifies chart 11's omission: its box
// defines its own Mods and does not print the "Select Caution, Bravery,
// or No Mod" line every other Risk & Reward chart carries.
//
// Asserted on the record rather than on the data, because the selection
// is a choice point rather than a field: what would be wrong is a Noble
// being asked. ChoiceEvent records the prompt and not the id, so the
// prompt is what a reader of the log has to go on too.
func TestTheNobleSelectsNoRiskMod(t *testing.T) {
	served := 0

	for seed := range uint64(400) {
		c := generate(t, chargen.Options{Seed: seed, Decider: intriguingNoble{}})

		if !servedAs(c, "Noble") {
			continue
		}

		served++

		// Bounded to the Noble's own terms. A character changes into
		// this career, and the one he left prints the selection line —
		// scanning the whole log would find that and call it a Noble's.
		inNoble := false

		for _, event := range c.Events {
			if event.Step != nil {
				inNoble = strings.HasPrefix(event.Step.Name, "Noble: ")
			}

			if inNoble && event.Choice != nil &&
				event.Choice.Prompt == "Select Caution, Bravery, or No Mod" {
				t.Fatalf("seed %d: a Noble was offered Caution or Bravery", seed)
			}
		}
	}

	if served == 0 {
		t.Fatalf("no seed under 400 serves a Noble term; the sweep is asserting nothing")
	}
}

// intriguingNoble changes into the Noble career, which the auto policy
// never does and which needs Soc B+ (p. 85).
type intriguingNoble struct{ playerKind }

func (intriguingNoble) Choose(c chargen.Choice) (int, error) {
	switch c.ID { //nolint:exhaustive // Only the choice points this decider steers.
	case chargen.ChooseCareerChange:
		return 1, nil // "Change careers"
	case chargen.ChooseCareer:
		for i, option := range c.Options {
			if option == "Noble" {
				return i, nil
			}
		}
	}

	return autoPolicy(c)
}

// servedAs reports whether the record shows a term of the named career.
func servedAs(c chargen.Character, name string) bool {
	for _, record := range c.Careers {
		if record.Career == name && record.Began && len(record.Terms) > 0 {
			return true
		}
	}

	return false
}
