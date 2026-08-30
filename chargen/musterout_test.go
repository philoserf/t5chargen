package chargen_test

import (
	"slices"
	"testing"

	"github.com/philoserf/t5chargen/benefit"
	"github.com/philoserf/t5chargen/chargen"
)

// benefitsColumn takes the Benefits column at every muster-out roll, which
// the auto policy never does — it falls back to first-listed, and Money is
// listed first on every table D (POLICY.md `select_benefit_column`).
type benefitsColumn struct{}

func (benefitsColumn) Choose(c chargen.Choice) (int, error) {
	if c.ID == chargen.ChooseBenefitColumn {
		return 1, nil
	}

	return autoPolicy(c)
}

func (benefitsColumn) Kind() chargen.DeciderKind { return chargen.DeciderPlayer }

// benefitsOf gathers a character's muster-out awards by kind.
func benefitsOf(c chargen.Character) map[benefit.Kind]int {
	got := map[benefit.Kind]int{}
	for _, b := range c.Benefits {
		got[b.Kind] += max(b.Count, 1)
	}

	return got
}

// TestMusterOutRollCount pins p. 68: "A character is allowed one Mustering
// Out roll for each term served in Career Resolution. He is allowed one
// additional roll per Commendation, MCG, or SEH. He is allowed one
// additional roll if Fame 19+".
func TestMusterOutRollCount(t *testing.T) {
	// Both deciders: the auto policy takes Money every time, so the
	// Benefits column — and with it the benefits that can be lost to the
	// characteristic maximum — is only reached by the other.
	for _, decider := range []chargen.Decider{chargen.DefaultPolicy{}, benefitsColumn{}} {
		for seed := uint64(1); seed <= 200; seed++ {
			assertRollCount(t, seed, decider)
		}
	}
}

// assertRollCount checks one character's benefit count against the rolls
// p. 68 allows him.
func assertRollCount(t *testing.T, seed uint64, decider chargen.Decider) {
	t.Helper()

	c, err := chargen.Generate(chargen.Options{Seed: seed, Decider: decider})
	if err != nil {
		t.Fatalf("seed %d: %v", seed, err)
	}

	// A dead character does not muster out at all (I-77).
	if c.Dead {
		if len(c.Benefits) != 0 {
			t.Fatalf("seed %d: a dead character took %d benefits", seed, len(c.Benefits))
		}

		return
	}

	want := 0

	for _, record := range c.Careers {
		if record.Began {
			want += rollsAllowed(record)
		}
	}

	if c.Fame >= 19 && want > 0 {
		want++
	}

	// A characteristic improvement that would pass the human maximum "is
	// lost" (p. 68) and records nothing, though it consumed a roll.
	if got := len(c.Benefits) + lostToTheMaximum(c); got != want {
		t.Fatalf("seed %d: %d benefits and %d lost for %d rolls",
			seed, len(c.Benefits), lostToTheMaximum(c), want)
	}
}

// rollsAllowed counts a career's muster-out rolls: one per term, one per
// Commendation, MCG or SEH, doubled by a Disability Muster Out (pp. 68-69).
func rollsAllowed(record chargen.CareerRecord) int {
	rolls := len(record.Terms) + record.Commendations

	for _, award := range record.Medals {
		if award.Code == "MCG" || award.Code == "SEH" || award.Code == "*SEH*" {
			rolls += award.Count
		}
	}

	if record.Disabled {
		rolls *= 2
	}

	return rolls
}

// lostToTheMaximum counts characteristic improvements p. 68's human
// maximum refused. Each consumed a roll and recorded nothing.
func lostToTheMaximum(c chargen.Character) int {
	lost := 0

	for _, e := range c.Events {
		if e.Kind == chargen.EventConsequence &&
			e.Consequence.Kind == chargen.ConsequenceBenefitLost &&
			e.Consequence.Characteristic != "" {
			lost++
		}
	}

	return lost
}

// TestDoubleBenefitsDoublesTheCount pins the wording precisely: "Double
// Benefits is twice the count of Benefits (rather than two of each
// Benefit)" (p. 69).
// It sweeps rather than pinning a seed. A pinned Soldier used to carry
// this rule, and when the fixture stopped being disabled the test skipped
// instead of failing — COVERAGE.md went on calling p. 69 covered while
// nothing checked it. The sweep cannot go quiet the same way: if no seed
// reaches a Disability Muster Out, that is a failure asking for a wider
// bound.
func TestDoubleBenefitsDoublesTheCount(t *testing.T) {
	checked := 0

	for seed := range uint64(200) {
		c := generate(t, chargen.Options{Seed: seed, Career: "Soldier"})

		// A characteristic improvement past the human maximum "is lost"
		// (p. 68): it consumed a roll and recorded nothing, so the
		// benefit count only equals the roll count when none was lost.
		if lostToTheMaximum(c) != 0 {
			continue
		}

		for i := range c.Careers {
			if !c.Careers[i].Disabled {
				continue
			}

			checked++

			checkDoubledBenefits(t, seed, c, c.Careers[i])
		}
	}

	if checked == 0 {
		t.Error("no seed in 200 mustered out disabled; widen the sweep")
	}
}

// checkDoubledBenefits asserts one disabled career drew twice its
// undoubled roll count in benefits.
func checkDoubledBenefits(t *testing.T, seed uint64, c chargen.Character, disabled chargen.CareerRecord) {
	t.Helper()

	from := 0

	for _, b := range c.Benefits {
		if b.Career == disabled.Career {
			from++
		}
	}

	if want := rollsAllowed(disabled); from != want {
		t.Errorf("seed %d: a disabled %s took %d benefits, want %d — Double Benefits is twice the count",
			seed, disabled.Career, from, want)
	}
}

// TestEveryBenefitIsInTheVocabulary holds that a muster-out award always
// resolves to a chart M1 benefit, never to a bare string.
func TestEveryBenefitIsInTheVocabulary(t *testing.T) {
	table, err := benefit.Load()
	if err != nil {
		t.Fatal(err)
	}

	for seed := uint64(1); seed <= 100; seed++ {
		for _, decider := range []chargen.Decider{chargen.DefaultPolicy{}, benefitsColumn{}} {
			c, err := chargen.Generate(chargen.Options{Seed: seed, Decider: decider})
			if err != nil {
				t.Fatalf("seed %d: %v", seed, err)
			}

			for _, got := range c.Benefits {
				entry, err := table.For(got.Kind)
				if err != nil {
					t.Fatalf("seed %d: %v", seed, err)
				}

				if got.Name != entry.Name {
					t.Errorf("seed %d: %q recorded as %q", seed, entry.Name, got.Name)
				}
			}
		}
	}
}

// TestCharacteristicBenefitRespectsTheMaximum pins p. 68: "Characteristics
// for Humans cannot exceed 15. If a benefit elevates a characteristic
// above 15, that benefit is lost".
func TestCharacteristicBenefitRespectsTheMaximum(t *testing.T) {
	awarded := 0

	for seed := uint64(1); seed <= 300; seed++ {
		c, err := chargen.Generate(chargen.Options{Seed: seed, Decider: benefitsColumn{}})
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}

		assertUnderTheMaximum(t, seed, c)

		awarded += benefitsOf(c)[benefit.Characteristic]
	}

	if awarded == 0 {
		t.Error("no characteristic improvement was awarded; the rule went untested")
	}

	// Three losses in four thousand seeds, so the case is pinned rather
	// than swept for.
	assertCapLoses(t, 2418, "Soc")
	assertCapLoses(t, 2633, "Int")
}

// assertUnderTheMaximum holds "Characteristics for Humans cannot exceed
// 15" (p. 68).
func assertUnderTheMaximum(t *testing.T, seed uint64, c chargen.Character) {
	t.Helper()

	for _, value := range []int{
		c.Characteristics.Str, c.Characteristics.Dex, c.Characteristics.End,
		c.Characteristics.Int, c.Characteristics.Edu, c.Characteristics.Soc,
	} {
		if value > chargen.CharacteristicMax {
			t.Fatalf("seed %d: a characteristic reached %d, above the human maximum", seed, value)
		}
	}
}

// assertCapLoses holds p. 68's "If a benefit elevates a characteristic
// above 15, that benefit is lost" on a seed that reaches the maximum.
func assertCapLoses(t *testing.T, seed uint64, characteristic string) {
	t.Helper()

	c, err := chargen.Generate(chargen.Options{Seed: seed, Decider: benefitsColumn{}})
	if err != nil {
		t.Fatal(err)
	}

	lost := 0

	for _, e := range c.Events {
		if e.Kind == chargen.EventConsequence &&
			e.Consequence.Kind == chargen.ConsequenceBenefitLost &&
			e.Consequence.Characteristic == characteristic {
			lost++
		}
	}

	if lost == 0 {
		t.Errorf("seed %d no longer loses a %s improvement to the maximum; find and pin another",
			seed, characteristic)

		return
	}

	// A lost benefit consumed its roll and recorded nothing, so the
	// record falls short of the rolls by exactly the losses.
	rolls := 0

	for _, record := range c.Careers {
		if record.Began {
			rolls += len(record.Terms) + record.Commendations
		}
	}

	if len(c.Benefits) >= rolls {
		t.Errorf("seed %d lost %d improvements to the maximum but recorded %d benefits for %d rolls; "+
			"a lost benefit must record nothing", seed, lost, len(c.Benefits), rolls)
	}
}

// TestKnighthoodFollowsItsThreeClauses pins p. 68: a Knighthood "raises
// any value of Soc to B; if the character is already Soc 11+, he receives
// Soc +1 instead", and "In the Spacer, Soldier, and Marine careers,
// Knighthood is only available to Officers. A non-officer receives Soc +1".
func TestKnighthoodFollowsItsThreeClauses(t *testing.T) {
	table, err := benefit.Load()
	if err != nil {
		t.Fatal(err)
	}

	knight, err := table.For(benefit.Knighthood)
	if err != nil {
		t.Fatal(err)
	}

	titled, services := 0, 0

	for _, name := range []string{"Merchant", "Soldier", "Spacer", "Marine", "Agent"} {
		restricted := slices.Contains(knight.OfficersOnlyCareers, name)

		for seed := uint64(1); seed <= 80; seed++ {
			c, err := chargen.Generate(chargen.Options{Seed: seed, Career: name, Decider: benefitsColumn{}})
			if err != nil || benefitsOf(c)[benefit.Knighthood] == 0 {
				continue
			}

			titled++

			if restricted {
				services++
			}

			assertKnightedProperly(t, name, seed, c, knight, restricted)
		}
	}

	if titled == 0 {
		t.Error("nobody was knighted; the rule went untested")
	}

	if services == 0 {
		t.Error("no Armed Forces knighthood; the officers-only clause went untested")
	}

	assertNonOfficersAreSubstituted(t, knight)
}

// assertNonOfficersAreSubstituted holds the half of the rule that the
// knighted characters cannot show: "In the Spacer, Soldier, and Marine
// careers, Knighthood is only available to Officers. A non-officer
// receives Soc +1" (p. 68).
//
// Without this the officers-only clause could be deleted and every test
// still pass, because every Armed Forces character who rolls a Knighthood
// in the sweep happens already to be an officer.
func assertNonOfficersAreSubstituted(t *testing.T, knight benefit.Benefit) {
	t.Helper()

	substituted := 0

	for _, name := range knight.OfficersOnlyCareers {
		for seed := uint64(1); seed <= 400; seed++ {
			c, err := chargen.Generate(chargen.Options{Seed: seed, Career: name, Decider: benefitsColumn{}})
			if err != nil {
				continue
			}

			substituted += countSubstitutions(t, name, seed, c)
		}
	}

	if substituted == 0 {
		t.Error("no non-officer was given Soc in place of a Knighthood; the substitution went untested")
	}
}

// careerNamed returns a character's record for a career he entered.
func careerNamed(c chargen.Character, name string) *chargen.CareerRecord {
	for i := range c.Careers {
		if c.Careers[i].Career == name && c.Careers[i].Began {
			return &c.Careers[i]
		}
	}

	return nil
}

// assertKnightedProperly checks one knighted character: Soc reached the
// floor, and in a restricted career he is an officer (p. 68).
func assertKnightedProperly(
	t *testing.T, name string, seed uint64, c chargen.Character,
	knight benefit.Benefit, restricted bool,
) {
	t.Helper()

	if c.Characteristics.Soc < knight.SocFloor {
		t.Errorf("%s seed %d: knighted at Soc %d, below the floor of %d",
			name, seed, c.Characteristics.Soc, knight.SocFloor)
	}

	if !restricted {
		return
	}

	if record := careerNamed(c, name); record != nil && (record.Rank == "" || record.Rank[0] != 'O') {
		t.Errorf("%s seed %d: a non-officer at %q was knighted", name, seed, record.Rank)
	}
}

// countSubstitutions reports how many Social Standing awards a
// non-officer took in place of a Knighthood, and fails if he took the
// title instead.
func countSubstitutions(t *testing.T, name string, seed uint64, c chargen.Character) int {
	t.Helper()

	record := careerNamed(c, name)
	if record == nil || (record.Rank != "" && record.Rank[0] == 'O') {
		return 0
	}

	if benefitsOf(c)[benefit.Knighthood] > 0 {
		t.Errorf("%s seed %d: a non-officer at %q holds a Knighthood", name, seed, record.Rank)
	}

	got := 0

	for _, b := range c.Benefits {
		if b.Kind == benefit.Characteristic && b.Detail == "Soc" && b.Career == name {
			got++
		}
	}

	return got
}

// TestFameBenefitRaisesFame pins interpretation I-72: charts 02, 03, 05 and
// 09 award "Fame +1" or "Fame +2" as a table D benefit cell, and a cell
// that moved nothing would award "something he has by default, which is no
// award at all".
func TestFameBenefitRaisesFame(t *testing.T) {
	awarded := 0

	// The four careers whose table D prints a Fame cell (charts 02, 03,
	// 05 and 09); the auto policy never opens with most of them.
	for _, name := range []string{"Scholar", "Entertainer", "Scout", "Agent"} {
		for seed := uint64(1); seed <= 120; seed++ {
			c, err := chargen.Generate(chargen.Options{
				Seed: seed, Career: name, Decider: benefitsColumn{},
			})
			if err != nil {
				t.Fatalf("%s seed %d: %v", name, seed, err)
			}

			from := benefitsOf(c)[benefit.Fame]
			if from > 0 {
				awarded++
			}

			if want := computedFame(c) + from; c.Fame != want {
				t.Fatalf("%s seed %d: Fame %d, want %d (chart F %d plus %d awarded at muster out)",
					name, seed, c.Fame, want, computedFame(c), from)
			}
		}
	}

	if awarded == 0 {
		t.Fatal("no Fame benefit awarded: the cell was never exercised")
	}
}

// computedFame reads the Fame chart F calculated, before muster out added
// to it.
func computedFame(c chargen.Character) int {
	for _, e := range c.Events {
		if e.Kind == chargen.EventConsequence &&
			e.Consequence.Kind == chargen.ConsequenceFameComputed {
			return e.Consequence.Value
		}
	}

	return 0
}

// TestBenefitsAreHeldNotSold pins interpretation I-76: chart M1 prices the
// passages and the Proxy, but a value is what a benefit is worth, not what
// it pays. Only the Money column pays credits.
//
// The engine did the other thing first, and a Craftsman showed Cr575,000
// of which Cr250,000 was an unsold StarPass his own sheet also listed
// among his benefits.
func TestBenefitsAreHeldNotSold(t *testing.T) {
	held := 0

	// Both deciders. The passages and StarPass — the priced benefits that
	// are not cash — sit in the Money column, which only the auto policy
	// takes, and they are what the engine cashed out by mistake.
	for _, decider := range []chargen.Decider{chargen.DefaultPolicy{}, benefitsColumn{}} {
		held += assertHeldNotSold(t, decider)
	}

	if held == 0 {
		t.Error("no priced benefit was held; the rule went untested")
	}
}

// assertHeldNotSold checks that a character's credits come only from Money
// cells, and reports how many priced benefits he holds unsold.
func assertHeldNotSold(t *testing.T, decider chargen.Decider) int {
	t.Helper()

	held := 0

	for seed := uint64(1); seed <= 300; seed++ {
		c, err := chargen.Generate(chargen.Options{Seed: seed, Decider: decider})
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}

		paid := 0

		for _, got := range c.Benefits {
			if got.Kind == benefit.Money {
				paid += got.Credits

				continue
			}

			held++

			if got.Credits != 0 {
				t.Errorf("seed %d: a held %s was cashed out for Cr%d", seed, got.Name, got.Credits)
			}
		}

		if c.Credits != paid {
			t.Errorf("seed %d: Cr%d in credits from Cr%d of money cells", seed, c.Credits, paid)
		}
	}

	return held
}

// TestTheDeadDoNotMusterOut pins interpretation I-77. Muster out notes
// "assets for the adventuring situations to come" (p. 67), and a character
// who died in generation has none.
func TestTheDeadDoNotMusterOut(t *testing.T) {
	dead := 0

	for seed := uint64(1); seed <= 400; seed++ {
		c := generate(t, chargen.Options{Seed: seed})
		if !c.Dead {
			continue
		}

		dead++

		if c.Credits != 0 || len(c.Benefits) != 0 {
			t.Errorf("seed %d: a dead character mustered out with Cr%d and %d benefits",
				seed, c.Credits, len(c.Benefits))
		}
	}

	if dead == 0 {
		t.Error("nobody died in the sweep; the rule went untested")
	}
}

// TestNoDuplicateBenefitSurvivesAReroll pins p. 69's rule as it is
// written: "may be rerolled until a different benefit is received", not
// rerolled once and then accepted.
//
// The invariant is checkable on every character, which the reroll itself
// is not: a broad sweep produces single rerolls in quantity and
// consecutive ones almost never, so counting rerolls proves nothing. What
// the rule guarantees is the outcome — a character never ends holding two
// of a benefit that is worth nothing twice, unless the rows his throw
// could reach had nothing else to give.
func TestNoDuplicateBenefitSurvivesAReroll(t *testing.T) {
	held := 0

	for _, decider := range []chargen.Decider{chargen.DefaultPolicy{}, benefitsColumn{}} {
		for seed := uint64(1); seed <= 400; seed++ {
			c, err := chargen.Generate(chargen.Options{Seed: seed, Decider: decider})
			if err != nil {
				t.Fatalf("seed %d: %v", seed, err)
			}

			counts := map[benefit.Kind]int{}

			for _, got := range c.Benefits {
				counts[got.Kind]++
			}

			for _, kind := range []benefit.Kind{
				benefit.WaferJack, benefit.TASLife, benefit.TASFellowship,
				benefit.Knighthood, benefit.LifeInsurance,
			} {
				if counts[kind] > 1 && !exhaustedItsTable(c, kind) {
					t.Errorf("seed %d: %d of %s, which is worth nothing twice", seed, counts[kind], kind)
				}

				if counts[kind] == 1 {
					held++
				}
			}
		}
	}

	if held == 0 {
		t.Error("no rerollable benefit was ever awarded; the rule went untested")
	}

	assertRerollsUntilDifferent(t)
}

// assertRerollsUntilDifferent pins the loop itself. A duplicate is
// rerolled "until a different benefit is received" (p. 69), so a run of
// consecutive rerolls must be possible; rerolling once and accepting the
// second duplicate would cap every run at one.
//
// Pinned rather than swept for: a Citizen sweep produces single rerolls in
// quantity and consecutive ones never, which is how rerolling once passed
// unnoticed. Chart 02's Benefits column repeats its rerollable cells often
// enough that seed 72 runs twelve deep.
func assertRerollsUntilDifferent(t *testing.T) {
	t.Helper()

	c, err := chargen.Generate(chargen.Options{Seed: 72, Career: "Scholar", Decider: benefitsColumn{}})
	if err != nil {
		t.Fatal(err)
	}

	run, longest := 0, 0

	for _, e := range c.Events {
		if e.Kind != chargen.EventConsequence {
			continue
		}

		if e.Consequence.Kind == chargen.ConsequenceBenefitLost &&
			e.Consequence.Skill != "" && e.Consequence.Characteristic == "" {
			run++
			longest = max(longest, run)

			continue
		}

		run = 0
	}

	if longest < 2 {
		t.Errorf("the pinned seed's longest reroll run is %d; the rule rerolls until the benefit differs, "+
			"so a run of two must be reachable", longest)
	}
}

// exhaustedItsTable reports whether the character's log shows a reroll
// span that ran out of alternatives — the one case where p. 69's rule has
// nothing left to give and a duplicate stands (I-74).
func exhaustedItsTable(c chargen.Character, kind benefit.Kind) bool {
	for _, e := range c.Events {
		if e.Kind == chargen.EventConsequence &&
			e.Consequence.Kind == chargen.ConsequenceBenefitLost &&
			e.Consequence.Skill == string(kind) {
			return true
		}
	}

	return false
}
