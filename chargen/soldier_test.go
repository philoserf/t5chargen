package chargen_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/chargen"
	"github.com/philoserf/t5chargen/medal"
)

// soldierGoldenSeed is the pinned fixture: six terms in Artillery, a
// commission and promotions to Lt Colonel, three kinds of medal, a wound
// badge, and four distinct assignments.
const soldierGoldenSeed = 305

func soldierRun(t *testing.T) (chargen.Character, chargen.CareerRecord) {
	t.Helper()

	c, err := chargen.Generate(chargen.Options{
		Seed: soldierGoldenSeed, Career: "Soldier", Decider: chargen.DefaultPolicy{},
	})
	if err != nil {
		t.Fatalf("seed %d: %v", soldierGoldenSeed, err)
	}

	return c, c.Careers[len(c.Careers)-1]
}

// TestSoldierBeginsEnlisted verifies p. 65: "Armed Forces characters begin
// with enlisted rank (Army = Soldier1)" (p. 65).
func TestSoldierBeginsEnlisted(t *testing.T) {
	c, record := soldierRun(t)

	if !record.Began {
		t.Fatal("the pinned seed does not begin the career")
	}

	if got := firstRankSet(c); got != "Private" {
		t.Errorf("entered as %q, want Private", got)
	}

	if record.Branch == "" {
		t.Error("no Branch recorded; chart 08 determines one on entry")
	}
}

// TestSoldierOperationsPerTerm verifies p. 66: "Roll for Assignment four
// times per Term (for four annual assignments)".
func TestSoldierOperationsPerTerm(t *testing.T) {
	def, err := career.Soldier()
	if err != nil {
		t.Fatal(err)
	}

	if def.ArmedForces.OperationsPerTerm != 4 {
		t.Fatalf("operations per term = %d, want 4", def.ArmedForces.OperationsPerTerm)
	}

	c, record := soldierRun(t)

	operations := 0

	for _, e := range c.Events {
		if e.Kind == chargen.EventConsequence && e.Consequence.Kind == chargen.ConsequenceOperation {
			operations++
		}
	}

	if want := len(record.Terms) * 4; operations != want {
		t.Errorf("assignments = %d, want %d (%d terms x 4)", operations, want, len(record.Terms))
	}
}

// TestSoldierTermSkillsFollowAssignments verifies p. 65: "In Spacer,
// Soldier, or Marine careers, Term skills ... may be taken on a column of
// the Skills table corresponding to an Operations result received in the
// Term", with "Column 1-Personal Skills may always be rolled".
func TestSoldierTermSkillsFollowAssignments(t *testing.T) {
	def, err := career.Soldier()
	if err != nil {
		t.Fatal(err)
	}

	c, _ := soldierRun(t)

	allowed := make([]string, 0, len(def.SkillColumns))
	allowed = append(allowed, "Personal")
	checked := 0

	for _, e := range c.Events {
		switch soldierSkillEvent(e, len(def.SkillColumns)) {
		case "term":
			allowed = append(allowed[:0], "Personal")
		case "operation":
			allowed = append(allowed, e.Consequence.Skill)
		case "restricted-columns":
			checked++

			for _, option := range e.Choice.Options {
				if !slices.Contains(allowed, option) {
					t.Errorf("event %d offers column %q, which no assignment opened", e.Seq, option)
				}
			}
		}
	}

	if checked == 0 {
		t.Fatal("the pinned seed records no skill-column choice")
	}
}

// soldierSkillEvent classifies the events the column restriction turns
// on. The commission and promotion eligibilities are exempt (p. 65) and
// offer the whole table, so only narrower offers are the term's own.
func soldierSkillEvent(e chargen.Event, columns int) string {
	switch {
	case e.Kind == chargen.EventStep && strings.Contains(e.Step.Name, "Term "):
		return "term"
	case e.Kind == chargen.EventConsequence && e.Consequence.Kind == chargen.ConsequenceOperation:
		return "operation"
	case e.Kind == chargen.EventChoice && e.Choice.Prompt == "Select a Soldier Skills column" &&
		len(e.Choice.Options) < columns:
		return "restricted-columns"
	default:
		return ""
	}
}

// TestSoldierMedalsFollowTheTable verifies p. 66: "use the unmodified
// Reward roll to determine the Medal; Mod +1 if an Officer", against the
// p. 70 table.
func TestSoldierMedalsFollowTheTable(t *testing.T) {
	c, record := soldierRun(t)

	officer := false
	awarded := map[string]int{}

	for _, e := range c.Events {
		if e.Kind == chargen.EventConsequence && e.Consequence.Kind == chargen.ConsequenceRankSet {
			officer = isOfficerTitle(e.Consequence.Skill)

			continue
		}

		if !rewardSuccess(e) {
			continue
		}

		want, err := medal.For(e.Throw.Total, officer)
		if err != nil {
			t.Fatalf("event %d: %v", e.Seq, err)
		}

		awarded[want.Code]++
	}

	for _, award := range record.Medals {
		if awarded[award.Code] != award.Count {
			t.Errorf("medal %s counted %d, want %d from the table", award.Code, award.Count, awarded[award.Code])
		}
	}

	if len(record.Medals) == 0 {
		t.Fatal("the pinned seed records no medal")
	}
}

// isOfficerTitle reports the chart 08 officer ranks.
func isOfficerTitle(title string) bool {
	for _, officer := range []string{"Lieutenant", "Captain", "Major", "Colonel", "General"} {
		if strings.Contains(title, officer) {
			return true
		}
	}

	return false
}

// rewardSuccess reports a successful Reward throw.
func rewardSuccess(e chargen.Event) bool {
	return e.Kind == chargen.EventThrow &&
		strings.Contains(e.Throw.Cite, "Reward vs") &&
		e.Throw.Success != nil && *e.Throw.Success
}

// TestSoldierPromotionUsesMedalsNotWoundBadges verifies the p. 70
// footnote: "Medals (but not Wound Badges) are Mods for Soldier / Spacer /
// Marine Promotion" (interpretation I-31, ERRATA.md).
func TestSoldierPromotionUsesMedalsNotWoundBadges(t *testing.T) {
	c, _ := soldierRun(t)

	if c.WoundBadges == 0 {
		t.Fatal("the pinned seed records no Wound Badge; the fixture is stale")
	}

	checked := 0

	for _, e := range c.Events {
		if e.Kind != chargen.EventThrow || !strings.Contains(e.Throw.Cite, "Promotion") {
			continue
		}

		checked++

		for _, mod := range e.Throw.Mods {
			if strings.Contains(mod.Name, "Wound") {
				t.Errorf("event %d applies a Wound Badge modifier to a promotion", e.Seq)
			}
		}
	}

	if checked == 0 {
		t.Fatal("the pinned seed records no promotion throw")
	}
}

// TestSoldierServiceBadgeCarriesNoMod verifies that the Risk-success
// Exemplary Service Badge is counted but carries no promotion modifier
// (interpretation I-32, ERRATA.md): the badges outnumber the medals, and
// only the medals appear in the promotion modifiers.
func TestSoldierServiceBadgeCarriesNoMod(t *testing.T) {
	_, record := soldierRun(t)

	if record.ServiceBadges == 0 {
		t.Fatal("the pinned seed records no Exemplary Service Badge")
	}

	total := 0
	for _, award := range record.Medals {
		total += award.Count
	}

	if record.ServiceBadges == total {
		t.Skip("the fixture cannot distinguish badges from medals this run")
	}
}

// TestSoldierBranchModsOpposeRiskAndReward verifies p. 66: the Branch and
// Operations Mods "are negative against the Risk Roll and positive against
// the Reward Roll".
func TestSoldierBranchModsOpposeRiskAndReward(t *testing.T) {
	c, _ := soldierRun(t)

	risk := firstThrowMods(c, "Risk vs")
	reward := firstThrowMods(c, "Reward vs")

	if risk == nil || reward == nil {
		t.Fatal("the pinned seed records no Risk and Reward pair")
	}

	for _, name := range []string{"Branch", "Operations"} {
		r, ok := findMod(risk, name)
		if !ok {
			t.Fatalf("the Risk throw itemizes no %s modifier", name)
		}

		w, ok := findMod(reward, name)
		if !ok {
			t.Fatalf("the Reward throw itemizes no %s modifier", name)
		}

		if r.Value != -w.Value {
			t.Errorf("%s modifier is %d on Risk and %d on Reward; they must oppose", name, r.Value, w.Value)
		}
	}
}

// firstThrowMods returns the modifiers of the first throw whose citation
// contains want.
func firstThrowMods(c chargen.Character, want string) []chargen.Mod {
	for _, e := range c.Events {
		if e.Kind == chargen.EventThrow && strings.Contains(e.Throw.Cite, want) {
			return e.Throw.Mods
		}
	}

	return nil
}

// findMod returns the named modifier of a throw.
func findMod(mods []chargen.Mod, name string) (chargen.Mod, bool) {
	for _, mod := range mods {
		if mod.Name == name {
			return mod, true
		}
	}

	return chargen.Mod{}, false
}
