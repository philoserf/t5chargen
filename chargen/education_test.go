package chargen_test

import (
	"testing"

	"github.com/philoserf/t5chargen/chargen"
)

// TestEducationInvariants sweeps seeds and checks step C rule invariants:
// exactly one pre-career record, the policy's college track, age equal to
// 18 plus every years_elapsed event, and graduation Edu benefits (chart C
// p. 60; prose p. 59).
func TestEducationInvariants(t *testing.T) {
	for seed := range uint64(40) {
		c := generate(t, chargen.Options{Seed: seed})

		if len(c.Education) != 1 {
			t.Fatalf("seed %d: education records = %+v", seed, c.Education)
		}

		record := c.Education[0]
		if record.Program != "University" && record.Program != "College" && record.Program != "ED5" {
			t.Errorf("seed %d: policy program = %q", seed, record.Program)
		}

		checkEducationAge(t, seed, c)
		checkGraduationEdu(t, seed, c, record)
	}
}

// checkEducationAge asserts age accounting: 18 plus the sum of
// years_elapsed events (education years plus career terms).
func checkEducationAge(t *testing.T, seed uint64, c chargen.Character) {
	t.Helper()

	years := 0

	for _, event := range c.Events {
		if event.Kind == chargen.EventConsequence && event.Consequence.Kind == chargen.ConsequenceYearsElapsed {
			years += event.Consequence.Value
		}
	}

	if want := chargen.StartAge + years; c.Age != want {
		t.Errorf("seed %d: age %d, want %d (18 + %d elapsed years)", seed, c.Age, want, years)
	}
}

// checkGraduationEdu asserts the Graduation column applied: College Edu=8,
// University Edu=9, ED5 Edu-5 — or Edu+1 when already at the level (p. 60).
func checkGraduationEdu(t *testing.T, seed uint64, c chargen.Character, record chargen.EducationRecord) {
	t.Helper()

	if !record.Graduated {
		return
	}

	minimums := map[string]int{"ED5": 5, "College": 8, "University": 9}

	if want := minimums[record.Program]; c.Characteristics.Edu < want {
		t.Errorf("seed %d: graduated %s with Edu %d, want >= %d",
			seed, record.Program, c.Characteristics.Edu, want)
	}

	if record.Program == "College" || record.Program == "University" {
		if record.Degree == "" {
			t.Errorf("seed %d: graduated %s without a degree", seed, record.Program)
		}
	}
}

// TestEducationWaiverPinned pins seed 1: a College year fails, the waiver
// succeeds ("Waiver may result in reinstatement, although no skill is
// received", p. 59), and the character still graduates with fewer passes
// than rolls.
func TestEducationWaiverPinned(t *testing.T) {
	c := generate(t, chargen.Options{Seed: 1})

	record := c.Education[0]
	if record.Program != "College" || !record.Graduated || record.Passes >= 4 {
		t.Fatalf("seed 1 education = %+v; expected a graduated College run with a waived year", record)
	}

	if c.WaiversAttempted < 1 {
		t.Errorf("seed 1 waivers attempted = %d, want >= 1", c.WaiversAttempted)
	}
}

// languageDecider is the default policy except that it majors in Language,
// to exercise the double-rate rule.
type languageDecider struct{ playerKind }

func (languageDecider) Choose(c chargen.Choice) (int, error) {
	if c.ID == chargen.ChooseMajor {
		for i, option := range c.Options {
			if option == "Language" {
				return i, nil
			}
		}
	}

	return autoPolicy(c)
}

// TestLanguageDoubleRate verifies "When a specific Language is specified
// as a Major or Minor, it is acquired at double rate" (p. 59), per the
// Saga Emm example (four passes of college = +8).
func TestLanguageDoubleRate(t *testing.T) {
	for seed := range uint64(20) {
		c := generate(t, chargen.Options{Seed: seed, Decider: languageDecider{}})

		record := c.Education[0]
		if record.Major != "Language" {
			continue // ED5 seeds have no major
		}

		total := 0

		for _, event := range c.Events {
			if event.Kind != chargen.EventConsequence {
				continue
			}

			consequence := event.Consequence
			if consequence.Kind == chargen.ConsequenceSkillAwarded && consequence.Skill == "Language" {
				if consequence.Delta%2 != 0 {
					t.Errorf("seed %d: Language award delta %d, want double-rate even values", seed, consequence.Delta)
				}

				total += consequence.Delta
			}
		}

		honors := 0
		if record.Honors {
			honors = 1
		}

		if want := 2 * (record.Passes + honors); total != want {
			t.Errorf("seed %d: Language total %d, want %d (2 x (%d passes + %d honors))",
				seed, total, want, record.Passes, honors)
		}
	}
}

// TestEducationFeedsRecord verifies graduation's Edu change lands in the
// stored characteristics and derived UPP.
func TestEducationFeedsRecord(t *testing.T) {
	c := generate(t, chargen.Options{Seed: 1})

	if c.Characteristics.Edu < 8 {
		t.Fatalf("seed 1 Edu = %d, want >= 8 after College graduation", c.Characteristics.Edu)
	}

	if c.UPP != c.Characteristics.UPP() {
		t.Errorf("UPP %q does not match post-education characteristics %q", c.UPP, c.Characteristics.UPP())
	}
}
