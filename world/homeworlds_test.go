package world_test

import (
	"testing"

	"github.com/philoserf/t5chargen/world"
)

// TestChartBIsWholeAndAsPrinted pins the transcription of chart B's world
// list (p. 56): thirty-six cells, the three that repeat, and the one that
// names no world.
func TestChartBIsWholeAndAsPrinted(t *testing.T) {
	worlds, err := world.ChartB()
	if err != nil {
		t.Fatal(err)
	}

	if len(worlds) != 36 {
		t.Fatalf("%d cells, want 36", len(worlds))
	}

	// Regina fills three cells, so it is three times as likely as any
	// other world. That is the chart, not a slip in transcribing it, and
	// a test that did not say so would look like one.
	regina := 0

	for _, w := range worlds {
		if w.Name == "Regina" {
			regina++
		}
	}

	if regina != 3 {
		t.Errorf("Regina fills %d cells, want the 3 the chart prints", regina)
	}
}

// TestChartBCellsResolve verifies the two dice index the chart in the order
// it prints them, D1 then D2 — a 3 and a 4 is not a 4 and a 3.
func TestChartBCellsResolve(t *testing.T) {
	for _, tc := range []struct {
		d1, d2    int
		name, uwp string
	}{
		{d1: 1, d2: 1, name: "Alell", uwp: "B56789C-A"},
		{d1: 2, d2: 4, name: "Earth", uwp: "A867A69-F"},
		{d1: 3, d2: 4, name: "Regina", uwp: "A788899-C"},
		{d1: 4, d2: 3, name: "Uakye", uwp: "B439598-D"},
		{d1: 6, d2: 5, name: "Pax Rulin", uwp: "A402231-E"},
	} {
		got, err := world.At(tc.d1, tc.d2)
		if err != nil {
			t.Fatal(err)
		}

		if got.Name != tc.name || got.UWP != tc.uwp {
			t.Errorf("cell %d %d is %q %q, want %q %q", tc.d1, tc.d2, got.Name, got.UWP, tc.name, tc.uwp)
		}
	}
}

// TestDeepSpaceHasNoUWP verifies chart B's last cell (interpretation
// I-97). "Born In Deep Space" names no world, so it carries no UWP — and
// it must still validate, because a homeworld the book prints without one
// is not the partial UWP docs/PRD.md FR2 refuses.
func TestDeepSpaceHasNoUWP(t *testing.T) {
	deepSpace, err := world.At(6, 6)
	if err != nil {
		t.Fatal(err)
	}

	if deepSpace.UWP != "" {
		t.Errorf("the deep space cell carries UWP %q, want none", deepSpace.UWP)
	}

	if err := deepSpace.Validate(); err != nil {
		t.Errorf("the deep space cell does not validate: %v", err)
	}

	if got := deepSpace.Label(); got != "Space (Ds)" {
		t.Errorf("label %q, want %q — a missing UWP must not leave a gap in it", got, "Space (Ds)")
	}

	// A supplied homeworld is still held to FR2. Trade classifications
	// with no UWP are a partial world, not deep space — the first reading
	// of I-97 used exactly that shape as the marker and would have made
	// FR2 conditional on nobody building one.
	for _, partial := range []world.Homeworld{
		{},
		{TradeClassifications: []string{"In"}},
		{Name: "Nowhere", TradeClassifications: []string{"Ds"}},
	} {
		if err := partial.Validate(); err == nil {
			t.Errorf("%+v validated; only a deep space birth may omit its UWP", partial)
		}
	}

	// And the mark is a claim about the world, not a licence: a marked
	// homeworld carrying a UWP is an error too.
	marked := world.Homeworld{Name: "Space", UWP: "A788899-C", DeepSpace: true, TradeClassifications: []string{"Ds"}}
	if err := marked.Validate(); err == nil {
		t.Error("a deep space birth with a UWP validated")
	}
}
