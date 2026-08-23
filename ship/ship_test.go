package ship_test

// Chart S, Book 1 p. 90.

import (
	"testing"

	"github.com/philoserf/t5chargen/ship"
)

func load(t *testing.T) *ship.Table {
	t.Helper()

	table, err := ship.Load()
	if err != nil {
		t.Fatal(err)
	}

	return table
}

// TestEveryPricedShipCostsItsTonnage pins the one thing chart S states in
// prose and then repeats in a column: "one Share acquires 50 tons of the
// ship (thus, a 200-ton Free Trader requires 4 Ship Shares to acquire full
// control)". Tons, shares, and the rate have to agree, so a typo in any of
// the three contradicts the page.
func TestEveryPricedShipCostsItsTonnage(t *testing.T) {
	t.Parallel()

	table := load(t)

	if table.TonsPerShare != 50 {
		t.Fatalf("a share buys %d tons, want 50", table.TonsPerShare)
	}

	priced := 0

	for _, s := range table.Ships {
		if s.Carried {
			continue
		}

		priced++

		if want := s.Tons / table.TonsPerShare; s.Shares != want {
			t.Errorf("%s is %d tons, so %d shares, not %d", s.Name, s.Tons, want, s.Shares)
		}
	}

	// Scout, Lab Ship, Free Trader, Yacht, Close Escort. The Lab Launch
	// and the Escort Gig are carried and priced in nothing.
	if priced != 5 {
		t.Errorf("chart S prices %d ships in shares, want 5", priced)
	}
}

// TestTheChartsWorkedExampleHolds is p. 90's own sentence, checked against
// the transcription rather than against itself.
func TestTheChartsWorkedExampleHolds(t *testing.T) {
	t.Parallel()

	best, ok := load(t).Largest(4)
	if !ok {
		t.Fatal("four shares reach nothing on chart S")
	}

	if best.Name != "Free Trader" || best.Tons != 200 || best.Shares != 4 {
		t.Errorf("four shares reach a %d-ton %s at %d shares; the page says a 200-ton Free Trader at 4",
			best.Tons, best.Name, best.Shares)
	}
}

// TestSharesReachTheBiggestHullTheyCoverAndNoMore walks the rungs of the
// chart: nothing under two shares, the Scout at two, the Free Trader at
// four, and the 400-ton hulls at eight.
func TestSharesReachTheBiggestHullTheyCoverAndNoMore(t *testing.T) {
	t.Parallel()

	table := load(t)

	for _, tc := range []struct {
		shares int
		want   string
		tons   int
	}{
		{0, "", 0},
		{1, "", 0},
		{2, "Scout", 100},
		{3, "Scout", 100},
		{4, "Free Trader", 200},
		{7, "Free Trader", 200},
		{8, "Lab Ship", 400},
		{20, "Lab Ship", 400},
	} {
		best, ok := table.Largest(tc.shares)

		if tc.want == "" {
			if ok {
				t.Errorf("%d shares reach a %s; want nothing", tc.shares, best.Name)
			}

			continue
		}

		if !ok {
			t.Errorf("%d shares reach nothing; want a %s", tc.shares, tc.want)

			continue
		}

		if best.Tons != tc.tons {
			t.Errorf("%d shares reach a %d-ton %s, want %d tons", tc.shares, best.Tons, best.Name, tc.tons)
		}
	}
}

// TestNoShipCarriesACreditValueForAShare is interpretation I-84 as a test:
// chart S prices ships, never shares, so nothing in the table may be read
// as what one share is worth.
func TestNoShipCarriesACreditValueForAShare(t *testing.T) {
	t.Parallel()

	for _, s := range load(t).Ships {
		if s.Carried {
			continue
		}

		// The MCr column is the ship's price. Dividing it by the shares
		// would manufacture a per-share value Book 1 declines to give,
		// so the only assertion here is that the two stay separate
		// units: a ship costs megacredits and takes shares, both.
		if s.CostMCr <= 0 {
			t.Errorf("%s has no MCr cost; chart S prices every ship it sells", s.Name)
		}

		if s.Shares <= 0 {
			t.Errorf("%s has no share price; chart S sells every ship it prices", s.Name)
		}
	}
}
