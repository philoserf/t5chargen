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

	// Fame is awarded by four career tables ("Fame +1", "Fame +2") though
	// chart M1 lists it only among the Automatics (interpretation I-72).
	Fame Kind = "fame"
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

	// AnnualCredits is what it pays each year instead, unconditionally:
	// "A Directorship provides an annual payment of Cr36,000".
	AnnualCredits int `json:"annual_credits,omitempty"`

	// CreditsPerTC prices a Land Grant by the world it sits on: "Cr10,000
	// per TC Trade Classification" (p. 68). It is the whole income, not a
	// bonus on top of anything: a world with three TCs pays the page's
	// Cr30,000, not Cr30,000 plus a base.
	CreditsPerTC int `json:"credits_per_tc,omitempty"`

	// NoTCAnnualCredits is the floor for the TC-priced case alone: "A
	// Land Grant on a World with no TCs generates Cr5,000 per year"
	// (p. 68). It applies instead of CreditsPerTC, never alongside it.
	NoTCAnnualCredits int `json:"no_tc_annual_credits,omitempty"`

	// SocFloor, AboveFloorBonus, OfficersOnlyCareers and
	// NonOfficerBonus carry the Knighthood's arithmetic, which is the one
	// benefit whose award depends on the character and the career: "A
	// Knighthood raises any value of Soc to B; if the character is
	// already Soc 11+, he receives Soc +1 instead" and "In the Spacer,
	// Soldier, and Marine careers, Knighthood is only available to
	// Officers. A non-officer receives Soc +1" (p. 68).
	//
	// They are fields rather than prose because a note cannot be tested
	// and this one was wrong twice before it was: it had a Soc-11+
	// character keeping his Social Standing, and omitted the non-officer
	// substitution entirely.
	SocFloor            int      `json:"soc_floor,omitempty"`
	AboveFloorBonus     int      `json:"above_floor_bonus,omitempty"`
	OfficersOnlyCareers []string `json:"officers_only_careers,omitempty"`
	NonOfficerBonus     int      `json:"non_officer_bonus,omitempty"`

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

	// Note is the page's own gloss on how the entitlement interacts with
	// the others: "a Functionary receives Cr15,000 per year (which
	// replaces a Citizen's pension, if any)" (p. 69).
	Note string `json:"note,omitempty"`
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

// LandGrantHexes says where one career's Land Grant hexes sit, which is
// the whole of what decides their income: a hex is priced by the Trade
// Classifications of the world it is on (p. 88), so the income question is
// which worlds the grant covers.
//
// The two granting careers differ, and the difference is printed. A Noble's
// grant starts at home — "The first hex in any grant is on the Noble's
// homeworld" (p. 88) — and p. 41 adds one more: "For each Terrain Hex
// granted on the Mainworld in a system, the Holder is awarded a Terrain Hex
// on another world in the system." A Scout's does not: "Each Discovery
// confers a Land Grant of one World Hex on a non-Mainworld within the
// Imperium" (p. 79).
type LandGrantHexes struct {
	Career string `json:"career"`

	// HomeworldHexes are hexes on the character's own homeworld, priced
	// by its Trade Classifications, which the record carries.
	HomeworldHexes int `json:"homeworld_hexes"`

	// NoTCHexes are hexes on worlds the record does not name, priced at
	// the no-TC floor (interpretation I-82, ERRATA.md).
	NoTCHexes int `json:"no_tc_hexes"`

	Cite string `json:"cite"`
}

// Table is chart M1.
type Table struct {
	Cite         string        `json:"cite"`
	Kinds        []Benefit     `json:"kinds"`
	Entitlements []Entitlement `json:"entitlements"`
	Automatics   []Automatic   `json:"automatics"`

	// LandGrantHexes carries one entry per career that grants land.
	LandGrantHexes []LandGrantHexes `json:"land_grant_hexes"`

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

// LandGrantHexesFor returns how one career's grants are laid out.
func (t *Table) LandGrantHexesFor(career string) (LandGrantHexes, bool) {
	for _, h := range t.LandGrantHexes {
		if h.Career == career {
			return h, true
		}
	}

	return LandGrantHexes{}, false
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

	if err := t.validateAutomatics(); err != nil {
		return err
	}

	if err := t.validateLandGrantHexes(); err != nil {
		return err
	}

	return t.validateEntitlements()
}

// validateLandGrantHexes checks every granting career is named once and
// actually holds ground. A grant covering no hexes would earn nothing,
// which is not a reading any of pp. 41, 79 or 88 supports.
func (t *Table) validateLandGrantHexes() error {
	if len(t.LandGrantHexes) == 0 {
		return fmt.Errorf("%w: no land grant hexes", errBadTable)
	}

	seen := make(map[string]bool, len(t.LandGrantHexes))

	for _, h := range t.LandGrantHexes {
		if h.Career == "" || h.Cite == "" {
			return fmt.Errorf("%w: land grant hexes for %q are incomplete", errBadTable, h.Career)
		}

		if seen[h.Career] {
			return fmt.Errorf("%w: land grant hexes for %q listed twice", errBadTable, h.Career)
		}

		seen[h.Career] = true

		if h.HomeworldHexes < 0 || h.NoTCHexes < 0 || h.HomeworldHexes+h.NoTCHexes == 0 {
			return fmt.Errorf("%w: land grant for %q covers no hexes", errBadTable, h.Career)
		}
	}

	return nil
}

// validateAutomatics checks "Automatics (Subject to Eligibility)" is named
// and unambiguous, as the kinds are.
func (t *Table) validateAutomatics() error {
	if len(t.Automatics) == 0 {
		return fmt.Errorf("%w: no automatics", errBadTable)
	}

	seen := make(map[string]bool, len(t.Automatics))

	for _, a := range t.Automatics {
		if a.ID == "" || a.Name == "" || a.Eligibility == "" {
			return fmt.Errorf("%w: automatic %q is incomplete", errBadTable, a.ID)
		}

		if seen[a.ID] {
			return fmt.Errorf("%w: automatic %q listed twice", errBadTable, a.ID)
		}

		seen[a.ID] = true
	}

	return nil
}

// validateEntitlements checks chart M1's two Entitlements blocks and its
// Forbidden Knowledge table.
func (t *Table) validateEntitlements() error {
	seen := make(map[string]bool, len(t.Entitlements))

	for _, e := range t.Entitlements {
		if e.ID == "" || e.Name == "" {
			return fmt.Errorf("%w: entitlement %q is unnamed", errBadTable, e.ID)
		}

		if seen[e.ID] {
			return fmt.Errorf("%w: entitlement %q listed twice", errBadTable, e.ID)
		}

		seen[e.ID] = true

		if e.AnnualCredits == 0 && e.CreditsPerYear == 0 && e.CreditsPerTerm == 0 {
			return fmt.Errorf("%w: entitlement %q pays nothing", errBadTable, e.ID)
		}
	}

	if err := closedEntitlements(seen); err != nil {
		return err
	}

	return t.validateForbiddenKnowledge()
}

// closedEntitlements holds the entitlement ids the engine names, the way
// closedVocabulary holds the benefit kinds: every id the engine asks for
// must be in the data, and the data may not smuggle in one the engine
// does not know.
//
// The asymmetry this closes is at the two call sites.
// chargen/entitlement.go errors loudly on an unknown id in one place and
// silently skips in the other, so a removed or misspelt
// enlisted_retirement costs an Armed Forces character his retirement pay
// with no error anywhere — a record showing no entitlement, which reads
// exactly like not qualifying.
func closedEntitlements(seen map[string]bool) error {
	declared := []string{
		"citizen_pension", "functionary_pension", "reserve_pension",
		"professor_pension", "enlisted_retirement", "officer_retirement",
	}

	known := make(map[string]bool, len(declared))

	for _, id := range declared {
		known[id] = true

		if !seen[id] {
			return fmt.Errorf("%w: entitlement %q is named by the engine but absent from the table", errBadTable, id)
		}
	}

	for id := range seen {
		if !known[id] {
			return fmt.Errorf("%w: entitlement %q is in the table but not named by the engine", errBadTable, id)
		}
	}

	return nil
}

// validateForbiddenKnowledge checks chart M1's 1D table: exactly one row
// per face, since ForbiddenSkillAt answers with the first match.
func (t *Table) validateForbiddenKnowledge() error {
	rows := make(map[int]int, len(t.ForbiddenKnowledge))
	for _, f := range t.ForbiddenKnowledge {
		rows[f.Roll]++
	}

	for roll := 1; roll <= 6; roll++ {
		switch rows[roll] {
		case 1:
		case 0:
			return fmt.Errorf("%w: no Forbidden Knowledge for a roll of %d", errBadTable, roll)
		default:
			return fmt.Errorf("%w: %d Forbidden Knowledge rows for a roll of %d",
				errBadTable, rows[roll], roll)
		}
	}

	if len(t.ForbiddenKnowledge) != 6 {
		return fmt.Errorf("%w: the 1D Forbidden Knowledge table has %d rows",
			errBadTable, len(t.ForbiddenKnowledge))
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

		if err := validatePrice(b); err != nil {
			return err
		}

		if err := validateKnighthood(b); err != nil {
			return err
		}
	}

	return closedVocabulary(seen)
}

// pricedKinds are the benefits p. 68 prices in the chart itself, and the
// field each states its price in.
//
// Not every financial benefit belongs here, so a blanket "financial rows
// carry a value" rule would be wrong: money's amount comes from the
// career's own table D cell, and pension_doubling and retirement_doubling
// multiply something else. All three legitimately carry none.
var pricedKinds = map[Kind]func(Benefit) int{
	StarPassage:   func(b Benefit) int { return b.Credits },
	HighPassage:   func(b Benefit) int { return b.Credits },
	MiddlePassage: func(b Benefit) int { return b.Credits },
	LowPassage:    func(b Benefit) int { return b.Credits },
	Directorship:  func(b Benefit) int { return b.AnnualCredits },
	Proxy:         func(b Benefit) int { return b.AnnualCredits },
}

// validatePrice refuses a chart-priced benefit that lost its price.
//
// validateEntitlements has refused an entitlement that "pays nothing"
// since it was written; this is the same check on the other half of the
// file. Without it a benefit is awarded, recorded, and silently worth
// Cr0 — a data omission producing a wrong number rather than an error.
func validatePrice(b Benefit) error {
	price, priced := pricedKinds[b.Kind]
	if !priced {
		return nil
	}

	if price(b) <= 0 {
		return fmt.Errorf("%w: benefit %q carries no price", errBadTable, b.Kind)
	}

	return nil
}

// validateKnighthood refuses a Knighthood row missing the arithmetic that
// makes it a promotion rather than a demotion.
//
// The four fields are fields rather than prose because a note cannot be
// tested, and the consequence of losing one is not a missing award but a
// wrong one: musterOut computes the raise as SocFloor-soc, so an absent
// soc_floor awards a Soc-7 knight minus seven, recorded as an ordinary
// characteristic change with the sign inverted. An absent
// officers_only_careers makes the Spacer/Soldier/Marine restriction
// vanish; an absent non_officer_bonus awards a non-officer nothing.
func validateKnighthood(b Benefit) error {
	if b.Kind != Knighthood {
		return nil
	}

	if b.SocFloor < 1 || b.AboveFloorBonus < 1 || b.NonOfficerBonus < 1 || len(b.OfficersOnlyCareers) == 0 {
		return fmt.Errorf("%w: the Knighthood is missing its arithmetic "+
			"(soc_floor %d, above_floor_bonus %d, non_officer_bonus %d, %d officers-only careers)",
			errBadTable, b.SocFloor, b.AboveFloorBonus, b.NonOfficerBonus, len(b.OfficersOnlyCareers))
	}

	return nil
}

// closedVocabulary checks the vocabulary is closed in both directions:
// every kind the engine names must be in the data, and the data may not
// smuggle in a name the engine does not know — For would otherwise
// resolve it, and nothing could act on what came back.
func closedVocabulary(seen map[Kind]bool) error {
	declared := []Kind{
		Money, StarPassage, HighPassage, MiddlePassage, LowPassage,
		PensionDoubling, RetirementDoubling, Characteristic, WaferJack,
		ForbiddenKnowledge, Knighthood, Directorship, Proxy, LifeInsurance,
		TASFellowship, TASLife, ShipShares, LandGrant, Fame,
	}

	known := make(map[Kind]bool, len(declared))

	for _, kind := range declared {
		known[kind] = true

		if !seen[kind] {
			return fmt.Errorf("%w: %q is declared but absent from the table", errBadTable, kind)
		}
	}

	for kind := range seen {
		if !known[kind] {
			return fmt.Errorf("%w: %q is in the table but not the vocabulary", errBadTable, kind)
		}
	}

	return nil
}
