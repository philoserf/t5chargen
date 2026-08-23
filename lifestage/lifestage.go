// Package lifestage holds chart A's Stages of Life table (Book 1 p. 89)
// and the arithmetic that maps an age onto a stage.
//
// The stage is what the Aging Check is thrown against — "2D < Life Stage"
// — and the stage a character has reached is what decides whether aging
// applies at all: Physical Aging "begins at age 34 (the beginning of Life
// Stage 5 Peak)" and Mental Aging "begins at age 66 (the beginning of Life
// Stage 9- Retirement)".
//
// Per the data/logic boundary (docs/PRD.md, Architecture notes) the table
// is embedded data; this file is lookup logic only.
package lifestage

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
)

// Stage is one row of the Stages of Life table.
type Stage struct {
	Number int    `json:"number"`
	Name   string `json:"name"`

	// First and Last are the human years the chart prints for the stage.
	// They are transcription, not arithmetic: the engine derives a stage
	// from Of, and one printed row disagrees with the derivation (see
	// interpretation I-48, ERRATA.md).
	First int `json:"first"`
	Last  int `json:"last"`
}

// Table is chart A's Stages of Life, with the thresholds the Aging Check
// reads from the same page.
type Table struct {
	Cite string `json:"cite"`

	// InfancyYears and StageYears are the chart's structure: "All sophonts
	// have an approximately 2-year-long infancy" and "After Infancy, each
	// Life Stage is two terms (8 years)".
	InfancyYears int `json:"infancy_years"`
	StageYears   int `json:"stage_years"`

	// PhysicalStage and MentalStage are where each kind of aging begins:
	// "Physical Aging begins at Peak (= 34 for Humans)" and "Mental aging
	// begins at Retirement (= 66 for Humans)".
	PhysicalStage int `json:"physical_stage"`
	MentalStage   int `json:"mental_stage"`

	// CheckIntervalYears: "Once Aging begins, it occurs every four years
	// on the character's birthday."
	CheckIntervalYears int `json:"check_interval_years"`

	// PhysicalCharacteristics and MentalCharacteristics are what each kind
	// of aging affects: "Human Physical Aging affects Strength, Dexterity,
	// and Endurance" and "Human Mental Aging affects Intelligence".
	PhysicalCharacteristics []string `json:"physical_characteristics"`
	MentalCharacteristics   []string `json:"mental_characteristics"`

	// TraditionalLifespan is flavor, not a bound: "The traditional
	// lifespan for humans is 74 years (although certainly some may live
	// longer, and some may live shorter lives)."
	TraditionalLifespan int `json:"traditional_lifespan"`

	Stages []Stage `json:"stages"`
}

//go:embed data/lifestages.json
var stagesJSON []byte

var errBadTable = errors.New("invalid life stages table")

// Load returns the embedded table.
func Load() (*Table, error) { return table() }

var table = sync.OnceValues(func() (*Table, error) {
	var t Table
	if err := json.Unmarshal(stagesJSON, &t); err != nil {
		return nil, fmt.Errorf("life stages table: %w", err)
	}

	if err := t.validate(); err != nil {
		return nil, err
	}

	return &t, nil
})

// Of returns the life stage an age falls in.
//
// Derived rather than looked up: "All sophonts have an approximately
// 2-year-long infancy" and "After Infancy, each Life Stage is two terms (8
// years)". The derivation reproduces every anchor the page states in
// prose — 18 is the start of Young Adult (3), 34 of Peak (5), 66 of
// Retirement (9) — where the printed year ranges do not (I-48).
//
// The last stage is the last: the chart prints no stage 10, and a
// character older than its span stays in Retirement.
func (t *Table) Of(age int) int {
	if age < t.InfancyYears {
		return 0
	}

	return min((age-t.InfancyYears)/t.StageYears+1, t.Stages[len(t.Stages)-1].Number)
}

// Name returns the printed name of a stage number.
func (t *Table) Name(stage int) string {
	for _, s := range t.Stages {
		if s.Number == stage {
			return s.Name
		}
	}

	return ""
}

// FirstYearOf returns the age a stage begins at, derived the same way as
// Of: "All sophonts have an approximately 2-year-long infancy" and each
// stage after it "is two terms (8 years)". Stage 0 begins at birth.
func (t *Table) FirstYearOf(stage int) int {
	if stage < 1 {
		return 0
	}

	return t.InfancyYears + (stage-1)*t.StageYears
}

// validate rejects embedded data the lookups assume is well-formed.
func (t *Table) validate() error {
	if len(t.Stages) == 0 {
		return fmt.Errorf("%w: no stages", errBadTable)
	}

	if t.InfancyYears < 1 || t.StageYears < 1 || t.CheckIntervalYears < 1 {
		return fmt.Errorf("%w: non-positive year counts", errBadTable)
	}

	for i, s := range t.Stages {
		if s.Number != i {
			return fmt.Errorf("%w: stage %d out of order at index %d", errBadTable, s.Number, i)
		}

		if s.Name == "" {
			return fmt.Errorf("%w: stage %d has no name", errBadTable, s.Number)
		}
	}

	return t.validateAging()
}

// validateAging checks the thresholds the Aging Check reads: where each
// kind of aging begins, and what it affects.
func (t *Table) validateAging() error {
	if t.PhysicalStage < 1 || t.PhysicalStage >= len(t.Stages) {
		return fmt.Errorf("%w: physical stage %d off the table", errBadTable, t.PhysicalStage)
	}

	if t.MentalStage < 1 || t.MentalStage >= len(t.Stages) {
		return fmt.Errorf("%w: mental stage %d off the table", errBadTable, t.MentalStage)
	}

	if t.MentalStage < t.PhysicalStage {
		return fmt.Errorf("%w: mental aging begins before physical", errBadTable)
	}

	if len(t.PhysicalCharacteristics) == 0 || len(t.MentalCharacteristics) == 0 {
		return fmt.Errorf("%w: aging affects no characteristics", errBadTable)
	}

	return nil
}
