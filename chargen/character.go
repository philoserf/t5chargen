package chargen

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/philoserf/t5chargen/benefit"
	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/dice"
	"github.com/philoserf/t5chargen/lifestage"
	"github.com/philoserf/t5chargen/skill"
	"github.com/philoserf/t5chargen/world"
)

// Provenance constants for the replay and provenance contract (docs/PRD.md):
// every character record carries them so old characters stay auditable after
// embedded tables change. Changing the RNG algorithm, the seeded stream's
// consumption order, or the default policy is a version bump; EngineVersion
// is hand-bumped in v1 (no build-info plumbing).
const (
	// SchemaVersion identifies the character JSON schema.
	SchemaVersion = "0.32.0"

	// Ruleset is pinned: all rule citations resolve against this artifact.
	Ruleset = "Traveller5 Core Rules Book 1, Print Edition 5.1"

	// EngineVersion identifies this implementation of the generation
	// procedure, including the seeded stream's consumption order.
	EngineVersion = "0.41.0"

	// PolicyVersion identifies the auto-mode decision table in POLICY.md
	// (docs/PRD.md, CLI sketch). Changing the policy is a version bump.
	PolicyVersion = "0.24.0"

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

// advanceYears elapses game time and records it, the single place a
// character ages. Time passes for two printed reasons: a term is four
// years ("the 4-year Term", p. 66) and a failed career entry costs one
// ("Each failed attempt (both Begin or Retry) takes one year", p. 65).
//
// One site rather than nine because aging hangs off it: characters reach
// Life Stage 5 at age 34 and are then "subject to Aging" every four years
// (chart A, p. 89), which is a rule about the passage of time and not
// about any of the particular events that pass it.
//
// cause is the throw or choice that consumed the time, never a step. FR10
// allows a step cause where no throw or choice produced the consequence
// (interpretation I-87); elapsed years are not such a case — something
// always consumes the time — so this site holds to the stricter rule. The
// Aging Checks the span crosses carry their own throws as causes.
func (c *Character) advanceYears(years int, roller *dice.Roller, log *Log, cause int) error {
	from := c.Age

	c.Age += years
	log.Consequence(ConsequenceEvent{Cause: cause, Kind: ConsequenceYearsElapsed, Value: years})

	// Aging is a rule about the passage of time, so it resolves here
	// rather than at any of the events that pass it: "Once Aging begins,
	// it occurs every four years on the character's birthday" (chart A
	// p. 89).
	return c.ageEffects(from, c.Age, roller, log)
}

// RNG records the random stream a character was generated from
// (docs/PRD.md, Replay and provenance contract).
type RNG struct {
	Algorithm string `json:"algorithm"`
	Seed      uint64 `json:"seed"`
}

// Inputs records the generation inputs the rest of the record does not
// already carry, so `t5chargen replay` can reconstruct a run from the file
// alone (docs/PRD.md, Replay and provenance contract: replay "re-runs the
// engine from the recorded seed and choice events"). The seed lives in RNG,
// and the name and homeworld are stored as character state; these two would
// otherwise be lost, and both change what the engine does.
type Inputs struct {
	// Career is the --career force ("--career forces the first career
	// only", docs/PRD.md CLI sketch). It is not merely a preference: a
	// force holds the first career's option list to one entry, so a
	// replay that did not know about it would present a different list
	// and read the recorded index against it.
	Career string `json:"career,omitempty"`

	// CurrentYear is the Imperial year generation ended in, which fixes
	// the birth year (p. 58). Recoverable as birth year plus age, but
	// that is reconstruction; the input is recorded as an input.
	CurrentYear int `json:"current_year"`

	// HomeworldAssigned records that the caller named the homeworld, so
	// step B had nothing to decide and offered it alone. Without it a
	// replay cannot tell a world that was assigned from one the character
	// chose off chart B: the record stores the world either way, and
	// handing it back as an assignment would offer one option against an
	// index that names one of thirty-four.
	HomeworldAssigned bool `json:"homeworld_assigned,omitempty"`

	// RolledHomeworld records that the homeworld was determined on chart
	// B rather than assigned (p. 56: "Select or determine a Homeworld").
	// The resulting world is stored like any other, but the two dice that
	// found it came out of the seeded stream, so a replay that did not
	// know to roll them would diverge from the next throw onward.
	RolledHomeworld bool `json:"rolled_homeworld,omitempty"`
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

	// Inputs are the generation inputs replay reconstructs the run from.
	Inputs Inputs `json:"inputs"`

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

	// Birthdate is the day the character was born, on the Imperial
	// Calendar, as p. 263 writes one: "Sigg's birthdate is Wonday
	// 044-1075." It is settled last (p. 58) and derives from the age, so
	// it schedules nothing: the Aging Checks fall on absolute ages
	// (interpretation I-50).
	Birthdate string `json:"birthdate,omitempty"`

	// LifeStage is the stage the character's age falls in (chart A p. 89),
	// stored derived like UPP so a record can be read without the table.
	LifeStage int `json:"life_stage"`

	// MajorIllnesses and ExtremelyMajorIllnesses count the Aging Check
	// passes that reduced two, and three or more, characteristics to zero
	// (chart A p. 89). The second extremely major illness is fatal.
	MajorIllnesses          int `json:"major_illnesses,omitempty"`
	ExtremelyMajorIllnesses int `json:"extremely_major_illnesses,omitempty"`

	// Credits is the character's money, in credits (docs/PRD.md, JSON
	// conventions). Muster out is what puts any there.
	Credits int `json:"credits,omitempty"`

	// Benefits are what muster out awarded, one entry per benefit
	// received (chart M1 p. 70; the career tables D).
	Benefits []BenefitRecord `json:"benefits,omitempty"`

	// Automatics are what a character already owns at muster out,
	// "Subject to Eligibility" (chart M1 p. 70).
	Automatics []string `json:"automatics,omitempty"`

	// ShipShares is every share the character holds, from both places
	// they come from: the Rewards a Merchant or Rogue accumulated during
	// a career, and the Benefits column cells muster out awarded. p. 69
	// and p. 80 describe the same thing — "a fractional ownership in a
	// starship", redeemable toward a ship — so they are one pool.
	//
	// Book 1 attaches no credit value to a share; chart S (p. 90) prices
	// ships in shares (interpretation I-84, ERRATA.md).
	ShipShares int `json:"ship_shares,omitempty"`

	// LandGrantIncome is the annual profit his Land Grants produce
	// (p. 88). It is deliberately not Credits: a grant is retained, not
	// sold — "Any character who has received a Land Grant retains it at
	// Mustering Out" (p. 68).
	LandGrantIncome int `json:"land_grant_income,omitempty"`

	// Entitlements are the annual payments muster out settles: the four
	// pensions from Life Stage 9, and the Armed Forces retirements from
	// muster out itself (chart M1 p. 70; p. 69).
	Entitlements []EntitlementRecord `json:"entitlements,omitempty"`

	// Fame is the character's Fame, computed over the finished record at
	// muster out rather than accumulated during it (chart F p. 91).
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
// Skill/Knowledge distinction lands with milestone 7 (docs/PRD.md), so a
// container skill is held whole: a character receiving Fighter five times
// leaves with Fighter-5, where p. 134 would give him Fighter-3 and Slug
// Thrower-2.
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

	// Masterpieces and PerfectMasterpieces count what a Craftsman
	// created: "A Perfect Masterpiece has 55 or more Master Points"
	// (chart 01 p. 75). Chart F multiplies them for Fame, and muster out
	// prices them.
	Masterpieces        int `json:"masterpieces,omitempty"`
	PerfectMasterpieces int `json:"perfect_masterpieces,omitempty"`

	// AssociatedCareer is the prior career a Functionary position belongs
	// to (chart 13 p. 87). Muster out adds this career's terms to that
	// career's benefit DM (p. 68).
	AssociatedCareer string `json:"associated_career,omitempty"`

	// EndAge is the character's age when the career ended, including a
	// career that never began (the year a failed To Begin cost, p. 65).
	// Muster out reads it: retirement and the Reserve Pension both run
	// from when the service stopped (p. 69).
	EndAge int `json:"end_age,omitempty"`

	// Disabled records that this career ended in a Disability Muster Out:
	// "When Risk Failure produces an Injury or Wound which reduces the
	// Controlling Characteristic by 4 or more points, the character is
	// Disabled ... Muster Out at Term End with Double Benefits" (p. 69).
	Disabled bool `json:"disabled,omitempty"`

	// Reserve records enrolment on leaving the Armed Forces: "A character
	// who leaves a military, naval, or marine career is automatically in
	// the Reserves until retirement at Life Stage 9" (p. 67), holding the
	// rank in Rank as a Reserve Rank.
	Reserve bool `json:"reserve,omitempty"`

	// SanityMod is the pending reduction the career's terms owe against
	// Sanity, recorded rather than applied: "reduce San= -1 for each TWO
	// Terms served" (chart 05 p. 79), but "Characters do not generate
	// Sanity until it is first called for" (p. 52). Negative or absent.
	// See interpretation I-47, ERRATA.md.
	SanityMod int `json:"sanity_mod,omitempty"`

	// Rank and RankTitle are the rank held on leaving the career, for the
	// careers with a rank table (chart 06, p. 80); empty for the careers
	// with no rank (p. 65).
	Rank      string `json:"rank,omitempty"`
	RankTitle string `json:"rank_title,omitempty"`

	// ControllingCharacteristic is the career-long pick of a career that
	// chooses one rather than rotating (chart 10, p. 84).
	ControllingCharacteristic string `json:"controlling_characteristic,omitempty"`

	// SuccessfulSchemes and FailedSchemes count what chart F prices at x2
	// and x3 (p. 91). Chart 10's Risk decides which: a caught Rogue's
	// scheme failed, whatever the Reward paid.
	SuccessfulSchemes int `json:"successful_schemes,omitempty"`
	FailedSchemes     int `json:"failed_schemes,omitempty"`

	// Scheme is the Rogue's current plan, PrisonYears a sentence owed at
	// the start of the next term, and SchemePayoff the credits his
	// schemes have paid (chart 10, p. 84). Muster out spends it.
	Scheme       string `json:"scheme,omitempty"`
	PrisonYears  int    `json:"prison_years,omitempty"`
	SchemePayoff int    `json:"scheme_payoff,omitempty"`

	// UndercoverCareer and UndercoverTitle are the Agent's current cover
	// identity (chart 09, p. 83); Commendations counts his Reward
	// successes.
	UndercoverCareer string `json:"undercover_career,omitempty"`
	UndercoverTitle  string `json:"undercover_title,omitempty"`
	Commendations    int    `json:"commendations,omitempty"`

	// CommendationAwards records each Commendation as chart F formats it:
	// "<Undercover Career> Commendation-N", where "N= C-R = the
	// Controlling Characteristic (without Mods) minus the Reward Die
	// Roll" (p. 91). Mod carries N. Chart F's Fame counts them rather
	// than summing N.
	CommendationAwards []Award `json:"commendation_awards,omitempty"`

	// Branch is the Armed Forces branch served in (chart 08, p. 82);
	// Medals are the decorations earned, in the card's own notation
	// ("MCUF-1. XS-1.", p. 65); ServiceBadges counts the Risk-success
	// Exemplary Service Badges, which carry no promotion modifier
	// (interpretation I-32, ERRATA.md).
	Branch        string  `json:"branch,omitempty"`
	Medals        []Award `json:"medals,omitempty"`
	ServiceBadges int     `json:"service_badges,omitempty"`

	// WoundBadges counts this career's Risk-failure wounds: "If the
	// Soldier, Spacer, or Marines Risk Roll fails, the character is
	// wounded and receives a Wound Badge (WB)" (p. 91). The character
	// carries the lifetime total across every career; chart F prices the
	// Armed Forces career's own, at x1 per badge, so the count has to be
	// attributable to the career that took them.
	WoundBadges int `json:"wound_badges,omitempty"`

	// Exiled, TimesExiled, and SuccessfulIntrigues are the Noble's state
	// (chart 11, p. 85): "Exile is a banishment to the edges of the empire
	// orchestrated by political enemies." The two counters modify the
	// Return and Intrigue rolls.
	Exiled              bool `json:"exiled,omitempty"`
	TimesExiled         int  `json:"times_exiled,omitempty"`
	SuccessfulIntrigues int  `json:"successful_intrigues,omitempty"`

	// LandGrants counts the career's Land Grants: for a Noble, his Soc
	// increases — "Each increase in Soc during CharGen awards a Land
	// Grant" (chart 11) — and for a Scout, his Discoveries (chart 05).
	// What they earn is priced at muster out, by career, since the two
	// careers' hexes sit on different worlds (chart M1; pp. 41, 79, 88).
	LandGrants int `json:"land_grants,omitempty"`

	// Major and Minor are the Scholar's areas: "Every Scholar has a Major
	// and a Minor" (chart 02, p. 76). A Scholar arriving without a degree
	// selects both (interpretation I-23, ERRATA.md).
	Major string `json:"major,omitempty"`
	Minor string `json:"minor,omitempty"`

	// Publications counts the Scholar's Reward successes (chart 02); an
	// Award-Winning publication counts as two.
	Publications int `json:"publications,omitempty"`

	// AwardWinningPublications counts only those, which Publications
	// cannot be read back from (I-25). Chart M1's TAS Life Membership is
	// the "Award-Winning Scholar's" (p. 67).
	AwardWinningPublications int `json:"award_winning_publications,omitempty"`

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
	// upon mustering out" (chart 06). A share is priced in tons of hull
	// rather than credits, Book 1 attaching no figure to one
	// (interpretation I-84); redeeming them for a ship is an act of play
	// (p. 90).
	ShipShares int `json:"ship_shares,omitempty"`

	Terms []TermRecord `json:"terms"`
}

// EntitlementRecord is one annual payment a character retires on.
type EntitlementRecord struct {
	ID   string `json:"id"`
	Name string `json:"name"`

	// AnnualCredits is what it pays each year, after any Pension x2 or
	// Retirement x2 doubling: "Each doubling is of the original" (p. 68).
	AnnualCredits int `json:"annual_credits"`

	// FromAge is when payment begins. A pension "begins when a character
	// reaches Life Stage 9 Retirement (= age 66 for Humans)"; the Armed
	// Forces retirements begin at muster out (p. 69).
	FromAge int `json:"from_age"`

	// CashOut is the lump sum instead: "Any Entitlement can be cashed
	// out for a lump sum equal to five years of payments" (p. 69).
	CashOut int `json:"cash_out"`
}

// BenefitRecord is one muster-out award (chart M1 p. 70).
type BenefitRecord struct {
	Kind benefit.Kind `json:"kind"`
	Name string       `json:"name"`

	// Career is the table it came from: "Use the Mustering Out Table
	// corresponding to the Career for the time spent in that career"
	// (p. 68).
	Career string `json:"career"`

	// Count is how many were received; Credits what it paid, where it
	// paid anything. Detail names a characteristic a "Str +1" raised.
	Count   int    `json:"count,omitempty"`
	Credits int    `json:"credits,omitempty"`
	Detail  string `json:"detail,omitempty"`
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
// Skill-15" (p. 134). The Knowledge-6 cap the same line prints is a v1
// non-goal with the rest of the Skill/Knowledge distinction
// (docs/PRD.md): nothing here holds a Knowledge to cap.
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

	// CurrentYear is the Imperial year generation ends in, which fixes
	// the birth year: "The Referee provides the current date of the
	// beginning of adventuring" (p. 58). Zero takes the page's own
	// default, calendar.DefaultYear.
	CurrentYear int

	// RollHomeworld determines the homeworld on chart B's world list
	// instead of taking the assigned one: "Homeworld. Select or determine
	// a Homeworld" (p. 56). It overrides Homeworld when set.
	RollHomeworld bool

	// Decider resolves every choice point. Required: silently
	// substituting the default policy would misrepresent who decided —
	// auto callers pass DefaultPolicy{} explicitly.
	Decider Decider
}

// newCharacter stamps the provenance and the inputs a replay rebuilds the
// run from, before anything is rolled.
func newCharacter(opts Options, policyVersion string, homeworldAssigned bool) Character {
	return Character{
		SchemaVersion: SchemaVersion,
		Ruleset:       Ruleset,
		EngineVersion: EngineVersion,
		PolicyVersion: policyVersion,
		RNG:           RNG{Algorithm: RNGAlgorithm, Seed: opts.Seed},
		Inputs: Inputs{
			Career:            opts.Career,
			CurrentYear:       currentYearOrDefault(opts.CurrentYear),
			HomeworldAssigned: homeworldAssigned,
			RolledHomeworld:   opts.RollHomeworld,
		},
		Name: opts.Name,
		Age:  StartAge,
	}
}

// newLog starts the record, showing it to a Decider that also watches.
func newLog(decider Decider) Log {
	var log Log

	if watcher, ok := decider.(Watcher); ok {
		log.watch = watcher.Watch
	}

	return log
}

// Generate runs the generation procedure and returns the character record:
// checklist steps A (Generate Characteristics), B (Determine A Homeworld),
// C (Education and Training), and D (Select Career) plus career
// resolution (chart E1, p. 72), with the Aging Checks the lifepath
// crosses (chart A, p. 89), the career changes p. 66 allows, muster out
// and Fame.
func Generate(opts Options) (Character, error) {
	if opts.Decider == nil {
		return Character{}, errNoDecider
	}

	if err := checkSharedData(); err != nil {
		return Character{}, err
	}

	roller := dice.New(opts.Seed)

	log := newLog(opts.Decider)

	// policy_version attests which decision table governed the run's
	// choices: the POLICY.md version only when the default policy itself
	// decided, "none" for any other Decider.
	policyVersion := "none"
	if _, isDefault := opts.Decider.(DefaultPolicy); isDefault {
		policyVersion = PolicyVersion
	}

	// A homeworld the caller named is assigned; one it did not is the
	// character's to select from chart B (p. 58).
	assigned := supplied(opts.Homeworld)

	character := newCharacter(opts, policyVersion, assigned)

	log.Step("Generate Characteristics", "Book 1 p. 72 chart E1 step A")

	character.Characteristics = RollCharacteristics(roller, &log)

	homeworld, err := homeworldOrDefault(opts.Homeworld)
	if err != nil {
		return Character{}, err
	}

	if err := runHomeworld(homeworld, assigned, opts.RollHomeworld, roller, &log, opts.Decider, &character); err != nil {
		return Character{}, err
	}

	if err := runEducation(roller, &log, opts.Decider, &character); err != nil {
		return Character{}, err
	}

	if err := runCareer(opts.Career, roller, &log, opts.Decider, &character); err != nil {
		return Character{}, err
	}

	// Fame is calculated over the finished record, not accumulated
	// (chart F p. 91), and muster out reads it — "one additional roll if
	// Fame 19+" (p. 68).
	if err := afterCareers(&character, roller, &log, opts.Decider, opts.CurrentYear); err != nil {
		return Character{}, err
	}

	return character, nil
}

// afterCareers runs checklist step E and what follows it: Fame, which
// muster out reads (p. 68), then muster out itself (p. 67).
func afterCareers(c *Character, roller *dice.Roller, log *Log, decider Decider, currentYear int) error {
	log.Step("Determine Fame", "Book 1 p. 91 chart F")

	if err := computeFame(c, roller, log, decider); err != nil {
		return err
	}

	// "Mustering Out counts up the character's belongings ... and notes
	// them as assets for the adventuring situations to come" (p. 67). A
	// dead character has none, and p. 69 is blunter: "the Character is
	// dead (and all efforts in this particular character creation
	// process are lost)". The record is kept — that much of the sentence
	// the engine declines, I-51 — but nothing is added to it, and the
	// step is not opened for a section that would hold nothing
	// (interpretation I-77).
	if !c.Dead {
		log.Step("Muster Out", "Book 1 p. 67; chart M1 p. 70")

		if err := musterOut(c, roller, log, decider); err != nil {
			return err
		}
	}

	// Last, because p. 58 puts it last and because it reads the age that
	// muster out is the final chance to change.
	if err := c.birthdate(roller, log, currentYear); err != nil {
		return err
	}

	return c.finalize(log)
}

// finalize fills the fields derived from the finished record — the UPP
// (p. 48) and the Life Stage the character's age falls in (chart A p. 89)
// — and freezes the event log.
func (c *Character) finalize(log *Log) error {
	c.UPP = c.Characteristics.UPP()

	stages, err := lifestage.Load()
	if err != nil {
		return fmt.Errorf("life stages: %w", err)
	}

	c.LifeStage = stages.Of(c.Age)
	c.Events = log.Events()

	return nil
}

// checkSharedData loads the two registries that several open-selection
// cells read through helpers with no error return, so a fault in either is
// reported once, before any dice are rolled.
//
// Without this, a broken Citizen transcription surfaces at whichever cell
// first wants Citizen Life Skills, as "not implemented until education/
// skill milestones" — sending the reader to COVERAGE.md for a deferral that
// does not exist instead of to the data file that is broken. The same
// lesson is recorded for careers that will not load: "A definition that
// will not load is a build fault, not an ineligible career".
func checkSharedData() error {
	if _, err := career.Citizen(); err != nil {
		return fmt.Errorf("citizen life skills: %w", err)
	}

	if err := skill.Err(); err != nil {
		return fmt.Errorf("master skill list: %w", err)
	}

	return nil
}

// ErrCareerUnavailable reports a forced career the character cannot
// enter: chart 01's entry is automatic only "if TWO skill-6 and
// Craftsman-1" (p. 75), and chart 13's "is never a first career" (p. 87).
// The career exists; this character may not open a lifepath with it.
var ErrCareerUnavailable = errors.New("career unavailable to this character")

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
// resolves the selected career through the careerRegistry (careerrun.go),
// which holds all thirteen.
func runCareer(forced string, roller *dice.Roller, log *Log, decider Decider, character *Character) error {
	options, err := firstCareerOptions(forced, character)
	if err != nil {
		return err
	}

	// The outer loop runs once per career served: a character who
	// voluntarily leaves one selects another (p. 66). The inner loop is
	// the To Begin failure path — "If both Begin and Retry fail, this
	// career may not be used" (p. 65) — where running out of options is a
	// legal dead-end (no career), not an error.
	for {
		began, end, err := runCareerOnce(options, roller, log, decider, character)
		if err != nil {
			return err
		}

		// "the Character is dead (and all efforts in this particular
		// character creation process are lost)" (p. 69). The engine keeps
		// the record rather than discarding the run, but a dead character
		// does not go on to try another career.
		if !began || end != termCareerChanged || character.Dead {
			return nil
		}

		options, err = eligibleForChange(character, character.Careers[len(character.Careers)-1].Career)
		if err != nil {
			return err
		}

		if len(options) == 0 {
			return nil
		}
	}
}

// firstCareerOptions is what the lifepath may open with: the eligible
// careers, narrowed by a --career force and by a Service Academy
// commission.
//
// The commission is applied after the force, not before, so a conflict
// between the two is reported as the commission it breaks rather than as a
// career the lifepath could not have opened with. Neither silently wins: a
// character who owes the Navy a term and was told to start as a Soldier is
// a contradiction, and generation says so.
func firstCareerOptions(forced string, character *Character) ([]string, error) {
	// The lifepath opens on the careers this character is eligible to
	// start: "Functionary is never a first career" (chart 13 p. 87), and
	// chart 01's entry is automatic only "if TWO skill-6 and
	// Craftsman-1" (p. 75), which a character leaving education never has.
	options, err := eligibleCareers(character, "", true)
	if err != nil {
		return nil, err
	}

	options, err = forcedCareer(forced, options)
	if err != nil {
		return nil, err
	}

	return commissionedCareer(character, options)
}

// commissionedCareer narrows the first career to the service a Service
// Academy graduate was commissioned into: "they provide graduates an Army
// or Navy Commission ... The character is required to serve one term in
// the service" (p. 62, interpretation I-99).
//
// Required, so the obligation is the option list rather than a suggestion
// on it. It is put through the choice funnel all the same, exactly as
// --career is, because the record has to show what was decided even where
// only one thing could be.
//
// The first career only. "At the end of that term, the character may try
// to continue, or may attempt any other career available" (p. 62) — one
// term is what is owed, and the career-change loop past this point is
// untouched.
func commissionedCareer(character *Character, options []string) ([]string, error) {
	service := character.academyCommission()
	if service == "" {
		return options, nil
	}

	owed, err := career.ForService(service)
	if err != nil {
		return nil, fmt.Errorf("the %s commission: %w", service, err)
	}

	if !slices.Contains(options, owed) {
		return nil, fmt.Errorf("%w: a %s commission owes a term as %s, which is not open to him (available: %s)",
			ErrCareerUnavailable, service, owed, strings.Join(options, ", "))
	}

	return []string{owed}, nil
}

// forcedCareer narrows the options to a career named on the command line,
// which may be one the lifepath could not have opened with on its own.
func forcedCareer(forced string, options []string) ([]string, error) {
	if forced == "" {
		return options, nil
	}

	if !slices.Contains(career.Available(), forced) {
		return nil, fmt.Errorf("%w: %q (available: %s)",
			ErrUnknownCareer, forced, strings.Join(career.Available(), ", "))
	}

	// A career the lifepath could not have opened with is refused rather
	// than entered: chart 01's automatic entry is conditional (p. 75) and
	// chart 13's career "is never a first career" (p. 87). Forcing one is
	// a request the rules deny, not an engine fault.
	if !slices.Contains(options, forced) {
		return nil, fmt.Errorf("%w: %q cannot be a first career (available: %s)",
			ErrCareerUnavailable, forced, strings.Join(options, ", "))
	}

	return []string{forced}, nil
}

// runCareerOnce selects a career and resolves it, retrying with the
// remaining options while To Begin fails.
//
// The step header belongs here rather than to the caller: a character who
// changes careers selects again, and a selection logged outside a step of
// its own reads as part of the term he just left.
func runCareerOnce(
	options []string, roller *dice.Roller, log *Log, decider Decider, character *Character,
) (bool, termEnd, error) {
	log.Step("Select Career", "Book 1 p. 72 chart E1 step D")

	for len(options) > 0 {
		chosen, chosenSeq, err := choose(log, decider, Choice{
			ID:         ChooseCareer,
			Prompt:     "Select career",
			Options:    options,
			Scores:     automaticEntries(options, character),
			ScoreLabel: ScoreAutomatic,
			Cite:       "Book 1 p. 72 chart E1 step D",
		})
		if err != nil {
			return false, termCareerEnded, err
		}

		began, end, err := runCareerByName(options[chosen], chosenSeq, roller, log, decider, character)
		if err != nil || began {
			return began, end, err
		}

		if character.Dead {
			return false, termDied, nil
		}

		options = slices.Concat(options[:chosen], options[chosen+1:])
	}

	return false, termCareerEnded, nil
}

// choose puts a choice to the decider, validates the answer, and logs the
// resolved choice event. It returns the chosen index and the choice
// event's sequence number (for consequences caused by the choice).
//
// A decider that refuses ends generation with its own error, named with
// the choice point it declined; that is how an abandoned interactive
// session and a replay divergence reach the caller intact, rather than
// arriving as the out-of-range answer of a decider that did reply.
func choose(log *Log, decider Decider, c Choice) (int, int, error) {
	if len(c.Options) == 0 {
		return 0, 0, fmt.Errorf("%w: %q presented no options", errBadChoice, c.ID)
	}

	chosen, err := decider.Choose(c)
	if err != nil {
		return 0, 0, fmt.Errorf("%q: %w", c.ID, err)
	}

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

// The two score labels that are a yes or a no rather than a quantity. A
// front end reads them to know how to put the score to a player; the
// engine and the policy read only the digit (see Choice).
const (
	// ScoreAutomatic marks a career entered without a To Begin throw.
	ScoreAutomatic = "automatic"

	// ScoreQualifies marks an education programme whose prerequisite the
	// character meets; a zero costs him a waiver attempt, not the row.
	ScoreQualifies = "qualifies"
)

// automaticEntries reports, for each career on offer, whether entry needs
// no throw.
//
// It is decision data rather than a rule — not recorded, and a front end
// may show it (see Choice) — but it is the thing chart E1's step D asks a
// player to weigh. A career that cannot fail costs him nothing; every
// other one risks the year p. 65 charges for a failed attempt.
func automaticEntries(options []string, character *Character) []int {
	scores := make([]int, len(options))

	for i, name := range options {
		def, err := career.ByName(name)
		if err != nil {
			continue // an unknown career scores nothing; selection will report it
		}

		scores[i] = boolScore(entersAutomatically(def, character))
	}

	return scores
}

// entersAutomatically applies a career's begin_automatic_if.
//
// Chart 01's Craftsman and chart 11's Noble print "Automatic*" against a
// prerequisite, and that prerequisite is checked before the career is
// offered (I-28) — so where they appear on the list at all, they are
// automatic, which is what "always" records for them.
func entersAutomatically(def *career.Definition, character *Character) bool {
	auto := def.BeginAutomaticIf
	if auto == nil {
		return false
	}

	if auto.Always {
		return true
	}

	value, ok := characteristicValue(&character.Characteristics, auto.Characteristic)

	return ok && value >= auto.Minimum
}
