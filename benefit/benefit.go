// Package benefit holds chart M1 (Book 1 p. 70) — the muster-out
// vocabulary every career's table D draws on — together with the values
// pp. 68-69 give each benefit.
//
// Chart M1 is cross-career data with its own citation, which is why it
// lives here rather than in career: the thirteen table Ds name benefits
// from this list, and the list is the same for all of them.
//
// Per the data/logic boundary (docs/PRD.md, Architecture notes) the tables
// are embedded data; this file is lookup only.
package benefit

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
)

// Kind identifies a benefit in the chart M1 vocabulary. Table D cells name
// one of these rather than free text, so a Knighthood is distinguishable
// from a Wafer Jack and a characteristic award can be applied.
type Kind string

// The chart M1 benefit kinds, split as the chart splits them. The
// Financial column is "Money, StarPass, High Passage, Middle Passage, Low
// Passage, Pension x 2, Retirement x 2"; the Non-Financial column is
// "Characteristic Improvements, Wafer Jack, Forbidden Knowledge,
// Knighthood, Directorship, Proxy, Life Insurance, TAS Fellowship, TAS
// Life Membership, Ship Shares". Land Grants are an Automatic rather than
// a table D cell, and are priced here too.
const (
	Money              Kind = "money"
	StarPassage        Kind = "star_passage"
	HighPassage        Kind = "high_passage"
	MiddlePassage      Kind = "middle_passage"
	LowPassage         Kind = "low_passage"
	PensionDoubling    Kind = "pension_doubling"
	RetirementDoubling Kind = "retirement_doubling"

	Characteristic     Kind = "characteristic"
	WaferJack          Kind = "wafer_jack"
	ForbiddenKnowledge Kind = "forbidden_knowledge"
	Knighthood         Kind = "knighthood"
	Directorship       Kind = "directorship"
	Proxy              Kind = "proxy"
	LifeInsurance      Kind = "life_insurance"
	TASFellowship      Kind = "tas_fellowship"
	TASLife            Kind = "tas_life"
	ShipShares         Kind = "ship_shares"
	LandGrant          Kind = "land_grant"
)

// Class is chart M1's split of the benefit tables.
type Class string

// The two columns chart M1 prints.
const (
	Financial    Class = "financial"
	NonFinancial Class = "non_financial"
)

// Benefit is one entry of the chart M1 vocabulary.
type Benefit struct {
	Kind  Kind   `json:"kind"`
	Name  string `json:"name"`
	Class Class  `json:"class"`

	// Credits is what the benefit is worth outright, where p. 68 prices
	// it: "StarPass ... has a value of Cr250,000".
	Credits int `json:"credits,omitempty"`

	// AnnualCredits is what it pays each year instead: "A Directorship
	// provides an annual payment of Cr36,000".
	AnnualCredits int `json:"annual_credits,omitempty"`

	// CreditsPerTC prices a Land Grant by the world it sits on: "Cr10,000
	// per TC Trade Classification ... A Land Grant on a World with no TCs
	// generates Cr5,000 per year" (p. 68), the latter in AnnualCredits.
	CreditsPerTC int `json:"credits_per_tc,omitempty"`

	// Note is the page's own gloss, kept so a reader need not go back to
	// the book to know what a Wafer Jack is.
	Note string `json:"note,omitempty"`
}

// Entitlement is one row of chart M1's two Entitlements blocks.
type Entitlement struct {
	ID   string `json:"id"`
	Name string `json:"name"`

	// AnnualCredits is a flat yearly payment; CreditsPerYear and
	// CreditsPerTerm scale by Reserve years and Active Duty terms.
	AnnualCredits  int `json:"annual_credits,omitempty"`
	CreditsPerYear int `json:"credits_per_year,omitempty"`
	CreditsPerTerm int `json:"credits_per_term,omitempty"`

	// FromLifeStage is when payment begins: "Entitlements (Annual at Life
	// Stage 9)". Zero means at muster out, which is when the Armed Forces
	// retirements begin.
	FromLifeStage int `json:"from_life_stage,omitempty"`

	// MinimumTerms is the Armed Forces bar: "(based on minimum 4 terms
	// Army Navy Marines)".
	MinimumTerms int `json:"minimum_terms,omitempty"`
}

// Automatic is one row of "Automatics (Subject to Eligibility)".
type Automatic struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Eligibility string `json:"eligibility"`
}

// ForbiddenSkill is one row of chart M1's Forbidden Knowledge table.
type ForbiddenSkill struct {
	Roll  int    `json:"roll"`
	Skill string `json:"skill"`
	Note  string `json:"note"`
}

// Table is chart M1.
type Table struct {
	Cite         string        `json:"cite"`
	Kinds        []Benefit     `json:"kinds"`
	Entitlements []Entitlement `json:"entitlements"`
	Automatics   []Automatic   `json:"automatics"`

	// CashOutYears: "Any Entitlement can be cashed out for a lump sum"
	// of this many years' payments (p. 69).
	CashOutYears int `json:"cash_out_years"`

	// ForbiddenKnowledge is the 1D table naming the skill received.
	ForbiddenKnowledge []ForbiddenSkill `json:"forbidden_knowledge"`
}

//go:embed data/benefits.json
var benefitsJSON []byte

// ErrUnknownKind reports a benefit name outside the chart M1 vocabulary.
var ErrUnknownKind = errors.New("benefit: unknown kind")

var errBadTable = errors.New("invalid benefits table")

// Load returns the embedded chart M1.
func Load() (*Table, error) { return table() }

var table = sync.OnceValues(func() (*Table, error) {
	var t Table
	if err := json.Unmarshal(benefitsJSON, &t); err != nil {
		return nil, fmt.Errorf("benefits table: %w", err)
	}

	if err := t.validate(); err != nil {
		return nil, err
	}

	return &t, nil
})

// For returns the vocabulary entry for a kind.
func (t *Table) For(kind Kind) (Benefit, error) {
	for _, b := range t.Kinds {
		if b.Kind == kind {
			return b, nil
		}
	}

	return Benefit{}, fmt.Errorf("%w: %q", ErrUnknownKind, kind)
}

// Entitlement returns a chart M1 entitlement by id.
func (t *Table) Entitlement(id string) (Entitlement, bool) {
	for _, e := range t.Entitlements {
		if e.ID == id {
			return e, true
		}
	}

	return Entitlement{}, false
}

// ForbiddenSkillAt returns the skill a 1D roll grants (chart M1).
func (t *Table) ForbiddenSkillAt(roll int) (ForbiddenSkill, bool) {
	for _, f := range t.ForbiddenKnowledge {
		if f.Roll == roll {
			return f, true
		}
	}

	return ForbiddenSkill{}, false
}

// validate rejects embedded data the lookups assume is well-formed.
func (t *Table) validate() error {
	if err := t.validateKinds(); err != nil {
		return err
	}

	if t.CashOutYears < 1 {
		return fmt.Errorf("%w: non-positive cash-out years", errBadTable)
	}

	if len(t.Automatics) == 0 {
		return fmt.Errorf("%w: no automatics", errBadTable)
	}

	return t.validateEntitlements()
}

// validateEntitlements checks chart M1's two Entitlements blocks and its
// Forbidden Knowledge table.
func (t *Table) validateEntitlements() error {
	for _, e := range t.Entitlements {
		if e.ID == "" || e.Name == "" {
			return fmt.Errorf("%w: entitlement %q is unnamed", errBadTable, e.ID)
		}

		if e.AnnualCredits == 0 && e.CreditsPerYear == 0 && e.CreditsPerTerm == 0 {
			return fmt.Errorf("%w: entitlement %q pays nothing", errBadTable, e.ID)
		}
	}

	// The Forbidden Knowledge table is rolled with 1D, so it needs a row
	// for every face.
	for roll := 1; roll <= 6; roll++ {
		if _, ok := t.ForbiddenSkillAt(roll); !ok {
			return fmt.Errorf("%w: no Forbidden Knowledge for a roll of %d", errBadTable, roll)
		}
	}

	return nil
}

// validateKinds checks the vocabulary is complete and unambiguous.
func (t *Table) validateKinds() error {
	seen := make(map[Kind]bool, len(t.Kinds))

	for _, b := range t.Kinds {
		if b.Kind == "" || b.Name == "" {
			return fmt.Errorf("%w: a benefit is unnamed", errBadTable)
		}

		if seen[b.Kind] {
			return fmt.Errorf("%w: %q listed twice", errBadTable, b.Kind)
		}

		seen[b.Kind] = true

		if b.Class != Financial && b.Class != NonFinancial {
			return fmt.Errorf("%w: %q is neither financial nor non-financial", errBadTable, b.Kind)
		}
	}

	// Every kind the engine names must be in the data.
	for _, kind := range []Kind{
		Money, StarPassage, HighPassage, MiddlePassage, LowPassage,
		PensionDoubling, RetirementDoubling, Characteristic, WaferJack,
		ForbiddenKnowledge, Knighthood, Directorship, Proxy, LifeInsurance,
		TASFellowship, TASLife, ShipShares, LandGrant,
	} {
		if !seen[kind] {
			return fmt.Errorf("%w: %q is declared but absent from the table", errBadTable, kind)
		}
	}

	return nil
}
