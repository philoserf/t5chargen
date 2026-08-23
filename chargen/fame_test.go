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
// Every character's Fame must equal what its own itemization stacks to.
func TestFameIsCalculatedNotAccumulated(t *testing.T) {
	table, err := fame.Load()
	if err != nil {
		t.Fatal(err)
	}

	for seed := uint64(1); seed <= 300; seed++ {
		c := generate(t, chargen.Options{Seed: seed})

		var points []int

		for _, e := range c.Events {
			if e.Kind != chargen.EventConsequence ||
				e.Consequence.Kind != chargen.ConsequenceFameComputed {
				continue
			}

			for _, mod := range e.Consequence.Mods {
				points = append(points, mod.Value)
			}
		}

		if want := table.Stack(points); c.Fame != want {
			t.Fatalf("seed %d: Fame %d, but its sources stack to %d", seed, c.Fame, want)
		}
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
