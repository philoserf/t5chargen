package chargen_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/chargen"
)

// trackDecider begins the Merchant career on a named berth and otherwise
// answers first-listed, exercising the rating tracks the default policy
// never reaches (it always takes the first-listed 4th Officer berth).
type trackDecider struct{ track string }

func (d trackDecider) Choose(c chargen.Choice) (int, error) {
	if c.ID == chargen.ChooseBeginTrack {
		if i := slices.Index(c.Options, d.track); i >= 0 {
			return i, nil
		}
	}

	// Everything else defers to the auto policy rather than answering 0.
	// A bare 0 is an answer about list position, so this fake used to
	// change what it tested whenever a chart's option list grew.
	return autoPolicy(c)
}

func (trackDecider) Kind() chargen.DeciderKind { return chargen.DeciderPlayer }

// merchantRun generates a Merchant and returns the character and its
// career record.
func merchantRun(t *testing.T, seed uint64, decider chargen.Decider) (chargen.Character, chargen.CareerRecord) {
	t.Helper()

	c, err := chargen.Generate(chargen.Options{Seed: seed, Career: "Merchant", Decider: decider})
	if err != nil {
		t.Fatalf("seed %d: %v", seed, err)
	}

	if len(c.Careers) == 0 {
		t.Fatalf("seed %d: no career record", seed)
	}

	return c, c.Careers[len(c.Careers)-1]
}

// firstRankSet returns the title of the first rank the character entered.
func firstRankSet(c chargen.Character) string {
	for _, e := range c.Events {
		if e.Kind == chargen.EventConsequence && e.Consequence.Kind == chargen.ConsequenceRankSet {
			return e.Consequence.Skill
		}
	}

	return ""
}

// TestMerchantBeginTracks verifies each chart 06 entry path lands on its
// own starting rank: "To Begin 4th Officer Int / To Begin Spacehand Dex /
// To Begin Temp Auto" (p. 80). Seed 0 begins on all three.
func TestMerchantBeginTracks(t *testing.T) {
	tests := []struct {
		track string
		rank  string
		title string
	}{
		{track: "4th Officer", rank: "M1", title: "Fourth Officer"},
		{track: "Spacehand", rank: "R0", title: "Spacehand"},
		{track: "Temp", rank: "RX", title: "Temp"},
	}

	for _, tt := range tests {
		t.Run(tt.track, func(t *testing.T) {
			c, record := merchantRun(t, 0, trackDecider{track: tt.track})
			if !record.Began {
				t.Fatalf("the %s track did not begin at seed 0", tt.track)
			}

			if got := firstRankSet(c); got != tt.title {
				t.Errorf("entered as %q, want %q", got, tt.title)
			}
		})
	}
}

// TestMerchantBeginFailure verifies a failed entry check costs a year and
// leaves the career unbegun (interpretation I-15): the automatic Temp
// berth is not a fallback. Seed 2 fails both checked tracks.
func TestMerchantBeginFailure(t *testing.T) {
	for _, track := range []string{"4th Officer", "Spacehand"} {
		t.Run(track, func(t *testing.T) {
			c, record := merchantRun(t, 2, trackDecider{track: track})
			if record.Began {
				t.Fatalf("the %s track began at seed 2; the fixture expects a failure", track)
			}

			if record.Rank != "" {
				t.Errorf("unbegun career holds rank %q", record.Rank)
			}

			if !slices.ContainsFunc(c.Events, func(e chargen.Event) bool {
				return e.Kind == chargen.EventConsequence &&
					e.Consequence.Kind == chargen.ConsequenceCareerNotBegun
			}) {
				t.Error("no career_not_begun consequence recorded")
			}
		})
	}
}

// TestMerchantTempIsAutomatic verifies "To Begin Temp Auto" (chart 06):
// the untrained berth needs no throw, so it begins even on a seed whose
// checked tracks both fail.
func TestMerchantTempIsAutomatic(t *testing.T) {
	c, record := merchantRun(t, 2, trackDecider{track: "Temp"})
	if !record.Began {
		t.Fatal("the automatic Temp berth did not begin")
	}

	if got := firstRankSet(c); got != "Temp" {
		t.Errorf("entered as %q, want Temp", got)
	}
}

// TestMerchantEventIntegrity verifies every Merchant consequence references
// an earlier throw or choice event (docs/PRD.md FR10) on every entry track —
// the automatic Temp berth makes no throw, so its rank_set must name the
// selecting choice as its cause.
func TestMerchantEventIntegrity(t *testing.T) {
	for _, track := range []string{"4th Officer", "Spacehand", "Temp"} {
		t.Run(track, func(t *testing.T) {
			for seed := uint64(1); seed <= 12; seed++ {
				c, _ := merchantRun(t, seed, trackDecider{track: track})

				kinds := map[int]chargen.EventKind{}
				for _, event := range c.Events {
					kinds[event.Seq] = event.Kind
				}

				for _, event := range c.Events {
					if event.Kind != chargen.EventConsequence {
						continue
					}

					cause := event.Consequence.Cause
					if cause <= 0 || cause >= event.Seq {
						t.Fatalf("seed %d: event %d (%s): cause %d is not an earlier event",
							seed, event.Seq, event.Consequence.Kind, cause)
					}

					if k := kinds[cause]; k != chargen.EventThrow && k != chargen.EventChoice {
						t.Fatalf("seed %d: event %d (%s): cause %d is a %q, want throw or choice",
							seed, event.Seq, event.Consequence.Kind, cause, k)
					}
				}
			}
		})
	}
}

// TestMerchantCommission verifies the row a rating may attempt — "Temp may
// attempt Officer Commission ... within the same Term" (chart 06) — and
// that a commission lands on the officer ladder at M1, the rank the chart
// names explicitly, rather than the next rating rank.
func TestMerchantCommission(t *testing.T) {
	c, record := merchantRun(t, 2, trackDecider{track: "Temp"})

	if !strings.HasPrefix(record.Rank, "M") {
		t.Fatalf("seed 2 ended at %q; the fixture expects a commission to the officer ladder", record.Rank)
	}

	titles := rankTitles(c)
	if !slices.Contains(titles, "Fourth Officer") {
		t.Errorf("rank history %v never passes through Fourth Officer", titles)
	}

	// The commission is the only route from a rating to the officer
	// ladder, so Fourth Officer must follow a rating rank.
	if i := slices.Index(titles, "Fourth Officer"); i == 0 {
		t.Error("entered directly as Fourth Officer; the Temp track enters as Temp")
	}
}

// TestMerchantRatingPromotion verifies the Rating Promotion row advances
// within the rating ladder (RX Temp to R0 Spacehand).
func TestMerchantRatingPromotion(t *testing.T) {
	c, record := merchantRun(t, 14, trackDecider{track: "Temp"})

	if record.Rank != "R0" {
		t.Fatalf("seed 14 ended at %q; the fixture expects a Rating Promotion to R0", record.Rank)
	}

	if got := rankTitles(c); !slices.Equal(got, []string{"Temp", "Spacehand"}) {
		t.Errorf("rank history = %v, want [Temp Spacehand]", got)
	}
}

// rankTitles returns the rank titles the character entered, in order.
func rankTitles(c chargen.Character) []string {
	var titles []string

	for _, e := range c.Events {
		if e.Kind == chargen.EventConsequence && e.Consequence.Kind == chargen.ConsequenceRankSet {
			titles = append(titles, e.Consequence.Skill)
		}
	}

	return titles
}

// TestMerchantRanksAreCharted verifies every rank a character can hold is
// a row of the chart table: the top of each ladder bars further attempts
// rather than running off the end (interpretation I-13).
func TestMerchantRanksAreCharted(t *testing.T) {
	def, err := career.Merchant()
	if err != nil {
		t.Fatal(err)
	}

	for seed := uint64(1); seed <= 12; seed++ {
		for _, track := range []string{"4th Officer", "Temp"} {
			c, record := merchantRun(t, seed, trackDecider{track: track})
			if record.Rank == "" {
				continue
			}

			if _, ok := def.RankByID(record.Rank); !ok {
				t.Fatalf("seed %d (%s): rank %q is not in the chart table", seed, track, record.Rank)
			}

			for _, title := range rankTitles(c) {
				if !slices.ContainsFunc(def.Ranks, func(r career.Rank) bool { return r.Title == title }) {
					t.Fatalf("seed %d (%s): rank title %q is not in the chart table", seed, track, title)
				}
			}
		}
	}
}

// TestMerchantAdvancementEarnsSkills verifies chart 06 table B: each rank
// gained in a term earns one extra table C roll on top of the per-term
// four. The skill-column choices count the rolls taken.
func TestMerchantAdvancementEarnsSkills(t *testing.T) {
	def, err := career.Merchant()
	if err != nil {
		t.Fatal(err)
	}

	if def.SkillsPerTerm != 4 || def.SkillsPerAdvancement != 1 {
		t.Fatalf("chart 06 table B = %d per term, %d per promotion; want 4 and 1",
			def.SkillsPerTerm, def.SkillsPerAdvancement)
	}

	c, record := merchantRun(t, 17, chargen.DefaultPolicy{})

	columns := 0

	for _, e := range c.Events {
		if e.Kind == chargen.EventChoice && e.Choice.Prompt == "Select a Merchant Skills column" {
			columns++
		}
	}

	// One rank_set is the entry rank, which earns no extra roll. A term
	// cut short by death takes no skills, and seed 17 survives.
	promotions := len(rankTitles(c)) - 1

	want := len(record.Terms)*def.SkillsPerTerm + promotions*def.SkillsPerAdvancement
	if columns != want {
		t.Errorf("skill-column rolls = %d, want %d (%d terms x 4 + %d promotions)",
			columns, want, len(record.Terms), promotions)
	}
}

// TestMerchantShipShareEscalation verifies "Ship Share Rewards are awarded
// equal to the receipt number" (chart 06): the nth award is n shares, and
// the record holds the running sum.
func TestMerchantShipShareEscalation(t *testing.T) {
	c, record := merchantRun(t, 17, chargen.DefaultPolicy{})

	receipt, total := 0, 0

	for _, e := range c.Events {
		if e.Kind != chargen.EventConsequence || e.Consequence.Kind != chargen.ConsequenceShipShares {
			continue
		}

		receipt++
		// The chart's last printed value caps later receipts (I-14).
		want := min(receipt, 6)
		total += want

		if e.Consequence.Delta != want {
			t.Errorf("receipt %d awarded %d shares, want %d", receipt, e.Consequence.Delta, want)
		}

		if e.Consequence.Value != total {
			t.Errorf("receipt %d total = %d, want %d", receipt, e.Consequence.Value, total)
		}
	}

	if receipt == 0 {
		t.Fatal("seed 17 records no Ship Share award")
	}

	if record.ShipShares != total {
		t.Errorf("record ShipShares = %d, want %d", record.ShipShares, total)
	}
}

// TestMerchantOfficerPromotionTarget verifies "Terms x2" counts completed
// terms (interpretation I-12): the first term's target is zero, plus the
// chart's "+3 if Int 8+" where it applies.
func TestMerchantOfficerPromotionTarget(t *testing.T) {
	c, _ := merchantRun(t, 17, chargen.DefaultPolicy{})

	term := 0
	seen := false

	for _, e := range c.Events {
		switch {
		case e.Kind == chargen.EventStep && strings.Contains(e.Step.Name, "Term "):
			term++
		case e.Kind == chargen.EventThrow && strings.Contains(e.Throw.Cite, "Officer Promotion"):
			seen = true

			want := (term - 1) * 2
			for _, mod := range e.Throw.Mods {
				want += mod.Value
			}

			if e.Throw.Target == nil || *e.Throw.Target != want {
				t.Errorf("term %d promotion target = %v, want %d (%d completed terms x2 plus mods)",
					term, e.Throw.Target, want, term-1)
			}
		}
	}

	if !seen {
		t.Fatal("seed 17 records no Officer Promotion throw")
	}
}
