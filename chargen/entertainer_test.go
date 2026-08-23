package chargen_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/chargen"
)

// entertainerRun generates an Entertainer and returns the character and
// its career record.
// entertainerGoldenSeed is the pinned fixture: seven terms exercising two
// Comebacks and three Fame increases
// (chargen/testdata/career_entertainer.json).
const entertainerGoldenSeed = 572

func entertainerRun(t *testing.T) (chargen.Character, chargen.CareerRecord) {
	t.Helper()

	c, err := chargen.Generate(chargen.Options{
		Seed: entertainerGoldenSeed, Career: "Entertainer", Decider: chargen.DefaultPolicy{},
	})
	if err != nil {
		t.Fatalf("seed %d: %v", entertainerGoldenSeed, err)
	}

	if len(c.Careers) == 0 {
		t.Fatalf("seed %d: no career record", entertainerGoldenSeed)
	}

	return c, c.Careers[len(c.Careers)-1]
}

// consequences returns the values of every consequence of one kind, in
// order.
func consequences(c chargen.Character, kind chargen.ConsequenceKind) []int {
	var values []int

	for _, e := range c.Events {
		if e.Kind == chargen.EventConsequence && e.Consequence.Kind == kind {
			values = append(values, e.Consequence.Value)
		}
	}

	return values
}

// TestEntertainerSpecialties verifies chart 03's six specialties and their
// per-specialty Begin checks ("Begin Actor C2 or C3 / Begin Artist C3 or
// Int / Begin Author Int or C5 / ... / Begin Chef C2 or Int", p. 77).
func TestEntertainerSpecialties(t *testing.T) {
	def, err := career.Entertainer()
	if err != nil {
		t.Fatal(err)
	}

	want := []struct {
		name   string
		checks []string
	}{
		{name: "Artist", checks: []string{"End", "Int"}},
		{name: "Actor", checks: []string{"Dex", "End"}},
		{name: "Author", checks: []string{"Int", "Edu"}},
		{name: "Dancer", checks: []string{"Dex", "End"}},
		{name: "Musician", checks: []string{"Dex", "End"}},
		{name: "Chef", checks: []string{"Dex", "Int"}},
	}

	if len(def.BeginTracks) != len(want) {
		t.Fatalf("specialties = %d, want %d", len(def.BeginTracks), len(want))
	}

	for i, w := range want {
		got := def.BeginTracks[i]
		if got.Name != w.name {
			t.Errorf("specialty %d = %q, want %q (chart order)", i+1, got.Name, w.name)
		}

		if !slices.Equal(got.Checks, w.checks) {
			t.Errorf("%s checks = %v, want %v", w.name, got.Checks, w.checks)
		}
	}
}

// TestEntertainerRotatesNoControllingCharacteristic verifies chart 03's
// "Risk & Reward Talent": the Entertainer has no series to rotate, so no
// controlling-characteristic choice is presented and no term records one.
func TestEntertainerRotatesNoControllingCharacteristic(t *testing.T) {
	def, err := career.Entertainer()
	if err != nil {
		t.Fatal(err)
	}

	if len(def.ControllingCharacteristics) != 0 {
		t.Errorf("controlling characteristics = %v, want none", def.ControllingCharacteristics)
	}

	c, record := entertainerRun(t)

	for _, e := range c.Events {
		if e.Kind == chargen.EventChoice && e.Choice.Prompt == "Select the term's controlling characteristic" {
			t.Fatalf("event %d presents a controlling-characteristic choice", e.Seq)
		}
	}

	for _, term := range record.Terms {
		if term.ControllingCharacteristic != "" {
			t.Errorf("term %d records CC %q, want none", term.Term, term.ControllingCharacteristic)
		}
	}
}

// TestEntertainerNoRiskAndReward verifies the Entertainer runs no Risk or
// Reward throws: Book 1 lists it among the careers with a variant of that
// cycle — "The Entertainer Career focuses on Fame and resolves the current
// level of Fame for the character" (p. 66; interpretation I-18).
func TestEntertainerNoRiskAndReward(t *testing.T) {
	c, _ := entertainerRun(t)

	for _, e := range c.Events {
		if e.Kind != chargen.EventThrow {
			continue
		}

		for _, banned := range []string{"Risk vs", "Reward vs"} {
			if strings.Contains(e.Throw.Cite, banned) {
				t.Errorf("event %d rolls %q; the Entertainer's variant is the Fame resolution", e.Seq, banned)
			}
		}
	}
}

// TestEntertainerInitialFameEqualsTalent verifies "Before Begin ... roll
// initial Fame and Talent (with one 2D roll; they are equal)" (chart 03).
func TestEntertainerInitialFameEqualsTalent(t *testing.T) {
	c, _ := entertainerRun(t)

	fame := consequences(c, chargen.ConsequenceFameChange)
	talent := consequences(c, chargen.ConsequenceTalentSet)

	if len(fame) == 0 || len(talent) == 0 {
		t.Fatal("the pinned seed records no initial Fame and Talent")
	}

	if fame[0] != talent[0] {
		t.Errorf("initial Fame %d and Talent %d differ; the chart sets both from one 2D", fame[0], talent[0])
	}
}

// TestEntertainerFirstTermHasNoFlux verifies the chart's Fame table: its
// Term 1 row reads "=2D" with no Flux, so the first term cannot raise Fame
// and cannot earn the "If Fame Increases" skills (interpretation I-19).
func TestEntertainerFirstTermHasNoFlux(t *testing.T) {
	c, _ := entertainerRun(t)

	for _, e := range firstTermEvents(c) {
		if e.Kind != chargen.EventThrow {
			continue
		}

		if strings.Contains(e.Throw.Cite, "Flux") {
			t.Errorf("event %d rolls Flux in term 1; the chart's Term 1 row is =2D", e.Seq)
		}
	}
}

// firstTermEvents returns the events between the Term 1 and Term 2 steps.
func firstTermEvents(c chargen.Character) []chargen.Event {
	var (
		events []chargen.Event
		inTerm bool
	)

	for _, e := range c.Events {
		if e.Kind == chargen.EventStep && strings.Contains(e.Step.Name, "Term 1") {
			inTerm = true

			continue
		}

		if e.Kind == chargen.EventStep && strings.Contains(e.Step.Name, "Term 2") {
			break
		}

		if inTerm {
			events = append(events, e)
		}
	}

	return events
}

// TestEntertainerTalentTracksFameIncreases verifies "If Fame increases,
// increase Talent +1" (chart 03): Talent rises exactly once per term whose
// Fame ended higher than it started, and never otherwise.
func TestEntertainerTalentTracksFameIncreases(t *testing.T) {
	// Its own seed rather than the golden's: this test needs several Fame
	// increases, and pinning it to the fixture means every stream change
	// that shortens that career silently guts the test instead of the
	// fixture. Seed 15 runs eight terms and raises Talent six times.
	const talentSeed = 15

	c, err := chargen.Generate(chargen.Options{
		Seed: talentSeed, Career: "Entertainer", Decider: chargen.DefaultPolicy{},
	})
	if err != nil {
		t.Fatal(err)
	}

	record := c.Careers[len(c.Careers)-1]

	talent := consequences(c, chargen.ConsequenceTalentSet)
	if len(talent) < 2 {
		t.Fatalf("the pinned seed records %d Talent values; the fixture expects increases", len(talent))
	}

	// The first is the initial value; each later one is exactly +1.
	for i := 1; i < len(talent); i++ {
		if talent[i] != talent[i-1]+1 {
			t.Errorf("Talent went %d -> %d, want +1", talent[i-1], talent[i])
		}
	}

	if record.Talent != talent[len(talent)-1] {
		t.Errorf("record Talent = %d, want %d", record.Talent, talent[len(talent)-1])
	}
}

// TestEntertainerComeback verifies "Comeback: Reset Fame to 2D; Talent is
// unchanged" (chart 03): the reset sets Fame to the rolled value and no
// Talent consequence accompanies it.
func TestEntertainerComeback(t *testing.T) {
	c, _ := entertainerRun(t)

	comebacks := 0

	for i, e := range c.Events {
		if e.Kind != chargen.EventConsequence || e.Consequence.Kind != chargen.ConsequenceComeback {
			continue
		}

		comebacks++

		// The Fame reset follows immediately, to the rolled value.
		next := c.Events[i+1]
		if next.Kind != chargen.EventConsequence || next.Consequence.Kind != chargen.ConsequenceFameChange {
			t.Fatalf("event %d: Comeback is not followed by a Fame change", e.Seq)
		}

		if next.Consequence.Value != e.Consequence.Value {
			t.Errorf("Comeback rolled %d but Fame became %d", e.Consequence.Value, next.Consequence.Value)
		}
	}

	if comebacks == 0 {
		t.Fatal("the pinned seed records no Comeback; the fixture expects one")
	}
}

// TestEntertainerContinueUsesFame verifies chart 03's "Continue Fame": the
// Continue throw targets the career's current Fame, not a characteristic.
func TestEntertainerContinueUsesFame(t *testing.T) {
	c, _ := entertainerRun(t)

	fame := 0
	checked := 0

	for _, e := range c.Events {
		switch {
		case e.Kind == chargen.EventConsequence && e.Consequence.Kind == chargen.ConsequenceFameChange:
			fame = e.Consequence.Value
		case e.Kind == chargen.EventThrow && strings.Contains(e.Throw.Cite, "Continue Fame"):
			checked++

			if e.Throw.Target == nil || *e.Throw.Target != fame {
				t.Errorf("Continue target = %v, want the current Fame %d", e.Throw.Target, fame)
			}
		}
	}

	if checked == 0 {
		t.Fatal("the pinned seed records no Continue Fame throw")
	}
}

// TestEntertainerFameIncreaseEarnsSkills verifies chart 03 table B: "Per
// Term 4 / If Fame Increases 2".
func TestEntertainerFameIncreaseEarnsSkills(t *testing.T) {
	def, err := career.Entertainer()
	if err != nil {
		t.Fatal(err)
	}

	if def.SkillsPerTerm != 4 {
		t.Fatalf("chart 03 table B per term = %d, want 4", def.SkillsPerTerm)
	}

	c, record := entertainerRun(t)

	columns := 0

	for _, e := range c.Events {
		if e.Kind == chargen.EventChoice && e.Choice.Prompt == "Select an Entertainer Skills column" {
			columns++
		}
	}

	// Talent rises once per Fame-increase term; the first value is the
	// initial Talent, which earns nothing.
	increases := len(consequences(c, chargen.ConsequenceTalentSet)) - 1

	want := len(record.Terms)*def.SkillsPerTerm + increases*2
	if columns != want {
		t.Errorf("skill-column rolls = %d, want %d (%d terms x 4 + %d Fame increases x 2)",
			columns, want, len(record.Terms), increases)
	}
}

// TestEntertainerComebackEarnsNoTalent verifies chart 03's "Talent is
// unchanged": a Comeback that lands above the Fame it replaced still earns
// neither the Talent nor the two extra skills a Fame increase carries
// (interpretation I-20, ERRATA.md).
func TestEntertainerComebackEarnsNoTalent(t *testing.T) {
	c, _ := entertainerRun(t)

	rebounds := 0

	for i, e := range c.Events {
		if e.Kind != chargen.EventConsequence || e.Consequence.Kind != chargen.ConsequenceComeback {
			continue
		}

		// The Fame change that follows carries the delta against the Fame
		// the Comeback replaced.
		reset := c.Events[i+1]
		if reset.Consequence.Delta > 0 {
			rebounds++
		}

		if seq := talentBeforeContinue(c.Events[i+1:]); seq != 0 {
			t.Errorf("event %d raises Talent in a Comeback term; the chart leaves Talent unchanged", seq)
		}
	}

	if rebounds == 0 {
		t.Fatal("the pinned seed records no Comeback that landed above the Fame it replaced")
	}
}

// talentBeforeContinue reports the sequence of the first Talent
// consequence in the rest of the term, or zero if the term's Continue
// throw comes first.
func talentBeforeContinue(events []chargen.Event) int {
	for _, e := range events {
		if e.Kind == chargen.EventThrow && strings.Contains(e.Throw.Cite, "Continue Fame") {
			return 0
		}

		if e.Kind == chargen.EventConsequence && e.Consequence.Kind == chargen.ConsequenceTalentSet {
			return e.Seq
		}
	}

	return 0
}
