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

	qualifying, names := qualifyingPrograms(programs, r.character)
	if len(qualifying) == 0 {
		return false, nil
	}

	options := append([]string{serveTheTerm + r.def.Name}, names...)

	chosen, seq, err := choose(r.log, r.decider, Choice{
		ID:      ChooseLaterEducation,
		Prompt:  "Suspend the term to return to school?",
		Options: options,
		Cite:    "Book 1 p. 59 (Later Education or Training); chart C p. 60",
	})
	if err != nil || chosen == 0 {
		return false, err
	}

	return r.attendMidCareer(qualifying[chosen-1], seq)
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

	if remaining := TermYears - (r.character.Age - began); remaining > 0 {
		if err := r.character.advanceYears(remaining, r.roller, r.log, cause); err != nil {
			return false, err
		}
	}

	return true, nil
}
