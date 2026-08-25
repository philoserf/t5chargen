package chargen

// Later Education (p. 59). "Characters may suspend career resolution to
// return to school or training. At the beginning of any term, the
// character may apply for any Educational Institution or Training, and if
// accepted substitutes that process for the entire term."
//
// The process itself is the one checklist step C runs (education.go): the
// same chart C institutions, the same Admission and Pass/Fail checks, the
// same educational waivers drawing on the same decaying pool. Only the
// entry point differs, so only the entry point is here.
//
// The sentence that follows in the book — "Some schools are attended
// during career resolution (assigned as part of career resolution)" — is
// about ANM School, Flight School and Command College, which a career
// assigns rather than a character choosing. That is a different mechanism
// and is not implemented here.

import (
	"fmt"

	"github.com/philoserf/t5chargen/education"
)

// serveTheTerm is the first-listed alternative to schooling: the term the
// character would otherwise suspend. It is first because declining is the
// policy's answer and the book lists no ordering of its own — the same
// shape as the career-change offer (p. 66).
const serveTheTerm = "Serve the term in "

// midCareerPrograms drops the programs a serving character cannot apply
// to, so "any Educational Institution or Training" (p. 59) means any he
// could actually enrol in.
//
// One program is withheld. A Service Academy "provides graduates an Army
// or Navy Commission" and "The character is required to serve one term in
// the service" (p. 62): a commission is granted on entering a service, and
// the obligation it carries is a first term. Neither can be honoured by a
// character who is already serving, so the Academy belongs to step C and
// not to a term already under way (interpretation I-98).
func midCareerPrograms(programs []education.Program) []education.Program {
	kept := make([]education.Program, 0, len(programs))

	for _, p := range programs {
		if !p.PreCareerOnly {
			kept = append(kept, p)
		}
	}

	return kept
}

// laterEducation offers the p. 59 suspension at the beginning of a term
// and reports whether the term was given over to schooling.
//
// A substituted term is not served: no Risk/Reward, no skills, no Continue
// throw, because "suspend career resolution" suspends the resolution the
// Continue throw is part of (interpretation I-90). It is also not a term:
// no TermRecord is appended, so it counts toward neither the muster-out
// benefit rolls nor a pension, both of which count terms served.
func (r *careerRun) laterEducation() (bool, error) {
	programs, err := education.Programs()
	if err != nil {
		return false, fmt.Errorf("later education: %w", err)
	}

	offered, names, qualified := offeredPrograms(midCareerPrograms(programs), r.character)
	if len(offered) == 0 {
		return false, nil
	}

	options := append([]string{serveTheTerm + r.def.Name}, names...)
	scores := append([]int{1}, qualified...)

	chosen, seq, err := choose(r.log, r.decider, Choice{
		ID:         ChooseLaterEducation,
		Prompt:     "Suspend the term to return to school?",
		Options:    options,
		Scores:     scores,
		ScoreLabel: ScoreQualifies,
		Cite:       "Book 1 p. 59 (Later Education or Training); chart C p. 60",
	})
	if err != nil || chosen == 0 {
		return false, err
	}

	// "The character may apply for any Educational Institution or
	// Training" (p. 59) — any, so a serving character may reach past what
	// he qualifies for and try the Prerequisite waiver, exactly as he may
	// before a career.
	program := offered[chosen-1]
	if scores[chosen] == 0 {
		waived, err := prerequisiteWaived(r.log, r.decider, r.roller, r.character, program)
		if err != nil {
			return false, err
		}

		if !waived {
			// The attempt is recorded even though nothing came of it
			// (I-95), the same as the refused applicant
			// attendMidCareer records. The term is not suspended: the
			// substitution is conditional on being accepted (I-89).
			r.character.Education = append(r.character.Education,
				EducationRecord{Program: program.Name})

			return false, nil
		}
	}

	return r.attendMidCareer(program, seq)
}

// attendMidCareer runs one chart C process in place of a career term.
//
// The years are the term's, not the program's: "substitutes that process
// for the entire term" gives the whole four-year slot over to schooling,
// so a one-year Trade School still costs the term and a College that fails
// in its second year is not refunded the rest (interpretation I-88). The
// process charges its own years as it goes, exactly as it does before a
// career, and what remains of the term is charged against the choice that
// spent it.
//
// A rejected applicant does not suspend anything: substitution is
// conditional ("if accepted"), so the term is served after all — having
// already cost the one year the refusal consumes (interpretation I-89).
func (r *careerRun) attendMidCareer(program education.Program, cause int) (bool, error) {
	r.log.Step("Later Education: "+program.Name, "Book 1 p. 59; chart C p. 60")

	run := &eduRun{
		roller:    r.roller,
		log:       r.log,
		decider:   r.decider,
		character: r.character,
		program:   program,
		record:    EducationRecord{Program: program.Name},
	}

	began := r.character.Age
	held := heldSkillLevels(r.character)

	admitted, err := run.apply()
	if err != nil {
		return false, err
	}

	if !admitted {
		run.finish()

		return false, nil
	}

	if err := run.selectMajors(); err != nil {
		return false, err
	}

	if err := run.attend(); err != nil {
		return false, err
	}

	run.finish()
	r.creditSchooling(held)

	if remaining := TermYears - (r.character.Age - began); remaining > 0 {
		if err := r.character.advanceYears(remaining, r.roller, r.log, cause); err != nil {
			return false, err
		}
	}

	return true, nil
}

// creditSchooling raises the career-entry baseline by what the schooling
// awarded, so a level earned at school is not read as a career receipt.
//
// "Receipts" under the Job/Hobby first-receipt rule are skills received
// during this career; levels held from education are not (interpretation
// I-2, ERRATA.md), and a suspended term is not career resolution at all.
// Without this, a mid-career Apprenticeship in the skill a later Job
// determination happens to land on would demote that determination from
// Skill-4 to Skill-1 — schooling making the character strictly worse off,
// which is the reading I-2 rejects. The baseline is raised rather than
// reset, so a genuine career receipt of the same skill in an earlier term
// still demotes.
func (r *careerRun) creditSchooling(before map[string]int) {
	for _, held := range r.character.Skills {
		if gained := held.Level - before[held.Name]; gained > 0 {
			r.entryLevels[held.Name] += gained
		}
	}
}
