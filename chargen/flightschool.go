package chargen

// Flight School (chart C p. 60): "Flight School is an intensive training
// program to create pilots for naval or military service. The character
// attends Flight School in the first year of his first term in the Navy,
// Army, or Marines."
//
// So it is not a step C row a school leaver applies to, whatever its
// place in chart C's table: it is attended inside a career, in one named
// term, like the Command College printed beside it. What separates the
// two is that Command College is assigned — "A Character must attend" —
// and Flight School is elective: both of the sentences that admit a
// character say he "may attend" (interpretation I-110).
//
// Two routes in, and either will do. "College or University Honors
// Graduates who participated in OTC or NOTC may attend Flight School"
// (p. 61), and "Service Academy Honors Graduates may attend Flight
// School" (p. 60). The Honors half of both is chart C's own Pre-Req,
// "Honors BA", and is waivable as the worked example shows; the course
// named beside it is not.

import "github.com/philoserf/t5chargen/education"

// The two answers. Attending is listed first because it is what the rule
// is about; the policy names the other by position.
const (
	attendFlightSchool  = "Attend"
	declineFlightSchool = "Decline"
)

// offerFlightSchool offers the row in the first year of the character's
// first Armed Forces term, and attends it if he accepts.
//
// The year it takes comes out of the term, which is what the worked
// example means by "Because Flight School took a year, this first Term is
// reduced to three years" (p. 66) — attendAssignedSchool already runs a
// school inside a term rather than in place of one.
func (r *careerRun) offerFlightSchool() error {
	program, err := programByID(flightSchoolID)
	if err != nil {
		return err
	}

	if !r.eligibleForFlightSchool(program) {
		return nil
	}

	chosen, _, err := choose(r.log, r.decider, Choice{
		ID:      ChooseFlightSchool,
		Prompt:  "Attend Flight School?",
		Options: []string{attendFlightSchool, declineFlightSchool},
		Cite:    "Book 1 p. 60 chart C (Flight School: Honors BA, auto, 1 year, C2, 1x Pilot-3)",
	})
	if err != nil {
		return err
	}

	if chosen != 0 {
		return nil
	}

	return r.attendAssignedSchool(flightSchoolID)
}

// eligibleForFlightSchool reports whether the offer is his to take: the
// first term of a service that flies, an Honors degree, one of the three
// courses that admit him, and no previous attempt.
//
// "his first term" is read as the first term of this career rather than
// of his life, because the sentence names the service — a Soldier who
// later changes to the Navy is in his first term in the Navy. A program
// is attempted once either way (I-100).
func (r *careerRun) eligibleForFlightSchool(program education.Program) bool {
	return r.def.ArmedForces != nil &&
		len(r.record.Terms) == 0 &&
		!r.character.attempted(program.Name) &&
		r.character.holdsDegree(program.Prerequisite.ValueName) &&
		attendedARequiredProgram(program, r.character)
}
