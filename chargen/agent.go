package chargen

// Agent career mechanics (Book 1 chart 09, p. 83; prose pp. 64-66).
// "To Begin C3 / Risk & Reward C1 C2 C3 C4 / Continue Str*" with "*Mod
// +Terms" (chart 09 box A).
//
// "The focus of the Agent is Missions: accomplishing the needs of a
// controlling organization. Each is one Term in length: two years
// Undercover in a different career (investigating, gathering information,
// preparing); followed by two years completing the Mission." (chart 09)
//
// The undercover half is the career's distinctive machinery: an assignment
// is rolled on a table of other careers, and the Agent may "Select (not
// Roll) one skill from the skill tables of that Career" — the only place
// one career reads another's chart. The mission half is the ordinary Risk
// and Reward, whose Reward success is a Commendation.
//
// Deferred: the muster-out table D and its "+Commends" money modifier
// (docs/PRD.md milestone 4).

import (
	"fmt"
	"slices"

	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/skill"
)

// agentMissionSkills is the table B eligibility a completed Mission earns
// ("Per Term 2 / Undercover 1 / Successful Mission 4", chart 09).
const agentMissionSkills = 4

// agentMechanics is the Agent careerMechanics implementation.
type agentMechanics struct{}

// newAgent is the Agent careerRegistry entry.
//
//nolint:ireturn // The registry's function type returns the interface.
func newAgent() (*career.Definition, careerMechanics, error) {
	def, err := career.Agent()
	if err != nil {
		return nil, nil, fmt.Errorf("agent career: %w", err)
	}

	// undercover indexes the table unconditionally: chart 09 has no term
	// without a Mission, so a definition without one is a data error.
	if def.Undercover == nil {
		return nil, nil, fmt.Errorf("%w: Agent has no Undercover Assignment table", errNotImplemented)
	}

	return def, &agentMechanics{}, nil
}

// begin rolls To Begin. A failed attempt costs one year (p. 65); chart 09
// lists no Begin retry.
func (*agentMechanics) begin(r *careerRun) (bool, error) {
	r.log.Step("Agent: To Begin", r.def.Cite)

	check, value, err := chooseCheckCharacteristic(r, r.def.BeginChecks)
	if err != nil {
		return false, err
	}

	throw := r.roller.Check(2, value)
	seq := r.log.Throw(throw, nil, r.def.Cite+" (To Begin vs "+check+")")

	if throw.Success {
		return true, nil
	}

	if err := r.character.advanceYears(1, r.roller, r.log, seq); err != nil {
		return false, err
	}

	r.log.Consequence(ConsequenceEvent{Cause: seq, Kind: ConsequenceCareerNotBegun, Career: r.def.Name})

	return false, nil
}

// resolveTerm runs the two halves of a Mission in the order the chart
// narrates them: the undercover assignment, then Risk and Reward. Nothing
// the undercover half awards modifies the mission rolls, so the order is
// the chart's rather than a mechanical requirement.
func (m *agentMechanics) resolveTerm(r *careerRun, cc string) (termOutcome, error) {
	if err := m.undercover(r); err != nil {
		return termOutcome{}, err
	}

	outcome, err := m.mission(r, cc)
	if err != nil || outcome.died {
		return outcome, err
	}

	outcome.skillRolls = r.def.SkillsPerTerm
	if outcome.success {
		outcome.skillRolls += agentMissionSkills
	}

	return outcome, nil
}

// undercover rolls the assignment — "Roll A (reroll if >3), then roll B,
// and finally top row C (reroll if >3) if required" (chart 09) — and
// awards the one skill the Agent selects from that career's table.
func (m *agentMechanics) undercover(r *careerRun) error {
	table := r.def.Undercover

	a := r.rollUnder(3, table.Cite+" (roll A, reroll if >3)")
	b := r.roller.Roll(1)
	seq := r.log.Roll(b, table.Cite+" (roll B)")

	row, ok := table.UndercoverAt(a, b.Total)
	if !ok {
		return fmt.Errorf("%w: undercover assignment A%d B%d", errNotImplemented, a, b.Total)
	}

	title := ""

	// "finally top row C (reroll if >3) if required": a row offering one
	// title needs no C roll, and one offering none needs no title
	// (interpretation I-38, ERRATA.md). The two Citizen rows print a roll
	// instruction where the others print titles, so they carry no cover
	// title at all (interpretation I-39).
	switch {
	case row.JobTable:
	case len(row.Titles) > 1:
		c := r.rollUnder(len(row.Titles), table.Cite+" (roll C, reroll if >3)")
		title = row.Titles[c-1]
	case len(row.Titles) == 1:
		title = row.Titles[0]
	}

	r.record.UndercoverCareer = row.Career
	r.record.UndercoverTitle = title

	r.log.Consequence(ConsequenceEvent{
		Cause: seq, Kind: ConsequenceUndercover, Career: row.Career, Skill: title,
	})

	if row.JobTable {
		return m.undercoverJobTable(r, row)
	}

	return m.undercoverSkill(r, row)
}

// undercoverJobTable resolves the two Citizen rows, which say "Roll on
// Citizen Life Skills" rather than offering a selection (chart 09). The
// skill is awarded at one level: the Agent is undercover, not a Citizen,
// and table B allows "Undercover 1" (interpretation I-39, ERRATA.md).
func (*agentMechanics) undercoverJobTable(r *careerRun, row career.UndercoverRow) error {
	def, err := loadUndercoverCareer(row.Source)
	if err != nil {
		return err
	}

	a := r.rollUnder(3, def.Cite+" table E (roll A, reroll if >3)")
	b := r.roller.Roll(1)
	r.log.Roll(b, def.Cite+" table E (roll B)")
	c := r.roller.Roll(1)
	seq := r.log.Roll(c, def.Cite+" table E (roll C)")

	entry := def.JobEntry(a, b.Total, c.Total)
	if entry.Kind == career.EntryNone {
		r.log.Consequence(ConsequenceEvent{Cause: seq, Kind: ConsequenceNoAward})

		return nil
	}

	name, err := r.resolveSkillName(entry.Name)
	if err != nil {
		return err
	}

	r.awardAndLog(name, 1, seq)

	return nil
}

// undercoverSkill offers the cover career's selectable skills: "Select
// (not Roll) one skill from the skill tables of that Career" (chart 09).
func (*agentMechanics) undercoverSkill(r *careerRun, row career.UndercoverRow) error {
	def, err := loadUndercoverCareer(row.Source)
	if err != nil {
		return err
	}

	options := undercoverSkills(def)
	if len(options) == 0 {
		return fmt.Errorf("%w: %s offers no selectable skill", errNotImplemented, row.Career)
	}

	chosen, seq, err := choose(r.log, r.decider, Choice{
		ID:      ChooseSkill,
		Prompt:  "Select a skill from the " + row.Career + " skills table",
		Options: options,
		Cite:    r.def.Cite + " (Select (not Roll) one skill from the skill tables of that Career)",
	})
	if err != nil {
		return err
	}

	r.awardAndLog(options[chosen], 1, seq)

	return nil
}

// undercoverCareers maps the labels chart 09 prints to the career whose
// skills table each refers to. Army is the Soldier chart, Navy the Spacer
// chart, and Functionary is transcribed for this table alone.
var undercoverCareers = map[string]func() (*career.Definition, error){
	"Soldier":     career.Soldier,
	"Marine":      career.Marine,
	"Spacer":      career.Spacer,
	"Scholar":     career.Scholar,
	"Entertainer": career.Entertainer,
	"Citizen":     career.Citizen,
	"Merchant":    career.Merchant,
	"Scout":       career.Scout,
	"Noble":       career.Noble,
	"Functionary": career.Functionary,
}

// loadUndercoverCareer resolves one cover career's chart.
func loadUndercoverCareer(source string) (*career.Definition, error) {
	load, ok := undercoverCareers[source]
	if !ok {
		return nil, fmt.Errorf("%w: undercover career %q", errNotImplemented, source)
	}

	def, err := load()
	if err != nil {
		return nil, fmt.Errorf("undercover career %q: %w", source, err)
	}

	return def, nil
}

// undercoverSkills flattens a career's table C into the skills a cover
// identity can be selected from, in chart order without repeats. Cells
// naming a Master Skill List group expand to their members, since "select
// one skill" replaces the roll that would otherwise have chosen among
// them; characteristic cells are not skills, and Major and Minor name the
// Agent's own areas rather than the cover career's (interpretation I-38,
// ERRATA.md).
func undercoverSkills(def *career.Definition) []string {
	var options []string

	for _, column := range def.SkillColumns {
		for _, entry := range column.Entries {
			for _, name := range undercoverCell(entry) {
				if name != "" && !slices.Contains(options, name) {
					options = append(options, name)
				}
			}
		}
	}

	return options
}

// undercoverCell returns the skills one table C cell offers a cover
// identity, which is nothing for the cells that are not skills.
//
//nolint:exhaustive // The remaining kinds offer a cover identity no skill.
func undercoverCell(entry career.Entry) []string {
	switch entry.Kind {
	case career.EntrySkill:
		return []string{entry.Name}
	case career.EntryArt:
		return skill.InGroup(skill.GroupArts)
	case career.EntryTrade:
		return skill.InGroup(skill.GroupTrades)
	case career.EntryScience:
		return skill.UnderParent(skill.ParentSciences)
	case career.EntryStarship:
		return skill.InGroup(skill.GroupStarship)
	case career.EntrySoldier:
		return skill.InGroup(skill.GroupSoldier)
	case career.EntryAnySkill:
		return citizenLifeSkills()
	case career.EntryAnyKnowledge:
		return allKnowledges()
	default:
		// Characteristic, Major, Minor, Capital, and No Skill cells.
		return nil
	}
}

// mission runs the chart 09 Risk & Reward box: the two years completing
// the Mission.
func (*agentMechanics) mission(r *careerRun, cc string) (termOutcome, error) {
	var outcome termOutcome

	mod, err := chooseRiskMod(r, r.def.Cite)
	if err != nil {
		return outcome, err
	}

	value, ok := characteristicValue(&r.character.Characteristics, cc)
	if !ok {
		return outcome, fmt.Errorf("%w: %q", errUnknownCharacteristic, cc)
	}

	risk := r.roller.Check(2, value+mod)
	riskSeq := r.log.Throw(risk, riskMods(mod, 1), r.def.Cite+" (Risk vs "+cc+"+Mods)")

	if !risk.Success {
		died, disabled := r.injury(cc, mod, riskSeq,
			r.def.Cite+" (Risk Failure: reduce CC by negative Mods and Flux)")
		outcome.endCause = riskSeq

		if died {
			outcome.died = true

			return outcome, nil
		}

		outcome.endCareer = disabled
	}

	reward := r.roller.Check(2, value-mod)
	rewardSeq := r.log.Throw(reward, riskMods(mod, -1), r.def.Cite+" (Reward vs "+cc+"+ opposite sign Mods)")

	if !reward.Success {
		r.log.Consequence(ConsequenceEvent{Cause: rewardSeq, Kind: ConsequenceNoAward})

		return outcome, nil
	}

	outcome.success = true
	r.record.Commendations++

	// "If the Reward Roll Succeeds, subtract the Reward Roll from the
	// Controlling Characteristic (ignore any Mods) and record the
	// Commendation in the format shown on the Commendation Table:
	// <Undercover Career> Commendation-N. N= C-R" (chart F p. 91).
	//
	// The value uses the unmodified characteristic, as the page says,
	// and the raw Reward roll — the same shape as the Medals table's
	// "unmodified Reward roll" on the facing column.
	award := Award{
		Code:  r.record.UndercoverCareer,
		Name:  r.record.UndercoverCareer + " Commendation",
		Count: 1,
		Mod:   value - reward.Total,
	}
	r.record.CommendationAwards = append(r.record.CommendationAwards, award)

	r.log.Consequence(ConsequenceEvent{
		Cause: rewardSeq, Kind: ConsequenceCommendation,
		Career: r.def.Name, Skill: award.Name, Value: r.record.Commendations, Delta: award.Mod,
	})

	return outcome, nil
}
