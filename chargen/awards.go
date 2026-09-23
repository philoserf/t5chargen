package chargen

// The award machinery a career's cells draw on: the open and group cells
// of table C, the Major and Minor columns, chart 04's Citizen Life Skills,
// and the characteristic raises. The term loop in careerrun.go calls into
// these; they are kept apart from it so the loop reads in order.

import (
	"fmt"
	"slices"
	"strings"

	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/skill"
)

// firstReceiptLevels applies the first-receipt rule: the stated level on
// first receipt, Skill-1 thereafter (docs/PRD.md FR5; for Citizen, "with
// Skill-4 (later receipts are Skill-1)", p. 78). A skill already received
// during this career counts as held, so the determination is a later
// receipt: +1. Pre-career levels (homeworld grants, education) are not
// career receipts and do not demote the award — interpretation I-2,
// ERRATA.md.
func (r *careerRun) firstReceiptLevels(name string, firstReceipt int) int {
	if r.character.skillLevel(name) > r.entryLevels[name] {
		return 1
	}

	return firstReceipt
}

// awardAndLog awards skill levels via the career-independent
// awardSkillAndLog.
func (r *careerRun) awardAndLog(name string, levels, cause int) error {
	return awardSkillAndLog(name, levels, cause, r.log, r.decider, r.character)
}

// groupCells maps the chart's open-selection cells to the Master Skill
// List group they select from: "One Art", "One Trade", "One Science", and
// "Starship Skill" (charts 04-06; p. 132 chart MS). Chart B's own Art and
// Trade lists (p. 56) are the same six and ten names.
var groupCells = map[career.EntryKind]struct {
	prompt string
	names  func() []string
}{
	career.EntryArt: {
		prompt: "Select One Art",
		names:  func() []string { return skill.InGroup(skill.GroupArts) },
	},
	career.EntryTrade: {
		prompt: "Select One Trade",
		names:  func() []string { return skill.InGroup(skill.GroupTrades) },
	},
	career.EntryScience: {
		prompt: "Select One Science",
		names:  func() []string { return skill.UnderParent(skill.ParentSciences) },
	},
	career.EntryStarship: {
		prompt: "Select a Starship Skill",
		names:  func() []string { return skill.InGroup(skill.GroupStarship) },
	},
	career.EntrySoldier: {
		prompt: "Select a Soldier Skill",
		names:  func() []string { return skill.InGroup(skill.GroupSoldier) },
	},

	// Chart 09's Vocation column prints "Any Knowledge" (p. 83) and chart
	// 13's General column "Any Skill*** ... from Citizen Life Skills and
	// Knowledges" (p. 87); both are open selections over a whole list
	// rather than a Master Skill List group.
	career.EntryAnyKnowledge: {
		prompt: "Select Any Knowledge",
		names:  allKnowledges,
	},
	career.EntryAnySkill: {
		prompt: "Select Any Skill from Citizen Life Skills and Knowledges",
		names:  citizenLifeSkills,
	},
}

// citizenLifeSkills is chart 13's "Any Skill*** from Citizen Life Skills
// and Knowledges" (p. 87): every table E entry of chart 04, in chart
// order. Chart 09's Undercover Assignment reads the same list.
func citizenLifeSkills() []string {
	def, err := career.ByName("Citizen")
	if err != nil {
		// Unreachable in a built binary: Generate checks the same load
		// once before any dice are rolled, so a broken transcription is
		// reported as one rather than reaching here and being reported
		// by awardFromGroup as a deferred milestone.
		return nil
	}

	return def.HobbyChoices()
}

// allKnowledges is chart 09's "Any Knowledge" cell (p. 83): every
// Knowledge on the Master Skill List (p. 132 chart MS), in name order.
func allKnowledges() []string {
	var names []string

	for _, name := range skill.Names() {
		if entry, ok := skill.Lookup(name); ok && entry.Kind == skill.KindKnowledge {
			names = append(names, name)
		}
	}

	return names
}

// article returns the indefinite article for a career name, so the prompt
// reads "an Entertainer" rather than "a Entertainer".
func article(name string) string {
	if name == "" {
		return "a"
	}

	if strings.ContainsRune("AEIOU", rune(name[0])) {
		return "an"
	}

	return "a"
}

// major and minor report the character's current Major and Minor,
// preferring any the career in progress selected for itself (chart 02's
// degreeless Scholar) over the education records, since its record is not
// on the character until the career ends.
func (r *careerRun) major() string {
	if r.record.Major != "" {
		return r.record.Major
	}

	return r.character.currentMajor()
}

func (r *careerRun) minor() string {
	if r.record.Minor != "" {
		return r.record.Minor
	}

	return r.character.currentMinor()
}

// awardOpenCell resolves the cells that select from a Master Skill List
// group, and rejects the ones still waiting on later milestones.
func (r *careerRun) awardOpenCell(kind career.EntryKind) error {
	if kind == career.EntryNewTrade {
		return r.awardNewTrade()
	}

	if kind == career.EntryCapital {
		// "Capital*** = World Knowledge (of world of highest held noble
		// Land Grant)" (chart 11 p. 85). Grant income is priced now, but
		// "highest held" ranks grants against each other, which needs the
		// per-title hex table interpretation I-83 declines to apply — and
		// no chart in Book 1 awards a World Knowledge, so there is no
		// naming convention to follow either.
		return fmt.Errorf("%w: Capital cell needs a ranking of Land Grants", errNotImplemented)
	}

	return r.awardFromGroup(kind)
}

// awardNewTrade resolves chart 01's "New Trade***" cell: "Any Trade not
// already held; if all are already held; this benefit is lost" (p. 75).
//
// It cannot go through groupCells, whose options do not depend on the
// character; this cell's do.
func (r *careerRun) awardNewTrade() error {
	trades := skill.InGroup(skill.GroupTrades)
	options := make([]string, 0, len(trades))

	for _, name := range trades {
		if r.character.skillLevel(name) == 0 {
			options = append(options, name)
		}
	}

	if len(options) == 0 {
		// "if all are already held; this benefit is lost". Skill names
		// the exhausted group, so the transcript says which benefit was
		// lost rather than falling through to the Major/Minor wording.
		// The exhaustion is the sum of every prior award and no throw
		// produced it, so the consequence names this step instead
		// (interpretation I-87, ERRATA.md).
		seq := r.log.Step("New Trade: every Trade is already held", r.def.Cite)
		r.log.Consequence(ConsequenceEvent{
			Cause: seq, Kind: ConsequenceBenefitLost, Career: r.def.Name, Skill: "Trade",
		})

		return nil
	}

	chosen, seq, err := choose(r.log, r.decider, Choice{
		ID:      ChooseSkill,
		Prompt:  "Select a New Trade",
		Options: options,
		Cite:    r.def.Cite + " (New Trade: any Trade not already held)",
	})
	if err != nil {
		return err
	}

	if err := r.awardAndLog(options[chosen], 1, seq); err != nil {
		return err
	}

	return nil
}

// awardFromGroup resolves an open-selection cell by choice and awards the
// selected skill, caused by the selecting choice event (docs/PRD.md FR10).
func (r *careerRun) awardFromGroup(kind career.EntryKind) error {
	cell, ok := groupCells[kind]
	if !ok {
		return fmt.Errorf("%w: %q cell", errNotImplemented, kind)
	}

	options := cell.names()
	if len(options) == 0 {
		return fmt.Errorf("%w: %q cell has no alternatives", errNotImplemented, kind)
	}

	chosen, seq, err := choose(r.log, r.decider, Choice{
		ID:      ChooseSkill,
		Prompt:  cell.prompt,
		Options: options,
		Cite:    r.def.Cite + " table C; Book 1 p. 132 chart MS",
	})
	if err != nil {
		return err
	}

	if err := r.awardAndLog(options[chosen], 1, seq); err != nil {
		return err
	}

	return nil
}

// resolveSkillName maps a career chart cell to its Master Skill List name.
// Most cells name one entry. Two chart 04 table E cells do not: "Grav" is
// printed once although the list holds a Grav knowledge under each of
// Driver, Flyer, and Seafarer, and "Spacecraft" covers both Spacecraft ACS
// and Spacecraft BCS (p. 132). Those are resolved by choice, in Master
// Skill List order (ERRATA.md I-10, I-11).
//
// exclude names entries the caller has already spent — the Citizen Hobby
// must differ from the Job (interpretation I-3), and the chart prints the
// label, not the resolved name, so the exclusion can only be applied here.
// A label always covers at least two entries, so at least one survives.
func (r *careerRun) resolveSkillName(name string, exclude ...string) (string, error) {
	options := skill.Options(name)
	if options == nil {
		return name, nil
	}

	options = slices.DeleteFunc(options, func(option string) bool {
		return slices.Contains(exclude, option)
	})

	chosen, _, err := choose(r.log, r.decider, Choice{
		ID:      ChooseSkill,
		Prompt:  "Select the specific " + name + " skill",
		Options: options,
		Cite:    "Book 1 p. 132 chart MS (Master Skill List); " + r.def.Cite,
	})
	if err != nil {
		return "", err
	}

	return options[chosen], nil
}

// rollUnder rolls 1D and rerolls until the face is within limit, which is
// how the charts read a column narrower than a die: chart 04 table E's
// "Roll A (reroll if >3)" (p. 78) and chart 09's A and C columns (p. 83).
// Every consumed face is logged, so the event log accounts for the
// rerolls.
func (r *careerRun) rollUnder(limit int, cite string) int {
	for {
		roll := r.roller.Roll(1)
		r.log.Roll(roll, cite)

		if roll.Total <= limit {
			return roll.Total
		}
	}
}

// columnIndex returns the table C column with the given name, or -1 for
// the Major and Minor options appended alongside them.
func (r *careerRun) columnIndex(name string) int {
	for i, column := range r.def.SkillColumns {
		if column.Name == name {
			return i
		}
	}

	return -1
}

// skillColumnOptions lists the table C columns, plus the character's Major
// and Minor where the chart allows taking one instead (chart 02 table C).
func (r *careerRun) skillColumnOptions() []string {
	columns := r.def.SkillColumnNames()
	if !r.def.MajorOrMinorColumn {
		return columns
	}

	// Made rather than cloned: slices.Clone returns nil for a nil input,
	// and a made slice is non-nil by construction.
	options := append(make([]string, 0, len(columns)+2), columns...)

	for _, name := range []string{r.major(), r.minor()} {
		if name != "" {
			options = append(options, name)
		}
	}

	return options
}

// awardMajorOrMinor awards a level in the named Major or Minor, taken
// instead of a table C roll. The selecting choice is the cause
// (docs/PRD.md FR10).
func (r *careerRun) awardMajorOrMinor(name string, cause int) error {
	if name == "" {
		return fmt.Errorf("%w: no Major or Minor to take", errNotImplemented)
	}

	if err := r.awardAndLog(name, 1, cause); err != nil {
		return err
	}

	return nil
}

// awardMajorCell applies a Major or Minor cell: "If the character does
// not have a Major/Minor this benefit is lost." (p. 78) The current
// Major and Minor are the most recent ones selected (p. 59).
func (r *careerRun) awardMajorCell(kind career.EntryKind, cause int) error {
	name := r.major()
	if kind == career.EntryMinor {
		name = r.minor()
	}

	if name == "" {
		r.log.Consequence(ConsequenceEvent{Cause: cause, Kind: ConsequenceBenefitLost})

		return nil
	}

	return r.awardAndLog(name, 1, cause)
}

// awardTableC applies one career skills table cell.
//
//nolint:exhaustive // Deliberately partitioned: the open-selection kinds are handled by awardOpenCell.
func (r *careerRun) awardTableC(entry career.Entry, cause int) error {
	switch entry.Kind {
	case career.EntrySkill:
		name, err := r.resolveSkillName(entry.Name)
		if err != nil {
			return err
		}

		if err := r.awardAndLog(name, 1, cause); err != nil {
			return err
		}
	case career.EntryCharacteristic:
		return r.awardCharacteristic(entry.Name, cause)
	case career.EntryMajor, career.EntryMinor:
		return r.awardMajorCell(entry.Kind, cause)
	case career.EntryNone:
		r.log.Consequence(ConsequenceEvent{Cause: cause, Kind: ConsequenceNoAward})
	default:
		// The remaining kinds select from a Master Skill List group, or
		// wait on a later milestone; an unknown kind is rejected there
		// rather than silently resolving to nothing.
		return r.awardOpenCell(entry.Kind)
	}

	return nil
}

// characteristicRaiser is the optional careerMechanics seam for a career
// whose rules react to a table C characteristic award. Chart 11's "Each
// increase in Soc during CharGen awards a Land Grant" is the only one so
// far (interpretation I-30, ERRATA.md); the notification carries the
// awarding cause so the consequence chains to it (docs/PRD.md FR10).
type characteristicRaiser interface {
	characteristicRaised(r *careerRun, name string, cause int)
}

// awardCharacteristic applies a Personal-column +1, subject to the p. 68
// maximum (awardCharacteristicAndLog), and notifies the career's mechanics
// when the increase actually lands.
func (r *careerRun) awardCharacteristic(name string, cause int) error {
	before, ok := characteristicValue(&r.character.Characteristics, name)
	if !ok {
		return fmt.Errorf("%w: %q", errUnknownCharacteristic, name)
	}

	awardCharacteristicAndLog(r.character, r.log, name, 1, cause)

	after, _ := characteristicValue(&r.character.Characteristics, name)
	if after == before {
		// The p. 68 maximum refused the increase; nothing was raised.
		return nil
	}

	if raiser, ok := r.mechanics.(characteristicRaiser); ok {
		raiser.characteristicRaised(r, name, cause)
	}

	return nil
}
