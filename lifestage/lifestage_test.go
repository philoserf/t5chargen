package lifestage_test

import (
	"testing"

	"github.com/philoserf/t5chargen/lifestage"
)

func load(t *testing.T) *lifestage.Table {
	t.Helper()

	table, err := lifestage.Load()
	if err != nil {
		t.Fatal(err)
	}

	return table
}

// TestOfMatchesThePrintedAnchors pins the derivation against every age
// chart A states in prose: adventuring begins at Young Adult "(=18 for
// Humans)", "Physical Aging begins at Peak (= 34 for Humans)", and "Mental
// aging begins at Retirement (= 66 for Humans)".
func TestOfMatchesThePrintedAnchors(t *testing.T) {
	table := load(t)

	for _, tt := range []struct{ age, stage int }{
		{age: 0, stage: 0},
		{age: 1, stage: 0},
		{age: 2, stage: 1},
		{age: 9, stage: 1},
		{age: 10, stage: 2},
		{age: 17, stage: 2},
		{age: 18, stage: 3},
		{age: 25, stage: 3},
		{age: 26, stage: 4},
		{age: 33, stage: 4},
		{age: 34, stage: 5},
		{age: 41, stage: 5},
		{age: 42, stage: 6},
		{age: 66, stage: 9},
		// "the traditional lifespan for humans is 74 years (although
		// certainly some may live longer)": the last stage is the last.
		{age: 74, stage: 9},
		{age: 200, stage: 9},
	} {
		if got := table.Of(tt.age); got != tt.stage {
			t.Errorf("Of(%d) = %d, want %d", tt.age, got, tt.stage)
		}
	}
}

// TestFirstYearOf pins the ages the two kinds of aging begin at, which the
// engine derives rather than reading from the printed year columns.
func TestFirstYearOf(t *testing.T) {
	table := load(t)

	if got := table.FirstYearOf(table.PhysicalStage); got != 34 {
		t.Errorf("physical aging begins at %d, want 34", got)
	}

	if got := table.FirstYearOf(table.MentalStage); got != 66 {
		t.Errorf("mental aging begins at %d, want 66", got)
	}
}

// TestPrintedRangesDivergeExactlyOnce guards a printed inconsistency
// rather than hiding it (interpretation I-48, ERRATA.md).
//
// Chart A states the structure twice and the two statements disagree for
// one row. "After Infancy, each Life Stage is two terms (8 years)" and
// "Humans have a 2-year infancy and nine stages of 8 years each. The
// traditional lifespan for humans is 74 years" both put Retirement at
// 66-73. The Stages of Life table prints 66-71.
//
// The engine derives stages from the arithmetic, so the printed columns
// are transcription only. If a later transcription pass "fixes" the row,
// or introduces a second divergence, this fails and sends the reader to
// the interpretation.
func TestPrintedRangesDivergeExactlyOnce(t *testing.T) {
	table := load(t)

	var diverged []string

	for _, s := range table.Stages {
		first := table.FirstYearOf(s.Number)

		last := first + table.StageYears - 1
		if s.Number == 0 {
			last = table.InfancyYears - 1
		}

		if s.First != first || s.Last != last {
			diverged = append(diverged, s.Name)
		}
	}

	if len(diverged) != 1 || diverged[0] != "Retirement" {
		t.Errorf("printed ranges diverge from the arithmetic at %v, want exactly [Retirement]", diverged)
	}

	// The lifespan the page states is the arithmetic's, not the table's.
	if got := table.FirstYearOf(9) + table.StageYears; got != table.TraditionalLifespan {
		t.Errorf("stages end at %d, but the page states a lifespan of %d", got, table.TraditionalLifespan)
	}
}
