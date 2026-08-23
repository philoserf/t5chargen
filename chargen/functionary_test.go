package chargen_test

import (
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/chargen"
)

// functionaryPath runs a first career and then changes into Functionary,
// which is the only way to reach chart 13: "Functionary is never a first
// career" (p. 87), and the auto policy declines every career change
// (POLICY.md `change_career`). Records generated this way carry
// policy_version "none".
type functionaryPath struct{ first string }

func (d functionaryPath) Choose(c chargen.Choice) int {
	//nolint:exhaustive // Deliberately partitioned: the rest defer to the auto policy.
	switch c.ID {
	case chargen.ChooseCareerChange:
		return 1
	case chargen.ChooseCareer:
		for i, option := range c.Options {
			if option == "Functionary" {
				return i
			}
		}

		for i, option := range c.Options {
			if option == d.first {
				return i
			}
		}
	default:
	}

	return chargen.DefaultPolicy{}.Choose(c)
}

func (functionaryPath) Kind() chargen.DeciderKind { return chargen.DeciderPlayer }

// functionaryRun generates the pinned Functionary character and returns
// its record: eight terms rising to F7, which passes through F6 and so
// exercises both titles the chart varies.
func functionaryRun(t *testing.T) (chargen.Character, chargen.CareerRecord) {
	t.Helper()

	c, err := chargen.Generate(chargen.Options{Seed: 305, Decider: functionaryPath{first: "Scholar"}})
	if err != nil {
		t.Fatal(err)
	}

	for _, record := range c.Careers {
		if record.Career == "Functionary" && record.Began {
			return c, record
		}
	}

	t.Fatal("seed 305 no longer reaches Functionary; find and pin another seed")

	return c, chargen.CareerRecord{}
}

// TestFunctionaryGolden pins the full record. Unlike the other eleven
// career fixtures this one is generated with a test Decider, because no
// auto-generated character can reach the career at all.
func TestFunctionaryGolden(t *testing.T) {
	c, _ := functionaryRun(t)
	goldenJSON(t, c, "testdata/career_functionary.json")
}

// TestFunctionaryNeedsAPriorCareer pins "To Begin Total Terms x3" against
// the box that states the same bar in words. A character with no prior
// service throws against zero and cannot succeed.
func TestFunctionaryNeedsAPriorCareer(t *testing.T) {
	c, err := chargen.Generate(chargen.Options{
		Seed: 1, Career: "Functionary", Decider: chargen.DefaultPolicy{},
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, record := range c.Careers {
		if record.Career == "Functionary" && record.Began {
			t.Fatal("a Functionary career began with no prior terms")
		}
	}
}

// TestOfficePoliticsIsTheContinue pins chart 13's "Continue Office
// Politics". The career has no Continue throw of its own: "Risk Failure:
// Functionary career ends. The character may not Continue. Risk Success:
// Functionary may continue in the career".
func TestOfficePoliticsIsTheContinue(t *testing.T) {
	c, record := functionaryRun(t)

	if len(record.Terms) < 2 {
		t.Fatalf("the pinned seed serves %d terms; the Continue rule needs more than one", len(record.Terms))
	}

	for _, e := range c.Events {
		if e.Kind != chargen.EventThrow {
			continue
		}

		if strings.Contains(e.Throw.Cite, "chart 13") && strings.Contains(e.Throw.Cite, "Continue") {
			t.Errorf("a Functionary rolled a Continue throw: %s", e.Throw.Cite)
		}
	}

	// Every term but the last continued, and the last ended the career.
	for i, term := range record.Terms {
		if want := i < len(record.Terms)-1; term.Continued != want {
			t.Errorf("term %d continued=%v, want %v", term.Term, term.Continued, want)
		}
	}
}

// TestFunctionaryRankTitles pins the two titles chart 13 varies: F6 takes
// the associated career's name ("Scholar F6 =College President") and F7 a
// rolled ordinal ("* N= 1D (ie: 3rd UnderSecretary)").
func TestFunctionaryRankTitles(t *testing.T) {
	c, record := functionaryRun(t)

	if record.AssociatedCareer != "Scholar" {
		t.Fatalf("associated with %q, want Scholar", record.AssociatedCareer)
	}

	titles := map[string]string{}

	for _, e := range c.Events {
		if e.Kind == chargen.EventConsequence &&
			e.Consequence.Kind == chargen.ConsequenceRankSet && e.Consequence.Career == "Functionary" {
			titles[e.Consequence.Skill] = ""
		}
	}

	if _, ok := titles["College President"]; !ok {
		t.Errorf("F6 was not renamed for a Scholar-associated Functionary; saw %v", keysOf(titles))
	}

	if !strings.HasSuffix(record.RankTitle, "UnderSecretary") {
		t.Errorf("final rank %q, want an Nth UnderSecretary", record.RankTitle)
	}
}

func keysOf(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}

	return out
}

// TestNobleMayNotBecomeAFunctionary pins chart 13's note. p. 66 already
// forbids a Noble leaving any career, so the two rules agree; this holds
// them together, since chart 13 states it independently.
func TestNobleMayNotBecomeAFunctionary(t *testing.T) {
	for seed := uint64(1); seed <= 200; seed++ {
		c, err := chargen.Generate(chargen.Options{Seed: seed, Decider: functionaryPath{first: "Noble"}})
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}

		for i, record := range c.Careers {
			if record.Career != "Functionary" || i == 0 {
				continue
			}

			if previous := c.Careers[i-1]; previous.Career == "Noble" && previous.Began {
				t.Fatalf("seed %d: a Noble became a Functionary", seed)
			}
		}
	}
}

// TestFunctionaryAutoSkills pins the Auto Skill column, whose rows the
// printed table aligns by eye: F0 Clerk gives Bureaucrat, F2 Senior
// Supervisor Admin, and F3 Manager Bureaucrat. The other six give none.
func TestFunctionaryAutoSkills(t *testing.T) {
	def, err := career.Functionary()
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]string{"F0": "Bureaucrat", "F2": "Admin", "F3": "Bureaucrat"}

	for _, rank := range def.Ranks {
		if got := rank.AutoSkill; got != want[rank.ID] {
			t.Errorf("%s %s auto skill %q, want %q", rank.ID, rank.Title, got, want[rank.ID])
		}
	}

	if len(def.Ranks) != 9 {
		t.Errorf("chart 13 prints %d ranks, want 9 (F0..F8)", len(def.Ranks))
	}
}

// TestSecretaryIsTheTopOfTheLadder pins what a Reward success buys a
// character who has nowhere left to go. Chart 13 prints nine ranks and no
// rule for passing Secretary, so an F8 who succeeds is not promoted — and
// must not collect the "Per Promotion 1" eligibility (chart 13 B) that
// only a promotion earns.
func TestSecretaryIsTheTopOfTheLadder(t *testing.T) {
	const secretarySeed = 447

	c, err := chargen.Generate(chargen.Options{Seed: secretarySeed, Decider: functionaryPath{first: "Scholar"}})
	if err != nil {
		t.Fatal(err)
	}

	record := functionaryRecord(c)
	if record.Rank != "F8" {
		t.Fatalf("seed %d ends at %s, not Secretary; find and pin another seed", secretarySeed, record.Rank)
	}

	// F0 on entry plus eight promotions is the whole ladder.
	ranks := countRankSets(c)
	if ranks != 9 {
		t.Errorf("%d rank changes, want 9 (F0 on entry and eight promotions)", ranks)
	}

	// Terms served at F8 cannot have been promotions, so they cannot have
	// scored a success.
	promotions := 0

	for _, term := range record.Terms {
		if term.Success {
			promotions++
		}
	}

	if promotions != 8 {
		t.Errorf("%d terms scored a promotion over %d served, want 8", promotions, len(record.Terms))
	}
}

// functionaryRecord returns the character's Functionary career record.
func functionaryRecord(c chargen.Character) chargen.CareerRecord {
	var record chargen.CareerRecord

	for _, r := range c.Careers {
		if r.Career == "Functionary" {
			record = r
		}
	}

	return record
}

// countRankSets counts the Functionary rank changes in a record's log.
func countRankSets(c chargen.Character) int {
	ranks := 0

	for _, e := range c.Events {
		if e.Kind == chargen.EventConsequence &&
			e.Consequence.Kind == chargen.ConsequenceRankSet && e.Consequence.Career == "Functionary" {
			ranks++
		}
	}

	return ranks
}
