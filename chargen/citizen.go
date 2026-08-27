package chargen

// Citizen career mechanics (Book 1 chart 04, p. 78; chart E1 panel 04,
// p. 72; prose pp. 65-66). "Begin Citizen Life is Automatic"; "The Citizen
// Career uses a variant of Risk and Reward called Citizen Life. Only one
// roll is made to determine Success or Failure. No Mods are used." (p. 65)
//
// Citizens have no rank ("The Citizen, Entertainer, Craftsman, Scout,
// Agent, and Rogue careers have no rank", p. 65). The generic term loop,
// controlling-characteristic rotation, skills table, and Continue throw
// live in careerrun.go.

import (
	"fmt"
	"slices"

	"github.com/philoserf/t5chargen/career"
)

// citizenMechanics is the Citizen careerMechanics implementation.
type citizenMechanics struct {
	// postSuccesses counts Citizen Life successes after Job and Hobby are
	// both determined: "In subsequent Terms, successes alternate between
	// Job or Hobby skill levels." (chart 04 p. 78)
	postSuccesses int
}

// newCitizen is the Citizen careerRegistry entry.
//
//nolint:ireturn // Registry constructors return the careerMechanics seam by design.
func newCitizen() (*career.Definition, careerMechanics, error) {
	def, err := career.Citizen()
	if err != nil {
		return nil, nil, fmt.Errorf("citizen career: %w", err)
	}

	return def, &citizenMechanics{}, nil
}

// begin records the automatic Begin: "Begin Citizen Life is Automatic"
// (chart E1 panel 04, p. 72).
func (*citizenMechanics) begin(r *careerRun) (bool, error) {
	r.log.Step("Citizen: Begin (automatic)", "Book 1 p. 72 chart E1 panel 04")

	return true, nil
}

// resolveTerm rolls the term's Citizen Life throw (2D <= CC, no mods,
// p. 65; chart 04 "Citizen Life C1 C2 C3 C4") and applies the success
// ladder.
func (m *citizenMechanics) resolveTerm(r *careerRun, cc string) (termOutcome, error) {
	value, ok := characteristicValue(&r.character.Characteristics, cc)
	if !ok {
		return termOutcome{}, fmt.Errorf("%w: %q", errUnknownCharacteristic, cc)
	}

	throw := r.roller.Check(2, value)
	seq := r.log.Throw(throw, nil, "Book 1 p. 78 chart 04 (Citizen Life vs "+cc+", no mods per p. 65)")

	if !throw.Success {
		// "If Citizen Life Fails... no Job or Hobby skills" (p. 78).
		r.log.Consequence(ConsequenceEvent{Cause: seq, Kind: ConsequenceNoAward})

		return termOutcome{}, nil
	}

	if err := m.awardCitizenLife(r, seq); err != nil {
		return termOutcome{}, err
	}

	return termOutcome{success: true}, nil
}

// awardCitizenLife applies the chart 04 success ladder: "First Success
// provides a Job ... with Skill-4 ... Second Success provides a Hobby ...
// with Skill-2 ... In subsequent Terms, successes alternate between Job or
// Hobby skill levels" (p. 78).
func (m *citizenMechanics) awardCitizenLife(r *careerRun, cause int) error {
	switch {
	case r.record.Job == "":
		return m.determineJob(r)
	case r.record.Hobby == "":
		return m.determineHobby(r)
	default:
		m.postSuccesses++

		// Third= Job-1, Fourth= Hobby-1, and alternating (chart 04).
		name := r.record.Job
		if m.postSuccesses%2 == 0 {
			name = r.record.Hobby
		}

		r.awardAndLog(name, 1, cause)
	}

	return nil
}

// determineJob rolls table E for the Job: "First Success provides a Job,
// randomly on Citizen Skills and Knowledges with Skill-4 (later receipts
// are Skill-1)." (p. 78) "Roll three dice for a specific Skill or
// Knowledge: Roll A (reroll if >3), then roll B, and finally top row C."
// (p. 78) Rerolled A faces are logged so the event log accounts for every
// consumed face.
//
// If the roll lands on the "No Skill" cell, the Job remains undetermined
// and the next success retries — an interpretation recorded in ERRATA.md.
func (*citizenMechanics) determineJob(r *careerRun) error {
	const cite = "Book 1 p. 78 chart 04 table E (roll A reroll if >3, then B, then C)"

	a := r.rollUnder(3, cite)

	b := r.roller.Roll(1)
	r.log.Roll(b, cite)

	c := r.roller.Roll(1)
	seq := r.log.Roll(c, cite)

	entry := r.def.JobEntry(a, b.Total, c.Total)
	if entry.Kind == career.EntryNone {
		r.log.Consequence(ConsequenceEvent{Cause: seq, Kind: ConsequenceJobUndetermined})

		return nil
	}

	name, err := r.resolveSkillName(entry.Name)
	if err != nil {
		return err
	}

	r.record.Job = name
	r.log.Consequence(ConsequenceEvent{Cause: seq, Kind: ConsequenceJobSet, Skill: name})
	r.awardAndLog(name, r.firstReceiptLevels(name, 4), seq)

	return nil
}

// determineHobby selects the Hobby: "Second Success provides a Hobby
// selected from Citizen Skills and Knowledges with Skill-2 (later receipts
// are Skill-1)." (p. 78) The alternatives are every table E skill in chart
// order, excluding the already-determined Job — the ladder alternates
// between two distinct pursuits (interpretation I-3, ERRATA.md). The
// hobby_set consequence and its award are caused by the selecting choice
// event (docs/PRD.md FR10).
func (*citizenMechanics) determineHobby(r *careerRun) error {
	options := r.def.HobbyChoices()
	if i := slices.Index(options, r.record.Job); i >= 0 {
		options = slices.Concat(options[:i], options[i+1:])
	}

	chosen, seq, err := choose(r.log, r.decider, Choice{
		ID:      ChooseHobby,
		Prompt:  "Select a Hobby from Citizen Skills and Knowledges",
		Options: options,
		Cite:    "Book 1 p. 78 chart 04 (Second Success provides a Hobby)",
	})
	if err != nil {
		return err
	}

	// The Job is excluded again here: table E prints the ambiguous "Grav"
	// and "Spacecraft" labels, so removing the Job by name above cannot
	// reach a Job that was itself resolved from one of them (ERRATA.md
	// I-3, I-10, I-11).
	name, err := r.resolveSkillName(options[chosen], r.record.Job)
	if err != nil {
		return err
	}

	r.record.Hobby = name
	r.log.Consequence(ConsequenceEvent{Cause: seq, Kind: ConsequenceHobbySet, Skill: name})
	r.awardAndLog(name, r.firstReceiptLevels(name, 2), seq)

	return nil
}
