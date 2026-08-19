package chargen

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/dice"
)

// Provenance constants for the replay and provenance contract (docs/PRD.md):
// every character record carries them so old characters stay auditable after
// embedded tables change. Changing the RNG algorithm, the seeded stream's
// consumption order, or the default policy is a version bump; EngineVersion
// is hand-bumped in v1 (no build-info plumbing).
const (
	// SchemaVersion identifies the character JSON schema.
	SchemaVersion = "0.2.0"

	// Ruleset is pinned: all rule citations resolve against this artifact.
	Ruleset = "Traveller5 Core Rules Book 1, Print Edition 5.1"

	// EngineVersion identifies this implementation of the generation
	// procedure, including the seeded stream's consumption order.
	EngineVersion = "0.2.0"

	// PolicyVersion identifies the auto-mode decision table in POLICY.md
	// (docs/PRD.md, CLI sketch). Changing the policy is a version bump.
	PolicyVersion = "0.1.0"

	// RNGAlgorithm names the recorded random stream: Go math/rand/v2 PCG,
	// seeded as documented at dice.New. The exact string is compared on
	// replay; changing it is a version bump.
	RNGAlgorithm = "math/rand/v2-pcg"
)

// StartAge is the age at which career resolution begins: "he begins career
// resolution at the start of Life Stage-3 (at age 18)" (p. 59).
const StartAge = 18

// TermYears is the length of one career term: "the 4-year Term" (p. 66).
const TermYears = 4

// RNG records the random stream a character was generated from
// (docs/PRD.md, Replay and provenance contract).
type RNG struct {
	Algorithm string `json:"algorithm"`
	Seed      uint64 `json:"seed"`
}

// Character is the character record (docs/PRD.md FR8). The JSON record is
// the source of truth; rendered sheets are derived from it. UPP is a stored
// derived value (docs/PRD.md, JSON conventions): replay recomputes and
// compares it.
type Character struct {
	SchemaVersion string `json:"schema_version"`
	Ruleset       string `json:"ruleset"`
	EngineVersion string `json:"engine_version"`
	PolicyVersion string `json:"policy_version"`
	RNG           RNG    `json:"rng"`
	// Errata lists applied ERRATA.md deviations. Unlike PolicyVersion
	// (whose "none" sentinel keeps the key always present), an empty list
	// is deliberately absent from the JSON: the PRD requires recording
	// "any applied deviations", so absence means none were applied.
	Errata []string `json:"errata,omitempty"`

	Name            string          `json:"name,omitempty"` // blank by default (docs/PRD.md, Decisions)
	Characteristics Characteristics `json:"characteristics"`
	UPP             string          `json:"upp"`
	Age             int             `json:"age"`
	Skills          []Skill         `json:"skills,omitempty"`  // sorted by name for canonical JSON
	Careers         []CareerRecord  `json:"careers,omitempty"` // in order served

	Events []Event `json:"events"`
}

// Skill is one acquired skill or knowledge at its current level. The
// Skill/Knowledge distinction sharpens with the Master Skill List
// (docs/PRD.md FR5, milestone 3).
type Skill struct {
	Name  string `json:"name"`
	Level int    `json:"level"`
}

// CareerRecord is one career's history, term by term (docs/PRD.md FR8).
// Job and Hobby are per-career: "Once determined, Job and Hobby cannot be
// changed" (chart 04 p. 78).
type CareerRecord struct {
	Career string       `json:"career"`
	Job    string       `json:"job,omitempty"`
	Hobby  string       `json:"hobby,omitempty"`
	Terms  []TermRecord `json:"terms"`
}

// TermRecord is one term's outcome.
type TermRecord struct {
	Term                      int    `json:"term"`
	ControllingCharacteristic string `json:"controlling_characteristic"`
	Success                   bool   `json:"success"`   // the Risk/Reward-variant outcome (Citizen Life)
	Continued                 bool   `json:"continued"` // the Continue roll's outcome
}

// SkillMax caps skill levels: "Skill, Knowledge, and Talent Maximums:
// Skill-15" (p. 134). The Knowledge-6 cap lands with the Master Skill List
// (docs/PRD.md FR5, milestone 3).
const SkillMax = 15

// awardSkill increases a skill by the given levels, capped at SkillMax and
// keeping Skills sorted by name. It returns the resulting level, then the
// levels actually applied.
func (c *Character) awardSkill(name string, levels int) (int, int) {
	for i := range c.Skills {
		if c.Skills[i].Name == name {
			applied := min(levels, SkillMax-c.Skills[i].Level)
			c.Skills[i].Level += applied

			return c.Skills[i].Level, applied
		}
	}

	applied := min(levels, SkillMax)
	c.Skills = append(c.Skills, Skill{Name: name, Level: applied})
	slices.SortFunc(c.Skills, func(a, b Skill) int { return strings.Compare(a.Name, b.Name) })

	return applied, applied
}

// skillLevel reports the current level of a skill, 0 if not held.
func (c *Character) skillLevel(name string) int {
	for _, skill := range c.Skills {
		if skill.Name == name {
			return skill.Level
		}
	}

	return 0
}

// Options configures one generation run.
type Options struct {
	Seed uint64
	Name string

	// Career forces the first career by name ("--career forces the first
	// career only", docs/PRD.md CLI sketch). Empty leaves the selection
	// to the Decider.
	Career string

	// Decider resolves every choice point; nil uses DefaultPolicy.
	Decider Decider
}

// Generate runs the generation procedure and returns the character record.
// It currently covers checklist steps A (Generate Characteristics) and D
// (Select Career) plus career resolution for the implemented careers
// (chart E1, p. 72); homeworld and education (steps B-C) land with
// docs/PRD.md milestone 2, and aging, career changes, muster out, and fame
// with milestone 4.
func Generate(opts Options) (Character, error) {
	decider := opts.Decider
	if decider == nil {
		decider = DefaultPolicy{}
	}

	roller := dice.New(opts.Seed)

	var log Log

	character := Character{
		SchemaVersion: SchemaVersion,
		Ruleset:       Ruleset,
		EngineVersion: EngineVersion,
		PolicyVersion: PolicyVersion,
		RNG:           RNG{Algorithm: RNGAlgorithm, Seed: opts.Seed},
		Name:          opts.Name,
		Age:           StartAge,
	}

	log.Step("Generate Characteristics", "Book 1 p. 72 chart E1 step A")

	character.Characteristics = RollCharacteristics(roller, &log)

	if err := runCareer(opts.Career, roller, &log, decider, &character); err != nil {
		return Character{}, err
	}

	character.UPP = character.Characteristics.UPP()
	character.Events = log.Events()

	return character, nil
}

// errUnknownCareer reports a forced career that is not implemented.
var errUnknownCareer = errors.New("unknown career")

// runCareer performs checklist step D (Select Career, chart E1 p. 72) and
// resolves the selected career.
func runCareer(forced string, roller *dice.Roller, log *Log, decider Decider, character *Character) error {
	options := career.Available()

	if forced != "" {
		if !slices.Contains(options, forced) {
			return fmt.Errorf("%w: %q (available: %s)", errUnknownCareer, forced, strings.Join(options, ", "))
		}

		options = []string{forced}
	}

	log.Step("Select Career", "Book 1 p. 72 chart E1 step D")

	chosen := choose(log, decider, Choice{
		ID:      ChooseCareer,
		Prompt:  "Select career",
		Options: options,
		Cite:    "Book 1 p. 72 chart E1 step D",
	})

	// Citizen is the only implemented career (docs/PRD.md milestone 1).
	if options[chosen] != "Citizen" {
		return fmt.Errorf("%w: %q", errUnknownCareer, options[chosen])
	}

	return runCitizen(roller, log, decider, character)
}

// choose puts a choice to the decider, clamps a misbehaving answer to the
// first-listed option, and logs the resolved choice event.
func choose(log *Log, decider Decider, c Choice) int {
	chosen := decider.Choose(c)
	if chosen < 0 || chosen >= len(c.Options) {
		chosen = 0
	}

	log.Choice(ChoiceEvent{
		Decider: decider.Kind(),
		Prompt:  c.Prompt,
		Options: c.Options,
		Chosen:  chosen,
		Cite:    c.Cite,
	})

	return chosen
}
