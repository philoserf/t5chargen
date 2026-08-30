package chargen_test

import (
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/chargen"
	"github.com/philoserf/t5chargen/world"
)

// TestRolledHomeworldComesFromChartB verifies "Select or determine a
// Homeworld" (p. 56): determining it draws a world off chart B's list
// rather than taking the assigned one.
func TestRolledHomeworldComesFromChartB(t *testing.T) {
	worlds, err := world.ChartB()
	if err != nil {
		t.Fatal(err)
	}

	onChart := map[string]bool{}
	for _, w := range worlds {
		onChart[w.Name] = true
	}

	seen := map[string]bool{}

	for seed := range uint64(40) {
		c := generate(t, chargen.Options{Seed: seed, RollHomeworld: true})

		if !onChart[c.Homeworld.Name] {
			t.Fatalf("seed %d rolled %q, which is not on chart B", seed, c.Homeworld.Name)
		}

		seen[c.Homeworld.Name] = true
	}

	// A stuck roll would give every character the same world and still
	// pass the membership check above.
	if len(seen) < 2 {
		t.Errorf("40 seeds produced %d distinct worlds; the roll is not varying", len(seen))
	}
}

// TestRolledHomeworldReadsTheDiceInOrder verifies the chart is indexed D1
// then D2 as it prints them: cell 3 4 is Regina and cell 4 3 is Uakye, so
// the order is the answer.
//
// Checking that the rolled world is somewhere on chart B cannot see this —
// a swapped pair still lands on a real world. The two throws are read back
// out of the record and the cell they name is compared to the world the
// run actually took.
func TestRolledHomeworldReadsTheDiceInOrder(t *testing.T) {
	checked := 0

	for seed := range uint64(30) {
		c := generate(t, chargen.Options{Seed: seed, RollHomeworld: true})

		dice := chartBThrows(c)
		if len(dice) != 2 {
			t.Fatalf("seed %d logged %d chart B throws, want 2", seed, len(dice))
		}

		want, err := world.At(dice[0], dice[1])
		if err != nil {
			t.Fatal(err)
		}

		if c.Homeworld.Name != want.Name {
			t.Errorf("seed %d rolled %d then %d and took %q, but cell %d %d is %q",
				seed, dice[0], dice[1], c.Homeworld.Name, dice[0], dice[1], want.Name)
		}

		if dice[0] != dice[1] {
			checked++ // a matched pair proves nothing about order
		}
	}

	if checked == 0 {
		t.Error("every seed rolled a matched pair; the order is not being tested")
	}
}

// chartBThrows returns the faces of the two chart B world-list rolls.
func chartBThrows(c chargen.Character) []int {
	var faces []int

	for _, event := range c.Events {
		if event.Kind == chargen.EventThrow && strings.Contains(event.Throw.Cite, "chart B (Select a Homeworld") {
			faces = append(faces, event.Throw.Total)
		}
	}

	return faces
}

// TestRolledHomeworldIsRecordedAsAnInput verifies the record carries that
// the world was determined rather than assigned. The two dice come out of
// the seeded stream, so a replay that did not know to roll them would
// diverge from the next throw onward — the same class of gap the career
// force had before the inputs block existed.
func TestRolledHomeworldIsRecordedAsAnInput(t *testing.T) {
	rolled := generate(t, chargen.Options{Seed: 1, RollHomeworld: true})
	if !rolled.Inputs.RolledHomeworld {
		t.Error("a determined homeworld is not recorded as one")
	}

	assigned := generate(t, chargen.Options{Seed: 1})
	if assigned.Inputs.RolledHomeworld {
		t.Error("an assigned homeworld is recorded as determined")
	}

	// Not a Skip: seed 1 rolls Yori against the default Regina, so the
	// two runs are distinguishable, and a skip here would hide the day
	// the flag stopped doing anything.
	if rolled.Homeworld.Name == assigned.Homeworld.Name {
		t.Errorf("determined and assigned both gave %q; the flag changed nothing", rolled.Homeworld.Name)
	}
}

// TestRolledHomeworldReplays verifies a determined homeworld survives the
// replay contract. No golden fixture rolls one, so TestReplayRoundTrip
// cannot reach this path.
func TestRolledHomeworldReplays(t *testing.T) {
	for seed := range uint64(10) {
		c := generate(t, chargen.Options{Seed: seed, RollHomeworld: true})

		if _, err := chargen.Replay(c); err != nil {
			t.Fatalf("seed %d: a determined homeworld does not replay: %v", seed, err)
		}
	}
}

// TestDeepSpaceBirthGrantsItsSkills verifies the one cell that names no
// world still grants what chart B gives its trade classification: "Ds Deep
// Space — Vacc Suit +Zero-G" (p. 56, interpretation I-97).
func TestDeepSpaceBirthGrantsItsSkills(t *testing.T) {
	deepSpace, err := world.At(6, 6)
	if err != nil {
		t.Fatal(err)
	}

	c := generate(t, chargen.Options{Seed: 1, Homeworld: deepSpace})

	for _, want := range []string{"Vacc Suit", "Zero-G"} {
		held := false

		for _, skill := range c.Skills {
			if skill.Name == want {
				held = true
			}
		}

		if !held {
			t.Errorf("a deep space birth did not grant %s", want)
		}
	}
}

// TestAnUnassignedHomeworldIsChosen verifies chart E1 step B offers what
// it says: "As assigned, selected, or random" (p. 58).
//
// Selecting was the one of the three the engine could not do. A homeworld
// nobody assigned was still Regina, presented as the only option, so the
// thirty-four worlds transcribed from chart B were unreachable except by
// rolling.
func TestAnUnassignedHomeworldIsChosen(t *testing.T) {
	worlds, err := world.Selectable()
	if err != nil {
		t.Fatal(err)
	}

	offered := homeworldChoice(t, generate(t, chargen.Options{Seed: 1}))

	if len(offered.Options) != len(worlds) {
		t.Errorf("offered %d worlds, want chart B's %d", len(offered.Options), len(worlds))
	}

	// Each named once. Regina fills three of the thirty-six cells so that
	// rolling lands on it three times as often; a list to choose from
	// wants it once.
	seen := map[string]bool{}
	for _, option := range offered.Options {
		if seen[option] {
			t.Errorf("%q is offered twice", option)
		}

		seen[option] = true
	}
}

// TestAnAssignedHomeworldIsNotAChoice verifies the other half: a caller
// that named a homeworld has settled it, and the choice holds only that
// one — the same shape --career gives the first career.
func TestAnAssignedHomeworldIsNotAChoice(t *testing.T) {
	for _, opts := range []chargen.Options{
		{Seed: 1, Homeworld: world.Homeworld{UWP: "A788899-C", TradeClassifications: []string{"Ri"}}},
		{Seed: 1, RollHomeworld: true},
	} {
		offered := homeworldChoice(t, generate(t, opts))
		if len(offered.Options) != 1 {
			t.Errorf("a settled homeworld offered %d options, want 1", len(offered.Options))
		}
	}
}

// TestTheChosenHomeworldIsTheOneUsed verifies the answer reaches the
// character: chart B grants "one specified skill for each Trade
// Classification" (p. 58), so choosing a different world must grant
// different skills.
func TestTheChosenHomeworldIsTheOneUsed(t *testing.T) {
	worlds, err := world.Selectable()
	if err != nil {
		t.Fatal(err)
	}

	var earth int

	for i, w := range worlds {
		if w.Name == "Earth" {
			earth = i
		}
	}

	c := generate(t, chargen.Options{Seed: 1, Decider: pickHomeworld{index: earth}})
	if c.Homeworld.Name != "Earth" {
		t.Fatalf("chose Earth and got %q", c.Homeworld.Name)
	}

	if got := c.Homeworld.TradeClassifications; len(got) == 0 {
		t.Error("the chosen world brought no trade classifications, so it grants no skills")
	}
}

// TestThePolicyAssignsRatherThanPicks verifies POLICY.md's rule survives
// the wider list. First-listed on chart B is Alell; the policy assigns the
// tool-owned default instead, so no auto-generated character moves house.
func TestThePolicyAssignsRatherThanPicks(t *testing.T) {
	home, err := world.Default()
	if err != nil {
		t.Fatal(err)
	}

	for seed := range uint64(20) {
		if got := generate(t, chargen.Options{Seed: seed}).Homeworld.Name; got != home.Name {
			t.Fatalf("seed %d: the policy took %q, want the default %q", seed, got, home.Name)
		}
	}
}

// homeworldChoice returns the record's homeworld choice event.
func homeworldChoice(t *testing.T, c chargen.Character) chargen.ChoiceEvent {
	t.Helper()

	for _, event := range c.Events {
		if event.Kind == chargen.EventChoice && event.Choice.Prompt == "Select a homeworld" {
			return *event.Choice
		}
	}

	t.Fatal("the record holds no homeworld choice")

	return chargen.ChoiceEvent{}
}

// pickHomeworld chooses one world off chart B and defers the rest.
type pickHomeworld struct {
	playerKind

	index int
}

func (d pickHomeworld) Choose(c chargen.Choice) (int, error) {
	if c.ID == chargen.ChooseHomeworld {
		return d.index, nil
	}

	return autoPolicy(c)
}
