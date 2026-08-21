package chargen_test

import (
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/chargen"
)

// scoutOptions forces the Scout career.
func scoutOptions(seed uint64, decider chargen.Decider) chargen.Options {
	return chargen.Options{Seed: seed, Career: "Scout", Decider: decider}
}

// TestScoutInvariants sweeps forced-Scout seeds: CC pool membership
// (chart 05: C1 C2 C3), Continue vs Int, Fame == Discoveries, wound-badge
// accounting, and the begin-failure fallback to Citizen (p. 65).
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

	if c.Fame != scout.Discoveries {
		t.Errorf("seed %d: fame %d != discoveries %d", seed, c.Fame, scout.Discoveries)
	}

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
type braveryDecider struct{ chargen.DefaultPolicy }

func (d braveryDecider) Choose(c chargen.Choice) int {
	if c.ID == chargen.ChooseRiskMod {
		for i, option := range c.Options {
			if option == "Bravery -9" {
				return i
			}
		}
	}

	return d.DefaultPolicy.Choose(c)
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
type scoutFirstDecider struct{ chargen.DefaultPolicy }

func (d scoutFirstDecider) Choose(c chargen.Choice) int {
	if c.ID == chargen.ChooseCareer {
		for i, option := range c.Options {
			if option == "Scout" {
				return i
			}
		}
	}

	return d.DefaultPolicy.Choose(c)
}

func (scoutFirstDecider) Kind() chargen.DeciderKind { return chargen.DeciderPlayer }
