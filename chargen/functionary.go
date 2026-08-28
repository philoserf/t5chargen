package chargen

// Functionary career mechanics (Book 1 chart 13, p. 87).
//
// "The focus of every Functionary is Office Politics and its associated
// efforts to preserve and expand the power of a specific office. Each
// Functionary is charged with supervising or managing the operations of a
// bureaucracy. The natural consequence of Office Politics is promotion for
// Success and job loss for Failure."
//
// Box A: "To Begin Total Terms x3 / Office Politics C2 C3 C4 C5 /
// Continue Office Politics".
//
// Office Politics replaces both the Risk & Reward cycle and the Continue
// throw. The page prints these as separate lines, and the breaks matter —
// run together they read as though the Reward were nested inside a Risk
// success, which it is not (interpretation I-59):
//
//	Roll for Risk against CC. No Mods are used for Office Politics.
//	Risk Failure: Functionary career ends. The character may not Continue.
//	Risk Success: Functionary may continue in the career.
//	Roll for Reward against CC
//	Reward Failure: Functionary is not promoted.
//	Reward Success: Functionary is promoted one rank."
//
// "Functionary is never a first career" (box Not A First Career), and "a
// Noble may not become a Functionary".

import (
	"fmt"

	"github.com/philoserf/t5chargen/career"
)

// functionaryDirectorRank is the rank whose title the associated career
// renames: "Scholar F6 =College President" (chart 13).
const functionaryDirectorRank = "F6"

// functionaryUnderSecretary is the rank whose title carries a rolled
// ordinal: "F7 Nth UnderSecretary*", "* N= 1D (ie: 3rd UnderSecretary)".
const functionaryUnderSecretary = "F7"

// functionaryMechanics is the Functionary careerMechanics implementation.
type functionaryMechanics struct{ rank int }

// newFunctionary is the Functionary careerRegistry entry.
//
//nolint:ireturn // The registry's function type returns the interface.
func newFunctionary() (*career.Definition, careerMechanics, error) {
	def, err := career.Functionary()
	if err != nil {
		return nil, nil, fmt.Errorf("functionary career: %w", err)
	}

	if len(def.Ranks) == 0 {
		return nil, nil, fmt.Errorf("%w: functionary career has no ranks", errNotImplemented)
	}

	return def, &functionaryMechanics{}, nil
}

// begin resolves "To Begin Total Terms x3": the target is the terms the
// character has already served, in every career, times three. A character
// with no prior service throws against zero and cannot succeed, which is
// the same bar the chart states in words — "Functionary is never a first
// career" — and the reason the career is absent from career.FirstCareers.
//
// The character then names the career the position is associated with:
// "The Functionary character must identify with which prior career his
// position is associated." That association is not decoration; muster out
// reads it, adding a later Functionary's terms to the earlier career's
// benefit DM (p. 68).
func (m *functionaryMechanics) begin(r *careerRun) (bool, error) {
	r.log.Step("Functionary: To Begin", r.def.Cite)

	target := totalTerms(r.character) * r.def.BeginTotalTermsMultiplier

	throw := r.roller.Check(2, target)
	seq := r.log.Throw(throw, []Mod{{Name: "Total Terms x3", Value: target}},
		r.def.Cite+" (To Begin vs Total Terms x3)")

	if !throw.Success {
		return failedToBegin(r, seq)
	}

	if err := m.associate(r, seq); err != nil {
		return false, err
	}

	return true, m.enterRank(r, r.def.Ranks[0].ID, seq)
}

// resolveTerm runs Office Politics, which is the whole term: the Risk
// decides whether the career continues and the Reward whether the
// character is promoted (chart 13 box Office Politics).
func (m *functionaryMechanics) resolveTerm(r *careerRun, cc string) (termOutcome, error) {
	var outcome termOutcome

	value, ok := characteristicValue(&r.character.Characteristics, cc)
	if !ok {
		return outcome, fmt.Errorf("%w: %q", errUnknownCharacteristic, cc)
	}

	// "Roll for Risk against CC. No Mods are used for Office Politics."
	risk := r.roller.Check(2, value)
	riskSeq := r.log.Throw(risk, nil, r.def.Cite+" (Office Politics Risk vs "+cc+")")

	// The Risk decides whether the career goes on, so it is what the
	// term's years hang from whichever way it fell.
	outcome.endCause = riskSeq

	if !risk.Success {
		// "Risk Failure: Functionary career ends. The character may not
		// Continue." Job loss, not injury: the chart prints no
		// characteristic reduction.
		outcome.endCareer = true
	}

	reward := r.roller.Check(2, value)
	rewardSeq := r.log.Throw(reward, nil, r.def.Cite+" (Office Politics Reward vs "+cc+")")

	if !reward.Success {
		// "Reward Failure: Functionary is not promoted."
		r.log.Consequence(ConsequenceEvent{Cause: rewardSeq, Kind: ConsequenceNoAward})

		return outcome, nil
	}

	// "Reward Success: Functionary is promoted one rank." A Secretary is
	// already at the top of the ladder, so the success buys nothing — not
	// a rank, and not the "Per Promotion 1" eligibility that comes with
	// one (chart 13 B).
	promoted, err := m.promote(r, rewardSeq)
	if err != nil {
		return outcome, err
	}

	if !promoted {
		r.log.Consequence(ConsequenceEvent{Cause: rewardSeq, Kind: ConsequenceNoAward})

		return outcome, nil
	}

	outcome.success = true
	outcome.bonusRolls = r.def.SkillsPerAdvancement

	return outcome, nil
}

// associate records the prior career the position belongs to: "The
// Functionary position is usually associated with a prior career: a
// soldier finds a position in the civilian defense establishment; a
// scholar becomes an educational administrator" (chart 13).
func (m *functionaryMechanics) associate(r *careerRun, cause int) error {
	// Distinct careers, in the order first served. A character may serve
	// one career twice (interpretation I-54), and offering it twice would
	// print the same line twice with no way for a player to tell the two
	// apart — and POLICY.md promises first-listed is "the earliest career
	// served".
	options := make([]string, 0, len(r.character.Careers))
	seen := make(map[string]bool, len(r.character.Careers))

	for _, record := range r.character.Careers {
		if !record.Began || record.Career == r.def.Name || seen[record.Career] {
			continue
		}

		seen[record.Career] = true

		options = append(options, record.Career)
	}

	if len(options) == 0 {
		return fmt.Errorf("%w: a Functionary has no prior career to associate with", errNotImplemented)
	}

	chosen, _, err := choose(r.log, r.decider, Choice{
		ID:      ChooseAssociatedCareer,
		Prompt:  "Which prior career is the Functionary position associated with?",
		Options: options,
		Cite:    r.def.Cite + " (What Type Of Functionary?)",
	})
	if err != nil {
		return err
	}

	r.record.AssociatedCareer = options[chosen]
	r.log.Consequence(ConsequenceEvent{
		Cause: cause, Kind: ConsequenceAssociated, Career: r.def.Name, Skill: options[chosen],
	})

	return nil
}

// promote advances one rank and reports whether it could: the chart prints
// nine ranks and no rule for passing Secretary, so a Secretary stays one.
func (m *functionaryMechanics) promote(r *careerRun, cause int) (bool, error) {
	if m.rank+1 >= len(r.def.Ranks) {
		return false, nil
	}

	m.rank++

	return true, m.enterRank(r, r.def.Ranks[m.rank].ID, cause)
}

// enterRank records a rank, names it, and awards its Auto Skill.
func (m *functionaryMechanics) enterRank(r *careerRun, id string, cause int) error {
	rank, ok := r.def.RankByID(id)
	if !ok {
		return fmt.Errorf("%w: %q", errUnknownRank, id)
	}

	r.record.Rank = rank.ID
	r.record.RankTitle = m.rankTitle(r, rank)

	r.log.Consequence(ConsequenceEvent{
		Cause: cause, Kind: ConsequenceRankSet, Career: r.def.Name, Skill: r.record.RankTitle,
	})

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

// rankTitle names a rank, applying the two the chart varies. F6 takes the
// title of the associated career ("Scholar F6 =College President"), and F7
// takes a rolled ordinal ("* N= 1D (ie: 3rd UnderSecretary)").
func (m *functionaryMechanics) rankTitle(r *careerRun, rank career.Rank) string {
	if rank.ID == functionaryDirectorRank {
		if title, ok := r.def.DirectorTitles[r.record.AssociatedCareer]; ok {
			return title
		}

		return rank.Title
	}

	if rank.ID != functionaryUnderSecretary {
		return rank.Title
	}

	roll := r.roller.Roll(1)
	r.log.Roll(roll, r.def.Cite+" (N= 1D for the Nth UnderSecretary)")

	return ordinal(roll.Total) + " UnderSecretary"
}

// ordinal renders 1..6 as the chart's example does ("3rd UnderSecretary").
func ordinal(n int) string {
	switch n {
	case 1:
		return "1st"
	case 2:
		return "2nd"
	case 3:
		return "3rd"
	default:
		return fmt.Sprintf("%dth", n)
	}
}

// totalTerms counts the terms served across every career the character has
// begun, which is what chart 13's "Total Terms" reads.
func totalTerms(c *Character) int {
	terms := 0
	for _, record := range c.Careers {
		terms += len(record.Terms)
	}

	return terms
}
