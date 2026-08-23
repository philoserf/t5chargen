package chargen

// Aging (Book 1 p. 89 chart A, The Aging Process).
//
// "Aging affects a character's physical and mental characteristics,
// ultimately reducing them to zero and bringing on inevitable death.
// Characters are immune to Aging for roughly the first half of their
// lives. Once Aging begins, it occurs every four years on the character's
// birthday."
//
// "Human Physical Aging affects Strength, Dexterity, and Endurance. It
// begins at age 34 (the beginning of Life Stage 5 Peak) and is resolved as
// an Aging Check. Human Mental Aging affects Intelligence. It begins at
// age 66 (the beginning of Life Stage 9- Retirement)."
//
// Deferred: Clone Aging, which accelerates the pattern by one stage for
// Relicts, Guests, and Meds — outside the human core lifepath (docs/PRD.md
// non-goals).

import (
	"fmt"

	"github.com/philoserf/t5chargen/dice"
	"github.com/philoserf/t5chargen/lifestage"
)

// agingDice is the Aging Check's roll: "2D < Life Stage".
const agingDice = 2

// The zero cascade's thresholds: "If two Characteristics are reduced to 0,
// the character suffers a major illness ... If three Characteristics are
// reduced to 0, the character suffers an extremely major illness".
const (
	majorIllnessZeroes          = 2
	extremelyMajorIllnessZeroes = 3

	// "The second time three characteristics are reduced to 0, the
	// character dies".
	fatalExtremelyMajorIllnesses = 2
)

// ageEffects resolves every Aging Check the character crossed in moving
// from one age to another. The birthdays are absolute — 34, 38, 42, and so
// on — because "it occurs every four years on the character's birthday"
// and a failed career entry costs a single year (p. 65), which would
// otherwise knock a character permanently off the four-year grid.
func (c *Character) ageEffects(from, to int, roller *dice.Roller, log *Log) error {
	table, err := lifestage.Load()
	if err != nil {
		return fmt.Errorf("aging: %w", err)
	}

	first := table.FirstYearOf(table.PhysicalStage)

	for age := from + 1; age <= to; age++ {
		if c.Dead || age < first || (age-first)%table.CheckIntervalYears != 0 {
			continue
		}

		c.agingPass(table, age, roller, log)
	}

	return nil
}

// agingPass resolves one birthday's checks: "The Crisis is rolled for each
// applicable Characteristic".
func (c *Character) agingPass(table *lifestage.Table, age int, roller *dice.Roller, log *Log) {
	stage := table.Of(age)

	names := table.PhysicalCharacteristics
	if stage >= table.MentalStage {
		names = append(append([]string{}, names...), table.MentalCharacteristics...)
	}

	zeroed, last := 0, 0

	for _, name := range names {
		// "2D < Life Stage. Success inflicts the effects of age on the
		// character. (A character wants to FAIL this action)."
		throw := roller.Under(agingDice, stage)
		seq := log.Throw(throw, nil, fmt.Sprintf("%s (Aging Check vs Life Stage %d, age %d)", table.Cite, stage, age))

		if !throw.Success {
			continue
		}

		// "If the Aging Check imposes an effect, the characteristic is
		// reduced -1."
		value := characteristicAdd(&c.Characteristics, name, -1)
		log.Consequence(ConsequenceEvent{
			Cause: seq, Kind: ConsequenceAgingEffect,
			Characteristic: name, Delta: -1, Value: value,
		})

		if value > 0 {
			continue
		}

		// "If one Characteristic is reduced to 0, it is reset to 1." The
		// reset is unconditional; the count of zeroes only decides which
		// illness comes with it.
		zeroed++
		last = seq

		reset := characteristicAdd(&c.Characteristics, name, 1)
		log.Consequence(ConsequenceEvent{
			Cause: seq, Kind: ConsequenceCharacteristicReset,
			Characteristic: name, Delta: 1, Value: reset,
		})
	}

	c.illness(zeroed, last, log)
}

// illness applies what the count of characteristics reduced to zero in one
// pass costs beyond the resets (p. 89 chart A).
//
// The chart names two, and three, and stops. Four is reachable once Mental
// Aging joins the three Physical characteristics at Life Stage 9, so the
// three-zero case is read as three or more (interpretation I-49,
// ERRATA.md).
//
// Neither illness costs game years: four weeks and four months are both
// shorter than the year, which is the finest unit the engine tracks.
func (c *Character) illness(zeroed, cause int, log *Log) {
	if zeroed < majorIllnessZeroes {
		return
	}

	if zeroed < extremelyMajorIllnessZeroes {
		// "a major illness and must spend four weeks in rest and
		// recuperation".
		c.MajorIllnesses++

		log.Consequence(ConsequenceEvent{Cause: cause, Kind: ConsequenceMajorIllness, Value: zeroed})

		return
	}

	// "an extremely major illness and must spend four months in rest and
	// recuperation".
	c.ExtremelyMajorIllnesses++

	log.Consequence(ConsequenceEvent{
		Cause: cause, Kind: ConsequenceExtremelyMajorIllness, Value: zeroed,
	})

	// "The second time three characteristics are reduced to 0, the
	// character dies."
	if c.ExtremelyMajorIllnesses >= fatalExtremelyMajorIllnesses {
		c.Dead = true

		log.Consequence(ConsequenceEvent{Cause: cause, Kind: ConsequenceDead})
	}
}
