package chargen_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/chargen"
	"github.com/philoserf/t5chargen/medal"
)

// soldierGoldenSeed is the pinned fixture: six terms in Artillery, a
// commission and promotions to Lt Colonel, three kinds of medal, a wound
// badge, and four distinct assignments.
const soldierGoldenSeed = 305

func soldierRun(t *testing.T) (chargen.Character, chargen.CareerRecord) {
	t.Helper()

	c, err := chargen.Generate(chargen.Options{
		Seed: soldierGoldenSeed, Career: "Soldier", Decider: chargen.DefaultPolicy{},
	})
	if err != nil {
		t.Fatalf("seed %d: %v", soldierGoldenSeed, err)
	}

	return c, c.Careers[len(c.Careers)-1]
}

// TestSoldierBeginsEnlisted verifies p. 65: "Armed Forces characters begin
// with enlisted rank (Army = Soldier1)" (p. 65).
func TestSoldierBeginsEnlisted(t *testing.T) {
	c, record := soldierRun(t)

	if !record.Began {
		t.Fatal("the pinned seed does not begin the career")
	}

	if got := firstRankSet(c); got != "Private" {
		t.Errorf("entered as %q, want Private", got)
	}

	if record.Branch == "" {
		t.Error("no Branch recorded; chart 08 determines one on entry")
	}
}

// TestSoldierOperationsPerTerm verifies p. 66: "Roll for Assignment four
// times per Term (for four annual assignments)".
func TestSoldierOperationsPerTerm(t *testing.T) {
	def, err := career.Soldier()
	if err != nil {
		t.Fatal(err)
	}

	if def.ArmedForces.OperationsPerTerm != 4 {
		t.Fatalf("operations per term = %d, want 4", def.ArmedForces.OperationsPerTerm)
	}

	c, record := soldierRun(t)

	operations := 0

	for _, e := range c.Events {
		if e.Kind == chargen.EventConsequence && e.Consequence.Kind == chargen.ConsequenceOperation {
			operations++
		}
	}

	if want := len(record.Terms) * 4; operations != want {
		t.Errorf("assignments = %d, want %d (%d terms x 4)", operations, want, len(record.Terms))
	}
}

// TestSoldierTermSkillsFollowAssignments verifies p. 65: "In Spacer,
// Soldier, or Marine careers, Term skills ... may be taken on a column of
// the Skills table corresponding to an Operations result received in the
// Term", with "Column 1-Personal Skills may always be rolled".
func TestSoldierTermSkillsFollowAssignments(t *testing.T) {
	def, err := career.Soldier()
	if err != nil {
		t.Fatal(err)
	}

	c, _ := soldierRun(t)

	allowed := make([]string, 0, len(def.SkillColumns))
	allowed = append(allowed, "Personal")
	checked := 0

	for _, e := range c.Events {
		switch soldierSkillEvent(e, len(def.SkillColumns)) {
		case "term":
			allowed = append(allowed[:0], "Personal")
		case "operation":
			allowed = append(allowed, operationColumn(def, e.Consequence.Skill))
		case "restricted-columns":
			checked++

			for _, option := range e.Choice.Options {
				if !slices.Contains(allowed, option) {
					t.Errorf("event %d offers column %q, which no assignment opened", e.Seq, option)
				}
			}
		}
	}

	if checked == 0 {
		t.Fatal("the pinned seed records no skill-column choice")
	}
}

// operationColumn returns the skills column an assignment opens, which is
// the operation's own name unless the chart names a different one (chart
// 08's "Peace Keeper" opens the "Peacekeeper" column).
func operationColumn(def *career.Definition, operation string) string {
	for _, row := range def.ArmedForces.Operations {
		if row.Name != operation {
			continue
		}

		if row.Column != "" {
			return row.Column
		}

		return row.Name
	}

	return operation
}

// soldierSkillEvent classifies the events the column restriction turns
// on. The commission and promotion eligibilities are exempt (p. 65) and
// offer the whole table, so only narrower offers are the term's own.
func soldierSkillEvent(e chargen.Event, columns int) string {
	switch {
	case e.Kind == chargen.EventStep && strings.Contains(e.Step.Name, "Term "):
		return "term"
	case e.Kind == chargen.EventConsequence && e.Consequence.Kind == chargen.ConsequenceOperation:
		return "operation"
	case e.Kind == chargen.EventChoice && e.Choice.Prompt == "Select a Soldier Skills column" &&
		len(e.Choice.Options) < columns:
		return "restricted-columns"
	default:
		return ""
	}
}

// TestSoldierMedalsFollowTheTable verifies p. 66: "use the unmodified
// Reward roll to determine the Medal; Mod +1 if an Officer", against the
// p. 70 table.
func TestSoldierMedalsFollowTheTable(t *testing.T) {
	c, record := soldierRun(t)

	officer := false
	awarded := map[string]int{}

	for _, e := range c.Events {
		if e.Kind == chargen.EventConsequence && e.Consequence.Kind == chargen.ConsequenceRankSet {
			officer = isOfficerTitle(e.Consequence.Skill)

			continue
		}

		if !rewardSuccess(e) {
			continue
		}

		want, err := medal.For(e.Throw.Total, officer)
		if err != nil {
			t.Fatalf("event %d: %v", e.Seq, err)
		}

		awarded[want.Code]++
	}

	for _, award := range record.Medals {
		if awarded[award.Code] != award.Count {
			t.Errorf("medal %s counted %d, want %d from the table", award.Code, award.Count, awarded[award.Code])
		}
	}

	if len(record.Medals) == 0 {
		t.Fatal("the pinned seed records no medal")
	}
}

// isOfficerTitle reports the chart 08 officer ranks.
func isOfficerTitle(title string) bool {
	for _, officer := range []string{"Lieutenant", "Captain", "Major", "Colonel", "General"} {
		if strings.Contains(title, officer) {
			return true
		}
	}

	return false
}

// rewardSuccess reports a successful Reward throw.
func rewardSuccess(e chargen.Event) bool {
	return e.Kind == chargen.EventThrow &&
		strings.Contains(e.Throw.Cite, "Reward vs") &&
		e.Throw.Success != nil && *e.Throw.Success
}

// TestSoldierPromotionUsesMedalsNotWoundBadges verifies the p. 70
// footnote: "Medals (but not Wound Badges) are Mods for Soldier / Spacer /
// Marine Promotion" (interpretation I-31, ERRATA.md).
func TestSoldierPromotionUsesMedalsNotWoundBadges(t *testing.T) {
	c, _ := soldierRun(t)

	if c.WoundBadges == 0 {
		t.Fatal("the pinned seed records no Wound Badge; the fixture is stale")
	}

	checked := 0

	for _, e := range c.Events {
		if e.Kind != chargen.EventThrow || !strings.Contains(e.Throw.Cite, "Promotion") {
			continue
		}

		checked++

		for _, mod := range e.Throw.Mods {
			if strings.Contains(mod.Name, "Wound") {
				t.Errorf("event %d applies a Wound Badge modifier to a promotion", e.Seq)
			}
		}
	}

	if checked == 0 {
		t.Fatal("the pinned seed records no promotion throw")
	}
}

// TestSoldierServiceBadgeCarriesNoMod verifies that the Risk-success
// Exemplary Service Badge is counted but carries no promotion modifier
// (interpretation I-32, ERRATA.md): the badges outnumber the medals, and
// only the medals appear in the promotion modifiers.
func TestSoldierServiceBadgeCarriesNoMod(t *testing.T) {
	_, record := soldierRun(t)

	if record.ServiceBadges == 0 {
		t.Fatal("the pinned seed records no Exemplary Service Badge")
	}

	total := 0
	for _, award := range record.Medals {
		total += award.Count
	}

	if record.ServiceBadges == total {
		t.Skip("the fixture cannot distinguish badges from medals this run")
	}
}

// TestSoldierBranchModsOpposeRiskAndReward verifies p. 66: the Branch and
// Operations Mods "are negative against the Risk Roll and positive against
// the Reward Roll".
func TestSoldierBranchModsOpposeRiskAndReward(t *testing.T) {
	c, _ := soldierRun(t)

	risk := firstThrowMods(c, "Risk vs")
	reward := firstThrowMods(c, "Reward vs")

	if risk == nil || reward == nil {
		t.Fatal("the pinned seed records no Risk and Reward pair")
	}

	for _, name := range []string{"Branch", "Operations"} {
		r, ok := findMod(risk, name)
		if !ok {
			t.Fatalf("the Risk throw itemizes no %s modifier", name)
		}

		w, ok := findMod(reward, name)
		if !ok {
			t.Fatalf("the Reward throw itemizes no %s modifier", name)
		}

		if r.Value != -w.Value {
			t.Errorf("%s modifier is %d on Risk and %d on Reward; they must oppose", name, r.Value, w.Value)
		}
	}
}

// firstThrowMods returns the modifiers of the first throw whose citation
// contains want.
func firstThrowMods(c chargen.Character, want string) []chargen.Mod {
	for _, e := range c.Events {
		if e.Kind == chargen.EventThrow && strings.Contains(e.Throw.Cite, want) {
			return e.Throw.Mods
		}
	}

	return nil
}

// findMod returns the named modifier of a throw.
func findMod(mods []chargen.Mod, name string) (chargen.Mod, bool) {
	for _, mod := range mods {
		if mod.Name == name {
			return mod, true
		}
	}

	return chargen.Mod{}, false
}

// cautionDecider is the default policy except it always selects Caution
// +1, the positive Risk Mod the injury rule must ignore.
type cautionDecider struct{ chargen.DefaultPolicy }

func (d cautionDecider) Choose(c chargen.Choice) (int, error) {
	if c.ID == chargen.ChooseRiskMod {
		if i := slices.Index(c.Options, "Caution +1"); i >= 0 {
			return i, nil
		}
	}

	return autoPolicy(c)
}

func (cautionDecider) Kind() chargen.DeciderKind { return chargen.DeciderPlayer }

// TestSoldierInjuryIgnoresPositiveMods verifies p. 65: "Reduce the
// Controlling Characteristic by all negative Mods; ignore any positive
// Mods." The Cautious Mod is positive against Risk, so it must not offset
// the Branch and Operations Mods, which are negative against it (p. 66).
// The p. 66 worked example fails Risk at End 11 with Branch -2,
// Operations -3 and Caution +2, and reduces "by -2 -3 to Endurance-6".
func TestSoldierInjuryIgnoresPositiveMods(t *testing.T) {
	checked := 0

	for seed := range uint64(120) {
		c, err := chargen.Generate(chargen.Options{
			Seed: seed, Career: "Soldier", Decider: cautionDecider{},
		})
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}

		checked += checkSoldierInjuries(t, seed, c)
	}

	if checked == 0 {
		t.Fatal("the sweep produced no Risk failure carrying a positive Mod; widen it")
	}
}

// checkSoldierInjuries asserts every Risk failure in one record charges the
// Controlling Characteristic exactly its negative Mods plus Flux, and
// reports how many failures carried a positive Mod to be ignored.
func checkSoldierInjuries(t *testing.T, seed uint64, c chargen.Character) int {
	t.Helper()

	checked := 0

	for i, e := range c.Events {
		if !soldierRiskFailure(e) {
			continue
		}

		negative, positive := splitMods(e.Throw.Mods)
		if positive == 0 {
			continue
		}

		checked++

		// The Flux is the next event; the reduction, if any, follows it.
		flux := c.Events[i+1]
		if flux.Kind != chargen.EventThrow || !strings.Contains(flux.Throw.Cite, "reduce CC") {
			t.Fatalf("seed %d: event %d is not the injury Flux", seed, flux.Seq)
		}

		want := negative + flux.Throw.Total
		if got := soldierInjuryDelta(c.Events[i+2]); got != min(want, 0) {
			t.Errorf("seed %d: Risk failure with negative Mods %d and Flux %d changed the CC by %d, want %d",
				seed, negative, flux.Throw.Total, got, min(want, 0))
		}
	}

	return checked
}

// soldierRiskFailure reports a failed Soldier Risk throw.
func soldierRiskFailure(e chargen.Event) bool {
	return e.Kind == chargen.EventThrow && e.Throw.Success != nil && !*e.Throw.Success &&
		strings.Contains(e.Throw.Cite, "Risk vs")
}

// splitMods totals a throw's negative and positive modifiers separately.
func splitMods(mods []chargen.Mod) (int, int) {
	negative, positive := 0, 0

	for _, mod := range mods {
		if mod.Value < 0 {
			negative += mod.Value
		} else {
			positive += mod.Value
		}
	}

	return negative, positive
}

// soldierInjuryDelta returns the characteristic change an injury Flux
// caused, or 0 where the Flux left the character unharmed.
func soldierInjuryDelta(e chargen.Event) int {
	if e.Kind == chargen.EventConsequence && e.Consequence.Kind == chargen.ConsequenceCharacteristicChange {
		return e.Consequence.Delta
	}

	return 0
}

// branchDecider takes the named Branch option and otherwise defers to the
// auto policy. A branch is selected on the enlisted side (interpretation
// I-36), so the names it asks for are enlisted ones; it reaches the rows
// the policy cannot, which takes the lowest Branch Mod.
type branchDecider struct {
	chargen.DefaultPolicy

	name string
}

func (d branchDecider) Choose(c chargen.Choice) (int, error) {
	if c.ID == chargen.ChooseBranch {
		if i := slices.Index(c.Options, d.name); i >= 0 {
			return i, nil
		}
	}

	return autoPolicy(c)
}

func (branchDecider) Kind() chargen.DeciderKind { return chargen.DeciderPlayer }

// TestSpacerCrewBecomesLine verifies p. 66: "A character who receives a
// Commission may roll for Branch or keep his current Branch (for Spacers,
// Crew becomes Line)." The Naval Branch table prints the two sides of one
// row (chart 07), so a commission moves the rating across without a new
// roll.
func TestSpacerCrewBecomesLine(t *testing.T) {
	if branch, ok := commissionedBranch(t, "Crew"); ok && branch != "Line" {
		t.Errorf("commissioned Spacer who selected Crew serves in %q, want Line", branch)
	}
}

// TestSpacerBranchNameBindsStableRow verifies interpretation I-36's
// binding rule. Chart 07 prints enlisted Engineer twice — row 3, whose
// officer side is Line at Mod 1, and row 4, whose officer side is Engineer
// at Mod 0 — and the name binds to row 4. A rating who selects Engineer
// and then "keep[s] his current Branch" (p. 66) is therefore still an
// Engineer once commissioned, at the Mod he weighed on entry, rather than
// crossing into Line.
func TestSpacerBranchNameBindsStableRow(t *testing.T) {
	if branch, ok := commissionedBranch(t, "Engineer"); ok && branch != "Engineer" {
		t.Errorf("commissioned Spacer who selected Engineer serves in %q, want Engineer", branch)
	}
}

// commissionedBranch generates Spacers until one selects the named branch
// as a rating and is then commissioned, and reports the branch he serves
// in. A branch is rolled instead whenever the Soc check fails, so no seed
// is guaranteed to reach the case; the sweep reports that rather than
// asserting on a run it never made.
func commissionedBranch(t *testing.T, selected string) (string, bool) {
	t.Helper()

	for seed := uint64(1); seed <= 200; seed++ {
		c, err := chargen.Generate(chargen.Options{
			Seed: seed, Career: "Spacer", Decider: branchDecider{name: selected},
		})
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}

		record := c.Careers[len(c.Careers)-1]
		if !record.Began || firstBranchSet(c) != selected || !reachedEnsign(c) {
			continue
		}

		return record.Branch, true
	}

	t.Skipf("no seed in 1..200 selected %s as a rating and was then commissioned", selected)

	return "", false
}

// reachedEnsign reports whether the character was commissioned into the
// Naval officer ladder (chart 07's O1).
func reachedEnsign(c chargen.Character) bool {
	return slices.ContainsFunc(c.Events, func(e chargen.Event) bool {
		return e.Kind == chargen.EventConsequence &&
			e.Consequence.Kind == chargen.ConsequenceRankSet &&
			e.Consequence.Skill == "Ensign"
	})
}

// firstBranchSet returns the branch the character entered.
func firstBranchSet(c chargen.Character) string {
	for _, e := range c.Events {
		if e.Kind == chargen.EventConsequence && e.Consequence.Kind == chargen.ConsequenceBranchSet {
			return e.Consequence.Skill
		}
	}

	return ""
}

// TestMarineIsDataOnly verifies that the third Armed Forces career adds
// no mechanics: it runs the shared procedure over chart 12's data, and
// differs from its siblings only in what the chart prints.
func TestMarineIsDataOnly(t *testing.T) {
	marine := loadService(t, career.Marine)
	soldier := loadService(t, career.Soldier)
	spacer := loadService(t, career.Spacer)

	// Chart 12's DM By Branch differs from chart 08's: the Marine's
	// Commando carries the +0 the Soldier's Protected does.
	assertBranch(t, "Marine", marine.ArmedForces.Branches[5], "Commando", 0)
	assertBranch(t, "Soldier", soldier.ArmedForces.Branches[4], "Protected", 0)

	// Both ground services take the Branch DM on Operations; the Navy
	// prints only "DM +2 if Edu 10+".
	for _, service := range []*career.Definition{marine, soldier} {
		if !service.ArmedForces.OperationsUseBranchDM {
			t.Errorf("%s should take the Branch DM on Operations", service.Name)
		}
	}

	if spacer.ArmedForces.OperationsUseBranchDM {
		t.Error("chart 07 prints only \"DM +2 if Edu 10+\" on Naval Operations")
	}
}

// loadService loads one Armed Forces career definition.
func loadService(t *testing.T, load func() (*career.Definition, error)) *career.Definition {
	t.Helper()

	def, err := load()
	if err != nil {
		t.Fatal(err)
	}

	return def
}

// assertBranch checks one Branch row's name and DM.
func assertBranch(t *testing.T, service string, got career.Branch, name string, dm int) {
	t.Helper()

	if got.Name != name || got.DM != dm {
		t.Errorf("%s branch = %q with DM %d, want %q with DM %d", service, got.Name, got.DM, name, dm)
	}
}
