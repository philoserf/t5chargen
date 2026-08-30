package chargen_test

import (
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/chargen"
)

// branchSeedSearch is how wide the sweeps look. Every Armed Forces
// character is promoted sooner or later, so the offers are common; what
// is scarce is a character promoted twice while still enlisted.
const branchSeedSearch = 300

// changesBranch takes every branch change offered, which the auto policy
// never does (POLICY.md).
type changesBranch struct{ playerKind }

func (changesBranch) Choose(c chargen.Choice) (int, error) {
	if c.ID == chargen.ChooseBranchChange {
		return 0, nil // "Change Branch"
	}

	return autoPolicy(c)
}

// branchesHeld returns the branches the record shows the character in,
// in order, which is what a change has to move.
func branchesHeld(c chargen.Character) []string {
	var held []string

	for _, event := range c.Events {
		if event.Consequence != nil && event.Consequence.Kind == chargen.ConsequenceBranchSet {
			held = append(held, event.Consequence.Skill)
		}
	}

	return held
}

// TestAnEnlistedPromotionMayChangeBranch verifies the sentence all three
// Armed Forces charts print: "Officers may not change Branch; Enlisted
// may select a new Branch upon Promotion" (charts 07 p. 81, 08 p. 82, 12
// p. 86).
//
// Taken over p. 66's "at the end of each Term" as the narrower statement,
// and the one three charts agree on (interpretation I-34).
func TestAnEnlistedPromotionMayChangeBranch(t *testing.T) {
	changed := 0

	for seed := range uint64(branchSeedSearch) {
		c := generate(t, chargen.Options{Seed: seed, Career: "Spacer", Decider: changesBranch{}})

		if held := branchesHeld(c); len(held) > 1 {
			changed++
		}
	}

	if changed == 0 {
		t.Fatalf("no seed under %d changes Branch; the sweep is asserting nothing", branchSeedSearch)
	}
}

// TestAnOfficerMayNotChangeBranch verifies the other half of the same
// sentence. An officer promoted to a higher officer rank is offered
// nothing at all — not offered and declined, which would be a different
// rule and a different record.
func TestAnOfficerMayNotChangeBranch(t *testing.T) {
	def := careerDef(t, "Spacer")
	officers := 0

	for seed := range uint64(branchSeedSearch) {
		c := generate(t, chargen.Options{Seed: seed, Career: "Spacer", Decider: changesBranch{}})

		promotions, offers := officerPromotions(t, c, def)
		if promotions == 0 {
			continue
		}

		officers++

		// One offer is allowed and one only: the commission itself,
		// which p. 66 governs and which is the promotion that made him
		// an officer in the first place.
		if offers > 1 {
			t.Errorf("seed %d: %d branch offers across %d officer promotions",
				seed, offers, promotions)
		}
	}

	if officers == 0 {
		t.Fatalf("no seed under %d is promoted as an officer; the sweep is asserting nothing",
			branchSeedSearch)
	}
}

// officerPromotions counts the rank changes made after the commission,
// and the branch offers made from the commission onward.
//
// The officer ladder is read off the chart rather than matched by title:
// a rank's Class is what the data says it is, and "Captain" is an officer
// in one service and not the word that decides it in another.
func officerPromotions(t *testing.T, c chargen.Character, def *career.Definition) (int, int) {
	t.Helper()

	promotions, offers, commissioned := 0, 0, false

	for _, event := range c.Events {
		if event.Consequence != nil && event.Consequence.Kind == chargen.ConsequenceRankSet {
			if commissioned {
				promotions++
			}

			if officerRank(def, event.Consequence.Skill) {
				commissioned = true
			}
		}

		if commissioned && event.Choice != nil && strings.HasSuffix(event.Choice.Prompt, "Branch?") {
			offers++
		}
	}

	return promotions, offers
}

// officerRank reports whether a rank title is on the career's officer
// ladder.
func officerRank(def *career.Definition, title string) bool {
	for _, rank := range def.Ranks {
		if rank.Title == title {
			return rank.Class == "officer"
		}
	}

	return false
}

// TestACommissionRollsForBranch verifies p. 66's own rule, which nothing
// disputes: "A character who receives a Commission may roll for Branch or
// keep his current Branch (for Spacers, Crew becomes Line)."
//
// A roll, not a selection — the page says roll, and that is the
// difference between this offer and the one an enlisted promotion makes.
func TestACommissionRollsForBranch(t *testing.T) {
	rolled := 0

	for seed := range uint64(branchSeedSearch) {
		c := generate(t, chargen.Options{Seed: seed, Career: "Spacer", Decider: changesBranch{}})

		for i, event := range c.Events {
			if event.Choice == nil || event.Choice.Prompt != "Roll for a new Branch?" {
				continue
			}

			if i+1 >= len(c.Events) || c.Events[i+1].Throw == nil {
				t.Errorf("seed %d: a commissioning branch change threw nothing", seed)

				continue
			}

			if got := c.Events[i+1].Throw.Expr; got != "1D" {
				t.Errorf("seed %d: a commissioning branch change threw %q, want a 1D roll", seed, got)
			}

			rolled++
		}
	}

	if rolled == 0 {
		t.Fatalf("no seed under %d is commissioned and rerolls; the sweep is asserting nothing",
			branchSeedSearch)
	}
}
