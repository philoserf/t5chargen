package chargen_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/chargen"
)

// The pinned Noble fixtures, found offline: seed 2978 climbs through both
// an elevation that raises Social Standing and the title-only step above
// it, invokes the once-per-career Flux, and is exiled once; seed 1060 is
// exiled too, and pins the Return path on its own.
const (
	nobleGoldenSeed = 2978
	nobleExileSeed  = 1060
)

func nobleRun(t *testing.T, seed uint64) (chargen.Character, chargen.CareerRecord) {
	t.Helper()

	c, open := generateIfOpen(t, chargen.Options{
		Seed: seed, Career: "Noble", Decider: careerOnly{},
	})
	// Guard the premise: nobleRun is called only with pinned seeds, so a
	// seed that stops qualifying is a fixture to repair, not a run to
	// skip. Skipping would leave four tests reporting green while
	// asserting nothing.
	if !open {
		t.Fatalf("seed %d falls below chart 11's Soc B+ prerequisite; re-pin the fixture", seed)
	}

	if len(c.Careers) == 0 {
		t.Fatalf("seed %d: no career record", seed)
	}

	return c, c.Careers[len(c.Careers)-1]
}

// TestNobleLadder pins the chart 11 rank table, including the rungs that
// share a Social Standing (p. 68).
func TestNobleLadder(t *testing.T) {
	def, err := career.Noble()
	if err != nil {
		t.Fatal(err)
	}

	want := []struct {
		title string
		soc   int
	}{
		{title: "Gentleman", soc: 10},
		{title: "Knight", soc: 11},
		{title: "Baronet", soc: 12},
		{title: "Baron", soc: 12},
		{title: "Marquis", soc: 13},
		{title: "Viscount", soc: 14},
		{title: "Count", soc: 14},
		{title: "Duke", soc: 15},
		{title: "Duke", soc: 15},
		{title: "Archduke", soc: 16},
		{title: "Imperial Family", soc: 16},
		{title: "Emperor", soc: 17},
	}

	if len(def.Ranks) != len(want) {
		t.Fatalf("ranks = %d, want %d", len(def.Ranks), len(want))
	}

	for i, w := range want {
		if def.Ranks[i].Title != w.title || def.Ranks[i].Soc != w.soc {
			t.Errorf("rung %d = %q at Soc %d, want %q at %d",
				i, def.Ranks[i].Title, def.Ranks[i].Soc, w.title, w.soc)
		}
	}
}

// TestNobleRequiresSocB verifies the chart 11 prerequisite: "To Begin
// Automatic* ... *if Soc B+".
//
// Measured at the offer, because that is where p. 65 puts it — "Some
// Careers have requirements before a character may attempt to Begin" —
// and where a player meets it. It used to be measured at the outcome, on
// a character who had already chosen the career and was then told he
// could not have it; the menu offered every one of them a title he had no
// way to hold (interpretation I-28).
func TestNobleRequiresSocB(t *testing.T) {
	below, above := 0, 0

	for seed := uint64(1); seed <= 60; seed++ {
		peek := &careerMenu{}

		c, err := chargen.Generate(chargen.Options{Seed: seed, CurrentYear: 1105, Decider: peek})
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}

		offered := slices.Contains(peek.first, "Noble")

		switch soc := c.Characteristics.Soc; {
		case soc < 11:
			below++

			if offered {
				t.Errorf("seed %d: Noble offered at Soc %d, below chart 11's B+", seed, soc)
			}
		default:
			above++

			if !offered {
				t.Errorf("seed %d: Noble withheld at Soc %d, which meets chart 11's B+", seed, soc)
			}
		}
	}

	if below == 0 || above == 0 {
		t.Fatalf("the sweep saw %d below and %d above the prerequisite; it is asserting one case only",
			below, above)
	}
}

// careerMenu records the first Select Career offer.
type careerMenu struct {
	playerKind

	first []string
}

func (d *careerMenu) Choose(c chargen.Choice) (int, error) {
	if c.ID == chargen.ChooseCareer && d.first == nil {
		d.first = slices.Clone(c.Options)
	}

	return autoPolicy(c)
}

// TestNobleEntersAtLowerTitle verifies that a Social Standing shared by
// two rungs enters at the lower title: "A character elevated to Soc = c
// (lower case) is initially a Baronet" (p. 68; interpretation I-29).
func TestNobleEntersAtLowerTitle(t *testing.T) {
	c, _ := nobleRun(t, nobleGoldenSeed)

	if c.Characteristics.Soc != 12 {
		t.Fatalf("the pinned seed ends at Soc %d; the fixture expects 12", c.Characteristics.Soc)
	}

	titles := rankTitles(c)
	if len(titles) == 0 {
		t.Fatal("no rank recorded")
	}

	// Whatever the entry rung, no Baron may appear before a Baronet.
	baronet, baron := slices.Index(titles, "Baronet"), slices.Index(titles, "Baron")
	if baron >= 0 && (baronet < 0 || baronet > baron) {
		t.Errorf("rank history %v reaches Baron without passing Baronet", titles)
	}
}

// TestNobleElevationRaisesSocOnlyWhereTheLadderDoes verifies chart 11's
// "increase in Social Standing (if any)": an elevation between two rungs
// sharing a Social Standing raises the title alone, and awards no Land
// Grant, since "Each increase in Soc during CharGen awards a Land Grant".
func TestNobleElevationRaisesSocOnlyWhereTheLadderDoes(t *testing.T) {
	c, record := nobleRun(t, nobleGoldenSeed)

	elevations, grants := 0, 0

	for _, e := range c.Events {
		if e.Kind != chargen.EventConsequence {
			continue
		}

		if e.Consequence.Kind == chargen.ConsequenceElevated {
			elevations++
		}

		if e.Consequence.Kind == chargen.ConsequenceLandGrant {
			grants++
		}
	}

	if elevations < 2 {
		t.Fatalf("the pinned seed records %d elevations; the fixture expects at least two", elevations)
	}

	if grants >= elevations {
		t.Errorf("%d elevations awarded %d Land Grants; a title-only step awards none", elevations, grants)
	}

	if record.LandGrants != grants {
		t.Errorf("record LandGrants = %d, want %d", record.LandGrants, grants)
	}
}

// TestNobleExileGovernsWhichRollHappens verifies chart 11's "Return: Roll
// only if in Exile" and "Intrigue: Roll only if not in Exile".
func TestNobleExileGovernsWhichRollHappens(t *testing.T) {
	c, _ := nobleRun(t, nobleExileSeed)

	exiled := false
	sawReturn := false

	for _, e := range c.Events {
		switch nobleEventKind(e) {
		case "intrigue-roll":
			if exiled {
				t.Errorf("event %d rolls Intrigue while in Exile", e.Seq)
			}
		case "return-roll":
			sawReturn = true

			if !exiled {
				t.Errorf("event %d rolls Return while not in Exile", e.Seq)
			}
		case "exiled":
			exiled = true
		case "returned":
			exiled = false
		}
	}

	if !sawReturn {
		t.Fatal("the pinned exile seed records no Return roll")
	}
}

// TestNobleElevationRollsHigh verifies that Elevation is a Roll High
// against Soc (p. 66): the recorded throws that succeed are those at or
// above the target, the reverse of every other career throw.
func TestNobleElevationRollsHigh(t *testing.T) {
	c, _ := nobleRun(t, nobleGoldenSeed)

	checked := 0

	for _, e := range c.Events {
		if e.Kind != chargen.EventThrow || !strings.Contains(e.Throw.Cite, "Elevation: Roll High") {
			continue
		}

		checked++

		if e.Throw.Target == nil || e.Throw.Success == nil {
			t.Fatalf("event %d records an Elevation with no target or outcome", e.Seq)
		}

		if want := e.Throw.Total >= *e.Throw.Target; *e.Throw.Success != want {
			t.Errorf("event %d: total %d vs target %d recorded success %v; Roll High wants %v",
				e.Seq, e.Throw.Total, *e.Throw.Target, *e.Throw.Success, want)
		}
	}

	if checked == 0 {
		t.Fatal("the pinned seed records no Elevation roll")
	}
}

// yearsElapsedInCareer reports whether any year elapsed after the Noble
// career began. Education charges a year for a refused admission, which

// nobleEventKind classifies the events the exile state machine turns on.
//
//nolint:exhaustive // Only the two exile consequences matter here.
func nobleEventKind(e chargen.Event) string {
	if e.Kind == chargen.EventThrow {
		switch {
		case strings.Contains(e.Throw.Cite, "Intrigue vs"):
			return "intrigue-roll"
		case strings.Contains(e.Throw.Cite, "Return vs"):
			return "return-roll"
		}

		return ""
	}

	if e.Kind != chargen.EventConsequence {
		return ""
	}

	switch e.Consequence.Kind {
	case chargen.ConsequenceExiled:
		return "exiled"
	case chargen.ConsequenceReturned:
		return "returned"
	default:
		return ""
	}
}

// personalColumnPolicy is the default policy with one override: it always
// takes chart 11 table C column 1 (Personal), whose line 6 is "C6 +1". The
// default policy prefers the General column, so the Soc-raising cell is
// unreachable without this. The embedded DefaultPolicy is what supplies
// Kind, so the run still attests as policy-decided.
type personalColumnPolicy struct{ chargen.DefaultPolicy }

func (personalColumnPolicy) Choose(c chargen.Choice) (int, error) {
	if c.ID == chargen.ChooseSkillColumn {
		for i, option := range c.Options {
			if option == "Personal" {
				return i, nil
			}
		}
	}

	return autoPolicy(c)
}

// TestNobleLandGrantPerSocIncrease verifies chart 11's "Land Grants. Each
// increase in Soc during CharGen awards a Land Grant": a Soc increase from
// table C earns one just as Elevation's does (interpretation I-30).
func TestNobleLandGrantPerSocIncrease(t *testing.T) {
	checked := 0

	for seed := uint64(1); seed <= 200; seed++ {
		c, open := generateIfOpen(t, chargen.Options{
			Seed: seed, Career: "Noble", Decider: personalColumnPolicy{},
		})
		if !open {
			continue // below chart 11's Soc B+ prerequisite (I-28)
		}

		if len(c.Careers) == 0 || !c.Careers[0].Began {
			continue
		}

		raises, grants := countSocRaisesAndGrants(c)
		if raises == 0 {
			continue
		}

		checked++

		if grants != raises {
			t.Errorf("seed %d: %d Soc increases awarded %d Land Grants; chart 11 awards one each",
				seed, raises, grants)
		}

		if c.Careers[0].LandGrants != grants {
			t.Errorf("seed %d: record LandGrants = %d, want %d", seed, c.Careers[0].LandGrants, grants)
		}
	}

	if checked == 0 {
		t.Fatal("no seed in 1..200 raised a Noble's Soc from table C")
	}
}

// countSocRaisesAndGrants counts the Soc increases and the Land Grant
// consequences in a generation record.
func countSocRaisesAndGrants(c chargen.Character) (int, int) {
	raises, grants := 0, 0

	for _, e := range c.Events {
		if e.Kind != chargen.EventConsequence {
			continue
		}

		if e.Consequence.Kind == chargen.ConsequenceCharacteristicChange &&
			e.Consequence.Characteristic == "Soc" && e.Consequence.Delta > 0 {
			raises++
		}

		if e.Consequence.Kind == chargen.ConsequenceLandGrant {
			grants++
		}
	}

	return raises, grants
}
