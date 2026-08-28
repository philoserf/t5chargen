package chargen

// The two Knowledges chart MS files under Specialized (p. 132) and p. 134
// computes rather than awards:
//
//	Specialized
//	Career: Academia   Career: Army   Career: Navy   Career: <Name>
//	World: Capital     World: Regina  World: <Name>
//	[others are possible]
//
// The list is a pattern with examples rather than a closed vocabulary —
// "[others are possible]" — so the names are generated from the record
// and nothing is transcribed. A character who served the Scouts holds
// "Career: Scout"; one raised on Regina holds "World: Regina".
//
// Neither is rolled for. "A character who has served in a career receives
// Knowledge equal to the number of terms served (to a maximum of 6)" and
// "A character who has spent time on a world receives Knowledge equal to
// the number of terms he has lived there (but a maximum 6)" (p. 134).

import (
	"slices"

	"github.com/philoserf/t5chargen/skill"
)

// firstTermAge is the age p. 134 counts a world's terms from: its worked
// example reaches "8 terms counting from age 2 through 34", which is
// thirty-two years divided by the four-year Term.
const firstTermAge = 2

// awardSpecializedKnowledges records the Career and World Knowledges over
// the finished record (p. 134), which is why they are computed here
// rather than as the careers run: a career's Knowledge is its total terms
// and is not known until the last of them is served.
func awardSpecializedKnowledges(c *Character, log *Log, careerStartAge int) {
	// The consequences name this step. p. 134 awards both by arithmetic
	// over the finished record — "Knowledge equal to the number of terms
	// served" — and names no die anywhere, so there is no throw or choice
	// to point at (interpretation I-87, ERRATA.md).
	seq := log.Step("Career and World Knowledges", "Book 1 p. 134; chart MS p. 132")

	for _, name := range careersServed(c) {
		awardSkillLevels(skill.CareerKnowledgePrefix+name, termsServed(c, name), seq, log, c)
	}

	if world := c.Homeworld.Name; world != "" {
		awardSkillLevels(skill.WorldKnowledgePrefix+world, termsLived(careerStartAge), seq, log, c)
	}
}

// careersServed lists the careers the character served, in the order he
// served them and without repeats: a career re-entered later is one
// career he has served, not two.
func careersServed(c *Character) []string {
	var names []string

	for _, record := range c.Careers {
		if record.Began && !slices.Contains(names, record.Career) {
			names = append(names, record.Career)
		}
	}

	return names
}

// termsServed totals the terms a character served in one career, which is
// what its Knowledge is worth. p. 134's example is explicit that the
// total keeps counting and the Knowledge does not: eight terms as a Scout
// still leave "Career: Scouts-6", because "he has also forgotten some
// things along the way".
func termsServed(c *Character, name string) int {
	terms := 0

	for _, record := range c.Careers {
		if record.Career == name {
			terms += len(record.Terms)
		}
	}

	return terms
}

// termsLived is the World Knowledge a character leaves home with: the
// terms from p. 134's age 2 to the age career resolution began.
//
// _Deviation, interpretation I-112._ p. 134's own example counts the
// whole life — a Citizen who never left home reaches age 34 with eight
// terms of it — because that character lived on his world the entire
// time. This engine does not know where a character lives once a career
// has him, so it counts only the years it can vouch for.
func termsLived(careerStartAge int) int {
	if careerStartAge <= firstTermAge {
		return 0
	}

	return (careerStartAge - firstTermAge) / TermYears
}
