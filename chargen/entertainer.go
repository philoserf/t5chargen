package chargen

// Entertainer career mechanics (Book 1 chart 03, p. 77; prose pp. 64-66).
// "Begin Actor C2 or C3 / Begin Artist C3 or Int / Begin Author Int or C5 /
// Begin Dancer C2 or C3 / Begin Musician C2 or C3 / Begin Chef C2 or Int /
// Risk & Reward Talent / Continue Fame" (chart 03 box A).
//
// The Entertainer runs no Risk and Reward throws. Book 1 lists it among
// the careers with a variant of that cycle: "The Citizen Career uses a
// variant of Risk and Reward called Citizen Life ... The Entertainer
// Career focuses on Fame and resolves the current level of Fame for the
// character." (p. 66) Box A's "Risk & Reward Talent" names Talent as the
// value governing that variant, and the chart prints no Risk or Reward
// outcome box — unlike every career that does roll them (interpretation
// I-18, ERRATA.md).
//
// The term's outcome is therefore whether Fame increased, which is what
// table B keys its extra skills to: "Per Term 4 / If Fame Increases 2 and
// Talent+1".
//
// Deferred: the stage names, which are flavour and which no chart
// prints a table for.

import (
	"fmt"

	"github.com/philoserf/t5chargen/career"
)

// entertainerMechanics is the Entertainer careerMechanics implementation.
type entertainerMechanics struct {
	// fame and talent are the career's tracked values (chart 03
	// Entertainer Fame And Talent).
	fame   int
	talent int
}

// newEntertainer is the Entertainer careerRegistry entry.
//
//nolint:ireturn // The registry's function type returns the interface.
func newEntertainer() (*career.Definition, careerMechanics, error) {
	def, err := career.Entertainer()
	if err != nil {
		return nil, nil, fmt.Errorf("entertainer career: %w", err)
	}

	return def, &entertainerMechanics{}, nil
}

// begin selects a specialty, rolls the shared Fame and Talent value, then
// rolls that specialty's To Begin check. "Before Begin (to evaluate
// potential for a career), roll initial Fame and Talent (with one 2D roll;
// they are equal)." (chart 03) A failed attempt costs one year (p. 65);
// chart 03 lists no Begin retry.
func (m *entertainerMechanics) begin(r *careerRun) (bool, error) {
	r.log.Step("Entertainer: To Begin", r.def.Cite)

	track, err := m.chooseSpecialty(r)
	if err != nil {
		return false, err
	}

	// The 2D is rolled before the To Begin check, as the chart says, so it
	// is consumed whether or not the career begins; its consequences are
	// recorded only once the character is an Entertainer.
	roll := r.roller.Roll(2)
	fameSeq := r.log.Roll(roll, r.def.Cite+" (initial Fame and Talent; they are equal)")

	check, value, err := chooseCheckCharacteristic(r, track.Checks)
	if err != nil {
		return false, err
	}

	throw := r.roller.Check(2, value)
	seq := r.log.Throw(throw, nil, r.def.Cite+" (Begin "+track.Name+" vs "+check+")")

	if !throw.Success {
		return failedToBegin(r, seq)
	}

	r.record.Specialty = track.Name
	r.log.Consequence(ConsequenceEvent{
		Cause: seq, Kind: ConsequenceSpecialtySet, Career: r.def.Name, Skill: track.Name,
	})

	m.setFame(r, roll.Total, fameSeq)
	m.setTalent(r, roll.Total, fameSeq)

	return true, nil
}

// chooseSpecialty selects the art the Entertainer practises.
func (*entertainerMechanics) chooseSpecialty(r *careerRun) (career.BeginTrack, error) {
	options := make([]string, len(r.def.BeginTracks))
	for i, track := range r.def.BeginTracks {
		options[i] = track.Name
	}

	chosen, _, err := choose(r.log, r.decider, Choice{
		ID:      ChooseSpecialty,
		Prompt:  "Select an Entertainer specialty",
		Options: options,
		Cite:    r.def.Cite + " (Select A Specialty)",
	})
	if err != nil {
		return career.BeginTrack{}, err
	}

	return r.def.BeginTracks[chosen], nil
}

// setFame records a new Fame value on the career. It is the career's own
// tracked value — its Continue target, and what chart F reads where it
// says "Entertainer detailed under Career" (p. 91) — not the character's
// Fame, which is calculated over the finished record.
func (m *entertainerMechanics) setFame(r *careerRun, value, cause int) {
	delta := value - m.fame
	m.fame = value
	r.record.Fame = value

	r.log.Consequence(ConsequenceEvent{
		Cause: cause, Kind: ConsequenceFameChange, Delta: delta, Value: value,
	})
}

// setTalent records a new Talent value.
func (m *entertainerMechanics) setTalent(r *careerRun, value, cause int) {
	m.talent = value
	r.record.Talent = value

	r.log.Consequence(ConsequenceEvent{
		Cause: cause, Kind: ConsequenceTalentSet, Value: value,
	})
}

// resolveTerm resolves the term's Fame, which is the Entertainer's variant
// of Risk and Reward (p. 66). The first term uses the Fame rolled before
// Begin — the chart's table reads "Term 1 =2D" with no Flux — so only
// later terms can raise Fame (interpretation I-19, ERRATA.md).
func (m *entertainerMechanics) resolveTerm(r *careerRun, _ string) (termOutcome, error) {
	var outcome termOutcome

	if len(r.record.Terms) == 0 {
		outcome.skillRolls = r.def.SkillsPerTerm

		return outcome, nil
	}

	increased, cause, err := m.resolveFame(r)
	if err != nil {
		return termOutcome{}, err
	}

	outcome.success = increased
	outcome.skillRolls = r.def.SkillsPerTerm

	if increased {
		// "If Fame Increases 2 and Talent+1" (chart 03 table B), caused by
		// the roll that raised Fame (docs/PRD.md FR10).
		outcome.skillRolls += entertainerFameSkills

		m.setTalent(r, m.talent+1, cause)
	}

	return outcome, nil
}

// entertainerFameSkills is the extra table C eligibility a Fame increase
// earns (chart 03 table B: "If Fame Increases 2").
const entertainerFameSkills = 2

// resolveFame runs the term's Fame determination and reports whether Fame
// increased: a Comeback, or the required Flux followed by up to two
// optional ones ("Roll Flux up to three times (the first is required; the
// second and third are optional)", chart 03).
func (m *entertainerMechanics) resolveFame(r *careerRun) (bool, int, error) {
	before := m.fame

	comeback, seq, err := m.offerComeback(r)
	if err != nil {
		return false, 0, err
	}

	if comeback {
		// "Comeback: Reset Fame to 2D; Talent is unchanged." (chart 03)
		// A Comeback that lands above the old Fame is still not a Fame
		// increase for table B: the chart denies the Talent the increase
		// would carry, and table B pairs that Talent with its two extra
		// skills (interpretation I-20, ERRATA.md).
		return false, seq, nil
	}

	flux := r.roller.Flux()
	seq = r.log.Flux(flux, r.def.Cite+" (Fame +F; the first Flux is required)")
	m.setFame(r, m.fame+flux.Value, seq)

	for attempt := range entertainerOptionalFlux {
		if m.fame > before {
			// The optional rolls are the character's to take; once Fame
			// has risen, another Flux would put the increase — and the
			// Talent it earns — back at risk. Below the starting Fame
			// they remain a gamble, not free upside: Flux is 1D-1D, so a
			// further roll can also sink Fame lower still (POLICY.md).
			break
		}

		again, err := m.offerOptionalFlux(r, attempt)
		if err != nil {
			return false, 0, err
		}

		if !again {
			break
		}

		flux = r.roller.Flux()
		seq = r.log.Flux(flux, r.def.Cite+" (Fame +F*; optional Flux)")
		m.setFame(r, m.fame+flux.Value, seq)
	}

	return m.fame > before, seq, nil
}

// entertainerOptionalFlux is the number of optional Flux rolls the chart
// allows after the required one ("+F* +F*").
const entertainerOptionalFlux = 2

// offerComeback offers the Comeback: "Reset Fame to 2D; Talent is
// unchanged. Comeback is possible any number of times." (chart 03) It
// replaces the term's Flux progression, and the term earns neither the
// Talent nor the extra skills a Fame increase would (interpretation I-20,
// ERRATA.md).
func (m *entertainerMechanics) offerComeback(r *careerRun) (bool, int, error) {
	chosen, choiceSeq, err := choose(r.log, r.decider, Choice{
		ID:      ChooseComeback,
		Prompt:  "Attempt a Comeback (reset Fame to 2D)?",
		Options: []string{"Comeback", "Continue the current run"},
		Scores:  []int{m.fame, 0},
		Cite:    r.def.Cite + " (Comeback: Reset Fame to 2D)",
	})
	if err != nil {
		return false, 0, err
	}

	if chosen != 0 {
		return false, choiceSeq, nil
	}

	roll := r.roller.Roll(2)
	seq := r.log.Roll(roll, r.def.Cite+" (Comeback: Reset Fame to 2D)")

	r.log.Consequence(ConsequenceEvent{Cause: seq, Kind: ConsequenceComeback, Value: roll.Total})
	m.setFame(r, roll.Total, seq)

	return true, seq, nil
}

// offerOptionalFlux offers one of the two optional Fame Flux rolls.
func (*entertainerMechanics) offerOptionalFlux(r *careerRun, attempt int) (bool, error) {
	chosen, _, err := choose(r.log, r.decider, Choice{
		ID:      ChooseOptionalFlux,
		Prompt:  "Roll the optional Fame Flux?",
		Options: []string{"Roll Flux", "Stand pat"},
		Scores:  []int{attempt, 0},
		Cite:    r.def.Cite + " (the second and third Flux rolls are optional)",
	})
	if err != nil {
		return false, err
	}

	return chosen == 0, nil
}
