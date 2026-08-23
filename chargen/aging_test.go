package chargen_test

import (
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/chargen"
	"github.com/philoserf/t5chargen/lifestage"
)

// agingCounts summarizes what a record's log says aging did.
type agingCounts struct{ throws, effects, resets int }

func countAging(c chargen.Character) agingCounts {
	var n agingCounts

	for _, e := range c.Events {
		switch {
		case e.Kind == chargen.EventThrow && e.Throw != nil &&
			strings.HasPrefix(e.Throw.Cite, "Book 1 p. 89 chart A"):
			n.throws++
		case e.Kind == chargen.EventConsequence && e.Consequence.Kind == chargen.ConsequenceAgingEffect:
			n.effects++
		case e.Kind == chargen.EventConsequence && e.Consequence.Kind == chargen.ConsequenceCharacteristicReset:
			n.resets++
		}
	}

	return n
}

// TestAgingCheckSchedule pins when the Crisis is rolled: "Once Aging
// begins, it occurs every four years on the character's birthday", from
// age 34 for the three Physical characteristics and from 66 with
// Intelligence added (chart A p. 89).
//
// The count is derived from the final age rather than pinned to fixtures,
// so it holds across every seed and both aging bands.
func TestAgingCheckSchedule(t *testing.T) {
	table, err := lifestage.Load()
	if err != nil {
		t.Fatal(err)
	}

	physical := table.FirstYearOf(table.PhysicalStage)
	mental := table.FirstYearOf(table.MentalStage)

	sawMental := false

	for seed := uint64(1); seed <= 400; seed++ {
		c := generate(t, chargen.Options{Seed: seed})

		want := 0

		for age := physical; age <= c.Age; age += table.CheckIntervalYears {
			want += len(table.PhysicalCharacteristics)
			if age >= mental {
				want += len(table.MentalCharacteristics)
				sawMental = true
			}
		}

		got := countAging(c).throws

		// A character killed by aging stops mid-schedule; a living one
		// must have rolled every check the schedule calls for.
		if (!c.Dead && got != want) || got > want {
			t.Fatalf("seed %d (age %d, dead=%v): %d aging throws, want %d", seed, c.Age, c.Dead, got, want)
		}
	}

	if !sawMental {
		t.Error("sweep never reached Mental Aging at 66; the second band went untested")
	}
}

// TestAgingNeverLeavesAZero pins the floor: "If one Characteristic is
// reduced to 0, it is reset to 1" (chart A p. 89). The reset is
// unconditional — the count of zeroes decides the illness, not whether the
// reset happens — so no living character may hold a zero.
func TestAgingNeverLeavesAZero(t *testing.T) {
	for seed := uint64(1); seed <= 400; seed++ {
		c := generate(t, chargen.Options{Seed: seed})

		for _, ch := range []struct {
			name  string
			value int
		}{
			{name: "Str", value: c.Characteristics.Str},
			{name: "Dex", value: c.Characteristics.Dex},
			{name: "End", value: c.Characteristics.End},
			{name: "Int", value: c.Characteristics.Int},
		} {
			// Injury can reduce a controlling characteristic to zero or
			// less, and that kills (p. 65); aging never leaves one.
			if ch.value <= 0 && !c.Dead {
				t.Fatalf("seed %d: living character has %s = %d", seed, ch.name, ch.value)
			}
		}
	}
}

// TestEveryAgingEffectIsMinusOne pins "the characteristic is reduced -1",
// and that every reset answers an effect that reached zero.
func TestEveryAgingEffectIsMinusOne(t *testing.T) {
	zeroes := 0

	for seed := uint64(1); seed <= 400; seed++ {
		c := generate(t, chargen.Options{Seed: seed})

		zeroes += assertAgingDeltas(t, seed, c)

		if got := countAging(c); got.resets != countZeroEffects(c) {
			t.Fatalf("seed %d: %d resets for %d effects reaching zero", seed, got.resets, countZeroEffects(c))
		}
	}

	if zeroes == 0 {
		t.Error("sweep never drove a characteristic to zero; the reset rule went untested")
	}
}

func countZeroEffects(c chargen.Character) int {
	n := 0

	for _, e := range c.Events {
		if e.Kind == chargen.EventConsequence &&
			e.Consequence.Kind == chargen.ConsequenceAgingEffect && e.Consequence.Value == 0 {
			n++
		}
	}

	return n
}

// TestTheSecondExtremelyMajorIllnessKills pins the death rule: "The second
// time three characteristics are reduced to 0, the character dies" (chart
// A p. 89), and that death then ends the lifepath — without which aging
// bounds nothing, since generation would go on aging a corpse.
func TestTheSecondExtremelyMajorIllnessKills(t *testing.T) {
	killed := 0

	for seed := uint64(1); seed <= 400; seed++ {
		c := generate(t, chargen.Options{Seed: seed})

		if c.ExtremelyMajorIllnesses >= 2 {
			killed++

			if !c.Dead {
				t.Fatalf("seed %d: %d extremely major illnesses, still alive", seed, c.ExtremelyMajorIllnesses)
			}
		}

		if c.ExtremelyMajorIllnesses > 2 {
			t.Fatalf("seed %d: %d extremely major illnesses; the second is fatal", seed, c.ExtremelyMajorIllnesses)
		}

		assertNothingFollowsDeath(t, seed, c)
	}

	if killed == 0 {
		t.Error("sweep never killed anyone by aging; the death rule went untested")
	}
}

// TestLifeStageIsDerived holds the stored value against the table, as
// TestGenerateDerivedUPP does for the UPP.
func TestLifeStageIsDerived(t *testing.T) {
	table, err := lifestage.Load()
	if err != nil {
		t.Fatal(err)
	}

	for seed := uint64(1); seed <= 200; seed++ {
		c := generate(t, chargen.Options{Seed: seed})
		if want := table.Of(c.Age); c.LifeStage != want {
			t.Fatalf("seed %d: life stage %d for age %d, want %d", seed, c.LifeStage, c.Age, want)
		}
	}
}

// assertNothingFollowsDeath holds the lifepath's end. Without it the
// engine ran on: aging skipped a corpse's checks, but the career loop kept
// serving terms, and sweeps produced characters who died in their nineties
// and were still accumulating skills at 401.
func assertNothingFollowsDeath(t *testing.T, seed uint64, c chargen.Character) {
	t.Helper()

	dead := false

	for _, e := range c.Events {
		if e.Kind != chargen.EventConsequence {
			continue
		}

		if dead {
			//nolint:exhaustive // Deliberately partitioned: only what must not happen after death.
			switch e.Consequence.Kind {
			case chargen.ConsequenceYearsElapsed:
				t.Fatalf("seed %d: %d years elapsed after dying (age %d)", seed, e.Consequence.Value, c.Age)
			case chargen.ConsequenceAgingEffect:
				t.Fatalf("seed %d: aged after dying", seed)
			default:
			}
		}

		if e.Consequence.Kind == chargen.ConsequenceDead {
			dead = true
		}
	}
}

// assertAgingDeltas checks every aging consequence in a record and reports
// how many effects drove a characteristic to zero.
func assertAgingDeltas(t *testing.T, seed uint64, c chargen.Character) int {
	t.Helper()

	zeroes := 0

	for _, e := range c.Events {
		if e.Kind != chargen.EventConsequence {
			continue
		}

		//nolint:exhaustive // Deliberately partitioned: only the aging kinds matter here.
		switch e.Consequence.Kind {
		case chargen.ConsequenceAgingEffect:
			if e.Consequence.Delta != -1 {
				t.Fatalf("seed %d: aging effect of %+d", seed, e.Consequence.Delta)
			}

			if e.Consequence.Value == 0 {
				zeroes++
			}
		case chargen.ConsequenceCharacteristicReset:
			if e.Consequence.Value != 1 {
				t.Fatalf("seed %d: reset to %d, want 1", seed, e.Consequence.Value)
			}
		default:
		}
	}

	return zeroes
}
