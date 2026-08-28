package chargen

import (
	"maps"
	"slices"

	"github.com/philoserf/t5chargen/career"
)

// RegistryNames is a test bridge to the careerRegistry key set, sorted.
func RegistryNames() []string {
	return slices.Sorted(maps.Keys(careerRegistry))
}

// RegistryDefinitionNames is a test bridge mapping each careerRegistry key
// to its definition's Name — dispatch keys on the registry key while every
// user-visible label derives from Definition.Name, so the two must match.
func RegistryDefinitionNames() (map[string]string, error) {
	names := make(map[string]string, len(careerRegistry))

	for key, entry := range careerRegistry {
		def, _, err := entry()
		if err != nil {
			return nil, err
		}

		names[key] = def.Name
	}

	return names, nil
}

// LoadUndercoverCareer is a test bridge to the Agent's cover-career
// resolution, so the transcribed Undercover table can be checked against
// the careers the engine can actually read.
func LoadUndercoverCareer(source string) (*career.Definition, error) {
	return loadUndercoverCareer(source)
}

// ScoutDuties exports chart 05's duty labels. They are a Go literal and
// their skill eligibility is a JSON map, so only a test can hold the two
// key sets together.
var ScoutDuties = scoutDuties

// HoldsDegree is a test bridge to the chart C prerequisite check. The four
// programs it gates are still data-only, so nothing in a generated
// lifepath reaches it yet and only a direct test can.
func HoldsDegree(c *Character, want string) bool { return c.holdsDegree(want) }

// GraduationEdu is a test bridge to chart C's Graduation column and the
// parenthetical above it, which is a pure function of two numbers and best
// pinned as one.
func GraduationEdu(edu, graduation int) (int, bool) { return graduationEdu(edu, graduation) }

// AwardForTest runs one skill award through the funnel every award goes
// through, and returns the character it landed on.
//
// The Knowledge-Knowledge-Skill sequence is the funnel's own business
// (p. 134), so exercising it needs no career, no seed and no term — which
// is the point of there being one funnel.
func AwardForTest(name string, levels int) (Character, error) {
	var (
		character Character
		log       Log
	)

	err := awardSkillAndLog(name, levels, 0, &log, DefaultPolicy{}, &character)

	return character, err
}

// TermsLivedForTest exports the World Knowledge count so the age it
// anchors on can be measured directly. p. 134 counts "from age 2", and
// at the ordinary career start of 18 that is indistinguishable from
// counting from 0 — both give four terms — so a sweep over generated
// characters cannot see the anchor at all.
func TermsLivedForTest(careerStartAge int) int { return termsLived(careerStartAge) }
