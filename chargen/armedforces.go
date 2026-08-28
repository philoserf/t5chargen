package chargen

// The Armed Forces career mechanics, shared by the careers Book 1 groups
// as "Spacers, Soldiers, and Marines, with background information as
// Branch and Assignment" (p. 66): chart 07 p. 81, chart 08 p. 82, and
// chart 12 p. 86. Everything that differs between them — the checks, the
// Branch and Operations tables, the rank ladders — is chart data; this
// file is the procedure they share.
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
// The ANM School assignment and the Command College a flag-rank footnote
// sends its officers to are both resolved as Education, in
// assignedschool.go.
//
// A Service Academy graduate enters his own service at its first officer
// rank rather than at the enlisted rank p. 65 gives every other recruit
// (interpretation I-94; entryRank, academy.go).
//

import (
	"fmt"
	"slices"

	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/medal"
)

// armedForcesMechanics is the careerMechanics implementation shared by
// the Armed Forces careers.
type armedForcesMechanics struct {
	rank string

	// commandCollege is set when the rank just entered carries the
	// flag-rank footnote, and read at the beginning of the next term:
	// "Command College in Year 1 of next Term (if Continue)" (chart 07's
	// Lt Commander, chart 08's Major, chart 12's Force Commander). It is
	// read in resolveTerm, so a term suspended for Later Education — "not
	// a term served" (I-90) — carries the college forward to the next
	// term actually served (interpretation I-91).
	commandCollege bool

	// branch is the row rolled or chosen; its officer or enlisted side is
	// resolved against the current rank, so a commission moves a Naval
	// rating from Crew to Line without a second roll (p. 66).
	branch career.Branch
}

// branchSide returns the branch name and Mod for the rank now held.
func (m *armedForcesMechanics) branchSide(r *careerRun) (string, int) {
	return m.branch.Side(m.isOfficer(r))
}

// newSoldier, newSpacer, and newMarine are the Armed Forces
// careerRegistry entries.
//
//nolint:ireturn // The registry's function type returns the interface.
func newSoldier() (*career.Definition, careerMechanics, error) {
	return newArmedForces(career.Soldier)
}

//nolint:ireturn // The registry's function type returns the interface.
func newSpacer() (*career.Definition, careerMechanics, error) {
	return newArmedForces(career.Spacer)
}

//nolint:ireturn // The registry's function type returns the interface.
func newMarine() (*career.Definition, careerMechanics, error) {
	return newArmedForces(career.Marine)
}

// newArmedForces builds the shared mechanics over one service's chart.
//
//nolint:ireturn // The registry's function type returns the interface.
func newArmedForces(load func() (*career.Definition, error)) (*career.Definition, careerMechanics, error) {
	def, err := load()
	if err != nil {
		return nil, nil, fmt.Errorf("armed forces career: %w", err)
	}

	return def, &armedForcesMechanics{}, nil
}

// begin rolls To Begin, then enters at the service's starting enlisted
// rank: "Armed Forces characters begin with enlisted rank (Army =
// Soldier1)" (p. 65). A failed attempt costs one year (p. 65).
//
// A Service Academy graduate of this service makes no throw: he was
// accepted years ago, and graduating is what turned that acceptance into
// his first term (interpretation I-101).
func (m *armedForcesMechanics) begin(r *careerRun) (bool, error) {
	if r.character.academyOfficer(r.def.ArmedForces.Service) {
		return m.beginOnCommission(r)
	}

	r.log.Step(r.def.Name+": To Begin", r.def.Cite)

	check, value, err := chooseCheckCharacteristic(r, r.def.BeginChecks)
	if err != nil {
		return false, err
	}

	throw := r.roller.Check(2, value)
	seq := r.log.Throw(throw, nil, r.def.Cite+" (To Begin vs "+check+")")

	if !throw.Success {
		return failedToBegin(r, seq)
	}

	if err := m.enterRank(r, m.entryRank(r), seq); err != nil {
		return false, err
	}

	return true, m.selectBranch(r)
}

// beginOnCommission enters the service a Service Academy graduate was
// commissioned into, without a To Begin throw.
//
// "Service Academies ... provide graduates an Army or Navy Commission ...
// The character is required to serve one term in the service" (p. 62).
// Acceptance to the Academy is acceptance to that term: a service does not
// train an officer for four years and then ask him to roll for a place in
// it, and the obligation I-99 collects would otherwise be one the character
// could fail to discharge through no decision of his own — which is what
// happened, to sixty-five of a hundred and sixty-three Army graduates in a
// three-hundred-seed sweep.
//
// A cadet who failed out is not covered. He holds no commission, owes no
// term, and applies to whatever career he likes on the same terms as
// anyone else (interpretation I-101).
func (m *armedForcesMechanics) beginOnCommission(r *careerRun) (bool, error) {
	seq := r.log.Step(r.def.Name+": Accepted on his Academy commission",
		"Book 1 p. 62 (required to serve one term in the service)")

	if err := m.enterRank(r, m.entryRank(r), seq); err != nil {
		return false, err
	}

	return true, m.selectBranch(r)
}

// entryRank is the rank the character joins at. Normally the first of the
// ladder — "Armed Forces characters begin with enlisted rank (Army =
// Soldier1)" (p. 65) — but a Service Academy graduate of this service
// enters as an officer instead: chart C p. 60 gives the Service Academy a
// Graduation of "C5=8 BA Officer1", and Officer1 is the first officer rank
// of the service he trained for (interpretation I-94).
func (m *armedForcesMechanics) entryRank(r *careerRun) string {
	if !r.character.academyOfficer(r.def.ArmedForces.Service) {
		return r.def.Ranks[0].ID
	}

	for _, rank := range r.def.Ranks {
		if rank.Class == officerRankClass {
			return rank.ID
		}
	}

	return r.def.Ranks[0].ID
}

// selectBranch applies "Determine Branch and Mod": the character checks
// the chart's characteristic to choose a branch and otherwise rolls one
// (p. 66 worked example: "He must roll Soc or less to select Branch ...
// and chooses Flight (otherwise a Flight School graduate does not
// automatically receive Branch= Flight)").
func (m *armedForcesMechanics) selectBranch(r *careerRun) error {
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
func (*armedForcesMechanics) eduDM(r *careerRun) int {
	forces := r.def.ArmedForces
	if r.character.Characteristics.Edu >= forces.EduDMAt {
		return forces.EduDM
	}

	return 0
}

// chooseBranch presents the distinct branches the table offers.
func (m *armedForcesMechanics) chooseBranch(r *careerRun) (career.Branch, error) {
	forces := r.def.ArmedForces
	officer := m.isOfficer(r)

	var (
		options  []string
		branches []career.Branch
		scores   []int
	)

	for _, branch := range forces.Branches {
		// A branch is selected on the side the character will serve
		// (interpretation I-36, ERRATA.md). On entry that is normally
		// the enlisted side — "Armed Forces characters begin with
		// enlisted rank" (p. 65) — so an entering Spacer picks among
		// Crew and Engineer, not Line and Flight. The one entrant who is
		// already commissioned is the Service Academy graduate of I-94,
		// and the officer side is the side he will serve on.
		name, mod := branch.Side(officer)

		// The score pairs the Branch Mod with its DM so a policy can
		// break ties on the DM, which decides how far the Operations
		// roll is pushed down its table (POLICY.md).
		score := mod*branchDMRange + branch.DM

		if at := slices.Index(options, name); at >= 0 {
			// Several rows can print the same name on the side being
			// selected and differ on the other: chart 07 prints enlisted
			// Engineer on row 3 (officer Line) and row 4 (officer
			// Engineer). A name binds to the row that reads the same on
			// both sides, so a character who "may roll for Branch or keep
			// his current Branch" (p. 66) keeps the branch he selected
			// instead of crossing into another one on commission
			// (interpretation I-36, ERRATA.md).
			if sameOnBothSides(branch) && !sameOnBothSides(branches[at]) {
				branches[at] = branch
				scores[at] = score
			}

			continue
		}

		options = append(options, name)
		branches = append(branches, branch)
		scores = append(scores, score)
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

// branchDMRange scales the Branch Mod above every Branch DM, so a score
// orders by Mod first and DM second.
const branchDMRange = 100

// sameOnBothSides reports whether a Branch row reads the same name for an
// officer and for an enlisted character, which is every row of a chart
// that prints one set (chart 08, chart 12) and the rows chart 07 prints
// identically on both sides.
func sameOnBothSides(b career.Branch) bool {
	officerName, _ := b.Side(true)
	enlistedName, _ := b.Side(false)

	return officerName == enlistedName
}

// enterBranch records the branch and awards its automatic skill: "if
// Medical Branch= Medic-1; If Technical Branch= any Trade" (chart 08).
func (m *armedForcesMechanics) enterBranch(r *careerRun, branch career.Branch, cause int) error {
	m.branch = branch

	name, _ := m.branchSide(r)
	r.record.Branch = name

	r.log.Consequence(ConsequenceEvent{
		Cause: cause, Kind: ConsequenceBranchSet, Career: r.def.Name, Skill: name,
	})

	if branch.AutoSkill != "" {
		name, err := r.resolveSkillName(branch.AutoSkill)
		if err != nil {
			return err
		}

		if err := r.awardAndLog(name, 1, cause); err != nil {
			return err
		}
	}

	if branch.AutoTrade {
		return r.awardFromGroup(career.EntryTrade)
	}

	return nil
}

// enterRank records a rank and awards its Auto Skill (chart 08 table B:
// "Automatic Skills by Rank").
func (m *armedForcesMechanics) enterRank(r *careerRun, id string, cause int) error {
	rank, ok := r.def.RankByID(id)
	if !ok {
		return fmt.Errorf("%w: %q", errUnknownRank, id)
	}

	// Captured before the rank moves: "Officers may not change Branch;
	// Enlisted may select a new Branch upon Promotion" turns on the side
	// he was on when the advancement was won, and a commission is
	// exactly the case where the two differ.
	wasOfficer := m.isOfficer(r)

	m.rank = rank.ID
	r.record.Rank = rank.ID
	r.record.RankTitle = rank.Title

	r.log.Consequence(ConsequenceEvent{
		Cause: cause, Kind: ConsequenceRankSet, Career: r.def.Name, Skill: rank.Title,
	})

	// The flag-rank footnote sends the officer to Command College next
	// term, not this one.
	if rank.CommandCollege {
		m.commandCollege = true
	}

	// A commission can move the character to the branch's officer side
	// ("for Spacers, Crew becomes Line", p. 66), so the branch follows the
	// rank as soon as it changes.
	if m.branch.Name != "" {
		name, _ := m.branchSide(r)
		if name != r.record.Branch {
			r.record.Branch = name

			r.log.Consequence(ConsequenceEvent{
				Cause: cause, Kind: ConsequenceBranchSet, Career: r.def.Name, Skill: name,
			})
		}
	}

	if err := m.offerBranchChange(r, wasOfficer); err != nil {
		return err
	}

	if rank.AutoSkill == "" {
		return nil
	}

	name, err := r.resolveSkillName(rank.AutoSkill)
	if err != nil {
		return err
	}

	if err := r.awardAndLog(name, 1, cause); err != nil {
		return err
	}

	return nil
}

// The two answers to a branch-change offer. Keeping is listed last, so
// the policy names it by position as it does elsewhere.
const (
	changeBranch = "Change Branch"
	keepBranch   = "Keep his current Branch"
)

// offerBranchChange offers the branch change an advancement entitles the
// character to. There are two, and the printed rules differ in kind about
// them.
//
// **On promotion, for an enlisted character.** All three Armed Forces
// charts print the same sentence: "Officers may not change Branch;
// Enlisted may select a new Branch upon Promotion" (charts 07 p. 81, 08
// p. 82, 12 p. 86). p. 66's prose says instead that "A non-officer
// character may change (reselect or reroll) Branch at the end of each
// Term". The charts win, being three statements agreeing with each other
// against one, and the narrower of the two (interpretation I-34). The
// change is the chart's own Select Branch procedure, which checks the
// characteristic and rolls on a failure — "select a new Branch" is what
// that procedure is called.
//
// **On commission.** "A character who receives a Commission may roll for
// Branch or keep his current Branch (for Spacers, Crew becomes Line)"
// (p. 66). Nothing disputes this one, and it is a roll rather than a
// selection — the page says roll. The parenthesis is the side-shift
// above, which happens either way.
//
// An officer promoted to a higher officer rank is offered nothing, all
// three charts saying he may not change.
func (m *armedForcesMechanics) offerBranchChange(r *careerRun, wasOfficer bool) error {
	// No branch yet: enterRank runs before selectBranch at career entry,
	// and the branch a character enters with is not a change.
	if wasOfficer || m.branch.Name == "" {
		return nil
	}

	forces := r.def.ArmedForces
	commissioned := m.isOfficer(r)

	prompt, cite := "Select a new Branch?",
		forces.BranchCite+" (Enlisted may select a new Branch upon Promotion)"
	if commissioned {
		prompt, cite = "Roll for a new Branch?",
			"Book 1 p. 66 (A character who receives a Commission may roll for Branch "+
				"or keep his current Branch)"
	}

	chosen, _, err := choose(r.log, r.decider, Choice{
		ID:      ChooseBranchChange,
		Prompt:  prompt,
		Options: []string{changeBranch, keepBranch},
		Cite:    cite,
	})
	if err != nil || chosen != 0 {
		return err
	}

	if !commissioned {
		return m.selectBranch(r)
	}

	roll := r.roller.Roll(1)
	seq := r.log.Roll(roll, cite)

	return m.enterBranch(r, forces.BranchAt(roll.Total+m.eduDM(r)), seq)
}

// resolveTerm rolls the term's assignments, runs Risk & Reward with their
// Mods, then the commission and promotion attempts.
func (m *armedForcesMechanics) resolveTerm(r *careerRun, cc string) (termOutcome, error) {
	// "Command College in Year 1 of next Term (if Continue)": the term
	// the footnote points at is this one, and Year 1 of it is here,
	// before the term's assignments are rolled. Reaching this term at all
	// is the "if Continue".
	if m.commandCollege {
		m.commandCollege = false

		if err := r.attendAssignedSchool(commandCollegeID); err != nil {
			return termOutcome{}, err
		}
	}

	if err := r.offerFlightSchool(); err != nil {
		return termOutcome{}, err
	}

	columns, opsMod, err := m.operations(r)
	if err != nil {
		return termOutcome{}, err
	}

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
func (m *armedForcesMechanics) operations(r *careerRun) ([]string, int, error) {
	forces := r.def.ArmedForces

	dm := m.eduDM(r)
	if forces.OperationsUseBranchDM {
		dm += m.branch.DM
	}

	// "Column 1-Personal Skills may always be rolled" (p. 65).
	columns := []string{r.def.SkillColumns[0].Name}
	highest := 0
	anm := false

	for range forces.OperationsPerTerm {
		roll := r.roller.Roll(1)
		seq := r.log.Roll(roll, forces.OperationsCite)

		operation := forces.OperationAt(roll.Total + dm)

		r.log.Consequence(ConsequenceEvent{
			Cause: seq, Kind: ConsequenceOperation,
			Career: r.def.Name, Skill: operation.Name, Value: operation.Mod,
		})

		highest = max(highest, operation.Mod)
		anm = anm || operation.Name == anmSchoolOperation

		column := operation.Column
		if column == "" {
			column = operation.Name
		}

		if r.columnIndex(column) >= 0 && !slices.Contains(columns, column) {
			columns = append(columns, column)
		}
	}

	// "Resolve ANM School using Education" (charts 07, 08, 12), after the
	// four assignment rolls rather than in the middle of them: the four
	// are one block that determines the term's assignments and its Mod
	// (p. 66), and interleaving a school's own throws would split it.
	// Once per term, however many of the four land on it — the school is
	// a year of the term, and chart C gives it one (interpretation I-93).
	if anm {
		if err := r.attendAssignedSchool(anmSchoolID); err != nil {
			return nil, 0, err
		}
	}

	return columns, highest, nil
}

// anmSchoolOperation is the assignment name the Operations tables print
// (charts 07, 08, 12).
const anmSchoolOperation = "ANM School"

// riskAndReward runs the chart 08 box with the Branch and Operations Mods,
// which are "negative against the Risk Roll and positive against the
// Reward Roll" (p. 66).
func (m *armedForcesMechanics) riskAndReward(r *careerRun, cc string, opsMod int) (termOutcome, error) {
	var outcome termOutcome

	caution, err := chooseRiskMod(r, r.def.Cite)
	if err != nil {
		return outcome, err
	}

	value, ok := characteristicValue(&r.character.Characteristics, cc)
	if !ok {
		return outcome, fmt.Errorf("%w: %q", errUnknownCharacteristic, cc)
	}

	_, branchMod := m.branchSide(r)
	service := branchMod + opsMod

	mods := riskMods(caution, 1)
	mods = append(mods,
		Mod{Name: "Branch", Value: -branchMod},
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
	} else if r.applyInjury(&outcome, cc, negativeMods(caution, branchMod, opsMod), riskSeq,
		r.def.Cite+" (Failure: reduce CC by negative Mods and Flux)") {
		return outcome, nil
	}

	rewardMods := riskMods(caution, -1)
	rewardMods = append(rewardMods,
		Mod{Name: "Branch", Value: branchMod},
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

// negativeMods sums only the Risk roll's negative modifiers, which are the
// ones a Risk failure charges to the Controlling Characteristic: "Reduce
// the Controlling Characteristic by all negative Mods; ignore any positive
// Mods" (p. 65). A Cautious Mod is positive on Risk and so contributes
// nothing; a Bravery Mod is negative and does. The Branch and Operations
// Mods are "negative against the Risk Roll" (p. 66), so their table values
// enter with the sign flipped.
//
// The p. 66 worked example is the discriminator: Eneri Dinsha fails Risk
// with End 11, Branch Mod -2, Operations Mod -3 and Caution Mod +2, and
// "his characteristic Endurance-11 reduces by -2 -3 to Endurance-6" — the
// Caution +2 does not offset the -5. Summing all four first would charge
// only -3 and, with his Flux of +4, leave him unharmed instead of wounded.
func negativeMods(caution, branchMod, opsMod int) int {
	return min(caution, 0) + min(-branchMod, 0) + min(-opsMod, 0)
}

// awardMedal consults the p. 70 table: "use the unmodified Reward roll to
// determine the Medal; Mod +1 if an Officer" (p. 66).
func (m *armedForcesMechanics) awardMedal(r *careerRun, reward, cause int) error {
	won, err := medal.For(reward, m.isOfficer(r))
	if err != nil {
		return fmt.Errorf("%s medal: %w", r.def.Name, err)
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
func (m *armedForcesMechanics) isOfficer(r *careerRun) bool {
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
func (m *armedForcesMechanics) advance(r *careerRun) (int, error) {
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
func (m *armedForcesMechanics) attempt(r *careerRun, a career.Advancement, rank career.Rank) (bool, error) {
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
