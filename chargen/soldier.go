package chargen

// Soldier career mechanics (Book 1 chart 08, p. 82; prose pp. 65-66).
// "To Begin Str / Select Branch Soc / Risk & Reward C1 C3 C4 / Officer
// Promotion Soc* / Officer Commission C3 / Enlisted Promotion C3* /
// Continue C3" with "*+Medals and WB Mods" (chart 08 box A).
//
// The first of the Armed Forces, which "are Spacers, Soldiers, and
// Marines, with background information as Branch and Assignment" (p. 66).
// A branch is selected or rolled on entry and carries a Mod; four
// assignments are rolled each term and the highest of their Mods joins it.
// Both are "negative against the Risk Roll and positive against the Reward
// Roll" (p. 66), alongside the character's own Caution or Bravery.
//
// A Reward success consults the Imperial Medals table: "use the unmodified
// Reward roll to determine the Medal; Mod +1 if an Officer" (p. 66; the
// table is p. 70). Medals modify later promotion throws; Wound Badges do
// not (interpretation I-31, ERRATA.md).
//
// Deferred: the ANM School assignment's schooling, "resolved as Education"
// (chart 08), and the Major's Command College, both mid-career schooling
// that lands with Later Education (docs/PRD.md milestone 4); the Reserves
// (p. 66); the Service Academy's Officer1 entry linkage.

import (
	"fmt"
	"slices"

	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/medal"
)

// soldierMechanics is the Soldier careerMechanics implementation.
type soldierMechanics struct {
	rank   string
	branch career.Branch
}

// newSoldier is the Soldier careerRegistry entry.
//
//nolint:ireturn // The registry's function type returns the interface.
func newSoldier() (*career.Definition, careerMechanics, error) {
	def, err := career.Soldier()
	if err != nil {
		return nil, nil, fmt.Errorf("soldier career: %w", err)
	}

	return def, &soldierMechanics{}, nil
}

// begin rolls To Begin, then enters at the service's starting enlisted
// rank: "Armed Forces characters begin with enlisted rank (Army =
// Soldier1)" (p. 65). A failed attempt costs one year (p. 65).
func (m *soldierMechanics) begin(r *careerRun) (bool, error) {
	r.log.Step("Soldier: To Begin", r.def.Cite)

	check, value, err := chooseCheckCharacteristic(r, r.def.BeginChecks)
	if err != nil {
		return false, err
	}

	throw := r.roller.Check(2, value)
	seq := r.log.Throw(throw, nil, r.def.Cite+" (To Begin vs "+check+")")

	if !throw.Success {
		r.character.Age++
		r.log.Consequence(ConsequenceEvent{Cause: seq, Kind: ConsequenceYearsElapsed, Value: 1})
		r.log.Consequence(ConsequenceEvent{Cause: seq, Kind: ConsequenceCareerNotBegun, Career: r.def.Name})

		return false, nil
	}

	if err := m.enterRank(r, r.def.Ranks[0].ID, seq); err != nil {
		return false, err
	}

	return true, m.selectBranch(r)
}

// selectBranch applies "Determine Branch and Mod": the character checks
// the chart's characteristic to choose a branch and otherwise rolls one
// (p. 65 worked example: "He must roll Soc or less to select Branch ...
// and chooses Flight (otherwise a Flight School graduate does not
// automatically receive Branch= Flight)").
func (m *soldierMechanics) selectBranch(r *careerRun) error {
	forces := r.def.ArmedForces

	value, ok := characteristicValue(&r.character.Characteristics, forces.BranchCheck)
	if !ok {
		return fmt.Errorf("%w: %q", errUnknownCharacteristic, forces.BranchCheck)
	}

	throw := r.roller.Check(2, value)
	seq := r.log.Throw(throw, nil, forces.BranchCite+" (Select Branch vs "+forces.BranchCheck+")")

	if throw.Success {
		branch, err := m.chooseBranch(r)
		if err != nil {
			return err
		}

		return m.enterBranch(r, branch, seq)
	}

	roll := r.roller.Roll(1)
	seq = r.log.Roll(roll, forces.BranchCite)

	return m.enterBranch(r, forces.BranchAt(roll.Total+m.eduDM(r)), seq)
}

// eduDM is the chart's education modifier on the Branch and Operations
// rolls ("DM +2 if Edu 10+").
func (*soldierMechanics) eduDM(r *careerRun) int {
	forces := r.def.ArmedForces
	if r.character.Characteristics.Edu >= forces.EduDMAt {
		return forces.EduDM
	}

	return 0
}

// chooseBranch presents the distinct branches the table offers.
func (*soldierMechanics) chooseBranch(r *careerRun) (career.Branch, error) {
	forces := r.def.ArmedForces

	var (
		options  []string
		branches []career.Branch
		scores   []int
	)

	for _, branch := range forces.Branches {
		if slices.Contains(options, branch.Name) {
			continue
		}

		options = append(options, branch.Name)
		branches = append(branches, branch)
		scores = append(scores, branch.Mod)
	}

	chosen, _, err := choose(r.log, r.decider, Choice{
		ID:      ChooseBranch,
		Prompt:  "Select a Branch",
		Options: options,
		Scores:  scores,
		Cite:    forces.BranchCite,
	})
	if err != nil {
		return career.Branch{}, err
	}

	return branches[chosen], nil
}

// enterBranch records the branch and awards its automatic skill: "if
// Medical Branch= Medic-1; If Technical Branch= any Trade" (chart 08).
func (m *soldierMechanics) enterBranch(r *careerRun, branch career.Branch, cause int) error {
	m.branch = branch
	r.record.Branch = branch.Name

	r.log.Consequence(ConsequenceEvent{
		Cause: cause, Kind: ConsequenceBranchSet, Career: r.def.Name, Skill: branch.Name,
	})

	if branch.AutoSkill != "" {
		name, err := r.resolveSkillName(branch.AutoSkill)
		if err != nil {
			return err
		}

		r.awardAndLog(name, 1, cause)
	}

	if branch.AutoTrade {
		return r.awardFromGroup(career.EntryTrade)
	}

	return nil
}

// enterRank records a rank and awards its Auto Skill (chart 08 table B:
// "Automatic Skills by Rank").
func (m *soldierMechanics) enterRank(r *careerRun, id string, cause int) error {
	rank, ok := r.def.RankByID(id)
	if !ok {
		return fmt.Errorf("%w: %q", errUnknownRank, id)
	}

	m.rank = rank.ID
	r.record.Rank = rank.ID
	r.record.RankTitle = rank.Title

	r.log.Consequence(ConsequenceEvent{
		Cause: cause, Kind: ConsequenceRankSet, Career: r.def.Name, Skill: rank.Title,
	})

	if rank.AutoSkill == "" {
		return nil
	}

	name, err := r.resolveSkillName(rank.AutoSkill)
	if err != nil {
		return err
	}

	r.awardAndLog(name, 1, cause)

	return nil
}

// resolveTerm rolls the term's assignments, runs Risk & Reward with their
// Mods, then the commission and promotion attempts.
func (m *soldierMechanics) resolveTerm(r *careerRun, cc string) (termOutcome, error) {
	columns, opsMod := m.operations(r)

	outcome, err := m.riskAndReward(r, cc, opsMod)
	if err != nil || outcome.died {
		return outcome, err
	}

	gained, err := m.advance(r)
	if err != nil {
		return termOutcome{}, err
	}

	outcome.skillRolls = r.def.SkillsPerTerm
	outcome.termColumns = columns
	outcome.bonusRolls = gained * r.def.SkillsPerAdvancement

	return outcome, nil
}

// operations rolls the term's assignments and reports the columns they
// open and the highest of their Mods: "Roll for Assignment four times per
// Term ... Determine the highest value for the Term: applied to Risk and
// Reward" (p. 66).
func (m *soldierMechanics) operations(r *careerRun) ([]string, int) {
	forces := r.def.ArmedForces

	dm := m.eduDM(r)
	if forces.OperationsUseBranchDM {
		dm += m.branch.DM
	}

	// "Column 1-Personal Skills may always be rolled" (p. 65).
	columns := []string{r.def.SkillColumns[0].Name}
	highest := 0

	for range forces.OperationsPerTerm {
		roll := r.roller.Roll(1)
		seq := r.log.Roll(roll, forces.OperationsCite)

		operation := forces.OperationAt(roll.Total + dm)

		r.log.Consequence(ConsequenceEvent{
			Cause: seq, Kind: ConsequenceOperation,
			Career: r.def.Name, Skill: operation.Name, Value: operation.Mod,
		})

		highest = max(highest, operation.Mod)

		if r.columnIndex(operation.Name) >= 0 && !slices.Contains(columns, operation.Name) {
			columns = append(columns, operation.Name)
		}
	}

	return columns, highest
}

// riskAndReward runs the chart 08 box with the Branch and Operations Mods,
// which are "negative against the Risk Roll and positive against the
// Reward Roll" (p. 66).
func (m *soldierMechanics) riskAndReward(r *careerRun, cc string, opsMod int) (termOutcome, error) {
	var outcome termOutcome

	caution, err := chooseRiskMod(r, "chart 08 p. 82")
	if err != nil {
		return outcome, err
	}

	value, ok := characteristicValue(&r.character.Characteristics, cc)
	if !ok {
		return outcome, fmt.Errorf("%w: %q", errUnknownCharacteristic, cc)
	}

	service := m.branch.Mod + opsMod

	mods := riskMods(caution, 1)
	mods = append(mods,
		Mod{Name: "Branch", Value: -m.branch.Mod},
		Mod{Name: "Operations", Value: -opsMod})

	risk := r.roller.Check(2, value+caution-service)
	riskSeq := r.log.Throw(risk, mods, r.def.Cite+" (Risk vs "+cc+"+Mods)")

	if risk.Success {
		// "Success: Receive XS Exemplary Service Badge. Character is
		// unharmed." (chart 08; interpretation I-32, ERRATA.md.)
		r.record.ServiceBadges++
		r.log.Consequence(ConsequenceEvent{
			Cause: riskSeq, Kind: ConsequenceServiceBadge, Career: r.def.Name, Value: r.record.ServiceBadges,
		})
	} else {
		died, disabled := r.injury(cc, caution-service, riskSeq,
			"Book 1 p. 82 chart 08 (Failure: reduce CC by negative Mods and Flux)")
		if died {
			outcome.died = true

			return outcome, nil
		}

		outcome.endCareer = disabled
	}

	rewardMods := riskMods(caution, -1)
	rewardMods = append(rewardMods,
		Mod{Name: "Branch", Value: m.branch.Mod},
		Mod{Name: "Operations", Value: opsMod})

	reward := r.roller.Check(2, value-caution+service)
	rewardSeq := r.log.Throw(reward, rewardMods, r.def.Cite+" (Reward vs "+cc+"+ opposite sign Mods)")

	if !reward.Success {
		r.log.Consequence(ConsequenceEvent{Cause: rewardSeq, Kind: ConsequenceNoAward})

		return outcome, nil
	}

	outcome.success = true

	return outcome, m.awardMedal(r, reward.Total, rewardSeq)
}

// awardMedal consults the p. 70 table: "use the unmodified Reward roll to
// determine the Medal; Mod +1 if an Officer" (p. 66).
func (m *soldierMechanics) awardMedal(r *careerRun, reward, cause int) error {
	won, err := medal.For(reward, m.isOfficer(r))
	if err != nil {
		return fmt.Errorf("soldier medal: %w", err)
	}

	index := slices.IndexFunc(r.record.Medals, func(a Award) bool { return a.Code == won.Code })
	if index < 0 {
		r.record.Medals = append(r.record.Medals, Award{Code: won.Code, Name: won.Name, Mod: won.Mod})
		index = len(r.record.Medals) - 1
	}

	r.record.Medals[index].Count++

	r.log.Consequence(ConsequenceEvent{
		Cause: cause, Kind: ConsequenceMedal, Career: r.def.Name,
		Skill: won.Code, Value: r.record.Medals[index].Count,
	})

	return nil
}

// isOfficer reports whether the current rank is on the officer ladder.
func (m *soldierMechanics) isOfficer(r *careerRun) bool {
	rank, ok := r.def.RankByID(m.rank)

	return ok && rank.Class == "officer"
}

// medalMod sums the promotion modifiers the character's medals carry:
// "Medals (but not Wound Badges) are Mods for Soldier / Spacer / Marine
// Promotion" (p. 70).
func medalMod(record *CareerRecord) int {
	total := 0
	for _, award := range record.Medals {
		total += award.Count * award.Mod
	}

	return total
}

// advance runs the term's commission and promotion attempts against the
// rank class held at the start of the phase, following the chart 06
// precedent where chart 08 is silent (interpretation I-13's rule).
func (m *soldierMechanics) advance(r *careerRun) (int, error) {
	entry, ok := r.def.RankByID(m.rank)
	if !ok {
		return 0, fmt.Errorf("%w: %q", errUnknownRank, m.rank)
	}

	gained := 0

	for _, advancement := range r.def.Advancements {
		current, ok := r.def.RankByID(m.rank)
		if !ok {
			return 0, fmt.Errorf("%w: %q", errUnknownRank, m.rank)
		}

		if !openToClass(advancement, entry) || !eligibleForAdvancement(advancement, current, r.def) {
			continue
		}

		promoted, err := m.attempt(r, advancement, current)
		if err != nil {
			return 0, err
		}

		if promoted {
			gained++
		}
	}

	return gained, nil
}

// attempt offers and rolls one commission or promotion.
func (m *soldierMechanics) attempt(r *careerRun, a career.Advancement, rank career.Rank) (bool, error) {
	value, ok := characteristicValue(&r.character.Characteristics, a.Check)
	if !ok {
		return false, fmt.Errorf("%w: %q", errUnknownCharacteristic, a.Check)
	}

	target := value

	var mods []Mod

	if a.MedalMods {
		if bonus := medalMod(&r.record); bonus != 0 {
			target += bonus
			mods = []Mod{{Name: "Medals", Value: bonus}}
		}
	}

	chosen, _, err := choose(r.log, r.decider, Choice{
		ID:      ChooseAdvancement,
		Prompt:  "Attempt " + a.Name + "?",
		Options: []string{"Attempt " + a.Name, "Decline"},
		Cite:    r.def.Cite + " (" + a.Name + ")",
	})
	if err != nil || chosen != 0 {
		return false, err
	}

	throw := r.roller.Check(2, target)
	seq := r.log.Throw(throw, mods, r.def.Cite+" ("+a.Name+" vs "+a.Check+")")

	if !throw.Success {
		return false, nil
	}

	next := a.ToRank
	if next == "" {
		promoted, ok := r.def.NextRank(rank.ID)
		if !ok {
			return false, nil
		}

		next = promoted.ID
	}

	return true, m.enterRank(r, next, seq)
}
