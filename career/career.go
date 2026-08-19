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
	"fmt"
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

	// CitizenLifeCharacteristics lists the characteristics available as
	// the Citizen Life controlling characteristic, in chart order
	// (chart 04: "Citizen Life C1 C2 C3 C4").
	CitizenLifeCharacteristics []string `json:"citizen_life_characteristics"`

	// ContinueTarget is the roll-low Continue target (chart 04:
	// "Continue 10-").
	ContinueTarget int `json:"continue_target"`

	// SkillsPerTerm is the table C eligibility (chart 04 table B:
	// "Per Term: 4 on Table C").
	SkillsPerTerm int `json:"skills_per_term"`

	// SkillColumns is table C, Citizen Skills.
	SkillColumns []Column `json:"skill_columns"`

	// JobTable is table E, Citizen Skills and Knowledges: "Roll three
	// dice for a specific Skill or Knowledge: Roll A (reroll if >3),
	// then roll B, and finally top row C." (p. 78) Indexed
	// [A-1][B-1][C-1]. The "No Skill" cell is EntryNone.
	JobTable [3][6][6]string `json:"job_table"`
}

// JobEntry returns the table E cell for die faces a, b, c (each 1-based).
func (d *Definition) JobEntry(a, b, c int) Entry {
	name := d.JobTable[a-1][b-1][c-1]
	if name == "No Skill" {
		return Entry{Kind: EntryNone}
	}

	return Entry{Kind: EntrySkill, Name: name}
}

//go:embed data/citizen.json
var citizenJSON []byte

// citizen parses the embedded Citizen definition once.
var citizen = sync.OnceValues(func() (*Definition, error) {
	var d Definition
	if err := json.Unmarshal(citizenJSON, &d); err != nil {
		return nil, fmt.Errorf("career: parsing citizen.json: %w", err)
	}

	return &d, nil
})

// Citizen returns the Citizen career definition (chart 04, p. 78).
func Citizen() (*Definition, error) {
	return citizen()
}

// Available lists the implemented careers in Book 1 chart order
// (chart D, p. 64). v1 milestone 1 ships Citizen only.
func Available() []string {
	return []string{"Citizen"}
}
