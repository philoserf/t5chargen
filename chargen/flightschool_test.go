package chargen_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/chargen"
)

// flightSeedSearch is how wide the sweeps look. The path is narrow: a
// character has to reach University, take Honors, volunteer for NOTC and
// then open a service career.
const flightSeedSearch = 600

// flightPrompt identifies the offer in a record's event log.
const flightPrompt = "Attend Flight School?"

// flyer takes every step of the route to Flight School: University,
// Honors, NOTC into the Navy, and the school itself.
//
// The auto policy declines officer training and never selects the Service
// Academy, so neither route in is reachable in auto mode and no golden
// record passes this way (POLICY.md).
type flyer struct{}

func (flyer) Choose(c chargen.Choice) (int, error) {
	switch c.ID { //nolint:exhaustive // Only the choice points this decider steers; the rest fall through to the policy.
	case chargen.ChooseFlightSchool:
		return 0, nil // "Attend"
	case chargen.ChooseOfficerTraining:
		if i := slices.Index(c.Options, "NOTC"); i >= 0 {
			return i, nil
		}
	case chargen.ChooseEducation:
		if i := slices.Index(c.Options, "University"); i >= 0 {
			return i, nil
		}
	case chargen.ChooseHonors:
		return 0, nil
	}

	return autoPolicy(c)
}

func (flyer) Kind() chargen.DeciderKind { return chargen.DeciderPlayer }

// flightRun finds a seed whose character graduates Flight School.
func flightRun(t *testing.T) (chargen.Character, bool) {
	t.Helper()

	for seed := range uint64(flightSeedSearch) {
		c := generate(t, chargen.Options{Seed: seed, Decider: flyer{}})

		for _, record := range c.Education {
			if record.Program == "Flight School" && record.Graduated {
				return c, true
			}
		}
	}

	return chargen.Character{}, false
}

// flightOffers counts the times a record was offered Flight School.
func flightOffers(c chargen.Character) int {
	n := 0

	for _, event := range c.Events {
		if event.Choice != nil && event.Choice.Prompt == flightPrompt {
			n++
		}
	}

	return n
}

// firstSeq returns the sequence number of the first event the predicate
// accepts, or -1.
func firstSeq(c chargen.Character, accept func(chargen.Event) bool) int {
	for _, event := range c.Events {
		if accept(event) {
			return event.Seq
		}
	}

	return -1
}

// termOfferedIn names the term step in force when Flight School was
// offered, or "" if it never was.
func termOfferedIn(c chargen.Character) string {
	term := ""

	for _, event := range c.Events {
		if event.Step != nil && strings.Contains(event.Step.Name, ": Term ") {
			term = event.Step.Name
		}

		if event.Choice != nil && event.Choice.Prompt == flightPrompt {
			return term
		}
	}

	return ""
}

// admittingCourse reports whether the record holds one of the three
// courses p. 60 and p. 61 admit a character on.
func admittingCourse(c chargen.Character) bool {
	return slices.ContainsFunc(c.Education, func(r chargen.EducationRecord) bool {
		return r.Program == "OTC" || r.Program == "NOTC" || r.Program == "Service Academy"
	})
}

// TestFlightSchoolAwardsPilotThree verifies chart C's "1x Pilot-3"
// (p. 60): one Pass/Fail roll carrying three levels, not three rolls of
// one.
//
// The worked example is what settles the reading: "He receives Pilot+3
// for a total of Pilot-4" (p. 61), from a character who already held
// Pilot-1.
func TestFlightSchoolAwardsPilotThree(t *testing.T) {
	c, ok := flightRun(t)
	if !ok {
		t.Fatalf("no seed under %d graduates Flight School; widen the search", flightSeedSearch)
	}

	awards := 0

	for _, event := range c.Events {
		if event.Consequence == nil || event.Consequence.Kind != chargen.ConsequenceSkillAwarded ||
			event.Consequence.Skill != "Pilot" {
			continue
		}

		if event.Consequence.Delta == 3 {
			awards++
		}
	}

	if awards == 0 {
		t.Error("Flight School graduated and awarded no Pilot+3")
	}
}

// TestFlightSchoolIsAttendedInTheFirstTerm verifies p. 60: "The character
// attends Flight School in the first year of his first term in the Navy,
// Army, or Marines."
//
// So it is not a step C row, whatever its place in chart C's table, and
// the offer never reaches a character who has served a term already.
func TestFlightSchoolIsAttendedInTheFirstTerm(t *testing.T) {
	c, ok := flightRun(t)
	if !ok {
		t.Fatalf("no seed under %d graduates Flight School; widen the search", flightSeedSearch)
	}

	// The school sits inside a term, so the term's own step marker
	// precedes the offer, and it is the first term: a later one would
	// mean "his first term in the Navy, Army, or Marines" had been read
	// as something else.
	term := termOfferedIn(c)
	offer := firstSeq(c, func(e chargen.Event) bool {
		return e.Choice != nil && e.Choice.Prompt == flightPrompt
	})
	school := firstSeq(c, func(e chargen.Event) bool {
		return e.Step != nil && e.Step.Name == "School: Flight School"
	})

	if offer < 0 || school < 0 {
		t.Fatalf("Flight School graduated with offer=%d school=%d; one of them never happened", offer, school)
	}

	if !strings.HasSuffix(term, ": Term 1") {
		t.Errorf("Flight School was offered during %q, want the first term", term)
	}

	if school < offer {
		t.Errorf("Flight School was attended at %d, before it was offered at %d", school, offer)
	}

	if got := flightOffers(c); got != 1 {
		t.Errorf("Flight School was offered %d times, want once in the first term", got)
	}
}

// declinesFlight is eligible for Flight School and turns it down, which
// is the only way to see the first-term limit. Accepting marks the row
// attempted (I-100) and would block a second offer whatever the term.
type declinesFlight struct{ flyer }

func (d declinesFlight) Choose(c chargen.Choice) (int, error) {
	if c.ID == chargen.ChooseFlightSchool {
		return 1, nil // "Decline"
	}

	return d.flyer.Choose(c)
}

// TestFlightSchoolIsOfferedOnlyInTheFirstTerm verifies "the first year of
// his first term" (p. 60) bounds the offer and not merely the attendance.
//
// Measured on a character who declines, because accepting closes the row
// by I-100 and would make the term limit unobservable — the mutation that
// drops it passes every other test in this file.
func TestFlightSchoolIsOfferedOnlyInTheFirstTerm(t *testing.T) {
	multiTerm, offered := 0, 0

	for seed := range uint64(flightSeedSearch) {
		c := generate(t, chargen.Options{Seed: seed, Decider: declinesFlight{}})

		offers := flightOffers(c)
		if offers == 0 {
			continue
		}

		offered++

		if len(c.Careers) > 0 && len(c.Careers[0].Terms) > 1 {
			multiTerm++

			if offers > 1 {
				t.Errorf("seed %d: declined and was offered Flight School %d times over %d terms",
					seed, offers, len(c.Careers[0].Terms))
			}
		}
	}

	if offered == 0 {
		t.Fatalf("no seed under %d is offered Flight School; the sweep is asserting nothing",
			flightSeedSearch)
	}

	if multiTerm == 0 {
		t.Fatalf("no seed under %d declines and then serves a second term; "+
			"the sweep cannot see the first-term limit", flightSeedSearch)
	}
}

// TestFlightSchoolNeedsACourseThatAdmitsHim verifies the condition the
// prose adds to chart C's Pre-Req column, which has two routes and takes
// either: OTC or NOTC from a College or University (p. 61), or the
// Service Academy (p. 60).
//
// An Honors graduate who took none of them is never offered it — that is
// the half of the requirement p. 59's waivers do not reach (I-110).
func TestFlightSchoolNeedsACourseThatAdmitsHim(t *testing.T) {
	offers, honours := 0, 0

	for seed := range uint64(flightSeedSearch) {
		// Forced into a service career, or he never reaches the term
		// the offer is made in and the sweep proves nothing.
		c := generate(t, chargen.Options{Seed: seed, Career: "Spacer", Decider: honoursOnly{}})
		admitted := admittingCourse(c)
		served := len(c.Careers) > 0 && c.Careers[0].Began

		if served && !admitted && slices.ContainsFunc(c.Education, honorsDegree) {
			honours++
		}

		if got := flightOffers(c); got > 0 && !admitted {
			offers += got

			t.Errorf("seed %d: offered Flight School having taken none of the three courses", seed)
		}
	}

	if honours == 0 {
		t.Fatalf("no seed under %d takes Honors, serves a term and took no admitting course; "+
			"the sweep is asserting nothing", flightSeedSearch)
	}

	if offers != 0 {
		t.Errorf("%d offers reached a character with no admitting course", offers)
	}
}

// honorsDegree reports whether an education record carries Honors.
func honorsDegree(r chargen.EducationRecord) bool {
	return strings.Contains(r.Degree, "Honors")
}

// honoursOnly takes University and Honors and declines officer training,
// which is the auto policy's answer — an Honors graduate with no course
// that admits him.
type honoursOnly struct{}

func (honoursOnly) Choose(c chargen.Choice) (int, error) {
	switch c.ID { //nolint:exhaustive // Only the choice points this decider steers; the rest fall through to the policy.
	case chargen.ChooseEducation:
		if i := slices.Index(c.Options, "University"); i >= 0 {
			return i, nil
		}
	case chargen.ChooseHonors:
		return 0, nil
	}

	return autoPolicy(c)
}

func (honoursOnly) Kind() chargen.DeciderKind { return chargen.DeciderPlayer }
