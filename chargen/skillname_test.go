package chargen_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/chargen"
	"github.com/philoserf/t5chargen/skill"
)

// TestAwardedSkillsAreCanonical verifies that every skill a generated
// character holds is a Master Skill List entry (Book 1 p. 132). The chart
// abbreviations are canonicalized in the transcriptions (ERRATA.md I-9),
// so a character sheet can never carry two spellings of one skill.
func TestAwardedSkillsAreCanonical(t *testing.T) {
	for seed := uint64(1); seed <= 200; seed++ {
		c, err := chargen.Generate(chargen.Options{Seed: seed, Decider: chargen.DefaultPolicy{}})
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}

		for _, held := range c.Skills {
			if _, ok := skill.Lookup(held.Name); !ok {
				t.Errorf("seed %d: skill %q is not a Master Skill List entry", seed, held.Name)
			}
		}

		for _, record := range c.Careers {
			for _, name := range []string{record.Job, record.Hobby} {
				if name == "" {
					continue
				}

				if _, ok := skill.Lookup(name); !ok {
					t.Errorf("seed %d: %q is not a Master Skill List entry", seed, name)
				}
			}
		}
	}
}

// TestAmbiguousChartCellsResolveByChoice pins the two chart 04 table E
// cells that name more than one Master Skill List entry: the character
// chooses which, and the choice is recorded (ERRATA.md I-10, I-11). The
// seeds are the first that reach each cell under the default policy.
func TestAmbiguousChartCellsResolveByChoice(t *testing.T) {
	tests := []struct {
		name    string
		seed    uint64
		prompt  string
		options []string
		chosen  string
	}{
		{
			name:    "Spacecraft",
			seed:    165,
			prompt:  "Select the specific Spacecraft skill",
			options: []string{"Spacecraft ACS", "Spacecraft BCS"},
			chosen:  "Spacecraft ACS",
		},
		{
			name:    "Grav",
			seed:    333,
			prompt:  "Select the specific Grav skill",
			options: []string{"Driver: Grav", "Flyer: Grav", "Seafarer: Grav"},
			chosen:  "Driver: Grav",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := generate(t, chargen.Options{Seed: tt.seed})

			found := findChoice(c, tt.prompt)
			if found == nil {
				t.Fatalf("seed %d records no %q choice", tt.seed, tt.prompt)
			}

			if !slices.Equal(found.Options, tt.options) {
				t.Errorf("options = %v, want %v (Master Skill List order)", found.Options, tt.options)
			}

			if got := found.Options[found.Chosen]; got != tt.chosen {
				t.Errorf("chose %q, want %q (POLICY.md first-listed)", got, tt.chosen)
			}

			if !strings.Contains(found.Cite, "chart MS") {
				t.Errorf("cite = %q, want the Master Skill List citation", found.Cite)
			}

			// The resolved name, not the printed cell, reaches the sheet.
			if slices.ContainsFunc(c.Skills, func(s chargen.Skill) bool { return s.Name == tt.name }) {
				t.Errorf("character holds the ambiguous chart label %q instead of a list entry", tt.name)
			}
		})
	}
}

// findChoice returns the first choice event with the given prompt.
func findChoice(c chargen.Character, prompt string) *chargen.ChoiceEvent {
	for _, e := range c.Events {
		if e.Kind == chargen.EventChoice && e.Choice.Prompt == prompt {
			return e.Choice
		}
	}

	return nil
}
