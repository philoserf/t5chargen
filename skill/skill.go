// Package skill holds the Master Skill List — the canonical identity of
// every skill, knowledge, talent, personal, and intuition a character can
// hold (Book 1 p. 132 chart MS). It is the single authority on skill
// names: the career, education, and world chart transcriptions are
// validated against it at load time, so two spellings of one skill cannot
// silently become two entries on a character sheet.
//
// The printed charts abbreviate ("Fwd Obs", "Naval Arch") and one chart
// prints two names for one skill (chart 04 table C "Navigator", table E
// "Navigation"); the Master Skill List governs. The chart-label to
// Master-Skill-List mapping is recorded in ERRATA.md I-9.
//
// Per the data/logic boundary (docs/PRD.md, Architecture notes) the list
// itself is embedded data; this file is lookup logic only.
package skill

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Kind discriminates the entry classes of chart MS. "A skill is a
// statement of ability based on a job, vocation, or interest. A knowledge
// is a body of information based on a field of science or experience. A
// talent is a personal ability not generally possible for a human, but
// which may be possible for some specific non-humans" (p. 132).
type Kind string

// The entry classes of the Master Skill List.
const (
	KindSkill     Kind = "skill"
	KindKnowledge Kind = "knowledge"
	KindTalent    Kind = "talent"
	KindPersonal  Kind = "personal"
	KindIntuition Kind = "intuition"
)

// SkillCount is the number of skills chart MS lists, printed on the page
// as its "64 SKILLS" heading. The loader checks the transcription against
// it.
const SkillCount = 64

// Entry is one Master Skill List entry.
type Entry struct {
	// Name is the canonical stored name. It is the printed name except
	// for a knowledge whose printed name appears under more than one
	// parent skill, which is qualified as "Parent: Name" — the book's own
	// convention for its Specialized knowledges ("Career: Scout-4",
	// "World: Egareva-6", p. 134). See ERRATA.md I-10.
	Name string

	// Printed is the name as chart MS prints it, unqualified.
	Printed string

	// Kind is the entry class.
	Kind Kind

	// Group is the chart MS heading a skill is listed under ("Skills",
	// "Starship Skills", "Trades", "Arts", "Soldier Skills"); empty for
	// other kinds.
	Group string

	// Parent is the containing skill of a knowledge ("Some skills include
	// within them several Knowledges", p. 133); empty otherwise.
	Parent string

	// Default reports membership of the chart MS Default Skills list —
	// "A skill identified as a Default skill may be used by any
	// character. The skill level used is 0" (p. 133).
	Default bool
}

// ErrUnknownSkill reports a name that is not in the Master Skill List.
var ErrUnknownSkill = errors.New("skill: unknown name")

// errBadList reports an invalid embedded Master Skill List.
var errBadList = errors.New("invalid master skill list")

// AmbiguousError reports a name that does not identify a single entry:
// either a knowledge name shared by several parent skills, or a chart
// label covering several list entries. Options holds the canonical
// alternatives in Master Skill List order.
type AmbiguousError struct {
	Name    string
	Options []string
}

// Error implements error.
func (e *AmbiguousError) Error() string {
	return fmt.Sprintf("skill: %q is ambiguous: %s", e.Name, strings.Join(e.Options, ", "))
}

// Lookup returns the entry with the given canonical name.
func Lookup(name string) (Entry, bool) {
	r, err := registry()
	if err != nil {
		return Entry{}, false
	}

	e, ok := r.byName[name]

	return e, ok
}

// Resolve maps a name as written in an embedded chart transcription to its
// canonical Master Skill List name. A name that identifies several entries
// — the chart 04 table E "Grav" and "Spacecraft" cells (ERRATA.md I-10,
// I-11) — returns an *AmbiguousError carrying the alternatives, which the
// engine presents as a choice.
func Resolve(name string) (string, error) {
	r, err := registry()
	if err != nil {
		return "", err
	}

	if _, ok := r.byName[name]; ok {
		return name, nil
	}

	if options, ok := r.ambiguous[name]; ok {
		return "", &AmbiguousError{Name: name, Options: options}
	}

	return "", fmt.Errorf("%w: %q", ErrUnknownSkill, name)
}

// Validate reports whether a name is usable in an embedded chart
// transcription: a canonical name, or a name whose ambiguity the engine
// resolves by choice.
func Validate(name string) error {
	if _, err := Resolve(name); err != nil {
		if _, ok := errors.AsType[*AmbiguousError](err); ok {
			return nil
		}

		return err
	}

	return nil
}

// Options returns the canonical alternatives a chart label covers, or nil
// when the name is unambiguous.
func Options(name string) []string {
	if _, err := Resolve(name); err != nil {
		if ambiguous, ok := errors.AsType[*AmbiguousError](err); ok {
			return ambiguous.Options
		}
	}

	return nil
}

// Skills returns the canonical names of the chart MS skills, in list
// order. "The list of Skills including Personals and Intuitions, is
// complete; there are no others available" (p. 132).
func Skills() []string {
	r, err := registry()
	if err != nil {
		return nil
	}

	return append([]string(nil), r.skills...)
}

// Names returns every canonical name in the list, sorted.
func Names() []string {
	r, err := registry()
	if err != nil {
		return nil
	}

	out := make([]string, 0, len(r.byName))
	for name := range r.byName {
		out = append(out, name)
	}

	sort.Strings(out)

	return out
}

// Err reports a failure to load or validate the embedded list.
func Err() error {
	_, err := registry()

	return err
}

//go:embed data/master_skill_list.json
var masterSkillList []byte

// file is the shape of the embedded Master Skill List.
type file struct {
	Cite          string       `json:"cite"`
	Note          string       `json:"note"`
	Skills        []skillRow   `json:"skills"`
	DefaultSkills []string     `json:"default_skills"`
	Talents       []string     `json:"talents"`
	Personals     []string     `json:"personals"`
	Intuitions    []string     `json:"intuitions"`
	Knowledges    []knowledges `json:"knowledges"`
	Labels        []label      `json:"labels"`
}

type skillRow struct {
	Name  string `json:"name"`
	Group string `json:"group"`
}

type knowledges struct {
	Parent string   `json:"parent"`
	Names  []string `json:"names"`
}

// label is a chart label that covers several list entries (ERRATA.md
// I-11).
type label struct {
	Label   string   `json:"label"`
	Options []string `json:"options"`
	Cite    string   `json:"cite"`
}

// table is the derived lookup structure.
type table struct {
	byName    map[string]Entry
	ambiguous map[string][]string
	skills    []string
	cite      string
}

// registry parses and validates the embedded list once.
var registry = sync.OnceValues(func() (*table, error) {
	var f file
	if err := json.Unmarshal(masterSkillList, &f); err != nil {
		return nil, fmt.Errorf("skill: parsing master skill list: %w", err)
	}

	t := &table{
		byName:    make(map[string]Entry),
		ambiguous: make(map[string][]string),
		cite:      f.Cite,
	}

	if err := t.addSkills(f.Skills); err != nil {
		return nil, err
	}

	if err := t.addKnowledges(f.Knowledges); err != nil {
		return nil, err
	}

	t.addSimple(f.Talents, KindTalent)
	t.addSimple(f.Personals, KindPersonal)
	t.addSimple(f.Intuitions, KindIntuition)

	if err := t.markDefaults(f.DefaultSkills); err != nil {
		return nil, err
	}

	if err := t.addLabels(f.Labels); err != nil {
		return nil, err
	}

	return t, t.validate()
})

// addSkills registers the 64 skills.
func (t *table) addSkills(rows []skillRow) error {
	for _, row := range rows {
		if row.Name == "" || row.Group == "" {
			return fmt.Errorf("%w: skill row %+v: name and group are required", errBadList, row)
		}

		if _, dup := t.byName[row.Name]; dup {
			return fmt.Errorf("%w: duplicate skill %q", errBadList, row.Name)
		}

		t.byName[row.Name] = Entry{
			Name:    row.Name,
			Printed: row.Name,
			Kind:    KindSkill,
			Group:   row.Group,
		}
		t.skills = append(t.skills, row.Name)
	}

	return nil
}

// addKnowledges registers the knowledges, qualifying any printed name
// shared by more than one parent skill (ERRATA.md I-10).
func (t *table) addKnowledges(groups []knowledges) error {
	parents := map[string][]string{}

	for _, group := range groups {
		if group.Parent == "" {
			return fmt.Errorf("%w: knowledge group without a parent", errBadList)
		}

		for _, name := range group.Names {
			parents[name] = append(parents[name], group.Parent)
		}
	}

	for _, group := range groups {
		for _, name := range group.Names {
			entry := Entry{
				Name:    name,
				Printed: name,
				Kind:    KindKnowledge,
				Parent:  group.Parent,
			}

			if len(parents[name]) > 1 {
				entry.Name = group.Parent + ": " + name
				t.ambiguous[name] = append(t.ambiguous[name], entry.Name)
			}

			if _, dup := t.byName[entry.Name]; dup {
				return fmt.Errorf("%w: duplicate knowledge %q", errBadList, entry.Name)
			}

			t.byName[entry.Name] = entry
		}
	}

	return nil
}

// addSimple registers the talents, personals, and intuitions.
func (t *table) addSimple(names []string, kind Kind) {
	for _, name := range names {
		if _, dup := t.byName[name]; dup {
			continue
		}

		t.byName[name] = Entry{Name: name, Printed: name, Kind: kind}
	}
}

// markDefaults flags the chart MS Default Skills.
func (t *table) markDefaults(names []string) error {
	for _, name := range names {
		entry, ok := t.byName[name]
		if !ok {
			return fmt.Errorf("%w: default skill %q is not in the list", errBadList, name)
		}

		entry.Default = true
		t.byName[name] = entry
	}

	return nil
}

// addLabels registers the chart labels that cover several entries.
func (t *table) addLabels(labels []label) error {
	for _, l := range labels {
		if len(l.Options) < 2 {
			return fmt.Errorf("%w: label %q needs at least two options", errBadList, l.Label)
		}

		if _, taken := t.byName[l.Label]; taken {
			return fmt.Errorf("%w: label %q is already a canonical name", errBadList, l.Label)
		}

		for _, option := range l.Options {
			if _, ok := t.byName[option]; !ok {
				return fmt.Errorf("%w: label %q option %q is not in the list", errBadList, l.Label, option)
			}
		}

		t.ambiguous[l.Label] = append([]string(nil), l.Options...)
	}

	return nil
}

// validate checks the transcription against the printed list.
func (t *table) validate() error {
	if len(t.skills) != SkillCount {
		return fmt.Errorf("%w: transcribed %d skills, chart MS prints %d", errBadList, len(t.skills), SkillCount)
	}

	if t.cite == "" {
		return fmt.Errorf("%w: missing its cite", errBadList)
	}

	return nil
}
