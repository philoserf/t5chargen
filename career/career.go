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

	"github.com/philoserf/t5chargen/benefit"
	"github.com/philoserf/t5chargen/skill"
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
	// p. 79); EntrySoldier is the "Soldier Skill" cell (chart 11 table C,
	// p. 85). Both select from a Master Skill List group.
	EntryStarship EntryKind = "starship"
	EntrySoldier  EntryKind = "soldier"

	// EntryAnySkill is chart 13's "Any Skill***" cell: "from Citizen Life
	// Skills and Knowledges" (p. 87). EntryAnyKnowledge is chart 09's
	// "Any Knowledge" cell (p. 83).
	EntryAnySkill     EntryKind = "any_skill"
	EntryAnyKnowledge EntryKind = "any_knowledge"

	// EntryCapital is chart 11's "Capital***" cell: "World Knowledge (of
	// world of highest held noble Land Grant)" (p. 85), which needs the
	// Land Grant worlds that land with muster out (docs/PRD.md
	// milestone 4).
	EntryCapital EntryKind = "capital"

	// EntryNewTrade is chart 01's "New Trade***" cell: "Any Trade not
	// already held; if all are already held; this benefit is lost"
	// (p. 75).
	EntryNewTrade EntryKind = "new_trade"

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
	// Citizen, chart 04's "Citizen Life C1 C2 C3 C4"). Empty where the
	// variant rotates none (chart 03's "Risk & Reward Talent").
	ControllingCharacteristics []string `json:"controlling_characteristics,omitempty"`

	// The Continue target takes one of three forms, exactly one of which
	// is set: a fixed roll-low target (chart 04: "Continue 10-"), a
	// characteristic (chart 05: "Continue Int"), or the career's own
	// tracked value (chart 03: "Continue Fame"). The third consolidates
	// into a kind enum when the other value-target careers land.
	ContinueTarget         int    `json:"continue_target,omitempty"`
	ContinueCharacteristic string `json:"continue_characteristic,omitempty"`
	ContinueFame           bool   `json:"continue_fame,omitempty"`

	// ContinueCC targets the career's own controlling characteristic
	// (chart 10: "Continue CC*"), which a Rogue chooses once and keeps.
	ContinueCC bool `json:"continue_cc,omitempty"`

	// ContinueMod adds a career-tracked value to the Continue target
	// (chart 02: "Continue Edu*" with "*Mod +Pubs"). Unlike ContinueTarget
	// it is not bounded to 2-11 (see validateContinue): the printed rule
	// carries the target past 12 routinely, where the only remaining exit
	// is the p. 134 automatic failure on a natural 12 (dice.resolveCheck),
	// so terms run long until aging lands (docs/PRD.md milestone 4).
	ContinueMod ContinueModKind `json:"continue_mod,omitempty"`

	// ContinueWaiver offers a Waiver on a failed Continue throw — the
	// sixth event chart 02's Waivers box lists ("An adverse die roll or
	// decision (in Position, Promotion, Research, Publication, Tenure, or
	// Continue) may be waived", p. 76). The other five belong to the
	// career's own mechanics; Continue belongs to the generic runner.
	ContinueWaiver bool `json:"continue_waiver,omitempty"`

	// CCFixed marks a career whose controlling characteristic is chosen
	// once rather than rotated: "A Rogue selects one Controlling
	// Characteristic ... which is then used throughout his career (not
	// just in the current Term)" (chart 10, p. 84).
	CCFixed bool `json:"cc_fixed,omitempty"`

	// Schemes is chart 10's Rogue Schemes table (p. 84); nil elsewhere.
	Schemes *Schemes `json:"schemes,omitempty"`

	// MajorOrMinorColumn offers the character's Major and Minor alongside
	// the skills-table columns: "A Scholar may always take a skill in his
	// Major or Minor instead of from this table" (chart 02 table C).
	MajorOrMinorColumn bool `json:"major_or_minor_column,omitempty"`

	// SkillsPerTerm is the table C eligibility (chart 04 table B:
	// "Per Term: 4 on Table C"). SkillEligibility carries per-duty
	// counts where the chart splits them (chart 05 table B: "If Courier
	// Duty 4 / If Explorer Duty 8"); mechanics consult it by duty label.
	SkillsPerTerm    int            `json:"skills_per_term"`
	SkillEligibility map[string]int `json:"skill_eligibility,omitempty"`

	// SkillsPerAdvancement is the extra table C eligibility each rank
	// gained in the term earns (chart 06 table B: "Promotion 1").
	SkillsPerAdvancement int `json:"skills_per_advancement,omitempty"`

	// MusterOut is the career's table D, and MusterOutM2 the same table
	// as chart M2 reprints it on p. 71 — present only for the six careers
	// where the two disagree. The career page governs (interpretation
	// I-71); M2 is transcribed so the disagreement is visible and pinned.
	MusterOut   *MusterOut `json:"muster_out,omitempty"`
	MusterOutM2 *MusterOut `json:"muster_out_m2,omitempty"`

	// Masterpiece is chart 01's Creating A Masterpiece box (p. 75); nil
	// for every other career.
	Masterpiece *Masterpiece `json:"masterpiece,omitempty"`

	// BeginPrerequisite is chart 01's "To Begin Automatic* — *if TWO
	// skill-6 and Craftsman-1" (p. 75): entry is automatic, but only for
	// a character who already qualifies. nil for every other career.
	BeginPrerequisite *Prerequisite `json:"begin_prerequisite,omitempty"`

	// ContinueSkill and ContinueSkillMultiplier are chart 01's "Continue
	// Craftsman x2": the target is a skill level times a multiplier.
	ContinueSkill           string `json:"continue_skill,omitempty"`
	ContinueSkillMultiplier int    `json:"continue_skill_multiplier,omitempty"`

	// NotAFirstCareer marks a career unreachable at the start of the
	// lifepath: "Functionary is never a first career" (chart 13 p. 87),
	// and p. 63's random-selection note that "Craftsman (1) and
	// Functionary (13) are unavailable as initial careers".
	NotAFirstCareer bool `json:"not_a_first_career,omitempty"`

	// BeginTotalTermsMultiplier is chart 13's "To Begin Total Terms x3":
	// the target is the character's terms in every prior career, times
	// this. A first career therefore has a target of zero, which is the
	// same rule as NotAFirstCareer said twice.
	BeginTotalTermsMultiplier int `json:"begin_total_terms_multiplier,omitempty"`

	// ContinueOfficePolitics marks chart 13's "Continue Office Politics":
	// the career has no Continue throw of its own, the Office Politics
	// Risk result deciding it ("Risk Failure: Functionary career ends.
	// The character may not Continue").
	ContinueOfficePolitics bool `json:"continue_office_politics,omitempty"`

	// DirectorTitles renames rank F6 by the career the Functionary
	// position is associated with: "Scholar F6 =College President"
	// (chart 13 p. 87).
	DirectorTitles map[string]string `json:"director_titles,omitempty"`

	// Reserves marks the careers p. 67 enrols a leaver from: "A character
	// who leaves a military, naval, or marine career is automatically in
	// the Reserves." Chart facts, so the three Armed Forces careers carry
	// it rather than the engine naming them.
	Reserves bool `json:"reserves,omitempty"`

	// NoCareerChange marks a career a character cannot leave for another:
	// "A Functionary or Noble cannot change to a new career" (p. 66).
	NoCareerChange bool `json:"no_career_change,omitempty"`

	// NoEntryByChange marks a career a character cannot change into: "A
	// Character may not change to the Citizen career" (p. 66).
	NoEntryByChange bool `json:"no_entry_by_change,omitempty"`

	// SanityPerTerms is the number of terms in the career that cost one
	// point of Sanity: "Because of the long-term isolation that a Scout
	// must endure, reduce San= -1 for each TWO Terms served" (chart 05
	// p. 79). Only chart 05 prints such a rule. The reduction is recorded
	// as a modifier rather than applied, because Sanity is not generated
	// during character generation (p. 52; interpretation I-47).
	SanityPerTerms int `json:"sanity_per_terms,omitempty"`

	// BeginTracks are the alternative entry paths where a chart offers
	// several (chart 06: "To Begin 4th Officer Int / To Begin Spacehand
	// Dex / To Begin Temp Auto"). Careers with a single To Begin use
	// BeginChecks instead.
	BeginTracks []BeginTrack `json:"begin_tracks,omitempty"`

	// Ranks is the career's rank table in ascending order within each
	// class (chart 06 Table Of Merchant Ranks); empty for the careers
	// with no rank (p. 65).
	Ranks []Rank `json:"ranks,omitempty"`

	// Advancements are the commission and promotion rows of box A, in the
	// order a character attempts them.
	Advancements []Advancement `json:"advancements,omitempty"`

	// Undercover is chart 09's Agent Undercover Assignment table (p. 83),
	// which sends the Agent into another career for two years and lets him
	// "Select (not Roll) one skill from the skill tables of that Career";
	// nil for every other career.
	Undercover *Undercover `json:"undercover,omitempty"`

	// ArmedForces carries the Branch and Operations tables the Spacer,
	// Soldier, and Marine careers share ("The Armed Forces are Spacers,
	// Soldiers, and Marines, with background information as Branch and
	// Assignment", p. 66); nil for every other career.
	ArmedForces *ArmedForces `json:"armed_forces,omitempty"`

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

// BeginTrack is one entry path of a career that offers several, each
// landing at its own starting rank (chart 06, p. 80).
type BeginTrack struct {
	Name string `json:"name"`

	// Checks lists the characteristics the To Begin throw may check; the
	// character picks one where a chart offers several (chart 03: "Begin
	// Actor C2 or C3"). Empty is the chart's "Auto" (chart 06: "To Begin
	// Temp Auto").
	Checks []string `json:"checks,omitempty"`

	// Rank is the rank id entry confers.
	Rank string `json:"rank,omitempty"`
}

// MusterOutCell is one cell of a career's table D: a benefit from the
// chart M1 vocabulary, with whatever the cell adds to it.
type MusterOutCell struct {
	Kind benefit.Kind `json:"kind"`

	// Detail names the characteristic a "Str +1" or "C5 +1" cell raises.
	Detail string `json:"detail,omitempty"`

	// Credits is what a money cell pays outright; Count is how many of a
	// countable benefit the cell gives ("Ship Share", "Proxy (3)").
	Credits int `json:"credits,omitempty"`
	Count   int `json:"count,omitempty"`

	// Dice rolls the count instead, for chart 11's "Proxy (2D)".
	Dice int `json:"dice,omitempty"`

	// Printed keeps a cell's wording where it differs from the
	// vocabulary's name — chart 12 row 9 says "Directorate" where every
	// other chart says "Directorship" (interpretation I-70).
	Printed string `json:"printed,omitempty"`
}

// MusterOutDM is a table D column's DM line: "+Terms", "+ Officer Rank",
// "+Fame/3". "The DMs on the Benefits Tables are optional (for Terms,
// Fame, or other values). They may be partially applied" (p. 68).
type MusterOutDM struct {
	Kind string `json:"kind"`

	// Divisor is the "/3" of "+Fame/3".
	Divisor int `json:"divisor,omitempty"`
}

// MusterOutAutomatic is an award a career's own table D grants outright,
// beyond chart M1's list: chart 13 prints two, "Automatic: Gold Watch
// (Value= 100 x Terms as Functionary)" and "Automatic: Directorship if
// Rank F6+" (p. 87).
type MusterOutAutomatic struct {
	ID   string `json:"id"`
	Name string `json:"name"`

	// CreditsPerTerm prices the Gold Watch by the terms served.
	CreditsPerTerm int `json:"credits_per_term,omitempty"`

	// MinimumRank gates the award on the rank ladder.
	MinimumRank string `json:"minimum_rank,omitempty"`

	Note string `json:"note,omitempty"`
}

// MusterOutRow is one row of a table D.
type MusterOutRow struct {
	Roll    int            `json:"roll"`
	Money   MusterOutCell  `json:"money"`
	Benefit MusterOutCell  `json:"benefit"`
	Power   *MusterOutCell `json:"power,omitempty"`
}

// MusterOut is a career's table D. "Use the Mustering Out Table
// corresponding to the Career for the time spent in that career" (p. 68).
type MusterOut struct {
	Cite string `json:"cite"`

	// Label is the box header as printed, which is not uniform: chart 01
	// heads it "D CRAFTSMAN" and chart 03 "D MUSTER OUT BENEFITS", where
	// the other eleven say "D MUSTER OUT".
	Label string `json:"label"`

	Rows      []MusterOutRow `json:"rows"`
	MoneyDM   MusterOutDM    `json:"money_dm"`
	BenefitDM MusterOutDM    `json:"benefit_dm"`
	PowerDM   *MusterOutDM   `json:"power_dm,omitempty"`

	// Automatics are awards the career's own box grants outright, beyond
	// chart M1's list.
	Automatics []MusterOutAutomatic `json:"automatics,omitempty"`

	// Note is the box's own footnote, where it has one.
	Note string `json:"note,omitempty"`
}

// Prerequisite is a condition a character must already meet to enter a
// career: chart 01's "*if TWO skill-6 and Craftsman-1" (p. 75).
type Prerequisite struct {
	// Skill and SkillLevel are the named skill the character must hold
	// ("Craftsman-1").
	Skill      string `json:"skill"`
	SkillLevel int    `json:"skill_level"`

	// SkillsAtLevel and SkillsAtLevelCount are the breadth requirement
	// ("TWO skill-6"): how many skills, at what level.
	SkillsAtLevel      int `json:"skills_at_level"`
	SkillsAtLevelCount int `json:"skills_at_level_count"`
}

// QREBS is one of the five qualities a Masterpiece carries: "Q R E B S",
// Quality on 1 to 10 and the rest on -5 to +5 (chart 01 p. 75).
type QREBS struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Minimum int    `json:"minimum"`
	Maximum int    `json:"maximum"`
}

// Masterpiece is chart 01's Creating A Masterpiece box (p. 75).
type Masterpiece struct {
	Cite string `json:"cite"`

	// Dice and MinimumPoints are the creation throw and its floor: "Roll
	// 9D for Master Points or less for success in creation. If the
	// Craftsman cannot show at least 40 Master Points, he cannot attempt
	// a Masterpiece (treat as Failure)."
	Dice          int `json:"dice"`
	MinimumPoints int `json:"minimum_points"`

	// PerfectPoints: "A Perfect Masterpiece has 55 or more Master Points."
	PerfectPoints int `json:"perfect_points"`

	// BonusSkillLevel, MaxBonusSkills, and ExcludedSkill are the Master
	// Points skills contribute: "Up to FIVE Skills at level 6+ (or
	// Knowledges at level-6) (but not languages)".
	BonusSkillLevel int    `json:"bonus_skill_level"`
	MaxBonusSkills  int    `json:"max_bonus_skills"`
	ExcludedSkill   string `json:"excluded_skill"`

	// SkillsPerSuccess and SkillsPerFailure are table B's "Per Success 3
	// +Craftsman-1 / Per Failure 1 +Craftsman-1".
	SkillsPerSuccess int `json:"skills_per_success"`
	SkillsPerFailure int `json:"skills_per_failure"`

	// BaseValue, ValuePerPoint, and PerfectMultiplier price the result:
	// "The Masterpiece can be sold at Cr150,000 plus Cr10,000 per Master
	// Point over 40. A Perfect Masterpiece (=55 points or more) sells for
	// Double". Spending it lands with muster out.
	BaseValue         int `json:"base_value"`
	ValuePerPoint     int `json:"value_per_point"`
	PerfectMultiplier int `json:"perfect_multiplier"`

	// QREBS are the five qualities the Master Points are allocated to.
	// The allocation itself is deferred (interpretation I-62).
	QREBS []QREBS `json:"qrebs,omitempty"`
}

// Rank is one row of a career's rank table.
type Rank struct {
	ID    string `json:"id"`
	Class string `json:"class"`
	Title string `json:"title"`

	// AutoSkill is the rank's "Auto Skill" column entry (chart 06 table B:
	// "Automatic Skills by Rank"); empty for ranks with none.
	AutoSkill string `json:"auto_skill,omitempty"`

	// Soc is the Social Standing a rank carries, where the ladder is keyed
	// to it ("Nobles begin with rank equal to their Social Standing",
	// p. 65). Consecutive rows may share one value: "A character elevated
	// to Soc = c (lower case) is initially a Baronet. The next increase in
	// Soc remains C (now upper case) but the title increases to Baron"
	// (p. 51), which is why chart 11 elevates to "the next higher Noble
	// rank and its increase in Social Standing (if any)".
	Soc int `json:"soc,omitempty"`
}

// ContinueModKind names a career-tracked value added to the Continue
// target.
type ContinueModKind string

// The Continue modifiers.
const (
	// ContinueModPublications is chart 02's "*Mod +Pubs".
	ContinueModPublications ContinueModKind = "publications"

	// ContinueModTerms is chart 09's "*Mod +Terms" (p. 83), counted as
	// completed terms (interpretation I-12).
	ContinueModTerms ContinueModKind = "terms"
)

// TargetKind discriminates how an advancement's throw target is derived.
type TargetKind string

// The advancement target forms.
const (
	// TargetCharacteristic checks a characteristic (chart 06: "Officer
	// Commission Int").
	TargetCharacteristic TargetKind = "characteristic"

	// TargetTermsTimesTwo is twice the terms served (chart 06: "Officer
	// Promotion Terms x2"; interpretation I-12, ERRATA.md).
	TargetTermsTimesTwo TargetKind = "terms_x2"
)

// Advancement is one commission or promotion row of box A.
type Advancement struct {
	Name string `json:"name"`

	// FromClasses lists the rank classes eligible to attempt it (chart 06:
	// "Temp may attempt Officer Commission and Rating Promotion ...
	// Officer may attempt Officer Promotion").
	FromClasses []string `json:"from_classes"`

	// ToRank is the rank id a success confers; empty advances to the next
	// rank of the character's current class.
	ToRank string `json:"to_rank,omitempty"`

	Target TargetKind `json:"target"`

	// Check names the characteristic for TargetCharacteristic.
	Check string `json:"check,omitempty"`

	// Mod is the chart's conditional modifier (chart 06: "*Mod +3 if
	// Int 8+").
	Mod *AdvancementMod `json:"mod,omitempty"`

	// MedalMods adds the character's medal modifiers to the target
	// (chart 08's "*+Medals and WB Mods"; interpretation I-31, ERRATA.md).
	MedalMods bool `json:"medal_mods,omitempty"`
}

// Schemes is chart 10's Rogue Schemes table, indexed by a Flux roll.
type Schemes struct {
	Cite string      `json:"cite"`
	Rows []SchemeRow `json:"rows"`
}

// SchemeRow is one scheme: the career it imitates and what it is worth.
// A row pays either credits or ship shares, never both.
type SchemeRow struct {
	Flux   int    `json:"flux"`
	Career string `json:"career"`

	Credits    int `json:"credits,omitempty"`
	ShipShares int `json:"ship_shares,omitempty"`
}

// SchemeAt returns the scheme for a Flux result, clamped to the printed
// range.
func (s *Schemes) SchemeAt(flux int) SchemeRow {
	best := s.Rows[0]

	for _, row := range s.Rows {
		if row.Flux <= flux {
			best = row
		}
	}

	return best
}

// Undercover is chart 09's Agent Undercover Assignment table.
type Undercover struct {
	Cite string          `json:"cite"`
	Rows []UndercoverRow `json:"rows"`
}

// UndercoverRow is one assignment: the cover career and the titles its C
// column offers. Titles are transcribed as printed, including chart 09's
// "Pilor" and "World Discover".
type UndercoverRow struct {
	A int `json:"a"`
	B int `json:"b"`

	// Career is the label chart 09 prints; Source names the career
	// definition whose skills table it refers to.
	Career string `json:"career"`
	Source string `json:"source"`

	Titles []string `json:"titles"`

	// JobTable marks the two Citizen rows, which say "Roll on Citizen Life
	// Skills" — chart 04's table E — rather than offering a selection.
	JobTable bool `json:"job_table,omitempty"`
}

// UndercoverAt returns the assignment for die faces a and b.
func (u *Undercover) UndercoverAt(a, b int) (UndercoverRow, bool) {
	for _, row := range u.Rows {
		if row.A == a && row.B == b {
			return row, true
		}
	}

	return UndercoverRow{}, false
}

// ArmedForces is the Branch and Operations machinery of the Spacer,
// Soldier, and Marine charts (pp. 81, 82, 86; prose p. 66). A row carries
// one Mod per rank class: the Spacer's Naval Branch table (p. 81) prints
// separate Officer and Enlisted name-and-Mod columns, and the Army table
// prints one set that serves both (see Branch).
type ArmedForces struct {
	// BranchCheck is the characteristic checked to select rather than roll
	// a Branch (chart 08: "Select Branch Soc").
	BranchCheck string `json:"branch_check"`

	BranchCite     string `json:"branch_cite"`
	OperationsCite string `json:"operations_cite"`

	// EduDM is added to the Branch and Operations rolls at EduDMAt and
	// above ("DM +2 if Edu 10+").
	EduDMAt int `json:"edu_dm_at"`
	EduDM   int `json:"edu_dm"`

	// OperationsUseBranchDM adds the Branch's DM to the Operations roll
	// (chart 08: "1D+Branch DM plus +2 if Edu 10+").
	OperationsUseBranchDM bool `json:"operations_use_branch_dm,omitempty"`

	// OperationsPerTerm is the number of assignments a term draws: "Roll
	// for Assignment four times per Term (for four annual assignments)."
	// (p. 66)
	OperationsPerTerm int `json:"operations_per_term"`

	Branches   []Branch    `json:"branches"`
	Operations []Operation `json:"operations"`
}

// Branch is one row of a service's Branch table, indexed by the 1D roll.
// The Naval table prints separate Officer and Enlisted names and Mods for
// the same row (chart 07, p. 81), which is how "for Spacers, Crew becomes
// Line" on commission (p. 66) falls out: the row is fixed and the side
// follows the rank class. Where a chart prints one set, as the Army does,
// the enlisted fields are empty and the officer's serve both.
type Branch struct {
	Name string `json:"name"`

	// Mod is the Branch Mod applied to Risk and Reward; DM is the Branch
	// DM added to the Operations roll.
	Mod int `json:"mod"`
	DM  int `json:"dm"`

	// EnlistedName and EnlistedMod are the row's enlisted side where the
	// chart prints one.
	EnlistedName string `json:"enlisted_name,omitempty"`
	EnlistedMod  int    `json:"enlisted_mod,omitempty"`

	// AutoSkill and AutoTrade are the branch's automatic skills:
	// "if Medical Branch= Medic-1; If Technical Branch= any Trade"
	// (chart 08).
	AutoSkill string `json:"auto_skill,omitempty"`
	AutoTrade bool   `json:"auto_trade,omitempty"`
}

// Side returns the branch name and Mod for a rank class.
func (b Branch) Side(officer bool) (string, int) {
	if officer || b.EnlistedName == "" {
		return b.Name, b.Mod
	}

	return b.EnlistedName, b.EnlistedMod
}

// Operation is one row of a service's Operations table.
type Operation struct {
	Name string `json:"name"`
	Mod  int    `json:"mod"`

	// Column names the skills-table column the assignment opens where it
	// differs from the operation's own name (chart 07's Patrol and Strike
	// both open "Patrol/Strike"); empty means the name itself.
	Column string `json:"column,omitempty"`

	// Implemented is false for the ANM School assignment, whose schooling
	// is "resolved as Education" (chart 08) and lands with Later Education
	// (docs/PRD.md milestone 4). The assignment still happens and still
	// contributes its Mod.
	Implemented *bool `json:"implemented,omitempty"`
}

// BranchAt returns the Branch for a 1D roll plus modifiers, clamped to the
// table (interpretation I-33, ERRATA.md).
func (a *ArmedForces) BranchAt(roll int) Branch {
	return a.Branches[clampIndex(roll, len(a.Branches))]
}

// OperationAt returns the Operation for a modified roll, clamped to the
// table.
func (a *ArmedForces) OperationAt(roll int) Operation {
	return a.Operations[clampIndex(roll, len(a.Operations))]
}

// clampIndex maps a 1-based roll onto a table, clamping both ends.
func clampIndex(roll, size int) int {
	if roll < 1 {
		return 0
	}

	if roll > size {
		return size - 1
	}

	return roll - 1
}

// AdvancementMod is a conditional throw modifier: Value applies when the
// named characteristic is at least Min.
type AdvancementMod struct {
	Characteristic string `json:"characteristic"`
	Min            int    `json:"min"`
	Value          int    `json:"value"`
}

// RankByID returns the named rank.
func (d *Definition) RankByID(id string) (Rank, bool) {
	for _, rank := range d.Ranks {
		if rank.ID == id {
			return rank, true
		}
	}

	return Rank{}, false
}

// NextRank returns the rank above id within its own class, reporting false
// at the top of the ladder.
func (d *Definition) NextRank(id string) (Rank, bool) {
	current, ok := d.RankByID(id)
	if !ok {
		return Rank{}, false
	}

	seen := false

	for _, rank := range d.Ranks {
		switch {
		case rank.ID == id:
			seen = true
		case seen && rank.Class == current.Class:
			return rank, true
		}
	}

	return Rank{}, false
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
	EntryTrade: true, EntryArt: true, EntryScience: true, EntryStarship: true,
	EntrySoldier: true, EntryCapital: true, EntryAnySkill: true,
	EntryAnyKnowledge: true, EntryNewTrade: true, EntryNone: true,
}

// validate rejects malformed-but-parseable career data at load time, so
// the engine can trust every definition unconditionally. All thirteen
// milestone-3 data files go through this same gate.
func (d *Definition) validate() error {
	if d.Name == "" {
		return fmt.Errorf("%w: nameless career", errBadDefinition)
	}

	if err := d.validateTermCounts(); err != nil {
		return err
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

	if err := d.validateRanks(); err != nil {
		return err
	}

	if err := d.validateArmedForces(); err != nil {
		return err
	}

	if err := d.validateSchemes(); err != nil {
		return err
	}

	if err := d.validateMusterOut(); err != nil {
		return err
	}

	return d.validateJobTable()
}

// validateTermCounts checks the counts a definition charges per term: the
// skill eligibilities every career grants, and the terms per point of
// Sanity where a chart charges some ("reduce San= -1 for each TWO Terms
// served", chart 05 p. 79, the only career that prints such a rule).
func (d *Definition) validateTermCounts() error {
	if d.SkillsPerTerm < 1 {
		return fmt.Errorf("%w: %q has %d skills per term", errBadDefinition, d.Name, d.SkillsPerTerm)
	}

	if d.SanityPerTerms < 0 {
		return fmt.Errorf("%w: negative sanity per terms %d", errBadDefinition, d.SanityPerTerms)
	}

	return nil
}

// validateSchemes checks chart 10's Rogue Schemes table, so that SchemeAt
// can index it unconditionally: an empty table would make its clamp
// address a slice with no rows, and out-of-order rows would make the clamp
// return the wrong scheme.
func (d *Definition) validateSchemes() error {
	if d.Schemes == nil {
		return nil
	}

	if len(d.Schemes.Rows) == 0 {
		return fmt.Errorf("%w: %q has an empty schemes table", errBadDefinition, d.Name)
	}

	for i, row := range d.Schemes.Rows {
		if i > 0 && row.Flux <= d.Schemes.Rows[i-1].Flux {
			return fmt.Errorf("%w: schemes table is not in ascending Flux order at row %d",
				errBadDefinition, i)
		}

		// "one Ship Share" or a credit Value, never both (chart 10).
		if (row.Credits == 0) == (row.ShipShares == 0) {
			return fmt.Errorf("%w: scheme Flux %+d pays both credits and ship shares, or neither",
				errBadDefinition, row.Flux)
		}
	}

	return nil
}

// validateArmedForces checks the Branch and Operations tables, so that
// BranchAt and OperationAt can index them unconditionally: an empty table
// would make their clamp address a slice with no rows.
func (d *Definition) validateArmedForces() error {
	forces := d.ArmedForces
	if forces == nil {
		return nil
	}

	if !characteristicNames[forces.BranchCheck] {
		return fmt.Errorf("%w: unknown Branch check characteristic %q", errBadDefinition, forces.BranchCheck)
	}

	if len(forces.Branches) == 0 || len(forces.Operations) == 0 {
		return fmt.Errorf("%w: armed forces need both a Branch and an Operations table", errBadDefinition)
	}

	if forces.OperationsPerTerm < 1 {
		return fmt.Errorf("%w: %d operations per term", errBadDefinition, forces.OperationsPerTerm)
	}

	if err := validateBranches(forces.Branches); err != nil {
		return err
	}

	return d.validateOperations(forces.Operations)
}

// validateBranches checks each Branch row names itself and, where it
// carries an enlisted Mod, names the enlisted side that Mod belongs to.
func validateBranches(branches []Branch) error {
	for _, branch := range branches {
		if branch.Name == "" {
			return fmt.Errorf("%w: a Branch row has no name", errBadDefinition)
		}

		// An enlisted Mod without an enlisted name is silently ignored by
		// Side, which would hand the officer's Mod to a rating.
		if branch.EnlistedMod != 0 && branch.EnlistedName == "" {
			return fmt.Errorf("%w: Branch %q has an enlisted Mod but no enlisted name",
				errBadDefinition, branch.Name)
		}
	}

	return nil
}

// validateOperations checks each Operations row names itself and that any
// skills-column override names a column the chart actually prints: an
// unmatched override would silently open no column at all, costing the
// term its skill eligibility with no error.
func (d *Definition) validateOperations(operations []Operation) error {
	columns := make(map[string]bool, len(d.SkillColumns))
	for _, column := range d.SkillColumns {
		columns[column.Name] = true
	}

	for _, operation := range operations {
		if operation.Name == "" {
			return fmt.Errorf("%w: an Operations row has no name", errBadDefinition)
		}

		if operation.Column != "" && !columns[operation.Column] {
			return fmt.Errorf("%w: Operation %q names skills column %q, which the chart does not have",
				errBadDefinition, operation.Name, operation.Column)
		}
	}

	return nil
}

// validateRanks checks the rank table, entry tracks, and advancement rows
// reference each other and the Master Skill List consistently.
func (d *Definition) validateRanks() error {
	ids := map[string]bool{}

	for _, rank := range d.Ranks {
		if rank.ID == "" || rank.Class == "" || rank.Title == "" {
			return fmt.Errorf("%w: rank %+v: id, class, and title are required", errBadDefinition, rank)
		}

		if ids[rank.ID] {
			return fmt.Errorf("%w: duplicate rank id %q", errBadDefinition, rank.ID)
		}

		ids[rank.ID] = true

		if rank.AutoSkill == "" {
			continue
		}

		if err := skill.Validate(rank.AutoSkill); err != nil {
			return fmt.Errorf("%w: rank %q: %w", errBadDefinition, rank.ID, err)
		}
	}

	classes := map[string]bool{}
	for _, rank := range d.Ranks {
		classes[rank.Class] = true
	}

	if err := d.validateBeginTracks(ids); err != nil {
		return err
	}

	return d.validateAdvancements(ids, classes)
}

// validateBeginTracks checks the entry paths. A track of a career with a
// rank table must name the rank it enters: the engine enters the rank
// unconditionally, so an unnamed one would fail at generation time.
func (d *Definition) validateBeginTracks(ids map[string]bool) error {
	if len(d.BeginTracks) > 0 && len(d.BeginChecks) > 0 {
		return fmt.Errorf("%w: begin_tracks and begin_checks are exclusive", errBadDefinition)
	}

	for _, track := range d.BeginTracks {
		if err := validateBeginTrack(track, ids, len(d.Ranks) > 0); err != nil {
			return err
		}
	}

	return nil
}

// validateBeginTrack checks one entry path.
func validateBeginTrack(track BeginTrack, ids map[string]bool, ranked bool) error {
	if track.Name == "" {
		return fmt.Errorf("%w: nameless begin track", errBadDefinition)
	}

	for _, check := range track.Checks {
		if !characteristicNames[check] {
			return fmt.Errorf("%w: begin track %q checks unknown characteristic %q",
				errBadDefinition, track.Name, check)
		}
	}

	if track.Rank == "" {
		if ranked {
			return fmt.Errorf("%w: begin track %q names no rank", errBadDefinition, track.Name)
		}

		return nil
	}

	if !ids[track.Rank] {
		return fmt.Errorf("%w: begin track %q enters unknown rank %q", errBadDefinition, track.Name, track.Rank)
	}

	return nil
}

// validateAdvancements checks the commission and promotion rows.
func (d *Definition) validateAdvancements(ids, classes map[string]bool) error {
	for _, a := range d.Advancements {
		if err := validateAdvancement(a, ids, classes); err != nil {
			return err
		}
	}

	return nil
}

// validateAdvancement checks one commission or promotion row. from_classes
// is matched against the rank table's classes: a misspelled class would
// otherwise make the row silently unreachable rather than fail to load.
func validateAdvancement(a Advancement, ids, classes map[string]bool) error {
	if a.Name == "" || len(a.FromClasses) == 0 {
		return fmt.Errorf("%w: advancement %+v: name and from_classes are required", errBadDefinition, a)
	}

	for _, class := range a.FromClasses {
		if !classes[class] {
			return fmt.Errorf("%w: advancement %q names unknown rank class %q", errBadDefinition, a.Name, class)
		}
	}

	if err := validateAdvancementTarget(a); err != nil {
		return err
	}

	if a.ToRank != "" && !ids[a.ToRank] {
		return fmt.Errorf("%w: advancement %q targets unknown rank %q", errBadDefinition, a.Name, a.ToRank)
	}

	if a.Mod != nil && !characteristicNames[a.Mod.Characteristic] {
		return fmt.Errorf("%w: advancement %q mod names unknown characteristic %q",
			errBadDefinition, a.Name, a.Mod.Characteristic)
	}

	return nil
}

// validateAdvancementTarget checks the row's throw-target form.
func validateAdvancementTarget(a Advancement) error {
	switch a.Target {
	case TargetCharacteristic:
		if !characteristicNames[a.Check] {
			return fmt.Errorf("%w: advancement %q checks unknown characteristic %q",
				errBadDefinition, a.Name, a.Check)
		}
	case TargetTermsTimesTwo:
		if a.Check != "" {
			return fmt.Errorf("%w: advancement %q: %s target takes no check",
				errBadDefinition, a.Name, a.Target)
		}
	default:
		return fmt.Errorf("%w: advancement %q has unknown target %q", errBadDefinition, a.Name, a.Target)
	}

	return nil
}

// countTrue counts the set flags.
func countTrue(flags ...bool) int {
	n := 0

	for _, flag := range flags {
		if flag {
			n++
		}
	}

	return n
}

// validateContinueForm enforces exactly one Continue form.
func (d *Definition) validateContinueForm() error {
	fixed := d.ContinueTarget != 0
	characteristic := d.ContinueCharacteristic != ""

	// Chart 13 is the one career with no Continue throw at all: "Continue
	// Office Politics" defers to the Risk result already rolled for the
	// term ("Risk Failure: Functionary career ends. The character may not
	// Continue", p. 87). So it declares that form and no other.
	forms := countTrue(fixed, characteristic, d.ContinueFame, d.ContinueCC,
		d.ContinueOfficePolitics, d.ContinueSkill != "")
	if forms != 1 {
		return fmt.Errorf("%w: want exactly one Continue form, have %d", errBadDefinition, forms)
	}

	if d.ContinueCC && !d.CCFixed {
		return fmt.Errorf("%w: continue_cc needs a career-long controlling characteristic", errBadDefinition)
	}

	return nil
}

// validateContinue enforces exactly one Continue form: a fixed target in
// 2..11 (11 is the largest target a 2D roll-low throw can miss on its
// merits, p. 66; above it the only exit left is the p. 134 automatic
// failure on a natural 12, which dice.resolveCheck applies, so the term
// loop terminates but runs long), or a characteristic target (chart 05:
// "Continue Int"). The bound is on the fixed form only: ContinueMod and a
// characteristic target both carry the printed rule past 12 by design.
func (d *Definition) validateContinue() error {
	if err := d.validateContinueForm(); err != nil {
		return err
	}

	switch d.ContinueMod {
	case "", ContinueModPublications, ContinueModTerms:
	default:
		return fmt.Errorf("%w: unknown continue mod %q", errBadDefinition, d.ContinueMod)
	}

	if d.ContinueTarget != 0 && (d.ContinueTarget < 2 || d.ContinueTarget > 11) {
		return fmt.Errorf("%w: continue target %d outside 2-11", errBadDefinition, d.ContinueTarget)
	}

	if d.ContinueCharacteristic != "" && !characteristicNames[d.ContinueCharacteristic] {
		return fmt.Errorf("%w: unknown continue characteristic %q", errBadDefinition, d.ContinueCharacteristic)
	}

	if (d.ContinueSkill != "") != (d.ContinueSkillMultiplier != 0) {
		return fmt.Errorf("%w: continue skill and its multiplier must both be set", errBadDefinition)
	}

	return nil
}

// validateCharacteristicNames checks the begin/retry/controlling
// characteristic names against the six standard abbreviations.
func (d *Definition) validateCharacteristicNames() error {
	// Chart 03's Risk & Reward row names Talent rather than a series of
	// characteristics, so the Entertainer rotates none.
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

	// Skill cells name Master Skill List entries: the chart abbreviations
	// are canonicalized in the transcription (ERRATA.md I-9), so a name
	// that does not resolve is a transcription error.
	if entry.Kind == EntrySkill {
		if err := skill.Validate(entry.Name); err != nil {
			return fmt.Errorf("%w: column %q: %w", errBadDefinition, column, err)
		}
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

				if cell == NoSkillCell {
					continue
				}

				if err := skill.Validate(cell); err != nil {
					return fmt.Errorf("%w: job table cell at A%d B%d C%d: %w", errBadDefinition, a+1, b+1, c+1, err)
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

//go:embed data/merchant.json
var merchantJSON []byte

//go:embed data/entertainer.json
var entertainerJSON []byte

//go:embed data/scholar.json
var scholarJSON []byte

//go:embed data/noble.json
var nobleJSON []byte

//go:embed data/soldier.json
var soldierJSON []byte

//go:embed data/spacer.json
var spacerJSON []byte

//go:embed data/marine.json
var marineJSON []byte

//go:embed data/agent.json
var agentJSON []byte

//go:embed data/functionary.json
var functionaryJSON []byte

//go:embed data/rogue.json
var rogueJSON []byte

//go:embed data/craftsman.json
var craftsmanJSON []byte

// The implemented careers parse and validate their embedded definitions
// once.
var (
	citizen = sync.OnceValues(func() (*Definition, error) {
		return load("citizen.json", citizenJSON)
	})
	scout = sync.OnceValues(func() (*Definition, error) {
		return load("scout.json", scoutJSON)
	})
	merchant = sync.OnceValues(func() (*Definition, error) {
		return load("merchant.json", merchantJSON)
	})
	entertainer = sync.OnceValues(func() (*Definition, error) {
		return load("entertainer.json", entertainerJSON)
	})
	scholar = sync.OnceValues(func() (*Definition, error) {
		return load("scholar.json", scholarJSON)
	})
	noble = sync.OnceValues(func() (*Definition, error) {
		return load("noble.json", nobleJSON)
	})
	soldier = sync.OnceValues(func() (*Definition, error) {
		return load("soldier.json", soldierJSON)
	})
	spacer = sync.OnceValues(func() (*Definition, error) {
		return load("spacer.json", spacerJSON)
	})
	marine = sync.OnceValues(func() (*Definition, error) {
		return load("marine.json", marineJSON)
	})
	agent = sync.OnceValues(func() (*Definition, error) {
		return load("agent.json", agentJSON)
	})
	craftsman = sync.OnceValues(func() (*Definition, error) {
		return load("craftsman.json", craftsmanJSON)
	})
	functionary = sync.OnceValues(func() (*Definition, error) {
		return load("functionary.json", functionaryJSON)
	})
	rogue = sync.OnceValues(func() (*Definition, error) {
		return load("rogue.json", rogueJSON)
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

// Merchant returns the Merchant career definition (chart 06, p. 80).
func Merchant() (*Definition, error) {
	return merchant()
}

// Entertainer returns the Entertainer career definition (chart 03, p. 77).
func Entertainer() (*Definition, error) {
	return entertainer()
}

// Scholar returns the Scholar career definition (chart 02, p. 76).
func Scholar() (*Definition, error) {
	return scholar()
}

// Rogue returns the Rogue career definition (chart 10, p. 84).
func Rogue() (*Definition, error) {
	return rogue()
}

// Agent returns the Agent career definition (chart 09, p. 83).
func Agent() (*Definition, error) {
	return agent()
}

// Craftsman returns the Craftsman career definition (chart 01, p. 75).
func Craftsman() (*Definition, error) {
	return craftsman()
}

// Functionary returns the Functionary career definition (chart 13, p. 87).
// Its skills table was transcribed first as a reference for chart 09's
// Undercover Assignment (interpretation I-40); the career became playable
// when career changes landed, chart 13 saying it "is never a first career".
func Functionary() (*Definition, error) {
	return functionary()
}

// Marine returns the Marine career definition (chart 12, p. 86).
func Marine() (*Definition, error) {
	return marine()
}

// Spacer returns the Spacer career definition (chart 07, p. 81).
func Spacer() (*Definition, error) {
	return spacer()
}

// Soldier returns the Soldier career definition (chart 08, p. 82).
func Soldier() (*Definition, error) {
	return soldier()
}

// Noble returns the Noble career definition (chart 11, p. 85).
func Noble() (*Definition, error) {
	return noble()
}

// Available lists the implemented careers in Book 1 chart order. The
// default policy names its career rather than taking the first listed, so
// this order is presentation only (POLICY.md).
//
// Not every career here can start a lifepath: see NotAFirstCareer and
// FirstCareers.
func Available() []string {
	return []string{
		"Craftsman", "Scholar", "Entertainer", "Citizen", "Scout", "Merchant",
		"Spacer", "Soldier", "Agent", "Rogue", "Noble", "Marine",
		"Functionary",
	}
}

// FirstCareers lists the careers a lifepath may open with, which is
// Available minus those a chart bars from the start: "Functionary is never
// a first career" (chart 13 p. 87).
func FirstCareers() ([]string, error) {
	all := Available()
	first := make([]string, 0, len(all))

	for _, name := range all {
		def, err := ByName(name)
		if err != nil {
			return nil, err
		}

		if def.NotAFirstCareer {
			continue
		}

		first = append(first, name)
	}

	return first, nil
}

// loaders maps each Available name to its definition loader.
var loaders = map[string]func() (*Definition, error){
	"Scholar": Scholar, "Entertainer": Entertainer, "Citizen": Citizen,
	"Scout": Scout, "Merchant": Merchant, "Spacer": Spacer,
	"Soldier": Soldier, "Agent": Agent, "Rogue": Rogue,
	"Noble": Noble, "Marine": Marine, "Functionary": Functionary,
	"Craftsman": Craftsman,
}

// ByName loads a career definition by its Available name.
func ByName(name string) (*Definition, error) {
	load, ok := loaders[name]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownCareer, name)
	}

	return load()
}

// ErrUnknownCareer reports a name absent from Available.
var ErrUnknownCareer = errors.New("career: unknown career")

// musterOutDMKinds is the closed vocabulary of table D DM lines: "+Terms"
// and "+Total Terms", "+Commends", "+ Officer Rank", "+ Scholar Rank", and
// "+Fame/N" (charts 01-13 box D, pp. 75-87). A DM is a string rather than a
// benefit kind, so without this a typo would load and then modify nothing.
var musterOutDMKinds = map[string]bool{
	"terms": true, "total_terms": true, "commendations": true,
	"officer_rank": true, "scholar_rank": true, "fame": true,
}

// validateMusterOut checks a career's table D, and the chart M2 reprint
// where it carries one, so the engine can index rows unconditionally.
func (d *Definition) validateMusterOut() error {
	if d.MusterOut == nil && d.MusterOutM2 == nil {
		return nil
	}

	benefits, err := benefit.Load()
	if err != nil {
		return fmt.Errorf("muster out: %w", err)
	}

	for _, t := range []*MusterOut{d.MusterOut, d.MusterOutM2} {
		if t == nil {
			continue
		}

		if err := d.validateMusterOutTable(benefits, t); err != nil {
			return err
		}
	}

	return nil
}

// validateMusterOutTable checks one transcribed table D: its rows, its
// cells, and the DM line under each column.
func (d *Definition) validateMusterOutTable(benefits *benefit.Table, t *MusterOut) error {
	if len(t.Rows) == 0 {
		return fmt.Errorf("%w: %q has an empty muster-out table", errBadDefinition, d.Name)
	}

	for i, row := range t.Rows {
		// A table D is read with 1D plus DMs, so its rows must run
		// from 1 without a gap.
		if row.Roll != i+1 {
			return fmt.Errorf("%w: %q muster-out row %d is numbered %d",
				errBadDefinition, d.Name, i+1, row.Roll)
		}

		if err := d.validateMusterOutRow(benefits, t, row); err != nil {
			return err
		}
	}

	for _, dm := range []*MusterOutDM{&t.MoneyDM, &t.BenefitDM, t.PowerDM} {
		if dm == nil {
			continue
		}

		if err := validateMusterOutDM(d.Name, *dm); err != nil {
			return err
		}
	}

	return nil
}

// validateMusterOutRow checks one row's cells and its third column.
func (d *Definition) validateMusterOutRow(benefits *benefit.Table, t *MusterOut, row MusterOutRow) error {
	// Chart 11 is the only table with a Power column, and it prints one on
	// every row: a Power cell without its DM line, or the reverse, is a
	// half-transcribed column.
	if (row.Power != nil) != (t.PowerDM != nil) {
		return fmt.Errorf("%w: %q muster-out row %d and its power DM disagree on the third column",
			errBadDefinition, d.Name, row.Roll)
	}

	for _, cell := range []*MusterOutCell{&row.Money, &row.Benefit, row.Power} {
		if cell == nil {
			continue
		}

		if err := validateMusterOutCell(benefits, d.Name, row.Roll, *cell); err != nil {
			return err
		}
	}

	return nil
}

// validateMusterOutDM checks one column's DM line, so that a "+Fame/3"
// never reaches the engine with the divisor missing.
func validateMusterOutDM(career string, dm MusterOutDM) error {
	if !musterOutDMKinds[dm.Kind] {
		return fmt.Errorf("%w: %q muster-out DM %q is not a printed DM line",
			errBadDefinition, career, dm.Kind)
	}

	// "+Fame/3" (chart 03) and "+Fame/2" (chart 05) are the only divided
	// DMs the book prints, and a Fame DM is never printed undivided.
	if (dm.Kind == "fame") != (dm.Divisor > 0) {
		return fmt.Errorf("%w: %q muster-out DM %q has divisor %d",
			errBadDefinition, career, dm.Kind, dm.Divisor)
	}

	return nil
}

// validateMusterOutCell checks one cell against the chart M1 vocabulary.
func validateMusterOutCell(benefits *benefit.Table, career string, roll int, cell MusterOutCell) error {
	if _, err := benefits.For(cell.Kind); err != nil {
		return fmt.Errorf("%w: %q row %d: %w", errBadDefinition, career, roll, err)
	}

	if cell.Kind == benefit.Characteristic && !characteristicNames[cell.Detail] {
		return fmt.Errorf("%w: %q row %d raises %q, which is not a characteristic",
			errBadDefinition, career, roll, cell.Detail)
	}

	// A money cell pays a printed sum and nothing else pays one; a count
	// and a rolled count are alternatives, never both.
	if (cell.Kind == benefit.Money) != (cell.Credits > 0) {
		return fmt.Errorf("%w: %q row %d pays Cr%d as a %q cell",
			errBadDefinition, career, roll, cell.Credits, cell.Kind)
	}

	if cell.Count < 0 || cell.Dice < 0 || (cell.Count > 0 && cell.Dice > 0) {
		return fmt.Errorf("%w: %q row %d gives count %d and dice %d",
			errBadDefinition, career, roll, cell.Count, cell.Dice)
	}

	return nil
}
