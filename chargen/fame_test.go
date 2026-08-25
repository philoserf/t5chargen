package chargen_test

import (
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/chargen"
	"github.com/philoserf/t5chargen/fame"
)

// fameSources reads the itemized calculation out of a record's log.
func fameSources(c chargen.Character) map[string]int {
	sources := map[string]int{}

	for _, e := range c.Events {
		if e.Kind == chargen.EventConsequence && e.Consequence.Kind == chargen.ConsequenceFameComputed {
			for _, mod := range e.Consequence.Mods {
				sources[mod.Name] = mod.Value
			}
		}
	}

	return sources
}

// TestFameIsCalculatedNotAccumulated is the point of the chunk: Fame is
// "based on a variety of accomplishments" priced once over the finished
// record (chart F p. 91), not a counter three careers increment.
//
// Every character's Fame must equal what its own itemization stacks to,
// plus the Flux Event: the stacking limit governs "the sum of all Fame
// points received", and the Flux is added to the Fame those points stack
// to ("add Flux to Fame"), not stacked with them.
func TestFameIsCalculatedNotAccumulated(t *testing.T) {
	table, err := fame.Load()
	if err != nil {
		t.Fatal(err)
	}

	for seed := uint64(1); seed <= 300; seed++ {
		c := generate(t, chargen.Options{Seed: seed})

		var (
			points []int
			flux   int
		)

		for _, e := range c.Events {
			if e.Kind != chargen.EventConsequence ||
				e.Consequence.Kind != chargen.ConsequenceFameComputed {
				continue
			}

			for _, mod := range e.Consequence.Mods {
				if mod.Name == "Fame Flux Event" {
					flux = mod.Value

					continue
				}

				points = append(points, mod.Value)
			}
		}

		if want := max(table.Stack(points)+flux, 0); c.Fame != want {
			t.Fatalf("seed %d: Fame %d, but its sources stack to %d", seed, c.Fame, want)
		}
	}
}

// TestFameFluxCanLose holds the property the stacking limit must not
// swallow: the Fame Flux Event is a gamble, so a negative Flux has to cost
// the character Fame even when one eligibility dominates his total. Before
// the Flux was applied after stacking, "beyond 20, only the highest Fame
// applies" absorbed the loss and the gamble was free.
func TestFameFluxCanLose(t *testing.T) {
	drawn, cost := 0, 0

	for _, forced := range []string{"Scout", "Scholar", "Noble", "Rogue", "Merchant"} {
		for seed := uint64(1); seed <= 200; seed++ {
			c, open := generateIfOpen(t, chargen.Options{Seed: seed, Career: forced})
			if !open {
				continue // the Noble's Soc B+ prerequisite (I-28)
			}

			if lost, drew := fluxCost(t, c, forced, seed); drew {
				drawn++

				if lost {
					cost++
				}
			}
		}
	}

	if drawn == 0 || cost == 0 {
		t.Errorf("saw %d negative Flux draws, %d of which cost Fame; both must be positive", drawn, cost)
	}
}

// TestNegativeCareerFameDoesNotSubtract pins interpretation I-68: chart
// 03's Fame is a Flux-driven running level that can end below zero, and a
// career the character is not known for contributes nothing to chart F
// rather than subtracting from the rest. Seed 144 ends its Entertainer
// career at Fame -2; before the clamp it left the character at Fame 0,
// with the "If NO other eligibility, 1D" fallback suppressed by the
// negative entry.
func TestNegativeCareerFameDoesNotSubtract(t *testing.T) {
	for _, seed := range []uint64{144, 885, 1525} {
		c := generate(t, chargen.Options{Seed: seed, Career: "Entertainer"})

		record, ok := begunRecord(c, "Entertainer")
		if !ok || record.Fame >= 0 {
			t.Fatalf("seed %d: the fixture expects an Entertainer ending below Fame 0, got %d (begun %v)",
				seed, record.Fame, ok)
		}

		if got, ok := fameSources(c)["Entertainer Fame"]; ok {
			t.Errorf("seed %d: a career at Fame %d contributed %d", seed, record.Fame, got)
		}

		if c.Fame < 1 {
			t.Errorf("seed %d: Fame %d; the 1D no-eligibility fallback should have fired", seed, c.Fame)
		}
	}
}

// TestWoundBadgesEarnFame pins "Wound Badge WB x1" (chart F's Armed Forces
// block). A Wound Badge is awarded by the Risk failure, not the Reward
// success, so it is not in Medals; it is still priced. An enlisted
// character earns nothing for his, which is interpretation I-65.
func TestWoundBadgesEarnFame(t *testing.T) {
	officers := 0

	for _, service := range []string{"Soldier", "Spacer", "Marine"} {
		for seed := uint64(1); seed <= 120; seed++ {
			c := generate(t, chargen.Options{Seed: seed, Career: service})

			record, ok := begunRecord(c, service)
			if !ok || record.WoundBadges == 0 {
				continue
			}

			got, priced := fameSources(c)[service+" WB x1"]

			rank, isOfficer := record.Rank, record.Rank != "" && record.Rank[0] == 'O'
			if !isOfficer {
				if priced {
					t.Errorf("%s seed %d: rank %q earned %d Fame for Wound Badges", service, seed, rank, got)
				}

				continue
			}

			officers++

			if got != record.WoundBadges {
				t.Errorf("%s seed %d: %d Wound Badges contributed %d Fame, want %d",
					service, seed, record.WoundBadges, got, record.WoundBadges)
			}
		}
	}

	if officers == 0 {
		t.Error("no wounded officer in the sweep; the Wound Badge line went untested")
	}
}

// TestEveryCharacterHasFame pins "If NO other eligibility, 1D" (chart F):
// nobody finishes generation unknown to everyone.
func TestEveryCharacterHasFame(t *testing.T) {
	fallback := 0

	for seed := uint64(1); seed <= 300; seed++ {
		c := generate(t, chargen.Options{Seed: seed})
		if c.Fame < 1 {
			t.Fatalf("seed %d: Fame %d", seed, c.Fame)
		}

		if _, ok := fameSources(c)["no other eligibility"]; ok {
			fallback++
		}
	}

	if fallback == 0 {
		t.Error("no character fell back to 1D; the no-eligibility rule went untested")
	}
}

// TestFameByCareer pins one chart F eligibility per career against a
// fixture, which is the whole table read back from generated records.
func TestFameByCareer(t *testing.T) {
	for _, tt := range []struct {
		career, source string
		seed           uint64
		want           int
	}{
		{career: "Scout", seed: 26, source: "Scout Discoveries x4", want: 24},
		{career: "Scholar", seed: 23, source: "Scholar Publications", want: 8},
		{career: "Agent", seed: 1717, source: "Agent Commendations", want: 6},
		{career: "Merchant", seed: 17, source: "Merchant Rank", want: 3},
		{career: "Entertainer", seed: 572, source: "Entertainer Fame", want: 5},
		{career: "Noble", seed: 2978, source: "Noble Soc x1.5", want: 18},
		{career: "Rogue", seed: 39, source: "Rogue Failed Schemes x3", want: 36},
		{career: "Soldier", seed: 305, source: "Soldier Officer Rank", want: 5},
	} {
		t.Run(tt.career, func(t *testing.T) {
			c := generate(t, chargen.Options{Seed: tt.seed, Career: tt.career})
			if got := fameSources(c)[tt.source]; got != tt.want {
				t.Errorf("%s contributed %d, want %d", tt.source, got, tt.want)
			}
		})
	}
}

// TestCitizenHasNoIntrinsicFame pins "Citizen no intrinsic Fame" (chart
// F): a Citizen career contributes nothing, so a lifelong Citizen falls
// through to the 1D no-eligibility roll.
func TestCitizenHasNoIntrinsicFame(t *testing.T) {
	for seed := uint64(1); seed <= 60; seed++ {
		c := generate(t, chargen.Options{Seed: seed, Career: "Citizen"})

		for name := range fameSources(c) {
			if name != "no other eligibility" && name != "Fame Flux Event" {
				t.Fatalf("seed %d: a Citizen earned Fame from %q", seed, name)
			}
		}
	}
}

// TestEnlistedServiceEarnsNoFame pins the footnote "*Armed Forces Enlisted
// = no Fame" (chart F), read as flat rather than scoped to the rank line
// (interpretation I-65).
func TestEnlistedServiceEarnsNoFame(t *testing.T) {
	enlisted, officers := 0, 0

	for _, service := range []string{"Soldier", "Spacer", "Marine"} {
		for seed := uint64(1); seed <= 60; seed++ {
			c := generate(t, chargen.Options{Seed: seed, Career: service})

			record, ok := begunRecord(c, service)
			if !ok {
				continue
			}

			// The officer ladder is O-numbered; ratings and enlisted
			// ranks are not.
			officer := record.Rank != "" && record.Rank[0] == 'O'
			if officer {
				officers++
			} else {
				enlisted++
			}

			if got := contributedFame(c, service); got != officer {
				t.Errorf("%s seed %d: rank %q contributed=%v, want %v", service, seed, record.Rank, got, officer)
			}
		}
	}

	if enlisted == 0 || officers == 0 {
		t.Errorf("saw %d enlisted and %d officers; both are needed", enlisted, officers)
	}
}

// begunRecord returns the character's record for a career he entered.
func begunRecord(c chargen.Character, name string) (chargen.CareerRecord, bool) {
	for _, r := range c.Careers {
		if r.Career == name && r.Began {
			return r, true
		}
	}

	return chargen.CareerRecord{}, false
}

// contributedFame reports whether a career appears in chart F's itemized
// calculation.
func contributedFame(c chargen.Character, career string) bool {
	for name := range fameSources(c) {
		if strings.HasPrefix(name, career+" ") {
			return true
		}
	}

	return false
}

// fluxCost reports whether a character drew a negative Fame Flux and, if
// so, whether it cost him Fame. It fails the test where the loss was
// swallowed, which is the property under examination.
//
// Returns (lost, drew).
func fluxCost(t *testing.T, c chargen.Character, forced string, seed uint64) (bool, bool) {
	t.Helper()

	sources := fameSources(c)

	flux, ok := sources["Fame Flux Event"]
	if !ok || flux >= 0 {
		return false, false
	}

	base := 0

	for name, value := range sources {
		if name != "Fame Flux Event" {
			base += value
		}
	}

	if c.Fame < min(base, 20) {
		return true, true
	}

	t.Errorf("%s seed %d: Flux %d left Fame at %d with a base of %d", forced, seed, flux, c.Fame, base)

	return false, true
}
