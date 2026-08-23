package benefit_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/philoserf/t5chargen/career"

	"github.com/philoserf/t5chargen/benefit"
	"github.com/philoserf/t5chargen/skill"
)

func load(t *testing.T) *benefit.Table {
	t.Helper()

	table, err := benefit.Load()
	if err != nil {
		t.Fatal(err)
	}

	return table
}

// TestPassageValues pins what p. 68 prices each passage at. These are the
// numbers muster out turns a benefit into credits with, so they are worth
// checking against the page rather than against themselves.
func TestPassageValues(t *testing.T) {
	table := load(t)

	for kind, want := range map[benefit.Kind]int{
		benefit.StarPassage:   250_000,
		benefit.HighPassage:   10_000,
		benefit.MiddlePassage: 8_000,
		benefit.LowPassage:    1_000,
	} {
		b, err := table.For(kind)
		if err != nil {
			t.Errorf("%s: %v", kind, err)

			continue
		}

		if b.Credits != want {
			t.Errorf("%s is worth Cr%d, want Cr%d", kind, b.Credits, want)
		}
	}
}

// TestAnnualBenefits pins the two benefits p. 68-69 price by the year.
func TestAnnualBenefits(t *testing.T) {
	table := load(t)

	for kind, want := range map[benefit.Kind]int{
		benefit.Directorship: 36_000,
		benefit.Proxy:        100_000,
	} {
		b, err := table.For(kind)
		if err != nil {
			t.Fatal(err)
		}

		if b.AnnualCredits != want {
			t.Errorf("%s pays Cr%d a year, want Cr%d", kind, b.AnnualCredits, want)
		}
	}
}

// TestLandGrantIncome pins "Cr10,000 per TC Trade Classification ... A
// Land Grant on a World with no TCs generates Cr5,000 per year" (p. 68),
// and the page's own worked example of Cr30,000 for three.
func TestLandGrantIncome(t *testing.T) {
	table := load(t)

	grant, err := table.For(benefit.LandGrant)
	if err != nil {
		t.Fatal(err)
	}

	if grant.CreditsPerTC != 10_000 {
		t.Errorf("a Land Grant pays Cr%d per TC, want Cr10,000", grant.CreditsPerTC)
	}

	if grant.NoTCAnnualCredits != 5_000 {
		t.Errorf("a Land Grant with no TCs pays Cr%d, want Cr5,000", grant.NoTCAnnualCredits)
	}

	// The no-TC figure stands in for the per-TC income rather than
	// topping it up, so a Land Grant carries no unconditional annual.
	if grant.AnnualCredits != 0 {
		t.Errorf("a Land Grant pays Cr%d a year on top of its TCs, want nothing", grant.AnnualCredits)
	}

	if got := grant.CreditsPerTC * 3; got != 30_000 {
		t.Errorf("three TCs pay Cr%d, want the page's Cr30,000", got)
	}
}

// TestEntitlements pins chart M1's two blocks, including which begin at
// Life Stage 9 and which at muster out.
func TestEntitlements(t *testing.T) {
	table := load(t)

	for _, tt := range []struct {
		id        string
		annual    int
		perYear   int
		perTerm   int
		lifeStage int
		minTerms  int
	}{
		{id: "citizen_pension", annual: 5_000, lifeStage: 9},
		{id: "functionary_pension", annual: 15_000, lifeStage: 9},
		{id: "reserve_pension", perYear: 100, lifeStage: 9},
		{id: "professor_pension", annual: 10_000, lifeStage: 9},
		// "Armed Forces Entitlements (Annual at Muster Out)".
		{id: "enlisted_retirement", perTerm: 2_000, minTerms: 4},
		{id: "officer_retirement", perTerm: 3_000, minTerms: 4},
	} {
		t.Run(tt.id, func(t *testing.T) {
			e, ok := table.Entitlement(tt.id)
			if !ok {
				t.Fatalf("%s is not in chart M1", tt.id)
			}

			if e.AnnualCredits != tt.annual || e.CreditsPerYear != tt.perYear ||
				e.CreditsPerTerm != tt.perTerm {
				t.Errorf("%s pays %+v, want annual %d / per year %d / per term %d",
					tt.id, e, tt.annual, tt.perYear, tt.perTerm)
			}

			if e.FromLifeStage != tt.lifeStage {
				t.Errorf("%s begins at Life Stage %d, want %d", tt.id, e.FromLifeStage, tt.lifeStage)
			}

			if e.MinimumTerms != tt.minTerms {
				t.Errorf("%s needs %d terms, want %d", tt.id, e.MinimumTerms, tt.minTerms)
			}
		})
	}

	if table.CashOutYears != 5 {
		t.Errorf("cashing out pays %d years, want the five of p. 69", table.CashOutYears)
	}
}

// TestForbiddenKnowledgeIsWholeAndReal pins chart M1's 1D table: a row for
// every face, each naming a skill the Master Skill List knows.
func TestForbiddenKnowledgeIsWholeAndReal(t *testing.T) {
	table := load(t)

	for roll := 1; roll <= 6; roll++ {
		f, ok := table.ForbiddenSkillAt(roll)
		if !ok {
			t.Errorf("no Forbidden Knowledge for a roll of %d", roll)

			continue
		}

		if err := skill.Validate(f.Skill); err != nil {
			t.Errorf("roll %d grants %q: %v", roll, f.Skill, err)
		}
	}

	if _, ok := table.ForbiddenSkillAt(7); ok {
		t.Error("the 1D table answers a roll of 7")
	}
}

// TestAutomaticsAreTranscribed pins the seven rows of "Automatics (Subject
// to Eligibility)", each with the eligibility the chart prints beside it.
func TestAutomaticsAreTranscribed(t *testing.T) {
	table := load(t)

	want := map[string]string{
		"personal_weapons": "if Fighter-1+",
		"medal":            "Spacer or Soldier",
		"commendations":    "Agent",
		"fame":             "any",
		"tas_life":         "Scholar, Scout, Craftsman, SEH",
		"land_grants":      "Nobles, Scouts",
		"masterpieces":     "Craftsman",
	}

	if len(table.Automatics) != len(want) {
		t.Errorf("chart M1 lists %d automatics, want %d", len(table.Automatics), len(want))
	}

	got := make(map[string]string, len(table.Automatics))

	for _, a := range table.Automatics {
		if _, ok := got[a.ID]; ok {
			t.Errorf("automatic %q is listed twice", a.ID)
		}

		got[a.ID] = a.Eligibility
	}

	// Compare both ways: counting the rows is not enough, since a
	// duplicated row hides a missing one.
	for id, eligibility := range want {
		switch have, ok := got[id]; {
		case !ok:
			t.Errorf("chart M1 automatic %q is missing", id)
		case have != eligibility:
			t.Errorf("%s is eligible for %q, want %q", id, have, eligibility)
		}
	}

	for id := range got {
		if _, ok := want[id]; !ok {
			t.Errorf("unexpected automatic %q", id)
		}
	}
}

// TestBenefitColumns pins which of chart M1's two columns each kind sits
// in. Load-time validation only checks a kind falls in one of them, so
// without this a Knighthood filed as Financial would ship unnoticed.
func TestBenefitColumns(t *testing.T) {
	table := load(t)

	want := map[benefit.Kind]benefit.Class{
		benefit.Money:              benefit.Financial,
		benefit.StarPassage:        benefit.Financial,
		benefit.HighPassage:        benefit.Financial,
		benefit.MiddlePassage:      benefit.Financial,
		benefit.LowPassage:         benefit.Financial,
		benefit.PensionDoubling:    benefit.Financial,
		benefit.RetirementDoubling: benefit.Financial,

		benefit.Characteristic:     benefit.NonFinancial,
		benefit.WaferJack:          benefit.NonFinancial,
		benefit.ForbiddenKnowledge: benefit.NonFinancial,
		benefit.Knighthood:         benefit.NonFinancial,
		benefit.Directorship:       benefit.NonFinancial,
		benefit.Proxy:              benefit.NonFinancial,
		benefit.LifeInsurance:      benefit.NonFinancial,
		benefit.TASFellowship:      benefit.NonFinancial,
		benefit.TASLife:            benefit.NonFinancial,
		benefit.ShipShares:         benefit.NonFinancial,
		// LandGrant is deliberately absent: chart M1 prints it as an
		// Automatic, not in either Benefits column.
	}

	for kind, class := range want {
		b, err := table.For(kind)
		if err != nil {
			t.Errorf("%s: %v", kind, err)

			continue
		}

		if b.Class != class {
			t.Errorf("%s is %s, want the chart's %s column", kind, b.Class, class)
		}
	}
}

// TestUnknownKind holds the vocabulary's edge: a table D cell naming a
// benefit chart M1 does not list must be an error, not a zero value.
func TestUnknownKind(t *testing.T) {
	table := load(t)

	if _, err := table.For("a knighthood of the realm"); !errors.Is(err, benefit.ErrUnknownKind) {
		t.Errorf("an unknown kind gave %v, want ErrUnknownKind", err)
	}
}

// TestKnighthoodArithmetic pins p. 68's three clauses as data rather than
// as prose. The note beside them was wrong twice before this test existed
// — it had a Soc-11+ character keep his Social Standing, and dropped the
// non-officer substitution — and a note cannot fail a build.
func TestKnighthoodArithmetic(t *testing.T) {
	table := load(t)

	k, err := table.For(benefit.Knighthood)
	if err != nil {
		t.Fatal(err)
	}

	// "A Knighthood raises any value of Soc to B" — B is 11 in eHex.
	if k.SocFloor != 11 {
		t.Errorf("a Knighthood raises Soc to %d, want 11 (B)", k.SocFloor)
	}

	// "if the character is already Soc 11+, he receives Soc +1 instead".
	if k.AboveFloorBonus != 1 {
		t.Errorf("a Soc-11+ character receives Soc +%d, want +1", k.AboveFloorBonus)
	}

	// "In the Spacer, Soldier, and Marine careers, Knighthood is only
	// available to Officers. A non-officer receives Soc +1".
	want := []string{"Spacer", "Soldier", "Marine"}
	if !slices.Equal(k.OfficersOnlyCareers, want) {
		t.Errorf("Knighthood is officers-only in %v, want %v", k.OfficersOnlyCareers, want)
	}

	if k.NonOfficerBonus != 1 {
		t.Errorf("a non-officer receives Soc +%d, want +1", k.NonOfficerBonus)
	}

	// The careers named must be careers.
	for _, name := range k.OfficersOnlyCareers {
		if !slices.Contains(career.Available(), name) {
			t.Errorf("Knighthood names %q, which is not a career", name)
		}
	}
}
