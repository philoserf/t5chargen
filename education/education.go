// Package education holds the pre-career education data the engine
// consumes: the chart C program table and the Available Skills matrix
// (Book 1 p. 60; prose p. 59). Per the data/logic boundary (docs/PRD.md,
// Architecture notes) this package is data and validation only; the
// educational process mechanics are typed Go in the chargen package.
//
// Twelve of the seventeen program rows are implemented: ED5, Trade School,
// Apprenticeship, College, University and the Service Academy, then the
// four graduate rows (Masters, Professors, Medical School, Law School),
// and the two a career assigns rather than a character choosing — ANM
// School and Command College (p. 59, "attended during career
// resolution"). Later Education, which suspends a term for schooling
// (p. 59), is implemented in chargen.
//
// Five rows are transcribed and marked unimplemented. Mentoring's "C5=
// Tra" prerequisite and "Tra+2" award both need the non-human Training
// characteristic, a v1 non-goal (docs/PRD.md); Training Course, OTC, NOTC
// and Flight School are career-integrated. Chart C's Honors row is not a
// program: it is the post-graduation extra roll (p. 59), modeled as a
// mechanic in chargen — hence 17 program rows for the chart's 18.
package education

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/philoserf/t5chargen/skill"
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

	// RequiresProgram names, by Name as the character record spells it,
	// the programs a character must have attended
	// for the row to be open to him, beyond the Pre-Req column's own
	// entry: "College or University Honors Graduates who participated in
	// OTC or NOTC may attend Flight School" (p. 61). Any one of them
	// satisfies it.
	//
	// Chart C's Pre-Req column has one cell and this is a second
	// condition printed in the prose, which is why it is a field of its
	// own rather than a Prereq kind (interpretation I-110).
	RequiresProgram []string `json:"requires_program,omitempty"`

	// InFirstTerm marks a row attended inside a career rather than
	// chosen at step C: "The character attends Flight School in the
	// first year of his first term in the Navy, Army, or Marines"
	// (p. 60). Command College says the same of itself and reaches the
	// character as an assigned school; Flight School is elective, so it
	// is offered there instead (interpretation I-110).
	InFirstTerm bool `json:"in_first_term,omitempty"`

	// PreCareerOnly withholds the program from Later Education, which
	// otherwise offers "any Educational Institution or Training" at the
	// beginning of any term (p. 59).
	//
	// The Service Academy is the one program that cannot mean what it
	// says mid-career: it graduates a character into a commission and
	// "The character is required to serve one term in the service"
	// (p. 62), which is a thing that happens on entering a career, not
	// to someone already three terms into another (interpretation I-98).
	PreCareerOnly bool `json:"pre_career_only,omitempty"`

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

	if err := t.validateSkills(); err != nil {
		return err
	}

	if err := t.validateRequiredPrograms(); err != nil {
		return err
	}

	return t.validateNamedAwards()
}

// validateSkills holds every Available Skills row to the Master Skill
// List.
//
// Chart C's matrix names Master Skill List entries; the chart
// abbreviations are canonicalized in the transcription (ERRATA.md I-9).
// Unlike the career charts, every row here is unambiguous — the three
// Grav rows are distinguished by their parent group (ERRATA.md I-10) — so
// an ambiguous name is a transcription error too.
func (t *tableData) validateSkills() error {
	for _, s := range t.Skills {
		if s.Name == "" || s.Group == "" {
			return fmt.Errorf("%w: skill row %+v", errBadTable, s)
		}

		if _, err := skill.Resolve(s.Name); err != nil {
			return fmt.Errorf("%w: skill row %q: %w", errBadTable, s.Name, err)
		}
	}

	return nil
}

// validateRequiredPrograms holds every RequiresProgram entry to a program
// the table actually has.
//
// The field names programs the way the character record spells them,
// which is by Name rather than by ID, so a typo would silently withhold
// the row from everyone instead of failing — Flight School would simply
// never be offered and nothing would say why.
func (t *tableData) validateRequiredPrograms() error {
	names := map[string]bool{}
	for _, p := range t.Programs {
		names[p.Name] = true
	}

	for _, p := range t.Programs {
		for _, required := range p.RequiresProgram {
			if !names[required] {
				return fmt.Errorf("%w: program %q requires %q, which no program is named",
					errBadTable, p.ID, required)
			}
		}
	}

	return nil
}

// validateNamedAwards holds the two professional schools to their own
// columns.
//
// Medical School and Law School are the only rows whose Provides names a
// skill outright — "Medic-4" and "Advocate-2" (p. 60) — and each has an
// Available Skills column of its own, M and L. The award and the column
// are two transcriptions of the same fact, so each is checked against the
// other: a Medical School that did not teach Medic would be a
// transcription error in one of them, and neither would otherwise say so.
func (t *tableData) validateNamedAwards() error {
	for _, named := range []struct {
		skill string
		inst  Institution
	}{
		{MedicalAward, InstitutionMedical},
		{LawAward, InstitutionLaw},
	} {
		if !t.teaches(named.skill, named.inst) {
			return fmt.Errorf("%w: the %s column does not list %q, which its Provides awards",
				errBadTable, named.inst, named.skill)
		}
	}

	return nil
}

// teaches reports whether an institution's column lists a skill.
func (t *tableData) teaches(name string, inst Institution) bool {
	for _, s := range t.Skills {
		if s.Name == name && s.flag(inst) {
			return true
		}
	}

	return false
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

	if p.Prerequisite.Kind != "" && !prereqKinds[p.Prerequisite.Kind] {
		return fmt.Errorf("%w: program %q has prerequisite kind %q", errBadTable, p.ID, p.Prerequisite.Kind)
	}

	if p.MajorsFrom != "" && !majorsFrom[p.MajorsFrom] {
		return fmt.Errorf("%w: program %q takes majors from %q", errBadTable, p.ID, p.MajorsFrom)
	}

	return nil
}

// The two vocabularies that select behaviour and were left open, closed
// the way checkNames closes the characteristic names.
//
// A typo in either loads clean today and fails somewhere else. An unknown
// Kind falls through prereqMet's switch to false, so the gate simply
// stops existing: the row is offered to everyone at qualifies 0 and is
// reachable only by waiver, and the generated record looks ordinary. An
// unknown MajorsFrom makes Majors return an empty list, and generation
// dies two layers away with `"select_major" presented no options` —
// blaming the choice funnel for a transcription error.
var (
	prereqKinds = map[PrereqKind]bool{
		PrereqNone: true, PrereqEduMin: true, PrereqEduMax: true,
		PrereqTraMin: true, PrereqC5IsTra: true, PrereqDegree: true,
		PrereqAssigned: true, PrereqVolunteer: true,
	}

	// majorsFrom is the Institution vocabulary plus one sentinel. The
	// Service Academy's column is not fixed by the chart: the character
	// picks a service first, so "academy" is intercepted by eduRun's
	// institution() and resolved to army, navy or marine before Majors is
	// ever called. Closing this against Institution alone rejects the
	// Academy's own row.
	majorsFrom = map[string]bool{
		string(InstitutionCollege): true, string(InstitutionArmy): true,
		string(InstitutionNavy): true, string(InstitutionMarine): true,
		string(InstitutionSchool): true, string(InstitutionLaw): true,
		string(InstitutionMedical): true,
		"academy":                  true,
	}
)

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
	InstitutionLaw     Institution = "law"
	InstitutionMedical Institution = "medical"
)

// The skills chart C's professional schools award by name (p. 60), held
// here because the loader checks each against the column its school draws
// from.
const (
	// MedicalAward is Medical School's "Medic-4".
	MedicalAward = "Medic"

	// FlightAward and FlightAwardLevels are Flight School's "1x Pilot-3"
	// (chart C p. 60): one Pass/Fail roll, and three levels on it rather
	// than the one level a Pass usually carries. The worked example is
	// unambiguous — "He receives Pilot+3 for a total of Pilot-4" (p. 61).
	FlightAward       = "Pilot"
	FlightAwardLevels = 3

	// LawAward is Law School's "Advocate-2".
	LawAward = "Advocate"
)

// Majors returns the institution's available skills in chart order,
// deduplicated by name (Grav appears under Driver, Flyer, and Seafarer).
// The returned slice is fresh per call.
func Majors(inst Institution) ([]string, error) {
	return skillNames(func(s SkillRow) bool { return s.flag(inst) })
}

// ANMKnowledges returns the Knowledges ANM School may award: "Knowledge-2
// from School=ANM" (chart C p. 60). ANM is Army-Navy-Marine, so the source
// is the A, N and M columns of the Available Skills matrix taken together,
// narrowed to the entries the Master Skill List calls Knowledges — which
// is what the row asks for, and what keeps the award clear of the open
// question about awarding a bare container skill (COVERAGE.md, p. 134).
// The returned slice is fresh per call.
func ANMKnowledges() ([]string, error) {
	names, err := skillNames(func(s SkillRow) bool { return s.Army || s.Navy || s.Marine })
	if err != nil {
		return nil, err
	}

	knowledges := make([]string, 0, len(names))

	for _, name := range names {
		if entry, ok := skill.Lookup(name); ok && entry.Kind == skill.KindKnowledge {
			knowledges = append(knowledges, name)
		}
	}

	return knowledges, nil
}

// AllSkillNames returns every matrix skill name in chart order,
// deduplicated — the unrestricted Apprenticeship selection list
// (interpretation I-7, ERRATA.md). The returned slice is fresh per call.
func AllSkillNames() ([]string, error) {
	return skillNames(func(SkillRow) bool { return true })
}

// skillNames returns the matrix names matching keep, in chart order,
// deduplicated first-wins by name.
func skillNames(keep func(SkillRow) bool) ([]string, error) {
	t, err := load()
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}

	var names []string

	for _, s := range t.Skills {
		if !keep(s) || seen[s.Name] {
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
	case InstitutionLaw:
		return s.Law
	case InstitutionMedical:
		return s.Medical
	}

	return false
}
