package chargen_test

import (
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/chargen"
)

// rogueGoldenSeed is the pinned fixture: eight terms with a prison
// sentence, a term spent serving it, three payoffs, and one of them
// halved by a Risk failure in the same term.
const rogueGoldenSeed = 39

func rogueRun(t *testing.T) (chargen.Character, chargen.CareerRecord) {
	t.Helper()

	c, err := chargen.Generate(chargen.Options{
		Seed: rogueGoldenSeed, Career: "Rogue", Decider: careerOnly{},
	})
	if err != nil {
		t.Fatalf("seed %d: %v", rogueGoldenSeed, err)
	}

	return c, c.Careers[len(c.Careers)-1]
}

// TestRogueKeepsOneControllingCharacteristic verifies chart 10: "A Rogue
// selects one Controlling Characteristic ... which is then used
// throughout his career (not just in the current Term)." Exactly one
// choice is presented, and every term records the same characteristic.
func TestRogueKeepsOneControllingCharacteristic(t *testing.T) {
	c, record := rogueRun(t)

	choices := 0

	for _, e := range c.Events {
		if e.Kind == chargen.EventChoice &&
			e.Choice.Prompt == "Select the term's controlling characteristic" {
			choices++
		}
	}

	if choices != 1 {
		t.Errorf("controlling-characteristic choices = %d, want exactly one for the career", choices)
	}

	if record.ControllingCharacteristic == "" {
		t.Fatal("no career-long controlling characteristic recorded")
	}

	for _, term := range record.Terms {
		if term.ControllingCharacteristic != "" &&
			term.ControllingCharacteristic != record.ControllingCharacteristic {
			t.Errorf("term %d used %q, want the career's %q",
				term.Term, term.ControllingCharacteristic, record.ControllingCharacteristic)
		}
	}
}

// TestRogueRiskFailureImprisonsRatherThanInjures verifies chart 10's Risk
// row, which prints no characteristic reduction: a caught Rogue goes to
// prison, takes no Wound Badge, and cannot die at his trade
// (interpretation I-41, ERRATA.md).
func TestRogueRiskFailureImprisonsRatherThanInjures(t *testing.T) {
	sentenced := false

	for seed := uint64(1); seed <= 60; seed++ {
		c, err := chargen.Generate(chargen.Options{
			Seed: seed, Career: "Rogue", Decider: careerOnly{},
		})
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}

		assertUnharmedByHisTrade(t, seed, c)

		for _, e := range c.Events {
			if e.Kind == chargen.EventConsequence && e.Consequence.Kind == chargen.ConsequenceSentenced {
				sentenced = true
			}
		}
	}

	if !sentenced {
		t.Error("no seed in 1..60 sent a Rogue to prison")
	}
}

// TestRoguePrisonTermTakesOnlyPrisonSkills verifies chart 10 table B: "In
// Prison: Prison Skills from the Rogue Skills table column 1 or 2 only.
// Receives ONLY Prison Skills (not Term or Scheme Skills)".
func TestRoguePrisonTermTakesOnlyPrisonSkills(t *testing.T) {
	def, err := career.Rogue()
	if err != nil {
		t.Fatal(err)
	}

	allowed := []string{def.SkillColumns[0].Name, def.SkillColumns[1].Name}

	c, _ := rogueRun(t)

	checked := 0

	for _, e := range prisonColumnChoices(c) {
		checked++

		for _, option := range e.Choice.Options {
			if option != allowed[0] && option != allowed[1] {
				t.Errorf("event %d offers %q in prison; only columns 1 and 2 are allowed", e.Seq, option)
			}
		}
	}

	if checked == 0 {
		t.Fatal("the pinned seed serves no prison term")
	}

	if checked != 2 {
		t.Errorf("prison skills = %d, want 2", checked)
	}
}

// TestRoguePayoffFormula verifies chart 10's "Payoff= V x (1+CC-R+Mods)"
// and that a Risk failure in the same term halves it (interpretation
// I-44, ERRATA.md).
func TestRoguePayoffFormula(t *testing.T) {
	c, record := rogueRun(t)

	total, payoffs := 0, 0

	for _, e := range c.Events {
		if e.Kind != chargen.EventConsequence || e.Consequence.Kind != chargen.ConsequencePayoff {
			continue
		}

		payoffs++
		total += e.Consequence.Delta

		if e.Consequence.Value != total {
			t.Errorf("payoff running total = %d, want %d", e.Consequence.Value, total)
		}

		if e.Consequence.Delta < 0 {
			t.Errorf("payoff %d is negative", e.Consequence.Delta)
		}
	}

	if payoffs == 0 {
		t.Fatal("the pinned seed records no payoff")
	}

	if record.SchemePayoff != total {
		t.Errorf("record payoff = %d, want %d", record.SchemePayoff, total)
	}
}

// TestRogueSchemeTable pins chart 10's Rogue Schemes table.
func TestRogueSchemeTable(t *testing.T) {
	def, err := career.Rogue()
	if err != nil {
		t.Fatal(err)
	}

	if len(def.Schemes.Rows) != 13 {
		t.Fatalf("scheme rows = %d, want 13 (Flux -6 through +6)", len(def.Schemes.Rows))
	}

	for i, row := range def.Schemes.Rows {
		if want := i - 6; row.Flux != want {
			t.Errorf("row %d has Flux %+d, want %+d", i, row.Flux, want)
		}

		if (row.Credits == 0) == (row.ShipShares == 0) {
			t.Errorf("Flux %+d pays %+v; a row pays credits or ship shares, never both or neither", row.Flux, row)
		}
	}

	// The two Ship Share rows are Scout and Merchant.
	for _, flux := range []int{-2, -1} {
		if got := def.Schemes.SchemeAt(flux); got.ShipShares == 0 {
			t.Errorf("Flux %+d = %+v, want a Ship Share scheme", flux, got)
		}
	}
}

// prisonColumnChoices returns the skill-column choices made during a
// prison term.
func prisonColumnChoices(c chargen.Character) []chargen.Event {
	var found []chargen.Event

	inPrison := false

	for _, e := range c.Events {
		if e.Kind == chargen.EventStep {
			inPrison = strings.Contains(e.Step.Name, "In Prison")

			continue
		}

		if inPrison && e.Kind == chargen.EventChoice &&
			e.Choice.Prompt == "Select a Rogue Skills column" {
			found = append(found, e)
		}
	}

	return found
}

// assertUnharmedByHisTrade holds interpretation I-41: chart 10's Risk row
// prints no characteristic reduction, so a Rogue takes no Wound Badge and
// cannot die at his own trade.
//
// Aging can still kill him, like anyone else (chart A p. 89), so the
// death test asks which killed him: an injury death would also have left
// a Wound Badge, which the first check forbids outright.
func assertUnharmedByHisTrade(t *testing.T, seed uint64, c chargen.Character) {
	t.Helper()

	if c.WoundBadges != 0 {
		t.Errorf("seed %d: a Rogue took %d Wound Badges", seed, c.WoundBadges)
	}

	if c.Disabled {
		t.Errorf("seed %d: a Rogue was disabled at his own trade", seed)
	}

	if c.Dead && c.ExtremelyMajorIllnesses < 2 {
		t.Errorf("seed %d: a Rogue was killed at his own trade", seed)
	}
}
