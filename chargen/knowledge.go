package chargen

//nolint:dupword // "Knowledge, Knowledge, Skill" is the rule's printed name.
// Knowledge, Knowledge, Skill (Book 1 p. 134).
//
// "Some skills include within them several Knowledges (Animals, Driver,
// Engineer, Fighter, Flyer, Gunner, Heavy Weapons, Language, Musician,
// Pilot, Seafarer). Acquisition of these skills (except Language which is
// handled differently) uses the following sequence:
//
//	Knowledge, Knowledge, Skill.
//
// The first two instances a character receives one of these Skills
// (typically in Character Generation), he instead receives one of the
// Skill's contained Knowledges. When (or If) the character acquires the
// skill the third time, he receives the Skill at level-1. Until then, he
// has the Knowledges but only Skill-0 (reflecting some familiarity with
// the overall Skill, but a concentration in the Knowledges)."
//
// The chart states it as a table:
//
//	First Receipt of Skill=  Skill-0. Knowledge-1
//	Second Receipt of Skill= Skill-0. Knowledge-2
//	Third Receipt of Skill=  Skill-1. Knowledge-2
//	Fourth Receipt of Skill= Skill-2. Knowledge-2
//
// so the Nth receipt leaves the container at max(0, N-2) and the
// Knowledge at min(N, 2).

import (
	"github.com/philoserf/t5chargen/skill"
)

// KnowledgeMax caps a Knowledge: "The maximum level of a Knowledge is 6"
// (p. 134). Skills cap at SkillMax, which is 15, and the two are
// different numbers for different things — a container's own level is a
// Skill and is capped as one.
const KnowledgeMax = 6

// knowledgeReceipts is how many receipts of a container go to its
// Knowledges before the Skill itself begins to rise (p. 134).
const knowledgeReceipts = 2

// containerKnowledges returns the Knowledges a skill contains, or nil
// where it contains none.
//
// Two of p. 134's eleven containers return nil, for reasons recorded
// rather than inferred. Language is excepted by p. 134 in the sentence
// that names them — "except Language which is handled differently" — and
// Musician has no list printed anywhere to select from (I-111), so both
// are awarded whole.
func containerKnowledges(name string) []string {
	if name == "Language" || name == "Musician" {
		return nil
	}

	return skill.UnderParent(name)
}

// awardContainer applies the Knowledge-Knowledge-Skill sequence to one
// award of a container skill, and reports whether it handled the award.
//
// Each level is a receipt. p. 134's worked example is unambiguous: a
// character who emerged from a term "with five levels of Fighter ... took
// the first two knowledges as Slug-Thrower, and mustered out with
// Fighter-3, Slug Thrower-2". Five levels, two of them Knowledges and
// three of them Skill, from one term's awards.
//
// The container is recorded at Skill-0 from the first receipt, which is
// what "he has the Knowledges but only Skill-0" asks for and what tells a
// reader of the sheet that the character has the skill at all.
func awardContainer(name string, levels, cause int, log *Log, decider Decider, character *Character) (bool, error) {
	knowledges := containerKnowledges(name)
	if len(knowledges) == 0 {
		return false, nil
	}

	for range levels {
		if err := awardOneReceipt(name, knowledges, cause, log, decider, character); err != nil {
			return true, err
		}
	}

	return true, nil
}

// awardOneReceipt applies a single receipt of a container skill.
func awardOneReceipt(name string, knowledges []string, cause int,
	log *Log, decider Decider, character *Character,
) error {
	// Past the second receipt the container itself rises, one level at a
	// time: "the third time, he receives the Skill at level-1", and the
	// fourth Skill-2. Awarded directly rather than back through the
	// funnel, which would send it round the progression again.
	if character.recordReceipt(name) > knowledgeReceipts {
		awardSkillLevels(name, 1, cause, log, character)

		return nil
	}

	chosen, seq, err := choose(log, decider, Choice{
		ID:      ChooseKnowledge,
		Prompt:  "Select a " + name + " Knowledge",
		Options: knowledges,
		Scores:  knowledgeLevels(character, knowledges),
		Cite:    "Book 1 p. 134 (Knowledge, Knowledge, Skill)",
	})
	if err != nil {
		return err
	}

	awardSkillLevels(knowledges[chosen], 1, seq, log, character)

	return nil
}

// knowledgeLevels reports the level held in each option, so a decider can
// see where a character has already concentrated without reading the
// prompt (POLICY.md).
func knowledgeLevels(character *Character, knowledges []string) []int {
	levels := make([]int, len(knowledges))
	for i, name := range knowledges {
		levels[i] = character.skillLevel(name)
	}

	return levels
}
