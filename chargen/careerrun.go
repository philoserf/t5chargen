package chargen

// The generic career runner: the term loop, controlling-characteristic
// rotation, career skills table, and Continue throw shared by every career
// (Book 1 chart D p. 64; prose pp. 65-66). Career-specific exceptional
// mechanics (docs/PRD.md, Architecture notes) plug in through the
// careerMechanics interface; Citizen's live in citizen.go.

import (
	"errors"
	"fmt"
	"slices"
	"strconv"

	"github.com/philoserf/t5chargen/career"
	"github.com/philoserf/t5chargen/dice"
	"github.com/philoserf/t5chargen/skill"
)

// errUnknownCharacteristic reports a data-file characteristic name outside
// the six standard abbreviations.
var errUnknownCharacteristic = errors.New("unknown characteristic")

// errNotImplemented reports a table cell whose resolution lands in a later
// milestone (docs/PRD.md milestones 2-3).
var errNotImplemented = errors.New("not implemented until education/skill milestones")

// errUnregisteredCareer reports a career present in career.Available but
// missing from careerRegistry — an internal wiring bug, distinct from the
// user-facing ErrUnknownCareer (which the CLI maps to a usage exit).
var errUnregisteredCareer = errors.New("career has no registered mechanics")

// careerMechanics is one career's exceptional mechanics. The interface is
// unexported and grows with the careers that need more seams (rank,
// commission, muster out land with milestones 3-4).
type careerMechanics interface {
	// begin resolves career entry: automatic for Citizen (chart 04), a
	// To Begin throw for most careers (chart D p. 64; p. 65). It reports
	// whether the career began; a failed attempt costs a year (p. 65).
	begin(r *careerRun) (bool, error)

	// resolveTerm runs the career's Risk/Reward variant for the term
	// (p. 65: Citizen Life for Citizens) and applies its awards.
	resolveTerm(r *careerRun, cc string) (termOutcome, error)
}

// termOutcome is a term's Risk/Reward-variant result.
type termOutcome struct {
	// success is the variant's outcome (Citizen Life success, a Scout
	// Discovery).
	success bool

	// skillRolls overrides the definition's SkillsPerTerm when non-zero
	// (chart 05 table B splits eligibility by duty).
	skillRolls int

	// endCareer ends the career after the term completes, without a
	// Continue roll — a disabled character "Musters Out at Term end"
	// (chart 05 p. 79; muster out itself is milestone 4).
	endCareer bool

	// died ends the term and the career immediately: no skills, no
	// Continue ("the Character is dead", p. 65).
	died bool
}

// careerRegistry maps canonical career names to their definition and
// mechanics. Its key set must match career.Available (tested).
var careerRegistry = map[string]func() (*career.Definition, careerMechanics, error){
	"Citizen": newCitizen,
	"Scout":   newScout,
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

// runCareerByName resolves one career through the registry, reporting
// whether the career began (a failed To Begin leaves a began:false record
// and the caller offers the remaining careers, p. 65).
func runCareerByName(name string, roller *dice.Roller, log *Log, decider Decider, character *Character) (bool, error) {
	entry, ok := careerRegistry[name]
	if !ok {
		return false, fmt.Errorf("%w: %q", errUnregisteredCareer, name)
	}

	def, mechanics, err := entry()
	if err != nil {
		return false, err
	}

	// The baseline is captured before mechanics.begin deliberately: a
	// skill granted during career entry (a To Begin outcome, milestone 3)
	// is a career receipt under interpretation I-2 (ERRATA.md) and demotes
	// a later Job/Hobby determination of the same skill to a later receipt.
	entryLevels := make(map[string]int, len(character.Skills))
	for _, held := range character.Skills {
		entryLevels[held.Name] = held.Level
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

	began, err := mechanics.begin(run)
	if err != nil {
		return false, err
	}

	run.record.Began = began

	if !began {
		character.Careers = append(character.Careers, run.record)

		return false, nil
	}

	for {
		continued, err := run.term(len(run.record.Terms) + 1)
		if err != nil {
			return false, err
		}

		if !continued {
			break
		}
	}

	character.Careers = append(character.Careers, run.record)

	return true, nil
}

// term resolves one 4-year term and reports whether the career continues.
func (r *careerRun) term(number int) (bool, error) {
	r.log.Step(r.def.Name+": Term "+strconv.Itoa(number), r.def.Cite)

	cc, err := r.chooseCC()
	if err != nil {
		return false, err
	}

	outcome, err := r.mechanics.resolveTerm(r, cc)
	if err != nil {
		return false, err
	}

	if outcome.died {
		// "the Character is dead" (p. 65): the term ends at the injury —
		// no skills, no Continue.
		r.record.Terms = append(r.record.Terms, TermRecord{
			Term: number, ControllingCharacteristic: cc,
		})

		return false, nil
	}

	if err := r.termSkills(outcome.skillRolls); err != nil {
		return false, err
	}

	continued := false
	if !outcome.endCareer {
		continued = r.continueRoll()
	}

	r.record.Terms = append(r.record.Terms, TermRecord{
		Term:                      number,
		ControllingCharacteristic: cc,
		Success:                   outcome.success,
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

// termSkills rolls the per-term career skills table eligibility (for
// Citizen, "Per Term: 4 on Table C", chart 04 table B); "For each skill,
// roll on the Career Skills Table. The character selects a column and
// rolls 1D for the specific skill" (p. 65).
func (r *careerRun) termSkills(rolls int) error {
	if rolls == 0 {
		rolls = r.def.SkillsPerTerm
	}

	columns := r.def.SkillColumnNames()

	for range rolls {
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
		name, err := r.resolveSkillName(entry.Name)
		if err != nil {
			return err
		}

		r.awardAndLog(name, 1, cause)
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
	case career.EntryTrade, career.EntryArt, career.EntryScience, career.EntryStarship:
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
	target := r.def.ContinueTarget
	label := "Continue " + strconv.Itoa(target) + "-"

	if r.def.ContinueCharacteristic != "" {
		// A characteristic Continue target (chart 05: "Continue Int").
		target, _ = characteristicValue(&r.character.Characteristics, r.def.ContinueCharacteristic)
		label = "Continue " + r.def.ContinueCharacteristic
	}

	throw := r.roller.Throw(2, target)
	seq := r.log.Throw(throw, nil, r.def.Cite+" ("+label+"; p. 66)")

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
