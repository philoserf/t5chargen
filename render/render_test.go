package render_test

import (
	"flag"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/chargen"
	"github.com/philoserf/t5chargen/render"
)

// update rewrites the golden fixtures instead of comparing against them:
// `task goldens`, or `go test ./render -update`.
var update = flag.Bool("update", false, "rewrite testdata golden files instead of comparing")

// golden compares got against the named testdata file, or rewrites it
// under -update.
func golden(t *testing.T, got, file string) {
	t.Helper()

	if *update {
		if err := os.WriteFile(file, []byte(got), 0o600); err != nil {
			t.Fatal(err)
		}

		return
	}

	want, err := os.ReadFile(file) //nolint:gosec // G304: fixed test-owned paths under testdata/.
	if err != nil {
		t.Fatal(err)
	}

	if got != string(want) {
		t.Errorf("output differs from %s:\n%s", file, got)
	}
}

// generate builds a character for render tests with the auto policy.
func generate(t *testing.T, opts chargen.Options) chargen.Character {
	t.Helper()

	opts.Decider = chargen.DefaultPolicy{}

	c, err := chargen.Generate(opts)
	if err != nil {
		t.Fatal(err)
	}

	return c
}

func TestSheetGolden(t *testing.T) {
	golden(t, render.Sheet(generate(t, chargen.Options{Seed: 1})), "testdata/seed1_sheet.md")
}

func TestHistoryGolden(t *testing.T) {
	golden(t, render.History(generate(t, chargen.Options{Seed: 1})), "testdata/seed1_history.md")
}

// TestScoutGoldens pin the Scout sheet and transcript: the career whose
// record exercises the status line (Fame, Wound Badges), Discoveries, and
// the injury consequences.
func TestScoutSheetGolden(t *testing.T) {
	golden(t, render.Sheet(generate(t, chargen.Options{Seed: 26, Career: "Scout"})), "testdata/scout_sheet.md")
}

func TestScoutHistoryGolden(t *testing.T) {
	golden(t, render.History(generate(t, chargen.Options{Seed: 26, Career: "Scout"})), "testdata/scout_history.md")
}

// TestMerchantSheetGolden pins the sheet for the career with rank: the
// rank line and Ship Shares (chart 06, p. 80).
func TestMerchantSheetGolden(t *testing.T) {
	golden(t, render.Sheet(generate(t, chargen.Options{Seed: 17, Career: "Merchant"})), "testdata/merchant_sheet.md")
}

// TestEntertainerSheetGolden pins the sheet for the career with no rank
// but its own tracked values: specialty, Talent, and Fame (chart 03).
func TestEntertainerSheetGolden(t *testing.T) {
	golden(t, render.Sheet(generate(t, chargen.Options{Seed: 572, Career: "Entertainer"})),
		"testdata/entertainer_sheet.md")
}

// TestScholarSheetGolden pins the sheet for the career with Tenure and
// Publications (chart 02).
func TestScholarSheetGolden(t *testing.T) {
	golden(t, render.Sheet(generate(t, chargen.Options{Seed: 23, Career: "Scholar"})), "testdata/scholar_sheet.md")
}

// TestNobleSheetGolden pins the sheet for the career whose rank is its
// Social Standing, with Land Grants (chart 11).
func TestNobleSheetGolden(t *testing.T) {
	golden(t, render.Sheet(generate(t, chargen.Options{Seed: 2978, Career: "Noble"})), "testdata/noble_sheet.md")
}

// TestSoldierSheetGolden pins the sheet for the first Armed Forces
// career: branch and medals (chart 08).
func TestSoldierSheetGolden(t *testing.T) {
	golden(t, render.Sheet(generate(t, chargen.Options{Seed: 305, Career: "Soldier"})), "testdata/soldier_sheet.md")
}

// TestSpacerSheetGolden pins the sheet for the second Armed Forces
// career (chart 07).
func TestSpacerSheetGolden(t *testing.T) {
	golden(t, render.Sheet(generate(t, chargen.Options{Seed: 659, Career: "Spacer"})), "testdata/spacer_sheet.md")
}

// TestMarineSheetGolden pins the sheet for the third Armed Forces career
// (chart 12), which is chart data over the shared mechanics.
func TestMarineSheetGolden(t *testing.T) {
	golden(t, render.Sheet(generate(t, chargen.Options{Seed: 529, Career: "Marine"})), "testdata/marine_sheet.md")
}

// TestAgentSheetGolden pins the sheet for the career that goes
// undercover in other careers (chart 09).
func TestAgentSheetGolden(t *testing.T) {
	golden(t, render.Sheet(generate(t, chargen.Options{Seed: 1717, Career: "Agent"})), "testdata/agent_sheet.md")
}

// TestRogueSheetGolden pins the sheet for the career with schemes,
// prison, and payoffs (chart 10).
func TestRogueSheetGolden(t *testing.T) {
	golden(t, render.Sheet(generate(t, chargen.Options{Seed: 39, Career: "Rogue"})), "testdata/rogue_sheet.md")
}

func TestSheetWithName(t *testing.T) {
	sheet := render.Sheet(generate(t, chargen.Options{Seed: 1, Name: "Eneri Dinsha"}))

	if !strings.Contains(sheet, "**Name**: Eneri Dinsha\n") {
		t.Errorf("sheet missing name line:\n%s", sheet)
	}

	if strings.Contains(sheet, " \n") {
		t.Error("sheet contains trailing whitespace")
	}
}

// TestHistoryMalformedEvents verifies malformed records render marked
// lines instead of panicking: kind without payload, and an out-of-range
// chosen index.
func TestHistoryMalformedEvents(t *testing.T) {
	c := chargen.Character{Events: []chargen.Event{
		{Seq: 1, Kind: chargen.EventStep},
		{Seq: 2, Kind: chargen.EventThrow},
		{Seq: 3, Kind: chargen.EventConsequence},
		{Seq: 4, Kind: chargen.EventChoice, Choice: &chargen.ChoiceEvent{
			Decider: chargen.DeciderPolicy,
			Options: []string{"a"},
			Chosen:  9,
		}},
	}}

	got := render.History(c)

	for _, want := range []string{
		"- #1 (step) [malformed event]\n",
		"- #2 (throw) [malformed event]\n",
		"- #3 (consequence) [malformed event]\n",
		"[chosen 9 out of range]",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("History() missing %q:\n%s", want, got)
		}
	}
}

// TestHistoryConsequenceTexts covers the consequence renderings the seed-1
// golden does not pin: the ERRATA I-1 job_undetermined line, the
// cap-absorbed no_award carrying the skill name, and chart 01's two —
// a Masterpiece and its price, and a New Trade cell with no Trade left to
// give, which must not read as the Major/Minor benefit.
func TestHistoryConsequenceTexts(t *testing.T) {
	c := chargen.Character{Events: []chargen.Event{
		{Seq: 1, Kind: chargen.EventConsequence, Consequence: &chargen.ConsequenceEvent{
			Cause: 1, Kind: chargen.ConsequenceJobUndetermined,
		}},
		{Seq: 2, Kind: chargen.EventConsequence, Consequence: &chargen.ConsequenceEvent{
			Cause: 1, Kind: chargen.ConsequenceNoAward, Skill: "Admin",
		}},
		{Seq: 3, Kind: chargen.EventConsequence, Consequence: &chargen.ConsequenceEvent{
			Cause: 1, Kind: chargen.ConsequenceMasterpiece,
			Career: "Craftsman", Value: 55, Delta: 600_000,
		}},
		{Seq: 4, Kind: chargen.EventConsequence, Consequence: &chargen.ConsequenceEvent{
			Cause: 1, Kind: chargen.ConsequenceBenefitLost, Career: "Craftsman", Skill: "Trade",
		}},
	}}

	got := render.History(c)

	for _, want := range []string{
		"Job undetermined (No Skill); retries next success — ERRATA I-1",
		"no award (Admin at the Skill-15 cap)",
		"Masterpiece created (55 Master Points, worth Cr600000)",
		"benefit lost (every Trade already held)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("History() missing %q:\n%s", want, got)
		}
	}
}

// TestSheetCraftsmanValues pins what a Craftsman career shows on the card:
// the Masterpieces it produced and how many are Perfect (chart 01 p. 75).
// Without it the sheet said only how many terms he served, which is the
// one thing the career is not about.
func TestSheetCraftsmanValues(t *testing.T) {
	for _, tc := range []struct {
		name          string
		made, perfect int
		want          string
	}{
		{"one", 1, 0, "**Career**: Craftsman (1 term), 1 Masterpiece\n"},
		{"several", 4, 2, "**Career**: Craftsman (1 term), 4 Masterpieces (2 Perfect)\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := chargen.Character{Careers: []chargen.CareerRecord{{
				Career: "Craftsman", Began: true,
				Terms:               []chargen.TermRecord{{Term: 1}},
				Masterpieces:        tc.made,
				PerfectMasterpieces: tc.perfect,
			}}}

			if got := render.Sheet(c); !strings.Contains(got, tc.want) {
				t.Errorf("Sheet() missing %q:\n%s", tc.want, got)
			}
		})
	}
}

// TestHistoryThrowLine covers the targeted-throw rendering (target, mods,
// success) that seed-1 characteristics generation does not yet exercise.
func TestHistoryThrowLine(t *testing.T) {
	target := 8
	success := true

	c := chargen.Character{Events: []chargen.Event{
		{Seq: 1, Kind: chargen.EventThrow, Throw: &chargen.ThrowEvent{
			Expr:    "2D",
			Dice:    []int{3, 4},
			Total:   7,
			Target:  &target,
			Success: &success,
			Mods:    []chargen.Mod{{Name: "homeworld", Value: 2}},
			Cite:    "test cite",
		}},
	}}

	want := "# Generation Record\n- #1 2D = 3+4 = 7 vs 8 (homeworld +2): success — test cite\n"
	if got := render.History(c); got != want {
		t.Errorf("History() =\n%q\nwant\n%q", got, want)
	}
}

// TestHistoryChoiceLine covers choice rendering ahead of the first engine
// choice point.
func TestHistoryChoiceLine(t *testing.T) {
	c := chargen.Character{Events: []chargen.Event{
		{Seq: 1, Kind: chargen.EventChoice, Choice: &chargen.ChoiceEvent{
			Decider: chargen.DeciderPolicy,
			Prompt:  "Select career",
			Options: []string{"Citizen", "Scout"},
			Chosen:  0,
			Cite:    "test cite",
		}},
	}}

	want := "# Generation Record\n- #1 policy chose \"Citizen\" of [Citizen, Scout]: Select career — test cite\n"
	if got := render.History(c); got != want {
		t.Errorf("History() =\n%q\nwant\n%q", got, want)
	}
}

// functionaryPath reaches chart 13, which no auto-generated character can:
// "Functionary is never a first career" (p. 87) and the auto policy
// declines every career change (POLICY.md `change_career`).
type functionaryPath struct{}

func (functionaryPath) Choose(c chargen.Choice) int {
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
			if option == "Scholar" {
				return i
			}
		}
	default:
	}

	return chargen.DefaultPolicy{}.Choose(c)
}

func (functionaryPath) Kind() chargen.DeciderKind { return chargen.DeciderPlayer }

// TestFunctionarySheetGolden pins the sheet for the first career a
// character can only reach by changing into one (chart 13, p. 87): two
// career lines, a rolled rank title, and the associated career.
func TestFunctionarySheetGolden(t *testing.T) {
	c, err := chargen.Generate(chargen.Options{Seed: 305, Decider: functionaryPath{}})
	if err != nil {
		t.Fatal(err)
	}

	golden(t, render.Sheet(c), "testdata/functionary_sheet.md")
}

// TestFunctionaryHistoryGolden pins the transcript for the same character:
// the career change, the association, and the promotions.
func TestFunctionaryHistoryGolden(t *testing.T) {
	c, err := chargen.Generate(chargen.Options{Seed: 305, Decider: functionaryPath{}})
	if err != nil {
		t.Fatal(err)
	}

	golden(t, render.History(c), "testdata/functionary_history.md")
}

// craftsmanPath reaches chart 01, which no auto-generated character does:
// entry is "Automatic* — *if TWO skill-6 and Craftsman-1" (p. 75), so the
// craft and the breadth must be acquired in another career first, and the
// auto policy declines every career change (POLICY.md `change_career`).
type craftsmanPath struct {
	offers  int
	arrived bool
}

func (d *craftsmanPath) Choose(c chargen.Choice) int {
	//nolint:exhaustive // Deliberately partitioned: the rest defer to the auto policy.
	switch c.ID {
	case chargen.ChooseCareerChange:
		d.offers++

		if d.arrived || d.offers < 4 {
			return 0
		}

		return 1
	case chargen.ChooseCareer:
		return d.pickCareer(c)
	case chargen.ChooseSkill, chargen.ChooseTrade:
		if i := prefer(c.Options, "Craftsman"); i >= 0 {
			return i
		}
	case chargen.ChooseSkillColumn:
		if i := preferColumn(c.Options, "Academic", "Vocation"); i >= 0 {
			return i
		}
	default:
	}

	return chargen.DefaultPolicy{}.Choose(c)
}

// preferColumn picks the first skills-table column named after one of the
// given words, or -1.
func preferColumn(options []string, words ...string) int {
	for i, option := range options {
		for _, word := range words {
			if strings.Contains(option, word) {
				return i
			}
		}
	}

	return -1
}

// prefer returns the index of an option by name, or -1.
func prefer(options []string, want string) int {
	for i, option := range options {
		if option == want {
			return i
		}
	}

	return -1
}

func (*craftsmanPath) Kind() chargen.DeciderKind { return chargen.DeciderPlayer }

// pickCareer takes Craftsman the moment it is offered, and otherwise
// builds toward it as a Citizen, whose table C offers Trades.
func (d *craftsmanPath) pickCareer(c chargen.Choice) int {
	if i := prefer(c.Options, "Craftsman"); i >= 0 {
		d.arrived = true

		return i
	}

	if i := prefer(c.Options, "Citizen"); i >= 0 {
		return i
	}

	return chargen.DefaultPolicy{}.Choose(c)
}

func craftsmanCharacter(t *testing.T) chargen.Character {
	t.Helper()

	c, err := chargen.Generate(chargen.Options{Seed: 177, Decider: &craftsmanPath{}})
	if err != nil {
		t.Fatal(err)
	}

	return c
}

// TestCraftsmanSheetGolden pins the sheet for the last career, whose
// output is its Masterpieces rather than a rank.
func TestCraftsmanSheetGolden(t *testing.T) {
	golden(t, render.Sheet(craftsmanCharacter(t)), "testdata/craftsman_sheet.md")
}

// TestCraftsmanHistoryGolden pins the transcript. Every career has one so
// that a mechanic without a renderer cannot ship: chart 01's Masterpiece
// event printed as the bare word "masterpiece" until this existed.
func TestCraftsmanHistoryGolden(t *testing.T) {
	golden(t, render.History(craftsmanCharacter(t)), "testdata/craftsman_history.md")
}

// TestEveryConsequenceKindRenders holds the repo's rule that a mechanic is
// not done until its events render in the transcript (docs/PRD.md FR10).
//
// consequenceText ends in a default that returns the kind's raw string, so
// a kind added without a case prints as "masterpiece" or "reserve" rather
// than as anything a reader can use. Only a history transcript would show
// it, and most careers have only a sheet golden — which is how chart 01's
// Masterpiece shipped unrendered.
//
// The check is structural rather than behavioural: a renderer may
// legitimately return the kind's own word, as ConsequenceWaived does.
func TestEveryConsequenceKindRenders(t *testing.T) {
	declared := readIdentifiers(t, "../chargen/event.go", `(Consequence[A-Za-z]+) ConsequenceKind = "`)
	if len(declared) < 20 {
		t.Fatalf("found %d consequence kinds in chargen/event.go; the scan is not working", len(declared))
	}

	handled := readIdentifiers(t, "render.go", `case chargen\.(Consequence[A-Za-z]+)`)

	for _, kind := range declared {
		if !slices.Contains(handled, kind) {
			t.Errorf("%s has no case in the transcript renderer", kind)
		}
	}
}

// readIdentifiers returns every capture of pattern in the named file.
func readIdentifiers(t *testing.T, file, pattern string) []string {
	t.Helper()

	source, err := os.ReadFile(file) //nolint:gosec // G304: fixed test-owned source paths.
	if err != nil {
		t.Fatal(err)
	}

	matches := regexp.MustCompile(pattern).FindAllStringSubmatch(string(source), -1)
	found := make([]string, 0, len(matches))

	for _, match := range matches {
		found = append(found, match[1])
	}

	return found
}

// TestBenefitNamesPluralize walks chart M1's own vocabulary through the
// sheet's pluralizer. Every name on the chart reaches it, because a
// character may hold two of anything.
func TestBenefitNamesPluralize(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct{ one, many string }{
		{"StarPass", "2 StarPasses"},
		{"Proxy", "2 Proxies"},
		{"Ship Shares", "2 Ship Shares"},
		{"Discovery", "2 Discoveries"},
		{"High Passage", "2 High Passages"},
		{"Wafer Jack", "2 Wafer Jacks"},
		{"Knighthood", "2 Knighthoods"},
		{"Land Grant", "2 Land Grants"},
		{"Life Insurance", "2 Life Insurances"},
		{"TAS Life Membership", "2 TAS Life Memberships"},
		{"Pension x2", "2 Pension x2"},
		{"Retirement x2", "2 Retirement x2"},
	} {
		if got := render.Plural(2, tc.one); got != tc.many {
			t.Errorf("two %q render as %q, want %q", tc.one, got, tc.many)
		}

		if got := render.Plural(1, tc.one); got != "1 "+tc.one {
			t.Errorf("one %q renders as %q, want %q", tc.one, got, "1 "+tc.one)
		}
	}
}
