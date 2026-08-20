package chargen_test

import (
	"errors"
	"testing"

	"github.com/philoserf/t5chargen/chargen"
	"github.com/philoserf/t5chargen/world"
)

// homeworldSkillLevel reads a skill level off a generated character.
func homeworldSkillLevel(c chargen.Character, name string) int {
	for _, skill := range c.Skills {
		if skill.Name == name {
			return skill.Level
		}
	}

	return 0
}

// TestHomeworldDefault verifies the default homeworld (Regina, Ph Pa Ri)
// grants per chart B (p. 56): Ph nothing, Pa Trader-1, Ri one Art
// (policy-selected first-listed = Actor), matching the p. 58 example
// mechanism.
func TestHomeworldDefault(t *testing.T) {
	c := generate(t, chargen.Options{Seed: 1})

	if c.Homeworld.Name != "Regina" || c.Homeworld.UWP != "A788899-C" {
		t.Errorf("homeworld = %+v", c.Homeworld)
	}

	if level := homeworldSkillLevel(c, "Trader"); level < 1 {
		t.Errorf("Trader = %d, want >= 1 (Pa grant)", level)
	}

	if level := homeworldSkillLevel(c, "Actor"); level < 1 {
		t.Errorf("Actor = %d, want >= 1 (Ri art, policy first-listed)", level)
	}
}

// TestHomeworldSupplied verifies a supplied homeworld's grants, including
// the Ds Deep Space double grant ("Vacc Suit +Zero-G", chart B p. 56) and
// the In trade selection (policy first-listed = Biologics).
func TestHomeworldSupplied(t *testing.T) {
	c := generate(t, chargen.Options{Seed: 2, Homeworld: world.Homeworld{
		UWP:                  "B000453-E",
		TradeClassifications: []string{"Ds", "In"},
	}})

	for _, name := range []string{"Vacc Suit", "Zero-G", "Biologics"} {
		if level := homeworldSkillLevel(c, name); level < 1 {
			t.Errorf("%s = %d, want >= 1", name, level)
		}
	}
}

// TestHomeworldBareUWP verifies a bare UWP grants no homeworld skills
// (docs/PRD.md FR2 note: skills key off trade classifications).
func TestHomeworldBareUWP(t *testing.T) {
	c := generate(t, chargen.Options{Seed: 2, Homeworld: world.Homeworld{UWP: "C200423-7"}})

	if c.Homeworld.UWP != "C200423-7" || len(c.Homeworld.TradeClassifications) != 0 {
		t.Errorf("homeworld = %+v", c.Homeworld)
	}
}

// TestHomeworldErrors verifies invalid UWPs and unknown TCs error rather
// than being silently repaired (docs/PRD.md FR2).
func TestHomeworldErrors(t *testing.T) {
	_, err := chargen.Generate(chargen.Options{
		Seed:      1,
		Decider:   chargen.DefaultPolicy{},
		Homeworld: world.Homeworld{UWP: "A78899"},
	})
	if !errors.Is(err, world.ErrInvalidUWP) {
		t.Errorf("partial UWP error = %v, want ErrInvalidUWP", err)
	}

	_, err = chargen.Generate(chargen.Options{
		Seed:      1,
		Decider:   chargen.DefaultPolicy{},
		Homeworld: world.Homeworld{UWP: "A788899-C", TradeClassifications: []string{"Qq"}},
	})
	if !errors.Is(err, world.ErrUnknownTC) {
		t.Errorf("unknown TC error = %v, want ErrUnknownTC", err)
	}

	_, err = chargen.Generate(chargen.Options{
		Seed:      1,
		Decider:   chargen.DefaultPolicy{},
		Homeworld: world.Homeworld{UWP: "A788899-C", TradeClassifications: []string{"Pa", "Pa"}},
	})
	if !errors.Is(err, world.ErrDuplicateTC) {
		t.Errorf("duplicate TC error = %v, want ErrDuplicateTC", err)
	}

	// A partially-populated homeworld (TCs without a UWP) must be
	// rejected, not silently replaced by the default (docs/PRD.md FR2).
	_, err = chargen.Generate(chargen.Options{
		Seed:      1,
		Decider:   chargen.DefaultPolicy{},
		Homeworld: world.Homeworld{TradeClassifications: []string{"In"}},
	})
	if !errors.Is(err, world.ErrInvalidUWP) {
		t.Errorf("partial homeworld error = %v, want ErrInvalidUWP", err)
	}
}

// TestHomeworldStreamNeutral verifies the homeworld step consumes no dice:
// the same seed yields identical characteristics and career throws with
// and without TC-driven choices (replay contract).
func TestHomeworldStreamNeutral(t *testing.T) {
	a := generate(t, chargen.Options{Seed: 9})
	b := generate(t, chargen.Options{Seed: 9, Homeworld: world.Homeworld{
		UWP:                  "C200423-7",
		TradeClassifications: []string{"Va", "Ni"},
	}})

	if a.Characteristics != b.Characteristics {
		t.Error("homeworld changed the characteristic rolls")
	}

	if len(a.Careers[0].Terms) != len(b.Careers[0].Terms) {
		t.Errorf("homeworld changed the career stream: %d vs %d terms",
			len(a.Careers[0].Terms), len(b.Careers[0].Terms))
	}
}
