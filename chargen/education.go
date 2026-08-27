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
// are never offered. Later Education — the same process, entered at the
// beginning of a career term instead of before the career (p. 59) — is in
// latereducation.go.

import (
	"fmt"
	"slices"
	"strings"

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

	// withinTerm marks a school the career assigned, which is sited
	// inside a term the character is already spending and so charges no
	// years of its own (p. 59, interpretation I-91).
	withinTerm bool

	// firstReceipt applies the career's first-receipt rule to an award
	// stated as a level ("Knowledge-2"): the stated level the first time,
	// one level after that. Nil outside a career, where nothing has been
	// received yet.
	firstReceipt func(name string, levels int) int
}

// receipt applies the first-receipt rule where one is in force.
func (r *eduRun) receipt(name string, levels int) int {
	if r.firstReceipt == nil {
		return levels
	}

	return r.firstReceipt(name, levels)
}

// prerequisiteWaived offers the p. 59 Prerequisite waiver to a character
// who chose a row he falls short of, and reports whether he may proceed to
// Admission. Declining, or failing the waiver, ends the attempt: he was
// never admitted, so no year is consumed — "a failure disallows admission
// and consumes one year" is the Application Check's cost, not this one.
func prerequisiteWaived(log *Log, decider Decider, roller *dice.Roller, character *Character,
	program education.Program,
) (bool, error) {
	waived, err := offerWaiver(log, decider, roller, character,
		educationWaiver("prerequisite for "+program.Name+" not met"))
	if err != nil {
		return false, err
	}

	return waived, nil
}

// runEducation performs checklist step C.
func runEducation(roller *dice.Roller, log *Log, decider Decider, character *Character) error {
	log.Step("Education and Training", "Book 1 p. 72 chart E1 step C")

	program, short, none, err := chooseProgram(log, decider, character)
	if err != nil || none {
		return err
	}

	if short {
		waived, err := prerequisiteWaived(log, decider, roller, character, program)
		if err != nil {
			return err
		}

		if !waived {
			character.Education = append(character.Education, EducationRecord{Program: program.Name})

			return nil
		}
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

// chooseProgram presents chart C's implemented rows plus "None"
// ("Pre-Requisites are minimums; higher are allowed", p. 59). It reports
// the chosen program, whether the character falls short of its
// prerequisite, and whether he declined education altogether.
func chooseProgram(log *Log, decider Decider, character *Character) (education.Program, bool, bool, error) {
	programs, err := education.Programs()
	if err != nil {
		return education.Program{}, false, false, fmt.Errorf("education: %w", err)
	}

	offered, options, qualified := offeredPrograms(programs, character)
	options = append(options, noEducation)
	qualified = append(qualified, 1)

	chosen, _, err := choose(log, decider, Choice{
		ID:         ChooseEducation,
		Prompt:     "Select pre-career education",
		Options:    options,
		Scores:     qualified,
		ScoreLabel: ScoreQualifies,
		Cite:       "Book 1 p. 60 chart C; p. 57 step C (education is optional)",
	})
	if err != nil {
		return education.Program{}, false, false, err
	}

	if chosen == len(offered) {
		return education.Program{}, false, true, nil
	}

	return offered[chosen], qualified[chosen] == 0, false, nil
}

// offeredPrograms returns chart C's implemented rows in chart order, their
// names, and a parallel 1/0 for whether the character meets each
// prerequisite.
//
// Every row is offered, not only the qualifying ones, because p. 59 lists
// "Prerequisite" first among the adverse decisions a Waiver may overturn —
// a rule with nothing to overturn while the unqualified rows are hidden.
// The qualification travels as a Score, which is engine-provided decision
// data rather than part of the printed rule, so it guides a decider
// without entering the record (see Choice).
//
// Assigned rows are still never offered: their prerequisite is not a
// threshold a character can fall short of, it is a career handing him a
// place (interpretation I-95). Nor are they reached through here at all,
// which is why the once-only rule below cannot affect them: an ANM School
// or a Command College is sited inside a term by the career, and a second
// promotion may well site a second one.
//
// A program already attempted is not offered again (interpretation
// I-100). ED5 is the case the book states outright — "The process can be
// attempted once" (p. 61) — and the reading generalises: Major and Minor
// are reselected "each time a new Educational Institution is attended"
// (p. 59), which is a sentence about attending different ones.
func offeredPrograms(programs []education.Program, character *Character) ([]education.Program, []string, []int) {
	var (
		offered   []education.Program
		options   []string
		qualified []int
	)

	for _, p := range programs {
		// Neither an assigned school nor a volunteer course is chosen
		// here. An assigned school is reached from a career (p. 59);
		// OTC and NOTC are "College or University based courses"
		// (p. 61) offered from inside the programme hosting them, by
		// eduRun.volunteer. Both would otherwise appear on the step C
		// menu as though a character could walk into one.
		if !p.Implemented ||
			p.Prerequisite.Kind == education.PrereqAssigned ||
			p.Prerequisite.Kind == education.PrereqVolunteer {
			continue
		}

		if character.attempted(p.Name) || !available(p, character, programs) {
			continue
		}

		offered = append(offered, p)
		options = append(options, p.Name)

		qualified = append(qualified, boolScore(prereqMet(p, character)))
	}

	return offered, options, qualified
}

// boolScore renders a yes/no decision aid as the 1/0 a Score carries.
func boolScore(yes bool) int {
	if yes {
		return 1
	}

	return 0
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
	case education.PrereqDegree:
		return character.holdsDegree(p.Prerequisite.ValueName)
	case education.PrereqVolunteer:
		// "volunteer auto" (chart C p. 60): there is no admission throw,
		// and the offer is made by eduRun.volunteer rather than reached
		// through the step C menu — a character volunteers from inside
		// the College or University he is already attending.
		return true
	case education.PrereqTraMin, education.PrereqC5IsTra, education.PrereqAssigned:
		// Out of v1 pre-career scope: humans have no Tra, so Mentor's
		// "C5= Tra" and Training Course's "Tra 5+" never hold, and an
		// assigned school is reached from a career rather than chosen.
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

	throw := r.roller.Check(2, target)
	seq := r.log.Throw(throw, nil, "Book 1 p. 60 chart C ("+r.program.Name+" To Apply Check "+name+")")

	if throw.Success {
		return true, nil
	}

	if err := r.elapseYear(seq); err != nil {
		return false, err
	}

	waived, err := offerWaiver(r.log, r.decider, r.roller, r.character, educationWaiver("admission refused"))
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

	// Offered before graduation is resolved: p. 61 says a character
	// "attending" College or University may volunteer, and the worked
	// example puts Eneri's NOTC check inside his College years
	// (interpretation I-108). A character who washes out still had it.
	if err := r.volunteer(); err != nil {
		return err
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

	throw := r.roller.Check(2, target)
	seq := r.log.Throw(throw, nil, "Book 1 p. 60 chart C ("+r.program.Name+" Pass/Fail Check "+r.checkName+")")
	r.lastThrowSeq = seq

	if err := r.elapseYear(seq); err != nil {
		return false, false, err
	}

	if throw.Success {
		return true, false, r.awardPass(seq)
	}

	waived, err := offerWaiver(r.log, r.decider, r.roller, r.character, educationWaiver("pass/fail failed"))
	if err != nil {
		return false, false, err
	}

	return false, !waived, nil
}

// elapseYear advances age by one year for timed programs (chart C
// Duration; ED5 and Apprenticeship take "no time").
func (r *eduRun) elapseYear(cause int) error {
	if r.program.DurationYears == 0 || r.withinTerm {
		return nil
	}

	return r.character.advanceYears(1, r.roller, r.log, cause)
}

// awardPass applies the program's per-pass Provides (chart C p. 60).
// awardSelection handles the rows whose Provides is a single selection
// from a named list, and reports whether it recognised the row.
//
// Separate from awardPass because the two shapes are different: these pick
// one name from a list and award it, while the rows below award a Major
// and Minor the character chose on admission.
func (r *eduRun) awardSelection() (bool, error) {
	switch r.program.ID {
	case anmSchoolID:
		return true, r.awardANMKnowledge()
	case commandCollegeID:
		return true, r.awardCommandCollege()
	case "apprenticeship":
		return true, r.awardApprenticeship()
	case "otc", "notc":
		return true, r.awardOfficerTraining()
	}

	return false, nil
}

func (r *eduRun) awardPass(cause int) error {
	if handled, err := r.awardSelection(); handled {
		return err
	}

	switch r.program.ID {
	case "ed5":
		return nil // ED5 provides only its graduation Edu-5
	case "trade_school":
		// "Major+2".
		awardSkillAndLog(r.record.Major, r.majorRate(r.record.Major, 2), cause, r.log, r.character)
	case "medical_school", "law_school":
		// "Medic-4" over four Pass/Fail rolls, "Advocate-2" over two
		// (I-104).
		r.awardNamedSkill(cause)
	case "college", "university", "academy", "masters", "professors":
		// "Major+1 per Pass and Minor+1 per 2 Passes" — one cell on the
		// chart, merged across these five rows (p. 60).
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

// awardANMKnowledge resolves "Knowledge-2 from School=ANM" (chart C
// p. 60). ANM is Army-Navy-Marine, so the source is those three columns of
// the Available Skills matrix, narrowed to entries the Master Skill List
// calls Knowledges — the row asks for a Knowledge, and taking it at its
// word keeps the award clear of the unresolved question about awarding a
// bare container skill (p. 134).
func (r *eduRun) awardANMKnowledge() error {
	names, err := education.ANMKnowledges()
	if err != nil {
		return fmt.Errorf("education: %w", err)
	}

	chosen, seq, err := choose(r.log, r.decider, Choice{
		ID:      ChooseSkill,
		Prompt:  "Select the ANM School Knowledge",
		Options: names,
		Cite:    "Book 1 p. 60 chart C (ANM School Provides Knowledge-2 from School=ANM)",
	})
	if err != nil {
		return err
	}

	awardSkillAndLog(names[chosen], r.receipt(names[chosen], 2), seq, r.log, r.character)

	return nil
}

// awardCommandCollege resolves "2x Skill-1" (chart C p. 60). The row names
// no source, so the selection is the full Available Skills matrix — the
// same reading interpretation I-7 gives the Apprenticeship's unqualified
// "Skill+4", and for the same reason: the chart states no list.
func (r *eduRun) awardCommandCollege() error {
	names, err := education.AllSkillNames()
	if err != nil {
		return fmt.Errorf("education: %w", err)
	}

	for range commandCollegeSkills {
		chosen, seq, err := choose(r.log, r.decider, Choice{
			ID:      ChooseSkill,
			Prompt:  "Select a Command College skill",
			Options: names,
			Cite:    "Book 1 p. 60 chart C (Command College Provides 2x Skill-1)",
		})
		if err != nil {
			return err
		}

		awardSkillAndLog(names[chosen], r.receipt(names[chosen], 1), seq, r.log, r.character)
	}

	return nil
}

// commandCollegeSkills is the "2x" of "2x Skill-1" (chart C p. 60).
const commandCollegeSkills = 2

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

// honors offers the optional extra roll: "success confers one level of his
// Major and confers Honors status. Failure has no effect." (p. 59) Only
// programs with a Major carry Honors. A failure is offered the fourth
// p. 59 waiver, which buys the status and not the level (honorsWaiver,
// interpretation I-96).
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

	throw := r.roller.Check(2, target)
	seq := r.log.Throw(throw, nil, "Book 1 p. 60 chart C (Honors: Int or Edu, simul)")

	if !throw.Success {
		// "Failure has no effect." (p. 59) — the graduation consequence
		// stays anchored to the last Pass/Fail throw, not this one.
		return r.honorsWaiver()
	}

	r.lastThrowSeq = seq
	r.record.Honors = true
	awardSkillAndLog(r.record.Major, r.majorRate(r.record.Major, 1), seq, r.log, r.character)

	return nil
}

// honorsWaiver offers the last of p. 59's four waiver-able events. A
// failed Honors roll "has no effect", so unlike the others there is no
// process to reinstate — what the waiver buys is the Honors status itself,
// and not the Major level the roll would have carried with it
// (interpretation I-96).
//
// The status is worth a waiver on its own: an Honors Degree is the
// prerequisite chart C prints for Medical School, Law School and Flight
// School.
func (r *eduRun) honorsWaiver() error {
	waived, err := offerWaiver(r.log, r.decider, r.roller, r.character, honorsWaiverPrompt())
	if err != nil || !waived {
		return err
	}

	r.record.Honors = true

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

	delta, awarded := graduationEdu(edu, r.program.GraduationEdu)
	if !awarded {
		return nil
	}

	awardCharacteristicAndLog(r.character, r.log, "Edu", delta, r.gradCause())

	return nil
}

// graduationEdu applies chart C's Graduation column and the parenthetical
// above it: "(If Edu already at this level, award Edu+1)" (p. 60).
//
// The values are positions rather than rewards. "C5 Education As A
// Characteristic reflects the individual's ability in an Educational
// setting, even if the person does not have the formal documentation ... a
// character with Edu=9 can function at the equivalent of a Masters" (p. 62)
// — so Edu 8 is where a Bachelors puts a character, Edu 9 a Masters, Edu 12
// a Professor. A programme moves him to its level.
//
// "Already at this level" is therefore read as the page writes it, at that
// level and not past it (interpretation I-105). A character who arrives
// above the value has nothing to gain from the schooling's Edu, and the
// consolation +1 is for the one who is exactly there.
func graduationEdu(edu, graduation int) (int, bool) {
	switch {
	case edu < graduation:
		return graduation - edu, true
	case edu == graduation:
		return 1, true
	default:
		return 0, false
	}
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

// attempted reports whether the character has already been through this
// program, successfully or not.
//
// Attempted, not graduated. Every path through schooling leaves a record —
// a refused applicant, a cadet who failed out, and a graduate alike — and
// the book counts attempts rather than successes where it counts at all:
// "The process can be attempted once. It takes no time. Failure has no
// other effect" (p. 61, of ED5). A rule that let a failure be retried
// would make failure free, which is the one thing that sentence rules out.
func (c *Character) attempted(program string) bool {
	for _, record := range c.Education {
		if record.Program == program {
			return true
		}
	}

	return false
}

// available reports whether a program is open to this character at all, as
// distinct from one he falls short of and might waive into.
//
// The distinction is p. 59's: "Pre-Requisites are minimums; higher are
// allowed", and a Waiver overturns "an adverse die roll or decision". Every
// row this repo implements states a floor, which a character can fall short
// of adversely and reach past by waiver — except one.
func available(p education.Program, character *Character, programs []education.Program) bool {
	return withinCeiling(p, character) &&
		notBelowWhatHeHolds(p, character, programs) &&
		credentialIsReachable(p, character)
}

// credentialIsReachable withholds a row gated on a credential from a
// character who has never been to school.
//
// The four such rows are the ones a University provides: "University ...
// can also provide a Masters Program leading to a Masters Degree and a
// Professors Program leading to a professorship. Often associated with a
// University are a Medical School ... and a Law School" (p. 61). A
// character choosing his first schooling is at no University.
//
// Their prerequisites are waivable — p. 61 says so outright — and that is
// why they are offered to a serving character who fell short of one.
// Step C is different in kind: it runs before any career and is a
// character's first education, so his history is empty and no credential
// is obtainable. A waiver overturns an adverse decision (p. 59), and a row
// nobody could ever qualify for at this step produces none. Offering it
// there put four rows scoring zero in front of every eighteen-year-old,
// none of which he could even attempt to earn (interpretation I-106).
func credentialIsReachable(p education.Program, character *Character) bool {
	if p.Prerequisite.Kind != education.PrereqDegree {
		return true
	}

	return len(character.Education) > 0
}

// withinCeiling holds a maximum prerequisite closed.
//
// ED5's "Edu 4 -" is the only prerequisite in chart C that is a ceiling
// rather than a floor, and a ceiling cannot be waived: a waiver overturns
// an adverse decision (p. 59), and being better educated than a remedial
// programme requires is not adversity. The page says what ED5 is for —
// "a program to raise low Edu to a minimally acceptable level ... a
// character with Edu less than 5 needs to take ED5" (p. 61) — so a
// character above the ceiling is not refused it, he has no business in it
// (interpretation I-102).
func withinCeiling(p education.Program, character *Character) bool {
	if p.Prerequisite.Kind != education.PrereqEduMax {
		return true
	}

	return character.Characteristics.Edu <= p.Prerequisite.Value
}

// notBelowWhatHeHolds withholds a Basic programme from a character who has
// already graduated one of chart C's Higher Education rows.
//
// The chart prints the two as separate blocks, and p. 61 gives the ladder
// in its own words: ED5 exists "Because Edu-5 is the minimum prerequisite
// for Trade Schools", so a Basic row is the rung you climb to reach a
// Higher one. A graduate of College or University has climbed past it, and
// the schooling below certifies nothing he does not hold (I-102).
func notBelowWhatHeHolds(p education.Program, character *Character, programs []education.Program) bool {
	if p.Group != groupBasic {
		return true
	}

	for _, record := range character.Education {
		if !record.Graduated {
			continue
		}

		for _, held := range programs {
			if held.Name == record.Program && held.Group == groupHigher {
				return false
			}
		}
	}

	return true
}

// The chart C blocks, printed beside their rows (p. 60).
const (
	groupBasic  = "basic"
	groupHigher = "higher"
)

// holdsDegree reports whether the character has earned the credential a
// chart C prerequisite names: "A University Masters Program requires a
// Bachelors. A Professors Program requires a Masters. Medical School or
// Law School requires an Honors Bachelors" (p. 61).
//
// The comparison is on the credential the degree carries, not on the whole
// string, because chart C prints a degree with what came with it: the
// Service Academy's Graduation is "C5=8 BA Officer1", and its graduate has
// a Bachelors like any other (interpretation I-103). An Honors run is
// recorded the same way, as "Honors BA".
//
// "Honors BA" therefore asks two things — a Bachelors, and the Honors that
// p. 59's optional roll confers — and reads them from the two places the
// record keeps them.
func (c *Character) holdsDegree(want string) bool {
	credential, honors := strings.CutPrefix(want, honorsPrefix)

	for _, record := range c.Education {
		if !record.Graduated || !carries(record.Degree, credential) {
			continue
		}

		if !honors || record.Honors {
			return true
		}
	}

	return false
}

// carries reports whether a recorded degree includes the named credential.
//
// Whole words only: a degree is a sequence of tokens ("Honors BA
// Officer1"), and matching on the substring would let "MA" answer for a
// credential that merely contains those letters.
func carries(degree, credential string) bool {
	return slices.Contains(strings.Fields(degree), credential)
}

// honorsPrefix is how an Honors graduation is recorded, and how chart C
// prints the prerequisite that asks for one.
const honorsPrefix = "Honors "

// awardNamedSkill applies the Provides of the two professional schools,
// the only chart C rows that name the skill they teach: "Medic-4" and
// "Advocate-2" (p. 60).
//
// One level per Pass, so the stated level is what a full run of passes
// reaches — four rolls to Medic-4, two to Advocate-2 — and a student who
// fails some of his years leaves with correspondingly less (I-104).
func (r *eduRun) awardNamedSkill(cause int) {
	name := education.MedicalAward
	if r.program.ID == "law_school" {
		name = education.LawAward
	}

	awardSkillAndLog(name, r.receipt(name, 1), cause, r.log, r.character)
}
