package chargen

// The schools a career assigns, as opposed to the ones a character
// chooses. "Some schools are attended during career resolution (assigned
// as part of career resolution)" (p. 59) — the sentence that follows Later
// Education, and a different mechanism from it.
//
// Two are implemented, both military:
//
//   - ANM School, an Operations assignment. "Resolve ANM School using
//     Education" (charts 07, 08, 12).
//   - Command College, which the flag-rank footnote sends an officer to:
//     "Command College in Year 1 of next Term (if Continue)" (chart 07's Lt
//     Commander, chart 08's Major, chart 12's Force Commander).
//
// Neither costs the character a term. Later Education substitutes a
// process "for the entire term" (p. 59, interpretation I-88); an assigned
// school is sited inside a term the character is already spending — the
// Operations assignment is one of the term's four, and Command College is
// expressly "in Year 1 of next Term". So the term runs as normal and the
// school's own year is one of its four (interpretation I-91).

import (
	"fmt"

	"github.com/philoserf/t5chargen/education"
)

// The assigned schools, by chart C row id.
const (
	anmSchoolID      = "anm_school"
	commandCollegeID = "command_college"
)

// attendAssignedSchool runs one chart C process the career assigned,
// inside the term rather than in place of it.
//
// No Major or Minor is selected. p. 59 requires them of "a character
// attending an Educational Institution", and chart C files these two under
// Military rather than Educational Institutions; more to the point, what
// the two rows provide is named outright — a Knowledge from School=ANM,
// and two Skill-1 — so neither reads a Major. Presenting two choices that
// change nothing would put noise in the record (interpretation I-92).
func (r *careerRun) attendAssignedSchool(id string) error {
	program, err := programByID(id)
	if err != nil {
		return err
	}

	r.log.Step("Assigned school: "+program.Name, "Book 1 p. 59; chart C p. 60")

	run := &eduRun{
		roller:     r.roller,
		log:        r.log,
		decider:    r.decider,
		character:  r.character,
		program:    program,
		record:     EducationRecord{Program: program.Name},
		withinTerm: true,

		// Inside a career, an award stated as a level follows the
		// career's first-receipt rule, the same as any other award the
		// term makes.
		firstReceipt: r.firstReceiptLevels,
	}

	admitted, err := run.apply()
	if err != nil {
		return err
	}

	if !admitted {
		run.finish()

		return nil
	}

	if err := run.attend(); err != nil {
		return err
	}

	run.finish()

	return nil
}

// programByID finds one chart C row.
func programByID(id string) (education.Program, error) {
	programs, err := education.Programs()
	if err != nil {
		return education.Program{}, fmt.Errorf("assigned school: %w", err)
	}

	for _, p := range programs {
		if p.ID == id {
			return p, nil
		}
	}

	return education.Program{}, fmt.Errorf("%w: chart C row %q", errUnknownProgram, id)
}
