package chargen

// Checklist step C: Education and Training (chart E1 p. 72; chart C p. 60;
// prose p. 59). "Education is a multi-step process. If Pre-Requisites are
// met, the character Applies for Admission. If successful, the character
// rolls for Pass/Fail for each year of the process. Pass awards one of the
// available skills; Failure terminates the process (but Waiver may result
// in reinstatement, although no skill is received). Finally, a character
// who Graduates (who Passes or who has Failure Waived) receives Graduation
// benefits." (p. 59)
//
// v1 implements the docs/PRD.md FR3 programs; unimplemented chart C rows
// are never offered. Later Education (suspending a career term, p. 59) is
// deferred with career changes (milestone 4).

import (
	"fmt"

	"github.com/philoserf/t5chargen/dice"
	"github.com/philoserf/t5chargen/education"
)

// noEducation is the always-available alternative: education is optional
// ("Consider acquiring an advanced education", p. 57 step C).
const noEducation = "None"

// eduRun is the state of one educational process.
type eduRun struct {
	roller    *dice.Roller
	log       *Log
	decider   Decider
	character *Character

	program      education.Program
	checkName    string // the chosen Pass/Fail characteristic
	lastThrowSeq int    // anchors graduation consequences
	record       EducationRecord
}

// runEducation performs checklist step C.
func runEducation(roller *dice.Roller, log *Log, decider Decider, character *Character) error {
	log.Step("Education and Training", "Book 1 p. 72 chart E1 step C")

	program, none, err := chooseProgram(log, decider, character)
	if err != nil || none {
		return err
	}

	run := &eduRun{
		roller:    roller,
		log:       log,
		decider:   decider,
		character: character,
		program:   program,
		record:    EducationRecord{Program: program.Name},
	}

	admitted, err := run.apply()
	if err != nil || !admitted {
		run.finish()

		return err
	}

	// Major and Minor are selected on admission: "The character attending
	// an Educational Institution must select a Major and a Minor" (p. 59)
	// — a refused applicant never attends and carries no Major.
	if err := run.selectMajors(); err != nil {
		return err
	}

	if err := run.attend(); err != nil {
		return err
	}

	run.finish()

	return nil
}

// finish appends the education record.
func (r *eduRun) finish() {
	r.character.Education = append(r.character.Education, r.record)
}

// chooseProgram presents the qualifying implemented programs plus "None"
// ("Pre-Requisites are minimums; higher are allowed", p. 59).
func chooseProgram(log *Log, decider Decider, character *Character) (education.Program, bool, error) {
	programs, err := education.Programs()
	if err != nil {
		return education.Program{}, false, fmt.Errorf("education: %w", err)
	}

	var qualifying []education.Program

	options := []string{}

	for _, p := range programs {
		if p.Implemented && prereqMet(p, character) {
			qualifying = append(qualifying, p)
			options = append(options, p.Name)
		}
	}

	options = append(options, noEducation)

	chosen, _, err := choose(log, decider, Choice{
		ID:      ChooseEducation,
		Prompt:  "Select pre-career education",
		Options: options,
		Cite:    "Book 1 p. 60 chart C; p. 57 step C (education is optional)",
	})
	if err != nil {
		return education.Program{}, false, err
	}

	if chosen == len(qualifying) {
		return education.Program{}, true, nil
	}

	return qualifying[chosen], false, nil
}

// prereqMet evaluates a chart C prerequisite for a v1 human character.
func prereqMet(p education.Program, character *Character) bool {
	edu := character.Characteristics.Edu

	switch p.Prerequisite.Kind {
	case education.PrereqNone:
		return true
	case education.PrereqEduMin:
		return edu >= p.Prerequisite.Value
	case education.PrereqEduMax:
		return edu <= p.Prerequisite.Value
	case education.PrereqTraMin, education.PrereqC5IsTra, education.PrereqDegree,
		education.PrereqAssigned, education.PrereqVolunteer:
		// Out of v1 pre-career scope: humans have no Tra (Mentor's
		// "C5= Tra" never holds), and degrees gate the unimplemented
		// higher programs.
		return false
	}

	return false
}

// selectMajors resolves the service (Service Academy only) and the Major
// and Minor: "The character attending an Educational Institution must
// select a Major and a Minor from the appropriate Skill and Knowledge
// list. A character may select any Major and Minor (they cannot be the
// same)" (p. 59).
func (r *eduRun) selectMajors() error {
	institution, err := r.institution()
	if err != nil || institution == "" {
		return err
	}

	majors, err := education.Majors(institution)
	if err != nil {
		return fmt.Errorf("education: %w", err)
	}

	majorIdx, _, err := choose(r.log, r.decider, Choice{
		ID:      ChooseMajor,
		Prompt:  "Select a Major",
		Options: majors,
		Cite:    "Book 1 p. 59 (Major and Minor); chart C p. 60 Available Skills",
	})
	if err != nil {
		return err
	}

	r.record.Major = majors[majorIdx]

	minors := append(append([]string{}, majors[:majorIdx]...), majors[majorIdx+1:]...)

	minorIdx, _, err := choose(r.log, r.decider, Choice{
		ID:      ChooseMinor,
		Prompt:  "Select a Minor",
		Options: minors,
		Cite:    "Book 1 p. 59 (Major and Minor; they cannot be the same)",
	})
	if err != nil {
		return err
	}

	r.record.Minor = minors[minorIdx]

	return nil
}

// institution maps the program's majors column, resolving the Service
// Academy's service first.
func (r *eduRun) institution() (education.Institution, error) {
	switch r.program.MajorsFrom {
	case "":
		return "", nil
	case "academy":
		services := []string{"Army", "Navy", "Marine"}

		chosen, _, err := choose(r.log, r.decider, Choice{
			ID:      ChooseService,
			Prompt:  "Select a service",
			Options: services,
			Cite:    "Book 1 p. 60 chart C (Service Academy)",
		})
		if err != nil {
			return "", err
		}

		r.record.Service = services[chosen]

		return education.Institution(map[string]string{
			"Army": "army", "Navy": "navy", "Marine": "marine",
		}[services[chosen]]), nil
	default:
		return education.Institution(r.program.MajorsFrom), nil
	}
}

// apply resolves admission: "To Apply (for Admission), a character must
// Check one of the stated Characteristics. A failure disallows admission
// and consumes one year." (p. 59) Reports whether admitted.
func (r *eduRun) apply() (bool, error) {
	if len(r.program.ApplyCheck) == 0 {
		return true, nil // "auto" admission
	}

	name, target, err := r.checkTarget(r.program.ApplyCheck)
	if err != nil {
		return false, err
	}

	throw := r.roller.Throw(2, target)
	seq := r.log.Throw(throw, nil, "Book 1 p. 60 chart C ("+r.program.Name+" To Apply Check "+name+")")

	if throw.Success {
		return true, nil
	}

	r.elapseYear(seq)

	waived, err := r.waiver("admission refused")
	if err != nil {
		return false, err
	}

	return waived, nil
}

// attend runs the Pass/Fail years, honors, and graduation.
func (r *eduRun) attend() error {
	name, _, err := r.checkTarget(r.program.PassCheck)
	if err != nil {
		return err
	}

	r.checkName = name

	completed := true

	for range r.program.Rolls {
		passed, ended, err := r.passFailYear()
		if err != nil {
			return err
		}

		if passed {
			r.record.Passes++
		}

		if ended {
			completed = false

			break
		}
	}

	// "a character who Graduates (who Passes or who has Failure Waived)
	// receives Graduation benefits" (p. 59): graduation requires
	// completing every year, passed or waived.
	if !completed {
		return nil
	}

	r.record.Graduated = true

	if err := r.honors(); err != nil {
		return err
	}

	return r.graduate()
}

// passFailYear resolves one Pass/Fail check: pass awards, failure ends
// attendance unless waived ("Waiver may result in reinstatement, although
// no skill is received", p. 59). A failed year still elapses
// (interpretation I-5, ERRATA.md). Returns whether the year passed, then
// whether attendance ended.
func (r *eduRun) passFailYear() (bool, bool, error) {
	_, target, err := r.checkTarget([]string{r.checkName})
	if err != nil {
		return false, false, err
	}

	throw := r.roller.Throw(2, target)
	seq := r.log.Throw(throw, nil, "Book 1 p. 60 chart C ("+r.program.Name+" Pass/Fail Check "+r.checkName+")")
	r.lastThrowSeq = seq
	r.elapseYear(seq)

	if throw.Success {
		return true, false, r.awardPass(seq)
	}

	waived, err := r.waiver("pass/fail failed")
	if err != nil {
		return false, false, err
	}

	return false, !waived, nil
}

// elapseYear advances age by one year for timed programs (chart C
// Duration; ED5 and Apprenticeship take "no time").
func (r *eduRun) elapseYear(cause int) {
	if r.program.DurationYears == 0 {
		return
	}

	r.character.Age++
	r.log.Consequence(ConsequenceEvent{Cause: cause, Kind: ConsequenceYearsElapsed, Value: 1})
}

// awardPass applies the program's per-pass Provides (chart C p. 60).
func (r *eduRun) awardPass(cause int) error {
	switch r.program.ID {
	case "ed5":
		return nil // ED5 provides only its graduation Edu-5
	case "trade_school":
		// "Major+2".
		awardSkillAndLog(r.record.Major, r.majorRate(r.record.Major, 2), cause, r.log, r.character)
	case "apprenticeship":
		return r.awardApprenticeship()
	case "college", "university", "academy":
		// "Major+1 per Pass and Minor+1 per 2 Passes".
		awardSkillAndLog(r.record.Major, r.majorRate(r.record.Major, 1), cause, r.log, r.character)

		if r.record.Passes%2 == 1 { // this call precedes the Passes increment: 2nd, 4th, ... pass
			awardSkillAndLog(r.record.Minor, r.majorRate(r.record.Minor, 1), cause, r.log, r.character)
		}
	default:
		return fmt.Errorf("%w: education program %q", errNotImplemented, r.program.ID)
	}

	return nil
}

// majorRate doubles Language acquisition: "When a specific Language is
// specified as a Major or Minor, it is acquired at double rate" (p. 59).
func (r *eduRun) majorRate(name string, levels int) int {
	if name == "Language" {
		return levels * 2
	}

	return levels
}

// awardApprenticeship resolves "Skill+4 or Knowledge+4": the skill is
// chosen from the full Available Skills matrix (interpretation I-7,
// ERRATA.md — the chart states no list); the award is caused by the
// selecting choice event.
func (r *eduRun) awardApprenticeship() error {
	names, err := education.AllSkillNames()
	if err != nil {
		return fmt.Errorf("education: %w", err)
	}

	chosen, seq, err := choose(r.log, r.decider, Choice{
		ID:      ChooseSkill,
		Prompt:  "Select the Apprenticeship skill",
		Options: names,
		Cite:    "Book 1 p. 60 chart C (Apprenticeship: Skill+4 or Knowledge+4)",
	})
	if err != nil {
		return err
	}

	r.record.Skill = names[chosen]
	awardSkillAndLog(names[chosen], 4, seq, r.log, r.character)

	return nil
}

// honors offers the optional extra roll (an honors failure is not offered
// a waiver in v1 — see the waiver doc comment): "success confers one level of his
// Major and confers Honors status. Failure has no effect." (p. 59) Only
// programs with a Major carry Honors.
func (r *eduRun) honors() error {
	if r.record.Major == "" || r.program.ID == "apprenticeship" {
		return nil
	}

	chosen, _, err := choose(r.log, r.decider, Choice{
		ID:      ChooseHonors,
		Prompt:  "Attempt the optional Honors roll?",
		Options: []string{"Attempt Honors", "Decline"},
		Cite:    "Book 1 p. 59 (Honors); chart C p. 60 (Honors row)",
	})
	if err != nil || chosen == 1 {
		return err
	}

	// The Honors row states its own check ("Int or Edu", chart C p. 60),
	// so the characteristic is chosen afresh rather than reusing the
	// program's Pass/Fail pick.
	_, target, err := r.checkTarget([]string{"Int", "Edu"})
	if err != nil {
		return err
	}

	throw := r.roller.Throw(2, target)
	seq := r.log.Throw(throw, nil, "Book 1 p. 60 chart C (Honors: Int or Edu, simul)")

	if !throw.Success {
		// "Failure has no effect." (p. 59) — the graduation consequence
		// stays anchored to the last Pass/Fail throw, not this one.
		return nil
	}

	r.lastThrowSeq = seq
	r.record.Honors = true
	awardSkillAndLog(r.record.Major, r.majorRate(r.record.Major, 1), seq, r.log, r.character)

	return nil
}

// graduate applies the Graduation column: "Edu=8" style values, "(If Edu
// already at this level, award Edu+1)" (p. 60), subject to the p. 68
// characteristic maximum; the degree is recorded.
func (r *eduRun) graduate() error {
	if r.program.GraduationDegree != "" {
		r.record.Degree = r.program.GraduationDegree
		if r.record.Honors {
			r.record.Degree = "Honors " + r.record.Degree
		}
	}

	if r.program.GraduationEdu == 0 {
		return nil
	}

	edu := r.character.Characteristics.Edu

	delta := r.program.GraduationEdu - edu
	if edu >= r.program.GraduationEdu {
		delta = 1
	}

	awardCharacteristicAndLog(r.character, r.log, "Edu", delta, r.gradCause())

	return nil
}

// gradCause anchors graduation consequences to the final pass/fail or
// honors throw.
func (r *eduRun) gradCause() int {
	return r.lastThrowSeq
}

// checkTarget resolves a chart C check column ("Int or Edu"): a
// score-guided choice when alternatives exist, and the human Tra
// substitution otherwise. Returns the chosen name and the roll-low target.
func (r *eduRun) checkTarget(names []string) (string, int, error) {
	name := names[0]

	if len(names) > 1 {
		scores := make([]int, len(names))
		for i, n := range names {
			scores[i] = r.checkValue(n)
		}

		chosen, _, err := choose(r.log, r.decider, Choice{
			ID:      ChooseCheck,
			Prompt:  "Select the characteristic to check",
			Options: names,
			Scores:  scores,
			Cite:    "Book 1 p. 59 (Check one of the stated Characteristics)",
		})
		if err != nil {
			return "", 0, err
		}

		name = names[chosen]
	}

	return name, r.checkValue(name), nil
}

// checkValue reads a check characteristic's value. Humans have no Tra:
// "Training and Education can be substituted for each other at half
// value" (p. 55), rounded in the roller's favor per the p. 19 practice
// (interpretation I-6, ERRATA.md).
func (r *eduRun) checkValue(name string) int {
	if name == "Tra" {
		return (r.character.Characteristics.Edu + 1) / 2
	}

	value, _ := characteristicValue(&r.character.Characteristics, name)

	return value
}

// waiver offers an Educational Waiver: "Check Soc or less (2D); Mod minus
// number of previous waivers rolled (successful or not)" (p. 59).
//
// P. 59 names four waiver-able events: "Prerequisite, Application Check,
// Pass/Fail Check, Honors". v1 offers waivers for the Application and
// Pass/Fail checks — the two the process description integrates
// ("Failure terminates the process (but Waiver may result in
// reinstatement...)"). Prerequisite waivers are unreachable while
// chooseProgram offers only qualifying programs (letting an unqualified
// character attempt admission is an interactive-mode concern), and Honors
// waivers are deferred with them; both land with milestone 5.
func (r *eduRun) waiver(reason string) (bool, error) {
	chosen, _, err := choose(r.log, r.decider, Choice{
		ID:      ChooseWaiver,
		Prompt:  "Attempt an Educational Waiver? (" + reason + ")",
		Options: []string{"Attempt waiver", "Accept the result"},
		Cite:    "Book 1 p. 59 (Educational Waivers)",
	})
	if err != nil || chosen == 1 {
		return false, err
	}

	previous := r.character.WaiversAttempted
	r.character.WaiversAttempted++

	target := r.character.Characteristics.Soc - previous
	throw := r.roller.Throw(2, target)
	r.log.Throw(throw, []Mod{{Name: "previous waivers", Value: -previous}},
		"Book 1 p. 59 (Waiver: Check Soc, Mod minus previous waivers)")

	return throw.Success, nil
}
