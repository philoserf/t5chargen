package chargen_test

import (
	"slices"
	"testing"

	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/chargen"
)

// schemeCareerPrompt identifies the offer in a record's event log.
// ChoiceEvent records the prompt rather than the ID, so this is what a
// reader of the log has to go on too.
const schemeCareerPrompt = "Select a previous career as the Scheme?"

// schemeSeedSearch is how wide the sweeps look. The shape wanted is
// narrow — a career served, a change into Rogue, and a Scheme term after
// it — so the search is wider than the career tests that need only one
// career to open.
const schemeSeedSearch = 400

// rollTheSchemeOption is the option that declines the chart's
// alternative, as a reader of the event log sees it.
const rollTheSchemeOption = "Roll for it"

// schemesOnHisPast serves a career, changes into Rogue at the first
// opportunity, stays there, and takes a previous career as every Scheme
// the chart offers one for.
//
// The auto policy rolls instead (POLICY.md), which is why no golden
// record reaches this path and a decider is what reaches it.
type schemesOnHisPast struct{ inRogue bool }

func (d *schemesOnHisPast) Choose(c chargen.Choice) (int, error) {
	switch c.ID { //nolint:exhaustive // Only the choice points this decider steers; the rest fall through to the policy.
	case chargen.ChooseCareerChange:
		if d.inRogue {
			return 0, nil // stay; the Scheme terms are the point
		}

		return 1, nil // "Change careers"
	case chargen.ChooseCareer:
		if i := slices.Index(c.Options, "Rogue"); i >= 0 {
			d.inRogue = true

			return i, nil
		}
	case chargen.ChooseSchemeCareer:
		return 0, nil // the earliest career served
	}

	return autoPolicy(c)
}

func (*schemesOnHisPast) Kind() chargen.DeciderKind { return chargen.DeciderPlayer }

// schemeSelectionRun finds a seed whose character serves a career, changes
// into Rogue, and is offered a previous career as a Scheme.
func schemeSelectionRun(t *testing.T) (chargen.Character, bool) {
	t.Helper()

	for seed := range uint64(schemeSeedSearch) {
		c, err := chargen.Generate(chargen.Options{
			Seed: seed, Career: "Scout", Decider: &schemesOnHisPast{},
		})
		if err != nil || len(c.Careers) < 2 || c.Careers[len(c.Careers)-1].Career != "Rogue" {
			continue
		}

		if !selectedScheme(c) {
			continue
		}

		return c, true
	}

	return chargen.Character{}, false
}

// selectedScheme reports whether any Scheme in the record was taken
// rather than rolled.
func selectedScheme(c chargen.Character) bool {
	return slices.ContainsFunc(c.Events, func(e chargen.Event) bool {
		return e.Consequence != nil &&
			e.Consequence.Kind == chargen.ConsequenceScheme &&
			e.Consequence.Detail != ""
	})
}

// schemes returns the record's Scheme consequences in log order.
func schemes(c chargen.Character) []*chargen.ConsequenceEvent {
	found := make([]*chargen.ConsequenceEvent, 0, len(c.Careers))

	for _, event := range c.Events {
		if event.Consequence != nil && event.Consequence.Kind == chargen.ConsequenceScheme {
			found = append(found, event.Consequence)
		}
	}

	return found
}

// offers returns the choice events where a previous career was offered.
func offers(c chargen.Character) []*chargen.ChoiceEvent {
	found := make([]*chargen.ChoiceEvent, 0, len(c.Careers))

	for _, event := range c.Events {
		if event.Choice != nil && event.Choice.Prompt == schemeCareerPrompt {
			found = append(found, event.Choice)
		}
	}

	return found
}

// careersServed reports which careers the record shows a term of, and
// which it shows only a refused To Begin for.
func careersServed(c chargen.Character) (map[string]bool, map[string]bool) {
	served, refused := make(map[string]bool), make(map[string]bool)

	for _, record := range c.Careers {
		if record.Began {
			served[record.Career] = true
		} else {
			refused[record.Career] = true
		}
	}

	for name := range served {
		delete(refused, name)
	}

	return served, refused
}

// TestRogueMaySelectAPreviousCareerAsHisScheme verifies chart 10: "A Rogue
// may select for his Scheme (rather than roll) any previous career."
//
// The selection names a career the record shows he served, and the chart
// prints a row for every career, so the Value is the table's own
// (interpretation I-109).
func TestRogueMaySelectAPreviousCareerAsHisScheme(t *testing.T) {
	// Not a Skip: a skip passes silently the day no seed in range
	// reaches the case, leaving the test asserting nothing about the
	// rule it names.
	c, ok := schemeSelectionRun(t)
	if !ok {
		t.Fatalf("no seed under %d serves a career, changes to Rogue, and selects a Scheme; widen the search",
			schemeSeedSearch)
	}

	served, _ := careersServed(c)
	selections := 0

	for _, scheme := range schemes(c) {
		if scheme.Detail == "" {
			continue
		}

		selections++

		if !served[scheme.Skill] {
			t.Errorf("%q was selected as the Scheme; the record shows no term of it", scheme.Skill)
		}
	}

	if selections == 0 {
		t.Fatal("the run was chosen for selecting a Scheme and then selected none")
	}
}

// TestASelectedSchemeThrowsNoFlux verifies the chart's "(rather than
// roll)": selection replaces the roll rather than steering it, so the
// stream carries one Scheme Flux for every Scheme rolled and none for
// any Scheme selected.
//
// This is the claim the "Flux may be modified (after roll) plus or minus
// 1" clause depends on — a modification of a roll that never happened is
// what the reading rules out (I-109). Counting the throws is what makes
// it checkable: a selection that logged its consequence and then rolled
// anyway would satisfy any weaker assertion about the consequence alone.
func TestASelectedSchemeThrowsNoFlux(t *testing.T) {
	c, ok := schemeSelectionRun(t)
	if !ok {
		t.Fatalf("no seed under %d selects a Scheme; widen the search", schemeSeedSearch)
	}

	selected, rolled := 0, 0

	for _, scheme := range schemes(c) {
		if scheme.Detail == "" {
			rolled++
		} else {
			selected++
		}
	}

	if selected == 0 {
		t.Fatal("the run was chosen for selecting a Scheme and then selected none")
	}

	if throws := countSchemeFlux(t, c); throws != rolled {
		t.Errorf("the record throws %d Scheme Flux for %d rolled Schemes; a selected Scheme throws none",
			throws, rolled)
	}

	// One Scheme per term. A selection that logged its consequence and
	// then rolled one anyway would leave those counts balanced — the
	// term gains a throw and a rolled Scheme together — and is visible
	// only as two Schemes standing in the same term.
	if worst := mostSchemesInATerm(c); worst > 1 {
		t.Errorf("a term resolved %d Schemes; chart 10 masterminds one", worst)
	}
}

// countSchemeFlux counts the Flux throws made against chart 10's Rogue
// Schemes table.
func countSchemeFlux(t *testing.T, c chargen.Character) int {
	t.Helper()

	def, err := career.Rogue()
	if err != nil {
		t.Fatalf("rogue career: %v", err)
	}

	throws := 0

	for _, event := range c.Events {
		if event.Throw != nil && event.Throw.Cite == def.Schemes.Cite {
			throws++
		}
	}

	return throws
}

// mostSchemesInATerm reports the largest number of Schemes resolved
// between two step markers, which is where a term begins.
func mostSchemesInATerm(c chargen.Character) int {
	inTerm, worst := 0, 0

	for _, event := range c.Events {
		switch {
		case event.Kind == chargen.EventStep:
			inTerm = 0
		case event.Consequence != nil && event.Consequence.Kind == chargen.ConsequenceScheme:
			inTerm++
			worst = max(worst, inTerm)
		}
	}

	return worst
}

// laterRogue serves whatever career comes first, then changes into Rogue
// and back out again, so a To Begin can fail somewhere in the record
// before a later Scheme offer is made.
type laterRogue struct{ picks int }

func (d *laterRogue) Choose(c chargen.Choice) (int, error) {
	switch c.ID { //nolint:exhaustive // Only the choice points this decider steers; the rest fall through to the policy.
	case chargen.ChooseCareerChange:
		return 1, nil // "Change careers", every time it is offered
	case chargen.ChooseCareer:
		d.picks++

		if d.picks >= 2 {
			if i := slices.Index(c.Options, "Rogue"); i >= 0 {
				return i, nil
			}
		}

		return 0, nil
	case chargen.ChooseSchemeCareer:
		return 0, nil
	}

	return autoPolicy(c)
}

func (*laterRogue) Kind() chargen.DeciderKind { return chargen.DeciderPlayer }

// TestTheSchemeOfferNamesOnlyCareersServed verifies the two limits on
// "any previous career": a career whose To Begin failed was never served
// ("this career may not be used", p. 65, the reading I-54 takes), and the
// stint in progress is not previous to itself.
//
// Swept rather than measured on one run, because a To Begin failure and a
// later Scheme offer have to land in the same record for the first limit
// to be observable at all, and no single seed can be relied on for both.
// A career served and later failed is still served — the exclusion is of
// a career the record shows no term of.
func TestTheSchemeOfferNamesOnlyCareersServed(t *testing.T) {
	offered, refusals := 0, 0

	for seed := range uint64(schemeSeedSearch) {
		c := generate(t, chargen.Options{Seed: seed, Decider: &laterRogue{}})
		served, refusedOutright := careersServed(c)
		refusals += len(refusedOutright)

		for _, offer := range offers(c) {
			offered++

			for _, option := range offer.Options {
				if option != rollTheSchemeOption && !served[option] {
					t.Errorf("seed %d offered %q as a previous career; the record holds no term of it",
						seed, option)
				}
			}
		}
	}

	if offered == 0 {
		t.Fatalf("no seed under %d is offered a Scheme career; the sweep is asserting nothing", schemeSeedSearch)
	}

	if refusals == 0 {
		t.Fatalf("no seed under %d refuses a To Begin outright; the sweep cannot see the p. 65 exclusion",
			schemeSeedSearch)
	}
}

// TestAFirstCareerRogueIsOfferedNoScheme verifies the offer is gated on
// there being something to take. A Rogue who has served nothing else has
// no previous career, and a one-option prompt asking him to confirm that
// is a choice point with no choice in it.
func TestAFirstCareerRogueIsOfferedNoScheme(t *testing.T) {
	terms := 0

	for seed := range uint64(schemeSeedSearch) {
		c := generate(t, chargen.Options{Seed: seed, Career: "Rogue", Decider: &schemesOnHisPast{inRogue: true}})

		if len(c.Careers) != 1 || c.Careers[0].Career != "Rogue" {
			continue // he left; a later career could make one previous
		}

		terms += len(c.Careers[0].Terms)

		for _, event := range c.Events {
			if event.Choice != nil && event.Choice.Prompt == schemeCareerPrompt {
				t.Fatalf("seed %d: a Rogue who served nothing else was offered a previous career", seed)
			}
		}
	}

	if terms == 0 {
		t.Fatalf("no seed under %d serves a Rogue term; the sweep is asserting nothing", schemeSeedSearch)
	}
}
