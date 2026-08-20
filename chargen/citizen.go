package chargen

// Citizen career resolution (Book 1 chart 04, p. 78; chart E1 panel 04,
// p. 72; prose pp. 65-66). "Begin Citizen Life is Automatic"; "The Citizen
// Career uses a variant of Risk and Reward called Citizen Life. Only one
// roll is made to determine Success or Failure. No Mods are used." (p. 65)
//
// Citizens have no rank ("The Citizen, Entertainer, Craftsman, Scout,
// Agent, and Rogue careers have no rank", p. 65). Muster out, aging, and
// career changes are deferred to docs/PRD.md milestone 4.

import (
	"errors"
	"fmt"
	"slices"
	"strconv"

	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/dice"
)

// citizenRun is the state of one Citizen career resolution.
type citizenRun struct {
	def       *career.Definition
	roller    *dice.Roller
	log       *Log
	decider   Decider
	character *Character

	// availableCCs rotates the controlling characteristic: "This
	// Controlling Characteristic cannot be used again until all of the
	// others in the sequence have been used." (p. 65)
	availableCCs []string

	// postSuccesses counts Citizen Life successes after Job and Hobby are
	// both determined: "In subsequent Terms, successes alternate between
	// Job or Hobby skill levels." (chart 04 p. 78)
	postSuccesses int

	record CareerRecord
}

// errUnknownCharacteristic reports a data-file characteristic name outside
// the six standard abbreviations.
var errUnknownCharacteristic = errors.New("unknown characteristic")

// errNotImplemented reports a table cell whose resolution lands in a later
// milestone (docs/PRD.md milestones 2-3).
var errNotImplemented = errors.New("not implemented until education/skill milestones")

// runCitizen resolves a full Citizen career, term by term, until the
// Continue roll fails.
func runCitizen(roller *dice.Roller, log *Log, decider Decider, character *Character) error {
	def, err := career.Citizen()
	if err != nil {
		return fmt.Errorf("citizen career: %w", err)
	}

	run := &citizenRun{
		def:       def,
		roller:    roller,
		log:       log,
		decider:   decider,
		character: character,
		record:    CareerRecord{Career: "Citizen"},
	}

	// "Begin Citizen Life is Automatic" (chart E1 panel 04, p. 72).
	log.Step("Citizen: Begin (automatic)", "Book 1 p. 72 chart E1 panel 04")

	for {
		continued, err := run.term(len(run.record.Terms) + 1)
		if err != nil {
			return err
		}

		if !continued {
			break
		}
	}

	character.Careers = append(character.Careers, run.record)

	return nil
}

// term resolves one 4-year term and reports whether the career continues.
func (r *citizenRun) term(number int) (bool, error) {
	r.log.Step("Citizen: Term "+strconv.Itoa(number), r.def.Cite)

	cc, err := r.chooseCC()
	if err != nil {
		return false, err
	}

	success, err := r.citizenLife(cc)
	if err != nil {
		return false, err
	}

	if err := r.termSkills(); err != nil {
		return false, err
	}

	continued := r.continueRoll()
	r.record.Terms = append(r.record.Terms, TermRecord{
		Term:                      number,
		ControllingCharacteristic: cc,
		Success:                   success,
		Continued:                 continued,
	})

	return continued, nil
}

// chooseCC selects the term's controlling characteristic: "The player
// picks one of these Characteristics (any one anywhere in the sequence)
// ... This Controlling Characteristic cannot be used again until all of
// the others in the sequence have been used" (p. 65).
func (r *citizenRun) chooseCC() (string, error) {
	if len(r.availableCCs) == 0 {
		r.availableCCs = slices.Clone(r.def.CitizenLifeCharacteristics)
	}

	scores := make([]int, len(r.availableCCs))
	for i, name := range r.availableCCs {
		scores[i], _ = characteristicValue(&r.character.Characteristics, name)
	}

	chosen, _, err := choose(r.log, r.decider, Choice{
		ID:      ChooseControllingCharacteristic,
		Prompt:  "Select the term's controlling characteristic",
		Options: slices.Clone(r.availableCCs),
		Scores:  scores,
		Cite:    "Book 1 p. 65 (Risk and Reward: Select the CC)",
	})
	if err != nil {
		return "", err
	}

	cc := r.availableCCs[chosen]
	r.availableCCs = slices.Delete(r.availableCCs, chosen, chosen+1)

	return cc, nil
}

// citizenLife rolls the term's Citizen Life throw (2D <= CC, no mods,
// p. 65; chart 04 "Citizen Life C1 C2 C3 C4") and applies the success
// ladder.
func (r *citizenRun) citizenLife(cc string) (bool, error) {
	value, ok := characteristicValue(&r.character.Characteristics, cc)
	if !ok {
		return false, fmt.Errorf("%w: %q", errUnknownCharacteristic, cc)
	}

	throw := r.roller.Throw(2, value)
	seq := r.log.Throw(throw, nil, "Book 1 p. 78 chart 04 (Citizen Life vs "+cc+", no mods per p. 65)")

	if !throw.Success {
		// "If Citizen Life Fails... no Job or Hobby skills" (p. 78).
		r.log.Consequence(ConsequenceEvent{Cause: seq, Kind: ConsequenceNoAward})

		return false, nil
	}

	if err := r.awardCitizenLife(seq); err != nil {
		return false, err
	}

	return true, nil
}

// awardCitizenLife applies the chart 04 success ladder: "First Success
// provides a Job ... with Skill-4 ... Second Success provides a Hobby ...
// with Skill-2 ... In subsequent Terms, successes alternate between Job or
// Hobby skill levels" (p. 78).
func (r *citizenRun) awardCitizenLife(cause int) error {
	switch {
	case r.record.Job == "":
		r.determineJob()
	case r.record.Hobby == "":
		return r.determineHobby()
	default:
		r.postSuccesses++

		// Third= Job-1, Fourth= Hobby-1, and alternating (chart 04).
		name := r.record.Job
		if r.postSuccesses%2 == 0 {
			name = r.record.Hobby
		}

		r.awardAndLog(name, 1, cause)
	}

	return nil
}

// awardAndLog awards skill levels (capped at SkillMax, p. 134) and emits
// the matching consequence: skill_awarded, or no_award if the cap absorbed
// the whole receipt.
func (r *citizenRun) awardAndLog(name string, levels, cause int) {
	level, applied := r.character.awardSkill(name, levels)
	if applied == 0 {
		r.log.Consequence(ConsequenceEvent{Cause: cause, Kind: ConsequenceNoAward, Skill: name})

		return
	}

	r.log.Consequence(ConsequenceEvent{
		Cause: cause,
		Kind:  ConsequenceSkillAwarded,
		Skill: name,
		Delta: applied,
		Value: level,
	})
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
func (r *citizenRun) determineJob() {
	const cite = "Book 1 p. 78 chart 04 table E (roll A reroll if >3, then B, then C)"

	a := r.roller.Roll(1)
	r.log.Roll(a, cite)

	for a.Total > 3 {
		a = r.roller.Roll(1)
		r.log.Roll(a, cite)
	}

	b := r.roller.Roll(1)
	r.log.Roll(b, cite)

	c := r.roller.Roll(1)
	seq := r.log.Roll(c, cite)

	entry := r.def.JobEntry(a.Total, b.Total, c.Total)
	if entry.Kind == career.EntryNone {
		r.log.Consequence(ConsequenceEvent{Cause: seq, Kind: ConsequenceJobUndetermined})

		return
	}

	r.record.Job = entry.Name
	r.log.Consequence(ConsequenceEvent{Cause: seq, Kind: ConsequenceJobSet, Skill: entry.Name})
	r.awardAndLog(entry.Name, r.firstReceiptLevels(entry.Name, 4), seq)
}

// determineHobby selects the Hobby: "Second Success provides a Hobby
// selected from Citizen Skills and Knowledges with Skill-2 (later receipts
// are Skill-1)." (p. 78) The alternatives are every table E skill in chart
// order, excluding the already-determined Job — the ladder alternates
// between two distinct pursuits (interpretation I-3, ERRATA.md). The
// hobby_set consequence and its award are caused by the selecting choice
// event (docs/PRD.md FR10).
func (r *citizenRun) determineHobby() error {
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

	name := options[chosen]
	r.record.Hobby = name
	r.log.Consequence(ConsequenceEvent{Cause: seq, Kind: ConsequenceHobbySet, Skill: name})
	r.awardAndLog(name, r.firstReceiptLevels(name, 2), seq)

	return nil
}

// firstReceiptLevels applies the first-receipt rule: the stated level on
// first receipt, Skill-1 thereafter ("with Skill-4 (later receipts are
// Skill-1)", p. 78). A skill already held from another source (table C)
// receives +1 — interpretation I-2, ERRATA.md.
func (r *citizenRun) firstReceiptLevels(name string, firstReceipt int) int {
	if r.character.skillLevel(name) > 0 {
		return 1
	}

	return firstReceipt
}

// termSkills rolls the per-term table C eligibility: "Per Term: 4 on Table
// C" (chart 04 table B); "For each skill, roll on the Career Skills Table.
// The character selects a column and rolls 1D for the specific skill"
// (p. 65).
func (r *citizenRun) termSkills() error {
	columns := r.def.SkillColumnNames()

	for range r.def.SkillsPerTerm {
		chosen, _, err := choose(r.log, r.decider, Choice{
			ID:      ChooseSkillColumn,
			Prompt:  "Select a Citizen Skills column",
			Options: columns,
			Cite:    "Book 1 p. 65 (the character selects a column and rolls 1D)",
		})
		if err != nil {
			return err
		}

		roll := r.roller.Roll(1)
		seq := r.log.Roll(roll, "Book 1 p. 78 chart 04 table C, column "+columns[chosen])

		entry := r.def.SkillColumns[chosen].Entries[roll.Total-1]
		if err := r.awardTableC(entry, seq); err != nil {
			return err
		}
	}

	return nil
}

// awardTableC applies one table C cell.
func (r *citizenRun) awardTableC(entry career.Entry, cause int) error {
	switch entry.Kind {
	case career.EntrySkill:
		r.awardAndLog(entry.Name, 1, cause)
	case career.EntryCharacteristic:
		value, ok := characteristicAdd(&r.character.Characteristics, entry.Name, 1)
		if !ok {
			return fmt.Errorf("%w: %q", errUnknownCharacteristic, entry.Name)
		}

		r.log.Consequence(ConsequenceEvent{
			Cause: cause, Kind: ConsequenceCharacteristicChange,
			Characteristic: entry.Name, Delta: 1, Value: value,
		})
	case career.EntryMajor, career.EntryMinor:
		// "If the character does not have a Major/Minor this benefit is
		// lost." (p. 78) Education lands with milestone 2.
		r.log.Consequence(ConsequenceEvent{Cause: cause, Kind: ConsequenceBenefitLost})
	case career.EntryNone:
		r.log.Consequence(ConsequenceEvent{Cause: cause, Kind: ConsequenceNoAward})
	case career.EntryTrade, career.EntryArt, career.EntryScience:
		return fmt.Errorf("%w: %q cell", errNotImplemented, entry.Kind)
	default:
		// The loader validates kinds, but a default keeps an unknown kind
		// from silently resolving to nothing (event-log-first contract).
		return fmt.Errorf("%w: unknown cell kind %q", errNotImplemented, entry.Kind)
	}

	return nil
}

// continueRoll rolls Continue: "Continue 10-" (chart 04); "the Character
// must successfully roll (2D) to Continue (or less) in the career. Failure
// ends Career Resolution. ... If the Continue roll is 2 exactly, the
// character is required to Continue" (p. 66). Each term elapses 4 years
// ("the 4-year Term", p. 66).
func (r *citizenRun) continueRoll() bool {
	throw := r.roller.Throw(2, r.def.ContinueTarget)
	seq := r.log.Throw(throw, nil, "Book 1 p. 78 chart 04 (Continue 10-; p. 66)")

	r.character.Age += TermYears
	r.log.Consequence(ConsequenceEvent{Cause: seq, Kind: ConsequenceYearsElapsed, Value: TermYears})

	if throw.Total == 2 {
		r.log.Consequence(ConsequenceEvent{Cause: seq, Kind: ConsequenceMandatoryContinue})

		return true
	}

	if !throw.Success {
		r.log.Consequence(ConsequenceEvent{Cause: seq, Kind: ConsequenceCareerEnded, Career: "Citizen"})

		return false
	}

	return true
}
