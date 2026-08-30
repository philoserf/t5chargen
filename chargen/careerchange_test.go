package chargen_test

import (
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/chargen"
)

// changer accepts every career change it is offered and otherwise defers
// to the default policy, so a test can reach the path the auto policy
// declines (POLICY.md `change_career`).
type changer struct {
	playerKind

	accepted int
}

func (d *changer) Choose(c chargen.Choice) (int, error) {
	if c.ID == chargen.ChooseCareerChange {
		d.accepted++

		return 1, nil
	}

	return autoPolicy(c)
}

// TestChangingCareersRunsAnotherCareer is the point of the chunk: "A
// character may avoid the Continue roll ... by voluntarily ending his
// service in the current career and selecting a different career for which
// he is eligible" (p. 66).
func TestChangingCareersRunsAnotherCareer(t *testing.T) {
	multi, offers := 0, 0

	for seed := uint64(1); seed <= 200; seed++ {
		d := &changer{}

		c, err := chargen.Generate(chargen.Options{Seed: seed, Decider: d})
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}

		offers += d.accepted

		if len(c.Careers) > 1 {
			multi++
		}

		// "The decision to change careers is irrevocable": a career the
		// character left must not be re-entered in the very next
		// selection, which is what "a different career" forbids.
		for i := 1; i < len(c.Careers); i++ {
			if c.Careers[i].Career == c.Careers[i-1].Career {
				t.Errorf("seed %d: changed from %s straight back into it", seed, c.Careers[i].Career)
			}
		}
	}

	if multi == 0 || offers == 0 {
		t.Fatalf("%d offers accepted, %d characters with more than one career", offers, multi)
	}

	t.Logf("%d of 200 characters served more than one career", multi)
}

// TestTheDefaultPolicyNeverChangesCareers pins POLICY.md's declining rule,
// and with it the claim that every auto-generated character has exactly
// one career.
func TestTheDefaultPolicyNeverChangesCareers(t *testing.T) {
	for seed := uint64(1); seed <= 300; seed++ {
		c := generate(t, chargen.Options{Seed: seed})

		began := 0

		for _, record := range c.Careers {
			if record.Began {
				began++
			}
		}

		if began > 1 {
			t.Fatalf("seed %d: the auto policy served %d careers", seed, began)
		}
	}
}

// TestCareerChangeEligibility holds the three exclusions p. 66 prints: a
// Functionary or Noble cannot leave, nobody may change into Citizen, and
// the career being left is not among the alternatives.
func TestCareerChangeEligibility(t *testing.T) {
	for seed := uint64(1); seed <= 200; seed++ {
		c, err := chargen.Generate(chargen.Options{Seed: seed, Decider: &changer{}})
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}

		// Only a career that began can be left. A began:false record is
		// a failed To Begin, after which the character tries the
		// remaining careers (p. 65) — that is not a career change and
		// none of p. 66's exclusions apply to it.
		for i, record := range c.Careers {
			if i == 0 || !c.Careers[i-1].Began {
				continue
			}

			// "A Functionary or Noble cannot change to a new career."
			if left := c.Careers[i-1].Career; left == "Noble" || left == "Functionary" {
				t.Errorf("seed %d: a %s went on to another career", seed, left)
			}

			// "A Character may not change to the Citizen career."
			if record.Career == "Citizen" {
				t.Errorf("seed %d: changed into Citizen", seed)
			}
		}
	}
}

// TestChangingCareersCostsTheTermYears guards a bug the change path would
// otherwise have shipped with. A voluntary change takes no Continue throw
// (p. 66), and the Continue throw is what elapses the term's four years —
// so without an explicit advance the term would cost nothing at all.
func TestChangingCareersCostsTheTermYears(t *testing.T) {
	changed := 0

	for seed := uint64(1); seed <= 200; seed++ {
		d := &changer{}

		c, err := chargen.Generate(chargen.Options{Seed: seed, Decider: d})
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}

		if d.accepted == 0 {
			continue
		}

		changed++

		terms, elapsed := termsAndFourYearElapses(c)
		if elapsed != terms {
			t.Fatalf("seed %d: %d terms served, %d elapsed four years", seed, terms, elapsed)
		}
	}

	if changed == 0 {
		t.Fatal("no character changed careers; the rule went untested")
	}
}

// termsAndFourYearElapses counts the terms served across every career and
// the four-year advances the log records for them.
func termsAndFourYearElapses(c chargen.Character) (int, int) {
	terms, elapsed := 0, 0

	for _, record := range c.Careers {
		terms += len(record.Terms)
	}

	for _, e := range c.Events {
		if e.Kind == chargen.EventConsequence &&
			e.Consequence.Kind == chargen.ConsequenceYearsElapsed &&
			e.Consequence.Value == chargen.TermYears {
			elapsed++
		}
	}

	return terms, elapsed
}

// TestEveryCareerRecordsWhenItEnded pins CareerRecord.EndAge, which muster
// out reads: retirement and the Reserve Pension both run from when the
// service stopped (p. 69).
func TestEveryCareerRecordsWhenItEnded(t *testing.T) {
	for seed := uint64(1); seed <= 200; seed++ {
		c, err := chargen.Generate(chargen.Options{Seed: seed, Decider: &changer{}})
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}

		previous := 0

		for _, record := range c.Careers {
			if record.EndAge == 0 || record.EndAge > c.Age {
				t.Fatalf("seed %d: %s ended at %d, character is %d", seed, record.Career, record.EndAge, c.Age)
			}

			// Careers run in order, so each ends no earlier than the last.
			if record.EndAge < previous {
				t.Fatalf("seed %d: %s ended at %d, before the previous career's %d",
					seed, record.Career, record.EndAge, previous)
			}

			previous = record.EndAge
		}
	}
}

// TestLeavingTheArmedForcesEntersTheReserves pins p. 67: "A character who
// leaves a military, naval, or marine career is automatically in the
// Reserves ... maintains his or her last held rank as a Reserve Rank".
func TestLeavingTheArmedForcesEntersTheReserves(t *testing.T) {
	seen := 0

	for _, career := range []string{"Soldier", "Spacer", "Marine"} {
		for seed := uint64(1); seed <= 40; seed++ {
			c, err := chargen.Generate(chargen.Options{
				Seed: seed, Career: career, Decider: chargen.DefaultPolicy{},
			})
			if err != nil {
				t.Fatalf("%s seed %d: %v", career, seed, err)
			}

			for _, record := range c.Careers {
				if !record.Began || c.Dead {
					continue
				}

				if !record.Reserve {
					t.Fatalf("%s seed %d: left the service without entering the Reserves", career, seed)
				}

				seen++
			}
		}
	}

	if seen == 0 {
		t.Fatal("no Armed Forces career completed; the Reserve rule went untested")
	}

	assertNoOtherCareerEnrols(t)
}

// assertNoOtherCareerEnrols holds the scope of p. 67: the Reserves take
// leavers from military, naval, and marine careers, and no others.
func assertNoOtherCareerEnrols(t *testing.T) {
	t.Helper()

	for _, career := range []string{"Citizen", "Scholar", "Merchant", "Rogue", "Noble"} {
		for seed := uint64(1); seed <= 20; seed++ {
			c, open := generateIfOpen(t, chargen.Options{Seed: seed, Career: career})
			if !open {
				continue // the Noble's Soc B+ prerequisite (I-28)
			}

			for _, record := range c.Careers {
				if record.Reserve {
					t.Fatalf("%s seed %d: a %s entered the Reserves", career, seed, record.Career)
				}
			}
		}
	}
}

// TestTheReserveRankIsNamed pins what p. 67 preserves. The rank is what
// the Reserves keep — "maintains his or her last held rank as a Reserve
// Rank ('I'm a Marine Reserve Sergeant')" — so the record must carry a
// name a reader recognises, not the internal rank id.
func TestTheReserveRankIsNamed(t *testing.T) {
	named := 0

	for _, career := range []string{"Soldier", "Spacer", "Marine"} {
		for seed := uint64(1); seed <= 40; seed++ {
			c, err := chargen.Generate(chargen.Options{
				Seed: seed, Career: career, Decider: chargen.DefaultPolicy{},
			})
			if err != nil {
				t.Fatalf("%s seed %d: %v", career, seed, err)
			}

			for _, e := range c.Events {
				if e.Kind != chargen.EventConsequence || e.Consequence.Kind != chargen.ConsequenceReserve {
					continue
				}

				rank := e.Consequence.Skill
				if rank == "" {
					continue
				}

				named++

				// The sheet's form is "Lt Colonel O5": a bare id carries
				// no rank the reader can read.
				if !strings.Contains(rank, " ") {
					t.Errorf("%s seed %d: Reserve Rank recorded as %q, want a titled rank", career, seed, rank)
				}
			}
		}
	}

	if named == 0 {
		t.Fatal("no ranked Armed Forces career completed; the Reserve Rank went untested")
	}
}
