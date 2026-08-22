package chargen_test

import (
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/chargen"
)

// The pinned Scholar fixtures, found offline: seed 23 is a degreed
// Scholar reaching Distinguished Professor with Tenure and two
// Award-Winning publications; seed 62 is a degreeless Amateur, Edu 4, who
// selects his own Major and Minor and can never be promoted.
const (
	scholarGoldenSeed  = 23
	scholarAmateurSeed = 62
)

func scholarRun(t *testing.T, seed uint64) (chargen.Character, chargen.CareerRecord) {
	t.Helper()

	c, err := chargen.Generate(chargen.Options{
		Seed: seed, Career: "Scholar", Decider: chargen.DefaultPolicy{},
	})
	if err != nil {
		t.Fatalf("seed %d: %v", seed, err)
	}

	if len(c.Careers) == 0 {
		t.Fatalf("seed %d: no career record", seed)
	}

	return c, c.Careers[len(c.Careers)-1]
}

// TestScholarRanks pins the chart 02 rank ladder.
func TestScholarRanks(t *testing.T) {
	def, err := career.Scholar()
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"Amateur", "Lecturer", "Instructor", "Assistant Professor",
		"Associate Professor", "Professor", "Distinguished Professor",
	}

	if len(def.Ranks) != len(want) {
		t.Fatalf("ranks = %d, want %d", len(def.Ranks), len(want))
	}

	for i, title := range want {
		if def.Ranks[i].Title != title {
			t.Errorf("rank %d = %q, want %q", i, def.Ranks[i].Title, title)
		}
	}
}

// TestScholarEduGatesEntry verifies the chart's entry gates: "A character
// with Edu 8+ is automatically Scholar1 to Begin"; a character with Edu 7
// or less rolls To Begin and enters as an Amateur.
func TestScholarEduGatesEntry(t *testing.T) {
	c, record := scholarRun(t, scholarGoldenSeed)
	if c.Characteristics.Edu < 8 {
		t.Fatalf("the degreed fixture has Edu %d; it should be 8+", c.Characteristics.Edu)
	}

	if first := firstRankSet(c); first != "Lecturer" {
		t.Errorf("Edu 8+ entered as %q, want Lecturer (automatically Scholar1)", first)
	}

	if !record.Began {
		t.Error("Edu 8+ entry is automatic but the career did not begin")
	}

	amateur, amateurRecord := scholarRun(t, scholarAmateurSeed)
	if amateur.Characteristics.Edu >= 8 {
		t.Fatalf("the Amateur fixture has Edu %d; it should be 7 or less", amateur.Characteristics.Edu)
	}

	if first := firstRankSet(amateur); first != "Amateur" {
		t.Errorf("Edu 7- entered as %q, want Amateur", first)
	}

	if amateurRecord.Rank != "S0" {
		t.Errorf("Amateur ended at %q; Edu 7- cannot be promoted", amateurRecord.Rank)
	}
}

// TestScholarDegreelessSelectsAreas verifies "Every Scholar has a Major
// and a Minor. If no degree ... then select any Skill or Knowledge from
// the Skills List" (chart 02; interpretation I-23).
func TestScholarDegreelessSelectsAreas(t *testing.T) {
	c, record := scholarRun(t, scholarAmateurSeed)

	if record.Major == "" || record.Minor == "" {
		t.Fatalf("degreeless Scholar has Major %q and Minor %q; the chart gives every Scholar both",
			record.Major, record.Minor)
	}

	if record.Major == record.Minor {
		t.Error("Major and Minor are the same; they cannot be (p. 59)")
	}

	// A degreed Scholar keeps the education's areas and selects none.
	_, degreed := scholarRun(t, scholarGoldenSeed)
	if degreed.Major != "" || degreed.Minor != "" {
		t.Errorf("degreed Scholar selected Major %q and Minor %q; the degree supplies them",
			degreed.Major, degreed.Minor)
	}

	_ = c
}

// TestScholarPublications verifies the Publication rules: a success adds
// one, and a roll four or more under the characteristic is Award-Winning
// and "counts as TWO" (chart 02; interpretation I-25).
func TestScholarPublications(t *testing.T) {
	c, record := scholarRun(t, scholarGoldenSeed)

	total, awards := 0, 0

	for _, e := range c.Events {
		if e.Kind != chargen.EventConsequence || e.Consequence.Kind != chargen.ConsequencePublication {
			continue
		}

		if e.Consequence.Delta != 1 && e.Consequence.Delta != 2 {
			t.Errorf("publication delta = %d, want 1 or 2", e.Consequence.Delta)
		}

		if e.Consequence.Delta == 2 {
			awards++
		}

		total += e.Consequence.Delta

		if e.Consequence.Value != total {
			t.Errorf("publication running total = %d, want %d", e.Consequence.Value, total)
		}
	}

	if awards == 0 {
		t.Error("the pinned seed records no Award-Winning publication")
	}

	if record.Publications != total {
		t.Errorf("record Publications = %d, want %d", record.Publications, total)
	}
}

// TestScholarTenureGatesPromotion verifies "Promotion beyond Scholar3 not
// possible without Tenure" and that Tenure needs Edu 10+ at Scholar3.
func TestScholarTenureGatesPromotion(t *testing.T) {
	c, record := scholarRun(t, scholarGoldenSeed)

	if !record.Tenured {
		t.Fatal("the pinned seed is expected to gain Tenure")
	}

	tenureSeq, beyondThird := tenureAndPromotionSeqs(c)

	if beyondThird == 0 {
		t.Fatal("the pinned seed never promotes beyond Assistant Professor")
	}

	if tenureSeq == 0 || tenureSeq > beyondThird {
		t.Errorf("promoted beyond Scholar3 at event %d before Tenure at %d", beyondThird, tenureSeq)
	}

	if c.Characteristics.Edu < 10 {
		t.Errorf("Tenure granted at Edu %d; the chart requires Edu 10+", c.Characteristics.Edu)
	}
}

// tenureAndPromotionSeqs returns the sequence of the Tenure grant and of
// the first promotion beyond Scholar3.
func tenureAndPromotionSeqs(c chargen.Character) (int, int) {
	tenure, beyond := 0, 0

	for _, e := range c.Events {
		if e.Kind != chargen.EventConsequence {
			continue
		}

		if e.Consequence.Kind == chargen.ConsequenceTenure && tenure == 0 {
			tenure = e.Seq
		}

		if e.Consequence.Kind == chargen.ConsequenceRankSet &&
			beyond == 0 && isBeyondAssistantProfessor(e.Consequence.Skill) {
			beyond = e.Seq
		}
	}

	return tenure, beyond
}

// isBeyondAssistantProfessor reports the chart 02 ranks above Scholar3.
func isBeyondAssistantProfessor(title string) bool {
	switch title {
	case "Associate Professor", "Professor", "Distinguished Professor":
		return true
	default:
		return false
	}
}

// TestScholarContinueAddsPublications verifies chart 02's "Continue Edu*"
// with "*Mod +Pubs": the Continue target is Edu plus the publications held.
func TestScholarContinueAddsPublications(t *testing.T) {
	c, _ := scholarRun(t, scholarGoldenSeed)

	pubs, checked := 0, 0

	for _, e := range c.Events {
		switch {
		case e.Kind == chargen.EventConsequence && e.Consequence.Kind == chargen.ConsequencePublication:
			pubs = e.Consequence.Value
		case e.Kind == chargen.EventThrow && strings.Contains(e.Throw.Cite, "Continue Edu"):
			checked++

			want := c.Characteristics.Edu + pubs
			if e.Throw.Target == nil || *e.Throw.Target != want {
				t.Errorf("Continue target = %v, want Edu %d + %d publications",
					e.Throw.Target, c.Characteristics.Edu, pubs)
			}
		}
	}

	if checked == 0 {
		t.Fatal("the pinned seed records no Continue throw")
	}
}

// TestScholarWaiverPolicy verifies POLICY.md's career-waiver rule: the
// auto policy spends a waiver only where the un-waived outcome would end
// the career, since waivers share one decaying pool with education (I-22).
func TestScholarWaiverPolicy(t *testing.T) {
	c, _ := scholarRun(t, scholarAmateurSeed)

	attempts := 0

	for _, e := range c.Events {
		if e.Kind != chargen.EventChoice || e.Choice.Prompt == "" {
			continue
		}

		if !strings.HasPrefix(e.Choice.Prompt, "Attempt a Waiver?") {
			continue
		}

		if e.Choice.Chosen != 0 {
			continue
		}

		attempts++

		if !strings.Contains(e.Choice.Prompt, "Continue") && !strings.Contains(e.Choice.Prompt, "To Begin") {
			t.Errorf("event %d waives a non-career-ending outcome: %q", e.Seq, e.Choice.Prompt)
		}
	}

	if attempts == 0 {
		t.Skip("the pinned seed never reaches a career-ending waiver")
	}
}
