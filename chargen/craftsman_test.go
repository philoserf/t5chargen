package chargen_test

import (
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/chargen"
)

// craftsmanPath reaches chart 01, which no auto-generated character does.
// Entry is "Automatic* — *if TWO skill-6 and Craftsman-1" (p. 75), so the
// character must first acquire the craft and the breadth, which takes
// several terms in another career; the auto policy meanwhile declines
// every career change (POLICY.md `change_career`).
//
// It takes the craft wherever a Trade is offered, changes careers once it
// has served a few terms, and settles as soon as Craftsman appears.
type craftsmanPath struct {
	offers  int
	arrived bool
}

const craftsmanPathWait = 4

func (d *craftsmanPath) Choose(c chargen.Choice) (int, error) {
	//nolint:exhaustive // Deliberately partitioned: the rest defer to the auto policy.
	switch c.ID {
	case chargen.ChooseCareerChange:
		d.offers++

		return d.changeCareer(), nil
	case chargen.ChooseCareer:
		return d.pickCareer(c)
	case chargen.ChooseSkill, chargen.ChooseTrade:
		if i := indexOf(c.Options, "Craftsman"); i >= 0 {
			return i, nil
		}
	case chargen.ChooseSkillColumn:
		if i := columnLike(c.Options, "Academic", "Vocation"); i >= 0 {
			return i, nil
		}
	default:
	}

	return autoPolicy(c)
}

func indexOf(options []string, want string) int {
	for i, option := range options {
		if option == want {
			return i
		}
	}

	return -1
}

func (*craftsmanPath) Kind() chargen.DeciderKind { return chargen.DeciderPlayer }

// pickCareer takes Craftsman the moment it is offered and otherwise builds
// toward it in a Citizen career, whose table C offers Trades.
func (d *craftsmanPath) pickCareer(c chargen.Choice) (int, error) {
	if i := indexOf(c.Options, "Craftsman"); i >= 0 {
		d.arrived = true

		return i, nil
	}

	if i := indexOf(c.Options, "Citizen"); i >= 0 {
		return i, nil
	}

	return autoPolicy(c)
}

// craftsmanRun generates the pinned Craftsman character: five terms
// producing four Masterpieces, two of them Perfect, and one term that
// could not attempt at all. The rolled-failure branch of the creation
// throw is not among them — all four throws succeed — so a reader of the
// fixture should not take it for the "Creation Attempt Fails" case.
func craftsmanRun(t *testing.T) (chargen.Character, chargen.CareerRecord) {
	t.Helper()

	c, err := chargen.Generate(chargen.Options{Seed: 177, Decider: &craftsmanPath{}})
	if err != nil {
		t.Fatal(err)
	}

	for _, record := range c.Careers {
		if record.Career == "Craftsman" && record.Began {
			return c, record
		}
	}

	t.Fatal("seed 177 no longer reaches Craftsman; find and pin another seed")

	return c, chargen.CareerRecord{}
}

// TestCraftsmanGolden pins the full record.
func TestCraftsmanGolden(t *testing.T) {
	c, _ := craftsmanRun(t)
	goldenJSON(t, c, "testdata/career_craftsman.json")
}

// TestCraftsmanCoversEveryCreationBranch guards the fixture's reach: a
// success, a failure, a Perfect Masterpiece and an ordinary one. A seed
// that stopped covering them would leave the branches untested while the
// golden still passed.
func TestCraftsmanCoversEveryCreationBranch(t *testing.T) {
	_, record := craftsmanRun(t)

	switch {
	case record.Masterpieces == 0:
		t.Error("no Masterpiece created")
	case record.PerfectMasterpieces == 0:
		t.Error("no Perfect Masterpiece")
	case record.PerfectMasterpieces == record.Masterpieces:
		t.Error("every Masterpiece is Perfect; the ordinary branch is untested")
	case record.Masterpieces >= len(record.Terms):
		t.Error("every attempt succeeded; the failure branch is untested")
	}
}

// vocationPath reaches chart 01's Vocation column, which craftsmanPath
// never rolls on: it prefers Academic, and Academic is offered first. The
// two "New Trade***" cells live in Vocation, so without this the only new
// cell kind the chart introduced would ship unexercised.
//
// The preference is applied only inside the Craftsman career, so the path
// to it — and every roll before it — is the pinned character's.
type vocationPath struct {
	craftsmanPath
}

func (d *vocationPath) Choose(c chargen.Choice) (int, error) {
	if c.ID == chargen.ChooseSkillColumn && strings.Contains(c.Prompt, "Craftsman") {
		if i := columnLike(c.Options, "Vocation"); i >= 0 {
			return i, nil
		}
	}

	return d.craftsmanPath.Choose(c)
}

// TestCraftsmanNewTradeCell pins chart 01's "New Trade***: Any Trade not
// already held; if all are already held; this benefit is lost" (p. 75):
// the cell offers only Trades the character lacks, and awards one.
func TestCraftsmanNewTradeCell(t *testing.T) {
	c, err := chargen.Generate(chargen.Options{Seed: 177, Decider: &vocationPath{}})
	if err != nil {
		t.Fatal(err)
	}

	choices := 0

	for _, e := range c.Events {
		if e.Kind != chargen.EventChoice || !strings.Contains(e.Choice.Cite, "New Trade") {
			continue
		}

		choices++

		for _, option := range e.Choice.Options {
			if option == "Craftsman" {
				t.Error("New Trade offered Craftsman, which every Craftsman already holds")
			}
		}
	}

	if choices == 0 {
		t.Fatal("the Vocation column never reached a New Trade cell; find another seed")
	}
}

// TestCraftsmanNeedsItsPrerequisite pins "To Begin Automatic* — *if TWO
// skill-6 and Craftsman-1" (p. 75). There is no throw: a character who
// does not qualify is never offered the career, so no auto-generated
// character enters it at all.
func TestCraftsmanNeedsItsPrerequisite(t *testing.T) {
	def, err := career.Craftsman()
	if err != nil {
		t.Fatal(err)
	}

	if def.BeginPrerequisite == nil {
		t.Fatal("chart 01 entry is conditional and the definition says otherwise")
	}

	if p := def.BeginPrerequisite; p.Skill != "Craftsman" || p.SkillLevel != 1 ||
		p.SkillsAtLevel != 6 || p.SkillsAtLevelCount != 2 {
		t.Errorf("prerequisite %+v, want Craftsman-1 and two skills at 6", p)
	}

	// Chart 01 prints no "never a first career" box, unlike chart 13:
	// the prerequisite is what keeps a school leaver out, not a rule.
	if def.NotAFirstCareer {
		t.Error("chart 01 bars no first career; only its prerequisite does")
	}

	assertNoAutoCraftsman(t)
}

// assertNoAutoCraftsman holds the practical consequence of I-60: the
// prerequisite is out of reach for a character leaving education, so the
// auto policy never enters the career.
func assertNoAutoCraftsman(t *testing.T) {
	t.Helper()

	for seed := uint64(1); seed <= 200; seed++ {
		c := generate(t, chargen.Options{Seed: seed})
		for _, record := range c.Careers {
			if record.Career == "Craftsman" {
				t.Fatalf("seed %d: an auto-generated character reached Craftsman", seed)
			}
		}
	}
}

// TestMasterPointsTotalTheChartsSources pins what a Craftsman brings to a
// Masterpiece: "Controlling Characteristics / Craftsman Skill / Up to FIVE
// Skills at level 6+ (or Knowledges at level-6) (but not languages)"
// (chart 01 p. 75), read as all four characteristics (interpretation I-61).
func TestMasterPointsTotalTheChartsSources(t *testing.T) {
	c, _ := craftsmanRun(t)

	def, err := career.Craftsman()
	if err != nil {
		t.Fatal(err)
	}

	checked := 0

	for _, e := range c.Events {
		if e.Kind != chargen.EventThrow || !strings.Contains(e.Throw.Cite, "Masterpiece vs Master Points") {
			continue
		}

		checked++

		assertCreationThrow(t, e.Throw, def.Masterpiece)
	}

	if checked == 0 {
		t.Fatal("the pinned seed makes no creation throw")
	}
}

// TestMasterpieceValue pins the price the page states: "The Masterpiece
// can be sold at Cr150,000 plus Cr10,000 per Master Point over 40. A
// Perfect Masterpiece (=55 points or more) sells for Double" — and the
// printed Masterpiece Value table it summarises.
func TestMasterpieceValue(t *testing.T) {
	c, _ := craftsmanRun(t)

	// Rows read off the printed table (chart 01 p. 75).
	want := map[int]int{40: 150_000, 41: 160_000, 54: 290_000, 55: 600_000, 63: 760_000}

	for _, e := range c.Events {
		if e.Kind != chargen.EventConsequence || e.Consequence.Kind != chargen.ConsequenceMasterpiece {
			continue
		}

		points, value := e.Consequence.Value, e.Consequence.Delta
		if expected, ok := want[points]; ok && value != expected {
			t.Errorf("%d Master Points valued at Cr%d, the table prints Cr%d", points, value, expected)
		}

		// The doubling boundary is the rule the table encodes.
		base := 150_000 + 10_000*(points-40)
		if points >= 55 {
			base *= 2
		}

		if value != base {
			t.Errorf("%d Master Points valued at Cr%d, want Cr%d", points, value, base)
		}
	}
}

// assertCreationThrow checks one creation throw against the chart: the
// target is the itemised Master Points, it clears the chart's floor, and
// it is rolled on the chart's dice.
func assertCreationThrow(t *testing.T, throw *chargen.ThrowEvent, box *career.Masterpiece) {
	t.Helper()

	sum := 0
	for _, mod := range throw.Mods {
		sum += mod.Value
	}

	if throw.Target == nil || *throw.Target != sum {
		t.Errorf("Master Points target %v does not equal its itemised sources %d", throw.Target, sum)

		return
	}

	if *throw.Target < box.MinimumPoints {
		t.Errorf("attempted a Masterpiece on %d points, below the chart's %d", *throw.Target, box.MinimumPoints)
	}

	if len(throw.Dice) != box.Dice {
		t.Errorf("creation rolled %d dice, want %d", len(throw.Dice), box.Dice)
	}
}

// changeCareer accepts a change once the character has served a few terms,
// and declines once he has arrived in the craft.
func (d *craftsmanPath) changeCareer() int {
	if d.arrived || d.offers < craftsmanPathWait {
		return 0
	}

	return 1
}

// columnLike picks the first skills-table column whose name contains one
// of the given words.
func columnLike(options []string, words ...string) int {
	for i, option := range options {
		for _, word := range words {
			if strings.Contains(option, word) {
				return i
			}
		}
	}

	return -1
}
