// Package career holds the data-driven career definitions: the tables,
// thresholds, and labels transcribed from the Book 1 career charts
// (pp. 75-88). Per the data/logic boundary (docs/PRD.md, Architecture
// notes), this package is data only — orchestration and career-specific
// mechanics are typed Go in the chargen package.
//
// v1 ships the Citizen career only (docs/PRD.md milestone 1); the remaining
// twelve careers land with milestone 3.
package career

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
)

// EntryKind discriminates what a career skills-table cell awards.
type EntryKind string

// The cell kinds of the career skills tables (chart 04 tables C and E,
// p. 78).
const (
	// EntrySkill awards the named skill.
	EntrySkill EntryKind = "skill"

	// EntryCharacteristic awards +1 to the named characteristic
	// (table C column 1 Personal).
	EntryCharacteristic EntryKind = "characteristic"

	// EntryMajor and EntryMinor award a level in the character's Major or
	// Minor: "If the character does not have a Major/Minor this benefit
	// is lost" (p. 78 table C note).
	EntryMajor EntryKind = "major"
	EntryMinor EntryKind = "minor"

	// EntryTrade, EntryArt, and EntryScience are the "One Trade", "One
	// Art", and "One Science" cells — a follow-on selection not
	// implemented until education and the Master Skill List land
	// (docs/PRD.md milestones 2-3).
	EntryTrade   EntryKind = "trade"
	EntryArt     EntryKind = "art"
	EntryScience EntryKind = "science"

	// EntryStarship is the "Starship Skill" cell (chart 05 table C,
	// p. 79) — a follow-on selection deferred with the Master Skill List
	// (docs/PRD.md milestone 3).
	EntryStarship EntryKind = "starship"

	// EntryNone is the "No Skill" cell of table E (p. 78).
	EntryNone EntryKind = "none"
)

// Entry is one cell of a career skills table.
type Entry struct {
	Kind EntryKind `json:"kind"`
	Name string    `json:"name,omitempty"`
}

// Column is one column of a career skills table (chart 04 table C): the
// character selects a column and rolls 1D for the specific cell (p. 65).
type Column struct {
	Name    string  `json:"name"`
	Entries []Entry `json:"entries"` // indexed by 1D-1
}

// Definition is one career's embedded data.
type Definition struct {
	Name string `json:"name"`
	Cite string `json:"cite"`

	// BeginChecks lists the characteristics the To Begin throw may check
	// (chart 05: "To Begin C1 or C2 or C3"); empty means Begin is
	// automatic (chart 04: "To Begin Auto").
	BeginChecks []string `json:"begin_checks,omitempty"`

	// RetryCheck is the characteristic for the career's Retry row
	// (chart 05: "Retry R&R C5"; interpretation I-8, ERRATA.md).
	RetryCheck string `json:"retry_check,omitempty"`

	// ControllingCharacteristics lists the characteristics available for
	// the career's Risk/Reward variant, in chart order (p. 64; for
	// Citizen, chart 04's "Citizen Life C1 C2 C3 C4").
	ControllingCharacteristics []string `json:"controlling_characteristics"`

	// ContinueTarget is a fixed roll-low Continue target (chart 04:
	// "Continue 10-"); ContinueCharacteristic names a characteristic
	// target instead (chart 05: "Continue Int"). Exactly one is set.
	ContinueTarget         int    `json:"continue_target,omitempty"`
	ContinueCharacteristic string `json:"continue_characteristic,omitempty"`

	// SkillsPerTerm is the table C eligibility (chart 04 table B:
	// "Per Term: 4 on Table C"). SkillEligibility carries per-duty
	// counts where the chart splits them (chart 05 table B: "If Courier
	// Duty 4 / If Explorer Duty 8"); mechanics consult it by duty label.
	SkillsPerTerm    int            `json:"skills_per_term"`
	SkillEligibility map[string]int `json:"skill_eligibility,omitempty"`

	// SkillColumns is table C, Citizen Skills.
	SkillColumns []Column `json:"skill_columns"`

	// JobTable is table E, Citizen Skills and Knowledges: "Roll three
	// dice for a specific Skill or Knowledge: Roll A (reroll if >3),
	// then roll B, and finally top row C." (p. 78) Indexed
	// [A-1][B-1][C-1]; nil for careers without one. The "No Skill" cell
	// is EntryNone.
	JobTable *[3][6][6]string `json:"job_table,omitempty"`

	cache derived
}

// NoSkillCell is the exact table E sentinel spelling; the loader validates
// cells against it so a transcription variant cannot silently become an
// awardable skill.
const NoSkillCell = "No Skill"

// JobEntry returns the table E cell for die faces a, b, c (each 1-based).
// Careers without a job table return EntryNone.
func (d *Definition) JobEntry(a, b, c int) Entry {
	if d.JobTable == nil {
		return Entry{Kind: EntryNone}
	}

	name := d.JobTable[a-1][b-1][c-1]
	if name == NoSkillCell {
		return Entry{Kind: EntryNone}
	}

	return Entry{Kind: EntrySkill, Name: name}
}

// SkillColumnNames returns the table C column names in chart order. The
// returned slice is shared; callers must not mutate it.
func (d *Definition) SkillColumnNames() []string {
	return d.cache.columnNames
}

// HobbyChoices returns every table E skill in chart order (A groups, then
// B rows, then C columns), deduplicated, excluding the No Skill cell — the
// alternatives for the Citizen Hobby selection (chart 04, p. 78). The
// returned slice is shared; callers must not mutate it.
func (d *Definition) HobbyChoices() []string {
	return d.cache.hobbyChoices
}

// derived caches computed at load time.
type derived struct {
	columnNames  []string
	hobbyChoices []string
}

// characteristicNames are the six standard human abbreviations valid in
// career data (p. 48).
var characteristicNames = map[string]bool{
	"Str": true, "Dex": true, "End": true, "Int": true, "Edu": true, "Soc": true,
}

// entryKinds are the cell kinds a career data file may use.
var entryKinds = map[EntryKind]bool{
	EntrySkill: true, EntryCharacteristic: true, EntryMajor: true, EntryMinor: true,
	EntryTrade: true, EntryArt: true, EntryScience: true, EntryStarship: true, EntryNone: true,
}

// validate rejects malformed-but-parseable career data at load time, so
// the engine can trust every definition unconditionally. All thirteen
// milestone-3 data files go through this same gate.
func (d *Definition) validate() error {
	if d.Name == "" || d.SkillsPerTerm < 1 {
		return fmt.Errorf("%w: name %q, skills per term %d", errBadDefinition, d.Name, d.SkillsPerTerm)
	}

	if err := d.validateContinue(); err != nil {
		return err
	}

	if err := d.validateCharacteristicNames(); err != nil {
		return err
	}

	if err := d.validateColumns(); err != nil {
		return err
	}

	return d.validateJobTable()
}

// validateContinue enforces exactly one Continue form: a fixed target in
// 2..11 (a target of 12 or more can never fail a 2D roll-low throw, p. 66,
// so the term loop would never end; 11 is the largest missable target), or
// a characteristic target (chart 05: "Continue Int").
func (d *Definition) validateContinue() error {
	fixed := d.ContinueTarget != 0
	characteristic := d.ContinueCharacteristic != ""

	if fixed == characteristic {
		return fmt.Errorf("%w: want exactly one of continue_target and continue_characteristic", errBadDefinition)
	}

	if fixed && (d.ContinueTarget < 2 || d.ContinueTarget > 11) {
		return fmt.Errorf("%w: continue target %d outside 2-11", errBadDefinition, d.ContinueTarget)
	}

	if characteristic && !characteristicNames[d.ContinueCharacteristic] {
		return fmt.Errorf("%w: unknown continue characteristic %q", errBadDefinition, d.ContinueCharacteristic)
	}

	return nil
}

// validateCharacteristicNames checks the begin/retry/controlling
// characteristic names against the six standard abbreviations.
func (d *Definition) validateCharacteristicNames() error {
	if len(d.ControllingCharacteristics) == 0 {
		return fmt.Errorf("%w: no controlling characteristics", errBadDefinition)
	}

	names := append(append([]string{}, d.BeginChecks...), d.RetryCheck)
	names = append(names, d.ControllingCharacteristics...)

	for _, name := range names {
		if name != "" && !characteristicNames[name] {
			return fmt.Errorf("%w: unknown characteristic %q", errBadDefinition, name)
		}
	}

	return nil
}

// validateColumns checks table C shape: named columns of six known cells.
func (d *Definition) validateColumns() error {
	if len(d.SkillColumns) == 0 {
		return fmt.Errorf("%w: no skill columns", errBadDefinition)
	}

	for _, column := range d.SkillColumns {
		if column.Name == "" || len(column.Entries) != 6 {
			return fmt.Errorf("%w: column %q has %d entries, want 6", errBadDefinition, column.Name, len(column.Entries))
		}

		for _, entry := range column.Entries {
			if err := validateEntry(column.Name, entry); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateEntry checks one table C cell.
func validateEntry(column string, entry Entry) error {
	if !entryKinds[entry.Kind] {
		return fmt.Errorf("%w: column %q has unknown kind %q", errBadDefinition, column, entry.Kind)
	}

	if entry.Kind == EntrySkill && entry.Name == "" {
		return fmt.Errorf("%w: column %q has a nameless skill cell", errBadDefinition, column)
	}

	if entry.Kind == EntryCharacteristic && !characteristicNames[entry.Name] {
		return fmt.Errorf("%w: column %q has unknown characteristic %q", errBadDefinition, column, entry.Name)
	}

	return nil
}

// validateJobTable checks table E: every cell non-empty, and any cell
// resembling the No Skill sentinel spelled exactly.
func (d *Definition) validateJobTable() error {
	if d.JobTable == nil {
		return nil
	}

	for a, group := range d.JobTable {
		for b, row := range group {
			for c, cell := range row {
				if cell == "" {
					return fmt.Errorf("%w: empty job table cell at A%d B%d C%d", errBadDefinition, a+1, b+1, c+1)
				}

				if cell != NoSkillCell && strings.EqualFold(cell, NoSkillCell) {
					return fmt.Errorf("%w: job table cell %q at A%d B%d C%d: want exactly %q",
						errBadDefinition, cell, a+1, b+1, c+1, NoSkillCell)
				}
			}
		}
	}

	return nil
}

// derive precomputes the shared column-name and hobby-choice lists.
func (d *Definition) derive() {
	d.cache.columnNames = make([]string, len(d.SkillColumns))
	for i, column := range d.SkillColumns {
		d.cache.columnNames[i] = column.Name
	}

	if d.JobTable == nil {
		return
	}

	seen := map[string]bool{}

	for a := 1; a <= len(d.JobTable); a++ {
		for b := 1; b <= 6; b++ {
			for c := 1; c <= 6; c++ {
				entry := d.JobEntry(a, b, c)
				if entry.Kind != EntrySkill || seen[entry.Name] {
					continue
				}

				seen[entry.Name] = true

				d.cache.hobbyChoices = append(d.cache.hobbyChoices, entry.Name)
			}
		}
	}
}

// errBadDefinition reports invalid career data.
var errBadDefinition = errors.New("invalid career definition")

//go:embed data/citizen.json
var citizenJSON []byte

//go:embed data/scout.json
var scoutJSON []byte

// citizen and scout parse and validate their embedded definitions once.
var (
	citizen = sync.OnceValues(func() (*Definition, error) {
		return load("citizen.json", citizenJSON)
	})
	scout = sync.OnceValues(func() (*Definition, error) {
		return load("scout.json", scoutJSON)
	})
)

// load parses, validates, and derives one career data file.
func load(name string, data []byte) (*Definition, error) {
	var d Definition
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, fmt.Errorf("career: parsing %s: %w", name, err)
	}

	if err := d.validate(); err != nil {
		return nil, fmt.Errorf("career: %s: %w", name, err)
	}

	d.derive()

	return &d, nil
}

// Citizen returns the Citizen career definition (chart 04, p. 78).
func Citizen() (*Definition, error) {
	return citizen()
}

// Scout returns the Scout career definition (chart 05, p. 79).
func Scout() (*Definition, error) {
	return scout()
}

// Available lists the implemented careers in Book 1 chart order (chart D,
// p. 64: Citizen is 04, Scout is 05). The default policy's first-listed
// career selection depends on this order (POLICY.md).
func Available() []string {
	return []string{"Citizen", "Scout"}
}
