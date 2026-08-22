package chargen

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/dice"
	"github.com/philoserf/t5chargen/world"
)

// Provenance constants for the replay and provenance contract (docs/PRD.md):
// every character record carries them so old characters stay auditable after
// embedded tables change. Changing the RNG algorithm, the seeded stream's
// consumption order, or the default policy is a version bump; EngineVersion
// is hand-bumped in v1 (no build-info plumbing).
const (
	// SchemaVersion identifies the character JSON schema.
	SchemaVersion = "0.15.0"

	// Ruleset is pinned: all rule citations resolve against this artifact.
	Ruleset = "Traveller5 Core Rules Book 1, Print Edition 5.1"

	// EngineVersion identifies this implementation of the generation
	// procedure, including the seeded stream's consumption order.
	EngineVersion = "0.15.0"

	// PolicyVersion identifies the auto-mode decision table in POLICY.md
	// (docs/PRD.md, CLI sketch). Changing the policy is a version bump.
	PolicyVersion = "0.11.0"

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
	// Errata lists applied ERRATA.md deviations. Unlike policy_version
	// (always present — the POLICY.md version, or "none" when the run's
	// choices were not governed by the default policy), an empty list is
	// deliberately absent from the JSON: the PRD requires recording "any
	// applied deviations", so absence means none were applied.
	Errata []string `json:"errata,omitempty"`

	Name            string          `json:"name,omitempty"` // blank by default (docs/PRD.md, Decisions)
	Characteristics Characteristics `json:"characteristics"`
	UPP             string          `json:"upp"`
	// Homeworld doubles as the birthworld in v1 (docs/PRD.md FR2).
	Homeworld world.Homeworld   `json:"homeworld"`
	Age       int               `json:"age"`
	Education []EducationRecord `json:"education,omitempty"` // in order attended
	Skills    []Skill           `json:"skills,omitempty"`    // sorted by name for canonical JSON
	Careers   []CareerRecord    `json:"careers,omitempty"`   // in order served

	// WaiversAttempted counts Educational Waiver rolls, successful or
	// not, lifetime ("Mod minus number of previous waivers rolled
	// (successful or not)", p. 59).
	WaiversAttempted int `json:"waivers_attempted,omitempty"`

	// Fame is the running Fame counter (chart 05 "Fame +1"; the full
	// Fame system, chart F p. 91, lands with milestone 4).
	Fame int `json:"fame,omitempty"`

	// WoundBadges counts Risk-failure wounds (p. 65).
	WoundBadges int `json:"wound_badges,omitempty"`

	// Disabled: a controlling characteristic was reduced by 4 or more
	// ("he is disabled. Muster Out at Term end", chart 05 p. 79; p. 65).
	Disabled bool `json:"disabled,omitempty"`

	// Dead: a controlling characteristic was reduced to zero or less
	// ("the Character is dead", p. 65). Generation ends at the injury.
	Dead bool `json:"dead,omitempty"`

	Events []Event `json:"events"`
}

// EducationRecord is one educational process (chart C p. 60; docs/PRD.md
// FR3). A slice on the record because Later Education (p. 59) will allow
// more than one.
type EducationRecord struct {
	Program string `json:"program"`
	Service string `json:"service,omitempty"` // Service Academy only
	Major   string `json:"major,omitempty"`
	Minor   string `json:"minor,omitempty"`

	// Skill is the Apprenticeship's chosen "Skill+4" (chart C p. 60) —
	// distinct from Major: p. 59 scopes Major/Minor to Educational
	// Institutions, and Apprenticeship is a Training institution.
	Skill     string `json:"skill,omitempty"`
	Passes    int    `json:"passes"`
	Graduated bool   `json:"graduated"`
	Honors    bool   `json:"honors,omitempty"`
	Degree    string `json:"degree,omitempty"`
}

// currentMajor and currentMinor report the character's Major and Minor:
// "A character's current Major and Minor are the most recent ones
// selected" (p. 59).
func (c *Character) currentMajor() string {
	// A career may select its own Major where the character arrived
	// without a degree (chart 02's Scholar); the most recent selection
	// wins, career records being later than education ones.
	for _, record := range slices.Backward(c.Careers) {
		if record.Major != "" {
			return record.Major
		}
	}

	for _, record := range slices.Backward(c.Education) {
		if record.Major != "" {
			return record.Major
		}
	}

	return ""
}

func (c *Character) currentMinor() string {
	for _, record := range slices.Backward(c.Careers) {
		if record.Minor != "" {
			return record.Minor
		}
	}

	for _, record := range slices.Backward(c.Education) {
		if record.Minor != "" {
			return record.Minor
		}
	}

	return ""
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
	Career string `json:"career"`

	// Began records the To Begin outcome; a failed Begin leaves a
	// began:false record with no terms ("this career may not be used",
	// p. 65).
	Began bool `json:"began"`

	Job   string `json:"job,omitempty"`
	Hobby string `json:"hobby,omitempty"`

	// Discoveries counts Scout Reward successes (chart 05, p. 79).
	Discoveries int `json:"discoveries,omitempty"`

	// Rank and RankTitle are the rank held on leaving the career, for the
	// careers with a rank table (chart 06, p. 80); empty for the careers
	// with no rank (p. 65).
	Rank      string `json:"rank,omitempty"`
	RankTitle string `json:"rank_title,omitempty"`

	// ControllingCharacteristic is the career-long pick of a career that
	// chooses one rather than rotating (chart 10, p. 84).
	ControllingCharacteristic string `json:"controlling_characteristic,omitempty"`

	// Scheme is the Rogue's current plan, PrisonYears a sentence owed at
	// the start of the next term, and SchemePayoff the credits his
	// schemes have paid (chart 10, p. 84). Spending it lands with muster
	// out (docs/PRD.md milestone 4).
	Scheme       string `json:"scheme,omitempty"`
	PrisonYears  int    `json:"prison_years,omitempty"`
	SchemePayoff int    `json:"scheme_payoff,omitempty"`

	// UndercoverCareer and UndercoverTitle are the Agent's current cover
	// identity (chart 09, p. 83); Commendations counts his Reward
	// successes.
	UndercoverCareer string `json:"undercover_career,omitempty"`
	UndercoverTitle  string `json:"undercover_title,omitempty"`
	Commendations    int    `json:"commendations,omitempty"`

	// Branch is the Armed Forces branch served in (chart 08, p. 82);
	// Medals are the decorations earned, in the card's own notation
	// ("MCUF-1. XS-1.", p. 65); ServiceBadges counts the Risk-success
	// Exemplary Service Badges, which carry no promotion modifier
	// (interpretation I-32, ERRATA.md).
	Branch        string  `json:"branch,omitempty"`
	Medals        []Award `json:"medals,omitempty"`
	ServiceBadges int     `json:"service_badges,omitempty"`

	// Exiled, TimesExiled, and SuccessfulIntrigues are the Noble's state
	// (chart 11, p. 85): "Exile is a banishment to the edges of the empire
	// orchestrated by political enemies." The two counters modify the
	// Return and Intrigue rolls.
	Exiled              bool `json:"exiled,omitempty"`
	TimesExiled         int  `json:"times_exiled,omitempty"`
	SuccessfulIntrigues int  `json:"successful_intrigues,omitempty"`

	// LandGrants counts the Noble's Soc increases: "Each increase in Soc
	// during CharGen awards a Land Grant" (chart 11). The hexes and their
	// economics land with muster out (docs/PRD.md milestone 4).
	LandGrants int `json:"land_grants,omitempty"`

	// Major and Minor are the Scholar's areas: "Every Scholar has a Major
	// and a Minor" (chart 02, p. 76). A Scholar arriving without a degree
	// selects both (interpretation I-23, ERRATA.md).
	Major string `json:"major,omitempty"`
	Minor string `json:"minor,omitempty"`

	// Publications counts the Scholar's Reward successes (chart 02); an
	// Award-Winning publication counts as two.
	Publications int `json:"publications,omitempty"`

	// Tenured records the chart 02 Tenure grant, which gates promotion
	// beyond Scholar3.
	Tenured bool `json:"tenured,omitempty"`

	// Specialty is the Entertainer's chosen art (chart 03 "Select A
	// Specialty", p. 77); Talent is the performance ability that career
	// tracks alongside Fame.
	Specialty string `json:"specialty,omitempty"`
	Talent    int    `json:"talent,omitempty"`

	// Fame is the reputation the Entertainer career tracks (chart 03).
	Fame int `json:"fame,omitempty"`

	// ShipShares counts the Merchant Reward awards: "Every Reward gives
	// the character Ship Shares, redeemable toward ownership of a ship
	// upon mustering out" (chart 06). The economics land with muster out
	// (docs/PRD.md milestone 4).
	ShipShares int `json:"ship_shares,omitempty"`

	Terms []TermRecord `json:"terms"`
}

// Award is one decoration and how many times it was earned, in the
// character card's notation (p. 65: "MCUF-1. XS-1. WB-1.").
type Award struct {
	Code  string `json:"code"`
	Name  string `json:"name,omitempty"`
	Count int    `json:"count"`
	Mod   int    `json:"mod,omitempty"`
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

// CharacteristicMax caps human characteristics: "Characteristics for
// Humans cannot exceed 15. If a benefit elevates a characteristic above
// 15, that benefit is lost" (p. 68).
const CharacteristicMax = 15

// awardSkill increases a skill by the given levels, capped at SkillMax and
// keeping Skills sorted by name (binary find-or-insert preserves the
// invariant without re-sorting). It returns the resulting level, then the
// levels actually applied.
func (c *Character) awardSkill(name string, levels int) (int, int) {
	i, found := slices.BinarySearchFunc(c.Skills, name, func(s Skill, target string) int {
		return strings.Compare(s.Name, target)
	})

	if found {
		applied := min(levels, SkillMax-c.Skills[i].Level)
		c.Skills[i].Level += applied

		return c.Skills[i].Level, applied
	}

	applied := min(levels, SkillMax)
	c.Skills = slices.Insert(c.Skills, i, Skill{Name: name, Level: applied})

	return applied, applied
}

// skillLevel reports the current level of a skill, 0 if not held.
func (c *Character) skillLevel(name string) int {
	i, found := slices.BinarySearchFunc(c.Skills, name, func(s Skill, target string) int {
		return strings.Compare(s.Name, target)
	})
	if !found {
		return 0
	}

	return c.Skills[i].Level
}

// Options configures one generation run.
type Options struct {
	Seed uint64
	Name string

	// Career forces the first career by name ("--career forces the first
	// career only", docs/PRD.md CLI sketch). Empty leaves the selection
	// to the Decider.
	Career string

	// Homeworld assigns the homeworld (docs/PRD.md FR2). Only the
	// all-zero struct falls back to the tool-owned default,
	// world.Default; a partially-populated homeworld is validated and
	// rejected, never silently repaired.
	Homeworld world.Homeworld

	// Decider resolves every choice point. Required: silently
	// substituting the default policy would misrepresent who decided —
	// auto callers pass DefaultPolicy{} explicitly.
	Decider Decider
}

// Generate runs the generation procedure and returns the character record:
// checklist steps A (Generate Characteristics), B (Determine A Homeworld),
// C (Education and Training), and D (Select Career) plus career resolution
// for the implemented careers (chart E1, p. 72); aging, career changes,
// muster out, and fame land with docs/PRD.md milestone 4.
func Generate(opts Options) (Character, error) {
	if opts.Decider == nil {
		return Character{}, errNoDecider
	}

	roller := dice.New(opts.Seed)

	var log Log

	// policy_version attests which decision table governed the run's
	// choices: the POLICY.md version only when the default policy itself
	// decided, "none" for any other Decider.
	policyVersion := "none"
	if _, isDefault := opts.Decider.(DefaultPolicy); isDefault {
		policyVersion = PolicyVersion
	}

	character := Character{
		SchemaVersion: SchemaVersion,
		Ruleset:       Ruleset,
		EngineVersion: EngineVersion,
		PolicyVersion: policyVersion,
		RNG:           RNG{Algorithm: RNGAlgorithm, Seed: opts.Seed},
		Name:          opts.Name,
		Age:           StartAge,
	}

	log.Step("Generate Characteristics", "Book 1 p. 72 chart E1 step A")

	character.Characteristics = RollCharacteristics(roller, &log)

	homeworld, err := homeworldOrDefault(opts.Homeworld)
	if err != nil {
		return Character{}, err
	}

	if err := runHomeworld(homeworld, &log, opts.Decider, &character); err != nil {
		return Character{}, err
	}

	if err := runEducation(roller, &log, opts.Decider, &character); err != nil {
		return Character{}, err
	}

	if err := runCareer(opts.Career, roller, &log, opts.Decider, &character); err != nil {
		return Character{}, err
	}

	character.UPP = character.Characteristics.UPP()
	character.Events = log.Events()

	return character, nil
}

// ErrUnknownCareer reports a forced career that is not implemented; the
// CLI matches it to distinguish usage errors from operational ones.
var ErrUnknownCareer = errors.New("unknown career")

// errNoDecider reports a Generate call without a Decider.
var errNoDecider = errors.New("chargen: Options.Decider is required (pass DefaultPolicy{} for auto mode)")

// errBadChoice reports a Decider answer outside the presented options, or
// a choice point with no options. No silent repair: the replay contract
// fails at the first divergence, and a clamped answer would mask it.
var errBadChoice = errors.New("invalid choice")

// runCareer performs checklist step D (Select Career, chart E1 p. 72) and
// resolves the selected career through the careerRegistry (careerrun.go);
// registered careers grow with docs/PRD.md milestone 3.
func runCareer(forced string, roller *dice.Roller, log *Log, decider Decider, character *Character) error {
	options := career.Available()

	if forced != "" {
		if !slices.Contains(options, forced) {
			return fmt.Errorf("%w: %q (available: %s)", ErrUnknownCareer, forced, strings.Join(options, ", "))
		}

		options = []string{forced}
	}

	log.Step("Select Career", "Book 1 p. 72 chart E1 step D")

	// A failed To Begin removes the career and offers the rest: "If both
	// Begin and Retry fail, this career may not be used." (p. 65) Running
	// out of options is a legal dead-end (no career), not an error.
	for len(options) > 0 {
		chosen, chosenSeq, err := choose(log, decider, Choice{
			ID:      ChooseCareer,
			Prompt:  "Select career",
			Options: options,
			Cite:    "Book 1 p. 72 chart E1 step D",
		})
		if err != nil {
			return err
		}

		began, err := runCareerByName(options[chosen], chosenSeq, roller, log, decider, character)
		if err != nil || began {
			return err
		}

		options = slices.Concat(options[:chosen], options[chosen+1:])
	}

	return nil
}

// choose puts a choice to the decider, validates the answer, and logs the
// resolved choice event. It returns the chosen index and the choice
// event's sequence number (for consequences caused by the choice).
func choose(log *Log, decider Decider, c Choice) (int, int, error) {
	if len(c.Options) == 0 {
		return 0, 0, fmt.Errorf("%w: %q presented no options", errBadChoice, c.ID)
	}

	chosen := decider.Choose(c)
	if chosen < 0 || chosen >= len(c.Options) {
		return 0, 0, fmt.Errorf("%w: %q answer %d outside 0-%d", errBadChoice, c.ID, chosen, len(c.Options)-1)
	}

	seq := log.Choice(ChoiceEvent{
		Decider: decider.Kind(),
		Prompt:  c.Prompt,
		Options: c.Options,
		Chosen:  chosen,
		Cite:    c.Cite,
	})

	return chosen, seq, nil
}
