// Package ship is chart S, "Ship Shares" (Book 1 p. 90): the ships a
// character's accumulated Ship Shares reach.
//
// The chart prices ships in shares and never prices a share. That is the
// whole of what Book 1 says a share is worth, and it is deliberate —
// p. 68 gives the benefits it means to price a credit figure each, and
// gives a share none on any of the four pages that discuss one
// (interpretation I-84, ERRATA.md).
package ship

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
)

// Ship is one row of chart S. The two rows with no shares are small craft
// carried by a larger ship — the Lab Launch by the Lab Ship, the Escort Gig
// by the Close Escort — and are not acquired with shares of their own.
type Ship struct {
	Name string `json:"name"`
	Tons int    `json:"tons"`

	// Jump and Maneuver are the chart's J and G columns.
	Jump     int `json:"jump"`
	Maneuver int `json:"maneuver"`

	// QSP is the chart's Quick Ship Profile.
	QSP string `json:"qsp,omitempty"`

	// Shares buys the ship outright: "one Share acquires 50 tons of the
	// ship (thus, a 200-ton Free Trader requires 4 Ship Shares to acquire
	// full control)" (p. 90).
	Shares int `json:"shares,omitempty"`

	// LoanShares is the chart's Loan? column, for the ships that "can be
	// acquired on loan, subject to basic eligibility requirements. The
	// Ship Shares are expended when the loan is made" (p. 90).
	LoanShares int `json:"loan_shares,omitempty"`

	// CostMCr is the chart's MCr column, the price of the ship itself.
	// It is not a price for a share: the two are different currencies,
	// and p. 90 converts between them nowhere.
	CostMCr float64 `json:"cost_mcr,omitempty"`

	// Carried marks the small craft that come with a larger ship rather
	// than being bought with shares.
	Carried bool `json:"carried,omitempty"`
}

// Table is chart S.
type Table struct {
	Cite string `json:"cite"`

	// TonsPerShare is the conversion the chart states in prose, and the
	// invariant every priced row satisfies.
	TonsPerShare     int    `json:"tons_per_share"`
	TonsPerShareCite string `json:"tons_per_share_cite"`

	Ships []Ship `json:"ships"`
}

//go:embed data/ships.json
var shipsJSON []byte

var errBadTable = errors.New("invalid ships table")

// Load returns the embedded chart S.
func Load() (*Table, error) { return table() }

var table = sync.OnceValues(func() (*Table, error) {
	var t Table
	if err := json.Unmarshal(shipsJSON, &t); err != nil {
		return nil, fmt.Errorf("ships table: %w", err)
	}

	if err := t.validate(); err != nil {
		return nil, err
	}

	return &t, nil
})

// Largest returns the biggest ship the given shares buy outright, and
// whether any does. Ties go to the first row on the chart.
func (t *Table) Largest(shares int) (Ship, bool) {
	var (
		best  Ship
		found bool
	)

	for _, s := range t.Ships {
		// Skipped on the flag that means "not for sale" rather than on
		// the price. Carried marks the Lab Launch and the Escort Gig;
		// Shares == 0 worked only because no priced ship is cheap enough
		// to round to zero — which is exactly what the floor above used
		// to do to a sub-50-ton craft, at which point validation demanded
		// shares: 0 and this treated it as carried.
		if s.Carried || s.Shares > shares {
			continue
		}

		if !found || s.Tons > best.Tons {
			best, found = s, true
		}
	}

	return best, found
}

// validate rejects a transcription the lookups assume is well-formed. The
// tonnage check is the one that earns its keep: p. 90 states the share
// conversion in prose, so a typo in any of the tons, shares, or the rate
// contradicts the page and fails the load rather than mispricing a ship.
func (t *Table) validate() error {
	if t.TonsPerShare < 1 {
		return fmt.Errorf("%w: non-positive tons per share", errBadTable)
	}

	if len(t.Ships) == 0 {
		return fmt.Errorf("%w: no ships", errBadTable)
	}

	seen := make(map[string]bool, len(t.Ships))

	for _, s := range t.Ships {
		if err := t.validateShip(s, seen); err != nil {
			return err
		}

		seen[s.Name] = true
	}

	return nil
}

// validateShip checks one row of chart S.
func (t *Table) validateShip(s Ship, seen map[string]bool) error {
	if s.Name == "" {
		return fmt.Errorf("%w: an unnamed ship", errBadTable)
	}

	if seen[s.Name] {
		return fmt.Errorf("%w: %q listed twice", errBadTable, s.Name)
	}

	if s.Tons < 1 {
		return fmt.Errorf("%w: %q displaces nothing", errBadTable, s.Name)
	}

	// A carried small craft has no share price; everything else has one,
	// and it must agree with the rate the page states in prose.
	if s.Carried {
		if s.Shares != 0 || s.LoanShares != 0 {
			return fmt.Errorf("%w: carried craft %q is priced in shares", errBadTable, s.Name)
		}

		return nil
	}

	// Rounded up: "one Share acquires 50 tons of the ship (thus, a
	// 200-ton Free Trader requires 4 Ship Shares to acquire full
	// control)" — and full control of a ship whose tonnage is not a
	// multiple of the rate needs the ceiling, not the floor. Integer
	// division truncates, so a 99-ton ship was held to 1 share where the
	// rule asks 2: the check would have accepted an under-priced row and
	// rejected a correctly-priced one, reporting good data as the fault.
	// Every row today is a multiple of 50, so floor and ceiling agree and
	// nothing moves.
	if want := (s.Tons + t.TonsPerShare - 1) / t.TonsPerShare; s.Shares != want {
		return fmt.Errorf("%w: %q is %d tons at %d tons a share, so %d shares, not %d",
			errBadTable, s.Name, s.Tons, t.TonsPerShare, want, s.Shares)
	}

	if s.LoanShares != 0 && s.LoanShares != s.Shares {
		return fmt.Errorf("%w: %q loans for %d shares but sells for %d",
			errBadTable, s.Name, s.LoanShares, s.Shares)
	}

	return nil
}
