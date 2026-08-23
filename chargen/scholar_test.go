package chargen_test

import (
	"slices"
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
	_, record := scholarRun(t, scholarAmateurSeed)

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

// TestScholarContinueWaiver verifies the sixth event chart 02's Waivers
// box names: "An adverse die roll or decision (in ... Continue) may be
// waived" (p. 76). A waived Continue failure carries the career into the
// next term instead of ending it (interpretation I-22).
func TestScholarContinueWaiver(t *testing.T) {
	// Repinned when aging landed: aging consumes stream, so seed 1 no
	// longer reaches a failed Continue. Seed 7 waives one and survives,
	// which keeps the "career did not end there" assertion below meaningful.
	const continueWaiverSeed = 7

	c, err := chargen.Generate(chargen.Options{
		Seed: continueWaiverSeed, Career: "Scholar", Decider: chargen.DefaultPolicy{},
	})
	if err != nil {
		t.Fatal(err)
	}

	if waived := waivedContinues(t, c); waived == 0 {
		t.Fatalf("seed %d no longer waives a failed Continue; the chart 02 Continue waiver is untested",
			continueWaiverSeed)
	}

	// The waiver negates the failure: the career did not end there.
	record := c.Careers[len(c.Careers)-1]
	if len(record.Terms) < 2 {
		t.Errorf("career ran %d terms; a waived Continue continues it", len(record.Terms))
	}
}

// waivedContinues counts the waivers caused by a failed Continue throw,
// reporting any that negated a throw which had not failed.
func waivedContinues(t *testing.T, c chargen.Character) int {
	t.Helper()

	bySeq := make(map[int]chargen.Event, len(c.Events))
	for _, e := range c.Events {
		bySeq[e.Seq] = e
	}

	waived := 0

	for _, e := range c.Events {
		if e.Kind != chargen.EventConsequence || e.Consequence.Kind != chargen.ConsequenceWaived {
			continue
		}

		cause, ok := bySeq[e.Consequence.Cause]
		if !ok || cause.Kind != chargen.EventThrow || !strings.Contains(cause.Throw.Cite, "Continue") {
			continue
		}

		if cause.Throw.Success == nil || *cause.Throw.Success {
			t.Errorf("event %d waives a Continue throw that did not fail", e.Seq)
		}

		waived++
	}

	return waived
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
		t.Fatal("the pinned seed no longer reaches a career-ending waiver; the rule is untested")
	}
}

// cautiousWaiverDecider takes a large Caution Mod and accepts every
// waiver, reaching the case the auto policy cannot: a Publication
// rejected on its own roll but rescued by a Waiver, whose roll can still
// sit inside the Award-Winning margin because Caution raises the
// characteristic above the target.
type cautiousWaiverDecider struct{ chargen.DefaultPolicy }

func (d cautiousWaiverDecider) Choose(c chargen.Choice) int {
	if c.ID == chargen.ChooseCareerWaiver {
		return 0
	}

	if c.ID == chargen.ChooseRiskMod {
		if i := slices.Index(c.Options, cautionMod); i >= 0 {
			return i
		}
	}

	return d.DefaultPolicy.Choose(c)
}

// cautionMod is the Caution the fixture selects; cautionValue is its
// numeric value, which raises the Risk target and lowers the Reward one.
const (
	cautionMod   = "Caution +6"
	cautionValue = 6
)

func (cautiousWaiverDecider) Kind() chargen.DeciderKind { return chargen.DeciderPlayer }

// TestScholarWaivedPublicationIsNotAwardWinning verifies that the
// Award-Winning margin is read off a roll that carried the Publication on
// its own: "If Publication Roll is 4 less than Characteristic, it is
// <Award-Winning>" describes a successful publication, not a rejection
// rescued by a Waiver (interpretation I-25, ERRATA.md).
func TestScholarWaivedPublicationIsNotAwardWinning(t *testing.T) {
	c, err := chargen.Generate(chargen.Options{
		Seed: 12, Career: "Scholar", Decider: cautiousWaiverDecider{},
	})
	if err != nil {
		t.Fatal(err)
	}

	waived := false

	for i, e := range c.Events {
		if !rejectedInsideAwardMargin(e) {
			continue
		}

		// Any publication recorded before the next Research or Publication
		// throw came from a Waiver, and must count as one. (The waiver's
		// own throw sits between them.)
		delta, ok := publicationAfter(c.Events[i+1:])
		if !ok {
			continue
		}

		waived = true

		if delta != 1 {
			t.Errorf("a waived rejection inside the Award-Winning margin counted as %d, want 1", delta)
		}
	}

	if !waived {
		t.Fatal("seed 12 records no waived Publication inside the Award-Winning margin; the fixture is stale")
	}
}

// rejectedInsideAwardMargin reports a Publication throw that failed on its
// own yet landed within four of the characteristic, which Caution makes
// possible by lowering the target below the characteristic.
func rejectedInsideAwardMargin(e chargen.Event) bool {
	if e.Kind != chargen.EventThrow || !strings.Contains(e.Throw.Cite, "Publication vs") {
		return false
	}

	if e.Throw.Success == nil || *e.Throw.Success || e.Throw.Target == nil {
		return false
	}

	characteristic := *e.Throw.Target + cautionValue

	return e.Throw.Total <= characteristic-4
}

// publicationAfter returns the delta of the next publication recorded
// before the term's next Research or Publication throw.
func publicationAfter(events []chargen.Event) (int, bool) {
	for _, e := range events {
		if e.Kind == chargen.EventThrow &&
			(strings.Contains(e.Throw.Cite, "Publication vs") ||
				strings.Contains(e.Throw.Cite, "Research vs")) {
			return 0, false
		}

		if e.Kind == chargen.EventConsequence && e.Consequence.Kind == chargen.ConsequencePublication {
			return e.Consequence.Delta, true
		}
	}

	return 0, false
}
