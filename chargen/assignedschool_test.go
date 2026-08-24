package chargen_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/chargen"
	"github.com/philoserf/t5chargen/education"
	"github.com/philoserf/t5chargen/skill"
)

// The pinned seeds reach each assigned school under the default policy.
// Every military career is covered because all three charts carry both:
// the ANM School Operations row and the flag-rank Command College footnote
// (charts 07, 08, 12).
// The anm seeds are ones whose ANM Pass/Fail succeeds, not merely ones
// that attend: a failed year awards nothing, so a seed that only attends
// leaves the Provides column untested. The first three pinned here all
// attended and all failed, which is how that was found.
var assignedSchoolSeeds = []struct {
	career  string
	anm     uint64
	college uint64
}{
	{career: "Soldier", anm: 9, college: 34},
	{career: "Spacer", anm: 5, college: 15},
	{career: "Marine", anm: 12, college: 11},
}

// attended reports the education records for one program.
func attended(c chargen.Character, program string) int {
	n := 0

	for _, record := range c.Education {
		if record.Program == program {
			n++
		}
	}

	return n
}

// TestANMSchoolIsResolvedAsEducation verifies "Resolve ANM School using
// Education" (charts 07, 08, 12): the Operations assignment runs the chart
// C row rather than passing as a Mod-0 assignment that does nothing.
func TestANMSchoolIsResolvedAsEducation(t *testing.T) {
	for _, tc := range assignedSchoolSeeds {
		t.Run(tc.career, func(t *testing.T) {
			c := generate(t, chargen.Options{Seed: tc.anm, Career: tc.career})

			if attended(c, "ANM School") == 0 {
				t.Fatalf("seed %d %s no longer reaches ANM School; find and pin another",
					tc.anm, tc.career)
			}

			// The seed must reach the award, not merely the school: a
			// failed Pass/Fail attends and provides nothing.
			passed := false

			for _, record := range c.Education {
				if record.Program == "ANM School" && record.Passes > 0 {
					passed = true
				}
			}

			if !passed {
				t.Errorf("seed %d %s attends ANM School but never passes; the Provides column is untested",
					tc.anm, tc.career)
			}
		})
	}
}

// TestANMSchoolAwardsAKnowledge verifies the Provides column: "Knowledge-2
// from School=ANM" (chart C p. 60). ANM is Army-Navy-Marine, so the source
// is those three columns of the Available Skills matrix, and the row asks
// for a Knowledge — not a skill, and not a bare container name, which is
// the p. 134 question this award stays clear of.
func TestANMSchoolAwardsAKnowledge(t *testing.T) {
	knowledges, err := education.ANMKnowledges()
	if err != nil {
		t.Fatal(err)
	}

	if len(knowledges) == 0 {
		t.Fatal("the ANM source list is empty, so the award has nothing to choose from")
	}

	awarded := 0

	for _, tc := range assignedSchoolSeeds {
		c := generate(t, chargen.Options{Seed: tc.anm, Career: tc.career})

		for _, event := range c.Events {
			if event.Kind != chargen.EventChoice || event.Choice.Prompt != "Select the ANM School Knowledge" {
				continue
			}

			awarded++

			if !slices.Equal(event.Choice.Options, knowledges) {
				t.Errorf("%s was offered %d options, want the %d ANM Knowledges",
					tc.career, len(event.Choice.Options), len(knowledges))
			}

			assertAllKnowledges(t, event.Choice.Options)
		}
	}

	if awarded == 0 {
		t.Error("no ANM School award was made on any pinned seed")
	}
}

// assertAllKnowledges checks an offer against the Master Skill List rather
// than against ANMKnowledges. Comparing the offer to the function that
// built it moves both sides together: that version of the test passed with
// the Knowledge filter removed.
func assertAllKnowledges(t *testing.T, options []string) {
	t.Helper()

	for _, option := range options {
		if entry, ok := skill.Lookup(option); !ok || entry.Kind != skill.KindKnowledge {
			t.Errorf("ANM School offered %q, which is not a Knowledge", option)
		}
	}
}

// TestCommandCollegeFollowsTheRank verifies the flag-rank footnote:
// "Command College in Year 1 of next Term (if Continue)". The officer
// attends after the rank, not with it — so the record must show the rank
// set before the school runs.
func TestCommandCollegeFollowsTheRank(t *testing.T) {
	for _, tc := range assignedSchoolSeeds {
		t.Run(tc.career, func(t *testing.T) {
			c := generate(t, chargen.Options{Seed: tc.college, Career: tc.career})

			if attended(c, "Command College") == 0 {
				t.Fatalf("seed %d %s no longer reaches Command College; find and pin another",
					tc.college, tc.career)
			}

			ranked := false

			for _, event := range c.Events {
				if event.Kind == chargen.EventConsequence &&
					event.Consequence.Kind == chargen.ConsequenceRankSet {
					ranked = true
				}

				if event.Kind == chargen.EventStep && event.Step.Name == "Assigned school: Command College" {
					if !ranked {
						t.Error("Command College ran before any rank was set")
					}

					return
				}
			}
		})
	}
}

// TestCommandCollegeAwardsTwoSkills verifies "2x Skill-1" (chart C p. 60):
// two selections, not one. The row names no source, so the list is the
// whole Available Skills matrix — the reading interpretation I-7 already
// gives the Apprenticeship's unqualified "Skill+4".
func TestCommandCollegeAwardsTwoSkills(t *testing.T) {
	all, err := education.AllSkillNames()
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range assignedSchoolSeeds {
		t.Run(tc.career, func(t *testing.T) {
			c := generate(t, chargen.Options{Seed: tc.college, Career: tc.career})

			picks := 0

			for _, event := range c.Events {
				if event.Kind == chargen.EventChoice && event.Choice.Prompt == "Select a Command College skill" {
					picks++

					if len(event.Choice.Options) != len(all) {
						t.Errorf("offered %d options, want the whole matrix's %d",
							len(event.Choice.Options), len(all))
					}
				}
			}

			// A Pass/Fail failure awards nothing, so the count is two per
			// pass rather than two per attendance.
			if picks == 0 || picks%2 != 0 {
				t.Errorf("%d Command College selections, want two per passed attendance", picks)
			}
		})
	}
}

// TestAnAssignedSchoolCostsNoExtraYears verifies interpretation I-91: an
// assigned school is sited inside a term the character is already spending
// — the Operations assignment is one of the term's four, and Command
// College is expressly "in Year 1 of next Term" — so it adds no years.
// Later Education is the contrast: it substitutes for the whole term and
// costs all four (I-88).
//
// The assertion is an exact equality on the whole lifepath: every year is
// either pre-career education or one of a served term's four, and an
// assigned school adds none. Two earlier attempts were weaker and both
// passed with the suppression removed — a "not too few years" comparison
// cannot see an extra one, and a span measured from the school's step runs
// on into the term, because the log marks where a school begins and not
// where it ends.
func TestAnAssignedSchoolCostsNoExtraYears(t *testing.T) {
	for _, tc := range assignedSchoolSeeds {
		for _, seed := range []uint64{tc.anm, tc.college} {
			c := generate(t, chargen.Options{Seed: seed, Career: tc.career})

			if attended(c, "ANM School")+attended(c, "Command College") == 0 {
				t.Fatalf("seed %d %s attends no assigned school; the case is not being tested", seed, tc.career)
			}

			before, total := preCareerYears(c), totalYears(c)

			if want := before + chargen.TermYears*servedTerms(c); total != want {
				t.Errorf("seed %d %s: %d years elapsed, want %d (%d before the career, %d terms of %d)",
					seed, tc.career, total, want, before, servedTerms(c), chargen.TermYears)
			}

			if c.Age != chargen.StartAge+total {
				t.Errorf("seed %d %s: age %d does not match its elapsed years", seed, tc.career, c.Age)
			}
		}
	}
}

// preCareerYears sums the years elapsed before the first career term, which
// is checklist step C's schooling.
func preCareerYears(c chargen.Character) int {
	years := 0

	for _, event := range c.Events {
		if event.Kind == chargen.EventStep && strings.Contains(event.Step.Name, ": Term ") {
			return years
		}

		if event.Kind == chargen.EventConsequence &&
			event.Consequence.Kind == chargen.ConsequenceYearsElapsed {
			years += event.Consequence.Value
		}
	}

	return years
}

// totalYears sums every elapsed-years consequence in the record.
func totalYears(c chargen.Character) int {
	years := 0

	for _, event := range c.Events {
		if event.Kind == chargen.EventConsequence &&
			event.Consequence.Kind == chargen.ConsequenceYearsElapsed {
			years += event.Consequence.Value
		}
	}

	return years
}
