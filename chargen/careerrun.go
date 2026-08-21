package chargen

// The generic career runner: the term loop, controlling-characteristic
// rotation, career skills table, and Continue throw shared by every career
// (Book 1 chart D p. 64; prose pp. 65-66). Career-specific exceptional
// mechanics (docs/PRD.md, Architecture notes) plug in through the
// careerMechanics interface; Citizen's live in citizen.go.

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/dice"
)

// careerMechanics is one career's exceptional mechanics. The interface is
// unexported and grows with the careers that need more seams (rank,
// commission, muster out land with milestones 3-4).
type careerMechanics interface {
	// begin resolves career entry: automatic for Citizen, a To Begin
	// throw with retry for most careers (chart D p. 64; p. 65).
	begin(r *careerRun) error

	// resolveTerm runs the career's Risk/Reward variant for the term
	// (p. 65: Citizen Life for Citizens) and applies its awards,
	// reporting the term's success.
	resolveTerm(r *careerRun, cc string) (bool, error)
}

// careerRegistry maps canonical career names to their definition and
// mechanics. Its key set must match career.Available (tested).
var careerRegistry = map[string]func() (*career.Definition, careerMechanics, error){
	"Citizen": newCitizen,
}

// careerRun is the shared state of one career resolution.
type careerRun struct {
	def       *career.Definition
	mechanics careerMechanics
	roller    *dice.Roller
	log       *Log
	decider   Decider
	character *Character

	// availableCCs rotates the controlling characteristic: "This
	// Controlling Characteristic cannot be used again until all of the
	// others in the sequence have been used." (p. 65)
	availableCCs []string

	// entryLevels are the skill levels held when the career began; the
	// first-receipt rule counts receipts against this baseline
	// (interpretation I-2, ERRATA.md).
	entryLevels map[string]int

	record CareerRecord
}

// runCareerByName resolves one career through the registry.
func runCareerByName(name string, roller *dice.Roller, log *Log, decider Decider, character *Character) error {
	entry, ok := careerRegistry[name]
	if !ok {
		return fmt.Errorf("%w: %q", ErrUnknownCareer, name)
	}

	def, mechanics, err := entry()
	if err != nil {
		return err
	}

	entryLevels := make(map[string]int, len(character.Skills))
	for _, skill := range character.Skills {
		entryLevels[skill.Name] = skill.Level
	}

	run := &careerRun{
		def:         def,
		mechanics:   mechanics,
		roller:      roller,
		log:         log,
		decider:     decider,
		character:   character,
		record:      CareerRecord{Career: def.Name},
		entryLevels: entryLevels,
	}

	if err := mechanics.begin(run); err != nil {
		return err
	}

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
func (r *careerRun) term(number int) (bool, error) {
	r.log.Step(r.def.Name+": Term "+strconv.Itoa(number), r.def.Cite)

	cc, err := r.chooseCC()
	if err != nil {
		return false, err
	}

	success, err := r.mechanics.resolveTerm(r, cc)
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
func (r *careerRun) chooseCC() (string, error) {
	if len(r.availableCCs) == 0 {
		r.availableCCs = slices.Clone(r.def.ControllingCharacteristics)
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
func (r *careerRun) awardAndLog(name string, levels, cause int) {
	awardSkillAndLog(name, levels, cause, r.log, r.character)
}

// termSkills rolls the per-term career skills table eligibility (for
// Citizen, "Per Term: 4 on Table C", chart 04 table B); "For each skill,
// roll on the Career Skills Table. The character selects a column and
// rolls 1D for the specific skill" (p. 65).
func (r *careerRun) termSkills() error {
	columns := r.def.SkillColumnNames()

	for range r.def.SkillsPerTerm {
		chosen, _, err := choose(r.log, r.decider, Choice{
			ID:      ChooseSkillColumn,
			Prompt:  "Select a " + r.def.Name + " Skills column",
			Options: columns,
			Cite:    "Book 1 p. 65 (the character selects a column and rolls 1D)",
		})
		if err != nil {
			return err
		}

		roll := r.roller.Roll(1)
		seq := r.log.Roll(roll, r.def.Cite+" table C, column "+columns[chosen])

		entry := r.def.SkillColumns[chosen].Entries[roll.Total-1]
		if err := r.awardTableC(entry, seq); err != nil {
			return err
		}
	}

	return nil
}

// awardTableC applies one career skills table cell.
func (r *careerRun) awardTableC(entry career.Entry, cause int) error {
	switch entry.Kind {
	case career.EntrySkill:
		r.awardAndLog(entry.Name, 1, cause)
	case career.EntryCharacteristic:
		return r.awardCharacteristic(entry.Name, cause)
	case career.EntryMajor, career.EntryMinor:
		// "If the character does not have a Major/Minor this benefit is
		// lost." (p. 78) The current Major/Minor are the most recent ones
		// selected (p. 59).
		name := r.character.currentMajor()
		if entry.Kind == career.EntryMinor {
			name = r.character.currentMinor()
		}

		if name == "" {
			r.log.Consequence(ConsequenceEvent{Cause: cause, Kind: ConsequenceBenefitLost})

			return nil
		}

		r.awardAndLog(name, 1, cause)
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

// awardCharacteristic applies a Personal-column +1, subject to the p. 68
// maximum (awardCharacteristicAndLog).
func (r *careerRun) awardCharacteristic(name string, cause int) error {
	if _, ok := characteristicValue(&r.character.Characteristics, name); !ok {
		return fmt.Errorf("%w: %q", errUnknownCharacteristic, name)
	}

	awardCharacteristicAndLog(r.character, r.log, name, 1, cause)

	return nil
}

// continueRoll rolls Continue (for Citizen, "Continue 10-", chart 04):
// "the Character must successfully roll (2D) to Continue (or less) in the
// career. Failure ends Career Resolution. ... If the Continue roll is 2
// exactly, the character is required to Continue" (p. 66). Each term
// elapses 4 years ("the 4-year Term", p. 66).
func (r *careerRun) continueRoll() bool {
	throw := r.roller.Throw(2, r.def.ContinueTarget)
	seq := r.log.Throw(throw, nil, r.def.Cite+" (Continue "+strconv.Itoa(r.def.ContinueTarget)+"-; p. 66)")

	r.character.Age += TermYears
	r.log.Consequence(ConsequenceEvent{Cause: seq, Kind: ConsequenceYearsElapsed, Value: TermYears})

	if throw.Total == 2 {
		r.log.Consequence(ConsequenceEvent{Cause: seq, Kind: ConsequenceMandatoryContinue})

		return true
	}

	if !throw.Success {
		r.log.Consequence(ConsequenceEvent{Cause: seq, Kind: ConsequenceCareerEnded, Career: r.def.Name})

		return false
	}

	return true
}
