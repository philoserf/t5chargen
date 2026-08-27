package chargen_test

import (
	"strconv"
	"testing"

	"github.com/philoserf/t5chargen/chargen"
)

// TestKnowledgeKnowledgeSkill verifies p. 134's own table, receipt by
// receipt:
//
//	First Receipt of Skill=  Skill-0. Knowledge-1
//	Second Receipt of Skill= Skill-0. Knowledge-2
//	Third Receipt of Skill=  Skill-1. Knowledge-2
//	Fourth Receipt of Skill= Skill-2. Knowledge-2
//
// The fifth row is not printed and is the one the page's worked example
// lands on: Darren Buck emerged from a term "with five levels of Fighter
// ... took the first two knowledges as Slug-Thrower, and mustered out
// with Fighter-3, Slug Thrower-2".
func TestKnowledgeKnowledgeSkill(t *testing.T) {
	for _, tc := range []struct {
		receipts  int
		skill     int
		knowledge int
	}{
		{receipts: 1, skill: 0, knowledge: 1},
		{receipts: 2, skill: 0, knowledge: 2},
		{receipts: 3, skill: 1, knowledge: 2},
		{receipts: 4, skill: 2, knowledge: 2},
		{receipts: 5, skill: 3, knowledge: 2}, // p. 134's Darren Buck
	} {
		t.Run(name(tc.receipts), func(t *testing.T) {
			c := receiveFighter(t, tc.receipts)

			if got := levelOf(c, "Fighter"); got != tc.skill {
				t.Errorf("%d receipts left Fighter-%d, want Fighter-%d", tc.receipts, got, tc.skill)
			}

			// The policy concentrates, so the first-listed Knowledge is
			// the one that grew.
			if got := levelOf(c, "Battle Dress"); got != tc.knowledge {
				t.Errorf("%d receipts left Battle Dress-%d, want -%d", tc.receipts, got, tc.knowledge)
			}
		})
	}
}

// name labels a subtest by its receipt count.
func name(n int) string {
	return strconv.Itoa(n) + " receipts"
}

// receiveFighter awards Fighter n times through the funnel every award
// goes through, and returns the resulting character.
func receiveFighter(t *testing.T, n int) chargen.Character {
	t.Helper()

	c, err := chargen.AwardForTest("Fighter", n)
	if err != nil {
		t.Fatalf("awarding Fighter %d times: %v", n, err)
	}

	return c
}

// levelOf reports the level held in a named skill or knowledge.
func levelOf(c chargen.Character, name string) int {
	for _, s := range c.Skills {
		if s.Name == name {
			return s.Level
		}
	}

	return -1
}

// TestAContainerIsHeldAtSkillZeroUntilTheThirdReceipt verifies the clause
// the table leaves implicit: "Until then, he has the Knowledges but only
// Skill-0" (p. 134).
//
// Skill-0 is not nothing. It says the character has the skill, and a
// record that omitted it would read as one who had never met it.
func TestAContainerIsHeldAtSkillZeroUntilTheThirdReceipt(t *testing.T) {
	for _, receipts := range []int{1, 2} {
		c := receiveFighter(t, receipts)

		if got := levelOf(c, "Fighter"); got != 0 {
			t.Errorf("after %d receipts Fighter is %d; want it held at 0", receipts, got)
		}
	}
}

// TestAKnowledgeCapsAtSix verifies "The maximum level of a Knowledge is
// 6" (p. 134), which is not the Skill-15 cap the same page prints beside
// it.
func TestAKnowledgeCapsAtSix(t *testing.T) {
	if chargen.KnowledgeMax != 6 {
		t.Fatalf("KnowledgeMax is %d, want 6", chargen.KnowledgeMax)
	}

	if chargen.SkillMax == chargen.KnowledgeMax {
		t.Error("the Skill and Knowledge caps are the same number; p. 134 prints two")
	}
}

// TestLanguageAndMusicianAreAwardedWhole verifies the two containers that
// take no Knowledge: Language is excepted by p. 134 in the sentence that
// lists them, and Musician has no list printed anywhere (I-111).
func TestLanguageAndMusicianAreAwardedWhole(t *testing.T) {
	for _, name := range []string{"Language", "Musician"} {
		c, err := chargen.AwardForTest(name, 3)
		if err != nil {
			t.Fatalf("awarding %s: %v", name, err)
		}

		if got := levelOf(c, name); got != 3 {
			t.Errorf("%s received three times is %d, want 3 — it takes no Knowledge", name, got)
		}
	}
}
