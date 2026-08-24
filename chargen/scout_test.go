package chargen_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/chargen"
)

// scoutOptions forces the Scout career.
func scoutOptions(seed uint64, decider chargen.Decider) chargen.Options {
	return chargen.Options{Seed: seed, Career: "Scout", Decider: decider}
}

// TestScoutInvariants sweeps forced-Scout seeds: CC pool membership
// (chart 05: C1 C2 C3), Continue vs Int, Fame priced from Discoveries
// (chart F p. 91), wound-badge accounting, and the begin-failure fallback
// to Citizen (p. 65).
func TestScoutInvariants(t *testing.T) {
	sawBeginFailure := false

	for seed := range uint64(60) {
		c := generate(t, scoutOptions(seed, nil))
		checkScoutRecord(t, seed, c, &sawBeginFailure)
	}

	if !sawBeginFailure {
		t.Error("no seed in the sweep exercised a failed Scout To Begin; widen the sweep")
	}
}

// checkScoutRecord asserts one forced-Scout character.
func checkScoutRecord(t *testing.T, seed uint64, c chargen.Character, sawBeginFailure *bool) {
	t.Helper()

	if len(c.Careers) == 0 || c.Careers[0].Career != "Scout" {
		t.Fatalf("seed %d: careers = %+v", seed, c.Careers)
	}

	scout := c.Careers[0]
	if !scout.Began {
		*sawBeginFailure = true

		// "this career may not be used" (p. 65): with --career forcing
		// only Scout, no career remains.
		if len(scout.Terms) != 0 || len(c.Careers) != 1 {
			t.Errorf("seed %d: unbegun scout has terms or extra careers: %+v", seed, c.Careers)
		}

		return
	}

	valid := map[string]bool{"Str": true, "Dex": true, "End": true}
	for _, term := range scout.Terms {
		if !valid[term.ControllingCharacteristic] {
			t.Errorf("seed %d: term %d CC %q outside chart 05's C1 C2 C3", seed, term.Term, term.ControllingCharacteristic)
		}
	}

	// "Scout Discoveries x4" (chart F p. 91). The contribution is checked
	// rather than the total, which also carries the Fame Flux Event and
	// the stacking limit — both tested where they belong.
	checkFameSource(t, seed, c, "Scout Discoveries x4", scout.Discoveries*4)

	checkScoutEvents(t, seed, c)
}

// checkScoutEvents asserts event accounting: wound badges match their
// events and a Continue-vs-Int throw appears for multi-term careers.
func checkScoutEvents(t *testing.T, seed uint64, c chargen.Character) {
	t.Helper()

	badges := 0
	sawContinueInt := false

	for _, event := range c.Events {
		if event.Kind == chargen.EventConsequence && event.Consequence.Kind == chargen.ConsequenceWoundBadge {
			badges++
		}

		if event.Kind == chargen.EventThrow && strings.Contains(event.Throw.Cite, "Continue Int") {
			sawContinueInt = true
		}
	}

	if badges != c.WoundBadges {
		t.Errorf("seed %d: wound badge events %d != recorded %d", seed, badges, c.WoundBadges)
	}

	if !c.Dead && !c.Disabled && !sawContinueInt {
		t.Errorf("seed %d: no Continue-vs-Int throw in a normally-ended scout career", seed)
	}
}

// braveryDecider is the default policy except it always selects Bravery -9
// (guaranteeing a 4+ reduction on any Risk failure) and Explorer Duty.
type braveryDecider struct{}

func (braveryDecider) Choose(c chargen.Choice) (int, error) {
	if c.ID == chargen.ChooseRiskMod {
		for i, option := range c.Options {
			if option == "Bravery -9" {
				return i, nil
			}
		}
	}

	return autoPolicy(c)
}

func (braveryDecider) Kind() chargen.DeciderKind { return chargen.DeciderPlayer }

// TestScoutInjuryOutcomes drives Bravery -9 scouts until the sweep has
// produced both a disabled scout ("If CC is reduced by 4 or more, then he
// is disabled. Muster Out at Term end", chart 05) and a dead one ("the
// Character is dead", p. 65), asserting each record's shape.
func TestScoutInjuryOutcomes(t *testing.T) {
	sawDisabled, sawDead := false, false

	for seed := range uint64(80) {
		c := generate(t, scoutOptions(seed, braveryDecider{}))
		checkInjuryOutcome(t, seed, c, &sawDisabled, &sawDead)
	}

	if !sawDisabled || !sawDead {
		t.Errorf("sweep missed outcomes: disabled=%v dead=%v; widen the sweep", sawDisabled, sawDead)
	}
}

// checkInjuryOutcome asserts one Bravery -9 scout's record shape.
func checkInjuryOutcome(t *testing.T, seed uint64, c chargen.Character, sawDisabled, sawDead *bool) {
	t.Helper()

	if len(c.Careers) == 0 || !c.Careers[0].Began || len(c.Careers[0].Terms) == 0 {
		return
	}

	last := c.Careers[0].Terms[len(c.Careers[0].Terms)-1]

	if c.Dead {
		*sawDead = true

		// Death ends the term at the injury: no Continue, no success.
		if last.Continued || last.Success {
			t.Errorf("seed %d: dead scout's last term = %+v", seed, last)
		}
	}

	if c.Disabled && !c.Dead {
		*sawDisabled = true

		// Disabled completes the term but never rolls Continue.
		if last.Continued {
			t.Errorf("seed %d: disabled scout continued: %+v", seed, last)
		}
	}
}

// TestScoutBeginFallback verifies an unforced run whose first-choice
// career fails To Begin falls back to the remaining careers (p. 65). The
// policy picks Citizen first (automatic begin), so the fallback is driven
// with a decider that prefers Scout.
func TestScoutBeginFallback(t *testing.T) {
	sawFallback := false

	for seed := range uint64(60) {
		c := generate(t, chargen.Options{Seed: seed, Decider: scoutFirstDecider{}})

		if len(c.Careers) == 2 && !c.Careers[0].Began {
			sawFallback = true

			if c.Careers[0].Career != "Scout" || c.Careers[1].Career != "Citizen" || !c.Careers[1].Began {
				t.Errorf("seed %d: fallback shape = %+v", seed, c.Careers)
			}
		}
	}

	if !sawFallback {
		t.Error("no seed exercised the begin-failure fallback; widen the sweep")
	}
}

// scoutFirstDecider prefers the Scout career, otherwise the default policy.
type scoutFirstDecider struct{}

func (scoutFirstDecider) Choose(c chargen.Choice) (int, error) {
	if c.ID == chargen.ChooseCareer {
		for i, option := range c.Options {
			if option == "Scout" {
				return i, nil
			}
		}
	}

	return autoPolicy(c)
}

func (scoutFirstDecider) Kind() chargen.DeciderKind { return chargen.DeciderPlayer }

// TestScoutSanityModifier pins chart 05's "Because of the long-term
// isolation that a Scout must endure, reduce San= -1 for each TWO Terms
// served" (p. 79). Swept rather than pinned to one fixture because the
// interesting cases are the boundaries: an odd term count owes the same
// as the even one below it, and one term owes nothing.
func TestScoutSanityModifier(t *testing.T) {
	seen := map[int]bool{}

	for seed := uint64(1); seed <= 300; seed++ {
		c := generate(t, chargen.Options{Seed: seed, Career: "Scout"})

		for _, record := range c.Careers {
			if record.Career != "Scout" {
				t.Fatalf("seed %d: forced Scout produced %q", seed, record.Career)
			}

			terms := len(record.Terms)
			seen[terms%2] = true

			if want := -(terms / 2); record.SanityMod != want {
				t.Errorf("seed %d: %d terms gave San %+d, want %+d",
					seed, terms, record.SanityMod, want)
			}
		}
	}

	// A sweep that only ever saw even term counts would pass while
	// leaving the rounding rule — three terms cost what two cost —
	// untested.
	if !seen[0] || !seen[1] {
		t.Errorf("sweep saw only %v term parities; both are needed to test the rounding", seen)
	}
}

// TestOnlyTheScoutChargesSanity holds the scope of the rule: chart 05 is
// the only career page that prints one, so no other career may record a
// reduction (p. 79). Swept over career.Available rather than a written-out
// list, so a career added later is covered without editing this test.
func TestOnlyTheScoutChargesSanity(t *testing.T) {
	for _, name := range career.Available() {
		if name == "Scout" {
			continue
		}

		t.Run(name, func(t *testing.T) {
			for seed := uint64(1); seed <= 40; seed++ {
				c, err := chargen.Generate(chargen.Options{
					Seed: seed, Career: name, Decider: chargen.DefaultPolicy{},
				})

				// Craftsman and Functionary cannot open a lifepath
				// (p. 75, p. 87), so forcing one is refused rather than
				// generating a character. Nothing to check, and nothing
				// wrong.
				if errors.Is(err, chargen.ErrCareerUnavailable) {
					t.Skipf("%s cannot be a first career", name)
				}

				if err != nil {
					t.Fatalf("seed %d: %v", seed, err)
				}

				for _, record := range c.Careers {
					if record.SanityMod != 0 {
						t.Fatalf("seed %d: %s recorded San %+d", seed, record.Career, record.SanityMod)
					}
				}
			}
		})
	}
}

// checkFameSource asserts one line of chart F's itemized calculation.
// A source worth nothing is not recorded, so zero means absent.
func checkFameSource(t *testing.T, seed uint64, c chargen.Character, source string, want int) {
	t.Helper()

	for _, e := range c.Events {
		if e.Kind != chargen.EventConsequence || e.Consequence.Kind != chargen.ConsequenceFameComputed {
			continue
		}

		for _, mod := range e.Consequence.Mods {
			if mod.Name == source {
				if mod.Value != want {
					t.Errorf("seed %d: %s contributed %d, want %d", seed, source, mod.Value, want)
				}

				return
			}
		}
	}

	if want != 0 {
		t.Errorf("seed %d: %s contributed nothing, want %d", seed, source, want)
	}
}

// TestEveryScoutDutyHasSkillEligibility holds together the two halves of
// chart 05 table B that live in different files: the duty labels are a Go
// literal and their eligibility is a JSON map. A key that does not match
// returns zero, which termSkills reads as "unspecified" and replaces with
// the definition's SkillsPerTerm — so a typo would award 8 skills where the
// chart says 4 and leave no trace.
func TestEveryScoutDutyHasSkillEligibility(t *testing.T) {
	t.Parallel()

	def, err := career.ByName("Scout")
	if err != nil {
		t.Fatal(err)
	}

	if len(def.SkillEligibility) != len(chargen.ScoutDuties) {
		t.Errorf("chart 05 lists %d duties and %d eligibilities",
			len(chargen.ScoutDuties), len(def.SkillEligibility))
	}

	for _, duty := range chargen.ScoutDuties {
		rolls, ok := def.SkillEligibility[duty]
		if !ok {
			t.Errorf("%q has no skill eligibility in scout.json", duty)

			continue
		}

		if rolls < 1 {
			t.Errorf("%q allows %d skill rolls", duty, rolls)
		}
	}
}
