package chargen_test

import (
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/chargen"
)

// reachPast selects a named program whether or not the character qualifies
// for it, and answers every waiver the way the case under test needs.
type reachPast struct {
	program string
	waive   bool
}

//nolint:exhaustive // Deliberately partitioned: the rest defer to the auto policy.
func (d reachPast) Choose(c chargen.Choice) (int, error) {
	switch c.ID {
	case chargen.ChooseEducation:
		return pick(c, d.program)
	case chargen.ChooseWaiver:
		if d.waive {
			return 0, nil // attempt
		}

		return 1, nil // accept the result
	default:
		return autoPolicy(c)
	}
}

func (reachPast) Kind() chargen.DeciderKind { return chargen.DeciderPlayer }

// waiverPrompts collects the reasons the record shows a waiver offered
// for.
func waiverPrompts(c chargen.Character) []string {
	var reasons []string

	for _, event := range c.Events {
		if event.Kind == chargen.EventChoice && strings.HasPrefix(event.Choice.Prompt, "Attempt an Educational Waiver?") {
			reasons = append(reasons, event.Choice.Prompt)
		}
	}

	return reasons
}

// TestPrerequisiteWaiverIsOffered verifies the first of p. 59's four
// waiver-able events: "Prerequisite". A character who reaches past what he
// qualifies for is turned away by a decision rather than a roll, and the
// waiver is what may overturn it.
//
// It was unreachable before this change, because the offer list held only
// the rows the character already qualified for — a waiver with nothing to
// waive.
func TestPrerequisiteWaiverIsOffered(t *testing.T) {
	seed, ok := shortOfUniversity(t)
	if !ok {
		t.Fatal("no seed in range falls short of University; the case is not being tested")
	}

	c := generate(t, chargen.Options{
		Seed: seed, Decider: reachPast{program: "University", waive: true},
	})

	var prerequisite bool

	for _, prompt := range waiverPrompts(c) {
		if strings.Contains(prompt, "prerequisite for University not met") {
			prerequisite = true
		}
	}

	if !prerequisite {
		t.Errorf("seed %d falls short of University and chose it, but was offered no Prerequisite waiver", seed)
	}
}

// TestPrerequisiteWaiverGatesAdmission verifies the waiver decides whether
// the unqualified applicant gets in at all. Refusing it ends the attempt
// before Admission, so no Pass/Fail is rolled and no year is consumed —
// "a failure disallows admission and consumes one year" is the Application
// Check's cost, not the prerequisite's (interpretation I-95).
func TestPrerequisiteWaiverGatesAdmission(t *testing.T) {
	seed, ok := shortOfUniversity(t)
	if !ok {
		t.Fatal("no seed in range falls short of University; widen the sweep")
	}

	declined := generate(t, chargen.Options{
		Seed: seed, Decider: reachPast{program: "University", waive: false},
	})

	// Measured over checklist step C only. The character goes on to a
	// career, so his final age says nothing about what the refusal cost.
	if years := preCareerYears(declined); years != 0 {
		t.Errorf("a refused prerequisite cost %d years before the career; it should cost none", years)
	}

	// Index 0 is checklist step C's own record: it runs before any
	// career, so a school a later career assigns cannot displace it.
	record := declined.Education[0]
	if record.Program != "University" || record.Passes != 0 || record.Graduated {
		t.Errorf("the record of a refused prerequisite is %+v, want an attempt with nothing earned", record)
	}
}

// shortOfUniversity finds a seed whose character does not meet chart C's
// University prerequisite at the moment step C asks.
//
// The final Edu cannot answer that: Aging decrements mental
// characteristics from Life Stage 5 on (chart A, p. 89), so a graduate who
// qualified at 18 can finish below the prerequisite he met. The auto
// policy takes University whenever the character qualifies for it (and
// reachPast defers to that policy for every choice before the education
// one, so the two runs are identical up to it) — so a policy run that did
// not attend University is exactly a character who fell short.
func shortOfUniversity(t *testing.T) (uint64, bool) {
	t.Helper()

	for seed := range uint64(60) {
		c, err := chargen.Generate(chargen.Options{Seed: seed, Decider: chargen.DefaultPolicy{}})
		if err != nil {
			continue
		}

		if len(c.Education) == 0 || c.Education[0].Program != "University" {
			return seed, true
		}
	}

	return 0, false
}

// TestHonorsWaiverBuysTheStatusOnly verifies the last of p. 59's four:
// Honors. Its failure "has no effect", so there is no process to
// reinstate — the waiver confers the Honors status and not the Major level
// the roll would have carried with it (interpretation I-96).
func TestHonorsWaiverBuysTheStatusOnly(t *testing.T) {
	// Seed 1's College Honors roll fails and its waiver succeeds, which
	// is the case: status without the level.
	c := generate(t, chargen.Options{Seed: 1, Decider: reachPast{program: "College", waive: true}})

	record := c.Education[0]
	if !record.Honors {
		t.Fatalf("seed 1 no longer reaches a waived Honors; %+v", record)
	}

	// The Major is awarded once per Pass, and the Honors level would be
	// one more. A waived Honors must not add it.
	major := 0

	for _, skill := range c.Skills {
		if skill.Name == record.Major {
			major = skill.Level
		}
	}

	if major != record.Passes {
		t.Errorf("Major %s is at %d for %d passes: a waived Honors awarded a level it should not",
			record.Major, major, record.Passes)
	}
}

// TestPolicyDeclinesTheHonorsWaiver verifies POLICY.md's stake rule. Every
// other educational waiver is attempted, because refusing ends the process
// or the admission; Honors ends nothing, and waivers decay for every
// attempt, so spending one on a status makes the next admission harder for
// nothing that was at risk.
func TestPolicyDeclinesTheHonorsWaiver(t *testing.T) {
	offered, attempted := 0, 0

	for seed := range uint64(60) {
		c := generate(t, chargen.Options{Seed: seed})

		for _, event := range c.Events {
			if event.Kind != chargen.EventChoice || !strings.Contains(event.Choice.Prompt, "Honors refused") {
				continue
			}

			offered++

			if event.Choice.Chosen == 0 {
				attempted++
			}
		}
	}

	if offered == 0 {
		t.Fatal("no auto-generated character failed an Honors roll; the rule is not being tested")
	}

	if attempted != 0 {
		t.Errorf("the policy attempted %d of %d Honors waivers, want none", attempted, offered)
	}
}
