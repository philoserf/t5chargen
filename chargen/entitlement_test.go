package chargen_test

import (
	"slices"
	"testing"

	"github.com/philoserf/t5chargen/chargen"
	"github.com/philoserf/t5chargen/lifestage"
)

// entitlementNamed finds one of a character's entitlements.
func entitlementNamed(c chargen.Character, name string) *chargen.EntitlementRecord {
	for i := range c.Entitlements {
		if c.Entitlements[i].Name == name {
			return &c.Entitlements[i]
		}
	}

	return nil
}

// TestPensionsBeginAtLifeStageNine pins p. 69: "Pensions are available for
// Citizens, Functionaries, Reservists, and Professors. A pension begins
// when a character reaches Life Stage 9 Retirement (= age 66 for Humans)."
//
// The two Armed Forces retirements are the exception chart M1 marks
// "Annual at Muster Out".
func TestPensionsBeginAtLifeStageNine(t *testing.T) {
	stages, err := lifestage.Load()
	if err != nil {
		t.Fatal(err)
	}

	retirement := stages.FirstYearOf(stages.MentalStage)
	pensions, retirements := 0, 0

	// The auto policy always picks Citizen, so an Armed Forces career has
	// to be asked for if the retirements are to appear at all.
	for _, career := range []string{"", "Soldier", "Spacer"} {
		for seed := uint64(1); seed <= 150; seed++ {
			c := generate(t, chargen.Options{Seed: seed, Career: career})
			pensions, retirements = countEntitlements(t, seed, c, retirement, pensions, retirements)
		}
	}

	if pensions == 0 || retirements == 0 {
		t.Errorf("saw %d pensions and %d retirements; both are needed", pensions, retirements)
	}
}

// countEntitlements checks each of a character's entitlements against the
// age it should begin at, and tallies the two kinds.
func countEntitlements(
	t *testing.T, seed uint64, c chargen.Character, retirement, pensions, retirements int,
) (int, int) {
	t.Helper()

	{
		for _, e := range c.Entitlements {
			military := e.ID == "enlisted_retirement" || e.ID == "officer_retirement"
			if military {
				retirements++

				if e.FromAge != c.Age {
					t.Errorf("seed %d: %s begins at %d, want muster out at %d", seed, e.Name, e.FromAge, c.Age)
				}

				continue
			}

			pensions++

			if e.FromAge != retirement {
				t.Errorf("seed %d: %s begins at %d, want Life Stage 9 at %d",
					seed, e.Name, e.FromAge, retirement)
			}
		}
	}

	return pensions, retirements
}

// TestTheFunctionarysPensionReplacesTheCitizens pins the one exception to
// duplicate entitlements: the Functionary's pension "replaces a Citizen's
// pension, if any" (p. 69), while everything else accumulates — "a
// character may receive duplicate Entitlements (for example, a Reserve and
// a Functionary pension)".
func TestTheFunctionarysPensionReplacesTheCitizens(t *testing.T) {
	// The character must have been both, or the replacement has nothing
	// to replace: seed 9 serves as a Citizen and then as a Functionary.
	c, err := chargen.Generate(chargen.Options{Seed: 9, Decider: functionaryPath{first: "Citizen"}})
	if err != nil {
		t.Fatal(err)
	}

	served := map[string]bool{}

	for _, record := range c.Careers {
		if record.Began {
			served[record.Career] = true
		}
	}

	if !served["Citizen"] || !served["Functionary"] {
		t.Fatalf("the pinned seed serves %v; both a Citizen and a Functionary career are needed", served)
	}

	if entitlementNamed(c, "Functionary's Pension") == nil {
		t.Error("a Functionary drew no pension")
	}

	if entitlementNamed(c, "Citizen's Pension") != nil {
		t.Error("a Functionary drew a Citizen's Pension as well; his replaces it")
	}
}

// TestArmedForcesRetirementNeedsFourTerms pins p. 69: "A Soldier, Spacer,
// or Marine who has served at least 4 terms receives an annual payment",
// "based on total combined terms served", and the Officer rate for one
// "who musters out as an Officer".
func TestArmedForcesRetirementNeedsFourTerms(t *testing.T) {
	short, long := 0, 0

	for _, service := range []string{"Soldier", "Spacer", "Marine"} {
		for seed := uint64(1); seed <= 80; seed++ {
			c := generate(t, chargen.Options{Seed: seed, Career: service})

			if serviceIsLong(t, service, seed, c) {
				long++
			} else {
				short++
			}
		}
	}

	if short == 0 || long == 0 {
		t.Errorf("saw %d short and %d long careers; both sides of the bar are needed", short, long)
	}
}

// TestDoublingsMultiplyTheOriginal pins p. 68: "Each doubling is of the
// original Pension: the first x2 doubles the Pension, the second x2
// triples the pension, the third x2 quadruples the original Pension."
// Three doublings give four times, not eight.
func TestDoublingsMultiplyTheOriginal(t *testing.T) {
	checked := 0

	// Pension x2 sits in chart 13's Money column, which needs a
	// Functionary, which needs a career change.
	for seed := uint64(1); seed <= 200; seed++ {
		c, err := chargen.Generate(chargen.Options{Seed: seed, Decider: functionaryPath{first: "Scholar"}})
		if err != nil {
			continue
		}

		doublings := 0

		for _, got := range c.Benefits {
			if got.Name == "Pension x2" {
				doublings += max(got.Count, 1)
			}
		}

		if assertDoubling(t, seed, c, doublings) {
			checked++
		}
	}

	if checked == 0 {
		t.Error("no pension was doubled; the rule went untested")
	}
}

// TestCashingOutIsFiveYears pins p. 69: "Any Entitlement can be cashed out
// for a lump sum equal to five years of payments".
func TestCashingOutIsFiveYears(t *testing.T) {
	checked := 0

	for seed := uint64(1); seed <= 200; seed++ {
		c := generate(t, chargen.Options{Seed: seed})
		for _, e := range c.Entitlements {
			checked++

			if want := e.AnnualCredits * 5; e.CashOut != want {
				t.Errorf("seed %d: %s cashes out at Cr%d, want Cr%d", seed, e.Name, e.CashOut, want)
			}
		}
	}

	if checked == 0 {
		t.Fatal("no entitlements in the sweep")
	}
}

// TestAutomaticsFollowTheirEligibility pins chart M1's Automatics column
// against the record: a character owns a personal weapon only with
// Fighter-1+, and TAS Life only on one of p. 67's four grounds.
func TestAutomaticsFollowTheirEligibility(t *testing.T) {
	weapons, tas := 0, 0

	// TAS Life needs an Award-Winning Scholar, a Scout with three
	// Discoveries, a Craftsman with three Perfect Masterpieces or a
	// holder of the Starburst — none of which a Citizen is, and the auto
	// policy makes everyone a Citizen.
	for _, career := range []string{"", "Scholar", "Scout", "Soldier"} {
		for seed := uint64(1); seed <= 100; seed++ {
			c := generate(t, chargen.Options{Seed: seed, Career: career})

			// A dead character does not muster out, so he owns none of
			// this (I-77).
			if c.Dead {
				continue
			}

			armed := slices.Contains(c.Automatics, "Personal Weapons")
			if fighter := skillLevelOf(c, "Fighter"); armed != (fighter >= 1) {
				t.Errorf("seed %d: Personal Weapons=%v at Fighter-%d", seed, armed, fighter)
			}

			if armed {
				weapons++
			}

			if slices.Contains(c.Automatics, "TAS Life Membership") {
				tas++
			}

			// "Fame — any": every character has some (chart F's 1D
			// fallback), so every one owns this.
			if !slices.Contains(c.Automatics, "Fame") {
				t.Errorf("seed %d: Fame %d but no Fame automatic", seed, c.Fame)
			}
		}
	}

	if weapons == 0 || tas == 0 {
		t.Errorf("saw %d armed and %d TAS members; both are needed", weapons, tas)
	}
}

// skillLevelOf reads a skill from a finished record.
func skillLevelOf(c chargen.Character, name string) int {
	for _, s := range c.Skills {
		if s.Name == name {
			return s.Level
		}
	}

	return 0
}

// TestTheRoguesPayoffIsMoney holds that what a Rogue amassed is counted:
// "Payoff= V x (1+CC-R+Mods)" (chart 10 p. 84).
func TestTheRoguesPayoffIsMoney(t *testing.T) {
	for seed := uint64(1); seed <= 120; seed++ {
		c := generate(t, chargen.Options{Seed: seed, Career: "Rogue"})
		if c.Dead {
			continue
		}

		payoff := 0
		for _, record := range c.Careers {
			payoff += record.SchemePayoff
		}

		if payoff > 0 && c.Credits < payoff {
			t.Fatalf("seed %d: Cr%d in payoffs but only Cr%d in credits", seed, payoff, c.Credits)
		}
	}
}

// serviceIsLong checks one character against p. 69's four-term bar and
// reports which side of it he fell.
func serviceIsLong(t *testing.T, service string, seed uint64, c chargen.Character) bool {
	t.Helper()

	terms := 0

	for _, record := range c.Careers {
		if record.Reserve {
			terms += len(record.Terms)
		}
	}

	drew := entitlementNamed(c, "Enlisted Retirement") != nil ||
		entitlementNamed(c, "Officer Retirement") != nil

	if terms >= 4 {
		if !drew {
			t.Errorf("%s seed %d: %d terms and no retirement", service, seed, terms)
		}

		return true
	}

	if drew {
		t.Errorf("%s seed %d: %d terms drew a retirement", service, seed, terms)
	}

	return false
}

// assertDoubling checks one character's pension against the doublings he
// took, and reports whether there was one to check.
func assertDoubling(t *testing.T, seed uint64, c chargen.Character, doublings int) bool {
	t.Helper()

	pension := entitlementNamed(c, "Citizen's Pension")
	if pension == nil {
		pension = entitlementNamed(c, "Functionary's Pension")
	}

	if pension == nil || doublings == 0 {
		return false
	}

	base := 5_000
	if pension.Name == "Functionary's Pension" {
		base = 15_000
	}

	if want := base * (doublings + 1); pension.AnnualCredits != want {
		t.Errorf("seed %d: %d doublings gave Cr%d a year, want Cr%d (%d times the original)",
			seed, doublings, pension.AnnualCredits, want, doublings+1)
	}

	return true
}
