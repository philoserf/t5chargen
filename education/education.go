// Package education holds the pre-career education data the engine
// consumes: the chart C program table and the Available Skills matrix
// (Book 1 p. 60; prose p. 59). Per the data/logic boundary (docs/PRD.md,
// Architecture notes) this package is data and validation only; the
// educational process mechanics are typed Go in the chargen package.
//
// v1 implements the docs/PRD.md FR3 programs (ED5, Trade School,
// College/University, Service Academy, Apprenticeship, Mentoring); the
// remaining chart C rows are transcribed but marked unimplemented, and
// Later Education (suspending a career term for schooling, p. 59) is
// deferred with career changes. Chart C's Honors row is not a program: it
// is the post-graduation extra roll (p. 59), modeled as a mechanic in
// chargen — hence 17 program rows for the chart's 18.
package education

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
)

// PrereqKind discriminates chart C prerequisite forms.
type PrereqKind string

// The prerequisite kinds of chart C (p. 60).
const (
	PrereqNone      PrereqKind = "none"
	PrereqEduMin    PrereqKind = "edu_min"   // "Edu 5+"
	PrereqEduMax    PrereqKind = "edu_max"   // ED5's "Edu 4 -"
	PrereqTraMin    PrereqKind = "tra_min"   // "Tra 5+"
	PrereqC5IsTra   PrereqKind = "c5_is_tra" // Mentor's "C5= Tra": v1 humans (C5=Edu) never qualify
	PrereqDegree    PrereqKind = "degree"    // "BA", "MA", "Honors BA"
	PrereqAssigned  PrereqKind = "assigned"  // assigned during career resolution
	PrereqVolunteer PrereqKind = "volunteer"
)

// Prereq is one program's prerequisite ("Pre-Requisites are minimums;
// higher are allowed", p. 59).
type Prereq struct {
	Kind      PrereqKind `json:"kind"`
	Value     int        `json:"value,omitempty"`
	ValueName string     `json:"value_name,omitempty"`
}

// Program is one chart C row (p. 60).
type Program struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Group string `json:"group"` // basic, higher, military

	Prerequisite Prereq `json:"prerequisite"`

	// ApplyCheck lists the characteristics for the admission check
	// ("To Apply ... Check one of the stated Characteristics", p. 59);
	// empty means automatic admission.
	ApplyCheck []string `json:"apply_check"`

	// DurationYears is the chart's Duration; an admission failure
	// consumes one year regardless ("A failure disallows admission and
	// consumes one year", p. 59).
	DurationYears int `json:"duration_years"`

	// PassCheck lists the characteristics for each Pass/Fail check;
	// Rolls is the chart's How Many Rolls.
	PassCheck []string `json:"pass_check"`
	Rolls     int      `json:"rolls"`

	// GraduationEdu is the chart's Graduation Edu value (0 = none):
	// "(If Edu already at this level, award Edu+1)" (p. 60).
	GraduationEdu    int    `json:"graduation_edu,omitempty"`
	GraduationDegree string `json:"graduation_degree,omitempty"`

	// MajorsFrom selects the Available Skills column for Major/Minor
	// selection: "college", "school", "academy" (per-service), or empty
	// for programs without majors.
	MajorsFrom string `json:"majors_from,omitempty"`

	// Implemented gates the engine: unimplemented rows are transcribed
	// for completeness but rejected if selected (v1 scope is docs/PRD.md
	// FR3).
	Implemented bool `json:"implemented"`
}

// SkillRow is one Available Skills matrix row (p. 60). The flags name the
// institutions whose lists include the skill; L and M cells map to law and
// medical. Bold rows are Knowledge-only skills.
type SkillRow struct {
	Name  string `json:"name"`
	Group string `json:"group"`

	College bool `json:"college,omitempty"`
	Army    bool `json:"army,omitempty"`
	Navy    bool `json:"navy,omitempty"`
	Marine  bool `json:"marine,omitempty"`
	School  bool `json:"school,omitempty"`
	Law     bool `json:"law,omitempty"`
	Medical bool `json:"medical,omitempty"`

	KnowledgeOnly bool `json:"knowledge_only,omitempty"`
}

// table is the parsed chart C data.
type tableData struct {
	Cite     string     `json:"cite"`
	Programs []Program  `json:"programs"`
	Skills   []SkillRow `json:"skills"`
}

//go:embed data/education.json
var educationJSON []byte

// load parses and validates the embedded chart C data once.
var load = sync.OnceValues(func() (*tableData, error) {
	var t tableData
	if err := json.Unmarshal(educationJSON, &t); err != nil {
		return nil, fmt.Errorf("education: parsing education.json: %w", err)
	}

	if err := t.validate(); err != nil {
		return nil, fmt.Errorf("education: education.json: %w", err)
	}

	return &t, nil
})

// errBadTable reports invalid chart C data.
var errBadTable = errors.New("invalid education table")

// checkNames are the characteristics chart C checks against.
var checkNames = map[string]bool{
	"Str": true, "Dex": true, "End": true, "Int": true, "Edu": true, "Soc": true,
	// Tra is the non-human Training characteristic (p. 55); humans
	// substitute Edu at half value (see chargen).
	"Tra": true,
}

// validate rejects malformed-but-parseable chart C data at load time.
func (t *tableData) validate() error {
	if len(t.Programs) == 0 || len(t.Skills) == 0 {
		return fmt.Errorf("%w: empty programs or skills", errBadTable)
	}

	ids := map[string]bool{}

	for _, p := range t.Programs {
		if ids[p.ID] {
			return fmt.Errorf("%w: duplicate program id %q", errBadTable, p.ID)
		}

		ids[p.ID] = true

		if err := p.validate(); err != nil {
			return err
		}
	}

	for _, s := range t.Skills {
		if s.Name == "" || s.Group == "" {
			return fmt.Errorf("%w: skill row %+v", errBadTable, s)
		}
	}

	return nil
}

// validate checks one program row.
func (p Program) validate() error {
	if p.ID == "" || p.Name == "" || p.Rolls < 1 {
		return fmt.Errorf("%w: program %+v", errBadTable, p)
	}

	for _, name := range append(append([]string{}, p.ApplyCheck...), p.PassCheck...) {
		if !checkNames[name] {
			return fmt.Errorf("%w: program %q checks unknown characteristic %q", errBadTable, p.ID, name)
		}
	}

	return nil
}

// Programs returns the chart C rows in chart order. The returned slice is
// shared; callers must not mutate it.
func Programs() ([]Program, error) {
	t, err := load()
	if err != nil {
		return nil, err
	}

	return t.Programs, nil
}

// ErrUnknownProgram reports a program id outside chart C.
var ErrUnknownProgram = errors.New("unknown education program")

// ProgramByID returns one chart C row.
func ProgramByID(id string) (Program, error) {
	t, err := load()
	if err != nil {
		return Program{}, err
	}

	for _, p := range t.Programs {
		if p.ID == id {
			return p, nil
		}
	}

	return Program{}, fmt.Errorf("%w: %q", ErrUnknownProgram, id)
}

// Institution selects an Available Skills column.
type Institution string

// The Available Skills columns (p. 60).
const (
	InstitutionCollege Institution = "college"
	InstitutionArmy    Institution = "army"
	InstitutionNavy    Institution = "navy"
	InstitutionMarine  Institution = "marine"
	InstitutionSchool  Institution = "school"
)

// Majors returns the institution's available skills in chart order,
// deduplicated by name (Grav appears under Driver, Flyer, and Seafarer).
// The returned slice is fresh per call.
func Majors(inst Institution) ([]string, error) {
	t, err := load()
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}

	var names []string

	for _, s := range t.Skills {
		if !s.flag(inst) || seen[s.Name] {
			continue
		}

		seen[s.Name] = true

		names = append(names, s.Name)
	}

	return names, nil
}

// AllSkillNames returns every matrix skill name in chart order,
// deduplicated — the unrestricted Apprenticeship selection list
// (interpretation I-7, ERRATA.md). The returned slice is fresh per call.
func AllSkillNames() ([]string, error) {
	t, err := load()
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}

	var names []string

	for _, s := range t.Skills {
		if seen[s.Name] {
			continue
		}

		seen[s.Name] = true

		names = append(names, s.Name)
	}

	return names, nil
}

// flag reads the institution's column.
func (s SkillRow) flag(inst Institution) bool {
	switch inst {
	case InstitutionCollege:
		return s.College
	case InstitutionArmy:
		return s.Army
	case InstitutionNavy:
		return s.Navy
	case InstitutionMarine:
		return s.Marine
	case InstitutionSchool:
		return s.School
	}

	return false
}
