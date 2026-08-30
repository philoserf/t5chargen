package chargen_test

import (
	"slices"
	"testing"

	"github.com/philoserf/t5chargen/chargen"
	"github.com/philoserf/t5chargen/skill"
)

// TestCareerKnowledgeIsTermsServed verifies p. 134: "A character who has
// served in a career receives Knowledge equal to the number of terms
// served (to a maximum of 6)."
//
// The cap is the interesting half, and the page says why it is a cap
// rather than a total: eight terms as a Scout still leave "Career:
// Scouts-6", because "he knows a lot, but he has also forgotten some
// things along the way".
func TestCareerKnowledgeIsTermsServed(t *testing.T) {
	capped, uncapped := 0, 0

	for seed := range uint64(200) {
		c := generate(t, chargen.Options{Seed: seed})

		for _, name := range servedCareers(c) {
			terms := 0

			for _, record := range c.Careers {
				if record.Career == name {
					terms += len(record.Terms)
				}
			}

			want := min(terms, chargen.KnowledgeMax)
			if got := levelOf(c, skill.CareerKnowledgePrefix+name); got != want {
				t.Errorf("seed %d: %d terms of %s left Career: %s-%d, want -%d",
					seed, terms, name, name, got, want)
			}

			if terms > chargen.KnowledgeMax {
				capped++
			} else {
				uncapped++
			}
		}
	}

	if capped == 0 || uncapped == 0 {
		t.Fatalf("saw %d capped and %d uncapped careers; the sweep sees only one side of the maximum",
			capped, uncapped)
	}
}

// servedCareers lists the careers a record shows terms of, without
// repeats.
func servedCareers(c chargen.Character) []string {
	var names []string

	for _, record := range c.Careers {
		if record.Began && !containsName(names, record.Career) {
			names = append(names, record.Career)
		}
	}

	return names
}

func containsName(names []string, want string) bool {
	return slices.Contains(names, want)
}

// TestWorldKnowledgeCountsTheTermsBeforeACareer verifies p. 134's World
// Knowledge as this engine reads it (interpretation I-112): the terms
// from age 2 to the age career resolution began, capped at 6.
//
// The page counts a whole life — its example reaches "8 terms counting
// from age 2 through 34" — for a character who never left his world.
// This engine does not know where a character lives once a career has
// him, so it counts only the years it can vouch for, which is the
// deviation I-112 records.
func TestWorldKnowledgeCountsTheTermsBeforeACareer(t *testing.T) {
	bounded, exact := 0, 0

	for seed := range uint64(200) {
		// Schooling declined, so career resolution begins at StartAge
		// and the terms from p. 134's age 2 are known without
		// reconstructing how long each of his terms ran — a school
		// inside a term makes that reconstruction wrong, which is why
		// it is not attempted.
		c := generate(t, chargen.Options{Seed: seed, Decider: noSchooling{}})

		world := c.Homeworld.Name
		if world == "" {
			continue
		}

		held := levelOf(c, skill.WorldKnowledgePrefix+world)

		bounded++

		if held < 0 || held > chargen.KnowledgeMax {
			t.Errorf("seed %d: World: %s-%d outside 0-%d", seed, world, held, chargen.KnowledgeMax)
		}

		if len(c.Education) > 0 {
			continue
		}

		exact++

		if want := (chargen.StartAge - 2) / chargen.TermYears; held != want {
			t.Errorf("seed %d: no schooling, so World: %s-%d, want -%d", seed, world, held, want)
		}
	}

	if bounded == 0 {
		t.Fatal("no seed names a homeworld; the sweep is asserting nothing")
	}

	if exact == 0 {
		t.Fatalf("no seed of %d skips education; the formula is never measured against a known age", bounded)
	}
}

// TestSpecializedKnowledgesCapAtSix verifies the two generated knowledges
// are Knowledges and take the Knowledge ceiling, not the Skill one — they
// are not Master Skill List rows, so nothing looks them up (p. 132's
// Specialized block is a pattern).
func TestSpecializedKnowledgesCapAtSix(t *testing.T) {
	for _, name := range []string{"Career: Scout", "World: Regina"} {
		if !skill.Specialized(name) {
			t.Errorf("%q is not recognised as a chart MS Specialized knowledge", name)
		}

		c, err := chargen.AwardForTest(name, 9)
		if err != nil {
			t.Fatalf("awarding %s: %v", name, err)
		}

		if got := levelOf(c, name); got != chargen.KnowledgeMax {
			t.Errorf("%s awarded nine levels reached %d, want the Knowledge cap of %d",
				name, got, chargen.KnowledgeMax)
		}
	}
}

// noSchooling declines pre-career education, which the auto policy never
// does — every generated character goes to school, so the age careers
// begin at is never StartAge on its own.
type noSchooling struct{ playerKind }

func (noSchooling) Choose(c chargen.Choice) (int, error) {
	if c.ID == chargen.ChooseEducation {
		return len(c.Options) - 1, nil // "No education"
	}

	return autoPolicy(c)
}

// TestWorldKnowledgeCountsFromAgeTwo pins the anchor p. 134 gives: its
// example reaches "8 terms counting from age 2 through 34", which is
// thirty-two years over the four-year Term.
//
// Measured directly because the ordinary career start hides it: at 18,
// counting from 2 and counting from 0 both give four terms, so nothing
// swept over generated characters can tell the two apart.
func TestWorldKnowledgeCountsFromAgeTwo(t *testing.T) {
	for _, tc := range []struct{ age, want int }{
		{age: 34, want: 8}, // p. 134's own example, before its cap
		{age: 18, want: 4}, // the ordinary start, where the anchor hides
		{age: 20, want: 4}, // from 0 this would be 5
		{age: 6, want: 1},
		{age: 2, want: 0},
		{age: 0, want: 0},
	} {
		if got := chargen.TermsLivedForTest(tc.age); got != tc.want {
			t.Errorf("a career beginning at %d gives %d terms lived, want %d", tc.age, got, tc.want)
		}
	}
}

// TestACareerServedTwiceIsOneKnowledge verifies "the number of terms
// served" totals across a career left and returned to (I-54 allows the
// return), rather than awarding the Knowledge once per stint.
//
// Unreachable under the auto policy, which never changes careers, so a
// decider that does is what reaches it.
func TestACareerServedTwiceIsOneKnowledge(t *testing.T) {
	repeats := 0

	for seed := range uint64(400) {
		c := generate(t, chargen.Options{Seed: seed, Decider: &laterRogue{}})

		for name, stint := range stints(c) {
			if stint.stints < 2 {
				continue
			}

			repeats++

			want := min(stint.terms, chargen.KnowledgeMax)
			if got := levelOf(c, skill.CareerKnowledgePrefix+name); got != want {
				t.Errorf("seed %d: %d terms of %s over %d stints left -%d, want -%d",
					seed, stint.terms, name, stint.stints, got, want)
			}
		}
	}

	if repeats == 0 {
		t.Fatal("no seed serves one career twice; the sweep is asserting nothing")
	}
}

// stint totals a career's terms and how many separate times it was
// served.
type stint struct{ terms, stints int }

// stints groups a record's careers by name.
func stints(c chargen.Character) map[string]stint {
	byName := map[string]stint{}

	for _, record := range c.Careers {
		if !record.Began {
			continue
		}

		held := byName[record.Career]
		held.terms += len(record.Terms)
		held.stints++
		byName[record.Career] = held
	}

	return byName
}
