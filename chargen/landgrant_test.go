package chargen_test

// Land Grant income: "An unimproved Land Grant generates income based on
// the Trade Classifications of the world and equal to Cr10,000 per TC
// annually (equal to Cr5,000 if there are no TCs)" (Book 1 p. 88).

import (
	"testing"

	"github.com/philoserf/t5chargen/benefit"
	"github.com/philoserf/t5chargen/chargen"
)

// The default homeworld is Regina A788899-C (Ph Pa Ri), three Trade
// Classifications, so a homeworld hex earns Cr30,000 (chart B p. 56).
const (
	homeworldHex = 30_000
	noTCHex      = 5_000
)

// TestANobleGrantEarnsItsHomeworldHexAndOneCompanion prices the Noble's
// grant the way p. 88's own example prices Sir Richard's: the homeworld hex
// by its Trade Classifications, and the companion hex p. 41 awards
// alongside it at the no-TC floor.
func TestANobleGrantEarnsItsHomeworldHexAndOneCompanion(t *testing.T) {
	t.Parallel()

	checked := 0

	for seed := uint64(1); seed <= 200; seed++ {
		c, open := generateIfOpen(t, chargen.Options{Seed: seed, Career: "Noble"})
		if !open {
			continue // below chart 11's Soc B+ prerequisite (I-28)
		}

		if c.Dead {
			continue // a dead character does not muster out (I-77)
		}

		grants := 0
		for _, rec := range c.Careers {
			grants += rec.LandGrants
		}

		if grants == 0 {
			continue
		}

		checked++

		if want := grants * (homeworldHex + noTCHex); c.LandGrantIncome != want {
			t.Fatalf("seed %d: %d Noble grants earn Cr%d a year, want Cr%d",
				seed, grants, c.LandGrantIncome, want)
		}
	}

	if checked == 0 {
		t.Fatal("no Noble in 200 seeds held a Land Grant; the test proves nothing")
	}
}

// TestAScoutGrantEarnsOneHexAndNotTheHomeworld holds the two careers apart.
// p. 79 sites a Discoverer's grant on "a non-Mainworld within the Imperium"
// — one hex, and not the one p. 88 puts on a Noble's homeworld — so a Scout
// earns the floor per grant however rich his homeworld is.
func TestAScoutGrantEarnsOneHexAndNotTheHomeworld(t *testing.T) {
	t.Parallel()

	checked := 0

	for seed := uint64(1); seed <= 200; seed++ {
		c := generate(t, chargen.Options{Seed: seed, Career: "Scout"})
		if c.Dead {
			continue // a dead character does not muster out (I-77)
		}

		grants, discoveries := 0, 0

		for _, rec := range c.Careers {
			grants += rec.LandGrants
			discoveries += rec.Discoveries
		}

		if grants == 0 {
			continue
		}

		checked++

		// One grant per Discovery, counted once. Reading the Discovery
		// count as a second source of grants doubles every Scout.
		if grants != discoveries {
			t.Fatalf("seed %d: %d Discoveries left %d Land Grants, want one each",
				seed, discoveries, grants)
		}

		if want := grants * noTCHex; c.LandGrantIncome != want {
			t.Fatalf("seed %d: %d Scout grants earn Cr%d a year, want Cr%d",
				seed, grants, c.LandGrantIncome, want)
		}
	}

	if checked == 0 {
		t.Fatal("no Scout in 200 seeds made a Discovery; the test proves nothing")
	}
}

// TestALandGrantIsIncomeAndNotMoney pins p. 68's "Any character who has
// received a Land Grant retains it at Mustering Out": there is no clause
// selling one, so the income is never added to the character's credits.
//
// It checks the money instead of the income, because that is where banking
// the grant would show: for a character with no entitlement to cash out and
// no Rogue payoff, every credit he holds came from a Money cell.
func TestALandGrantIsIncomeAndNotMoney(t *testing.T) {
	t.Parallel()

	checked := 0

	for seed := uint64(1); seed <= 200; seed++ {
		c := generate(t, chargen.Options{Seed: seed, Career: "Scout"})
		if c.Dead || c.LandGrantIncome == 0 || len(c.Entitlements) > 0 {
			continue
		}

		fromCells := 0
		for _, b := range c.Benefits {
			fromCells += b.Credits
		}

		for _, rec := range c.Careers {
			fromCells += rec.SchemePayoff
		}

		checked++

		if c.Credits != fromCells {
			t.Fatalf("seed %d: holds Cr%d but the Money cells paid Cr%d, with Cr%d a year in grants unaccounted for",
				seed, c.Credits, fromCells, c.LandGrantIncome)
		}
	}

	if checked == 0 {
		t.Fatal("no Scout in 200 seeds held a grant without an entitlement; the test proves nothing")
	}
}

// TestShipSharesPoolBothLedgers checks the two places a share comes from
// add up. A Merchant's Rewards are "redeemable toward ownership of a ship
// upon mustering out" (p. 80) and a Benefits-column share is the same
// fractional ownership (p. 69), so the character's total is both.
func TestShipSharesPoolBothLedgers(t *testing.T) {
	t.Parallel()

	checked := 0

	for seed := uint64(1); seed <= 300; seed++ {
		c := generate(t, chargen.Options{Seed: seed, Career: "Merchant", Decider: benefitsColumn{}})
		if c.Dead {
			continue
		}

		fromCareers, fromBenefits := 0, 0

		for _, rec := range c.Careers {
			fromCareers += rec.ShipShares
		}

		for _, b := range c.Benefits {
			if b.Kind == benefit.ShipShares {
				fromBenefits += b.Count
			}
		}

		if fromCareers == 0 || fromBenefits == 0 {
			continue // only a character drawing on both proves anything
		}

		checked++

		if want := fromCareers + fromBenefits; c.ShipShares != want {
			t.Fatalf("seed %d: %d career shares and %d benefit shares total %d, want %d",
				seed, fromCareers, fromBenefits, c.ShipShares, want)
		}
	}

	if checked == 0 {
		t.Fatal("no Merchant in 300 seeds drew shares from both ledgers; the test proves nothing")
	}
}
