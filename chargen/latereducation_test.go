package chargen_test

import (
	"strings"
	"testing"

	"github.com/philoserf/t5chargen/chargen"
)

// schoolAt takes Later Education at the nth offer and serves every other
// term, so a test can suspend exactly one term and compare the record
// against the same seed run straight through.
type schoolAt struct {
	program string
	at      int
	seen    int
}

func (d *schoolAt) Choose(c chargen.Choice) (int, error) {
	if c.ID != chargen.ChooseLaterEducation {
		return autoPolicy(c)
	}

	d.seen++
	if d.seen != d.at {
		return 0, nil
	}

	for i, option := range c.Options {
		if option == d.program {
			return i, nil
		}
	}

	return 0, nil
}

func (*schoolAt) Kind() chargen.DeciderKind { return chargen.DeciderPlayer }

// schoolAlways never serves a term while any program is on offer, which is
// the unbounded case: the offer recurs every term and Apprenticeship has
// no prerequisite, so nothing but the passage of time ends the lifepath.
type schoolAlways struct{}

func (schoolAlways) Choose(c chargen.Choice) (int, error) {
	if c.ID == chargen.ChooseLaterEducation && len(c.Options) > 1 {
		return 1, nil
	}

	return autoPolicy(c)
}

func (schoolAlways) Kind() chargen.DeciderKind { return chargen.DeciderPlayer }

// laterEducationSeed is pinned: seed 1's Citizen serves enough terms to be
// offered school more than once, so "at the second offer" is reachable.
const laterEducationSeed = 1

// TestLaterEducationSubstitutesTheTerm verifies the p. 59 rule: a
// suspended term costs the term's four years, not the program's duration
// — "substitutes that process for the entire term" (interpretation I-88).
//
// The claim is measured inside one run rather than by comparing against
// the same seed run straight through. Schooling consumes dice, so an
// unsuspended run of the same seed is a different lifepath from the first
// throw onward; its age says nothing about the cost of a term.
func TestLaterEducationSubstitutesTheTerm(t *testing.T) {
	// Trade School is one Pass/Fail year against a four-year term, so a
	// program-duration reading and a whole-term reading differ by three.
	c := generate(t, chargen.Options{
		Seed:    laterEducationSeed,
		Decider: &schoolAt{program: "Trade School", at: 1},
	})

	if got := suspendedTermYears(t, c); got != chargen.TermYears {
		t.Errorf("the suspended term cost %d years, want the term's %d", got, chargen.TermYears)
	}

	last := c.Education[len(c.Education)-1]
	if last.Program != "Trade School" {
		t.Fatalf("the mid-career record is %q, want the program the decider chose", last.Program)
	}
}

// suspendedTermYears sums the years elapsed between the Later Education
// step and the step that follows it, which is the whole cost of the
// suspended term.
func suspendedTermYears(t *testing.T, c chargen.Character) int {
	t.Helper()

	years, counting := 0, false

	for _, event := range c.Events {
		if event.Kind == chargen.EventStep {
			if counting {
				return years
			}

			counting = strings.HasPrefix(event.Step.Name, "Later Education:")

			continue
		}

		if counting && event.Kind == chargen.EventConsequence &&
			event.Consequence.Kind == chargen.ConsequenceYearsElapsed {
			years += event.Consequence.Value
		}
	}

	if !counting {
		t.Fatal("no Later Education step in the transcript")
	}

	return years
}

// servedTerms counts the terms actually resolved, across every career.
func servedTerms(c chargen.Character) int {
	terms := 0
	for _, record := range c.Careers {
		terms += len(record.Terms)
	}

	return terms
}

// TestLaterEducationIsNotATermServed verifies a suspended term is not
// counted as one. Muster out reads len(Terms) for its benefit rolls and
// its pensions, so a term spent at school must leave no TermRecord behind
// or the character is paid for service he did not give.
func TestLaterEducationIsNotATermServed(t *testing.T) {
	suspended := generate(t, chargen.Options{
		Seed:    laterEducationSeed,
		Decider: &schoolAt{program: "Trade School", at: 1},
	})

	for _, record := range suspended.Careers {
		for i, term := range record.Terms {
			if term.Term != i+1 {
				t.Errorf("%s term %d is numbered %d: a suspended term left a gap",
					record.Career, i+1, term.Term)
			}
		}
	}
}

// TestLaterEducationSuspendsResolution verifies the suspended term runs no
// career mechanics. "Suspend career resolution" (p. 59) suspends the
// Continue throw with everything else (interpretation I-90), so the
// transcript must show the education step where the term's step would be,
// with no career step opened for it.
func TestLaterEducationSuspendsResolution(t *testing.T) {
	suspended := generate(t, chargen.Options{
		Seed:    laterEducationSeed,
		Decider: &schoolAt{program: "Trade School", at: 1},
	})

	steps, found := 0, false

	for _, event := range suspended.Events {
		if event.Kind != chargen.EventStep {
			continue
		}

		if strings.HasPrefix(event.Step.Name, "Later Education:") {
			found = true
		}

		if strings.Contains(event.Step.Name, ": Term ") {
			steps++
		}
	}

	if !found {
		t.Fatal("no Later Education step in the transcript")
	}

	if want := servedTerms(suspended); steps != want {
		t.Errorf("%d career term steps for %d terms served: a suspended term opened one", steps, want)
	}

	// The counts above stay consistent even if a suspended term is also
	// served, because then both go up together. What cannot happen if the
	// term is truly suspended is a career term between two consecutive
	// suspensions: a decider that accepts every offer must produce
	// schooling back to back, broken only where an application was
	// refused.
	if !hasBackToBackSchooling(t) {
		t.Error("every suspension was followed by a served term, so the term is not being suspended")
	}
}

// hasBackToBackSchooling reports whether a character who accepts every
// offer ever attends twice with no term served in between.
func hasBackToBackSchooling(t *testing.T) bool {
	t.Helper()

	for seed := range uint64(5) {
		c := generate(t, chargen.Options{Seed: seed, Decider: schoolAlways{}})
		previousWasSchool := false

		for _, event := range c.Events {
			if event.Kind != chargen.EventStep {
				continue
			}

			school := strings.HasPrefix(event.Step.Name, "Later Education:")
			if school && previousWasSchool {
				return true
			}

			if school || strings.Contains(event.Step.Name, ": Term ") {
				previousWasSchool = school
			}
		}
	}

	return false
}

// TestLaterEducationReplays verifies a suspended term survives the replay
// contract. No golden fixture exercises this path — the auto policy
// declines every offer — so TestReplayRoundTrip cannot reach it, and a
// mid-career education that did not replay would be a hole the fixture
// sweep could not see.
func TestLaterEducationReplays(t *testing.T) {
	c := generate(t, chargen.Options{
		Seed:    laterEducationSeed,
		Decider: &schoolAt{program: "Trade School", at: 1},
	})

	if _, err := chargen.Replay(c); err != nil {
		t.Fatalf("a record with a suspended term does not replay: %v", err)
	}
}

// TestLaterEducationTerminates verifies the unbounded case ends. The offer
// recurs at the beginning of every term and Apprenticeship has no
// prerequisite, so a character can accept forever; a suspended term throws
// no Continue, so nothing inside the career stops him.
//
// Two things do, and the test requires one of them. Aging kills (chart A
// p. 89), because the years pass whether they are spent serving or
// studying. Or an application is refused, which suspends nothing
// ("if accepted", p. 59, interpretation I-89) — the term is served after
// all and throws its Continue. A run that ended alive having served no
// term would mean neither holds and the lifepath can run forever.
//
// A regression here hangs the suite rather than failing it, which is why
// it is pinned.
func TestLaterEducationTerminates(t *testing.T) {
	for seed := range uint64(5) {
		c := generate(t, chargen.Options{Seed: seed, Decider: schoolAlways{}})

		if len(c.Education) < 2 {
			t.Errorf("seed %d: %d education records, want the mid-career path exercised", seed, len(c.Education))
		}

		if !c.Dead && servedTerms(c) == 0 {
			t.Errorf("seed %d: ended alive at age %d having served no term, so nothing ended the lifepath",
				seed, c.Age)
		}
	}
}
