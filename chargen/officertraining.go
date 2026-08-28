package chargen

// OTC and NOTC, the two chart C rows a character volunteers for rather
// than applies to (p. 60; prose p. 61):
//
//	OTC   volunteer auto simul Int or Edu 1x  Soldier Skill-1  Army Officer1
//	NOTC  volunteer auto simul Int or Edu 1x  Ship Skill-1     Navy Officer1
//
// "OTC Officer Training Corps and NOTC Naval Officer Training Corps are
// College or University based courses that produce officers for the armed
// forces. Success confers a Commission (OTC= Army Officer1; NOTC= Navy
// Officer1 or Marine Officer1). The character is required to serve one
// term in the service. At the end of that term, the character may try to
// continue, or may attempt any other career available. He is in the
// Reserves." (p. 61)
//
// The commission is the same obligation the Service Academy's Officer1
// carries (I-99), so nothing here reproduces it: academyOfficer and
// academyCommission read any education record whose Degree names Officer1,
// and career selection is already narrowed by commissionedCareer.

import (
	"fmt"

	"github.com/philoserf/t5chargen/education"
	"github.com/philoserf/t5chargen/skill"
)

// The two rows, the services each commissions into, and the chart MS group
// each draws its skill from.
var officerTraining = map[string]struct {
	services []string
	group    string
}{
	"OTC":  {services: []string{"Army"}, group: skill.GroupSoldier},
	"NOTC": {services: []string{"Navy", "Marine"}, group: skill.GroupStarship},
}

// hostsOfficerTraining are the programmes p. 61 names: "College or
// University based courses". The Service Academy is not one — it
// commissions on its own Graduation line.
var hostsOfficerTraining = map[string]bool{"college": true, "university": true}

// volunteer offers OTC and NOTC to a character attending College or
// University.
//
// Offered after the programme's years and before graduation is resolved,
// because p. 61 says "attending" rather than "graduating" and the worked
// example places Eneri's NOTC check during his College years, before the
// BA he later earns (interpretation I-108). A character who washes out
// still had the chance.
//
// Once only, and only one of the two: the pair produce officers for
// different services, and a character cannot owe a term to both.
func (r *eduRun) volunteer() error {
	if !hostsOfficerTraining[r.program.ID] || r.character.attemptedOfficerTraining() {
		return nil
	}

	rows, err := offeredOfficerTraining()
	if err != nil || len(rows) == 0 {
		return err
	}

	options := make([]string, 0, len(rows)+1)
	for _, row := range rows {
		options = append(options, row.Name)
	}

	options = append(options, declineOfficerTraining)

	chosen, _, err := choose(r.log, r.decider, Choice{
		ID:      ChooseOfficerTraining,
		Prompt:  "Volunteer for officer training?",
		Options: options,
		Cite:    "Book 1 p. 60 chart C (OTC, NOTC: volunteer, auto); p. 61",
	})
	if err != nil || options[chosen] == declineOfficerTraining {
		return err
	}

	return r.runOfficerTraining(rows[chosen])
}

// declineOfficerTraining is the option that takes neither row.
const declineOfficerTraining = "Decline"

// offeredOfficerTraining returns the implemented officer-training rows in
// chart order.
func offeredOfficerTraining() ([]education.Program, error) {
	programs, err := education.Programs()
	if err != nil {
		return nil, fmt.Errorf("education: %w", err)
	}

	var rows []education.Program

	for _, program := range programs {
		if _, ok := officerTraining[program.Name]; ok && program.Implemented {
			rows = append(rows, program)
		}
	}

	return rows, nil
}

// attemptedOfficerTraining reports whether the character has already taken
// one of the two rows, passed or failed.
func (c *Character) attemptedOfficerTraining() bool {
	for _, record := range c.Education {
		if _, ok := officerTraining[record.Program]; ok {
			return true
		}
	}

	return false
}

// runOfficerTraining resolves the volunteered row: the service it
// commissions into, then its single Pass/Fail check.
//
// It runs as its own eduRun with its own record, the way an assigned
// school does, because the row has its own Graduation line and
// academyCommission reads records rather than programmes. Its Duration is
// zero, so elapseYear costs the character nothing — the course runs inside
// the College years he is already spending.
func (r *eduRun) runOfficerTraining(program education.Program) error {
	row := officerTraining[program.Name]

	service, err := r.chooseCommissioningService(program.Name, row.services)
	if err != nil {
		return err
	}

	// The Graduation column carries the service in its text — "Army
	// Officer1", "Navy Officer1 or Marine Officer1" — and the data can
	// hold only one string, so the degree is built from the service
	// actually chosen rather than read off the row. OTC arrives at the
	// same answer the data prints; NOTC's Marine branch does not.
	program.GraduationDegree = service + " " + officer1

	run := &eduRun{
		roller:    r.roller,
		log:       r.log,
		decider:   r.decider,
		character: r.character,
		program:   program,
		record:    EducationRecord{Program: program.Name, Service: service},
	}

	defer run.finish()

	r.log.Step(program.Name, "Book 1 p. 60 chart C; p. 61")

	return run.attend()
}

// chooseCommissioningService resolves which service the commission is
// into. OTC names one, so nothing is asked; NOTC confers "Navy Officer1 or
// Marine Officer1" (p. 61) and the character picks.
func (r *eduRun) chooseCommissioningService(program string, services []string) (string, error) {
	if len(services) == 1 {
		return services[0], nil
	}

	chosen, _, err := choose(r.log, r.decider, Choice{
		ID:      ChooseService,
		Prompt:  "Select the service " + program + " commissions into",
		Options: services,
		Cite:    "Book 1 p. 61 (NOTC= Navy Officer1 or Marine Officer1)",
	})
	if err != nil {
		return "", err
	}

	return services[chosen], nil
}

// awardOfficerTraining awards the row's single skill: "Soldier Skill-1"
// for OTC, "Ship Skill-1" for NOTC (chart C p. 60). Both name a chart MS
// group, and chart C transcribes the same two sets in its own Available
// Skills matrix — held to each other at load, so this reads the master
// list rather than the matrix copy.
func (r *eduRun) awardOfficerTraining() error {
	row, ok := officerTraining[r.program.Name]
	if !ok {
		return fmt.Errorf("%w: officer training %q", errNotImplemented, r.program.Name)
	}

	options := skill.InGroup(row.group)
	if len(options) == 0 {
		return fmt.Errorf("%w: chart MS lists no %s", errNotImplemented, row.group)
	}

	chosen, seq, err := choose(r.log, r.decider, Choice{
		ID:      ChooseSkill,
		Prompt:  "Select the " + r.program.Name + " skill",
		Options: options,
		Cite:    "Book 1 p. 60 chart C (" + r.program.Name + " Provides)",
	})
	if err != nil {
		return err
	}

	if err := awardSkillAndLog(options[chosen], r.receipt(options[chosen], 1),
		seq, r.log, r.decider, r.character); err != nil {
		return err
	}

	return nil
}
