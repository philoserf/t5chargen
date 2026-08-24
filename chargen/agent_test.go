package chargen_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/chargen"
	"github.com/philoserf/t5chargen/skill"
)

// agentGoldenSeed is the pinned fixture: eight terms covering eight
// distinct careers, including both special rows — Citizen, which rolls on
// table E, and Functionary, which is transcribed for this table alone.
const agentGoldenSeed = 1717

func agentRun(t *testing.T) (chargen.Character, chargen.CareerRecord) {
	t.Helper()

	c, err := chargen.Generate(chargen.Options{
		Seed: agentGoldenSeed, Career: "Agent", Decider: chargen.DefaultPolicy{},
	})
	if err != nil {
		t.Fatalf("seed %d: %v", agentGoldenSeed, err)
	}

	return c, c.Careers[len(c.Careers)-1]
}

// undercoverAssignments returns the cover careers the Agent took, in
// order.
func undercoverAssignments(c chargen.Character) []string {
	var covers []string

	for _, e := range c.Events {
		if e.Kind == chargen.EventConsequence && e.Consequence.Kind == chargen.ConsequenceUndercover {
			covers = append(covers, e.Consequence.Career)
		}
	}

	return covers
}

// TestAgentUndercoverEveryTerm verifies chart 09's Mission structure: each
// term is "two years Undercover in a different career ... followed by two
// years completing the Mission", so every term rolls an assignment.
func TestAgentUndercoverEveryTerm(t *testing.T) {
	c, record := agentRun(t)

	if got := len(undercoverAssignments(c)); got != len(record.Terms) {
		t.Errorf("undercover assignments = %d, want one per term (%d)", got, len(record.Terms))
	}
}

// TestAgentUndercoverTableIsWhole verifies the transcription: chart 09's
// table is 3 x 6 and every row names a career the engine can read.
func TestAgentUndercoverTableIsWhole(t *testing.T) {
	def, err := career.Agent()
	if err != nil {
		t.Fatal(err)
	}

	if len(def.Undercover.Rows) != 18 {
		t.Fatalf("undercover rows = %d, want 18", len(def.Undercover.Rows))
	}

	for a := 1; a <= 3; a++ {
		for b := 1; b <= 6; b++ {
			row, ok := def.Undercover.UndercoverAt(a, b)
			if !ok {
				t.Errorf("no undercover row for A%d B%d", a, b)

				continue
			}

			if row.Career == "" || row.Source == "" {
				t.Errorf("A%d B%d = %+v, want a career and a source", a, b, row)

				continue
			}

			// A source the engine cannot resolve would fail only on the
			// terms that roll it, so it is checked for the whole table.
			if _, err := chargen.LoadUndercoverCareer(row.Source); err != nil {
				t.Errorf("A%d B%d source %q: %v", a, b, row.Source, err)
			}
		}
	}
}

// agentVocationPolicy forces the Agent's Vocation column, whose first cell
// is chart 09's "Any Knowledge". The default policy prefers the Mission
// column, so no pinned seed ever reads it.
type agentVocationPolicy struct{ chargen.DefaultPolicy }

func (p agentVocationPolicy) Choose(c chargen.Choice) (int, error) {
	if c.ID == chargen.ChooseSkillColumn {
		if i := slices.Index(c.Options, "Vocation"); i >= 0 {
			return i, nil
		}
	}

	return autoPolicy(c)
}

// TestAgentAnyKnowledgeCellResolves verifies chart 09's "Any Knowledge"
// cell (Vocation column, p. 83) offers the Master Skill List Knowledges
// rather than aborting generation.
func TestAgentAnyKnowledgeCellResolves(t *testing.T) {
	c, err := chargen.Generate(chargen.Options{Seed: 1, Career: "Agent", Decider: agentVocationPolicy{}})
	if err != nil {
		t.Fatalf("Vocation column: %v", err)
	}

	offered := 0

	for _, e := range c.Events {
		if e.Kind != chargen.EventChoice || e.Choice.Prompt != "Select Any Knowledge" {
			continue
		}

		offered++

		for _, option := range e.Choice.Options {
			if entry, ok := skill.Lookup(option); !ok || entry.Kind != skill.KindKnowledge {
				t.Errorf("event %d offers %q, which is not a Knowledge", e.Seq, option)
			}
		}
	}

	if offered == 0 {
		t.Error("seed 1 never reads the Vocation column's Any Knowledge cell")
	}
}

// TestAgentSelectsFromTheCoverCareersTable verifies "Select (not Roll) one
// skill from the skill tables of that Career" (chart 09): the alternatives
// are that career's own skills, and never a characteristic or the Agent's
// Major (interpretation I-38, ERRATA.md).
func TestAgentSelectsFromTheCoverCareersTable(t *testing.T) {
	c, _ := agentRun(t)

	checked := 0

	for _, e := range c.Events {
		if e.Kind != chargen.EventChoice || !strings.HasPrefix(e.Choice.Prompt, "Select a skill from the ") {
			continue
		}

		checked++

		for _, option := range e.Choice.Options {
			for _, banned := range []string{"Str", "Dex", "End", "Int", "Edu", "Soc"} {
				if option == banned {
					t.Errorf("event %d offers the characteristic %q as a skill", e.Seq, option)
				}
			}
		}
	}

	if checked == 0 {
		t.Fatal("the pinned seed records no undercover skill selection")
	}
}

// TestAgentCitizenRowRollsTableE verifies the two Citizen rows, which say
// "Roll on Citizen Life Skills" rather than offering a selection
// (interpretation I-39, ERRATA.md).
func TestAgentCitizenRowRollsTableE(t *testing.T) {
	c, _ := agentRun(t)

	if !slices.Contains(undercoverAssignments(c), "Citizen") {
		t.Fatal("the pinned seed never goes undercover as a Citizen")
	}

	rolled := false

	for i, e := range c.Events {
		if e.Kind != chargen.EventConsequence ||
			e.Consequence.Kind != chargen.ConsequenceUndercover ||
			e.Consequence.Career != "Citizen" {
			continue
		}

		if selected, tableE := citizenCoverEvents(c.Events[i+1:]); selected > 0 {
			t.Errorf("event %d offers a selection for a Citizen cover", selected)
		} else if tableE {
			rolled = true
		}
	}

	if !rolled {
		t.Error("a Citizen cover recorded no table E roll")
	}
}

// citizenCoverEvents scans a Citizen cover's half of the term, ending at
// the Risk throw that begins the mission. It reports the sequence of any
// skill selection offered (which would be wrong) and whether table E was
// rolled (which is right).
func citizenCoverEvents(events []chargen.Event) (int, bool) {
	tableE := false

	for _, e := range events {
		if e.Kind == chargen.EventThrow && strings.Contains(e.Throw.Cite, "Risk vs") {
			return 0, tableE
		}

		if e.Kind == chargen.EventChoice &&
			strings.HasPrefix(e.Choice.Prompt, "Select a skill from the ") {
			return e.Seq, tableE
		}

		if e.Kind == chargen.EventThrow && strings.Contains(e.Throw.Cite, "table E") {
			tableE = true
		}
	}

	return 0, tableE
}

// TestAgentFunctionaryRowResolves verifies that chart 09's Functionary row
// is playable: chart 13's skills table is transcribed as a reference
// career so the row offers a selection rather than aborting.
func TestAgentFunctionaryRowResolves(t *testing.T) {
	c, _ := agentRun(t)

	if !slices.Contains(undercoverAssignments(c), "Functionary") {
		t.Fatal("the pinned seed never goes undercover as a Functionary")
	}

	found := false

	for _, e := range c.Events {
		if e.Kind == chargen.EventChoice &&
			e.Choice.Prompt == "Select a skill from the Functionary skills table" {
			found = true

			if len(e.Choice.Options) == 0 {
				t.Error("the Functionary cover offers no skills")
			}
		}
	}

	if !found {
		t.Error("the Functionary cover recorded no skill selection")
	}
}

// TestFunctionaryIsNeverAFirstCareer verifies the gate chart 13 states in
// its own box and p. 63 restates: "Functionary is never a first career",
// and "Craftsman (1) and Functionary (13) are unavailable as initial
// careers". It is a full career now — the Agent's Undercover table read it
// as a reference career before it was playable (I-40) and reads the same
// skills table still.
func TestFunctionaryIsNeverAFirstCareer(t *testing.T) {
	def, err := career.Functionary()
	if err != nil {
		t.Fatal(err)
	}

	if !def.NotAFirstCareer {
		t.Error("chart 13 says Functionary is never a first career")
	}

	if !slices.Contains(career.Available(), "Functionary") {
		t.Error("Functionary is playable and should be available")
	}

	first, err := career.FirstCareers()
	if err != nil {
		t.Fatal(err)
	}

	if slices.Contains(first, "Functionary") {
		t.Error("Functionary can open a lifepath")
	}

	if len(def.SkillColumns) != 7 {
		t.Errorf("chart 13 table C has %d columns, want 7", len(def.SkillColumns))
	}
}

// TestAgentContinueAddsTerms verifies chart 09's "Continue Str*" with
// "*Mod +Terms", counted as completed terms (interpretation I-12).
func TestAgentContinueAddsTerms(t *testing.T) {
	c, _ := agentRun(t)

	// Str is tracked as the log reports it, not read off the finished
	// record: aging reduces characteristics mid-career (chart A p. 89), so
	// the target a term threw against is the Str the character had then.
	str, terms, checked := 0, 0, 0

	for _, e := range c.Events {
		switch {
		case e.Kind == chargen.EventConsequence && e.Consequence.Characteristic == "Str":
			str = e.Consequence.Value
		case e.Kind == chargen.EventThrow && strings.Contains(e.Throw.Cite, "Continue Str"):
			checked++

			assertTarget(t, e.Throw.Target, str+terms, str, terms)

			terms++
		default:
		}
	}

	if str != c.Characteristics.Str {
		t.Errorf("tracked Str ended at %d, record says %d", str, c.Characteristics.Str)
	}

	if checked == 0 {
		t.Fatal("the pinned seed records no Continue throw")
	}
}

// assertTarget checks one Continue throw's target against the Str the
// character held at the time plus his completed terms (chart 09 "Continue
// Str*", "*Mod +Terms"; interpretation I-12).
func assertTarget(t *testing.T, got *int, want, str, terms int) {
	t.Helper()

	if got == nil || *got != want {
		t.Errorf("Continue target = %v, want Str %d + %d terms", got, str, terms)
	}
}
